# V1-E-1 运行弹窗抽取证据包

## 1. V1-E-1 范围说明

本次 V1-E-1 只做前端运行能力抽取：

- 抽取 `ReportRuntimeDialog`
- 抽取 `useReportRuntime`
- 抽取 `useReportExport`
- `center/manage` 复用运行弹窗

本次不做：

- 不改后端
- 不改设计器
- 不改 `ReportSheetPreview`
- 不抽 `ReportWorkspace`
- 不合并菜单
- 不做快速表格模式
- 不做图表大屏
- 不做打印分页
- 不做填报
- 不做版本回滚

## 2. ReportRuntimeDialog 证据

文件：`frontend/src/pages/report/components/ReportRuntimeDialog.vue`

组件：`ReportRuntimeDialog`

关键代码片段：

```vue
<template>
  <q-dialog
    :model-value="modelValue"
    maximized
    @hide="handleDialogValue(false)"
    @update:model-value="handleDialogValue"
  >
    <q-card class="runtime-dialog">
      <q-card-section class="runtime-head">
        <div>
          <div class="report-title">{{ runtimeReport?.report_name || runtimeTitle }}</div>
          <div class="report-caption">
            {{ runtimeReport?.data_source_name || '-' }} ·
            {{ runtimeReport?.description || '运行预览会应用后端数据权限' }}
          </div>
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
        </div>
        <q-space />
        <q-btn
          v-if="allowExport"
          outline
          color="primary"
          icon="download"
          label="导出 CSV"
          :disable="!runtimeRows.length || exporting"
          :loading="exporting"
          @click="exportCurrentRuntime"
        />
        <q-btn flat round icon="close" @click="handleDialogValue(false)" />
      </q-card-section>
      <q-separator />
      <q-card-section class="runtime-filters">
        <q-input
          v-model="runtimeKeyword"
          dense
          outlined
          clearable
          label="关键词"
          class="runtime-filter"
          @keyup.enter="loadRuntimePreview"
        />
```

说明：弹窗顶部展示报表名称、说明、`meta.version_no` 当前版本；导出按钮不在页面内拼 CSV，而是触发 `exportCurrentRuntime`。

```vue
        <template v-for="param in runtimeParameters" :key="param.id">
          <sweet-date-time-picker
            v-if="param.type === 'date'"
            :model-value="runtimeScalarValue(param.id)"
            type="date"
            dense
            :label="param.label"
            class="runtime-filter"
            @update:model-value="runtimeFilterValues[param.id] = $event"
          />
          <div v-else-if="param.type === 'date_range'" class="runtime-range-filter">
            <sweet-date-time-picker
              :model-value="runtimeRangeValue(param.id, 0)"
              type="date"
              dense
              :label="`${param.label}开始`"
              class="runtime-filter"
              @update:model-value="setRuntimeRangeValue(param.id, 0, $event)"
            />
            <sweet-date-time-picker
              :model-value="runtimeRangeValue(param.id, 1)"
              type="date"
              dense
              :label="`${param.label}结束`"
              class="runtime-filter"
              @update:model-value="setRuntimeRangeValue(param.id, 1, $event)"
            />
          </div>
          <q-input
            v-else
            :model-value="runtimeScalarValue(param.id)"
            dense
            outlined
            clearable
            :type="param.type === 'number' ? 'number' : 'text'"
            :label="param.label"
            :placeholder="param.placeholder"
            class="runtime-filter"
            @update:model-value="runtimeFilterValues[param.id] = $event"
            @keyup.enter="loadRuntimePreview"
          />
        </template>
        <q-select
          dense
          outlined
          label="权限范围"
          model-value="继承当前菜单数据权限"
          class="runtime-filter"
          :options="['继承当前菜单数据权限']"
        />
        <q-btn color="primary" icon="search" label="查询" @click="loadRuntimePreview" />
        <q-btn
          outline
          color="primary"
          icon="restart_alt"
          label="重置"
          @click="resetRuntimeFilters"
        />
```

说明：参数区保留 `date`、`date_range`、普通输入参数；查询按钮调用 `loadRuntimePreview`，重置按钮调用 `resetRuntimeFilters`。

