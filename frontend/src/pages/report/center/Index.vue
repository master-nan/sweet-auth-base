<template>
  <base-content class="q-pa-sm report-center-page">
    <div class="report-workspace">
      <section class="report-head">
        <div>
          <div class="report-title">报表中心</div>
          <div class="report-caption">
            面向业务用户查看和运行已发布报表，设计、发布和停用请到报表管理处理。
          </div>
        </div>
        <div class="report-head-actions">
          <q-input
            v-model="query.quick_query!.keyword"
            dense
            outlined
            clearable
            debounce="300"
            class="report-search"
            placeholder="搜索报表名称 / 编码 / 分类"
            @keyup.enter="handleSearch"
          >
            <template #prepend>
              <q-icon name="search" />
            </template>
          </q-input>
          <q-btn outline color="primary" icon="refresh" label="刷新" @click="fetchData" />
        </div>
      </section>

      <section class="report-metrics">
        <div class="metric-item">
          <q-icon name="assessment" />
          <div>
            <strong>{{ publishedCount }}</strong>
            <span>已发布报表</span>
          </div>
        </div>
        <div class="metric-item">
          <q-icon name="folder_special" />
          <div>
            <strong>{{ categories.length }}</strong>
            <span>报表分类</span>
          </div>
        </div>
        <div class="metric-item">
          <q-icon name="dataset" />
          <div>
            <strong>{{ dataSourceCount }}</strong>
            <span>可用数据集</span>
          </div>
        </div>
        <div class="metric-item">
          <q-icon name="security" />
          <div>
            <strong>{{ permissionCount }}</strong>
            <span>继承数据权限</span>
          </div>
        </div>
      </section>

      <section class="report-main-grid">
        <aside class="category-panel">
          <div class="section-title">报表分类</div>
          <button
            class="category-item"
            :class="{ active: activeCategory === '' }"
            @click="selectCategory('')"
          >
            <span>全部报表</span>
            <q-badge color="primary" outline>{{ rows.length }}</q-badge>
          </button>
          <button
            v-for="category in categories"
            :key="category.name"
            class="category-item"
            :class="{ active: activeCategory === category.name }"
            @click="selectCategory(category.name)"
          >
            <span>{{ category.name }}</span>
            <q-badge color="primary" outline>{{ category.count }}</q-badge>
          </button>

          <q-separator class="q-my-md" />

          <div class="section-title">使用说明</div>
          <div class="flow-list">
            <div class="flow-step"><b>1</b><span>选择分类或搜索报表</span></div>
            <div class="flow-step"><b>2</b><span>运行报表并输入查询参数</span></div>
            <div class="flow-step"><b>3</b><span>导出当前结果为 CSV</span></div>
          </div>
        </aside>

        <section class="report-list-panel">
          <div class="list-head">
            <div>
              <div class="section-title">可运行报表</div>
              <div class="report-caption">只展示已发布并可运行的报表。</div>
            </div>
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
            <template #body-cell-report_kind="props">
              <q-td :props="props">
                <q-chip dense square color="primary" outline>
                  {{ kindLabel(props.row.report_kind) }}
                </q-chip>
              </q-td>
            </template>
            <template #body-cell-permission="props">
              <q-td :props="props">
                <q-chip
                  dense
                  square
                  :color="props.row.permission_table_code ? 'positive' : 'warning'"
                  outline
                >
                  {{ props.row.permission_table_code ? '继承数据权限' : '未绑定权限表' }}
                </q-chip>
              </q-td>
            </template>
            <template #body-cell-actions="props">
              <q-td :props="props">
                <div class="row no-wrap justify-center q-gutter-xs">
                  <q-btn
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="play_arrow"
                    @click="openRuntime(props.row)"
                  >
                    <q-tooltip>运行</q-tooltip>
                  </q-btn>
                </div>
              </q-td>
            </template>
            <template #no-data>
              <div class="empty-state">
                <q-icon name="assignment_late" size="36px" />
                <span>{{ emptyText }}</span>
              </div>
            </template>
            <template #bottom>
              <q-space />
              <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
            </template>
          </q-table>
        </section>
      </section>
    </div>

    <q-dialog v-model="runtimeVisible" maximized>
      <q-card class="runtime-dialog">
        <q-card-section class="runtime-head">
          <div>
            <div class="report-title">{{ runtimeReport?.report_name || '报表运行' }}</div>
            <div class="report-caption">
              {{ runtimeReport?.data_source_name || '-' }} ·
              {{ runtimeReport?.description || '运行预览会应用后端数据权限' }}
            </div>
          </div>
          <q-space />
          <q-btn
            outline
            color="primary"
            icon="download"
            label="导出 CSV"
            :disable="!runtimeRows.length"
            @click="exportRuntimeCsv"
          />
          <q-btn flat round icon="close" v-close-popup />
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
        </q-card-section>
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
              v-model:page-size="runtimePagination.rowsPerPage"
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
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_center' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import SweetDateTimePicker from 'components/DateTime/SweetDateTimePicker.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import type { Query } from 'src/types/global'
import {
  defaultReportSheet,
  useReportApi,
  type Report,
  type ReportDataset,
  type ReportParameter,
  type ReportPreviewRes,
  type ReportKind,
  type ReportSheetConfig,
} from 'src/api/services/report'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import ReportSheetPreview from '../components/ReportSheetPreview.vue'

