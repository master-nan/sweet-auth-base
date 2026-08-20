# PFCR-S2 Frontend & Test Structural Simplification

## 审计基线

- 仓库：`sweet-auth-base`
- 实施基线：`d48f3c6afc1360fd4099cbaaec3410506b627f1b`
- 分支：`main`
- 实施前工作区：干净，`main` 相对 `origin/main` 为 ahead 1 / behind 0
- 审计方法：重新读取当前 HEAD 的页面、组件、composable、API、utils、CSS、Router、动态菜单、Query Scope、MenuButton 和测试；历史 Review 只作为检查线索

实施前已执行 `git status`、`git diff`、`git log -n 10 --oneline`，未发现用户未提交修改。

## 实施前结构

统计口径：`frontend/src`；production 排除 `*.spec.ts`、`*.test.ts`、`*.test.mjs`。

| 指标 | Before |
| --- | ---: |
| Frontend `src` 文件 | 304 |
| Vue 文件 | 126 |
| TS 文件 | 175 |
| Page Vue | 80 |
| Component Vue | 44 |
| Production composable | 8 |
| Production utils | 23 |
| Test 文件 | 72（71 个 Vitest 文件 + 1 个未被 Vitest 收集的 `node:test` 文件） |
| Vitest case | 278 |
| Frontend LOC | 77610 |
| Production LOC | 69701 |
| Test LOC | 7909 |
| 页面 scoped style 文件 | 59 |
| 页面 scoped style 行 | 6906 |
| Query Center 接入页面 | 18 |
| 18 页面 LOC | 9868 |
| `DynamicFormDialog.vue` | 2244 |

18 个页面均使用 `useQuerySchemePage`，且均重复装配 Selector、QuickPresets、AdvancedQuery 和 SaveDialog。`useQuerySchemePage` 已统一 scope、initialization、default、dirty、apply、save 和 reset，是稳定状态边界；剩余问题主要在 Template 和 Advanced/Save UI 胶水。

`DynamicFormDialog` 当前同时承担 Dialog/form 生命周期、字段分组布局、整表验证与提交、联动、预览、按钮编排，以及单字段 Input Type 到控件的 17 类渲染。后半部分的单字段渲染树具有清晰可拆职责，但 form-level orchestration 必须继续留在 Dialog。

## 候选裁决

### KEEP

| 对象 | 当前职责与调用方 | 结论与理由 | 风险/验证 |
| --- | --- | --- | --- |
| `useQuerySchemePage` | 18 页统一 Scheme scope、default、dirty、save、apply | KEEP；这是已有窄业务边界，不创建第二套状态 | 保留 composable 行为测试 |
| Query Scheme Selector/SaveDialog/Preview/QuickPresets/Manager/Drawer/Edit/RoleSelect | 独立产品语义 | KEEP；不合并成 QueryScheme God Component | 组件行为测试、Manager 浏览器回归 |
| `AdvancedQuery` | Simple/Advanced、Binding、Nested、Preview | KEEP；只调整被组合组件调用的边界，不拆协议/引擎 | Advanced/Simple 全量测试 |
| `StandardTableToolbar` | 平台 Toolbar 布局与 View Refresh | KEEP；不重做 Toolbar，只承载组合组件 | Toolbar 与 1366 宽度验收 |
| `FormDialogShell` | Dialog shell、preview rail、响应式布局 | KEEP；与字段控件职责不同 | 现有 shell 测试 |
| `field-metadata.ts`、query/column/menu/decimal utils | Metadata/查询/显示的单一规则 | KEEP；具有跨页规则和测试价值 | 单元与 DynamicForm 回归 |
| `auth-session-storage.ts` | 集中清理认证 Session keys | KEEP；虽小但属于认证安全边界 | user store + 现有单测 |
| `button-actions.ts` | 10 个生产页面的标准 action dispatch | KEEP；不是单调用薄 wrapper | 页面按钮与单测 |
| Integration/Organization/System API Service | 按领域集中 API 契约 | KEEP；不按实体拆文件 | typecheck/API tests |
| MasterDetail、Report | 稳定 shell / `REPORT_DEFERRED` | KEEP；不扩产品能力 | 编译兼容回归 |
| file preview disabled shim | Quasar alias 显式使用 | KEEP；不是 dead util | build |

### MERGE