```vue
      <q-card-section class="runtime-body">
        <report-sheet-preview
          :sheet="runtimeSheet"
          :datasets="runtimeDatasets"
          :preview-data="runtimeData"
          :loading="runtimeLoading"
          :report-kind="runtimeReport?.report_kind || 'detail'"
        />
        <div v-if="runtimeDisplayMode === 'paged'" class="runtime-pagination">
          <table-pagination
            v-model:page="runtimePagination.page"
            v-model:pageSize="runtimePagination.rowsPerPage"
            :total="runtimePagination.rowsNumber"
            @update:page="loadRuntimePreview"
          />
        </div>
        <div v-else class="runtime-pagination">
          <span>共 {{ runtimePagination.rowsNumber }} 行</span>
          <q-badge color="primary" label="全部展示" />
        </div>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import TablePagination from 'components/Table/TablePagination.vue'
import SweetDateTimePicker from 'components/DateTime/SweetDateTimePicker.vue'
import type { Report } from 'src/api/services/report'
import ReportSheetPreview from './ReportSheetPreview.vue'
import { useReportRuntime } from '../composables/useReportRuntime'
import { useReportExport } from '../composables/useReportExport'
```

说明：结果展示继续使用 `ReportSheetPreview`；分页变化触发 `loadRuntimePreview`，仍走 `/run`。

```ts
const props = withDefaults(defineProps<{
  modelValue: boolean
  report: Report | null
  defaultPageSize?: number | undefined
  allowExport?: boolean
  mode?: 'center' | 'manage'
}>(), {
  allowExport: true,
  mode: 'center',
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const {
  runtimeReport,
  runtimeData,
  runtimeLoading,
  runtimeKeyword,
  runtimeFilterValues,
  runtimePagination,
  runtimeRows,
  runtimeDatasets,
  runtimeSheet,
  runtimeDisplayMode,
  runtimeVersionNo,
  runtimeParameters,
  openRuntime,
  clearRuntime,
  loadRuntimePreview,
  resetRuntimeFilters,
  buildRuntimeParameterValues,
  runtimeScalarValue,
  runtimeRangeValue,
  setRuntimeRangeValue,
} = useReportRuntime()

const { exporting, exportRuntimeCsv } = useReportExport()
```

说明：组件 props/emits 明确；运行状态来自 `useReportRuntime`，导出能力来自 `useReportExport`。

```ts
watch(
  () => props.modelValue,
  (visible) => {
    if (visible && props.report) {
      openRuntime(props.report, props.defaultPageSize)
      void loadRuntimePreview()
      return
    }
    if (!visible) clearRuntime()
  },
  { immediate: true },
)

watch(
  () => props.report?.id,
  () => {
    if (props.modelValue && props.report) {
      openRuntime(props.report, props.defaultPageSize)
      void loadRuntimePreview()
    }
  },
)

function handleDialogValue(value: boolean) {
  emit('update:modelValue', value)
  if (!value) clearRuntime()
}

async function exportCurrentRuntime() {
  await exportRuntimeCsv(runtimeReport.value, {
    keyword: runtimeKeyword.value,
    parameters: buildRuntimeParameterValues(),
    total: runtimePagination.value.rowsNumber,
    rowCount: runtimeRows.value.length,
    pageSize: runtimePagination.value.rowsPerPage,
  })
}
```

说明：打开时初始化并运行，关闭时清理状态；导出通过 composable 走后端 `/export`，没有 `previewReport`，没有前端拼 CSV。

## 3. useReportRuntime 证据

文件：`frontend/src/pages/report/composables/useReportRuntime.ts`

函数：`useReportRuntime`

关键代码片段：

```ts
export function useReportRuntime() {
  const $q = useQuasar()
  const reportApi = useReportApi()

  const runtimeReport = ref<Report | null>(null)
  const runtimeData = ref<ReportPreviewRes>({ columns: [], rows: [], total: 0 })
  const runtimeLoading = ref(false)
  const runtimeKeyword = ref('')
  const runtimeFilterValues = ref<Record<string, ReportRuntimeFilterValue>>({})
  const runtimePagination = ref({
    page: 1,
    rowsPerPage: 20,
    rowsNumber: 0,
  })

  const runtimeRows = computed(() => runtimeData.value.rows)
  const runtimeDatasets = computed<ReportDataset[]>(() =>
    runtimeReport.value?.layout_config?.datasets?.length
      ? runtimeReport.value.layout_config.datasets
      : runtimeData.value.datasets || [],
  )
  const runtimeSheet = computed<ReportSheetConfig>(
    () => runtimeReport.value?.layout_config?.sheet || defaultReportSheet(),
  )
  const runtimeDisplayMode = computed(
    () => runtimeReport.value?.layout_config?.runtime_display || 'paged',
  )
  const runtimeConfiguredPageSize = computed(() =>
    Number(runtimeReport.value?.layout_config?.runtime_page_size || 20),
  )
  const runtimePrimaryDataset = computed(() => {
    const datasets = runtimeDatasets.value
    return datasets.find((dataset) => dataset.primary) || datasets[0]
  })
  const runtimeDatasetId = computed(() => runtimePrimaryDataset.value?.id || '')
  const runtimeSourceCode = computed(() =>
    String(runtimePrimaryDataset.value?.source_code || runtimeReport.value?.data_source_id || ''),
  )
  const runtimeMenuId = computed(() => runtimeReport.value?.permission_menu_id || 0)
  const runtimeVersionNo = computed(() => runtimeData.value.meta?.version_no)
```

