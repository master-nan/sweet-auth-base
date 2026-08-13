# 报表模块 V1-D 前端实现证据包

## 1. V1-D 范围说明

V1-D 只做前端接入，把已有报表前端页面接到 V1-A / V1-B 后端接口闭环：

- 接入设计时预览 `/admin/report/:id/design-preview`
- 接入发布 `/admin/report/:id/publish`
- 接入运行 `/admin/report/:id/run`
- 接入后端受控导出 `/admin/report/:id/export`
- 接入版本列表 `/admin/report/:id/versions`

本阶段不做：

- 后端改造
- 重写设计器
- `report_datasource`
- `report_dataset`
- 图表大屏
- 打印分页
- 填报
- 版本回滚
- 版本 diff
- 异步导出任务

## 2. API service 证据

文件路径：`frontend/src/api/services/report.ts`

组件/函数：`useReportApi`

关键代码片段：

```ts
  const publishReport = async (
    id: number,
    req?: ReportPublishReq,
  ): Promise<ReportPublishRes> => {
    return instance
      .post<ResponseData<ReportPublishRes>>(`/admin/report/${id}/publish`, req || {})
      .then((res) => res.data.data)
  }

  const previewReport = async (req: ReportPreviewReq) => {
    if (!req.report_id) {
      throw new Error('report_id is required for backend preview')
    }
    return instance
      .post<ResponseData<BackendReportPreview>>(
        `/admin/report/${req.report_id}/preview`,
        toPreviewPayload(req),
      )
      .then((res) => {
        return {
          ...res.data,
          data: toPreview(res.data.data),
        } as ResponseData<ReportPreviewRes>
      })
  }

  const designPreviewReport = async (
    id: number,
    req: ReportPreviewReq,
  ): Promise<ReportPreviewRes> => {
    return instance
      .post<ResponseData<BackendReportPreview>>(
        `/admin/report/${id}/design-preview`,
        toPreviewPayload(req),
      )
      .then((res) => toPreview(res.data.data))
  }

  const runReport = async (id: number, req: ReportPreviewReq): Promise<ReportPreviewRes> => {
    return instance
      .post<ResponseData<BackendReportPreview>>(`/admin/report/${id}/run`, toPreviewPayload(req))
      .then((res) => toPreview(res.data.data))
  }
```

说明：

- `previewReport` 仍保留，继续兼容旧 `/preview`。
- 新逻辑不再用 `previewReport` 作为报表中心运行接口，报表中心和管理页运行均调用 `runReport`。
- `designPreviewReport` 专门用于设计器保存草稿后的设计时预览。

关键代码片段：

```ts
  const exportReport = async (
    id: number,
    req: ReportExportReq,
  ): Promise<ReportExportFile> => {
    const format = req.format || 'csv'
    const fallbackFilename = `report_${id}.csv`

    try {
      const res = await instance.post<Blob>(
        `/admin/report/${id}/export`,
        {
          ...req,
          format,
        },
        {
          responseType: 'blob',
        },
      )
      const contentType =
        getResponseHeader(res.headers, 'content-type') || res.data.type || 'text/csv;charset=utf-8'
      const blob = toBlob(res.data, contentType)

      if (contentType.toLowerCase().includes('json')) {
        const errorMessage = await parseBlobJsonError(blob)
        throw new Error(errorMessage || '导出失败')
      }

      const filename =
        parseContentDispositionFilename(getResponseHeader(res.headers, 'content-disposition')) ||
        fallbackFilename

      return {
        blob,
        filename,
        contentType,
      }
    } catch (error) {
      const response = (error as { response?: { data?: unknown; headers?: unknown } }).response
      if (response?.data) {
        const contentType = getResponseHeader(response.headers, 'content-type')
        const blob = toBlob(response.data, contentType)
        const errorMessage = await parseBlobJsonError(blob)
        if (errorMessage) {
          throw new Error(errorMessage)
        }
      }
      throw error
    }
  }

  const queryReportVersions = async (id: number): Promise<ReportVersion[]> => {
    return instance
      .get<ResponseData<ReportVersion[]>>(`/admin/report/${id}/versions`)
      .then((res) => res.data.data || [])
  }
```

说明：

