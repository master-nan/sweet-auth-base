# Sweet Platform 前端一致性评审

> Audience: Frontend Consistency 实施、架构评审和项目维护人员
>
> Lifecycle: construction
>
> Final Action: DELETE_AFTER_STABLE
>
> Removal Gate: FE-002、FE-003 完成，最终规则已同步到 `FrontendArchitectureGuide.md`

## 1. 评审结论

FE-001 完成了 Audit 和 Standard Freeze，不代表页面迁移完成。

- 当前前端没有页面直接导入原生 Axios，也没有按角色名或 `isAdmin` 硬编码权限；未发现必须在 FE-001 立即修复的 P0。
- Quasar、`BaseContent`、`TablePagination`、`AdvancedQuery`、`DynamicFormDialog`、`usePageButtons`、Runtime Metadata 等基础能力已经成立，应保留并扩展。
- 一致性问题主要来自页面重复编排：列解析、Toolbar、查询状态、分页、按钮动作、状态显示、Loading/Empty/Error 和样式。
- Integration 配置页是最接近新标准的页面族，但内部仍存在五套相似查询与 Toolbar 代码。
- System/Develop 是历史迁移重点；Organization 必须保留树、Assignment 和同步诊断等领域 Pattern；Report 标记为 `REPORT_DEFERRED`。
- FE-002 应先完善公共能力，FE-003 再按页面族迁移。

审计基线 Commit：`0283a71e38f14c5d8464fbc796b11f41326c6d96`。

## 2. 统计口径

扫描范围为 `frontend/src/` 的受跟踪生产代码和测试文件；页面统计以 `frontend/src/pages/**/*.vue` 为准。名称、调用和重复模式使用精确文本/正则扫描，并对 Router、关键组件和页面族人工复核。

- “正式路由组件”包括静态、隐藏和受控动态路由映射中的唯一 Page Vue 文件。
- “静态 Columns”统计页面内 `columns`/`*Columns*` 的 ref、computed 和常量声明；它是代码声明数，不等同于 QTable 数。
- “硬编码中文”是去除 HTML、块注释和整行注释后，连续两个以上 CJK 字符的候选片段数；它用于判断规模，不等同于最终翻译 key 数。
- CSS 行数同时给出总行和非空行；颜色 token 是词法命中，需在 FE-003 逐页判断是否属于领域颜色。
- Report 文件纳入规模统计，但不纳入普通 CRUD 迁移要求。

## 3. 文件与页面基线

| 指标 | 数量 |
| --- | ---: |
| `frontend/src` 文件 | 245 |
| Vue 文件 | 117 |
| TypeScript 文件 | 125 |
| MJS 文件 | 1 |
| `pages` Vue 文件 | 77 |
| 正式路由 Page 组件 | 38 |
| `Index.vue` | 28 |
| `Detail.vue` | 2 |
| 页面区命名 `*Dialog.vue` | 19 |
| 全前端命名 `*Dialog.vue` | 22 |
| 页面内 `q-dialog` 文件 | 14 |
| 共享 Drawer | 1（Layout 侧栏） |
| 业务页面 Drawer | 0 |

77 个 Page Vue 包含路由页面和页面族内部 Dialog/组件；因此不能把 77 直接理解为 77 个菜单页面。

## 4. 表格、查询和表单基线

| 能力 | 页面文件数 | 补充 |
| --- | ---: | --- |
| QTable | 35 | 全 `src` 为 36，额外 1 个是 `TreeTable` |
| 静态 Columns | 38 | 70 处声明 |
| `usePageButtons` | 22 | 菜单页主要入口 |
| `queryTableByCode` | 16 | 21 次调用 |
| `AdvancedQuery` | 18 | 字段多数来自 Runtime Metadata |
| `DynamicFormDialog` | 11 | 简单 CRUD 主入口 |
| `FormDialogShell` | 14 | 显式 Form/Detail Dialog 外壳 |
| `TablePagination` | 26 | Report 等特殊表仍有独立分页 |
| `BaseContent` | 33 | 标准页面外壳 |
| `useConfirmDialog` | 17 | 危险操作仍未完全覆盖 |
| `visibleColumns` | 13 | 当前均为页面会话状态 |

