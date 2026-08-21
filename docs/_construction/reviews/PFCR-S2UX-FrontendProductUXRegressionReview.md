# PFCR-S2UX Frontend Product UX Regression Review

## 1. 审计基线与任务边界

- 审计基线：`ea025fe147f2b3fc35b8f40fb560365cb5d3114d`
- 审计开始时工作区：干净
- 性质：Product UX Regression Correction
- 范围：修复 Query Center 接入造成的查询语义和页面形态回归，不新增查询协议、Metadata 能力或业务功能。
- 暂停项：PFCR-S3、Report、HR Production、Editable Grid、Section、Unit、CI/TLS/Migration Ledger。

本报告在生产代码修改前建立。实施过程中若真实代码或浏览器结果推翻初始判断，将同步修订本报告，不保留与最终实现不一致的预设结论。

## 2. 为什么旧 18/18 结论错误

旧结论把“具备 Query Scheme 后端 Scope”误等同于“适合标准列表 Toolbar”。当前真实页面至少包含标准列表、配置工作台、诊断工作台和 Master-Detail 四种产品形态。平台应统一 Query、Metadata、Capability、Pagination 等机制，但不能据此抹平页面结构。

已确认的回归包括：

1. `submitQuickSearch()`清空 Advanced expressions 和 bindings，使关键词与高级条件无法 AND 组合。
2. 7 个页面共 10 个顶部字段筛选游离在 Scheme Payload 外，导致页面真实请求不等于保存方案。
3. Dictionary 的 372px 主栏被塞入完整 Scheme Toolbar，破坏 Master-Detail 工作台。
4. Execution 在两块状态 Card 后叠加 `fit` Table，存在可用高度溢出。
5. Toolbar 永久陈列 Selector、Advanced、Save 等入口，查询方案的辅助能力抢占业务操作空间。
6. AdvancedQuery 字段选项暴露 field code、dictionary code 等 Metadata 调试信息。

## 3. 页面重新分类矩阵

| Scope | 页面 | 页面类型 | 初始结论 | 必要整改 / 特殊边界 |
|---|---|---|---|---|
| `system.application.list` | System Application | STANDARD_LIST | ENABLE | 保留密钥等业务动作 |
| `system.user.list` | System User | STANDARD_LIST | ENABLE | 角色、密码、账号动作不进入查询方案 |
| `system.role.list` | System Role | CONFIG_WORKBENCH | ENABLE | 列表查询适用，权限配置保持领域 Dialog |
| `system.sms.list` | System SMS | STANDARD_LIST | ENABLE | 标准列表 |
| `system.audit.list` | System Audit | STANDARD_LIST / READ_ONLY | ENABLE | 只读日志查询 |
| `organization.employee.list` | Organization Employee | STANDARD_LIST + DETAIL | ENABLE | `only_effective`为页面 View Mode，不猜 HR 主职 |
| `organization.position.list` | Organization Position | STANDARD_LIST + DETAIL | ENABLE | `only_effective`为页面 View Mode |
| `organization.sync_batch.list` | Organization Sync Batch | DIAGNOSTIC_WORKSPACE | PARTIAL | Scheme 只作用批次列表，诊断布局保持专属 |
| `organization.sync_error.list` | Organization Sync Error | DIAGNOSTIC_WORKSPACE | PARTIAL | 固定失败视图保留；路由对象条件转标准 Expression |
| `integration.external_system.list` | Integration External System | STANDARD_LIST | ENABLE | 标准列表 |
| `integration.interface_definition.list` | Integration Interface Definition | STANDARD_LIST | ENABLE | 所属系统移入 Relation Advanced Field；路由上下文转 Expression |
| `integration.credential.list` | Integration Credential | STANDARD_LIST | ENABLE | 所属系统移入 Relation Advanced Field；秘密字段继续排除 |
| `integration.retry_policy.list` | Integration Retry Policy | STANDARD_LIST | ENABLE | 状态、退避方式移入 AdvancedQuery |
| `integration.sync_task.list` | Integration Sync Task | CONFIG_WORKBENCH | ENABLE | 列表查询适用；状态、调度方式移入 AdvancedQuery |
| `integration.sync_batch.list` | Integration Sync Batch | DIAGNOSTIC_WORKSPACE | PARTIAL | 状态、触发方式移入 Advanced；状态机不变 |
| `integration.execution.list` | Integration Execution | DIAGNOSTIC_WORKSPACE | PARTIAL | Worker 状态改紧凑状态条；状态移入 Advanced |
| `integration.log.list` | Integration Log | DIAGNOSTIC_WORKSPACE | PARTIAL | 状态和 execution route context 转 Expression；详情直达不属于过滤 |
| `develop.dictionary.master` | Develop Dictionary | MASTER_DETAIL / CONFIG_WORKBENCH | EXEMPT | 恢复双数据域工作台，退出 Query Scope |

