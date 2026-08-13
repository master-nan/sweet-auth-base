# V1-D 前端实现计划

## 1. 阶段目标

V1-D 的目标是把当前前端报表模块接入 V1-A / V1-B 后端闭环，让现有报表设计器、报表管理页、报表中心页能够使用后端已经完成的发布版本、运行态隔离、后端受控导出和版本列表能力。

后端已具备以下接口：

- `POST /admin/report/:id/design-preview`
- `POST /admin/report/:id/publish`
- `POST /admin/report/:id/run`
- `POST /admin/report/:id/export`
- `GET /admin/report/:id/versions`

V1-D 只做前端接入，不重写设计器，不扩展复杂 UI，不修改后端，不引入新的数据源或数据集实体。

## 2. 范围边界

### 2.1 V1-D 只做

1. API service 增加 `publish` / `design-preview` / `run` / `export` / `versions`。
2. 类型定义补充 `ReportVersion` / `ReportPublishRes` / `ReportExportReq` / `PreviewMeta`。
3. 解决 export blob 下载和 Axios 拦截器兼容。
4. 新增轻量 `ReportVersionDialog`。
5. 报表中心运行改为 `/run`。
6. 报表中心导出改为 `/export`。
7. 报表管理页发布改为 `/publish`。
8. 报表管理页运行改为 `/run`。
9. 报表管理页导出改为 `/export`。
10. 报表管理页增加版本列表入口。
11. 报表设计器预览改为“保存并预览”：保存草稿后调用 `/design-preview`。
12. 报表设计器发布改为：轻量确认后保存草稿，再调用 `/publish`。
13. 旧 `previewReport` 方法保留兼容，但新逻辑不再用它作为报表中心运行接口。

### 2.2 V1-D 不做

1. 不做前端复杂 UI 改造。
2. 不重写报表设计器。
3. 不修改后端。
4. 不新增 `report_datasource`。
5. 不新增 `report_dataset`。
6. 不做外部数据库数据源。
7. 不做图表大屏。
8. 不做打印分页。
9. 不做填报。
10. 不做定时订阅。
11. 不做邮件发送。
12. 不做版本回滚。
13. 不做版本 diff。
14. 不做异步导出任务中心。
15. 不做 PDF / Excel 模板导出。

## 3. 预计修改文件清单

预计修改：

- `frontend/src/api/services/report.ts`
- `frontend/src/modules/report/types.ts`
- `frontend/src/pages/report/components/ReportVersionDialog.vue`
- `frontend/src/pages/report/center/Index.vue`
- `frontend/src/pages/report/manage/Index.vue`
- `frontend/src/pages/report/design/Index.vue`
- `frontend/src/utils/download.ts`

视实现情况可能修改：

- `frontend/src/pages/report/design/components/ReportDesignerTopbar.vue`
- `frontend/src/boot/axios.ts`

不建议修改：

- `frontend/src/modules/report/schema.ts`
- `frontend/src/modules/report/options.ts`
- `frontend/src/modules/report/sheet.ts`
- `frontend/src/pages/report/components/ReportSheetPreview.vue`
- `frontend/src/pages/report/design/components/ReportSheetCanvas.vue`
- `frontend/src/pages/report/design/components/ReportInspectorPanel.vue`
- `frontend/src/pages/report/design/components/ReportResourcePanel.vue`
- `frontend/src/pages/report/design/components/ReportDatasetDialog.vue`
- `frontend/src/pages/report/design/components/ReportParameterDialog.vue`
- `frontend/src/pages/report/design/components/ReportJoinDialog.vue`

## 4. API service 设计

文件：`frontend/src/api/services/report.ts`

### 4.1 方法签名

前端 API 方法必须显式传 `id`，不要从 `req.report_id` 中取。

```ts
publishReport(id: number, req?: ReportPublishReq): Promise<ReportPublishRes>

designPreviewReport(id: number, req: ReportPreviewReq): Promise<ReportPreviewRes>

runReport(id: number, req: ReportPreviewReq): Promise<ReportPreviewRes>

exportReport(id: number, req: ReportExportReq): Promise<ReportExportFile>

queryReportVersions(id: number): Promise<ReportVersion[]>
```

### 4.2 接口映射

```ts
publishReport        -> POST /admin/report/:id/publish
designPreviewReport  -> POST /admin/report/:id/design-preview
runReport            -> POST /admin/report/:id/run
exportReport         -> POST /admin/report/:id/export
queryReportVersions  -> GET  /admin/report/:id/versions
```