当前没有页面级列偏好 LocalStorage 或后端用户偏好。`BaseContent` 的 LocalStorage 只保存页面滚动位置，与列偏好无关。

## 5. API 与类型基线

| 指标 | 数量/结论 |
| --- | --- |
| 页面直接导入 `axios` | 0 |
| 页面直接导入 `boot/axios` | 2 |
| 页面硬编码固定 API URL 调用 | 0 |
| 页面受控动态 API path 调用 | 4 处，分布在 2 个泛化页面 |
| 生产 API Service 文件 | 15 |
| API Service 内契约类型定义 | 199 |
| 页面内 type/interface | 311，分布在 60 个页面文件 |

两个直接导入 `boot/axios` 的页面是：

- `pages/detail/RecordDetail.vue`：执行后端菜单按钮提供的动态 action；
- `pages/develop/generalization/Index.vue`：执行泛化页面动态 action/export。

它们不是固定 URL 旁路，但仍应在 FE-002 收口到受控 Dynamic Action API Service。

公共 Runtime 能力中还有三处直接 Axios 依赖：`AdvancedQuery.vue`、`DynamicFormDialog.vue`、`column-format.ts`，合计 7 个 relation query 调用点。它们具备明确 Runtime 语义，不是页面随意 HTTP，但应合并到唯一 Relation Runtime API。

API 类型目前主要和 Service 同文件放置；Integration、Organization、Report 的 Service 已较大。页面内 311 个类型包含合理 View Model，也包含可能重复的 API shape，FE-002 只迁移确认重复的契约，不做机械搬运。

## 6. CSS、Theme 与 i18n 基线

| 指标 | 数量 |
| --- | ---: |
| 全前端 scoped style 文件 | 92 |
| 页面 scoped style 文件 | 58 |
| 全前端 Vue style 总行/非空行 | 9,283 / 7,805 |
| 页面 Vue style 总行/非空行 | 6,992 / 5,903 |
| 全局 SCSS | 2 文件，1,281 行 |
| 页面 style 中 hex token | 565，分布在 43 个页面文件 |
| 页面 style 中 rgb token | 58，分布在 18 个页面文件 |
| Dark/Theme 响应式引用 | 52 行，分布在 30 个页面文件 |
| 硬编码中文候选片段 | 3,346，77/77 页面文件均命中 |
| 使用 i18n 的页面 | 4 |
| zh-CN/en-US 资源 | 各 179 行 |

结论：CSS 不是越少越好，但页面级基础间距、flex、边框、颜色和深色补丁过多。FE-003 应优先使用 Quasar utility、palette 和平台 token，保留复杂 grid、稳定尺寸、树、设计器和领域可视化样式。

i18n 当前是半套状态。正式决策为中文平台：保留登录、布局、菜单和 Router 的现有 i18n，不在 FE-002/003 启动全量业务翻译，也不对外宣称完整双语。未来多语言应独立立项。

## 7. 公共能力评审

| 能力 | 结论 | 理由/后续动作 |
| --- | --- | --- |
| `BaseContent` | KEEP | 页面壳、滚动恢复和主题背景有稳定语义 |
| `TablePagination` | KEEP/EXTEND | 已有 26 个页面使用；补响应式、可访问性和统一事件约束 |
| `AdvancedQuery` | KEEP/EXTEND | 表达式能力稳定；收口 Relation API，不重写协议 |
| `AdvancedQueryRuleRow` | KEEP | `AdvancedQuery` 内部稳定子组件，不作为页面公共入口 |
| `DynamicFormDialog` | KEEP/EXTEND | Metadata 表单真值；逐步拆可测试逻辑，不另造表单系统 |
| `FormDialogShell` | KEEP | 14 个页面复用，承载显式 Dialog 的稳定布局 |
| `DetailFieldGrid` | KEEP/EXTEND | 详情字段白名单模式成立；补 Metadata adapter 和 token |
| `DetailSectionNavigation` | KEEP | 独立详情分区导航语义成立 |
| JsonEditor 组件族 | KEEP | JSON/数组/键值编辑是稳定特殊控件 |
| `OrganizationSelect` | KEEP | Organization selector 是明确平台语义 |
| `MasterDetailPage` | KEEP/EXTEND | Dictionary/Generalization 等主从页面需要；补小屏策略 |
| `TreeTable` | DELETE | FE-003 二次确认无生产消费者后已删除 |
| `usePageButtons` | KEEP/EXTEND | 权限按钮唯一入口；补 capability helper |
| `useConfirmDialog` | KEEP | 危险操作统一确认入口 |
| `field-metadata.ts` | KEEP/EXTEND | 类型、控件、选择器和联动解析真值 |
| `column-format.ts` | EXTEND/MERGE | 扩展为唯一 Runtime Column Resolver，合并旧 Builder |
| `query-state.ts` | EXTEND | 增加 `useTableQueryState`，不重写表达式协议 |
| `menu-button-display.ts` | KEEP | 动态按钮 icon/text/aria 默认真值 |
| `select-display.ts` | KEEP | 多选紧凑展示在 12 个消费者复用 |
| Layout Toolbar 组件 | KEEP | 服务应用 Header，不冒充 Table Toolbar |

