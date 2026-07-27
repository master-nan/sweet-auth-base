# Sweet Platform 功能权限来源统一审计

状态：审计完成，存在阻塞性缺口；完成独立修复并通过回归前，不得将当前权限 Seed 视为统一来源基线。

审计日期：2026-07-27

审计基线 Commit：`cdf73df17426face400143a17a02a7294480efd3`

## 1. 审计目标

本次审计确认所有受 Casbin 严格覆盖的管理端路由，是否均由 `sys_menu_button` 中的页面按钮或 API-only 权限项定义，并核对：

1. 所有直接调用 `AddPolicy` / `AddPolicies` 的 Seed 和生产代码。
2. `seedSuperAdminRoutePolicies`。
3. `backend/initialize/router.go` 注册的管理端路由与 HTTP Method。
4. `sys_menu_button.path + method` 的源码 Seed 和当前数据库记录。
5. `casbin_rule` 的当前运行时投影。

本次审计不修改数据权限模型、数据权限配置或数据权限执行链路。

## 2. 统一结论

当前功能权限来源尚未统一，审计结论为“不通过”。

主要原因：

1. `seedSuperAdminRoutePolicies` 直接维护 130 个路由策略，绕过 `sys_menu_button`，形成第二个权限定义来源。
2. 13 个受严格 Casbin 覆盖的源码路由没有对应按钮或 API-only 元数据。
3. 7 个组织权限项已进入按钮元数据和 Casbin，但对应 Router 尚未实现。
4. 当前运行库中另有 1 个已在源码补齐、但尚未重跑 Seed 的字典 API-only 权限缺口。
5. 32 组 `path + method` 被多个按钮元数据拥有；多数属于共享技术接口，但部分不同业务 action 实际无法由当前 Casbin 模型区分。

因此，`casbin_rule` 当前不能仅由角色按钮授权完整、无损地重建。角色权限重新保存时按按钮元数据重建策略，会删除直接 Seed 且无按钮来源的策略，这也是此前权限异常的结构性诱因。

## 3. 审计口径

### 3.1 严格覆盖范围

`backend/initialize/router.go` 中 `/admin` 认证组的中间件顺序为：

```text
AuthHandler
  -> CasbinHandler
  -> Controller
```

当前源码共识别 156 组 `/admin` 路由和 Method。

以下 9 组属于已登录公共接口或由 Controller 做资源级权限判断的复用接口，不纳入“严格 Casbin 路由必须有固定按钮来源”的 A 类统计：

| Path | Method | 原因 |
| --- | --- | --- |
| `/admin/logout` | POST | 已登录公共接口 |
| `/admin/user/me` | GET | 已登录公共接口 |
| `/admin/menu/my` | GET | 已登录公共接口 |
| `/admin/user/password` | POST | 当前账号自助能力 |
| `/admin/generalization/query/code/:code` | POST | 低代码 Controller 按菜单、表和 action 二次校验 |
| `/admin/generalization/detail/code/:code/:id` | GET | 低代码 Controller 按菜单、表和 action 二次校验 |
| `/admin/generalization/create` | POST | 低代码 Controller 按菜单、表和 action 二次校验 |
| `/admin/generalization/update` | PUT | 低代码 Controller 按菜单、表和 action 二次校验 |
| `/admin/generalization/delete` | DELETE | 低代码 Controller 按菜单、表和 action 二次校验 |

扣除上述例外后，严格覆盖路由共 147 组。

`/admin/generalization/batch-delete` 和 `/admin/generalization/export` 当前不在 Controller-scoped 例外中，其权限来源是发布低代码菜单生成的 API-only 按钮，不属于缺口。

### 3.2 两套核对口径

本审计同时使用：

1. **源码口径**：Router、按钮 Seed、低代码按钮模板和直接 policy Seed。
2. **运行库口径**：当前 PostgreSQL 中有效 `sys_menu_button`、`sys_role_menu_button` 和 `casbin_rule`。

当前工作区存在用户此前未提交的 Report、Frontend 和 design 改动。本审计读取当前 Router 和 Seed 以避免漏项，但不修改、回滚或提交这些历史改动。

### 3.3 数量摘要

