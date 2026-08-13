# Report V2 领域模型

## 命名原则

数据库表不使用 `_v2` 后缀。

继续使用：

1. `report_definition`
2. `report_definition_version`
3. `report_execution_log`

后续正式新增：

1. `report_dataset`
2. `report_datasource`
3. `report_permission`

不要使用：

1. `report_definition_v2`
2. `report_dataset_v2`
3. `report_datasource_v2`

原因：

1. 数据库表表达领域概念，不表达开发迭代版本。
2. V1 已形成的 `report_definition`、`report_definition_version`、`report_execution_log` 是正式报表领域模型的一部分。
3. V2 不应因为前端体验重做而制造一套重复表。
4. 后续新增数据集、数据源、权限模型时，应进入正式 report 领域命名。

## 核心概念模型

### ReportDefinition

报表定义主表，承载草稿态配置和报表基础信息。

第一阶段继续使用现有 `report_definition`。

核心职责：

1. 报表编码、名称、分类、描述。
2. 当前草稿的 `query_config`。
3. 当前草稿的 `layout_config`。
4. 当前状态：draft / published / disabled。
5. 当前发布版本指针：`published_version_id`。
6. 权限相关字段：`permission_menu_id`、`permission_table_code`。

### ReportVersion

发布版本快照。

第一阶段继续使用现有 `report_definition_version`。

核心职责：

1. 保存发布时的 `query_config` 快照。
2. 保存发布时的 `layout_config` 快照。
3. 保存版本号。
4. 保存发布人、发布时间、发布说明。
5. 支撑运行态只读发布版本。

### ReportExecutionLog

报表执行日志。

第一阶段继续使用现有 `report_execution_log`。

核心职责：

1. 记录设计预览。
2. 记录运行。
3. 记录导出。
4. 记录成功/失败、耗时、行数、错误。
5. action 区分 `design_preview` / `runtime_run` / `runtime_export`。

### ReportDataset

报表数据集。

V2 文档必须把 Dataset 作为一等概念预留，不能继续只当成临时 JSON 小片段来思考。

第一阶段可以继续内嵌在 `query_config.datasets` 中，不落表。

第二阶段评估落地 `report_dataset`。

核心职责：

1. 数据集名称。
2. 数据集类型：table / sql / future external。
3. 来源表或 SQL。
4. 字段 schema。
5. 参数 schema。
6. 是否可复用。
7. 权限和数据权限策略。

### ReportDatasource

报表数据源。

第一阶段不落表，继续复用 `sys_table / sys_table_field` 数据源发现。

第二阶段评估落地 `report_datasource`。

核心职责：

1. 系统主库。
2. 外部 MySQL/PostgreSQL 等数据源。
3. 连接配置。
4. 凭据加密。
5. 可用性测试。
6. 数据源级权限。

### ReportPermission

报表级权限。

第一阶段不落表，继续复用菜单、按钮、Casbin、数据权限体系。

第二阶段评估落地 `report_permission`。

核心职责：

1. 报表可见范围。
2. 报表运行权限。
3. 报表导出权限。
4. 报表设计权限。
5. 与组织、角色、用户的绑定。

## 第一阶段必须落地 / 继续使用

第一阶段继续使用：

1. `report_definition`
2. `report_definition_version`
3. `report_execution_log`
4. 内嵌 dataset 配置。
5. `sys_table / sys_table_field` 数据源发现。
6. 菜单、按钮、Casbin、数据权限体系。

第一阶段不新增：

1. `report_datasource`
2. `report_dataset`
3. `report_permission`

## 第二阶段评估落地

第二阶段评估：

1. `report_dataset`
2. `report_datasource`
3. `report_permission`

判断条件：

1. 是否需要跨报表复用数据集。
2. 是否需要外部数据库。
3. 是否需要报表级授权，不再完全依赖菜单权限。
4. 是否需要数据集字段解析、血缘、缓存、可见性管理。

## V2 配置模型

V2 第一阶段仍然生成现有：

1. `query_config`
2. `layout_config`
3. `sheet/cell/binding`

快速表格模式可以在 `layout_config` 中加入可选 UI 元数据，例如：

```json
{
  "designer_mode": "quick_table",
  "quick_table_config": {
    "dataset_id": "main",
    "columns": [],
    "parameters": [],
    "generated_by": "quick_table",
    "generated_at": "2026-07-04T00:00:00+08:00"
  }
}
```

这些字段只服务前端设计体验，不改变后端运行态规则。