没有公共组件因“Quasar 已经有相似控件”而机械删除。平台组件只有在承载平台语义时保留。

## 8. 页面 Pattern 样本

| Pattern | 样本 | 结论 |
| --- | --- | --- |
| 标准列表 | Integration External System、Retry Policy、System User | 标准参考，仍需收口重复 Query/Toolbar |
| 树 + 详情/列表 | Organization Structure、Legal Entity、Develop Dictionary | 保留领域/主从 Pattern，不转万能表格 |
| 配置管理 | Develop Database、Configure、Data Permission | 复杂配置分区，不能只套标准 Toolbar |
| 列表 + Dialog 详情 | Credential、External System、Retry Policy | 适合保留列表上下文 |
| 独立 Detail | Integration Execution Detail、Generic Record Detail、Organization Sync Batch Detail | 内容多或可链接，需隐藏路由权限保护 |
| 表单 | Generic Record Form、Sync Task Form、Interface Definition Form | 简单 Metadata 与复杂显式 Form 分流 |
| Dashboard | Dashboard | 不套 CRUD Pattern |
| Runtime/Execution 只读 | Execution、Integration Log、Sync Batch、Org Sync Error | 状态/诊断优先，不提供伪编辑 |
| Report 特殊 | Report Designer、Runtime、Workbench | `REPORT_DEFERRED` |

## 9. 重复模式 Top 10

词法命中用于识别重复热点，FE-002 实施前仍需逐页确认语义。

| 排名 | 模式 | 文件数 | 命中数 | 处理方向 |
| ---: | --- | ---: | ---: | --- |
| 1 | 分页状态和联动 | 39 | 241 | `useTableQueryState` + `TablePagination` |
| 2 | Quick Search 状态/处理 | 41 | 219 | 统一 Query State |
| 3 | 按钮 action 分发 | 27 | 150 | 通用动作 helper，领域动作留页面 |
| 4 | Detail 打开/加载流程 | 18 | 131 | 统一 Dialog/Route 选择和 capability guard |
| 5 | 状态映射 | 22 | 99 | domain map + shared StatusChip |
| 6 | visibleColumns | 13 | 96 | Runtime resolver 输出默认可见列 |
| 7 | 删除/危险确认 | 17 | 60 | `useConfirmDialog` |
| 8 | Advanced Query 接入 | 18 | 44 | `useTableQueryState` |
| 9 | 静态列声明 | 25（严格命名模式） | 40 | `resolveRuntimeColumns`；扩大模式为 38 文件/70 声明 |
| 10 | Metadata 加载/列构建 | 16 | 38 | `useRuntimeTableMetadata` |

另外有 21 个页面使用 QTable top slot，足以支持新增窄职责 Table Toolbar；表单/Dialog 提交流程在 25 个文件出现，但领域差异较大，不做万能 Form Composable。

## 10. 最大 Vue 文件