说明：集中管理 `runtimeData`、`runtimeLoading`、`runtimeKeyword`、`runtimeFilterValues`、`runtimePagination`，页面不再重复维护这些状态。

```ts
  function openRuntime(report: Report, defaultPageSize?: number) {
    runtimeReport.value = report
    runtimeKeyword.value = ''
    runtimeFilterValues.value = buildRuntimeDefaultFilters()
    runtimeData.value = { columns: [], rows: [], total: 0 }
    runtimePagination.value.page = 1
    runtimePagination.value.rowsPerPage = Number(
      defaultPageSize || runtimeConfiguredPageSize.value || 20,
    )
    runtimePagination.value.rowsNumber = 0
  }

  function clearRuntime() {
    runtimeReport.value = null
    runtimeKeyword.value = ''
    runtimeFilterValues.value = {}
    runtimeData.value = { columns: [], rows: [], total: 0 }
    runtimePagination.value.page = 1
    runtimePagination.value.rowsPerPage = 20
    runtimePagination.value.rowsNumber = 0
    runtimeLoading.value = false
  }

  async function loadRuntimePreview() {
    if (!runtimeReport.value?.id) return false
    runtimeLoading.value = true
    try {
      const res = await reportApi.runReport(
        runtimeReport.value.id,
        buildRuntimePreviewReq(),
      )
      runtimeData.value = res
      runtimePagination.value.rowsNumber = res.total ?? res.rows.length
      return true
    } catch (error) {
      runtimeData.value = { columns: [], rows: [], total: 0 }
      runtimePagination.value.rowsNumber = 0
      const message = error instanceof Error && error.message
        ? error.message
        : '报表运行失败，请检查报表配置、数据权限或后端接口'
      $q.notify({ type: 'negative', message })
      return false
    } finally {
      runtimeLoading.value = false
    }
  }
```

说明：运行只调用 `reportApi.runReport`，没有回退 `previewReport`。

```ts
  function resetRuntimeFilters() {
    runtimeKeyword.value = ''
    runtimeFilterValues.value = buildRuntimeDefaultFilters()
    runtimePagination.value.page = 1
    void loadRuntimePreview()
  }

  function buildRuntimeDefaultFilters() {
    const values: Record<string, ReportRuntimeFilterValue> = {}
    runtimeParameters.value.forEach((param) => {
      if (
        param.default_value === null ||
        param.default_value === undefined ||
        param.default_value === ''
      )
        return
      if (Array.isArray(param.default_value)) {
        const next = param.default_value.filter(
          (item) => item !== '' && item !== null && item !== undefined,
        )
        if (next.length) values[param.id] = next
        return
      }
      values[param.id] = param.default_value
    })
    return values
  }

  function buildRuntimeParameterValues() {
    const values: Record<string, unknown> = {}
    runtimeParameters.value.forEach((param) => {
      const value = runtimeFilterValues.value[param.id]
      if (value === '' || value === null || value === undefined) return
      if (
        Array.isArray(value) &&
        !value.some((item) => item !== '' && item !== null && item !== undefined)
      )
        return
      values[param.id] = value
    })
    return values
  }
```

说明：保留默认参数、重置参数、参数值构造逻辑。