### 4.3 `previewReport` 兼容策略

保留现有：

```ts
previewReport(req: ReportPreviewReq): Promise<ReportPreviewRes>
```

继续调用：

```http
POST /admin/report/:id/preview
```

但 V1-D 新逻辑中不再用它作为报表中心或报表管理页的运行接口。

### 4.4 export blob 处理

`exportReport` 使用 `responseType: 'blob'`，并返回：

```ts
{
  blob: Blob
  filename: string
  contentType: string
}
```

文件名解析优先级：

1. `Content-Disposition` 中的 `filename*`
2. `Content-Disposition` 中的 `filename`
3. 前端兜底文件名，例如 `report.csv`

### 4.5 blob 错误 JSON 处理

导出失败时，后端可能返回 JSON 错误响应。由于请求使用 `responseType: 'blob'`，前端需要识别：

```ts
Content-Type: application/json
```

或 Blob 内容可解析为 JSON 时，读取错误信息并抛出业务错误，不应把错误 JSON 当作 CSV 下载。

### 4.6 是否复用 existing generalization 下载逻辑

可以参考：

- `frontend/src/pages/develop/generalization/Index.vue`

但不建议继续在页面内复制下载逻辑。V1-D 建议新增轻量下载工具：

- `frontend/src/utils/download.ts`

页面只负责调用 `downloadBlob`。

## 5. 类型设计

文件：`frontend/src/modules/report/types.ts`

### 5.1 `ReportVersion`

建议定义：

```ts
export interface ReportVersion {
  id: number
  report_id: number
  version_no: number
  status: 'published' | 'archived' | 'draft' | string
  state?: boolean
  published_at?: string
  published_by?: number | string
  published_name?: string
  published_by_name?: string
  change_log?: string
  created_at?: string
  updated_at?: string
  is_current?: boolean
}
```

展示发布人时优先使用：

```ts
published_name || published_by_name || published_by
```

### 5.2 `ReportPublishReq`

建议定义：

```ts
export interface ReportPublishReq {
  change_log?: string
}
```

`change_log` 可选，不强制填写。

### 5.3 `ReportPublishRes`

建议定义：

```ts
export interface ReportPublishRes {
  report_id: number
  version_id: number
  version_no: number
  status: ReportStatus
  published_at?: string
}
```

### 5.4 `ReportExportReq`

`ReportExportReq` 要贴近后端结构。第一版建议字段：

```ts
export interface ReportExportReq {
  format?: 'csv' | 'xlsx'
  menu_id?: number
  dataset_id?: string
  parameters?: Record<string, unknown>
  params?: Record<string, unknown>
  query?: ReportQuery
  max_rows?: number
}
```

规则：

1. 前端不要主动传 `maxRows`。
2. 前端不要把 `filters` / `keyword` / `expressions` 散在顶层。
3. 查询相关内容继续放在 `query` 中。
4. 第一版默认 `format = 'csv'`。
5. 如果用户选择 `xlsx`，后端不支持时展示明确错误。

### 5.5 `ReportExportFile`

建议定义：

```ts
export interface ReportExportFile {
  blob: Blob
  filename: string
  contentType: string
}
```

### 5.6 `ReportPreviewMeta`

扩展 `ReportPreviewRes.meta`：

```ts
export interface ReportPreviewMeta {
  report_id?: number
  report_code?: string
  source_code?: string
  dataset_id?: string
  dataset_type?: ReportDatasetType
  applied_menu_id?: number
  runtime_type?: 'design_preview' | 'runtime_run' | 'runtime_export'
  version_id?: number
  version_no?: number
}
```

### 5.7 `Report` 发布版本字段

建议扩展：

```ts
published_version_id?: number
published_version_no?: number
```

如果后端列表暂时不返回 `published_version_no`，前端可以只展示 `published_version_id` 或通过版本列表判断当前版本。

## 6. blob 下载工具设计

建议新增：

- `frontend/src/utils/download.ts`

包含：

```ts
downloadBlob(blob: Blob, filename: string): void

parseContentDispositionFilename(contentDisposition?: string): string

parseBlobJsonError(blob: Blob): Promise<string | undefined>
```

### 6.1 `downloadBlob`

职责：

1. 创建 object URL。
2. 创建临时 `<a>`。
3. 设置 `download` 文件名。
4. 触发点击。
5. 回收 object URL。

### 6.2 `parseContentDispositionFilename`

职责：