| 行数 | 文件 |
| ---: | --- |
| 3,174 | `pages/report-v2/designer/ReportDesignerPage.vue` |
| 2,968 | `pages/develop/database/Index.vue` |
| 2,267 | `components/FormDialog/DynamicFormDialog.vue` |
| 2,077 | `pages/develop/generalization/Index.vue` |
| 1,899 | `pages/system/menu/Index.vue` |
| 1,751 | `pages/report/design/Index.vue` |
| 1,302 | `pages/develop/configure/Index.vue` |
| 1,271 | `components/Query/AdvancedQuery.vue` |
| 1,170 | `pages/system/data-permission/Index.vue` |
| 1,123 | `pages/detail/RecordDetail.vue` |
| 1,040 | `pages/system/role/PermissionDialog.vue` |
| 1,027 | `pages/system/data-permission/components/DataPermissionConfigDialog.vue` |
| 985 | `pages/report-v2/workbench/ReportWorkbenchPage.vue` |
| 943 | `pages/develop/dictionary/Index.vue` |
| 913 | `pages/organization/structure/Index.vue` |
| 851 | `pages/report/manage/Index.vue` |
| 714 | `pages/report/design/components/ReportSheetCanvas.vue` |
| 709 | `pages/organization/employee/Index.vue` |
| 664 | `pages/system/user/Index.vue` |
| 581 | `pages/organization/legal-entity/Index.vue` |

大文件不是自动拆分理由。FE-002 只拆 Runtime Column、Query State、Relation API 等重复平台能力；Report 大文件保持保护，Database/Menu/Data Permission 在 FE-003 按真实职责小步迁移。

## 11. 最大 TypeScript 文件

| 行数 | 文件 |
| ---: | --- |
| 748 | `api/services/integration.ts` |
| 636 | `api/services/org.ts` |
| 572 | `api/services/report.ts` |
| 504 | `utils/params-schema.ts` |
| 477 | `router/routes.ts` |
| 442 | `utils/field-metadata.ts` |
| 393 | `utils/column-format.ts` |
| 387 | `api/services/sys-table.ts` |
| 370 | `api/services/data-permission-config.ts` |
| 346 | `types/enum.ts` |
| 321 | `modules/report/types.ts` |
| 316 | `utils/button-handlers.ts` |
| 295 | `pages/report-v2/mock/index.ts` |
| 273 | `modules/report/sheet.ts` |
| 266 | `pages/report-v2/composables/useReportParameterControls.ts` |
| 246 | `boot/table-overlay-scrollbar.ts` |
| 228 | `utils/query-state.ts` |
| 218 | `pages/report/composables/useReportRuntime.ts` |
| 216 | `api/services/sys-menu.ts` |
| 214 | `router/utils/index.ts` |

Service 文件较大但仍按稳定领域组织。只有在 API 子领域和调用方也稳定分离时才拆，不能按 HTTP method 或技术层机械拆文件。

## 12. 权限与路由

- 38 个路由 Page 由静态路由、后端菜单过滤和三个受控动态组件映射组成。
- 没有 `isAdmin`、角色名或 `admin` 判断。
- 12 个页面直接判断稳定 button code，主要用于跨资源查询和隐藏详情的 capability guard；这是安全的能力探测，不是角色硬编码，但调用形态应统一。
- `usePageButtons` 覆盖 22 个菜单页面。
- `defineOptions.name` 在 36 个 Page 文件出现；菜单页多数与 route/page code 一致，Generic Detail/Form、Dashboard 和部分 Report 是例外。
- 隐藏路由不会在菜单显示，但目前由页面分别阻止无权限数据请求；FE-002 应形成统一 capability guard，后端权限继续是最终安全边界。

前端按钮隐藏不是授权。所有直接路由和 API 均必须依赖后端 Casbin/Data Permission。

## 13. Loading、Empty、Error 与 Accessibility

### Loading

- 27 个页面引用全局 Loading，64 个页面有局部 loading 状态。
- 历史列表页经常同时使用全局和表格 Loading，可能导致全局遮罩闪烁。
- 标准冻结为：应用启动/会话切换用全局；QTable、Dialog、Detail 和按钮使用局部。

### Empty/Error

40 个页面包含 empty/no-data/error 状态相关实现，但表现不统一：部分使用 QTable no-data，部分 QBanner，部分请求失败后只保留空表。FE-003 必须区分首次无数据、查询无结果、无权限和失败。

### Accessibility