```ts
  function buildRuntimePreviewReq(): ReportPreviewReq {
    return {
      report_id: runtimeReport.value?.id,
      dataset_id: runtimeDatasetId.value,
      menu_id: runtimeMenuId.value,
      data_source_id: runtimeSourceCode.value,
      page: runtimeDisplayMode.value === 'all' ? 1 : runtimePagination.value.page,
      num: runtimeDisplayMode.value === 'all' ? 10000 : runtimePagination.value.rowsPerPage,
      keyword: runtimeKeyword.value,
      parameters: buildRuntimeParameterValues(),
    }
  }

  function runtimeScalarValue(id: string) {
    const value = runtimeFilterValues.value[id]
    return Array.isArray(value) ? String(value[0] || '') : value === undefined ? null : String(value)
  }

  function runtimeRangeValue(id: string, index: number) {
    const value = runtimeFilterValues.value[id]
    return Array.isArray(value) ? String(value[index] || '') : ''
  }

  function setRuntimeRangeValue(id: string, index: number, value: string | null) {
    const current = Array.isArray(runtimeFilterValues.value[id])
      ? [...(runtimeFilterValues.value[id] as Array<string | number>)]
      : ['', '']
    current[index] = value || ''
    runtimeFilterValues.value[id] = current
  }
```

说明：`text` / `number` 使用 `runtimeScalarValue`；`date` 使用日期控件单值；`date_range` 使用 `runtimeRangeValue` 和 `setRuntimeRangeValue` 保留双日期输入。分页组件的 `@update:page="loadRuntimePreview"` 会在页码变化和 pageSize 重置页码时继续调用 `/run`。

## 4. useReportExport 证据

文件：`frontend/src/pages/report/composables/useReportExport.ts`

函数：`useReportExport`

关键代码片段：

```ts
export function useReportExport() {
  const $q = useQuasar()
  const reportApi = useReportApi()
  const exporting = ref(false)
  const exportingReportId = ref<number | null>(null)

  function resolvePrimaryDataset(report: Report) {
    const datasets = report.layout_config?.datasets?.length
      ? report.layout_config.datasets
      : report.query_config?.datasets || []
    return datasets.find((dataset) => dataset.primary) || datasets[0]
  }

  function buildReportExportReq(
    report: Report,
    options: ReportExportOptions = {},
  ): ReportExportReq {
    const dataset = resolvePrimaryDataset(report)
    const sourceCode = String(dataset?.source_code || report.data_source_id || '')
    const querySize = Math.max(
      options.total || 0,
      options.rowCount || 0,
      options.pageSize || 0,
      5000,
    )
    return {
      format: 'csv',
      dataset_id: dataset?.id || '',
      menu_id: report.permission_menu_id || 0,
      parameters: options.parameters || {},
      query: {
        page: 1,
        num: querySize,
        table_code: sourceCode,
        expressions: [],
        quick_query: {
          keyword: options.keyword || '',
        },
        filters: {},
        include_deleted: false,
      },
    }
  }
```

说明：`buildReportExportReq` 是本次抽取后的导出请求构造函数，等价承接原页面内 `buildRuntimeExportReq` 职责。

```ts
  async function exportReportWithReq(report: Report, req: ReportExportReq) {
    const file = await reportApi.exportReport(report.id, req)
    downloadBlob(file.blob, file.filename)
    $q.notify({ type: 'positive', message: '报表导出成功' })
  }

  async function exportRuntimeCsv(report: Report | null, options: ReportExportOptions = {}) {
    if (!report?.id || exporting.value) return false
    exporting.value = true
    try {
      await exportReportWithReq(report, buildReportExportReq(report, options))
      return true
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : '报表导出失败'
      $q.notify({ type: 'negative', message })
      return false
    } finally {
      exporting.value = false
    }
  }

  async function exportReportRow(row: Report) {
    if (row.status !== 'published' || exportingReportId.value) return false
    exportingReportId.value = row.id
    try {
      await exportReportWithReq(row, buildReportExportReq(row))
      return true
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : '报表导出失败'
      $q.notify({ type: 'negative', message })
      return false
    } finally {
      exportingReportId.value = null
    }
  }
```

说明：导出只调用 `reportApi.exportReport` 和 `downloadBlob`；没有恢复 `new Blob` / `createObjectURL` / 前端拼 CSV。`exportReportRow` 只允许 `published`，因此 `draft` / `disabled` 不会导出。

## 5. center 页面改造证据

文件：`frontend/src/pages/report/center/Index.vue`

组件：`report_center`

关键代码片段：

```vue
    <report-runtime-dialog
      v-model="runtimeVisible"
      :report="runtimeReport"
      mode="center"
      :allow-export="true"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_center' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import type { Query } from 'src/types/global'
import {
  useReportApi,
  type Report,
  type ReportKind,
} from 'src/api/services/report'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import ReportRuntimeDialog from '../components/ReportRuntimeDialog.vue'
```