| 对象 | 当前问题 | 计划结构 | 验证 |
| --- | --- | --- | --- |
| 18 页 Query Scheme Template 组合 | 每页重复 Selector、Preset、Advanced trigger、Save trigger、两个 Dialog 和事件连线 | 新增最多一个 `QuerySchemeControls.vue`，只组合 UI；页面继续持有 query state、业务加载、columns、权限和领域行为 | 先迁移 ExternalSystem、RetryPolicy、Application、Position，再扩 14 页 |
| `typeof.ts` | 仅 Dashboard CountTo 使用 `isNumber`，其余 10 个导出无调用 | 使用语言原生判断并删除整文件 | Dashboard/build |
| `utils/index.ts` | `deepClone` 仅 permission boot 使用，`getFirst` 无调用 | 改用已有 `lodash/cloneDeep`，删除 barrel/薄实现 | Router/permission tests |

### SIMPLIFY

| 对象 | 当前问题 | 计划结构 | 验证 |
| --- | --- | --- | --- |
| 18 页 Advanced/Save 临时状态 | 同一 begin/apply/close/save wiring 重复 | Controls 管理展示态；调用既有 page/query-state 动作，不加载 API、不初始化 default | Query Scheme 行为与页面专项 |
| `EligiblePageMatrix.spec.ts` | `it.each` 对 18 个 `.vue` 重复断言多个标签字符串 | 保留一个 Architecture Guard：冻结 scope/route 唯一性并验证统一 Controls + composable；行为由组件/composable 测试负责 | Vitest |
| 页面 Dark Patch | 多页重复 light/dark surface、border、text literal | 仅把重复颜色语义收敛到现有 `app.scss` semantic tokens；页面专属布局保留 | 亮/暗浏览器验收 |
| 测试 mount/fixture | 审计三处以上完全相同 setup 后才抽取 | 只抽真实重复的局部 helper，不创建通用测试框架 | Test LOC 与可读性 |

### SPLIT

| 对象 | 当前职责 | 计划结构 | 风险/验证 |
| --- | --- | --- | --- |
| `DynamicFormDialog.vue` | 2244 行；form lifecycle 与 17 类字段控件渲染同文件 | 新增一个 `DynamicFieldControl.vue`，承接单字段 Input Type→Component、字段级交互；Dialog 保留 formData、分组、联动、整表验证、submit、preview、按钮 | DynamicForm、Relation、Decimal、File、Dict、Organization 浏览器与单测 |

`DynamicFieldControl` 不按 input type 再拆文件，不拥有 form submit、Scheme/API、页面权限或 form lifecycle。Input Type、decimal、dictionary、relation、logical type、hint/default/boolean 的规则继续复用现有 metadata/runtime 边界，不复制第二份映射。

### DELETE

| 对象 | 调用证据 | 结论 | 验证 |
| --- | --- | --- | --- |
| `pages/organization/legal-entity/Index.vue` | 当前静态 Router 无入口；Seed 将旧独立菜单迁移并删除，正式入口已并入 Organization Structure；动态 importer 无该路径 | DELETE；删除历史 dead page，不删除 Legal Entity API/selector/Structure 能力 | Router、Org seed、Organization 浏览器验收 |
| `utils/typeof.ts` | 仅一个 `isNumber` 调用，其余导出为零 | DELETE after inline | Dashboard/build |
| `utils/index.ts` | 一个可替代调用 + 一个 dead export | DELETE after merge | permission boot tests |

### DEFER

- `ReportFilterBar.vue` 等 Report 内部疑似无引用文件：`REPORT_DEFERRED`，不以 grep 单点删除。
- Data Permission、Designer、Organization Tree 的大块页面专属 CSS：真实产品布局，不做视觉重构。
- DynamicForm formula/section/editable grid、Query Center 新能力、i18n、移动专项：超出 PFCR-S2。
- Chunk 依赖、CI/Release、Migration Ledger、TLS/Operations：PFCR-S3。

## 禁止处理清单

不修改后端业务、DB Schema、Migration、权限/Data Permission、Query Scheme 协议、AdvancedQuery 语义、Report 产品、HR Production、Metadata 能力、Editable Grid、Aggregate、国际化或移动端设计。不创建 BaseCrudPage、QueryCenterPage、QueryEnabledTable、PageMixin、`useEverything`、UniversalMount 或 Theme V2。

## 实施顺序