静态扫描识别 57 个 icon-only 按钮，其中 29 个在本地模板块内未发现 tooltip、`aria-label` 或动态 display props，分布在 13 个文件；多数集中于 Integration Execution 和 Report。扫描不能识别所有父级语义，因此这是 P2 核对清单，不直接判定缺陷。动态菜单按钮已有 `menuButtonDisplayProps` 提供 `aria-label`。

## 14. Integration 页面族

### 已统一

- External System、Interface Definition、Credential、Retry Policy、Sync Task 均使用 QTable、Runtime Metadata、Advanced Query、动态页面按钮和 Dialog；
- 多数使用 `visibleColumns`、`compactSelectionDisplay` 和 `useConfirmDialog`；
- 新页不按角色名显示按钮；独立查询 capability 已有测试。

### 仍分散

- 五个配置页各自维护 Metadata 请求、columns、visible columns、Advanced draft/applied、分页和 Toolbar；
- Sync Task 为压缩实现存在大量单行编排，可读性低；
- Execution、Log、Sync Batch 采用另一套静态只读模式，合理但缺共享 Runtime 状态/详情组件；
- 状态映射在列表和多个 Detail Dialog 重复。

结论：Integration 配置页是 FE-002 参考实现和 FE-003 第一迁移批次；Runtime 页面形成独立只读 Pattern，不强迫与配置页完全相同。

## 15. System/Develop 历史页面族

- Application、Role、SMS、User 已使用 Metadata Columns、Advanced Query、TablePagination、Page Buttons 和 Confirm Dialog，但查询样板高度重复。
- Audit 仍以静态列为主；Menu 是大体量树/按钮配置页；Data Permission 是多子领域配置页，均不应机械套普通 CRUD。
- Database、Generalization 和 Dictionary 已有 Metadata/主从能力，但文件体量大、API 与页面动作混杂。
- Configure 是独立配置 Form，不属于标准列表。

结论：FE-003 以 Application/Role/SMS/User 为标准列表迁移批次，以 Audit 为只读批次；Menu/Database/Data Permission 只抽公共能力，不做产品重写。

## 16. Organization 特殊页面

- Legal Entity 与 Structure 使用 `OrganizationReadOnlyTree` + `OrganizationReadOnlyDetail`，领域语义清晰；
- Employee 是列表 + Assignment scope/detail，不能降为简单 CRUD；
- Position 是领域列表；
- Sync Batch/Error 是业务同步诊断页面，状态、Reason 和安全摘要优先。

Organization 不使用通用 Metadata Dynamic Form 是合理选择。FE-003 只统一 Toolbar、Query State、Pagination、Status、Loading/Error 和按钮显示，不替换领域树或 Assignment UI。

## 17. Metadata 驱动边界与后端缺口

当前 Runtime Metadata 已提供：字段 code/name/type/input、dict、list/query/sort、sequence、form/detail span、binding/category/expression/tag/linkage。它足以驱动基础列、查询字段、简单 Form 和 Detail。

当前未提供：列表默认宽度、最小/最大宽度、alignment、format hint。

最小后端缺口只建议：

| 字段 | 语义 | 约束 |
| --- | --- | --- |
| `list_width` | 列默认像素宽度 | 可空正整数；`NULL/0` 为 auto |

不建议在 V1 增加 `min_width`、`max_width`、renderer、slot、CSS class 或组件名。min/max/alignment 可由前端字段类型默认和页面 Override 表达；格式继续由 logical/input type、dict 和页面领域 renderer 共同决定。

## 18. 风险分级

### P0：0

未发现原生 Axios 旁路、硬编码角色授权、未受控固定 API URL 或需要在 FE-001 立即修改业务代码的安全问题。

### P1：10

1. Runtime Columns 有多套静态/动态组装方式。
2. 16 个页面自行加载 Runtime Metadata。
3. Quick/Advanced/Applied/Page/Order 查询状态重复。
4. 21 个 QTable top slot 重复 Toolbar 布局。
5. 27 个页面重复 action 分发。
6. Dynamic Action 与 Relation Runtime 仍直接依赖 `boot/axios`。
7. 隐藏 Detail 的 capability guard 分散在页面。
8. 22 个页面重复状态映射。
9. API 契约与页面类型存在重复风险。
10. 同类 Detail/Form 打开模式缺少统一选择规则。

