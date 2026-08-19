# Sweet Platform Query Center V1 Gap Analysis

> Audience: QC-001 Reviewer、QC-002 实施人员
>
> Lifecycle: construction
>
> Final Action: DELETE_AFTER_STABLE
>
> Removal Gate: QC V1 实现完成，Gap 已关闭或转入正式 Backlog
>
> Audit Baseline: `291f22e231883dfab66ac32db4b40b9444fd3d66`

## 1. 审计范围

本审计直接读取以下当前代码，而非历史 Design：

- `frontend/src/components/Query/AdvancedQuery.vue`
- `frontend/src/components/Query/AdvancedQueryRuleRow.vue`
- `frontend/src/utils/query-state.ts`
- `frontend/src/composables/table-query-state.ts`
- `frontend/src/types/global.ts`
- `frontend/src/utils/field-metadata.ts`
- `frontend/src/composables/runtime-table-metadata.ts`
- `frontend/src/components/Table/StandardTableToolbar.vue`
- `frontend/src/composables/page-buttons.ts`
- `backend/dto/request/basic.go`
- `backend/repository/util/query.go`
- `backend/model/sys.go`
- 当前 Router、Menu DTO、Runtime Metadata 和 FE-003 页面矩阵。

## 2. AdvancedQuery 能力对照

状态含义：

- `EXISTING`：已有并可复用；
- `NEEDS_UI_REFACTOR`：协议/核心能力已有，V1 需收口交互；
- `MISSING`：当前不存在，V1 需新增；
- `NOT_NEEDED`：V1 明确不建设；
- `V1_DEFERRED`：有产品价值但不进入 V1。

| 能力 | 状态 | 真实结论 |
| --- | --- | --- |
| AND / OR | EXISTING | `ExpressionLogic` 支持 AND/OR；同组规则和 nested 复用 group logic |
| 多 Expression Group | EXISTING | Query 顶层支持多个 Group，后端以 AND 组合顶层组 |
| Nested Group 数据结构 | EXISTING | 前后端结构递归；Repository `buildQuery` 递归处理 |
| Nested Group UI | NEEDS_UI_REFACTOR | 当前只完整渲染顶层 + 一层 nested，缺深层编辑和折叠 |
| 字段类型约束 | EXISTING | VARCHAR/TEXT/数值/Boolean/DATE/DATETIME/TIME 映射不同 operator |
| 字典字段 | EXISTING | Runtime Dict 选项，值按字段类型转换 |
| 关系字段 | EXISTING | Linkage config + Generalization Runtime API，含分页/回填已选项 |
| Organization 字段 | EXISTING | Organization selector metadata，EQ/IN，提交内部 ID |
| Boolean | EXISTING | Boolean metadata 和固定“是/否”控件 |
| Range | EXISTING | BETWEEN/NOT_BETWEEN 双值校验 |
| Multi Value | EXISTING | IN/NOT_IN；自由文本支持中英文逗号、分号、换行拆分 |
| NULL 操作符 | EXISTING | IS_NULL/IS_NOT_NULL，不要求 value |
| LIKE 多关键字 | EXISTING | 前端 normalize 为数组；后端 OR/AND 参数化 LIKE |
| Query value normalize | EXISTING | trim、number、boolean、range、multi-value、空规则清理 |
| 完整 canonical hash | MISSING | 无稳定序列化、source baseline 或 Scheme equality |
| Simple Mode | MISSING | 当前只有 Group 卡片高级编辑器 |
| Simple -> Advanced 无损转换 | NEEDS_UI_REFACTOR | 同协议可天然转换，但需模式规则和测试 |
| Advanced -> Simple 安全判定 | MISSING | 需拒绝 OR/nested/多复杂 Group 的有损降级 |
| 条件树 | V1_DEFERRED | 附件有左树；当前 Group 卡片已表达结构，V1 收益不足 |
| 条件预览 | MISSING | V1 必须新增可读摘要 |
| Group 折叠 | MISSING | V1 高级模式新增 |
| 字段说明 | V1_DEFERRED | Runtime Metadata 无可靠 description/help text |
| Dynamic Date operator | MISSING | ExpressionType 无 relative date；用 Scheme binding 解析，不改协议 |
| CURRENT_USER/EMPLOYEE | MISSING | 需服务端 binding resolver，不能保存实际用户 ID |
| SQL/DSL | NOT_NEEDED | 明确禁止 |

## 3. Query State Gap