1. 建立本报告并冻结真实调用图。
2. 新增 Query Scheme UI 组合组件，迁移四个 Reference 页面并运行专项测试/浏览器检查。
3. 迁移其余 14 页，精炼 Architecture Guard。
4. 拆分单一 `DynamicFieldControl`，运行 DynamicForm/metadata/relation 专项。
5. 删除确认 dead page/utils，最小收敛重复 semantic color。
6. 运行浏览器验收、前后端完整门禁、PostgreSQL release-check，回填报告。

## 回归测试计划

- Frontend：Query Scheme component/composable/page tests、AdvancedQuery、DynamicForm、metadata/relation、Router、Organization、全量 `yarn test`。
- 静态门禁：`yarn lint`、`yarn typecheck`、`yarn build`。
- Backend：`go test ./... -count=1`；API 契约不变。
- 完整门禁：使用当前 Makefile 的 PostgreSQL 16 `make release-check`。
- 文档：`make docs-check`、`git diff --check`。
- 浏览器：Application、User、Employee、Position、ExternalSystem、Credential、SyncTask、Query Scheme Manager、Dictionary、Generalization Dynamic Form，亮色/深色和 Console。

## 预计文件变化

- 新增生产 Vue 最多 2 个：Query Scheme UI 组合、Dynamic Field Control。
- 删除确认 dead 的 Legal Entity 页面和两个无独立价值 utils。
- 不拆 API Service，不合并 Query Scheme 领域组件，不新增页面/路由/后端接口。

## 实际修改结果

### KEEP

- `useQuerySchemePage` 继续是 scope、初始化、default、dirty、apply、save 和 reset 的唯一页面状态边界。本轮只导出其稳定返回类型，未在组合组件中再次请求 Scope/Scheme API。
- Selector、SaveDialog、Preview、QuickPresets、Manager、DetailDrawer、EditDialog、RoleSelect 继续保持独立产品组件，没有合并成巨型 `QueryScheme.vue`。
- `AdvancedQuery` 的 Simple/Advanced、Binding 和 Query 协议未改；`StandardTableToolbar` 仍只提供布局和 View Refresh。
- Integration、Organization、System API Service 未拆分；MasterDetail 和 Report 未改产品结构。
- `auth-session-storage.ts`、`button-actions.ts`、field/query/column/menu/decimal 规则工具经调用图复核后保留。

### MERGE

- 新增 `QuerySchemeControls.vue`，以一个 154 行的 UI-only 组件组合 Selector、QuickPresets、AdvancedQuery Dialog 和 SaveDialog。它接收 `useQuerySchemePage` controller 与页面 query state，不拥有列表请求、分页、数据权限、列、业务状态或默认初始化。
- 18 个 Eligible 页面均改用该组件；页面内 `<query-scheme-selector>`、`<query-quick-presets>`、`<query-scheme-save-dialog>` 直接装配归零。
- `utils/index.ts` 的唯一有效能力改为直接使用已有 `lodash/cloneDeep`；`typeof.ts` 的唯一调用使用现有类型约束表达，不保留薄工具文件。

### SIMPLIFY

- `StandardTableToolbar` 将四个 Query Scheme 细粒度 slot 收敛为一个 `query-controls` slot；没有生产调用方依赖旧 slot。
- Controls 保留原有“方案｜查询组”分隔；没有高级查询字段的运行页不再挂载隐藏的 `AdvancedQuery`，条件数量继续进入 Tooltip 与 aria-label。
- 18 页面 Query Scheme 总 LOC 从 9868 降至 8484，减少 1384 行（14.0%）；领域筛选、业务按钮、列 override、领域 Dialog 和诊断信息仍显式保留。
- `EligiblePageMatrix.spec.ts` 从逐页重复源码断言收敛为一个 18 页 Architecture Guard；它冻结页面 Route Identity、共享 composable + Controls 的接入边界，不维护 scope code 第二真值。
- Dictionary 页面删除失效的局部 `dictCache` 包装，改为清理真正被运行时消费的 Pinia dict store。
- `buildAllColumnFormats`、`resolveRelationMenuId` 等零调用导出删除；单调用且无公共语义的 button handler 降为私有函数。

### SPLIT

- 新增唯一的 `DynamicFieldControl.vue`（492 行），负责单字段 Input Type 到控件的渲染、字段值更新及字段级事件。
- `DynamicFormDialog.vue` 从 2244 行降至 1786 行；继续负责 formData、Dialog 生命周期、字段分组和布局、relation/dict 数据、联动、整表验证、submit、preview 与 form-level button。
- 表单初始化通过窄字段级 reset token 恢复敏感输入的隐藏状态，避免切换记录后沿用上一条记录的密码可见状态；Decimal-safe 字符串提交新增行为回归。
- 未出现一个输入类型一个文件；FileUpload、RichText、Organization、DateTime、Json、Cascader 等既有复杂控件继续复用。字段类型解析、Decimal、Relation、Dictionary、Logical Type 和 Boolean 规则仍只有现有 metadata/runtime 真值。