### P2：8

1. 页面 scoped CSS 5,903 非空行。
2. 硬编码颜色和页面 dark-mode patch 较多。
3. i18n 半套状态。
4. Empty/No Result/Forbidden/Error 表达不统一。
5. 29 个 icon-only 按钮需要人工补查可访问名称。
6. 全局和局部 Loading 混用。
7. Toolbar/Dialog/主从页面响应式策略不一致。
8. visibleColumns 缺少统一默认与偏好规则。

### P3：4

1. 多个 Vue 文件超过 1,000 行。
2. API Service、Runtime utility 和生产 chunk 体积持续增长；当前构建存在压缩后超过 900 kB 的 chunk warning。
3. `app.scss` 达 1,256 行，包含多个领域页面规则。
4. Component/Route/Page Name 映射仍有历史例外。

## 19. FE-002 建议范围

FE-002 先实现公共能力，不逐页大搬家：

1. 扩展 `column-format.ts`，实现唯一 `resolveRuntimeColumns`，让 `buildTableColumns` 委托它；
2. 新增 `useRuntimeTableMetadata`，统一加载、权限、局部 loading 和安全 fallback；
3. 基于 `query-state.ts` 新增 `useTableQueryState`；
4. 新增窄职责 `StandardTableToolbar`，只提供布局与 slot；
5. 新增共享 `StatusChip` renderer，领域 mapping 由调用方提供；
6. 新增 Relation Runtime API 与 Dynamic Action API Service，移除组件/页面直接 `boot/axios`；
7. 扩展 Page Button capability helper 和通用 action matching；
8. 选 External System 或 Retry Policy 作为参考页面，证明列 Override、Virtual Column、Query 和权限组合；
9. 为 resolver、query composable、toolbar、status 和 API boundary 补测试。

FE-002 不修改后端 Metadata，不迁移全部页面，不触碰 Report 产品模式。

## 20. FE-003 建议批次

| 批次 | 页面族 | 目标 |
| --- | --- | --- |
| 1 | Integration 配置页 | 建立标准列表参考族 |
| 2 | Integration Runtime + Organization Sync | 建立只读执行/诊断 Pattern |
| 3 | Organization | 统一表格外壳，保留树与 Assignment 领域 UI |
| 4 | System Application/Role/SMS/User/Audit | 迁移历史标准列表 |
| 5 | Develop Database/Dictionary/Generalization、System Menu/Data Permission | 只抽共享能力，小步处理大文件 |
| 6 | Login/Dashboard | Theme、API、i18n、Accessibility 核对 |
| Deferred | Report V1/V2 | `REPORT_DEFERRED`，等待 Report 专项 |

每个批次完成后执行前端 test、lint、typecheck、build，并进行桌面、小屏和深色模式浏览器验收。

## 21. FE-002 实施结果

FE-002 已建立公共页面基线，但没有宣称全部页面完成迁移：

- Platform View Action、Metadata Capability、Business Capability 已在长期指南中分开；
- 当前列表/详情刷新不再依赖纯页面 reload 型 MenuButton；业务刷新、同步、重试、缓存重建等能力保持不变；
- `resolveRuntimeColumns`、`useRuntimeTableMetadata`、`useTableQueryState`、`StandardTableToolbar`、`StatusChip` 和 capability helper 已落地并有测试；
- RecordDetail 保留固定 Back/Refresh，业务详情按钮来自父 Menu 的 `DETAIL_TOP`/`DETAIL_BOTTOM`；
- Generalization 页面、RecordDetail、AdvancedQuery、DynamicFormDialog 和 relation formatter 不再直接依赖 `boot/axios`；
- External System、Retry Policy、Application、Position 是 FE-003 的参考实现；
- 参考页的列表、Runtime Metadata 和字典运行时读取使用局部 loading，不再触发全局页面遮罩；
- Dashboard 低代码入口通过真实菜单树解析完整路由，不再假定固定 `/admin/develop` 父路径；
- `list_width` 暂不增加，列宽继续由字段类型默认与页面 Override 管理。