const $q = useQuasar()
const reportApi = useReportApi()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const query = ref<Query>({
  page: 1,
  num: 15,
  order: { field: 'gmt_modify', is_asc: false },
  expressions: [],
  quick_query: { keyword: '' },
  include_deleted: false,
})

const pagination = ref({
  page: 1,
  rowsPerPage: 0,
  rowsNumber: 0,
  sortBy: 'gmt_modify',
  descending: true,
})

const rows = ref<Report[]>([])
const total = ref(0)
const activeCategory = ref('')
const dataSourceCount = ref(0)
const runtimeVisible = ref(false)
const runtimeReport = ref<Report | null>(null)
const runtimeData = ref<ReportPreviewRes>({ columns: [], rows: [] })
const runtimeLoading = ref(false)
const runtimeKeyword = ref('')
const runtimeFilterValues = ref<
  Record<string, string | number | Array<string | number> | null | undefined>
>({})
const runtimePagination = ref({
  page: 1,
  rowsPerPage: 20,
  rowsNumber: 0,
})

const columns = computed<QTableProps['columns']>(() => [
  { name: 'report_name', field: 'report_name', label: '报表名称', align: 'left' },
  { name: 'report_kind', field: 'report_kind', label: '展开方式', align: 'center' },
  { name: 'category', field: 'category', label: '分类', align: 'left' },
  { name: 'data_source_name', field: 'data_source_name', label: '数据集', align: 'left' },
  { name: 'permission', field: 'permission_table_code', label: '权限', align: 'center' },
  {
    name: 'updated_at',
    field: (row) => row.updated_at || row.gmt_modify || '-',
    label: '最近更新',
    align: 'left',
  },
  { name: 'actions', field: 'actions', label: '操作', align: 'center' },
])

const categories = computed(() => {
  const map = new Map<string, number>()
  rows.value.forEach((item) => {
    const name = item.category || '未分类'
    map.set(name, (map.get(name) || 0) + 1)
  })
  return Array.from(map.entries()).map(([name, count]) => ({ name, count }))
})

const filteredRows = computed(() => rows.value)

const emptyText = computed(() => '暂无可运行报表，请先在报表管理中发布。')

const publishedCount = computed(
  () => rows.value.filter((item) => item.status === 'published').length,
)
const permissionCount = computed(
  () => rows.value.filter((item) => item.permission_table_code).length,
)
const runtimeColumns = computed<QTableProps['columns']>(() =>
  runtimeData.value.columns.map((field) => ({
    name: field.code,
    field: field.code,
    label: field.name,
    align: 'left',
  })),
)
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
const runtimeParameters = computed<ReportParameter[]>(() => {
  const report = runtimeReport.value
  return report?.layout_config?.parameters?.length
    ? report.layout_config.parameters
    : report?.query_config?.parameters || []
})

onMounted(() => {
  void Promise.all([fetchData(), loadDataSources()])
})

function handleSearch() {
  resetToFirstPageOrFetch()
}

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

function selectCategory(category: string) {
  if (activeCategory.value === category) return
  activeCategory.value = category
  resetToFirstPageOrFetch()
}

function resetToFirstPageOrFetch() {
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  void fetchData()
}

async function loadDataSources() {
  try {
    const res = await reportApi.queryDataSources()
    dataSourceCount.value = res.total ?? res.data.length
  } catch {
    dataSourceCount.value = 0
    $q.notify({ type: 'warning', message: '数据集列表加载失败，设计器可能无法选择数据源' })
  }
}

async function openRuntime(row: Report) {
  runtimeReport.value = row
  runtimeKeyword.value = ''
  runtimeFilterValues.value = buildRuntimeDefaultFilters()
  runtimePagination.value.page = 1
  runtimePagination.value.rowsPerPage = runtimeConfiguredPageSize.value
  runtimeVisible.value = true
  await loadRuntimePreview()
}