初始分类：`ENABLE 12 / PARTIAL 5 / EXEMPT 1`。Dictionary 清理完成后，固定 Query Scope 数量应从 18 收敛为 17。

`PARTIAL`并不表示绕过统一 Query 协议，而是表示列表查询可复用 Query Scheme，但页面保留诊断状态、钻取入口或工作台专属布局。

## 4. Quick / Advanced 统一语义

Query Center 可保存且页面可执行的唯一查询状态为：

- `quick_query.keyword`
- `expressions`
- `order`
- `bindings`

Quick Search 只更新 keyword 并重置 page，不得清空 expressions、bindings、order 或 scheme source。业务请求中的 keyword 与 expressions 按现有后端协议组合执行。

字段级筛选必须进入 Advanced Expression；动态日期和当前身份使用受控 Binding；只有真正的页面 View Mode 可留在 Scheme Payload 之外。Route Context 若属于字段过滤，必须初始化为标准 Expression。

## 5. Scheme Toolbar 最终设计

标准列表保持单行、按语义分组：

`[查询方案] [关键词输入] [搜索] [高级查询/条件数]  |  [业务主操作] [列] [刷新]`

约束：

- 不永久展示独立“保存方案”按钮。
- Selector 菜单底部承载“保存当前查询为方案 / 保存当前方案修改 / 另存为我的方案”、“恢复默认查询”、“管理查询方案”。
- Advanced 保留一个明确入口并显示活动条件数。
- Query Scheme 是查询辅助能力，视觉权重不得高于新增、执行等业务动作。
- Controls 只负责 Scheme UI 组合，可提供标准/紧凑展示差异，但不得成为布局引擎或第二套 Query 状态。

## 6. Fixed Filter 治理

### FIELD_FILTER：移入 AdvancedQuery

- Interface Definition：`external_system_id`
- Credential：`external_system_id`
- Retry Policy：`status`、`backoff_type`
- Sync Task：`status`、`schedule_type`
- Integration Sync Batch：`status`、`trigger_type`
- Execution：`status`
- Integration Log：`status`

Interface Definition 和 Credential 的 `external_system_id` 必须使用现有 Relation Display Contract，展示系统名称而非 raw ID。Sync Task / Sync Batch 的枚举字段必须提供业务 label，不能把 `none/cron/manual/scheduled` 当普通用户值展示。

### VIEW_MODE：允许保留

- Employee / Position：`only_effective`表示“当前有效档案”视图语义。
- Organization Sync Error：固定失败记录视图属于诊断页边界。

### ROUTE_CONTEXT：转换为 Expression

- Interface Definition / Credential：`external_system_id`
- Integration Log：优先使用稳定的 `execution_id`；历史 `execution_no` 链接仅退回关键词查询
- Organization Sync Error：`object_type`、`local_id`

`log_id`仅用于打开详情，不是列表查询条件。

当前 18 个 Scope 均没有有效 Quick Preset 配置，本轮不为视觉完整度虚构 Preset。

## 7. Dictionary 结论

Dictionary 最终方向为 `EXEMPT`：

- 左侧保持字典类型搜索、新增、列表、分页。
- 右侧保持当前字典上下文、字典项搜索、新增、列表。
- 不在 372px master pane 放 Query Scheme Selector、Advanced Query 和 Save Scheme。
- 保留 Dictionary 原有 MenuButton、权限、Runtime Metadata 和 Master-Detail 结构。

后端清理必须幂等且只影响 `develop.dictionary.master`：清理角色关系，安全停用/软删除已有方案，清空该菜单 Scope 绑定，并确保 Seed 不重新注册；其他 17 个 Scope 不受影响。

