# Sweet Platform 前端架构与页面模式指南

本文面向 Sweet Platform 前端开发和维护人员，规定 Vue 3、Quasar 与 TypeScript 前端的长期分层、页面模式和扩展约束。管理员操作请阅读[平台管理员使用手册](../user-guide/PlatformAdministrationGuide.md)，通用模块扩展步骤请阅读[平台扩展开发指南](ExtensionDevelopmentGuide.md)。

## 1. 适用范围

本指南适用于 `frontend/src/` 下的新页面、公共组件、API 服务、Composable、Store、路由和样式。目标是让相同产品形态遵循相同交互和代码边界，同时保留领域页面表达业务语义的能力。

以下能力不由本指南重新设计：

- Quasar 基础组件；
- `AdvancedQuery` 查询表达式协议；
- 后端菜单、按钮和数据权限；
- Metadata Runtime 契约；
- Report Platform 产品架构；
- Query Center 和用户查询方案。

## 2. 总体架构

```text
Route / Menu
     |
     v
Page
  |- Page Metadata
  |- Query State
  |- Page Buttons
  |- Table / Form / Detail Pattern
  `- Domain Renderers and Actions
     |
     +--> Shared Platform UI
     |      |- BaseContent
     |      |- Table Toolbar and Pagination
     |      |- AdvancedQuery
     |      |- DynamicFormDialog / FormDialogShell
     |      |- Detail components
     |      `- Shared status and display helpers
     |
     +--> Runtime Capabilities
     |      |- Metadata
     |      |- Dictionary
     |      `- Permission
     |
     `--> API Service
            `--> boot/axios
```

基本依赖方向是：页面编排公共能力和领域表达，公共能力读取受控 Runtime 事实，所有业务 HTTP 调用通过 API Service。不得让页面重新实现权限、查询协议或 HTTP 基础设施。

## 3. 页面类型

页面先按产品形态选 Pattern，再决定组件组合，不按后端表数量机械生成页面。

| 类型 | 适用场景 | 标准形态 |
| --- | --- | --- |
| 标准列表页 | 单一资源的查询、分页和行操作 | `BaseContent` + `QTable` + 标准 Toolbar + `TablePagination` |
| 树加列表/详情 | 层级资源、主从资源 | `MasterDetailPage` 或领域专用树 + 详情区 |
| 配置管理页 | 多区域配置、字段或规则管理 | 明确分区的配置页面；复杂逻辑不塞入列表 Toolbar |
| 列表加 Dialog 详情 | 快速查看、上下文不需要独立 URL | 列表 + `FormDialogShell`/领域详情 Dialog |
| 独立 Detail 页 | 可链接、内容多、需要分区导航 | 隐藏 Detail Route + `DetailSectionNavigation` + `DetailFieldGrid` |
| 表单页 | 多步骤或需独立 URL 的编辑 | 独立 Route；简单表单仍优先 Dialog |
| Dashboard | 概览与导航 | 领域化布局，不套 CRUD Pattern |
| Runtime/Execution 只读页 | 执行、日志、同步结果 | 只读列表/详情，强调状态和诊断，不提供伪编辑能力 |
| Report 特殊页 | 设计器、运行时、工作台 | 遵守公共安全和基础交互，但由 Report 专项定义产品模式 |

登录页、Dashboard 和 Report 特殊页不强制套标准 CRUD 页面。

## 4. 标准列表页

标准列表页使用以下结构：

```text
BaseContent
  `- QTable
       |- Toolbar
       |    |- Quick Search
       |    |- Search / Refresh
       |    |- Advanced Query
       |    |- Column Visibility
       |    `- Top Buttons
       |- Rows
       |    `- Line Buttons
       `- TablePagination