- `exportReport` 显式使用 `responseType: 'blob'`。
- 文件名优先从 `Content-Disposition` 解析，失败回退 `report_${id}.csv`。
- 如果后端以 blob 形式返回 JSON 错误，会调用 `parseBlobJsonError` 解析并抛出业务错误，不会把错误 JSON 下载成 CSV。

## 3. 类型定义证据

文件路径：`frontend/src/modules/report/types.ts`

类型：`Report`、`ReportPublishReq`、`ReportPublishRes`、`ReportVersion`、`ReportExportReq`、`ReportExportFile`、`ReportPreviewMeta`

关键代码片段：

```ts
  layout_config?: ReportLayoutConfig
  report_name: string
  report_code: string
  report_kind: ReportKind
  category?: string
  description?: string
  data_source_id?: number | string
  data_source_name?: string
  status: ReportStatus
  published_version_id?: number
  published_version_no?: number
  owner?: string
  updated_at?: string
}

export interface ReportPublishReq {
  change_log?: string
}

export interface ReportPublishRes {
  report_id: number
  version_id: number
  version_no: number
  status: ReportStatus
  published_at?: string
}

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

关键代码片段：

```ts
export interface ReportExportReq {
  format?: ReportExportFormat
  menu_id?: number | undefined
  dataset_id?: string | undefined
  parameters?: Record<string, unknown>
  params?: Record<string, unknown>
  query?: ReportQuery
  max_rows?: number
}

export interface ReportExportFile {
  blob: Blob
  filename: string
  contentType: string
}

export interface ReportPreviewMeta {
  report_id?: number
  report_code?: string
  source_code?: string
  dataset_id?: string
  dataset_type?: ReportDatasetType | string
  applied_menu_id?: number
  runtime_type?: ReportRuntimeType
  version_id?: number
  version_no?: number
}