| 能力 | 当前 | QC V1 |
| --- | --- | --- |
| `query` | EXISTING | 保留 |
| Quick keyword | EXISTING | 纳入 Scheme/Dirty |
| draftAdvanced / appliedAdvanced | EXISTING | 保留 |
| Advanced Apply page=1 | EXISTING | 保留 |
| Sorting allowlist/page=1 | EXISTING | 保留 |
| page/pageSize | EXISTING | 保留，但不持久化 Scheme |
| Refresh 不重置 | EXISTING | 保留 |
| Clear Query | EXISTING | 保留并增加 dirty guard |
| Scheme Source | MISSING | 新增 id/name/type/revision/source status |
| Normalized Baseline | MISSING | 新增 |
| Dirty | MISSING | 新增；quick/advanced/order 参与 |
| Apply Resolved Scheme | MISSING | 新增；写入状态并 page=1 |
| Default Load | MISSING | `useQuerySchemes` 编排，不写入 table state API 层 |
| Binding provenance | MISSING | 新增受控 UI state；表达式结构编辑时安全失效 |
| Saved/default query hook | PREPARED | FE-003 文档已预留，本次正式设计接口 |

不新增 `useQueryCenterState` 或 `useStandardPage`。`useTableQueryState` 继续是标准列表唯一 Query State。

## 4. Runtime Metadata Gap

### 4.1 已有

- `field_code` / `field_name`；
- field/input type；
- dict code；
- quick/advanced search；
- sort；
- list/form/detail visibility 与 sequence/span；
- relation/linkage；
- Organization selector；
- protected field Runtime 过滤；
- Dictionary/Metadata Runtime Read 与 Administration 权限分离。

### 4.2 缺口与决定

| 缺口 | 决定 |
| --- | --- |
| 字段 description/help text | V1 不扩，字段说明面板延期 |
| 业务主时间字段 | 不放 SysTableField；由 Page Query Config `quick_date_field` 显式声明 |
| Query Scope | 新增 `SysMenu.QueryScopeCode`，不是字段 Metadata |
| 业务 Preset | Scope Registry 显式注册，不从字段名猜 |
| Virtual Sort | Scope Config 白名单；Metadata 继续只管实体字段 |
| Dynamic Binding allowlist | Scope Config 声明并由后端验证 |

## 5. 后端 Query 协议审计

### 5.1 已有

- `Query` 对应后端 `request.Basic`；
- ExpressionGroup、Rule、Nested、QuickQuery、Order；
- 参数化 SQL 和字段 Metadata allowlist；
- 敏感字段拒绝；
- 数值/Boolean/日期/时间类型转换；
- IN、LIKE、BETWEEN、NULL；
- Data Permission 条件在独立 Adapter 路径应用；
- 排序只接受真实表字段。

### 5.2 当前风险/缺口

- 基础 `request.Basic` 没有统一 expressions 数量 binding；部分模块 DTO 只限制顶层最多 8；
- 后端递归 nested 没有统一深度/总条件限制；
- 不认识相对日期和当前主体变量；
- 非法字段在通用 Query Builder 中 fail closed 为 `1=0`，但 Scheme 保存需要返回可编辑 issue，而不是只有空结果；
- 没有 Scheme payload JSON Schema/version/size validator；
- 没有 Query Scope、方案可见性、owner/revision/default 语义。

QC V1 在 `internal/queryscheme` 进行更严格的保存/解析验证，不改变现有业务 Query Builder。业务端现有 fail-closed 行为继续作为第二道防线。

## 6. UI 基线对照

| 附件区域 | 结论 | 组件边界 |
| --- | --- | --- |
| 查询方案下拉 | V1 REQUIRED | NEW `QuerySchemeSelector` |
| 快捷时间/业务条件 | V1 REQUIRED | NEW `QueryQuickPresets` + Scope Config |
| 高级查询入口 | EXISTING/EXTEND | Standard Toolbar slot + AdvancedQuery |
| 保存方案 | V1 REQUIRED | NEW Save Dialog；只保存 PERSONAL |
| Simple/Advanced Tabs | V1 REQUIRED | EXTEND AdvancedQuery |
| 左侧条件树 | V1 DEFERRED | 不实现重复结构导航 |
| 条件卡片 | EXISTING | REUSE AdvancedQuery/RuleRow |
| 条件预览 | V1 REQUIRED | NEW `QuerySchemePreview` |
| 字段说明 | V1 DEFERRED | Metadata 无可靠说明 |
| 我的/公共/角色/Page Default 管理 | V1 REQUIRED | NEW Manager Page |
| 收藏星标 | NOT_NEEDED | 产品明确不做收藏 |
| 共享范围下拉 | REFACTOR | PERSONAL Save Dialog 不出现；共享管理页单独维护 |
| 方案 Detail | V1 REQUIRED | Drawer，不新增独立 Route |
| 空状态 | V1 REQUIRED | Manager/Selector 使用统一状态语义 |