```

### 4.1 页面职责

页面负责：

- 指定 `table_code`、路由名和领域 API；
- 提供领域虚拟列、列覆盖、状态映射和特殊 slot；
- 连接查询状态、分页、排序和页面按钮；
- 处理领域动作和导航；
- 为加载、无数据、查询无结果、无权限和失败提供不同状态。

页面不得：

- 复制 Runtime Metadata 到另一套页面字段定义；
- 直接拼后端 URL 或直接使用原生 Axios；
- 根据角色名或管理员身份显示按钮；
- 在当前页对后端分页结果做局部排序冒充全量排序；
- 用全局 Loading 遮罩替代局部表格加载状态。

### 4.2 QTable 基线

- 标准列表使用服务端分页，`row-key` 必须稳定；
- `loading` 使用页面或表格局部状态；
- 小屏可用 `$q.screen.lt.md` 切换 `dense`，Toolbar 必须换行；
- 操作列尺寸固定，不因按钮加载或权限变化挤压业务列；
- 分页统一使用 `TablePagination`；Report 内部数据表等特殊 Runtime 场景可保留自己的分页语义。

## 5. Runtime Metadata

### 5.1 Metadata 决定的事实

以下基础事实由 Runtime Metadata 决定：

- `field_code` 和 `field_name`；
- 字段类型和输入类型；
- 默认列表显示、顺序和可排序性；
- 快捷查询和高级查询字段；
- 简单表单和详情字段、顺序、控件与 span；
- 字典、受控关联、联动和 Organization Selector 配置。

页面不得为同一事实再维护一份静态副本。

### 5.2 页面保留的表达能力

Metadata 不是万能动态表格。页面继续负责：

- 业务虚拟列和组合列；
- 状态、类型等领域 renderer；
- `QChip`、链接、菜单和特殊 slot；
- 页面特有 formatter；
- 操作列；
- 为产品语义覆盖 label、对齐、显示与顺序。

例如同一列同时展示编码与名称、负责人名称与标识、状态 Chip，均属于页面表达，不应把 renderer 名称或 CSS 写进数据库。

### 5.3 统一列解析边界

`frontend/src/utils/column-format.ts` 的 `resolveRuntimeColumns` 是唯一 Runtime Column Resolver，不得并存第二套 Runtime Column Builder。

概念输入：

```ts
resolveRuntimeColumns(metadataFields, {
  overrides,
  virtualColumns,
  formatContext,
  includeActions,
})
```

概念输出：

```ts
{
  columns,
  visibleColumns,
  quickSearchFields,
  advancedSearchFields,
  formFields,
  detailFields,
  sortableFields,
}
```

解析优先级固定为：

1. Runtime Metadata 基础列；
2. 页面 Column Overrides；
3. 页面 Virtual Columns；
4. 平台操作列。

`buildTableColumns` 在迁移期可作为兼容入口，但最终必须委托同一解析实现，不能继续成为另一套真值。

### 5.4 列宽

当前 Runtime Metadata 没有列表列宽字段。现有字段类型默认值和页面 Override 已能覆盖正式页面，暂时没有足够价值为列宽增加持久化契约。未来只有在多个 Metadata 页面确需统一默认宽度时，才评审可空、受控正整数 `list_width`；`min-width`、`max-width`、对齐和特殊布局仍由前端管理。

不得把组件名、CSS class、slot 名或 renderer 名称写入 Metadata。

### 5.5 列显示偏好

V1 采用页面会话状态：

- 默认值来自 `defaultVisibleColumns`；
- 用户在当前页面可临时调整；
- 不在每个页面各自写 LocalStorage key；
- 当前不建设后端个人偏好表。

以后如需跨会话偏好，应由统一用户偏好能力提供，并与 Query Center 的默认/保存查询方案分开。

## 6. Toolbar 与查询

### 6.1 Toolbar

标准 Toolbar 只承载稳定布局和插槽：

- Quick Search；
- Search/Refresh；
- Advanced Query；
- Column Visibility；
- Top Buttons；
- 页面扩展 slot。

`StandardTableToolbar` 是标准列表与主从列表的窄职责 Toolbar，不得让它请求数据、解析 Metadata、分发领域动作或成为超级组件。Quasar 的 `row`、`items-center`、`q-gutter-*`、`q-space` 和响应式 class 负责布局。

### 6.2 Quick Search

快捷查询规则固定为：

- 查询结构使用现有 `Query.quick_query.keyword`；
- Enter 立即查询；输入防抖只用于确有即时搜索需求的页面；
- 新查询先将页码重置为 1；
- 清空 Quick Search 不自动清除已应用 Advanced Query；
- “重置全部”才同时清空 Quick、Advanced、排序和页码；
- Quick Search 字段来自 `is_quick_search`，页面不另写字段列表。

### 6.3 Advanced Query

- 保留现有 `AdvancedQuery` 和 `AdvancedQueryRuleRow`；
- 字段由 Runtime Metadata 过滤 `is_advanced_search` 得到；
- 页面只维护 draft 和 applied 两个状态；
- 打开 Dialog 时从 applied 克隆 draft；
- “应用”时清洗表达式、写入 applied、重置页码并查询；
- 关闭未应用的 Dialog 不修改当前查询；
- 特殊 Runtime 页面无 Metadata 时，必须在页面文档中说明受控字段来源。

### 6.4 Query Center 边界

Frontend Consistency 只统一当前查询状态，不实现 Query Center。页面查询状态至少区分：

- quick query；
- draft advanced query；
- applied advanced query；
- order；
- page/page size。

未来 saved query/default query 只通过 Query Center 注入 applied state。页面不得继续增加只能由本页理解的私有查询协议。

`frontend/src/composables/table-query-state.ts` 的 `useTableQueryState` 是标准列表查询状态入口，避免每页复制克隆、计数、应用、重置和页码联动。刷新只重新读取当前查询，不修改查询、分页、排序或列显示；只有明确的清空查询操作才恢复默认状态。

## 7. 分页与排序

- 标准列表统一使用 `TablePagination`；
- 页码从 1 开始，修改页大小时回到第 1 页；
- 查询条件或排序变化时回到第 1 页；
- `QTable` 排序事件转换为后端 `Query.order`；
- 只有 `is_sort=true` 或页面明确受控的虚拟排序字段可触发服务端排序；
- 后端分页结果不得在前端局部排序后宣称为全量顺序。

## 8. 菜单、路由与按钮权限

### 8.1 三类前端事实

页面实现必须区分三类事实：

| 类型 | 来源 | 示例 |
| --- | --- | --- |
| Platform View Action | 页面和平台组件 | 刷新当前视图、返回、分页、排序、列显示、展开/收起、关闭 Dialog |
| Metadata Capability | Runtime Metadata | 字段标题、顺序、默认显示、排序能力、查询字段、表单控件、详情布局 |
| Business Capability | MenuButton + Casbin | 新增、编辑、删除、启停、审核、吊销、执行、同步、轮换、业务导出 |

View Action 不进入 MenuButton，也不作为 Casbin 业务按钮。名称含“刷新”但会改变业务状态或访问外部资源的动作，例如刷新 Token、重建缓存、重新同步、重新拉取和重新执行，仍属于 Business Capability。

### 8.2 导航、路由与能力

Navigation、Route 和 Capability 是三个概念：

- Navigation 决定是否显示在左侧菜单；
- Route 决定页面地址和生命周期；
- Capability 决定能否读取资源或执行业务动作。

当前菜单模型继续承担 Navigation 和父资源 Capability Container。普通详情使用隐藏 Route，不需要独立可见菜单；详情读取仍要求父资源 `detail` capability，后端 API 继续执行 Casbin 校验。隐藏路由不得因“不在菜单里”跳过前端预加载保护。

### 8.3 稳定映射

对菜单页面，以下标识应一致：

```text
menu.name == route.name == defineOptions.name == usePageButtons(pageCode)
```

泛化 Detail/Form、登录、Dashboard 等非菜单页面可以使用独立稳定名称，但必须显式记录来源页面和权限上下文。

### 8.4 页面按钮

- 页面按钮统一通过 `usePageButtons` 获取 `top_buttons`、`line_buttons` 等位置；
- `menuButtonDisplayProps` 统一决定 icon/text/round/`aria-label`；
- 不按角色名、`isAdmin` 或硬编码管理员身份显示按钮；
- `findCapability`、`hasCapability`、`findActionCapability` 和 `hasActionCapability` 统一能力判断，页面不重复手写 `some(button.event_action === ...)`；
- `TOP`、`LINE`、`BOTTOM`、`FORM_TOP`、`FORM_BOTTOM`、`DETAIL_TOP`、`DETAIL_BOTTOM` 是冻结的位置集合；特殊布局使用 slot，不增加技术布局枚举；
- 前端隐藏不是安全边界，后端仍必须执行 Casbin 和数据权限检查。

### 8.5 View Refresh 与详情动作

- 标准列表 Toolbar 和详情页固定提供当前视图刷新；刷新保留查询、分页、排序和列显示；
- 返回、刷新、关闭和取消属于 View Action，不依赖 MenuButton；
- 普通详情业务按钮继续挂在父 Menu 下，通过 `DETAIL_TOP`/`DETAIL_BOTTOM` 显示；
- `FORM_TOP`/`FORM_BOTTOM` 服务表单业务动作，表单关闭/取消仍是 View Action；
- 只有真正出现多 Tab、独立 API 权限、独立生命周期和不同访问范围的详情，才评审 Complex Detail Workspace；不得仅因详情有路由就新增菜单。

### 8.6 通用与领域动作

平台通用动作包括：

```text
create, edit, detail, delete, enable, disable, revoke, version, run
```

公共 helper 提供通用动作匹配、确认和导航；页面只绑定实际支持的动作。Credential rotate、Assignment bind、Execution retry 等领域动作继续留在领域页面，不进入全局 switch。

删除、吊销、停用等危险操作统一使用 `useConfirmDialog`。不得每页维护一个相同的确认 Dialog。

### 8.7 隐藏路由

- 隐藏 Detail Route 只在用户主动导航时加载；
- 页面在加载受保护数据前校验其查询/详情 capability；
- 无权限时显示稳定拒绝状态并停止请求；
- 不能仅依赖“菜单中看不见”。

## 9. 状态显示

状态值属于领域语义，颜色与文案由领域 mapping 提供；渲染形态可以共享。

标准组合：

```text
domain status map
        +