export interface ReportPreviewRes {
  columns: ReportField[]
  rows: Record<string, unknown>[]
  total?: number
  datasets?: ReportDataset[]
  joins?: ReportDatasetJoin[]
  meta?: ReportPreviewMeta
}
```

说明：

- `Report` 增加 `published_version_id` 和 `published_version_no`，用于管理页、设计器顶部、版本弹窗的当前版本展示。
- `ReportPreviewMeta` 增加 `runtime_type`、`version_id`、`version_no`，用于运行弹窗展示版本信息。
- `ReportExportReq` 贴近后端结构，仅主动使用 `max_rows`，不主动传 `maxRows`。

## 4. Blob 下载证据

文件路径：`frontend/src/utils/download.ts`

函数：`parseContentDispositionFilename`、`parseBlobJsonError`、`downloadBlob`

关键代码片段：

```ts
export const parseContentDispositionFilename = (contentDisposition?: string): string => {
  if (!contentDisposition) return ''

  const encodedFilenameMatch = /filename\*\s*=\s*(?:UTF-8''|utf-8'')?([^;]+)/i.exec(
    contentDisposition,
  )
  if (encodedFilenameMatch?.[1]) {
    const value = encodedFilenameMatch[1].trim().replace(/^"|"$/g, '')
    try {
      return decodeURIComponent(value)
    } catch {
      return value
    }
  }

  const filenameMatch = /filename\s*=\s*([^;]+)/i.exec(contentDisposition)
  if (!filenameMatch?.[1]) return ''

  const value = filenameMatch[1].trim().replace(/^"|"$/g, '')
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

export const parseBlobJsonError = async (blob: Blob): Promise<string | undefined> => {
  if (!blob.size) return undefined

  try {
    const text = await blob.text()
    if (!text.trim()) return undefined

    const parsed = JSON.parse(text) as {
      message?: string
      msg?: string
      error?: string
      error_message?: string
      data?: { message?: string; msg?: string; error?: string; error_message?: string } | string
    }

    if (typeof parsed.data === 'string') return parsed.data

    return (
      parsed.error_message ||
      parsed.message ||
      parsed.msg ||
      parsed.error ||
      parsed.data?.error_message ||
      parsed.data?.message ||
      parsed.data?.msg ||
      parsed.data?.error
    )
  } catch {
    return undefined
  }
}

export const downloadBlob = (blob: Blob, filename: string): void => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename || 'download'
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
```

文件路径：`frontend/src/boot/axios.ts`

函数：Axios response interceptor

关键代码片段：

```ts
instance.interceptors.response.use(
  (response: AxiosResponse<ResponseData<any>>): AxiosResponse<ResponseData<any>> => {
    // 在拦截器内部获取 store 实例
    const loadingStore = useLoadingStore()

    loadingStore.setLoading(false)
    if (response.config.responseType === 'blob') {
      return response
    }

    const res = response.data
    if (!res.success) {
      notifyRequestError(
        res.error_message || '未知错误',
        `business:${response.config.method || 'get'}:${response.config.url || ''}:${res.error_code || res.code || ''}:${res.error_message || ''}`,
      )
      const error = new Error(res.error_message || '未知错误') as Error & {
        response: typeof response
      }
      error.response = response
      throw error
    } else {
      const method = (response.config.method || '').toLowerCase()
      const url = response.config.url || ''
      const isQueryApi = url.includes('/query')
      if (method !== 'get' && !isQueryApi) {
        Notify.create({
          position: 'top-right',
          progress: true,
          message: res.message as string,
          type: 'positive',
          timeout: 8 * 1000,
          actions: [
            {
              icon: 'close',
              color: 'white',
              round: true,
              size: 'sm',
            },
          ],
        })
      }
    }
    return response
  },
```

说明：

- 普通 JSON 接口仍走原有 `success` 判断、错误通知和成功通知逻辑。
- `responseType === 'blob'` 时直接返回响应，由 `reportApi.exportReport` 负责解析文件名和 JSON 错误。
- 错误 JSON 不会进入下载函数，因此不会被保存成 CSV。

## 5. 版本弹窗证据

文件路径：`frontend/src/pages/report/components/ReportVersionDialog.vue`

组件：`ReportVersionDialog`

关键代码片段：

```vue
<template>
  <q-dialog
    :model-value="modelValue"
    persistent
    @hide="clearState"
    @update:model-value="handleDialogValue"
  >
    <q-card class="version-dialog">
      <q-card-section class="dialog-head">
        <div>
          <div class="dialog-title">发布版本</div>
          <div class="dialog-caption">查看当前报表的发布快照，报表中心运行时只读取已发布版本。</div>
        </div>
      </q-card-section>

      <q-card-section class="dialog-body">
        <q-table
          flat
          bordered
          dense
          row-key="id"
          separator="cell"
          class="version-table"
          :rows="versions"
          :columns="columns"
          :loading="loading"
          :pagination="{ rowsPerPage: 0 }"
          hide-pagination
        >
          <template #body-cell-version_no="slotProps">
            <q-td :props="slotProps">
              <strong>V{{ slotProps.row.version_no }}</strong>
            </q-td>
          </template>

          <template #body-cell-status="slotProps">
            <q-td :props="slotProps">
              <q-chip
                dense
                square
                :color="statusColor(slotProps.row.status)"
                text-color="white"
              >
                {{ statusLabel(slotProps.row.status) }}
              </q-chip>
            </q-td>
          </template>
```

关键代码片段：

```ts
const props = defineProps<{
  modelValue: boolean
  reportId?: number | undefined
  currentVersionId?: number | undefined
  currentVersionNo?: number | undefined
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const versions = ref<ReportVersion[]>([])
const loading = ref(false)
const errorMessage = ref('')
let requestSeq = 0

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      void loadVersions()
      return
    }
    clearState()
  },
  { immediate: true },
)

watch(
  () => props.reportId,
  () => {
    if (props.modelValue) void loadVersions()
  },
)

function handleDialogValue(value: boolean) {
  emit('update:modelValue', value)
  if (!value) clearState()
}