### DELETE

- 删除 `pages/organization/legal-entity/Index.vue`：静态 Router 无入口，动态 Seed 已退休旧独立菜单，正式法人能力位于 Organization Structure；后端 API、selector 与组织结构能力未删。
- 删除 `composables/dictCache.ts`、`utils/index.ts`、`utils/typeof.ts`。
- 删除重复的 `system/application/QuerySchemeIntegration.spec.ts`；其架构约束由统一 Matrix Guard 覆盖，行为由组件/composable 测试覆盖。
- 将未被 Vitest 收集的 `menu-button.test.mjs` 删除并迁移为 `menu-button.spec.ts`，3 个 capability 行为测试现在进入正式测试门禁。

### DEFER

- Report 继续 `REPORT_DEFERRED`；Designer、Data Permission 和 Organization Tree 的领域 CSS 不做视觉重构。
- Quasar dev checker 当前报告 14 个存量 type-aware ESLint 问题，而正式 `yarn lint`/CI 配置通过；规则入口漂移留 PFCR-S3 统一，生产构建与控制台不受影响。
- TMS 演示 `tms_company` 页面初始化会提交包含空 `field` 的表达式并收到 400“参数错误”；DynamicForm 与 Relation 选择功能可用，但该 Generalization 初始 Query State 问题需后续单独修复。
- JSON/Array/KeyValue/RichText 的统一 required 校验及前三者 readonly 支持是拆分前已存在的控件契约缺口；本轮没有扩大为 Dynamic Form 产品整改，留后续 Freeze Review 专项处理。
- Chunk warning、CI/Release 架构、Migration Ledger、TLS/Operations 留 PFCR-S3。

## Query Center 页面变化

Reference 页 External System、Retry Policy、Application、Position 验证组件边界后，再迁移其余 14 页。18 页仍各自持有 `queryState`、业务 list load、columns、capabilities、domain filters 和业务 Dialog；组合组件不直接调用业务 API，也没有新增页面专属 Store、Dirty 或 Default 实现。

页面理解路径由“Toolbar 内 4 个 slot + 2 个 Dialog + 多组事件”缩短为“一个 Query Controls + 页面领域内容”。`useQuerySchemePage` 调用页仍为 18，Controls 页面为 18，旧三组件页面直连为 0。

## DynamicForm 变化

字段控件树搬入 `DynamicFieldControl` 后，Dialog 主文件可直接阅读表单级生命周期。拆分后两文件合计 2278 行，较原文件多 34 行类型契约、敏感字段 reset 契约和组件边界成本，但核心 Dialog 减少 458 行；没有复制 Input Type resolver、默认值、Relation/Dictionary 或 Decimal 规则。

浏览器在 Generalization Dynamic Form 中验证了新增 Dialog、普通输入和 Relation Cascader。验收发现 Cascader 右侧面板的硬编码白底后，将该公共控件改用 `--app-surface` / `--app-border`，未增加页面 dark patch。

## Utils 变化

- MERGE：`utils/index.ts` → 既有 `lodash/cloneDeep`；`typeof.ts` → 调用点已有类型语义；`dictCache.ts` → Pinia `dictStore.clearDict`。
- DELETE：上述 3 文件，以及 2 个 dead export；没有创建 `utils.ts` 或第二套 auth/button 边界。
- KEEP：auth session、button action、field metadata、column format、menu display/query/decimal 等稳定规则。

## CSS / Theme 变化

`app.scss` 补充已有视觉所需的 `surface`、`surface-muted`、`border`、`text-strong`、`text-muted` 语义别名，并在暗色模式映射到既有 dark token。External System、Credential、Interface Definition 的 Detail Dialog 删除重复 dark patch；Retry Policy、Sync Task 修正不存在的 border token；Cascader 使用公共语义 token。

页面 scoped style 文件由 59 降至 58，样式块内容行数由 6906 降至 6718。下降主要来自 dead Legal Entity 页面和重复 dark patch；没有把页面 CSS 搬到全局伪造下降。

## Tests 变化