| 对象 | 数量 | 说明 |
| --- | ---: | --- |
| `/admin` 路由 + Method | 156 | 当前工作区 Router |
| 严格 Casbin 覆盖路由 | 147 | 排除 9 组公共/Controller-scoped 路由 |
| `seedSuperAdminRoutePolicies` 直接策略 | 130 | 全部属于第二来源 |
| 当前库有效按钮元数据路由 | 144 | 按 `path + method` 去重 |
| 当前库 Casbin 路由投影 | 145 | 按 `path + method` 去重 |
| 源码严格路由缺元数据 | 13 | A 类源码缺口 |
| 当前库严格路由缺元数据 | 14 | 额外包含尚未重跑 Seed 的字典接口 |
| 元数据存在但 Router 不存在 | 7 | B 类 |
| 直接 Seed 且源码无按钮来源 | 14 | C 类，含 1 个非严格例外 |
| Path/Method 不一致 | 0 | D 类 |
| 多按钮拥有同一 Path/Method | 32 | E 类 |

## 4. A 类：路由存在、按钮元数据不存在

### 4.1 源码缺口

以下 13 组路由受严格 Casbin 覆盖，但没有 `sys_menu_button` 或 API-only 权限定义：

| 模块 | Path | Method | 当前权限来源 | 风险 | 修复要求 |
| --- | --- | --- | --- | --- | --- |
| 角色 | `/admin/role/menu/:id` | GET | 仅 `seedSuperAdminRoutePolicies` | 角色权限重建后丢失 | 在角色菜单增加 API-only 权限项 |
| 角色 | `/admin/role/menu/buttons/:roleId/:menuId` | GET | 仅 `seedSuperAdminRoutePolicies` | 角色权限重建后丢失 | 在角色菜单增加 API-only 权限项 |
| 文件 | `/admin/file/upload` | POST | 仅 `seedSuperAdminRoutePolicies` | 非超管无法从按钮授权获得能力 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/:id` | GET | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/:id` | DELETE | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/preview-url/:uuid` | GET | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/download-url/:uuid` | GET | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/preview/:uuid` | GET | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/download/:uuid` | GET | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/upload/init` | POST | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/upload/chunk` | POST | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/upload/merge/:upload_id` | POST | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |
| 文件 | `/admin/file/upload/progress/:upload_id` | GET | 仅 `seedSuperAdminRoutePolicies` | 同上 | 建立文件能力 API-only 权限来源 |

文件接口内部已有“文件所有者或超管”和“业务记录 + 菜单 + action + 数据范围”的二次校验。该二次校验不能替代功能权限来源。文件接口属于跨模块平台能力，修复时应使用稳定 API-only 权限元数据，不得重新启用会渲染成页面按钮的旧文件模板，也不得只给 `super_admin` 直接写 policy。

### 4.2 当前运行库额外缺口

当前运行库还缺少：

| Path | Method | 说明 |
| --- | --- | --- |
| `/admin/dict/code/:code` | GET | 源码已在 `develop_dictionary_code_query` 中补为 API-only 权限，但当前数据库尚未重跑对应 Seed，按钮和 policy 投影均未落库 |

该项不是新的源码设计缺口，但部署验证必须包含 Seed 重跑和策略投影重建。

## 5. B 类：按钮元数据存在、路由不存在

以下 7 组组织权限已由 `backend/migrate/org_seed.go` 提前 Seed，但当前 Router、Controller 和 Service 没有对应 API：

| 权限 code | Path | Method | 问题 |
| --- | --- | --- | --- |
| `organization_unit_ancestors` | `/admin/org/unit/:id/ancestors` | GET | 祖先服务尚未实现 |
| `organization_unit_descendants` | `/admin/org/unit/:id/descendants` | GET | 后代服务尚未实现 |
| `organization_sync_batch_query` | `/admin/org/sync-batch/query` | POST | 同步批次 API 尚未实现 |
| `organization_sync_batch_detail` | `/admin/org/sync-batch/:id` | GET | 同步批次 API 尚未实现 |
| `organization_sync_error_query` | `/admin/org/sync-record/query` | POST | 同步记录 API 尚未实现 |
| `organization_sync_error_detail`、`organization_sync_error_view_error` | `/admin/org/sync-record/:id` | GET | 同一路由被两个 action 提前拥有，且 API 尚未实现 |
| `organization_sync_error_retry` | `/admin/org/sync-record/:id/retry` | POST | 同步重试 API 尚未实现 |

处理原则：

1. 未实现的接口不得预先形成可授权的按钮和 Casbin policy。
2. 本次权限修复只应退休这些权限元数据及投影，不得顺手开发祖先、后代或同步 API。
3. 对应 API 在未来独立 Task 实现时，再以同一稳定 code 幂等恢复。

## 6. C 类：直接 Seed policy 且无按钮来源

### 6.1 源码直接调用

Seed 路径中，`seedMenuButton -> seedCasbinPolicy` 是按钮元数据到运行时 policy 的正常投影。

异常来源为：

```text
seedMenusAndRole
  -> seedSuperAdminRoutePolicies
  -> seedCasbinPolicy
