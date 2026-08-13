# 报表设计器第一阶段范围说明

## 背景

当前报表模块已经具备轻量 sheet 报表设计器雏形，包括报表定义、查询配置、布局配置、基础预览、执行日志、`sys_table` / `sys_table_field` 数据源发现、菜单按钮权限和 Casbin 接入。

第一阶段不是重做设计器，也不是复刻 FineReport。目标是把现有能力产品化，补齐运行态和发布态必须具备的工程闭环。

## 第一阶段目标

第一阶段聚焦以下能力：

1. 发布版本
2. 运行态隔离
3. 后端导出
4. 执行日志
5. SQL 安全
6. 权限和数据权限
7. 报表中心运行 published version

完成后，报表模块应能支持普通后台管理项目中的轻量级报表设计、发布、运行和受控导出。

## 第一阶段必须保留

以下现有能力必须保留，不在第一阶段重写：

1. `ReportDefinition`
2. `ReportExecutionLog`
3. `query_config`
4. `layout_config`
5. `backend/internal/reportconfig`
6. 当前 sheet/cell/binding 设计器
7. 当前 table/sql dataset 配置能力
8. 当前 `sys_table` / `sys_table_field` 数据源发现能力
9. 当前 `ReportSheetPreview`
10. 当前菜单、按钮、Casbin、数据权限体系

## 第一阶段只允许新增

第一阶段只围绕以下内容做增量建设：

1. `report_definition_version`
2. `published_version_id` 或等价发布版本指针
3. 设计时预览接口
4. 发布接口
5. 运行接口
6. 后端导出接口
7. 版本列表接口
8. SQL 安全守卫
9. 执行日志 action 区分

## 第一阶段明确不做

第一阶段不做以下内容：

1. 不新增 `report_datasource` 表
2. 不新增 `report_dataset` 表
3. 不支持外部数据库数据源
4. 不实现完整 AST SQL 白名单
5. 不做完整服务拆分
6. 不增强自由画布
7. 不做图表大屏
8. 不做打印分页
9. 不做填报
10. 不做定时调度
11. 不做邮件订阅
12. 不增强多数据集复杂联动

## 范围边界

### 后端边界

- 保持 `ReportDefinition` 作为草稿和编辑态主表。
- 新增发布版本快照，运行态只读发布版本。
- 保持现有 `query_config` / `layout_config` JSON 结构，第一阶段只做校验和兼容。
- 保持现有数据源发现逻辑，不新增外部数据源管理。
- 增加设计预览、发布、运行、导出、版本列表等接口。
- 增强 SQL 安全守卫和执行日志 action。

### 前端边界

- 保留当前报表设计器整体形态。
- 保留当前 sheet/cell/binding 交互。
- 保留当前 `ReportSheetPreview`。
- 只做必要入口调整：设计预览、发布、运行、导出、版本查看。
- 不扩展复杂 UI，不重写设计器。

### 数据库边界

- 保留 `report_definition`。
- 保留 `report_execution_log`。
- 新增 `report_definition_version`。
- 在 `report_definition` 增加 `published_version_id` 或等价指针。
- 不新增 `report_datasource`。
- 不新增 `report_dataset`。

## 第一阶段完成标准

第一阶段完成后应满足：

1. 设计器能保存草稿。
2. 设计器能设计时预览。
3. 发布时生成版本快照。
4. 报表中心只运行 published version。
5. 修改草稿不会影响已发布运行结果。
6. 后端支持受控导出。
7. 运行和导出都写执行日志。
8. disabled 报表不能运行。
9. 普通表数据集能继续复用现有数据权限。
10. SQL 报表不会默认绕过权限开放给普通用户。
