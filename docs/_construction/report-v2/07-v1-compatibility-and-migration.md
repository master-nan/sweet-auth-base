# V1 兼容与迁移策略

## 基本原则

V1 不立即删除。V1 的后端闭环是 V2 的底座；V1 的前端体验需要被替换。

V2 初期并行开发，不替换旧菜单，不破坏旧路由。

## V1 必须保留

必须保留：

1. `report_definition`
2. `report_definition_version`
3. `report_execution_log`
4. `POST /admin/report/:id/design-preview`
5. `POST /admin/report/:id/publish`
6. `POST /admin/report/:id/run`
7. `POST /admin/report/:id/export`
8. `GET /admin/report/:id/versions`
9. 运行态只读发布版本。
10. 后端受控导出。
11. 执行日志 action 区分 `design_preview` / `runtime_run` / `runtime_export`。
12. 菜单、按钮、Casbin、数据权限体系。
13. SQL 数据集管理员限制和 SQL 安全守卫思路。

## V1 可复用能力

可以复用但不作为默认体验：

1. `ReportSheetCanvas`
2. `ReportSheetPreview`
3. `ReportInspectorPanel`
4. `ReportResourcePanel`
5. 当前 `sheet/cell/binding` 模型。
6. 当前 `query_config / layout_config`。
7. 当前 `sys_table / sys_table_field` 数据源发现能力。
8. 当前 table/sql dataset 配置能力。

## V1 建议归档或重写的体验

建议废弃或重写：

1. V1 报表中心页面体验。
2. V1 报表管理页面体验。
3. V1 设计器默认进入空白 Sheet 画布的流程。
4. V1 报表中心和报表管理重复的运行/导出体验。
5. 旧的“直接 Sheet 画布开始设计”的默认流程。
6. V1 中把报表中心和报表管理割裂成两个重复页面的交互方式。

## V1 路由保留策略

V1 路由不立即删除。

开发期继续保留：

```text
/report/center
/report/manage
/report/design
```

转正后旧 V1 路由建议归档为：

```text
/report-legacy/center
/report-legacy/manage
/report-legacy/design
```

或者隐藏旧菜单，仅保留代码和路由一段时间用于回退。

## V1 页面归档策略

阶段 1：并行开发

- 保留旧 `report`。
- 新增 `report-v2`。
- 不替换旧菜单。

阶段 2：V2 验收

- 新建快速表格报表。
- 保存草稿。
- 设计时预览。
- 发布版本。
- 报表工作台运行。
- 后端导出。
- 查看版本。
- 修改草稿不影响线上。

阶段 3：转正

- 旧 `report` -> `report-legacy`。
- 新 `report-v2` -> `report`。
- 菜单切换到新路由。
- 旧路由隐藏或只给管理员临时访问。

## 数据兼容策略

1. V1 数据表不迁移到 `_v2` 表。
2. 不新增 `report_definition_v2`。
3. 不新增 `report_definition_version_v2`。
4. 不新增 `report_execution_log_v2`。
5. 旧 V1 报表默认进入高级 Sheet 模式。
6. V2 快速表格模式不尝试自动解析旧复杂 Sheet。
7. 旧报表继续使用原 `query_config / layout_config`。

## 菜单迁移策略

开发期：

1. 可以新增“报表工作台 Beta”菜单。
2. 仅给管理员或开发人员使用。
3. 不替换现有 `report_center / report_manage / report_design`。

转正期：

1. 正式菜单改为“报表工作台”。
2. 隐藏旧“报表中心”“报表管理”菜单。
3. 旧菜单可保留一段时间作为 legacy 入口，但不推荐给普通用户使用。

## 回滚策略

如果 V2 转正后出现严重问题：

1. 重新启用旧 V1 菜单。
2. 路由切回 `report-legacy` 或旧 `report`。
3. 数据无需回滚到 `_v2` 表，因为没有创建 `_v2` 表。
4. 后端 API 没有复制为 `/report-v2`，因此运行态能力保持一致。

## 风险控制

1. V2 不直接删除 V1 页面。
2. V2 初期不改菜单和权限。
3. V2 初期不改后端运行态规则。
4. V2 初期不新增外部数据源。
5. V2 初期不做图表大屏。
6. V2 初期不做填报。
7. V2 初期不做打印分页。
8. V2 初期不做定时订阅。
9. V2 先做普通表格报表，再扩展高级能力。
10. 旧 V1 报表默认进入高级 Sheet 模式。
11. Dataset 是正式概念，但第一阶段可以先不落表。
12. 避免过度设计成 FineReport。