1. 优先解析 `filename*=`。
2. 其次解析 `filename=`。
3. 处理 URL decode。
4. 处理中文文件名。

### 6.3 `parseBlobJsonError`

职责：

1. 读取 Blob 文本。
2. 尝试 JSON parse。
3. 从常见字段中提取错误：
   - `message`
   - `msg`
   - `error`
   - `data.message`
4. 解析失败则返回 `undefined`。

## 7. 报表中心页设计

文件：`frontend/src/pages/report/center/Index.vue`

### 7.1 列表继续只查 published

保留现有查询条件：

```ts
status: 'published'
```

报表中心仍然只展示已发布报表。

### 7.2 运行弹窗调用 `/run`

将当前：

```ts
reportApi.previewReport(...)
```

替换为：

```ts
reportApi.runReport(reportId, req)
```

运行请求继续复用当前参数、分页、dataset_id、menu_id、query 组装逻辑。

### 7.3 导出按钮调用 `/export`

将当前前端 CSV 导出替换为：

```ts
const file = await reportApi.exportReport(reportId, {
  format: 'csv',
  menu_id,
  dataset_id,
  parameters,
  query,
  max_rows
})
downloadBlob(file.blob, file.filename)
```

第一版可以不暴露复杂导出配置，默认使用后端 `5000` 行限制。

### 7.4 导出失败展示错误

失败时展示后端错误信息，例如：

```txt
导出行数超过系统限制，请缩小查询条件后重试
```

不应下载错误 JSON 文件。

### 7.5 是否展示当前运行版本号

建议轻量展示：

```txt
版本：V{previewData.meta.version_no}
```

没有版本号时不展示，保持兼容。

### 7.6 保留现有组件

保留：

- `ReportSheetPreview`
- `TablePagination`
- `SweetDateTimePicker`
- `BaseContent`
- 当前运行弹窗结构

## 8. 报表管理页设计

文件：`frontend/src/pages/report/manage/Index.vue`

### 8.1 发布按钮改为 `/publish`

当前发布逻辑使用状态接口，不再适合 V1-A 后端规则。

应从：

```ts
updateReportStatus(row.id, 'published')
```

改为：

```ts
reportApi.publishReport(row.id, { change_log })
```

### 8.2 发布交互

发布需要轻量确认弹窗，不要直接静默发布。

建议弹窗内容：

1. 标题：`发布报表`
2. 提示：`发布后报表中心将运行新的发布版本。`
3. 可选输入：`发布说明 change_log`
4. 操作：`取消` / `确认发布`

`change_log` 可选，不强制。

### 8.3 运行改为 `/run`

管理页运行弹窗改为：

```ts
reportApi.runReport(reportId, req)
```

不再用 `/preview` 作为运行接口。

### 8.4 导出改为 `/export`

当前前端 CSV 替换为后端导出：

```ts
reportApi.exportReport(reportId, req)
```

页面只调用：

```ts
downloadBlob(file.blob, file.filename)
```

### 8.5 版本列表入口

建议放在行操作区：

```txt
版本
```

位置建议在“设计”之后、“复制”之前，或者进入更多菜单。

点击后打开：

```ts
ReportVersionDialog
```

### 8.6 按钮启用禁用规则

`draft`：

- 可设计
- 可发布
- 不可运行
- 不可导出
- 可删除

`published`：

- 可运行
- 可导出
- 可设计
- 可查看版本
- 可停用
- 可复制

`disabled`：

- 可设计
- 可查看版本
- 不可运行
- 不可导出
- 可删除或按现有语义保留删除能力

V1-D 不新增恢复逻辑。

### 8.7 disabled 报表规则

V1-D 只处理：

1. disabled 报表不能运行。
2. disabled 报表不能导出。
3. disabled 报表不能设计时预览。

不新增恢复为草稿或重新启用的前端流程。

### 8.8 保留现有结构

保留：

- 列表字段
- 筛选区
- 指标卡片
- 运行弹窗
- `ReportSheetPreview`
- `TablePagination`

只替换 API 调用并新增版本入口。

## 9. 报表设计器页设计

文件：`frontend/src/pages/report/design/Index.vue`

### 9.1 保存草稿保持现有 create / update

保留现有：

```ts
saveReport('draft')
```

继续走：

```ts
createReport
updateReport
```

### 9.2 预览改为“保存并预览”

设计器预览必须明确是“保存并预览”。

逻辑：

1. 保存草稿。
2. 调用 `/design-preview`。

建议函数流程：

