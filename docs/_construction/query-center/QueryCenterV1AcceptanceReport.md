# Sweet Platform Query Center V1 验收报告

> Lifecycle: construction
>
> Final Action: DELETE_AFTER_STABLE
>
> Acceptance Baseline: `55e4f9fa9ba865084d24e888d37321511946b717`

## 1. 验收结论

Query Center V1 后端安全边界、18 个 Eligible 页面接入、权限隔离、Data Permission AND 语义、失效方案处理、UX、PostgreSQL 约束、Race 和浏览器验收均通过。P0=0，P1=0，允许冻结。

Query Center 只保存和解析当前页面的查询方法，不代理业务列表查询，不扩大页面权限或数据权限。Report、跨表查询、SQL、分享/收藏等能力保持在 V1 范围之外。

## 2. Scope 接入矩阵

| Scope | 页面 | Table Code | 状态 | 特殊处理与结论 |
| --- | --- | --- | --- | --- |
| `system.application.list` | `system/application/Index.vue` | `system_application` | ENABLE | FE Reference；保留应用状态与业务按钮 |
| `system.user.list` | `system/user/Index.vue` | `system_user` | ENABLE | 保留账号、角色和认证语义 |
| `system.role.list` | `system/role/Index.vue` | `system_role` | ENABLE | 保留菜单/按钮授权工作流 |
| `system.sms.list` | `system/sms/Index.vue` | `system_sms` | ENABLE | 保留短信领域状态与发送能力 |
| `system.audit.list` | `system/audit/Index.vue` | `system_audit` | ENABLE | 只读列表，方案不改变审计访问边界 |
| `organization.employee.list` | `organization/employee/Index.vue` | `organization_employee` | ENABLE | 保留 Employee/User、任职和 HR 边界 |
| `organization.position.list` | `organization/position/Index.vue` | `organization_position` | ENABLE | FE Reference |
| `organization.sync_batch.list` | `organization/sync-batch/Index.vue` | `organization_sync_batch` | ENABLE | 只读诊断列表，保留 Sync 语义 |
| `organization.sync_error.list` | `organization/sync-error/Index.vue` | `organization_sync_error` | ENABLE | 只读异常列表，保留安全摘要 |
| `integration.external_system.list` | `integration/external-system/Index.vue` | `integration_external_system` | ENABLE | QC Reference；浏览器深度验收页 |
| `integration.interface_definition.list` | `integration/interface-definition/Index.vue` | `integration_interface_definition` | ENABLE | 保留 Interface 配置语义 |
| `integration.credential.list` | `integration/credential/Index.vue` | `integration_credential` | ENABLE | 不展示 Credential secret |
| `integration.retry_policy.list` | `integration/retry-policy/Index.vue` | `integration_retry_policy` | ENABLE | QC Reference；Retry 状态机未改变 |
| `integration.sync_task.list` | `integration/sync-task/Index.vue` | `integration_sync_task` | ENABLE | 保留 Runner/Sync Business Actions |
| `integration.sync_batch.list` | `integration/sync-batch/Index.vue` | `integration_sync_batch` | ENABLE | 只读运行诊断与业务动作分离 |
| `integration.execution.list` | `integration/execution/Index.vue` | `integration_execution` | ENABLE | Execution/Attempt 权限继续隔离 |
| `integration.log.list` | `integration/log/Index.vue` | `integration_log` | ENABLE | 只读日志，不暴露请求秘密 |
| `develop.dictionary.master` | `develop/dictionary/Index.vue` | `sys_dictionary` | ENABLE | 只接主表查询；复用 Runtime Metadata/Dict |

最终统计：Scope=18，ENABLE=18，PARTIAL=0，EXEMPT=0。Generalization 未成为固定 V1 scope，Report 为 `REPORT_DEFERRED`。

## 3. 前端接入与 UX

- 18 个页面统一复用 `useTableQueryState`、`useQueryScope`、`useQuerySchemes`、`useQuerySchemePage`、`QuerySchemeSelector`、`QuerySchemeSaveDialog`、`AdvancedQuery` 和 `StandardTableToolbar`。
- 业务页面 Advanced Query 使用“重置 / 搜索”；Scheme 条件编辑使用“取消 / 确定”。Simple 仅编辑单 AND Group；复杂结构不能压平到 Simple；Schema 第三层无损只读。
- 默认只在首次初始化或明确恢复默认时应用，优先级为 PERSONAL > PAGE_DEFAULT > 页面原始 Query。Refresh、分页和返回列表不会重新应用默认。
- Quick、Advanced、Sorting 和 Preset 参与 Dirty；page、pageSize、列偏好不参与。Dirty 切换提供取消/继续选择，取消后原条件完整保留。
- Selector 只显示有数据的分组，长名称使用受限宽度、ellipsis 和完整 title。Manager 使用明确“使用方案”文字动作，低频能力进入 More Menu，revision 不显示。
- DEGRADED/INVALID 不应用业务 Query；分别显示安全的“需要修复/不可用”反馈，无强制运行入口。
- 删除正在使用的 PERSONAL 不回写业务 Query；现有页面状态继续作为临时条件，重新进入后按默认优先级初始化。

