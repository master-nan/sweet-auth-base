# 报表设计器第一阶段架构决策

## 决策摘要

| 编号 | 决策 | 结论 |
|---|---|---|
| ADR-001 | 是否重写设计器 | 不重写，保留当前 sheet/cell/binding 设计器 |
| ADR-002 | 报表主表 | 保留 `ReportDefinition` |
| ADR-003 | 配置结构 | 保留 `query_config` 和 `layout_config` |
| ADR-004 | 数据源 | 第一阶段继续复用 `sys_table` / `sys_table_field` |
| ADR-005 | 数据集 | 第一阶段不新增 `report_dataset` |
| ADR-006 | 发布模型 | 新增 `report_definition_version` |
| ADR-007 | 运行态读取 | 运行态必须读取发布版本快照 |
| ADR-008 | 预览和运行 | 设计预览和运行接口分离 |
| ADR-009 | 导出 | 新增后端受控导出接口 |
| ADR-010 | SQL 安全 | 采用工程可落地的 SQL 安全守卫 |
| ADR-011 | 权限 | 复用现有菜单、按钮、Casbin、数据权限体系 |
| ADR-012 | 执行日志 | 继续使用 `ReportExecutionLog`，增加 action 区分 |

## ADR-001 不重写设计器

### 决策

第一阶段不重写前端设计器，不做自由画布增强，不做复杂 UI。

### 原因

当前设计器已经具备轻量 sheet 报表能力，包括：

- 数据源选择
- table/sql dataset 配置
- 参数配置
- 字段绑定
- 单元格绑定
- sheet 预览

第一阶段的主要风险不在 UI，而在发布、运行、安全、导出和权限边界。

## ADR-002 保留 ReportDefinition

### 决策

`ReportDefinition` 继续作为报表主定义表，承载草稿态和设计态数据。

### 规则

- 草稿编辑继续写入 `report_definition.query_config`。
- 草稿编辑继续写入 `report_definition.layout_config`。
- `report_definition.status` 继续表达 `draft`、`published`、`disabled`。
- 新增发布版本指针，指向当前已发布快照。

## ADR-003 保留 query_config 和 layout_config

### 决策

第一阶段继续保留 `query_config` 和 `layout_config`，不引入新的 template schema 或 dataset schema 表。

### 规则

- `query_config` 继续保存数据集、字段、参数等查询配置。
- `layout_config` 继续保存 sheet 布局、单元格、绑定和运行展示配置。
- 发布时将两份 JSON 固化到版本快照。
- 运行态不能直接读取草稿 JSON。

## ADR-004 不新增 report_datasource

### 决策

第一阶段不新增 `report_datasource` 表。

### 原因

当前阶段聚焦系统主库和低代码元数据复用，避免引入外部连接、密码存储、连接池、安全边界等额外复杂度。

### 规则

- 数据源发现继续来自 `sys_table` / `sys_table_field`。
- 不支持外部 MySQL / PostgreSQL。
- 不保存外部数据源密码。

## ADR-005 不新增 report_dataset

### 决策

第一阶段不新增 `report_dataset` 表。

### 原因

现有 table/sql dataset 配置已经保存在 `query_config` / `layout_config` 中，第一阶段优先解决发布和运行隔离。

### 规则

- 数据集继续作为 JSON 配置存在。
- 不做跨报表数据集复用。
- 不做数据集版本。
- 不增强多数据集复杂联动。

## ADR-006 新增 report_definition_version

### 决策

新增 `report_definition_version` 保存发布快照。

### 作用

- 固化发布时的 `query_config`。
- 固化发布时的 `layout_config`。
- 固化发布时的报表基础信息。
- 确保草稿修改不影响已发布运行结果。

## ADR-007 运行态只读发布版本

### 决策

报表中心运行和导出必须读取 `report_definition_version`。

### 规则

- `runtime_run` 不允许读取草稿 `query_config` / `layout_config`。
- `runtime_export` 不允许读取草稿 `query_config` / `layout_config`。
- `design_preview` 可以读取草稿。

## ADR-008 分离设计预览和运行

### 决策

第一阶段新增独立接口：

- 设计时预览：`design_preview`
- 发布后运行：`runtime_run`
- 发布后导出：`runtime_export`

### 原因

当前 `/preview` 同时承担设计预览和运行，无法表达发布态、安全策略和日志 action。

## ADR-009 新增后端受控导出

### 决策

第一阶段新增后端导出接口。

### 规则

- 只允许 published 报表导出。
- 导出必须读取发布版本。
- 导出必须校验权限和数据权限。
- 导出必须限制最大导出行数。
- 导出必须写执行日志。

## ADR-010 SQL 安全守卫

### 决策

第一阶段采用工程可落地的 SQL 安全守卫，不实现完整 AST SQL 白名单。

### 最低规则

1. 只允许 SELECT / WITH。
2. 禁止多语句。
3. 禁止分号。
4. 禁止 DDL / DML。
5. 禁止 `copy`、`exec`、`pg_sleep`、`explain analyze` 等危险语句或函数。
6. 参数必须绑定。
7. 必须设置查询超时。
8. 必须限制 pageSize。
9. 必须限制导出行数。
10. SQL 报表默认仅管理员可运行。
11. SQL 数据集暂不承诺自动数据权限注入。

## ADR-011 复用现有权限体系

### 决策

第一阶段继续复用现有菜单、按钮、Casbin 和数据权限体系。

### 规则

- 接口继续走认证和 Casbin。
- 报表中心只展示可访问的 published 报表。
- 普通表数据集继续复用现有数据权限。
- SQL 报表默认不开放给普通用户。

## ADR-012 执行日志 action 区分

### 决策

继续使用 `ReportExecutionLog`，但 action 必须区分不同场景。

### action

- `design_preview`
- `runtime_run`
- `runtime_export`

### 记录要求

- 报表 ID
- 发布版本 ID
- 用户 ID
- 用户名称
- action
- 请求参数
- 成功状态
- 耗时
- 返回行数或导出行数
- 错误信息