shared StatusChip renderer
```

共享 renderer 负责 `QChip` 的 dense、square、fallback 和可访问文本，不维护一个跨领域“成功=绿、失败=红”的万能字典。未知状态必须显示安全原值或 `-`，不能静默为空。

## 10. Detail 模式

| 模式 | 使用条件 |
| --- | --- |
| Dialog | 内容较短；查看后仍需保留列表上下文；不需要独立链接 |
| Drawer | 需要并排上下文且宽度稳定；当前没有业务页面采用，新增前必须证明比 Dialog 更合适 |
| 独立 Detail Route | 内容多、可分享链接、有多个分区、需要浏览器历史 |

独立详情优先组合：

- `DetailSectionNavigation`：分区导航；
- `DetailFieldGrid`：白名单字段；
- Metadata 可提供 label、sequence、detail span；
- 特殊业务内容通过 slot 或领域组件提供。

独立详情固定提供返回和刷新；业务动作从父 Menu 的 `DETAIL_TOP`/`DETAIL_BOTTOM` 读取。现有 Integration Detail、Organization Sync Batch Detail、Generalization Record Detail 和 System Detail 均可由该模型覆盖，当前不需要新增详情菜单或按钮位置。

详情模式不得直接渲染后端 Model，也不得因打开方式不同绕过详情权限。

## 11. Form 模式

- 简单 Metadata CRUD 使用 `DynamicFormDialog`；
- 稳定的显式 Dialog 外壳使用 `FormDialogShell`；
- 复杂安全流程、跨资源配置和多步骤业务使用显式 Form Component；
- 表单校验、提交中状态和错误由表单边界负责；
- 页面负责成功后的列表刷新和 Dialog 关闭；
- 不为强行动态化把业务规则写入 Metadata。

`DynamicFormDialog` 是平台表单能力，应保留并逐步拆分内部可测试逻辑；不得重写成另一套组件。

## 12. Quasar 与平台组件

原则是：Quasar 提供视觉和交互基础，平台组件提供稳定平台语义。

新增公共组件必须满足至少一项：

1. 已有三个以上真实页面出现相同完整模式；
2. 承载明确的平台稳定语义，例如权限按钮、Metadata 表单、Advanced Query；
3. 封装必须统一的安全或可访问性行为。

仅包装 `QCard`、`QInput`、`QSelect` 或间距 class 且没有稳定语义的组件不应新增。优先选择 Composable、utility、slot 或 Quasar 原生能力。

## 13. CSS、Theme 与响应式

### 13.1 样式优先级

1. Quasar utility class；
2. Quasar palette 和 CSS variables；
3. 平台 Theme token；
4. 组件 scoped CSS；
5. 页面 scoped CSS，仅用于页面特有布局。

margin、padding、flex、对齐、常见宽高、基础文字色和基础边框优先使用 Quasar class。复杂 grid、稳定尺寸、滚动区域、设计器画布和领域可视化可以保留 scoped CSS。

### 13.2 颜色与深色模式

- 语义色优先 `primary`、`positive`、`warning`、`negative`、`grey-*`；
- 跨组件主题事实使用 `--app-*` 或 `--q-*` token；
- 页面不得新增与 Theme Store 重复的硬编码明暗配色；
- 深色模式由 Quasar Dark 与平台 token 驱动，不依赖每个页面追加 `body--dark` 补丁；
- 业务图表或设计器的专有颜色可以保留，但必须同时验证深色模式。

### 13.3 响应式

平台以桌面管理后台为主。小屏最低要求：

- Toolbar 可换行；
- Dialog 在 `$q.screen.lt.md` 时可 maximized；
- 表格可 dense 并保持横向滚动；
- 固定格式组件使用稳定 min/max/aspect 约束；
- 不在小屏重新设计另一套后台。

## 14. i18n 决策

当前产品冻结为中文管理平台。`vue-i18n` 继续服务登录、布局、菜单和路由等已有框架级文本；当前不开展全量业务页面翻译，也不宣称产品已经支持完整双语。

规则：

- 新增框架级、菜单级文本继续进入现有 i18n；
- 领域页面可以使用统一中文文案，但不得同一页面一半 key、一半硬编码；
- 后续只有在产品正式确认多语言范围、翻译责任和验收流程后，才启动全量 i18n 专项；
- 不删除现有英文资源，避免破坏当前框架切换能力。

## 15. API 与类型

### 15.1 API 调用

- 页面调用 `frontend/src/api/services/` 中的领域 API；
- 页面不得直接导入原生 Axios、拼固定 API URL 或调用 `instance.get/post`；
- 服务器配置的动态按钮 URL 只能通过受控 Dynamic Action API 边界执行；
- `AdvancedQuery`、`DynamicFormDialog` 和列关联查询通过窄职责 Runtime API Service 读取通用数据；
- API Service 不处理页面通知、Dialog 或路由。

### 15.2 Runtime Read 与管理权限

- 字典展示读取使用 `/admin/runtime/dict/:code`，Metadata 展示读取使用 `/admin/runtime/table/:code`；
- Runtime Read 只要求已认证会话，返回经过白名单投影的字典项或字段事实，不要求 Dictionary/SysTable 管理权限；
- `/admin/dict/*` 和 `/admin/table/*` 继续属于配置管理 API，页面不得为了展示标签或列定义调用它们；
- Runtime Read 不是业务数据授权。页面查询、详情和业务动作仍分别接受 MenuButton、Casbin 和 Data Permission 校验；
- Runtime API 不返回管理审计字段、内部标识、受保护字段或配置秘密。

### 15.3 类型放置

| 类型 | 位置 |
| --- | --- |
| 后端 API Request/Response | 对应 `api/services/<domain>.ts`；过大时放稳定领域 `types.ts` |
| 跨页面领域类型 | `modules/<domain>/types.ts` |
| 平台通用 Query、Response | `types/` |
| 仅组件内部 View Model | 组件文件或同目录 `types.ts` |

同一 API DTO 不得在 Service、页面和组件重复定义。页面内类型只允许表达 UI 临时状态，不重新描述后端契约。

## 16. Composable 与 Store

Composable 用于跨三个以上页面重复、带 Vue 响应式生命周期的行为。优先候选：

- Runtime Metadata 加载；
- Table Query State；
- 通用按钮动作和确认；
- 分页/排序联动。

不得创建 `useEverything` 或让 Composable 同时负责 API、路由、权限、通知和领域状态。

Pinia Store 只保存跨路由、跨页面生命周期的全局状态，例如用户、权限菜单、主题、字典缓存、布局和全局会话状态。查询草稿、Dialog 开关、当前页分页和行选择默认留在页面，不进入 Store。

## 17. Loading、空状态和错误

- 页面首次启动或登录初始化可使用全局 Loading；
- 列表请求使用 `QTable.loading`；
- Dialog、按钮和详情各自使用局部 Loading；
- 并发局部请求不能让整个应用反复闪烁遮罩；
- “首次无数据”“查询无结果”“无权限”“加载失败”必须是四种不同文案/状态；
- 接口失败保留当前页面内容并提供重试入口，不把错误当空数组；
- 客户端只显示后端安全错误，不直接拼技术异常。

## 18. Accessibility

最低规则：

- icon-only 按钮必须有 `aria-label` 和可见于 hover/focus 的 `QTooltip`；
- 表单输入必须有 label 或等价可访问名称；
- Dialog 有明确标题、关闭入口并支持 Escape，强制 persistent 需有业务理由；
- 行操作不能只靠颜色区分；
- 键盘可到达主要操作和查询控件；
- `menuButtonDisplayProps` 应作为动态按钮可访问属性的默认来源。

## 19. 特殊页面族

### 19.1 Organization

Organization 保留领域专用 Pattern：

- 法人和组织结构使用只读树 + 详情；
- Employee 使用列表 + Assignment 领域详情；
- Position 使用领域列表；
- Sync Batch/Error 使用 Runtime 只读诊断 Pattern。

不得把 Organization Tree、Assignment 或同步诊断机械改成普通动态表格。

### 19.2 Integration

Integration 配置页是标准列表页的首选参考族。External System、Interface Definition、Credential、Retry Policy 和 Sync Task 应共享 Metadata、Toolbar、Query、权限按钮和 Dialog Pattern。Sync Batch、Execution 和 Log 使用只读 Runtime Pattern。

### 19.3 System 与 Develop

User、Role、Application、SMS、Audit 和 Dictionary 已使用统一查询、Toolbar、按钮与局部状态能力。Menu、Data Permission 和 Database 属于复杂配置页面，复用适用的公共能力，但不强制套普通列表。

### 19.4 Report

Report 标记为 `REPORT_DEFERRED`。Frontend Consistency 只修复公共安全、Theme、Accessibility 和 API 边界问题，不重写 Report 设计器、运行态、Prototype 或产品模式。

## 20. 公共机制与页面覆盖

### 20.1 已建立的公共能力

1. `resolveRuntimeColumns`：Metadata Base + typed Override + Virtual Column；
2. `useRuntimeTableMetadata`：Metadata 局部加载及字段分组；
3. `useTableQueryState`：Quick/Advanced/Order/Page 的单一状态语义；
4. `StandardTableToolbar`：布局、slot 和固定 View Refresh；
5. `StatusChip`：共享渲染，领域状态映射留在页面；
6. Generalization API 的受控 `/admin/` Runtime Action 边界；
7. Page Capability helper、公共动作分发和统一确认；
8. `table-state.ts` 的无数据、无结果、无权限和错误最低文案。

Integration 配置页、System 标准列表和 Organization Position 是标准参考。它们证明公共 Pattern 可以保留组合列、状态映射、业务按钮和领域 formatter。

### 20.2 页面覆盖规则

1. Integration 配置页使用标准实体列表 Pattern，Execution、Log、Sync Batch 使用只读 Runtime Pattern；
2. Organization 标准列表使用公共机制，组织树、Assignment 和同步诊断保留领域 Pattern；
3. System 标准列表使用公共机制，Menu 和 Data Permission 保留复杂配置 Pattern；
4. Dictionary 使用主从列表 Pattern，Database 和 Generalization 保留复杂 Runtime/配置布局；
5. Login、Dashboard、Change Password 和 404 不套 CRUD Pattern；
6. Report 保持 `REPORT_DEFERRED`。

## 21. 架构保护规则

1. 页面不直接使用 Repository 概念，也不直接拼后端 URL。
2. 业务 HTTP 调用只进入领域 API Service。
3. Metadata 提供基础字段事实，页面保留领域 renderer 和虚拟列。
4. Runtime Column Resolver 只有一个实现。
5. Quick、Advanced、排序和分页共享同一 Query State 语义。
6. 菜单页名、路由名、组件名和页面权限码保持稳定映射。
7. `usePageButtons` 决定动态按钮；前端隐藏不替代后端权限。
8. 危险操作统一确认，icon-only 按钮提供 tooltip 和 `aria-label`。
9. Quasar 提供基础 UI，公共组件只承载稳定平台语义。
10. 局部请求使用局部 Loading；错误不能伪装成空数据。
11. 页面不直接返回或复刻后端 Model，API DTO 只有一个定义位置。
12. Report、Organization Tree 和 Dashboard 不被万能 CRUD Pattern 强行改造。
