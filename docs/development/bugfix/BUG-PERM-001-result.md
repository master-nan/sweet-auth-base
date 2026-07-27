# BUG-PERM-001 Random Authorization 403 修复结果

状态：根因确认并完成最小修复；权限相关循环、并发、race 和后端全量测试已完成。

## 1. 结论

本问题是功能授权问题，不是认证失败或登录状态丢失。

确定性根因是 `/admin/dict/code/:code + GET` 仅由直接 Seed 创建 Casbin policy，没有对应 `sys_menu_button` 权限元数据。角色权限保存会删除角色全部旧 policy，再仅依据选中的按钮重建，导致该路由 policy 永久丢失。页面并发请求多个不同路由时，只有缺失策略的字典接口失败，因此表现为“随机接口 403”。

同时确认一个独立的并发风险：应用原先共享普通 `*casbin.Enforcer`，请求读和权限写并发时 race detector 能稳定发现数据竞争；角色策略的“先删后逐条添加”也会暴露空策略窗口。

## 2. 30006 产生位置

- 定义：`backend/internal/errors/errors.go` 的 `ErrPermissionDenied`。
- 本事故准确出口：`backend/middleware/casbin.go` 的严格策略覆盖检查。
- 最终为 false 的判断：`hasPolicyForRequest` 对 `/admin/dict/code/:code + GET` 查询不到任何 policy。
- 请求已经通过 `AuthHandler`，Gin Context 中 user ID、token subject 和 `super_admin` 角色稳定。

## 3. 成功与失败差异

| 项目 | 成功请求 | 失败请求 |
| --- | --- | --- |
| backend 实例 | 同一容器 | 同一容器 |
| user_id | 1 | 1 |
| role/subject | `super_admin` | `super_admin` |
| method | GET | GET |
| FullPath/object | `/admin/dict/code/:code` | `/admin/dict/code/:code` |
| 用户缓存 | 同一用户和角色 | 清除重载后仍相同 |
| 路由 policy | 存在 | 角色保存后缺失 |
| 判定阶段 | policy allowed | policy coverage missing |

因此不涉及 JWT 随机解析、用户或角色串扰、权限缓存 key、多个后端实例、query string、动态 path 或 method 大小写漂移。

## 4. 修复内容

1. 新增字典编码查询的 API-only `sys_menu_button` Seed，使所有可重建 Casbin 路由都由统一权限元数据拥有。
2. Seed 保持稳定 code，重复执行不会重复创建按钮、角色授权或 policy。
3. 角色权限保存使用单次 `UpdateFilteredPolicies` 替换该角色策略，不再先清空后逐条添加。
4. 应用运行时改用 `casbin.SyncedEnforcer`，保护 Enforce、查询和 Add/Remove/Replace 的并发访问。
5. `LoadPolicy` 或策略替换失败时保留最后一份有效内存策略。
6. Auth 中间件将 JWT subject 放入当前 Gin Context，仅供当前请求诊断使用。
7. 增加默认关闭的结构化授权诊断开关 `SWEET_AUTHZ_DIAGNOSTICS`。

## 5. 诊断日志

设置 `SWEET_AUTHZ_DIAGNOSTICS=true` 后，每次授权判定记录：

- `request_id`、`trace_id`、timestamp；
- process ID、instance ID、build commit；
- user ID、token subject、role IDs/names、Casbin subjects；
- method、URL path、Gin FullPath、Casbin objects；
- 实际命中的 subject/object/action、Enforce result、判定阶段；
- policy count；当前没有 policy version，明确记录 `not_available`；
- 当前没有独立权限缓存，明确记录 `not_configured/not_applicable`；
- 拒绝时记录错误码 30006。

日志不记录完整 Token、密码、敏感请求体或凭证。

## 6. 测试结果

### 6.1 修复前证据

- 角色保存测试稳定复现字典 route policy 丢失。
- Casbin 并发读写测试在 race detector 下稳定报告数据竞争。

### 6.2 修复后权限回归

- 同一用户同一接口连续 1000 次一致。
- 多接口、多用户、允许和拒绝并发无串扰。
- 多角色用户连续 1000 次结果一致。
- 策略更新与 Enforce 并发通过 race detector。
- 动态路由、query string、method 和尾斜杠边界稳定。
- policy reload 失败保留旧策略。
- 字典 API-only 权限 Seed 连续执行两次保持幂等。
- 成功和失败结构化诊断字段完整，且不包含 Token。

### 6.3 命令结果

```text
cd backend && go test ./...
PASS

cd backend && go test -race ./middleware ./initialize ./repository/impl
PASS

cd backend && go test -race ./service -run 'TestAssignPermissions(...)'
PASS

cd backend && go test -race ./migrate -run TestDictionaryCodePermissionSeedIsIdempotentAndRoleAssignable
PASS
```

全仓 `go test -race ./...` 已执行；权限相关包全部通过，但命令整体被两个既有 Organization 测试竞争阻断。问题位于员工账号绑定测试的 Gin Context 生命周期和并发 `gin.SetMode`，不涉及本次权限根因，按任务边界记录为遗留。

前端未修改，因此未执行前端测试。

## 7. 影响与边界

- 不改变 JWT、角色、菜单、按钮或 Casbin 模型。
- 不引入数据权限、ABAC 或新的权限缓存。
- 不修改 Organization、Report、低代码或前端业务逻辑。
- 当前开发环境是单 backend 实例；多实例策略广播仍需未来平台能力支持，不是本次单实例事故根因。
- 角色授权数据库事务与 Casbin 内存替换仍是两个顺序边界；本次保证 Casbin adapter 内的 policy replacement 原子且失败保留旧内存策略，未扩展为分布式事务。

## 8. 人工验证

1. 启动包含本修复的单 backend 实例，确认启动 Seed 成功。
2. 登录 `admin`，打开依赖字典元数据的菜单，确认 `/admin/dict/code/{code}` 返回 200。
3. 保存 `super_admin` 角色权限，不改变字典菜单授权。
4. 连续请求 `/admin/dict/code/whether` 1000 次，确认全部为 200。
5. 并发打开用户、菜单、角色、字典和组织页面，确认有权限接口不再出现 30006。
6. 临时设置 `SWEET_AUTHZ_DIAGNOSTICS=true`，分别访问一个允许和一个无权限接口，按 request ID 对比 subject/object/action、policy count 和判定阶段。
7. 完成验证后关闭诊断开关，避免长期输出高频授权明细。