## 7. 权限与身份 Gap

### 7.1 现有可复用

- Menu 是 Navigation + Resource Capability Container；
- `Menu.Name` 连接 Route、Page Name、`usePageButtons`；
- MenuButton + Casbin 是 Business Capability；
- hidden Route 不代表授权；
- Runtime Dict/Metadata 已验证“安全基础读取不要求管理权限”的模式。

### 7.2 新增

- `query_scope_code` 作为稳定 Scheme 隔离身份；
- Runtime Scheme Read：scope 页面访问 + owner/public/role/page-default 可见性；
- PERSONAL 写：owner；
- 共享写：单一 `query_scheme_shared_manage`；
- 跨 scope/IDOR 的统一 Service 校验；
- Query Manager hidden Route。

不复用 `menu_id` 作为持久化 Scheme 身份，但 Runtime 授权仍通过 scope 解析到 Menu。

## 8. 数据模型 Gap

当前不存在任何 Query Scheme/Role relation/default/revision 表。V1 新增：

- `query_scheme`；
- `query_scheme_role`；
- `sys_menu.query_scope_code`；
- PostgreSQL partial unique/check/FK/JSONB size；
- 不新增 condition EAV、history、favorite、share、folder、tenant、cache 表。

## 9. API Gap

当前业务页面只能调用业务 Query API和 Runtime Metadata/Dict。V1 需要：

- Runtime Scope Config；
- Runtime Available Summary；
- Runtime Resolve；
- Management Query/Detail；
- PERSONAL CRUD/default；
- shared CRUD/enable；
- copy-to-personal。

Runtime 与 Management 必须分路由/DTO；普通 Selector 不能调用管理全量查询。

## 10. Eligible Matrix 摘要

| 结论 | 页面 |
| --- | --- |
| ENABLE 19 | System Application/User/Role/SMS/Audit；Organization Employee/Position/Sync Batch/Sync Error；Integration 8 个列表；Dictionary Master |
| CONDITIONAL | 已发布 Generalization 页面 |
| NO | Menu、Data Permission、Structure、Dictionary Item、Database、Configure、Detail、Dashboard/Login/404 |
| REPORT_DEFERRED | Report V1/V2 全部页面 |

具体 Scope Code 和原因见 [QueryCenterV1Design.md](QueryCenterV1Design.md#20-eligible-page-matrix)。

## 11. 需要扩展、新增与不做

### KEEP

- Query/Expression 协议；
- RuleRow 字段值控件；
- Runtime Metadata/Dict；
- Page Buttons/Casbin/Data Permission；
- TablePagination、StatusChip、ConfirmDialog。

### EXTEND

- AdvancedQuery；
- StandardTableToolbar slots；
- useTableQueryState；
- query-state normalization；
- SysMenu scope identity；
- Router/Wire/Migration。

### NEW

- Query Scheme backend module；
- Runtime/Management API；
- Selector/Presets/Save/Preview/Manager/Detail；
- Scope Registry；
- Payload Validator/Binding Resolver；
- PostgreSQL tables/constraints；
- hidden manager route。

### NOT NEEDED / DEFERRED

- Query V2、SQL/DSL、跨表；
- 条件 EAV；
- 字段说明、条件树；
- 收藏、分享、角色默认、审批、导入导出；
- column/pageSize preference；
- Redis Cache；
- Report compatibility layer。

## 12. QC-002 Gate

以下问题必须先由 Reviewer 书面确认：

1. 标准 Query 快照 + bindings envelope 是否满足动态值和协议不变；
2. `sys_menu.query_scope_code` 是否接受为稳定身份；
3. Advanced UI 最终支持深度 2 还是 3；
4. 一个 shared manage capability 是否足够；
5. Manager 使用 Hidden Route；
6. 条件树和字段说明延期。

在这六项确认前，状态是 **DESIGN_COMPLETE / IMPLEMENTATION_NOT_APPROVED**。QC-001 的完成不等于允许直接实现。