## 8. Execution 布局

Execution 保留诊断语义，但把两块大 Card 收为一个紧凑 Runtime Status Strip。页面使用 column/no-wrap，状态条占自然高度，Table 使用 `col` 和 `min-height: 0`，不得再用“额外内容 + fit table”叠加 100% 高度。

必须在 1366x768 和 1920x1080 验证表格底部、Pagination 和 Footer 可见。

## 9. AdvancedQuery 展示边界

普通用户字段选项：

- 主行：业务字段标题。
- 副行：最多展示业务可理解的“字段类型 · 输入控件”。
- 不显示 field code、dictionary code、内部 Metadata code。

Backend query capability 仍是 Operator 安全真值；本轮不修改查询协议、Simple/Advanced 结构、Binding 或 Schema 深度。

## 10. Browser 验收矩阵

实施后必须真实登录并遍历全部 17 个保留 Scope 页面，同时重点验收：

| 页面 | 重点 | 状态 |
|---|---|---|
| Dictionary | Master-Detail恢复、左右搜索、分页 | PASS |
| Position | 标准单行Toolbar、View Mode、Scheme | PASS |
| Interface Definition | Relation Advanced、Route Expression | PASS |
| Retry Policy | 状态/退避方式Advanced、预览label | PASS |
| Execution | 状态条、可用高度、Pagination | PASS |
| 其余12个Scope页面 | Toolbar、Scheme、Advanced、业务按钮、分页 | PASS |

验收覆盖亮色/深色、1366x768/常用宽屏，并检查 Console：0 Error、0 Warning、0 Unhandled Promise、0 意外 403/404。截图仅保存本地开发记录，不进入 Git。

## 11. 测试计划

必须新增或强化真实行为测试：

1. Advanced + Quick AND 组合。
2. Quick Search 不清 expressions / bindings / order / scheme source。
3. Scheme 保存包含当前 Advanced 条件。
4. Route Context 转标准 Expression。
5. Fixed Field Filter 进入 AdvancedQuery。
6. Selector 提供保存入口，Toolbar 不再永久显示 Save。
7. Dictionary EXEMPT 与 Scope 幂等清理。
8. Execution 高度结构守卫。

最终运行：Frontend 全量 test/lint/typecheck/build、Backend 全量测试、release-check、docs-check。

## 12. 实施结果

### 12.1 Quick / Advanced 查询语义

- `useTableQueryState.submitQuickSearch()`现在只更新 keyword 语义并将 page 重置为 1。
- expressions、bindings、order 和 scheme source 均保持不变；页面请求可同时携带 keyword 与 Advanced expressions。
- 删除了无效的 `createEmptyExpressions`页面配置，避免页面重新引入“快捷搜索清空高级查询”的分叉语义。
- Scheme baseline / dirty 继续使用既有 normalize 边界，分页、page size 和列偏好不参与 dirty。

### 12.2 Query Scope 与页面形态

- 最终固定 Scope：17。
- 最终分类：`ENABLE 12 / PARTIAL 5 / EXEMPT 1`。
- Dictionary 从 Registry 和菜单 `query_scope_code` 中退出；幂等迁移只软退役 `develop.dictionary.master` 的已有 Scheme 及其角色关系，不触碰其他 Scope。
- Dictionary 页面恢复 Master-Detail 工作台，不再渲染 Query Scheme、AdvancedQuery 或保存入口。
- Organization Sync Error 保留固定失败诊断视图；Employee / Position 保留“仅当前有效”View Mode。这些均不是可保存字段条件。

### 12.3 Toolbar 与 Scheme 操作

- `StandardTableToolbar`保持单行 no-wrap，查询辅助区、业务动作区和平台动作区仍是独立语义组。
- `QuerySchemeControls`不再永久渲染保存按钮；Advanced 使用单一 tune 图标、tooltip、aria-label 和活动条件数 Badge。
- `QuerySchemeSelector`底部提供保存当前查询、恢复默认查询和管理查询方案；PERSONAL dirty 显示“保存当前方案修改”，共享来源显示“另存为我的方案”。
- Selector 文本使用 ellipsis/title；64 字符/长方案名不会撑坏 Toolbar。活动条件 Badge 预留了稳定空间，不再产生横向溢出。

