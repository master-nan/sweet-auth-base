# 报表模块 V1-E 前端 UI 重设计方案

## 一、V1-E 目标

V1-E 不是重写报表系统，而是在 V1-A / V1-B / V1-D 已完成发布版本、运行态隔离、后端导出和前端接口接入的基础上，优化前端信息架构和使用体验。

目标：

1. 减少报表中心和报表管理重复。
2. 统一运行弹窗和导出逻辑。
3. 让报表设计器更容易上手。
4. 增加快速表格报表模式。
5. 保留高级 Sheet 模式。
6. 不破坏现有 `query_config` / `layout_config` / `sheet/cell/binding`。

## 二、V1-E 不做内容

V1-E 明确不做：

1. 不改后端发布版本模型。
2. 不改运行态读取版本快照规则。
3. 不新增 `report_datasource`。
4. 不新增 `report_dataset`。
5. 不合并菜单和权限。
6. 不做图表大屏。
7. 不做打印分页。
8. 不做填报。
9. 不做版本回滚。
10. 不重写 `ReportSheetPreview`。
11. 不重写 `ReportSheetCanvas`。
12. 不改 `schema.ts` / `sheet.ts` / `options.ts` 核心结构。

## 三、报表中心和报表管理重构方向

当前 `report_center` 和 `report_manage` 已经都接入 `/run` 和 `/export`，但两者存在大量重复：

- 运行弹窗结构重复。
- 参数表单逻辑重复。
- 运行分页逻辑重复。
- 导出请求和下载逻辑重复。
- `ReportSheetPreview` 使用方式重复。

V1-E 第一阶段先抽取：

1. `ReportRuntimeDialog.vue`
2. `useReportRuntime.ts`
3. `useReportExport.ts`

暂时不抽 `ReportWorkspace`。

暂时不合并 `report_center` / `report_manage` 菜单。

理由：

- 运行弹窗和导出逻辑边界清晰，抽取风险最低。
- 不触碰菜单、路由、权限，避免引入权限回归。
- 后续再抽 `ReportWorkspace` 时，运行弹窗已经稳定可复用。

## 四、ReportRuntimeDialog 设计

`ReportRuntimeDialog.vue` 用于统一报表中心和报表管理中的运行弹窗。

### 顶部

顶部展示：

- 报表名称
- 当前版本
- 报表说明
- 导出按钮

建议结构：

```text
报表名称                当前版本：V{version_no}        导出 CSV | 关闭
报表说明 / 数据权限说明
```

### 参数区

参数区根据报表参数生成表单：

- 文本参数
- 数字参数
- 日期参数
- 日期范围参数
- 查询按钮
- 重置按钮

要求：

- 参数来源沿用现有 `layout_config.parameters` / `query_config.parameters`。
- 参数值结构沿用现有 `parameters`。
- 不新增新的参数 schema。

### 结果区

结果区继续使用：

- `ReportSheetPreview`

要求：

- 不重写 `ReportSheetPreview`。
- 保持现有 `sheet`、`datasets`、`previewData`、`reportKind` 入参。
- 运行结果继续显示后端 `/run` 返回数据。

### 分页区

分页区展示：

- 当前页
- 每页条数
- 总数

分页行为：

- 查询时调用 `/run`。
- 页码变化时重新调用 `/run`。
- 每页条数沿用当前运行配置。

### 导出

导出行为：

- 调用 `/export`。
- 使用 V1-D 已有 `exportReport + downloadBlob`。
- 不前端拼 CSV。
- 错误信息沿用 blob JSON 解析结果。

### 兼容要求

`ReportRuntimeDialog.vue` 必须兼容：

- 报表中心运行 published 报表。
- 报表管理运行 published 报表。
- 管理页行级导出。
- 后续 `ReportWorkspace` 复用。

## 五、报表设计器新 UI 方向

设计器采用两种模式：

1. 快速表格模式。
2. 高级 Sheet 模式。

### 快速表格模式

快速表格模式面向普通后台管理项目中的常见报表。

能力：

- 选择数据来源。
- 勾选字段。
- 配置列名、宽度、格式、排序。
- 配置查询参数。
- 一键生成现有 sheet 配置。
- 保存并预览。
- 发布。

设计原则：

- 快速表格模式不新增底层报表模型。
- 快速表格模式生成现有 `layout_config.sheet`。
- 字段绑定继续使用现有 `sheet/cell/binding`。
- 预览继续使用 `ReportSheetPreview`。
- 发布继续调用 `/publish`。

### 高级 Sheet 模式

高级 Sheet 模式保留当前设计器能力。

必须保留：