```

`seedSuperAdminRoutePolicies` 直接维护 130 组策略，不读取 `sys_role_menu_button`，属于独立权限定义来源。

### 6.2 无按钮来源的 14 组直接策略

| 分类 | 数量 | Path |
| --- | ---: | --- |
| 严格角色接口 | 2 | `/admin/role/menu/:id GET`、`/admin/role/menu/buttons/:roleId/:menuId GET` |
| 严格文件接口 | 11 | A 类所列全部 `/admin/file/*` 管理接口 |
| 已登录公共接口 | 1 | `/admin/user/password POST` |

`/admin/user/password POST` 不需要固定 Casbin policy。保留该直接 policy 没有授权价值，也会使运行库无法由按钮关系完整重建。

低代码详情按钮虽然没有直接保存 API Path，但角色授权 Service 的 `buttonAPIPolicies` 会根据 `detail` action 投影 `/admin/generalization/detail/code/:code/:id GET`。因此该路由属于按钮元数据的隐式来源，不计入 C 类；直接 Seed 的同一路由仍属于重复 ownership。

### 6.3 其他直接策略

剩余 116 组直接策略已有按钮、动态低代码按钮或 `detail` action 隐式来源，但仍与按钮投影形成重复 ownership。它们必须从 `seedSuperAdminRoutePolicies` 移除，统一由角色按钮授权投影生成。

生产运行期还存在以下合法 policy 操作：

1. 角色授权 Service 根据 `sys_role_menu_button` 原子替换角色 policy。
2. Report 发布菜单时根据新建菜单按钮增量投影 policy。
3. 通用 Casbin Repository/Service 提供底层持久化能力。

这些调用必须保持“先有权限元数据，再投影 policy”的顺序；不得被用作新的权限定义来源。

## 7. D 类：Path 或 Method 不一致

本轮未发现 Router 与现有按钮元数据之间同一路径的 Method 集合不一致。

仍需固化以下规范：

1. Path 统一保存 Gin route template，不保存具体资源 ID、query string 或 `RequestURI`。
2. Method 统一大写。
3. 动态路径参数命名必须与 Router 模板一致。
4. 新增、修改或删除 Router 时必须有自动化差异测试。

## 8. E 类：重复 ownership

### 8.1 Seed 层重复来源

116 组已有按钮来源的路由，又被 `seedSuperAdminRoutePolicies` 直接 Seed。这是本次最主要的重复 ownership，必须消除。

### 8.2 元数据层重复 Path/Method

当前库有 32 组 `path + method` 被多个 `sys_menu_button` 拥有：

| Path | Method | Owner 数 | 判断 |
| --- | --- | ---: | --- |
| `/admin/generalization/batch-delete` | DELETE | 3 | 多个发布低代码菜单共享 Controller，允许多 owner |
| `/admin/generalization/create` | POST | 3 | 多个发布低代码菜单共享 Controller，允许多 owner |
| `/admin/generalization/delete` | DELETE | 3 | 多个发布低代码菜单共享 Controller，允许多 owner |
| `/admin/generalization/export` | POST | 3 | 多个发布低代码菜单共享 Controller，允许多 owner |
| `/admin/generalization/query/code/:code` | POST | 3 | 多个发布低代码菜单共享 Controller，允许多 owner |
| `/admin/generalization/update` | PUT | 3 | 多个发布低代码菜单共享 Controller，允许多 owner |
| `/admin/menu` | POST | 3 | `create`、`create_child`、`duplicate` 共用接口；Casbin 无法区分 action |
| `/admin/menu/query` | POST | 2 | 页面查询与角色授权菜单查询共享，允许多 owner |
| `/admin/org/sync-record/:id` | GET | 2 | `detail` 与 `view_error` 共用未实现接口；必须退休，未来拆分权限语义 |
| `/admin/report` | POST | 4 | 新建与复制等操作共享接口；Casbin 无法区分 action |
| `/admin/report/:id` | DELETE | 2 | 多页面共享同一删除接口 |
| `/admin/report/:id` | GET | 6 | 多页面共享详情接口 |
| `/admin/report/:id` | PUT | 2 | 多页面共享更新接口 |
| `/admin/report/:id/export` | POST | 2 | 多页面共享导出接口 |
| `/admin/report/:id/preview` | POST | 3 | 多页面共享预览接口 |
| `/admin/report/:id/publish` | POST | 3 | 多页面共享发布接口 |
| `/admin/report/:id/publish-menu` | DELETE | 2 | 多页面共享取消发布接口 |
| `/admin/report/:id/publish-menu` | POST | 2 | 多页面共享发布菜单接口 |
| `/admin/report/:id/run` | POST | 3 | 多页面共享运行接口 |
| `/admin/report/:id/status` | POST | 2 | 多页面共享状态接口 |
| `/admin/report/:id/versions` | GET | 3 | 多页面共享版本接口 |
| `/admin/report/data-sources` | GET | 3 | 多页面共享数据源接口 |
| `/admin/report/query` | POST | 3 | 多页面共享列表接口 |
| `/admin/report/sql-fields` | POST | 2 | 多页面共享 SQL 字段解析接口 |
| `/admin/role/:id/data-permissions` | GET | 2 | 角色页和数据权限页共享，旧数据权限链路暂保留 |
| `/admin/role/:id/data-permissions` | PUT | 2 | 同上 |
| `/admin/role/query` | POST | 2 | 角色列表与用户角色 options 共享 |
| `/admin/table/code/:code` | GET | 10 | 平台元数据公共查询，允许多 owner |
| `/admin/user/:id/data-permissions` | GET | 2 | 用户页和数据权限页共享，旧数据权限链路暂保留 |
| `/admin/user/:id/data-permissions` | PUT | 2 | 同上 |
| `/admin/user/:id/dimension-values` | GET | 2 | 用户页和数据权限页共享，旧数据权限链路暂保留 |
| `/admin/user/:id/dimension-values` | PUT | 2 | 同上 |

多 owner 本身不一定错误：同一技术接口可被多个菜单授权。但必须承认当前 Casbin 模型只比较 `subject + path + method`，不能区分 `event_action`。

冻结规则：

1. 相同 `path + method` 的多个 owner 如果只是不同页面入口，可以保留。
2. 如果不同 owner 代表不同安全语义，不得假设 Casbin 能区分；必须拆分 API、增加 Controller 内明确校验，或统一为同一 action。
3. `organization_sync_error_detail` 与 `organization_sync_error_view_error` 当前属于无效且高风险的重复 ownership。
4. 本审计不修改 Report 或旧数据权限的共享 ownership，只记录后续专项评审风险。

## 9. Casbin 可重建性判断

当前角色授权 Service 已按以下链路重建角色策略：

```text
sys_role_menu_button
  -> 读取有效 sys_menu_button.path + method
  -> 去重
  -> ReplaceSubjectPolicies
```

该运行时链路方向正确，但 Seed 仍保留直接 policy 清单，导致：

1. 首次 Seed 后 `super_admin` 拥有直接策略。
2. 角色权限保存后，直接且无按钮来源的策略被删除。
3. 不同角色或不同时间点的 policy 来源不一致。
4. `casbin_rule` 无法证明是按钮授权的纯投影。

修复后必须满足：

1. 任意角色的 `p` policy 都可由有效角色按钮授权完整重建。
2. 重建过程对 `path + method` 去重。
3. 没有按钮来源的旧 policy 被清理。
4. policy 重建失败时不得留下部分结果。
5. 数据权限相关表和执行逻辑不参与本次重建。

## 10. 修复分组

审计文档提交后，代码修复必须使用独立 Commit。

### P0：本次必须修复

1. 删除 `seedSuperAdminRoutePolicies` 作为直接权限定义来源。
2. 增加角色菜单两个 API-only 权限项。
3. 为 11 个严格文件接口建立可授权的 API-only 权限来源，保留文件自身的所有者/业务数据二次校验。
4. 退休 7 组未实现组织路由对应的提前 Seed 权限及 Casbin 投影。
5. Seed 结束时从有效 `sys_role_menu_button` 重建角色 Casbin policy，清理无来源旧 policy。
6. 增加 Router、按钮元数据和 policy 投影一致性测试。

### P1：后续专项处理

1. 对 `menu/create_child/duplicate` 和 Report 多 action 共用 Path/Method 的权限粒度进行专项评审。
2. 文件与附件能力进入独立平台 Capability 开发时，复核 API-only 权限的菜单归属和非超管授权体验。
3. 组织祖先、后代和同步 API 实现时，按稳定 code 恢复对应权限项。

## 11. 必须冻结的统一原则

1. `sys_menu_button` 是功能权限定义的唯一来源。
2. 有页面操作的接口使用页面按钮元数据。
3. 无页面按钮但受严格 Casbin 覆盖的接口使用 `is_button=false` 的 API-only 权限项。
4. `casbin_rule` 只是角色按钮授权的运行时投影，可完整重建，不是配置源。
5. 禁止新增只调用 `AddPolicy` / `AddPolicies`、却没有按钮元数据来源的接口 Seed。
6. 角色授权、菜单发布和模块 Seed 都必须先持久化权限元数据，再投影 Casbin policy。
7. Router、按钮 Path/Method 和 Casbin 投影必须使用同一规范化 route template 与大写 Method。
8. 公共已登录接口和 Controller-scoped 复用接口必须显式列入例外清单，不得依赖“恰好没有 policy”实现放行。
9. 功能权限不得混入数据范围；本次修复不得改变任何数据权限行为。
10. 未实现的 Router 不得提前 Seed 可授权按钮和 Casbin policy。

## 12. 验收标准

独立修复完成后必须满足：

1. 所有严格覆盖 Router 均至少有一个有效 `sys_menu_button` owner。
2. 所有有效、带 `path + method` 的按钮均能匹配真实 Router，明确登记的共享/未来项除外；本次不得保留未实现组织项。
3. 不再存在 `seedSuperAdminRoutePolicies` 或等价硬编码全路由直接 policy 清单。
4. 所有角色 policy 可从 `sys_role_menu_button` 完整重建。
5. 角色权限保存前后，未变更授权的接口结果保持一致。
6. Seed 连续执行两次，按钮、角色按钮关系和 policy 均不重复。
7. 路由与按钮元数据差异测试通过。
8. `cd backend && go test ./...` 通过。
9. 不修改数据权限、Report、Frontend 或其他历史未提交改动。

## 13. 最终审计回答

1. **是否所有严格管理端路由都有按钮或 API-only 来源？** 否。源码缺 13 组，当前运行库缺 14 组。
2. **`seedSuperAdminRoutePolicies` 是否符合统一来源原则？** 否。它是独立的第二权限来源。
3. **当前 Casbin policy 是否可由按钮授权完整重建？** 否。14 组直接 Seed 策略在源码中没有按钮来源，其中 13 组属于严格路由。
4. **是否存在元数据先于 Router 的情况？** 是。组织模块有 7 组。
5. **是否存在 Path/Method 不一致？** 本轮未发现。
6. **是否存在重复 ownership？** 是。Seed 层有 116 组双重来源，元数据层有 32 组多 owner。
7. **是否允许继续新增只直接 Seed Casbin policy 的接口？** 不允许。
8. **是否修改数据权限？** 不修改，本审计及后续最小修复均保持数据权限链路不变。