## 4. 权限与安全验收

| 场景 | 结果 |
| --- | --- |
| Admin | 18 页面 Selector/Advanced/Save/Refresh/Business Action 可用；Manager 四 Tab 与共享管理可用 |
| 普通页面用户 | 可使用 PERSONAL/PUBLIC/PAGE_DEFAULT，业务写按钮不出现，平台查询与刷新可用 |
| 只读用户 | 可进入授权页面和 Hidden Manager，不能执行共享管理 |
| 无页面权限用户 | Hidden Manager 只显示自身可管理范围；直接进入未授权业务路由落入 404，未加载 Selector/业务 API |
| Shared Manager | 通过 `query_scheme_shared_manage` 显示新建、编辑、启停和删除；不依赖角色名 |
| PERSONAL 隔离 | 用户 A 的方案对用户 B 不可见；后端 detail/resolve/update/delete IDOR 测试拒绝 |
| ROLE | 匹配角色用户可见，未匹配用户不可见；角色事实变化后后端实时收敛 |
| PAGE_DEFAULT | 无个人默认时自动应用；个人默认覆盖；ROLE/PUBLIC 不参与自动默认 |
| Revision | 双标签同 revision 编辑，后提交稳定提示“方案已被其他操作更新，请刷新后重试”，不展示 revision 数值 |
| Payload | 非法 field/operator/sort/binding、深度/数量/大小限制由后端重新验证 |

## 5. Data Permission E2E

最终固定 scope 没有 Data Permission 演示业务页，因此不增加生产 scope，也不在 Query Scheme Service 中加入页面特例。验收测试通过现有 Generalization + Data Permission 正式查询链执行同一标准化 Scheme 条件 `amount >= 2000`：

- Admin 的 ALL data scope 返回授权范围内两条记录；
- Limited User 的组织范围仅返回东区一条记录；
- 方案条件与 Data Scope 在 Repository 查询前做 AND；方案未保存、返回或替换 `DataScopeResult`。

对应自动化测试位于 `backend/service/data_permission_demo_acceptance_test.go`。

## 6. 浏览器验收

- Admin 实际遍历全部 18 个 ENABLE 页面；每页均能加载唯一 Selector、Advanced Query、Save、Refresh，业务按钮语义未改变。
- External System 完成 PERSONAL 保存、个人默认优先、Dirty、切换确认、PUBLIC/ROLE/PAGE_DEFAULT、DEGRADED、INVALID 和长名称专项。
- 用户 A/B 证明 PERSONAL 与 ROLE 可见性隔离；无页面权限用户证明 Hidden Route 和业务 Route 边界；Shared Manager 证明共享写权限。
- 两个浏览器 Tab 完成 revision 冲突验收。
- 1366 和常用宽屏、亮色与深色均检查 Toolbar、Selector、Advanced Query、Save Dialog、Manager 与 Drawer；无水平溢出。
- 页面 Console Error、Unhandled Promise、Vue Warning、404 API 和误 403 为 0。

## 7. 自动化与构建

| 门禁 | 结果 |
| --- | --- |
| Frontend Unit | 69 files / 269 tests PASS |
| `yarn lint` | PASS |
| `yarn typecheck` | PASS |
| `yarn build` | PASS；保留既有 >900 KiB chunk warning |
| Backend `go test ./... -count=1` | PASS |
| Race | `service repository controller internal/queryscheme initialize` PASS |
| PostgreSQL 16 Query Scheme | Schema、幂等、CHECK、FK、partial unique、并发 default PASS |
| PostgreSQL 16 全仓强制门禁 | `SWEET_REQUIRE_POSTGRES_TESTS=true go test ./... -count=1` PASS |
| `make docs-check` | PASS |

## 8. 非阻塞项

- P2：用户自定义的深蓝 primary 色在深色主题下对比度偏低，属于全局 Theme token 治理，不以 Query Center 页面补丁处理。
- P3：构建仍提示文件预览相关大 chunk；Query Center 页面本身已路由懒加载，不在本 Task 手工拆 vendor。
- 当前 18 个 scope 无 relation/linkage 查询字段；未来启用此类字段前需接入受控 relation display resolver。

P0=0，P1=0。