- 当前 `ReportSheetCanvas`
- 当前 `ReportInspectorPanel`
- 当前 `ReportResourcePanel`

适用场景：

- 复杂布局。
- 合并单元格。
- 分组。
- 汇总。
- 公式。
- 类 Excel 报表。

设计原则：

- 高级 Sheet 模式不重写现有画布。
- 快速表格模式可以生成初始 sheet，用户再切换到高级模式细调。
- 切换到高级模式后，后续布局由用户手动维护。

## 六、设计器新布局

### 顶部

顶部结构：

```text
返回 | 报表名称 | 状态 | 线上版本 | 保存草稿 | 保存并预览 | 发布 | 版本
```

顶部只承载高频全局动作：

- 返回
- 报表名称
- 当前状态
- 当前线上版本
- 保存草稿
- 保存并预览
- 发布
- 版本

### 左侧

左侧改为步骤导航：

1. 基本信息
2. 数据来源
3. 字段配置
4. 查询参数
5. 布局设计
6. 预览发布

每个步骤可以显示完成状态：

- 未开始
- 配置中
- 已完成
- 有错误

### 中间

中间展示当前步骤内容或 Sheet 画布。

不同步骤对应内容：

- 基本信息：报表名称、编码、分类、说明。
- 数据来源：表数据集、SQL 数据集入口、字段解析。
- 字段配置：字段选择、列名、宽度、格式、排序。
- 查询参数：参数字段、操作符、默认值。
- 布局设计：快速表格模式 / 高级 Sheet 模式。
- 预览发布：预览结果、校验结果、发布摘要。

### 右侧

右侧定位为当前对象属性。

右侧不再承载过多全局配置。

适合放在右侧的内容：

- 当前单元格属性。
- 当前字段绑定属性。
- 当前样式属性。
- 当前选中对象的上下文配置。

不建议继续主要放在右侧的内容：

- 数据来源主流程。
- 参数主流程。
- Join 主流程。
- 发布主流程。

这些应进入左侧步骤导航对应的中间内容区。

## 七、V1-E 分阶段计划

### V1-E-1

抽取：

- `ReportRuntimeDialog.vue`
- `useReportRuntime.ts`
- `useReportExport.ts`

目标：

- 统一报表中心和报表管理中的运行弹窗。
- 统一 `/run` 调用。
- 统一 `/export` 调用。
- 统一导出 loading、错误提示和下载逻辑。

### V1-E-2

抽取：

- `ReportWorkspace`

目标：

- 复用报表中心和报表管理页列表能力。
- 保留 `report_center` 和 `report_manage` 两个路由。
- 暂不合并菜单。

建议：

- `mode = runtime` 时只展示 published 报表和运行入口。
- `mode = manage` 时展示全部状态和管理操作。

### V1-E-3

新增快速表格模式 UI。

目标：

- 选择数据来源。
- 勾选字段。
- 配置列。
- 配置参数。
- 一键生成现有 sheet 配置。

要求：

- 不改 `schema.ts` / `sheet.ts` / `types.ts` 核心结构。
- 不新增后端模型。
- 不破坏高级 Sheet 模式。

### V1-E-4

设计器增加步骤导航。

目标：

- 让用户知道创建报表的下一步。
- 把数据来源、字段、参数、布局、预览发布组织成清晰流程。

要求：

- 保留当前设计器组件。
- 只调整信息架构和入口。

### V1-E-5

优化右侧属性面板。

目标：

- 右侧只作为当前对象属性面板。
- 减少全局配置堆叠。
- 单元格、绑定、样式配置更清晰。

要求：

- 不重写 `ReportInspectorPanel`。
- 可先小步拆分 tab 或调整展示顺序。

### V1-E-6

评估是否合并菜单为“报表工作台”。

前置条件：

- `ReportRuntimeDialog` 稳定。
- `ReportWorkspace` 稳定。
- 权限可见性策略明确。
- 用户角色路径明确。

注意：

- 本阶段只是评估。
- 不在 V1-E 前半段合并菜单和权限。

## 八、风险控制

1. 先抽公共运行弹窗，风险最低。
2. 后抽列表工作台。
3. 再改设计器。
4. 不一次性重写所有页面。
5. 每个阶段都要执行：
   - `yarn lint`
   - `yarn typecheck`
   - `yarn build`

补充约束：

- 不重写 `ReportSheetPreview`。
- 不重写 `ReportSheetCanvas`。
- 不改 `schema.ts` / `sheet.ts` / `options.ts` 核心结构。
- 不改发布版本模型。
- 不改运行态读取版本快照规则。
- 不合并菜单和权限。
- 每个阶段只做一个明确目标，完成后再进入下一阶段。