```ts
preview():
  validate
  saveReport('draft')
  reportApi.designPreviewReport(reportId, req)
```

按钮或 loading 文案需要体现保存动作，避免用户误解。例如：

```txt
保存并预览
```

或 loading：

```txt
保存并预览中...
```

### 9.3 发布改为“确认后保存并发布”

当前：

```ts
saveReport('published')
```

应改为：

```ts
publishReport():
  1. 打开轻量确认弹窗
  2. validate
  3. saveReport('draft')
  4. reportApi.publishReport(reportId, { change_log })
  5. 刷新当前报表详情
```

发布需要轻量确认，不要直接静默发布。

### 9.4 未保存新报表点击预览

建议行为：

```ts
先自动保存为 draft，再 design-preview
```

如果保存失败，则不调用预览。

### 9.5 disabled 报表预览和发布

如果当前报表状态是 `disabled`：

- 设计时预览：前端提示 `已停用报表不能设计时预览`。
- 发布：前端提示 `已停用报表不能发布` 或直接展示后端错误。

V1-D 不新增恢复逻辑。

### 9.6 顶部状态展示

可以轻量增加：

```txt
状态：草稿 / 已发布 / 已停用
线上版本：V{published_version_no}
```

如果需要修改 `ReportDesignerTopbar.vue`，只增加展示或文案，不调整整体布局。

### 9.7 不重写设计器核心

明确保留：

- `ReportSheetCanvas`
- `ReportInspectorPanel`
- `ReportResourcePanel`
- `ReportDatasetDialog`
- `ReportParameterDialog`
- `ReportJoinDialog`
- `ReportSheetPreview`
- 当前 sheet / cell / binding 模型

## 10. 版本列表组件设计

文件：

- `frontend/src/pages/report/components/ReportVersionDialog.vue`

### 10.1 props / emits

建议：

```ts
props:
  modelValue: boolean
  reportId?: number
  currentVersionId?: number

emits:
  update:modelValue
  refresh
```

第一版只读展示，不需要复杂事件。

### 10.2 查询时机

当弹窗打开且 `reportId` 存在时调用：

```ts
reportApi.queryReportVersions(reportId)
```

关闭后清理列表状态。

### 10.3 展示字段

最小字段：

- 版本号
- 状态
- 发布时间
- 发布人
- 发布说明
- 是否当前版本

发布人展示优先级：

```ts
published_name || published_by_name || published_by
```

### 10.4 当前版本标识

优先使用：

```ts
version.id === currentVersionId
```

如果接口返回 `is_current`，则使用：

```ts
version.is_current
```

展示文案：

```txt
当前发布版本
```

### 10.5 第一版不做

不做：

- 版本回滚
- 版本 diff
- 复制版本
- 按版本运行
- 查看完整 JSON 快照

## 11. V1-D 子任务拆分

### 11.1 V1-D-1：API service + types + blob 下载工具

范围：

- 修改 `frontend/src/api/services/report.ts`
- 修改 `frontend/src/modules/report/types.ts`
- 新增 `frontend/src/utils/download.ts`
- 如有必要，兼容 `frontend/src/boot/axios.ts` 的 blob 响应

验收：

- `publishReport` / `designPreviewReport` / `runReport` / `exportReport` / `queryReportVersions` 类型完整。
- `exportReport` 能返回 `ReportExportFile`。
- blob JSON 错误不会被下载为 CSV。

### 11.2 V1-D-2：ReportVersionDialog

范围：

- 新增 `frontend/src/pages/report/components/ReportVersionDialog.vue`

验收：

- 能打开弹窗。
- 能查询版本列表。
- 能展示当前版本。
- 不做回滚和 diff。

### 11.3 V1-D-3：报表中心页接 `/run` 和 `/export`

范围：

- 修改 `frontend/src/pages/report/center/Index.vue`

验收：

- 报表中心运行走 `/run`。
- 报表中心导出走 `/export`。
- 导出成功下载后端 CSV。
- 导出失败展示后端错误。

### 11.4 V1-D-4：报表管理页接 `/publish` / `/run` / `/export` / `/versions`

范围：

- 修改 `frontend/src/pages/report/manage/Index.vue`

验收：

- 发布走 `/publish`。
- 发布前有轻量确认。
- 运行走 `/run`。
- 导出走 `/export`。
- 能打开版本列表。
- disabled / draft 按钮规则符合预期。

### 11.5 V1-D-5：报表设计器接保存并预览 / 保存并发布

范围：