async function loadRuntimePreview() {
  if (!runtimeReport.value?.id) return
  runtimeLoading.value = true
  try {
    const res = await reportApi.previewReport({
      report_id: runtimeReport.value.id,
      data_source_id: runtimeReport.value.data_source_id,
      page: runtimeDisplayMode.value === 'all' ? 1 : runtimePagination.value.page,
      num: runtimeDisplayMode.value === 'all' ? 10000 : runtimePagination.value.rowsPerPage,
      keyword: runtimeKeyword.value,
      parameters: buildRuntimeParameterValues(),
    })
    runtimeData.value = res.data
    runtimePagination.value.rowsNumber = res.data.total ?? res.data.rows.length
  } catch {
    runtimeData.value = { columns: [], rows: [], total: 0 }
    runtimePagination.value.rowsNumber = 0
    $q.notify({ type: 'negative', message: '报表运行失败，请检查报表配置、数据权限或后端接口' })
  } finally {
    runtimeLoading.value = false
  }
}

function resetRuntimeFilters() {
  runtimeKeyword.value = ''
  runtimeFilterValues.value = buildRuntimeDefaultFilters()
  runtimePagination.value.page = 1
  void loadRuntimePreview()
}

function buildRuntimeDefaultFilters() {
  const values: Record<string, string | number | Array<string | number> | null | undefined> = {}
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

function exportRuntimeCsv() {
  if (!runtimeRows.value.length) return
  const columns = runtimeColumns.value || []
  const headers = columns.map((column) => String(column.label || column.name))
  const fields = columns.map((column) => String(column.field || column.name))
  const lines = [
    headers,
    ...runtimeRows.value.map((row) => fields.map((field) => csvCell(row[field]))),
  ]
  const csv = lines.map((line) => line.join(',')).join('\n')
  const blob = new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${runtimeReport.value?.report_code || 'report'}_${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

function csvCell(value: unknown) {
  const text = value === null || value === undefined ? '' : String(value)
  return `"${text.replaceAll('"', '""')}"`
}

function kindLabel(kind: ReportKind) {
  const map: Record<ReportKind, string> = {
    detail: '明细行',
    summary: '汇总行',
  }
  return map[kind] || '明细行'
}

function kindIcon(kind: ReportKind) {
  const map: Record<ReportKind, string> = {
    detail: 'table_rows',
    summary: 'summarize',
  }
  return map[kind] || 'table_rows'
}

watch(
  () => [query.value.page, query.value.num] as const,
  ([page]) => {
    pagination.value.page = page
    void fetchData()
  },
)

</script>

<style scoped lang="scss">
.report-center-page {
  min-height: 0;
}

.report-workspace {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.report-head,
.report-list-panel,
.category-panel {
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fff;
}

.report-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 16px;
}

.report-title {
  font-size: 22px;
  font-weight: 800;
  color: #172033;
}

.report-caption {
  margin-top: 4px;
  color: #71809a;
  line-height: 1.5;
}

.report-head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.report-search {
  width: 280px;
}

.report-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.metric-item {
  min-height: 88px;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fff;
}

.metric-item .q-icon {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  color: var(--q-primary);
  background: #f0eeff;
}

.metric-item strong {
  display: block;
  font-size: 26px;
  line-height: 1;
}

.metric-item span {
  display: block;
  margin-top: 6px;
  color: #71809a;
}

.report-main-grid {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 12px;
}

.category-panel {
  padding: 14px;
}

.section-title {
  font-size: 17px;
  font-weight: 800;
  color: #172033;
}

.category-item {
  width: 100%;
  min-height: 44px;
  margin-top: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fbfcff;
  color: #172033;
  padding: 0 12px;
  cursor: pointer;
}

.category-item.active {
  border-color: var(--q-primary);
  background: #f7f5ff;
  color: var(--q-primary);
  font-weight: 800;
}

.flow-list {
  display: grid;
  gap: 8px;
  margin-top: 10px;
}

.flow-step {
  display: grid;
  grid-template-columns: 28px 1fr;
  gap: 8px;
  align-items: start;
  color: #5f6f88;
}

.flow-step b {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--q-primary);
  color: #fff;
}

.report-list-panel {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.list-head {
  min-height: 70px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #dfe5f2;
}

.report-table {
  flex: 1;
}

.report-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.report-name-cell strong,
.report-name-cell span {
  display: block;
}

.report-name-cell span {
  color: #71809a;
  margin-top: 2px;
}

.empty-state {
  width: 100%;
  min-height: 240px;
  display: grid;
  place-items: center;
  gap: 8px;
  color: #71809a;
}

.runtime-dialog {
  display: flex;
  flex-direction: column;
}

.runtime-head,
.runtime-filters {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.runtime-filter {
  width: 220px;
}

.runtime-range-filter {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.runtime-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.runtime-pagination {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  color: #71809a;
}

@media (max-width: 1200px) {
  .report-metrics,
  .report-main-grid {
    grid-template-columns: 1fr;
  }

  .report-head {
    flex-direction: column;
  }

  .report-head-actions,
  .report-search {
    width: 100%;
  }
}
</style>