浏览器验收发现一个 FE-003 权限依赖项：只有列表查询和 Metadata 权限、没有业务按钮的账号可以搜索、分页、列选择和固定刷新，且不能加载 Detail；但带字典字段的参考页还会请求字典 Runtime API。角色若未同时获得对应字典读取权限，后端会稳定返回 403。FE-003 需要把只读页面的字典依赖纳入权限 Seed/配置检查，不能靠前端吞掉权限问题。

Report V2 Workbench 仍保留现有 `refresh` MenuButton，归入 `REPORT_DEFERRED`，本轮未借公共刷新治理改动 Report 产品页面。

### 21.1 FE-003 正式迁移清单

| 批次 | 页面 | 结论 |
| --- | --- | --- |
| A Integration | External System | PARTIAL（FE-002 reference，FE-003 只做族内一致性复核） |
| A Integration | Retry Policy | PARTIAL（FE-002 reference） |
| A Integration | Interface Definition | MIGRATE |
| A Integration | Credential | MIGRATE |
| A Integration | Sync Task | MIGRATE |
| B Runtime / Sync | Integration Sync Batch | MIGRATE |
| B Runtime / Sync | Integration Execution List/Detail | MIGRATE |
| B Runtime / Sync | Integration Log | MIGRATE |
| B Runtime / Sync | Organization Sync Batch Detail/List | MIGRATE |
| B Runtime / Sync | Organization Sync Error | MIGRATE |
| C Organization | Position | PARTIAL（FE-002 reference） |
| C Organization | Structure / Legal Entity | PARTIAL（保留树 + 详情 Pattern） |
| C Organization | Employee / Assignment | PARTIAL（保留领域详情） |
| D System | Application | PARTIAL（FE-002 reference） |
| D System | User / Role / SMS | MIGRATE |
| D System | Audit | MIGRATE（只读 Pattern） |
| D System | Menu / Data Permission | PARTIAL（复杂配置页） |
| E Develop | Dictionary | MIGRATE |
| E Develop | Database / Generalization | PARTIAL（大页面，仅抽公共能力） |
| E Develop | Configure | EXEMPT（配置 Form） |
| F Special | RecordDetail / RecordForm | PARTIAL（保留泛化语义） |
| F Special | Login / Dashboard / ChangePassword / 404 | EXEMPT（Theme、API、可访问性复核） |
| G Report | Report V1/V2 全页面 | REPORT_DEFERRED |

`MIGRATE` 表示按标准 Pattern 迁移；`PARTIAL` 表示只接入适用的公共能力并保留领域布局；`EXEMPT` 表示不套列表 Pattern；`REPORT_DEFERRED` 表示等待 Report 专项。

## 22. Freeze 结论

Frontend Architecture Standard：**通过冻结**。

Frontend Consistency Implementation：**已完成并冻结**。最终页面矩阵、指标和双账号浏览器验收见 [FrontendConsistencyFreezeReview](FrontendConsistencyFreezeReview.md)。

冻结内容包括：

- 页面 Pattern 分类；
- Metadata Base + Page Override + Virtual Column；
- 单一 Runtime Column Resolver；
- Standard Toolbar 和 Query State 语义；
- Button、Status、Detail、Form、CSS/Theme、API/Type、Composable 规则；
- Integration/System/Organization 迁移策略；
- Report `REPORT_DEFERRED`；
- Backend Metadata 仅有 `list_width` 最小候选缺口。

## 23. FE-001 验证结果

本 Task 未修改前端业务代码。验证使用 Node `v24.19.0` 和 Yarn `1.22.18`。

| 命令 | 结果 |
| --- | --- |
| `yarn test` | 36 个 Test Files、136 个 Tests 全部通过 |
| `yarn lint` | 通过 |
| `yarn typecheck` | 通过 |
| `yarn build` | SPA 构建成功；存在压缩后 chunk 大于 900 kB warning |
| `make docs-check` | 56 个 Markdown 文件，0 断链 |

首次运行测试时本地 `.quasar/tsconfig.json` 不存在，导致 Vitest/TypeScript 无法解析 `src/` 别名；执行项目现有 `yarn postinstall`（`quasar prepare`）重建 ignored 生成配置后，原命令全部通过。生成目录未进入 Git。
