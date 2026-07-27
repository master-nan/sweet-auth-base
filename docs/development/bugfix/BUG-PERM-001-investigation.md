# BUG-PERM-001 Random Authorization 403 调查记录

状态：调查完成；根因已由源码、数据库、同一运行实例的访问时间线、修复前失败测试和 race detector 共同确认。

## 1. 问题与复现方式

同一登录会话、同一账号在未发生认证失败时，页面并发加载多个管理端接口，其中部分请求返回：

```json
{
  "status_code": 403,
  "error_code": 30006,
  "error_message": "无权限访问",
  "success": false
}
```

开发环境中的稳定复现步骤：

1. 使用 `admin` 登录。
2. 确认 `/admin/dict/code/:code` 在角色权限保存前可返回 200。
3. 保存 `super_admin` 的菜单和按钮权限。
4. 再请求任意 `/admin/dict/code/{dictCode}`。
5. 请求稳定返回 403/30006；重新登录、等待缓存过期或切换字典 code 均不能恢复。

## 2. 任务开始前工作区

任务开始前工作区已经存在 Report、Frontend、design、migrate、model、repository 等未提交改动。本 Bug 不覆盖、不回滚、不清理这些改动，提交时只精确暂存本 Bug 文件。基线文件清单以任务开始时的 `git status --short` 输出为准。

## 3. 30006 的定义与产生位置

### 3.1 错误定义

- `backend/internal/errors/errors.go`
- `ErrPermissionDenied = NewError(http.StatusForbidden, 30006, "无权限访问")`

### 3.2 全部产生路径

`backend/middleware/casbin.go` 有两条 30006 返回路径：

1. 严格策略覆盖开启，且请求的 `object + action` 在 Enforcer 中不存在任何 policy。
2. 请求存在 policy，但当前用户的全部 Casbin subject 对全部候选 object 执行 `Enforce` 均为 `false`。

当前事故命中第一条：`/admin/dict/code/:code + GET` 的策略覆盖被角色权限保存删除。

仓库中其他返回同一错误码的路径如下：

| 层 | 文件 | 语义 |
| --- | --- | --- |
| Middleware | `backend/middleware/casbin.go` | Casbin 策略缺失或全部 subject 执行拒绝 |
| Controller | `backend/controller/sys_table_controller.go` | 元数据、菜单、按钮和表级功能权限拒绝 |
| Controller | `backend/controller/generalization_controller.go` | 低代码通用页面功能或数据范围拒绝 |
| Controller | `backend/controller/file_controller.go` | 文件访问权限拒绝 |
| Service | `backend/service/sys_menu_service.go` | 菜单访问上下文无法授权 |
| Service | `backend/service/data_permission_service.go` | 旧数据权限运行链路拒绝 |
| Service | `backend/service/report_service.go` | Report 功能权限拒绝 |
| Service | `backend/service/file_service.go` | 文件复用权限拒绝 |

本次请求在进入 Controller 前即由 `CasbinHandler` 终止，不命中上述业务层路径。

## 4. 认证与授权顺序

管理端中间件顺序为：

```text
CorsHandler
  -> LogHandler
  -> ResponseHandler
  -> adminGroup.AuthHandler
  -> adminGroup.CasbinHandler
  -> Controller
```

`AuthHandler` 完成 JWT 解析、Token 类型检查、用户查询及密码修改时间检查，然后将 `model.SysUser` 和 user ID 写入当前 Gin Context。`CasbinHandler` 随后从同一 Context 读取用户和角色。

事故请求的访问日志均记录：

- `user_id = 1`
- `user_name = admin`
- HTTP 状态为 403，而不是 401
- 同批次 `/admin/table/code/:code` 请求成功

结论：JWT 解析和身份认证成功，问题属于功能授权，不是“登录状态丢失”。

## 5. 成功与失败请求对比

在同一个 backend 容器、同一个用户、同一个 URL `/sweet_admin/admin/dict/code/whether` 下：