- 最终 Vitest 为 72 文件、228 cases，全部通过。Before 的 278 cases 中包含大量 `it.each` 展开的页面源码字符串断言；本轮净减少 50 个生成式/重复架构检查，并补入 Controls 挂载边界、敏感字段 reset 与 Decimal-safe 提交，不是删除权限、Metadata、Query Scheme 或业务行为场景。
- 保留 Default、Dirty、Apply、Save、DEGRADED、INVALID、Revision Conflict、Visibility、Role、PERSONAL isolation、Advanced/Simple 及各领域行为测试。
- 新增 `QuerySchemeControls.spec.ts` 3 个交互测试；原来门禁外的 MenuButton 3 个行为测试迁入 Vitest。五个页面中已失效的 `AdvancedQuery` mock/stub 一并删除。
- 未发现 3 处以上完全相同且适合共享的 mount/router/pinia fixture，因此没有创建 UniversalMount 或新的测试框架。

## 文件与 LOC 变化

同一统计口径：`frontend/src`，production 排除 `*.spec.ts`、`*.test.ts`、`*.test.mjs`。

| 指标 | Before | After | 变化 |
| --- | ---: | ---: | ---: |
| Frontend `src` 文件 | 304 | 302 | -2 |
| Vue 文件 | 126 | 127 | +1 |
| TS 文件 | 175 | 173 | -2 |
| Page Vue | 80 | 79 | -1 |
| Component Vue | 44 | 46 | +2 |
| Production composable | 8 | 7 | -1 |
| Production utils | 23 | 21 | -2 |
| Test 文件 | 72 | 72 | 0 |
| Vitest 文件 | 71 | 72 | +1 |
| Vitest cases | 278 | 228 | -50 |
| Frontend LOC | 77610 | 75625 | -1985 |
| Production LOC | 69701 | 67699 | -2002 |
| Test LOC | 7909 | 7926 | +17 |
| 页面 scoped style 文件 | 59 | 58 | -1 |
| 页面 scoped style 行 | 6906 | 6718 | -188 |
| 18 Query Center 页面 LOC | 9868 | 8484 | -1384 |
| `DynamicFormDialog.vue` | 2244 | 1786 | -458 |

本轮新增 2 个生产 Vue 文件，没有新增 composable、API Service 或后端文件；删除 6 个文件，修改/新增/删除共 63 个提交文件。

## Browser 结果

使用当前源码重新构建 Docker 生产前端并真实登录 Admin，覆盖 Application、User、Employee、Position、External System、Credential、Sync Task、Query Scheme Manager、Dictionary、TMS Generalization Dynamic Form。

- Query Selector、AdvancedQuery、SaveDialog、列设置、刷新、业务新增、Manager、DynamicForm、Relation Cascader 均完成交互检查；未提交或删除业务数据。
- 亮色与深色均检查；1357 CSS px（1366 级桌面）为单行 Toolbar、无页面横向溢出；同时在 962 CSS px 窄桌面检查了受控换行。
- 最终生产标签页 Console：0 Error、0 Warning、0 Vue Warning、0 Unhandled Promise；验收请求未出现 404/403。
- TMS 演示页存在上述 400 初始查询问题，已作为未处理问题记录，未掩盖为通过。

## 测试结果

| 门禁 | 结果 |
| --- | --- |
| `yarn test` | PASS，72 files / 228 tests |
| `yarn lint` | PASS |
| `yarn typecheck` | PASS |
| `yarn build` | PASS；仅保留既有 >900 KiB chunk warning |
| `go test ./... -count=1` | PASS |
| PostgreSQL 16 强制门控 | PASS |
| `go test -race ./... -count=1` | PASS |
| `make release-check` | PASS |
| `make docs-check` | PASS，65 Markdown files |
| `git diff --check` | PASS |

## 未处理项

- PFCR-S3：dev checker/lint 配置漂移、chunk 分析、CI/Release、Migration Ledger、TLS/Docker/Operations。
- 后续 Freeze Review：TMS Generalization 空字段初始表达式的 400；不在本轮改 Query 协议或后端。
- 后续 Freeze Review：JSON/Array/KeyValue/RichText 的 required/readonly 控件契约缺口；不在结构精炼中改变 Dynamic Form 产品行为。
- DOC-FINAL：吸收长期结构结论并删除 construction 阶段材料。

本轮未修改 API URL、Request/Response、Query/权限/Data Permission、数据库、Migration、Scheme Default/Dirty、分页排序或 Form 保存语义。
