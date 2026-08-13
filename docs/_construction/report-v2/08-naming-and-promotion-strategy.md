# Report V2 命名与转正策略

## 命名总原则

开发期可以临时使用 `report-v2` 目录，避免和旧 report 前端代码混淆。

最终正式模块必须命名为 `report`，不长期保留 `v2` 命名。

旧 V1 前端最终改名为 `report-legacy` 或归档。

后端 API 不使用 `/report-v2`，继续使用 `/admin/report`。

数据库表不使用 `_v2` 后缀。

## 开发期命名策略

推荐开发期采用：

```text
frontend/src/pages/report-v2/
  workbench/
  designer/
  runtime/
  components/
  composables/
```

原因：

1. 避免和旧 `report` 页面混在一起。
2. 可以并行开发 V2。
3. 不破坏 V1 已能运行的功能。
4. 方便 V2 验收后整体转正。

备选方案：

```text
frontend/src/pages/report/
  workbench-v2/
  designer-v2/
```

不推荐备选方案作为首选，因为容易让 V1 和 V2 代码边界混乱。

## 转正期命名策略

转正期目录策略：

```text
frontend/src/pages/report/          新版正式报表模块
frontend/src/pages/report-legacy/   旧版 V1 归档
```

也就是说：

开发期：

- 新模块临时叫 `report-v2`。
- 旧模块继续叫 `report`。

转正期：

- 旧 `report` 改名为 `report-legacy`。
- 新 `report-v2` 改名为 `report`。

最终正式状态：

- 正式产品路径中不要长期出现 `report-v2`。
- 用户菜单中不要出现“报表 V2”。
- 正式菜单叫“报表工作台”或“报表”。

## 前端目录迁移策略

开发期：

```text
frontend/src/pages/report-v2/
```

转正期：

```text
frontend/src/pages/report-v2/      -> frontend/src/pages/report/
frontend/src/pages/report/         -> frontend/src/pages/report-legacy/
```

最终：

```text
frontend/src/pages/report/
frontend/src/pages/report-legacy/
```

## 路由迁移策略

开发期临时路由可以是：

```text
/report-v2/workbench
/report-v2/designer
/report-v2/designer/:id
/report-v2/runtime/:id
```

正式转正后路由应该是：

```text
/report/workbench
/report/designer
/report/designer/:id
/report/runtime/:id
```

旧 V1 路由最终归档为：

```text
/report-legacy/center
/report-legacy/manage
/report-legacy/design
```

或者隐藏旧菜单，仅保留代码和路由一段时间用于回退。

## 菜单迁移策略

开发期：

1. 可以新增“报表工作台 Beta”菜单。
2. 仅给管理员或开发人员使用。
3. 不替换现有 `report_center / report_manage / report_design`。

转正期：

1. 正式菜单改为“报表工作台”。
2. 隐藏旧“报表中心”“报表管理”菜单。
3. 旧菜单可保留一段时间作为 legacy 入口，但不推荐给普通用户使用。

最终正式菜单建议：

```text
报表工作台
报表设计器（是否单独显示由权限和产品体验决定）
```

不要最终保留：

1. 报表 V2。
2. 新版报表。
3. 报表中心 + 报表管理两个重复入口。

## 后端 API 命名策略

后端 API 不使用 `/report-v2`。

正式 API 继续使用：

```http
POST /admin/report/query
GET  /admin/report/data-sources
GET  /admin/report/:id
POST /admin/report
PUT  /admin/report/:id
DELETE /admin/report/:id
POST /admin/report/:id/design-preview
POST /admin/report/:id/publish
POST /admin/report/:id/run
POST /admin/report/:id/export
GET  /admin/report/:id/versions
```

原因：

1. 后端 API 表达的是领域能力，不是前端版本。
2. V1-A / V1-B 已完成的发布、运行、导出、版本能力本身是正式能力。
3. V2 主要重做前端产品体验，不需要为了前端重构复制一套 `/report-v2` 后端 API。
4. 避免长期维护两套后端 API。

如果未来出现完全不兼容的新后端 API，再考虑版本化；当前 Report V2 不需要。

## 数据库表命名策略

数据库表不使用 `_v2` 后缀。

继续使用：

```text
report_definition
report_definition_version
report_execution_log
```

后续正式新增：

```text
report_dataset
report_datasource
report_permission
```

不要使用：

```text
report_definition_v2
report_dataset_v2
report_datasource_v2
```

原因：

1. 数据库表表达领域概念，不表达开发迭代版本。
2. V1 已形成的 `report_definition`、`report_definition_version`、`report_execution_log` 是正式报表领域模型的一部分。
3. V2 不应因为前端体验重做而制造一套重复表。
4. 后续新增数据集、数据源、权限模型时，应进入正式 report 领域命名。

## 为什么最终不用 v2 命名

`v2` 是开发期识别符，不是产品领域概念。

长期保留 `v2` 会带来问题：

1. 用户会感知到新旧版本割裂。
2. 后续 V3、V4 命名会继续膨胀。
3. 菜单、路由、目录、权限难以收敛。
4. 正式模块语义不清。

因此最终正式模块应回归 `report`。

## 为什么旧版叫 report-legacy

`report-legacy` 表达“旧版兼容实现”，便于：

1. 保留回退能力。
2. 保留历史报表编辑入口。
3. 避免和正式 `report` 混淆。
4. 逐步下线旧页面。

`legacy` 是归档状态，不是面向普通用户的长期产品入口。