| 时间 | 结果 | user_id | 路由模板 |
| --- | --- | --- | --- |
| 2026-07-27 11:06:49 | 200 | 1 | `/admin/dict/code/:code` |
| 2026-07-27 11:19:49 | 403/30006 | 1 | `/admin/dict/code/:code` |

关键事件：

| 时间 | 事件 |
| --- | --- |
| 2026-07-27 11:19:47 | `POST /admin/role/assign-permissions`，`role_id=1`，返回 200 |
| 2026-07-27 11:19:49 | 同一进程首次出现 `/admin/dict/code/:code` 的 403/30006 |

当前数据库中：

- `/admin/table/code/:code + GET` 仍有 `super_admin` policy。
- `/admin/dict/code/:code + GET` 已无任何 policy。
- `sys_menu_button` 没有对应 `/admin/dict/code/:code + GET` 的 API-only 按钮定义。

当前 25 条 30006 访问日志全部归一到 `/admin/dict/code/:code + GET`，没有发现 subject、method、尾斜杠或动态 path 模板随机变化。

## 6. 最终根因

平台同时存在两类 Casbin 策略来源，但角色权限保存只认识其中一类：

1. `seedSuperAdminRoutePolicies` 直接写入的基础路由 policy；
2. `sys_menu_button.path + method` 生成的可授权 policy。

`/admin/dict/code/:code + GET` 只存在于第一类，没有对应 `sys_menu_button`。

`SysRoleService.AssignPermissions` 保存角色权限时会：

1. `RemoveFilteredPolicy(0, role.Name)` 删除该角色的全部 Casbin policy；
2. 仅遍历本次选中的 `sys_menu_button` 重建 policy。

因此保存 `super_admin` 权限后，字典按 code 查询的基础 policy 被永久删除且无法重建。之后即使不再修改角色权限，同一登录会话访问依赖字典元数据的页面仍会持续出现 403。页面同时请求有覆盖和无覆盖的接口，使现象看起来像“随机接口 403”，实际授权结果由路由模板确定。

## 7. 并发、缓存、多实例与共享状态审计

### 7.1 Enforcer 并发

修复前应用使用全局单例 `*casbin.Enforcer`，请求侧执行 `GetFilteredPolicy`/`Enforce`，角色、菜单和 Report 侧可执行 Add/Remove。普通 Enforcer 不提供并发读写保护；此外角色更新采用“先全删、再逐条添加”，存在空策略或部分策略可见窗口。

修复前执行：

```text
go test -race ./middleware -run TestCasbinHandlerConcurrentUnrelatedPolicyUpdatesKeepStableDecision -count=1
```

race detector 稳定报告 Casbin `GetFilteredPolicy`/`Enforce` 与 `AddPolicy`/`RemovePolicy` 对共享 model 的并发读写。该风险不是本次策略永久丢失的必要条件，但会在权限写入期间制造真正的随机判定，因此同属本 Bug 的最小修复范围。

### 7.2 请求级共享状态

当前 user、user ID 和 JWT claims 均为局部变量或 Gin Context 数据。未发现将 current user、role、subject、path 或 method 写入包级可变变量，也未发现相关 `sync.Pool` 复用。

### 7.3 用户/角色缓存

权限中间件没有独立的 Redis 或本地 permission cache。认证使用 Redis `SysUserCache`，当前 `USER_CACHE_KEY_1` 中用户 ID、用户名和 `super_admin` 角色稳定。

删除该用户缓存前后的对照结果均为 403，重新加载后的角色仍为 `super_admin`：

```text
before_cache_delete=403
after_cache_delete=403
reloaded_roles=super_admin
```

结论：缓存不是本次 403 的根因，也不存在可关闭的独立权限缓存。

### 7.4 多实例

开发环境端口 9009 仅由 Docker 端口转发监听，Docker Compose 中只有一个 backend 容器 `sweet-auth-base-backend-1`。成功和失败请求均进入该容器并写入同一 PostgreSQL `access_log`。未发现 IDE、`go run` 或旧二进制同时监听该端口。

## 8. Casbin 输入审计