### 12.4 Fixed Filter 与 Route Context

- 已移除 7 页共 10 个顶部 FIELD_FILTER：Interface Definition、Credential、Retry Policy、Sync Task、Integration Sync Batch、Execution、Integration Log。
- Interface Definition / Credential 的 `external_system_id`改为 Relation Advanced Field；route 参数初始化为标准 EQ expression，并在初始化时保留该显式页面上下文。
- Integration Log 详情跳转改传稳定 `execution_id`，列表将其转为标准 Relation expression；旧 `execution_no` URL 仅作为 keyword 回退，不再形成隐藏 top-level 字段。
- Organization Sync Error 的 route object/local 条件转为标准 expressions。
- Retry Policy 的状态、退避方式以及 Sync Task / Sync Batch 的调度、触发类型均由 Runtime Metadata 和字典提供业务 label。

### 12.5 Runtime Metadata 契约修复

浏览器验收发现 `AllowedOperators []enum.ExpressionType` 的底层类型为 `uint8`，Go JSON 会把切片编码成 Base64 字符串，导致 AdvancedQuery 选字段时报 `map is not a function`。最终 DTO 明确投影为 JSON number array，并增加响应序列化回归测试。

Integration relation metadata 同时补齐：

- canonical relation logical/display type；
- linkage config；
- 受控 relation operator；
- Runtime Relation API 所需 target/value/display contract。

Interface Definition 的“所属系统”浏览器实测仅提供等于、不等于、集合和空值操作符，值控件为 Relation Select，不再接受大小比较或展示 raw ID。字段副说明使用“关联 · 下拉选择”，不暴露 storage ID 语义。

### 12.6 Execution 布局

- 两块大 Worker Card 合并为 58px 左右的紧凑 Runtime Status Strip。
- 页面采用 column/no-wrap，Table 使用 `col` 和 `min-height: 0`，不再使用 `fit` 与顶部内容叠加。
- 1366 与 1920 验收中 Table、Empty State、Pagination 和 Footer 均在视口内，无 body 滚动。

### 12.7 自动化验证

- Frontend：72 个测试文件、243 个测试全部通过。
- `yarn lint`：PASS。
- `yarn typecheck`：PASS。
- `yarn build`：PASS；保留既有大于 900 KiB chunk warning，本 Task 不做依赖拆分。
- Backend `go test ./... -count=1`：PASS。
- PostgreSQL 16 强制门控：PASS，包括 Dictionary Scope 幂等退役、其他 Scope 隔离、Relation Metadata 修复。
- Backend 全量 race：PASS。
- `make release-check`：PASS。
- `make docs-check`：PASS，67 个 Markdown 文件。

### 12.8 浏览器验收

- 真实 admin 登录遍历全部 17 个保留 Scope；Selector、Advanced、业务按钮、列设置、Refresh、Table、Pagination 均可用。
- Dictionary、Position、Interface Definition、Retry Policy、Execution 完成重点交互验收。
- Interface Definition 实测 Relation 字段菜单、受控 operator 与 Relation value control。
- Retry Policy 实测状态/退避方式进入 Advanced，状态值显示“草稿 / 已启用 / 已停用”。
- Selector 实测保存入口、恢复默认与管理入口；Toolbar 中独立保存按钮为 0。
- 亮色、深色、1366x768、1920x1080 均通过；长方案名与 Badge 不溢出。
- 浏览器 Console：0 Error、0 Warning、0 Unhandled Promise；验收期间后端访问日志无 403/404。
- 截图保存在本地 `work/PFCR-S2UX/`，不进入 Git。

### 12.9 未扩大处理的问题

- 全局悬浮设置按钮在窄视口会遮挡最右侧表头/操作区，属于既有全局外观入口，不是 Query Center 页面局部 Patch；记录为 P2，交由后续全局 UX / Theme Review 统一处理。
- Frontend build 仍有大于 900 KiB chunk warning，属于既有 P2 性能治理项。
- Report 继续 `REPORT_DEFERRED`，本轮未修改其产品语义。

结论：本 Task 的 P1 查询语义与页面形态回归已关闭，可进入 PFCR-S3；上述 P2 不阻塞。