async function loadVersions() {
  requestSeq += 1
  const currentRequest = requestSeq
  versions.value = []
  errorMessage.value = ''

  if (!props.reportId) {
    loading.value = false
    return
  }

  loading.value = true
  try {
    const items = await reportApi.queryReportVersions(props.reportId)
    if (currentRequest !== requestSeq || !props.modelValue) return
    versions.value = items
  } catch (error) {
```

关键代码片段：

```ts
function clearState() {
  requestSeq += 1
  versions.value = []
  errorMessage.value = ''
  loading.value = false
}

function publisherName(version: ReportVersion) {
  return version.published_name || version.published_by_name || version.published_by || '-'
}

function isCurrentVersion(version: ReportVersion) {
  if (version.is_current) return true
  if (props.currentVersionId !== undefined && version.id === props.currentVersionId) return true
  return (
    props.currentVersionNo !== undefined &&
    version.version_no === props.currentVersionNo
  )
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    draft: '草稿',
    published: '已发布',
    archived: '已归档',
    disabled: '已停用',
  }
  return labels[status] || status || '-'
}
```

说明：

- 打开弹窗且 `reportId` 存在时调用 `queryReportVersions(reportId)`。
- 关闭时调用 `clearState` 清理列表、错误和 loading。
- 使用 `q-dialog`、`q-card`、`q-table`、`q-chip` 展示版本号、状态、发布时间、发布人、发布说明、当前版本标识。
- 当前版本优先使用 `is_current`，其次比较 `currentVersionId` 和 `currentVersionNo`。
- 第一版只读，不做回滚、diff、复制、按版本运行。

## 6. 报表中心页接入证据

文件路径：`frontend/src/pages/report/center/Index.vue`

组件：`report_center`

关键代码片段：

```vue
            <div class="section-title">可运行报表</div>
            <div class="report-caption">只展示已发布并可运行的报表。</div>
          </div>

          <q-table
            flat
            bordered
            separator="cell"
            row-key="id"
            class="report-table"
            :dense="$q.screen.lt.md"
            :rows="filteredRows"
            :columns="columns"
            :loading="loading"
            v-model:pagination="pagination"
            hide-pagination
          >
            <template #body-cell-report_name="props">
              <q-td :props="props">
                <div class="report-name-cell">
                  <q-icon :name="kindIcon(props.row.report_kind)" color="primary" size="24px" />
                  <div>
                    <strong>{{ props.row.report_name }}</strong>
                    <span>{{ props.row.report_code }}</span>
                  </div>
                </div>
              </q-td>
            </template>
```

关键代码片段：

```ts
function buildListFilters() {
  const filters: Record<string, string> = { status: 'published' }
  if (activeCategory.value) filters.category = activeCategory.value
  return filters
}

async function openRuntime(row: Report) {
  runtimeReport.value = row
  runtimeKeyword.value = ''
  runtimeFilterValues.value = buildRuntimeDefaultFilters()
  runtimeData.value = { columns: [], rows: [], total: 0 }
  runtimePagination.value.page = 1
  runtimePagination.value.rowsPerPage = runtimeConfiguredPageSize.value
  runtimeVisible.value = true
  await loadRuntimePreview()
}

async function loadRuntimePreview() {
  if (!runtimeReport.value?.id) return
  runtimeLoading.value = true
  try {
    const res = await reportApi.runReport(
      runtimeReport.value.id,
      buildRuntimePreviewReq(),
    )
    runtimeData.value = res
    runtimePagination.value.rowsNumber = res.total ?? res.rows.length
  } catch {
    runtimeData.value = { columns: [], rows: [], total: 0 }
    runtimePagination.value.rowsNumber = 0
    $q.notify({ type: 'negative', message: '报表运行失败，请检查报表配置、数据权限或后端接口' })
  } finally {
    runtimeLoading.value = false
  }
}
```

关键代码片段：

```vue
          <report-sheet-preview
            :sheet="runtimeSheet"
            :datasets="runtimeDatasets"
            :preview-data="runtimeData"
            :loading="runtimeLoading"
            :report-kind="runtimeReport?.report_kind || 'detail'"
          />
```

关键代码片段：

```ts
function buildRuntimeExportReq(): ReportExportReq {
  const exportQuerySize = Math.max(
    runtimePagination.value.rowsNumber || 0,
    runtimeRows.value.length,
    runtimePagination.value.rowsPerPage,
    5000,
  )
  return {
    format: 'csv',
    dataset_id: runtimeDatasetId.value,
    menu_id: runtimeMenuId.value,
    parameters: buildRuntimeParameterValues(),
    query: buildRuntimeQuery(exportQuerySize),
  }
}

async function exportRuntimeCsv() {
  if (!runtimeReport.value?.id || !runtimeRows.value.length || runtimeExporting.value) return
  runtimeExporting.value = true
  try {
    const file = await reportApi.exportReport(
      runtimeReport.value.id,
      buildRuntimeExportReq(),
    )
    downloadBlob(file.blob, file.filename)
    $q.notify({ type: 'positive', message: '报表导出成功' })
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : '报表导出失败'
    $q.notify({ type: 'negative', message })
  } finally {
    runtimeExporting.value = false
  }
}
```

运行版本展示：

```vue
            <q-chip
              v-if="runtimeVersionNo"
              dense
              square
              color="primary"
              text-color="white"
              class="runtime-version-chip"
            >
              当前版本：V{{ runtimeVersionNo }}
            </q-chip>
```

指定搜索结果：

```bash
$ rg "previewReport" frontend/src/pages/report/center/Index.vue

$ rg "new Blob|createObjectURL|csv" frontend/src/pages/report/center/Index.vue
    format: 'csv',
```

说明：

- 报表中心列表查询仍固定 `status: 'published'`。
- 报表中心运行调用 `runReport`，不再使用 `/preview`。
- 报表中心导出调用 `exportReport + downloadBlob`，不再前端拼 CSV；`csv` 仅作为后端导出格式参数存在。
- 运行结果继续使用 `ReportSheetPreview`。

## 7. 报表管理页接入证据

文件路径：`frontend/src/pages/report/manage/Index.vue`

组件：`report_manage`

关键代码片段：

```vue
                  <q-btn
                    :disable="props.row.status !== 'published'"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="play_arrow"
                    @click="openRuntime(props.row)"
                  >
                    <q-tooltip>运行</q-tooltip>
                  </q-btn>
                  <q-btn
                    :disable="props.row.status !== 'published'"
                    :loading="exportingReportId === props.row.id"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="download"
                    @click="exportReportRow(props.row)"
                  >
                    <q-tooltip>导出</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="design_services"
                    @click="openDesigner(props.row)"
                  >
                    <q-tooltip>设计</q-tooltip>
                  </q-btn>
                  <q-btn
                    :disable="props.row.status === 'disabled'"
                    flat
                    size="sm"
                    round
                    color="positive"
                    icon="publish"
                    @click="publishReport(props.row)"
                  >
                    <q-tooltip>发布</q-tooltip>
                  </q-btn>
                  <q-btn
                    :disable="!canViewVersions(props.row)"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="history"
                    @click="openVersionDialog(props.row)"
                  >
                    <q-tooltip>版本</q-tooltip>
                  </q-btn>
```

关键代码片段：

```ts
async function loadRuntimePreview() {
  if (!runtimeReport.value?.id) return
  runtimeLoading.value = true
  try {
    const res = await reportApi.runReport(
      runtimeReport.value.id,
      buildRuntimePreviewReq(),
    )
    runtimeData.value = res
    runtimePagination.value.rowsNumber = res.total ?? res.rows.length
  } catch (error) {
    runtimeData.value = { columns: [], rows: [], total: 0 }
    runtimePagination.value.rowsNumber = 0
    const message = error instanceof Error && error.message
      ? error.message
      : '报表运行失败，请检查报表配置、数据权限或后端接口'
    $q.notify({ type: 'negative', message })
  } finally {
    runtimeLoading.value = false
  }
}

async function exportReportRow(row: Report) {
  if (row.status !== 'published' || exportingReportId.value) return
  exportingReportId.value = row.id
  try {
    const file = await reportApi.exportReport(row.id, buildReportExportReq(row))
    downloadBlob(file.blob, file.filename)
    $q.notify({ type: 'positive', message: '报表导出成功' })
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : '报表导出失败'
    $q.notify({ type: 'negative', message })
  } finally {
    exportingReportId.value = null
  }
}
```

关键代码片段：

```vue
    <report-version-dialog
      v-model="versionDialogVisible"
      :report-id="versionDialogReport?.id"
      :current-version-id="versionDialogReport?.published_version_id"
      :current-version-no="versionDialogReport?.published_version_no"
    />
```

关键代码片段：

```ts
function canViewVersions(row: Report) {
  return row.status !== 'draft' || Boolean(row.published_version_id || row.published_version_no)
}

function openVersionDialog(row: Report) {
  versionDialogReport.value = row
  versionDialogVisible.value = true
}

function confirmPublishReport(row: Report) {
  return new Promise<string | null>((resolve) => {
    $q.dialog({
      title: '发布报表',
      message: `确认发布「${row.report_name}」吗？发布后报表中心将运行新的发布版本。`,
      prompt: {
        model: '',
        type: 'textarea',
        label: '发布说明（可选）',
      },
      cancel: true,
      persistent: true,
    })
      .onOk((value) => resolve(String(value || '').trim()))
      .onCancel(() => resolve(null))
      .onDismiss(() => resolve(null))
  })
}

async function publishReport(row: Report) {
  if (row.status === 'disabled') return
  const changeLog = await confirmPublishReport(row)
  if (changeLog === null) return
  try {
    await reportApi.publishReport(row.id, changeLog ? { change_log: changeLog } : {})
    $q.notify({ type: 'positive', message: '报表已发布' })
    await fetchData()
  } catch (error) {
```

关键代码片段：

```ts
async function changeReportStatus(row: Report, status: ReportStatus) {
  if (status === 'published') {
    await publishReport(row)
    return
  }
  const actionText = status === 'draft' ? '改为草稿' : '停用'
  const confirmed = await new Promise<boolean>((resolve) => {
    $q.dialog({
      title: `${actionText}报表`,
      message: `确认${actionText}「${row.report_name}」吗？`,
      cancel: true,
      persistent: true,
    })
      .onOk(() => resolve(true))
      .onCancel(() => resolve(false))
      .onDismiss(() => resolve(false))
  })
  if (!confirmed) return
  try {
    await reportApi.updateReportStatus(row.id, status)
    $q.notify({ type: 'positive', message: `报表已${actionText}` })
    await fetchData()
  } catch {
    $q.notify({ type: 'negative', message: `${actionText}失败` })
  }
}
```

指定搜索结果：

```bash
$ rg "updateReportStatus.*published|saveReport\('published'\)|previewReport" frontend/src/pages/report/manage/Index.vue

$ rg "new Blob|createObjectURL|csv" frontend/src/pages/report/manage/Index.vue
    format: 'csv',
  if (!runtimeReport.value) return { format: 'csv' }
```

说明：

- 发布入口从直接更新状态改为 `publishReport(row.id, { change_log })`。
- `changeReportStatus(row, 'published')` 只作为兼容兜底转入 `publishReport`，不会直接调用 status 接口发布。
- 运行改为 `/run`，导出改为 `/export + downloadBlob`。
- 管理页行操作已按状态控制：draft 不能运行/导出，published 可运行/导出/发布/版本/停用，disabled 不能运行/导出/发布。
- `csv` 仅作为后端导出格式参数存在，没有 `new Blob` 或 `createObjectURL` 的前端拼装导出逻辑。

## 8. 报表设计器页接入证据

文件路径：`frontend/src/pages/report/design/Index.vue`

组件：`report_design`

关键代码片段：

```vue
    <report-designer-topbar
      :report-name="form.report_name"
      :report-code="form.report_code"
      :primary-source-code="primaryDataset?.source_code"
      :report-status="form.status || 'draft'"
      :published-version-no="publishedVersionNo"
      :saving="saving"
      :previewing="saving || previewLoading"
      :publishing="saving || publishing"
      :preview-disabled="form.status === 'disabled'"
      :publish-disabled="form.status === 'disabled'"
      :version-disabled="!form.id"
      @update:report-name="form.report_name = $event"
      @update:report-code="form.report_code = $event"
      @back="goBack"
      @add-parameter="addParameter"
      @preview="preview"
      @validate="validateAndNotify"
      @save-draft="saveReport('draft')"
      @publish="publishReport"
      @versions="openVersionDialog"
    />
```

关键代码片段：

```ts
async function preview() {
  if (form.status === 'disabled') {
    $q.notify({ type: 'warning', message: '已停用报表不能设计时预览' })
    return
  }
  const id = await saveReport('draft', { strict: true, notify: false })
  if (!id) return
  previewDialogVisible.value = true
  previewLoading.value = true
  try {
    const res = await reportApi.designPreviewReport(id, {
      report_id: id,
      dataset_id: datasetJoins.value.length ? undefined : primaryDataset.value?.id,
      data_source_id: primaryDataset.value?.source_code,
      page: 1,
      num: form.runtime_display === 'all' ? 10000 : Number(form.runtime_page_size || 20),
    })
    previewData.value = res
  } catch (error) {
    buildLocalPreview()
    const message = error instanceof Error && error.message
      ? error.message
      : '设计时预览失败，已显示本地结构预览'
    $q.notify({ type: 'negative', message })
  } finally {
    previewLoading.value = false
  }
}

async function saveReport(
  status: 'draft' = 'draft',
  options: { strict?: boolean; notify?: boolean } = {},
): Promise<number | null> {
  syncForm()
  const strict = options.strict ?? false
  const shouldNotify = options.notify ?? true
  if (!validateReport(strict)) return null
```

关键代码片段：

```ts
async function publishReport() {
  if (form.status === 'disabled') {
    $q.notify({ type: 'warning', message: '已停用报表不能发布' })
    return
  }
  const id = await saveReport('draft', { strict: true, notify: false })
  if (!id) return
  const changeLog = await confirmPublishReport()
  if (changeLog === null) return
  publishing.value = true
  try {
    const res = await reportApi.publishReport(id, changeLog ? { change_log: changeLog } : {})
    form.status = res.status || 'published'
    publishedVersionId.value = res.version_id
    publishedVersionNo.value = res.version_no
    await refreshCurrentReport(id)
    $q.notify({ type: 'positive', message: '报表已发布，可在报表中心运行' })
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : '发布失败'
    $q.notify({ type: 'negative', message })
  } finally {
    publishing.value = false
  }
}

function confirmPublishReport() {
  return new Promise<string | null>((resolve) => {
    $q.dialog({
      title: '发布报表',
      message: '确认发布当前报表吗？发布后报表中心将运行新的发布版本。',
      prompt: {
        model: '',
        type: 'textarea',
        label: '发布说明（可选）',
      },
      cancel: true,
      persistent: true,
    })
```

关键代码片段：

```vue
    <report-version-dialog
      v-model="versionDialogVisible"
      :report-id="form.id"
      :current-version-id="publishedVersionId"
      :current-version-no="publishedVersionNo"
    />
```

文件路径：`frontend/src/pages/report/design/components/ReportDesignerTopbar.vue`

组件：`ReportDesignerTopbar`

关键代码片段：

```vue
      <div class="status-strip">
        <q-chip dense square :color="statusColor" text-color="white">
          {{ statusLabel }}
        </q-chip>
        <q-chip
          v-if="publishedVersionNo"
          dense
          square
          outline
          color="primary"
        >
          线上版本 V{{ publishedVersionNo }}
        </q-chip>
      </div>
    </div>

    <div class="topbar-actions">
      <q-btn outline color="primary" icon="tune" label="参数" @click="$emit('addParameter')" />
      <q-btn
        outline
        color="primary"
        icon="preview"
        label="保存并预览"
        :disable="previewDisabled"
        :loading="previewing"
        @click="$emit('preview')"
      />
      <q-btn outline color="primary" icon="rule" label="校验" @click="$emit('validate')" />
      <q-btn
        unelevated
        color="primary"
        icon="save"
        label="保存草稿"
        :loading="saving"
        @click="$emit('saveDraft')"
      />
      <q-btn
        unelevated
        color="primary"
        icon="publish"
        label="保存并发布"
        :disable="publishDisabled"
        :loading="publishing"
        @click="$emit('publish')"
      />
      <q-btn
        outline
        color="primary"
        icon="history"
        label="版本"
        :disable="versionDisabled"
        @click="$emit('versions')"
      />
```

指定搜索结果：

```bash
$ rg "saveReport\('published'\)|previewReport" frontend/src/pages/report/design

$ rg "designPreviewReport|publishReport" frontend/src/pages/report/design
frontend/src/pages/report/design/Index.vue:      @publish="publishReport"
frontend/src/pages/report/design/Index.vue:    const res = await reportApi.designPreviewReport(id, {
frontend/src/pages/report/design/Index.vue:async function publishReport() {
frontend/src/pages/report/design/Index.vue:    const res = await reportApi.publishReport(id, changeLog ? { change_log: changeLog } : {})
```

说明：

- 预览流程为：校验配置 -> 保存草稿 -> `designPreviewReport`。
- 发布流程为：校验配置 -> 保存草稿 -> 确认弹窗 -> `publishReport`。
- 设计器内没有 `saveReport('published')`。
- 设计器主预览流程不再使用 `previewReport`。
- disabled 报表禁用“保存并预览 / 保存并发布”，函数内也有提示兜底。
- 未改动 `ReportSheetPreview`、`ReportSheetCanvas`、`ReportInspectorPanel`、`ReportResourcePanel`。

## 9. 核心组件未重写证据

检查文件：

- `frontend/src/pages/report/components/ReportSheetPreview.vue`
- `frontend/src/pages/report/design/components/ReportSheetCanvas.vue`
- `frontend/src/pages/report/design/components/ReportInspectorPanel.vue`
- `frontend/src/pages/report/design/components/ReportResourcePanel.vue`
- `frontend/src/modules/report/schema.ts`
- `frontend/src/modules/report/sheet.ts`
- `frontend/src/modules/report/options.ts`

指定检查命令结果：

```bash
$ git diff --name-only -- frontend/src/pages/report/components/ReportSheetPreview.vue frontend/src/pages/report/design/components/ReportSheetCanvas.vue frontend/src/pages/report/design/components/ReportInspectorPanel.vue frontend/src/pages/report/design/components/ReportResourcePanel.vue frontend/src/modules/report/schema.ts frontend/src/modules/report/sheet.ts frontend/src/modules/report/options.ts

```

说明：

- 命令无输出，表示上述核心组件和报表模型工具文件没有被本次 V1-D 修改。

## 10. 构建验证证据

执行目录：`frontend`

Node 版本：`v22.22.0`

命令结果：

```bash
$ yarn lint
yarn run v1.22.21
$ eslint -c ./eslint.config.js "./src*/**/*.{ts,js,cjs,mjs,vue}"
Done in 12.49s.
```

```bash
$ yarn typecheck
yarn run v1.22.21
$ vue-tsc --noEmit
Done in 5.88s.
```

```bash
$ yarn build
yarn run v1.22.21
$ quasar build
Build succeeded
Done in 3.44s.
```

构建 warning：

- `ag-psd` 依赖中的 `util` 被 externalized for browser compatibility。
- 部分 chunk 超过 900 kB。

说明：

- warning 未阻断构建。
- `quasar build` 最终输出 `Build succeeded`。

## 11. V1-D 验收清单

| 验收项 | 结果 | 证据文件 |
| --- | --- | --- |
| API service 已新增 publish / design-preview / run / export / versions | yes | `frontend/src/api/services/report.ts` |
| 报表中心运行调用 `/run` | yes | `frontend/src/pages/report/center/Index.vue` |
| 报表中心导出调用 `/export` | yes | `frontend/src/pages/report/center/Index.vue` |
| 报表中心不再前端拼 CSV | yes | `frontend/src/pages/report/center/Index.vue` |
| 报表管理页发布调用 `/publish` | yes | `frontend/src/pages/report/manage/Index.vue` |
| 报表管理页运行调用 `/run` | yes | `frontend/src/pages/report/manage/Index.vue` |
| 报表管理页导出调用 `/export` | yes | `frontend/src/pages/report/manage/Index.vue` |
| 报表管理页能打开版本列表 | yes | `frontend/src/pages/report/manage/Index.vue`、`frontend/src/pages/report/components/ReportVersionDialog.vue` |
| 报表设计器预览调用 `/design-preview` | yes | `frontend/src/pages/report/design/Index.vue` |
| 报表设计器发布调用 `/publish` | yes | `frontend/src/pages/report/design/Index.vue` |
| 报表设计器不再 `saveReport('published')` | yes | `frontend/src/pages/report/design/Index.vue` |
| disabled 报表不能运行和导出 | yes | `frontend/src/pages/report/manage/Index.vue` |
| disabled 报表不能设计时预览和发布 | yes | `frontend/src/pages/report/design/Index.vue`、`ReportDesignerTopbar.vue` |
| 旧 `previewReport` 方法保留兼容 | yes | `frontend/src/api/services/report.ts` |
| 未重写设计器核心组件 | yes | `ReportSheetPreview.vue`、`ReportSheetCanvas.vue`、`ReportInspectorPanel.vue`、`ReportResourcePanel.vue` 无 diff |

## 12. 已知风险

- 前端尚未做真实后端联调，仍需要在实际登录、权限、Casbin 按钮权限和数据权限环境下手工验收。
- 线上版本号展示依赖后端返回 `published_version_no` 或运行结果 `meta.version_no`。
- blob 错误解析依赖后端错误 JSON 字段结构，目前兼容 `error_message`、`message`、`msg`、`error` 和 `data` 中的同名字段。
- 设计器“保存并预览”会写入草稿，这是预期行为，因为后端 `/design-preview` 读取数据库草稿快照，不读取前端内存状态。
- 建议 V1-D 后续补 e2e 或手工验收脚本，覆盖：草稿保存、保存并预览、保存并发布、中心运行、中心导出、管理页版本列表、disabled 禁用态。