说明：center 页面改为挂载 `ReportRuntimeDialog`，不再内置运行弹窗模板。

```ts
const dataSourceCount = ref(0)
const runtimeVisible = ref(false)
const runtimeReport = ref<Report | null>(null)

async function fetchData() {
  try {
    query.value.filters = buildListFilters()
    const res = await reportApi.queryReports(query.value)
    rows.value = res.data || []
    total.value = res.total ?? rows.value.length
    pagination.value.page = query.value.page
    pagination.value.rowsNumber = total.value
  } catch {
    rows.value = []
    total.value = 0
    pagination.value.rowsNumber = 0
    $q.notify({ type: 'negative', message: '报表列表加载失败，请检查后端服务或接口权限' })
  }
}

function buildListFilters() {
  const filters: Record<string, string> = { status: 'published' }
  if (activeCategory.value) filters.category = activeCategory.value
  return filters
}

function openRuntime(row: Report) {
  runtimeReport.value = row
  runtimeVisible.value = true
}
```

说明：center 只保留 `runtimeVisible` / `runtimeReport`，`openRuntime(row)` 只负责打开弹窗；列表仍固定 `status: published`。

搜索结果：

```bash
$ rg "previewReport" frontend/src/pages/report/center/Index.vue
# 无输出

$ rg "new Blob|createObjectURL" frontend/src/pages/report/center/Index.vue
# 无输出

$ rg "ReportRuntimeDialog|report-runtime-dialog" frontend/src/pages/report/center/Index.vue
    <report-runtime-dialog
import ReportRuntimeDialog from '../components/ReportRuntimeDialog.vue'
```

说明：center 页面没有 `previewReport`，没有前端 Blob 拼接，已使用运行弹窗组件。

## 6. manage 页面改造证据

文件：`frontend/src/pages/report/manage/Index.vue`

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
                    v-if="props.row.status === 'published'"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="content_copy"
                    @click="copyReport(props.row)"
                  >
                    <q-tooltip>复制</q-tooltip>
                  </q-btn>
```

说明：`published` 报表运行/导出可用；`draft` / `disabled` 因 `status !== 'published'` 被禁用。设计、复制入口保留。

```vue
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
                  <q-btn
                    v-if="props.row.status === 'published'"
                    flat
                    size="sm"
                    round
                    color="warning"
                    icon="pause_circle"
                    @click="changeReportStatus(props.row, 'disabled')"
                  >
                    <q-tooltip>停用</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    size="sm"
                    round
                    color="negative"
                    icon="delete"
                    @click="deleteReport(props.row)"
                  >
                    <q-tooltip>删除</q-tooltip>
                  </q-btn>
```

说明：发布、版本、停用、删除入口仍保留。

```vue
    <report-runtime-dialog
      v-model="runtimeVisible"
      :report="runtimeReport"
      mode="manage"
      :allow-export="true"
    />

    <report-version-dialog
      v-model="versionDialogVisible"
      :report-id="versionDialogReport?.id"
      :current-version-id="versionDialogReport?.published_version_id"
      :current-version-no="versionDialogReport?.published_version_no"
    />
```

说明：manage 页面也复用 `ReportRuntimeDialog`，版本弹窗保留。

```ts
import ReportRuntimeDialog from '../components/ReportRuntimeDialog.vue'
import ReportVersionDialog from '../components/ReportVersionDialog.vue'
import { useReportExport } from '../composables/useReportExport'

const runtimeVisible = ref(false)
const runtimeReport = ref<Report | null>(null)
const versionDialogVisible = ref(false)
const versionDialogReport = ref<Report | null>(null)
const { exportingReportId, exportReportRow } = useReportExport()

function openRuntime(row: Report) {
  runtimeReport.value = row
  runtimeVisible.value = true
}
```

说明：行级导出复用 `useReportExport.exportReportRow`；`openRuntime(row)` 只打开统一运行弹窗。

搜索结果：

```bash
$ rg "previewReport" frontend/src/pages/report/manage/Index.vue
# 无输出

$ rg "new Blob|createObjectURL" frontend/src/pages/report/manage/Index.vue
# 无输出

$ rg "ReportRuntimeDialog|report-runtime-dialog" frontend/src/pages/report/manage/Index.vue
    <report-runtime-dialog