- 修改 `frontend/src/pages/report/design/Index.vue`
- 视情况轻量修改 `frontend/src/pages/report/design/components/ReportDesignerTopbar.vue`

验收：

- 预览体现“保存并预览”。
- 预览先保存草稿，再调用 `/design-preview`。
- 发布有轻量确认。
- 发布先保存草稿，再调用 `/publish`。
- disabled 报表不能设计时预览。

## 12. 测试和验证方案

### 12.1 构建和检查命令

项目使用 Yarn，前端目录存在 `yarn.lock`。

建议执行：

```bash
cd frontend
yarn lint
yarn typecheck
yarn build
```

完整 CI 脚本：

```bash
yarn ci
```

其内容为：

```bash
yarn lint && yarn typecheck && quasar build
```

### 12.2 手工验证流程

至少验证：

1. 设计器保存草稿。
2. 设计器“保存并预览”走 `/design-preview`。
3. 设计器发布走 `/publish`。
4. 报表中心运行走 `/run`。
5. 报表中心导出走 `/export`。
6. 报表管理页发布走 `/publish`。
7. 报表管理页运行走 `/run`。
8. 报表管理页导出走 `/export`。
9. 报表管理页版本列表能打开并展示版本。

### 12.3 验证修改草稿不影响报表中心

流程：

1. 发布报表 V1。
2. 打开报表中心运行，记录结果。
3. 回到设计器修改草稿但不发布。
4. 再次在报表中心运行。
5. 确认结果仍为 V1。
6. 再发布生成 V2。
7. 报表中心再次运行，确认结果变为 V2。

### 12.4 验证导出走后端接口

通过浏览器 Network 确认：

```http
POST /admin/report/:id/export
```

确认不再由前端拼接当前页 CSV。

### 12.5 验证 disabled 报表不能运行和导出

流程：

1. 管理页停用 published 报表。
2. 管理页运行按钮禁用或点击后提示错误。
3. 管理页导出按钮禁用或点击后提示错误。
4. 报表中心列表不展示 disabled 报表。
5. 直接触发运行 / 导出接口时展示后端错误。

## 13. 风险点

### 13.1 不能重写的前端代码

不要重写：

- `ReportSheetPreview.vue`
- `ReportSheetCanvas.vue`
- `ReportInspectorPanel.vue`
- `ReportResourcePanel.vue`
- `ReportDatasetDialog.vue`
- `ReportParameterDialog.vue`
- `ReportJoinDialog.vue`
- `schema.ts`
- `sheet.ts`
- `options.ts`

这些是当前轻量设计器的核心资产。

### 13.2 Axios blob 拦截器风险

当前 Axios 响应拦截器默认按 JSON 判断 `success`，对 blob 下载可能误判。

V1-D 必须处理：

```ts
responseType === 'blob'
```

否则 `/export` 可能无法正常下载，或者错误 JSON 被当成 CSV 文件下载。

### 13.3 预览前自动保存草稿风险

设计器预览改为“保存并预览”后，会改变用户心智：

```txt
点击预览也会保存当前编辑
```

因此按钮或 loading 文案必须体现保存动作。

### 13.4 旧 `previewReport` 兼容风险

`previewReport` 应保留，避免影响旧入口或未迁移代码。

但新逻辑中：

- 报表中心运行不用 `previewReport`
- 管理页运行不用 `previewReport`
- 设计器预览不用 `previewReport`

### 13.5 disabled 报表风险

V1-D 不新增恢复逻辑。

只处理：

- disabled 报表不能运行
- disabled 报表不能导出
- disabled 报表不能设计时预览

### 13.6 导出错误处理风险

导出失败时如果后端返回 JSON，而前端仍按 blob 文件下载，会导致用户下载错误文件。

必须通过 `parseBlobJsonError` 识别错误并展示。

## 14. 完成标准

V1-D 完成后应满足：

1. 报表中心只通过 `/run` 运行 published version。
2. 报表中心通过 `/export` 使用后端受控导出。
3. 报表管理页发布通过 `/publish`。
4. 报表管理页运行通过 `/run`。
5. 报表管理页导出通过 `/export`。
6. 报表管理页可查看版本列表。
7. 报表设计器预览是“保存并预览”，并调用 `/design-preview`。
8. 报表设计器发布是“确认后保存并发布”，并调用 `/publish`。
9. disabled 报表不能运行、不能导出、不能设计时预览。
10. 旧 `previewReport` 保留兼容，但不再作为新运行态入口。
