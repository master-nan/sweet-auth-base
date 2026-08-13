# 报表设计器第一阶段验收清单

## 范围验收

- [ ] 未重写现有设计器。
- [ ] 未引入 `report_datasource` 表。
- [ ] 未引入 `report_dataset` 表。
- [ ] 未引入外部数据库数据源。
- [ ] 未做图表大屏。
- [ ] 未做打印分页。
- [ ] 未做填报。
- [ ] 未做定时调度。
- [ ] 未做邮件订阅。
- [ ] 未增强多数据集复杂联动。

## 保留能力验收

- [ ] `ReportDefinition` 保留并继续作为报表主定义。
- [ ] `ReportExecutionLog` 保留并继续记录执行日志。
- [ ] `query_config` 保留。
- [ ] `layout_config` 保留。
- [ ] `backend/internal/reportconfig` 保留。
- [ ] 当前 sheet/cell/binding 设计器保留。
- [ ] 当前 table/sql dataset 配置能力保留。
- [ ] 当前 `sys_table` / `sys_table_field` 数据源发现能力保留。
- [ ] 当前 `ReportSheetPreview` 保留。
- [ ] 当前菜单、按钮、Casbin、数据权限体系保留。

## 数据模型验收

- [ ] 新增 `report_definition_version`。
- [ ] `report_definition` 有 `published_version_id` 或等价发布版本指针。
- [ ] 发布版本保存 `query_config` 快照。
- [ ] 发布版本保存 `layout_config` 快照。
- [ ] 发布版本保存必要报表基础信息。
- [ ] 修改草稿不会修改已发布版本快照。

## 设计时预览验收

- [ ] 设计器能保存草稿。
- [ ] 设计器能调用设计时预览接口。
- [ ] 设计时预览读取 `report_definition` 草稿。
- [ ] 设计时预览允许 `draft`。
- [ ] 设计时预览允许 `published`。
- [ ] 设计时预览不允许 `disabled`。
- [ ] 设计时预览必须分页。
- [ ] 设计时预览写执行日志。
- [ ] 设计时预览日志 action 为 `design_preview`。

## 发布验收

- [ ] 发布接口校验 `query_config`。
- [ ] 发布接口校验 `layout_config`。
- [ ] 发布接口校验 SQL 安全。
- [ ] 发布成功后创建 `report_definition_version`。
- [ ] 发布成功后更新 `report_definition.published_version_id`。
- [ ] 发布成功后更新 `report_definition.status = published`。
- [ ] 发布失败时不创建版本。
- [ ] 发布失败时不更新 `published_version_id`。

## 运行验收

- [ ] 报表中心只展示可运行的 published 报表。
- [ ] 运行接口只允许 `published`。
- [ ] 运行接口必须读取 `report_definition_version`。
- [ ] 运行接口不允许直接读取草稿 `query_config`。
- [ ] 运行接口不允许直接读取草稿 `layout_config`。
- [ ] 运行接口必须校验权限。
- [ ] 运行接口必须应用数据权限。
- [ ] 运行接口必须分页。
- [ ] 运行接口必须写执行日志。
- [ ] 运行日志 action 为 `runtime_run`。
- [ ] disabled 报表不能运行。
- [ ] 修改草稿不会影响已发布运行结果。

## 导出验收

- [ ] 后端提供受控导出接口。
- [ ] 导出接口只允许 `published`。
- [ ] 导出接口必须读取发布版本。
- [ ] 导出接口不能读取草稿配置。
- [ ] 导出接口必须校验权限。
- [ ] 导出接口必须应用数据权限。
- [ ] 导出接口必须限制最大导出行数。
- [ ] 导出接口必须写执行日志。
- [ ] 导出日志 action 为 `runtime_export`。
- [ ] disabled 报表不能导出。

## SQL 安全验收

- [ ] SQL 只允许 `SELECT` / `WITH`。
- [ ] SQL 禁止多语句。
- [ ] SQL 禁止分号。
- [ ] SQL 禁止 DDL。
- [ ] SQL 禁止 DML。
- [ ] SQL 禁止 `copy`。
- [ ] SQL 禁止 `exec`。
- [ ] SQL 禁止 `pg_sleep`。
- [ ] SQL 禁止 `explain analyze`。
- [ ] SQL 参数必须绑定。
- [ ] SQL 查询必须设置超时。
- [ ] 运行接口必须限制 pageSize。
- [ ] 导出接口必须限制导出行数。
- [ ] SQL 报表默认仅管理员可运行。
- [ ] SQL 数据集第一阶段不承诺自动数据权限注入。
- [ ] SQL 报表不会默认绕过权限开放给普通用户。

## 权限和数据权限验收

- [ ] 所有新增接口位于后台认证组下。
- [ ] 所有新增接口受 Casbin 控制。
- [ ] 发布需要对应按钮或接口权限。
- [ ] 运行需要对应按钮或接口权限。
- [ ] 导出需要对应按钮或接口权限。
- [ ] 普通 table 数据集继续复用现有数据权限。
- [ ] join 报表至少对主表应用数据权限。
- [ ] SQL 报表普通用户默认无运行权限。

## 前端验收

- [ ] 设计器保存仍可用。
- [ ] 设计器预览改为设计时预览接口。
- [ ] 设计器发布按钮调用发布接口。
- [ ] 报表管理页能展示发布状态。
- [ ] 报表管理页能查看版本列表。
- [ ] 报表中心运行调用运行接口。
- [ ] 报表中心导出调用后端导出接口。
- [ ] 报表中心不运行草稿。
- [ ] 报表中心不运行 disabled 报表。

## 日志验收

- [ ] `design_preview` 写入执行日志。
- [ ] `runtime_run` 写入执行日志。
- [ ] `runtime_export` 写入执行日志。
- [ ] 日志记录用户。
- [ ] 日志记录报表。
- [ ] 日志记录版本。
- [ ] 日志记录参数。
- [ ] 日志记录成功状态。
- [ ] 日志记录耗时。
- [ ] 日志记录行数或导出行数。
- [ ] 日志记录错误信息。

## 最小闭环验收

- [ ] 创建或编辑报表草稿。
- [ ] 设计时预览草稿。
- [ ] 发布生成版本快照。
- [ ] 报表中心运行 published version。
- [ ] 修改草稿后，报表中心结果不变。
- [ ] 再次发布后，报表中心运行新版本。
- [ ] 后端导出 published version。
- [ ] 运行和导出都记录执行日志。
- [ ] disabled 后不能运行和导出。