import ReportRuntimeDialog from '../components/ReportRuntimeDialog.vue'
```

说明：manage 页面没有 `previewReport`，没有前端 Blob 拼接，已使用运行弹窗组件。

## 7. 核心组件未修改证据

检查命令：

```bash
$ git status --short -- frontend/src/pages/report/components/ReportSheetPreview.vue \
  frontend/src/pages/report/design/Index.vue \
  frontend/src/pages/report/design/components/ReportSheetCanvas.vue \
  frontend/src/pages/report/design/components/ReportInspectorPanel.vue \
  frontend/src/pages/report/design/components/ReportResourcePanel.vue \
  frontend/src/modules/report/schema.ts \
  frontend/src/modules/report/sheet.ts \
  frontend/src/modules/report/options.ts
 M frontend/src/pages/report/design/Index.vue
```

当前结论：

- `frontend/src/pages/report/components/ReportSheetPreview.vue`：未修改
- `frontend/src/pages/report/design/components/ReportSheetCanvas.vue`：未修改
- `frontend/src/pages/report/design/components/ReportInspectorPanel.vue`：未修改
- `frontend/src/pages/report/design/components/ReportResourcePanel.vue`：未修改
- `frontend/src/modules/report/schema.ts`：未修改
- `frontend/src/modules/report/sheet.ts`：未修改
- `frontend/src/modules/report/options.ts`：未修改
- `frontend/src/pages/report/design/Index.vue`：当前工作区已有修改，但不属于 V1-E-1 本次改动范围；V1-E-1 未改设计器页面。

## 8. V1-D 行为是否保持

- 报表中心仍只查 `published`：是，`center/Index.vue` 的 `buildListFilters()` 固定 `status: 'published'`。
- 报表中心运行仍走 `/run`：是，运行逻辑进入 `ReportRuntimeDialog` -> `useReportRuntime` -> `reportApi.runReport`。
- 报表中心导出仍走 `/export`：是，导出逻辑进入 `ReportRuntimeDialog` -> `useReportExport` -> `reportApi.exportReport`。
- 管理页运行仍走 `/run`：是，运行逻辑复用同一弹窗和 `useReportRuntime`。
- 管理页导出仍走 `/export`：是，行级导出复用 `useReportExport.exportReportRow`。
- 管理页 `disabled` 报表不能运行/导出：是，按钮禁用条件为 `props.row.status !== 'published'`，导出函数也拒绝非 `published`。
- `ReportSheetPreview` 仍用于结果展示：是，`ReportRuntimeDialog` 内部继续使用 `ReportSheetPreview`。
- 旧 `previewReport` 没有重新用于中心/管理运行：是，center/manage 搜索均无 `previewReport`。

## 9. 构建验证

V1-E-1 实现完成后已执行：

```bash
$ cd frontend
$ source ~/.nvm/nvm.sh && nvm use 22.22.0 >/dev/null && node -v && yarn lint
v22.22.0
yarn run v1.22.21
$ eslint -c ./eslint.config.js "./src*/**/*.{ts,js,cjs,mjs,vue}"
Done in 12.58s.

$ source ~/.nvm/nvm.sh && nvm use 22.22.0 >/dev/null && node -v && yarn typecheck
v22.22.0
yarn run v1.22.21
$ vue-tsc --noEmit
Done in 5.49s.

$ source ~/.nvm/nvm.sh && nvm use 22.22.0 >/dev/null && node -v && yarn build
v22.22.0
yarn run v1.22.21
$ quasar build
Build succeeded
```

说明：

- Node 版本：`v22.22.0`
- warning：`ag-psd` 依赖中的 `util` browser externalized；部分 chunk 大于 `900 kB`
- warning 是否影响构建：不影响
- build 是否成功：成功

## 10. 已知风险

- 尚未真实后端联调，仍需用实际报表验证 `/run` 和 `/export` 请求参数。
- 复杂参数组合需要手工回归，尤其是 `date_range`、默认值、空值过滤。
- 数据权限效果需要真实账号和菜单权限验证。
- 管理页行级导出和弹窗内导出的参数存在差异：行级导出不带弹窗内当前 keyword / parameters，弹窗内导出会带当前运行参数。
- V1-E-2 建议再抽列表工作台能力，但不要在 V1-E-1 中做。
- V1-E-3 建议新增快速表格模式 UI，但需要保持现有 `query_config` / `layout_config` / sheet 结构兼容。