- subject：角色名 `super_admin`，并保留用户名 `admin` 作为兼容 subject。
- object：优先使用 Gin `FullPath`，同时兼容去除应用名前缀后的 `/admin/...`。
- action：来自 HTTP method，当前请求均为大写 `GET`。
- matcher：精确匹配 subject、object、action。
- 多角色：遍历全部角色 subject，任一 subject 允许即放行；未使用 `First` 随机选取角色。
- query string 不进入 object。
- 当前失败与 path 尾斜杠、method 大小写、空 FullPath 无关。

## 9. 修复方案

最终采用以下最小修复：

1. 将 `/admin/dict/code/:code + GET` 纳入 `sys_menu_button` 的 API-only 权限来源，消除“Seed 直接 policy”与“角色可重建 policy”之间的所有权缺口。
2. Seed 幂等补齐现有 `super_admin` 的按钮授权和 Casbin policy。
3. 角色权限保存后该 policy 仍存在。
4. 全局 Enforcer 改为 `casbin.SyncedEnforcer`，统一保护请求侧读取和运行期策略写入。
5. 角色策略更新使用 `UpdateFilteredPolicies` 单次替换，替代“全删后逐条添加”；GORM adapter 在同一数据库事务内删除旧策略并写入新策略。
6. Load/替换失败时保留旧内存 policy，不得切换为空策略。
7. 增加由 `SWEET_AUTHZ_DIAGNOSTICS=true` 控制的结构化授权诊断。成功和失败均记录请求关联、进程实例、用户和角色、subject/object/action、策略数量、判定阶段和错误码，不记录 Token、密码或请求体。

## 10. 未采用方案

- 不在 403 后重试授权。
- 不对 `super_admin` 做硬编码放行。
- 不关闭严格策略覆盖。
- 不将全部管理接口加入白名单。
- 不永久关闭缓存。
- 不在每次请求执行 `LoadPolicy`。
- 不仅把 `/admin/dict/code/:code` 加入匿名或无条件放行列表，因为该接口仍应服从现有功能权限模型。

## 11. 修复前后验证

### 11.1 修复前

- 角色保存回归测试失败：`seeded dictionary code policy was lost after role permission assignment`。
- Casbin 并发读写测试在 race detector 下失败，定位到共享 model 的读写竞争。
- 同一实例、同一 user、同一 URL 在角色保存前 200、保存后稳定 403/30006。

### 11.2 修复后

- 同一用户同一接口连续 1000 次结果一致。
- 两用户、允许/拒绝接口并发 1000 次无串扰。
- 多角色用户连续 1000 次按“任一有效角色允许”稳定通过。
- 策略写入与 Enforce 并发在 race detector 下通过。
- 动态路由、query string、method 和尾斜杠边界结果稳定。
- 角色保存后 `/admin/dict/code/:code + GET` policy 保留。
- Seed 连续执行两次，API-only 按钮、角色按钮关联和 Casbin policy 各保持一条。
- 诊断日志测试确认允许和拒绝记录具有相同字段集，差异为 `decision_stage`、`enforce_result`、命中 subject/object 和 `error_code`。

## 12. Race detector 边界

权限相关范围通过：

```text
go test -race ./middleware ./initialize ./repository/impl
go test -race ./service -run 'TestAssignPermissions(...)'
go test -race ./migrate -run TestDictionaryCodePermissionSeedIsIdempotentAndRoleAssignable
```

全仓 `go test -race ./...` 已执行，但被两个既有 Organization 测试竞争阻断：

1. `TestOrgControllerEmployeeUserBindingUsesPermissionsAndSafeResponse`：测试复用 Gin Context 时，数据库 Rows 的异步收尾仍读取 request context。
2. `TestOrgServiceBindEmployeeUserConcurrentConflict`：并发 goroutine 内重复调用全局 `gin.SetMode`。

这两处不涉及 Casbin、认证或本次修改，普通后端全量测试通过；按任务边界仅记录，不在本 Bug 中修改 Organization。
