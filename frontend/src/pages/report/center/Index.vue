<template>
  <base-content class="q-pa-sm report-center-page">
    <div class="report-workspace">
      <section class="report-head">
        <div>
          <div class="report-title">报表中心</div>
          <div class="report-caption">按目录查找、运行、导出和维护报表，设计配置只在设计器中处理。</div>
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
          <q-btn color="primary" icon="add" label="新建报表" @click="openDesigner()" />
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
          <div class="section-title">目录</div>
          <button
            class="category-item"
            :class="{ active: activeCategory === '' }"
            @click="activeCategory = ''"
          >
            <span>全部报表</span>
            <q-badge color="primary" outline>{{ rows.length }}</q-badge>
          </button>
          <button
            v-for="category in categories"
            :key="category.name"
            class="category-item"
            :class="{ active: activeCategory === category.name }"
            @click="activeCategory = category.name"
          >
            <span>{{ category.name }}</span>
            <q-badge color="primary" outline>{{ category.count }}</q-badge>
          </button>

          <q-separator class="q-my-md" />

          <div class="section-title">快速流程</div>
          <div class="flow-list">
            <div class="flow-step"><b>1</b><span>新建报表并选择数据集</span></div>
            <div class="flow-step"><b>2</b><span>进入设计器配置参数、组件和字段</span></div>
            <div class="flow-step"><b>3</b><span>运行预览，继承菜单数据权限</span></div>
            <div class="flow-step"><b>4</b><span>发布给业务角色使用</span></div>
          </div>
        </aside>

        <section class="report-list-panel">
          <div class="list-head">
            <div>
              <div class="section-title">常用报表</div>
              <div class="report-caption">运行页只面向使用者，设计和发布由管理员进入设计器维护。</div>
            </div>
            <q-select
              v-model="statusFilter"
              dense
              outlined
              emit-value
              map-options
              class="status-filter"
              :options="statusOptions"
            />
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
            :rows-per-page-options="[10, 20, 50]"
            @request="onTableRequest"
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
            <template #body-cell-status="props">
              <q-td :props="props">
                <q-chip dense square :color="statusColor(props.row.status)" text-color="white">
                  {{ statusLabel(props.row.status) }}
                </q-chip>
              </q-td>
            </template>
            <template #body-cell-actions="props">
              <q-td :props="props">
                <div class="row no-wrap justify-center q-gutter-xs">
                  <q-btn
                    flat
                    dense
                    round
                    color="primary"
                    icon="play_arrow"
                    @click="openRuntime(props.row)"
                  >
                    <q-tooltip>运行</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    dense
                    round
                    color="primary"
                    icon="design_services"
                    @click="openDesigner(props.row)"
                  >
                    <q-tooltip>设计</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    dense
                    round
                    color="primary"
                    icon="content_copy"
                    @click="copyReport(props.row)"
                  >
                    <q-tooltip>复制</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    dense
                    round
                    color="negative"
                    icon="delete"
                    @click="deleteReport(props.row)"
                  >
                    <q-tooltip>删除</q-tooltip>
                  </q-btn>
                </div>
              </q-td>
            </template>
            <template #no-data>
              <div class="empty-state">
                <q-icon name="assignment_late" size="36px" />
                <span>暂无报表，点击右上角新建报表开始配置。</span>
              </div>
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
              {{ runtimeReport?.data_source_name || '-' }} · {{ runtimeReport?.description || '运行预览会应用后端数据权限' }}
            </div>
          </div>
          <q-space />
          <q-btn outline color="primary" icon="download" label="导出 CSV" :disable="!runtimeRows.length" @click="exportRuntimeCsv" />
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
          <q-btn outline color="primary" icon="restart_alt" label="重置" @click="resetRuntimeFilters" />
        </q-card-section>
        <q-card-section class="runtime-body">
          <q-table
            flat
            bordered
            separator="cell"
            row-key="id"
            :rows="runtimeRows"
            :columns="runtimeColumns"
            :loading="runtimeLoading"
            v-model:pagination="runtimePagination"
            :rows-per-page-options="[20, 50, 100]"
            @request="onRuntimeTableRequest"
          />
        </q-card-section>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_center' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import SweetDateTimePicker from 'components/DateTime/SweetDateTimePicker.vue'
import { computed, onMounted, ref } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import { useRouter } from 'vue-router'
import type { Query } from 'src/types/global'
import {
  defaultReportSheet,
  useReportApi,
  type Report,
  type ReportParameter,
  type ReportPreviewRes,
  type ReportStatus,
  type ReportKind,
} from 'src/api/services/report'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'

const $q = useQuasar()
const router = useRouter()
const reportApi = useReportApi()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const query = ref<Query>({
  page: 1,
  num: 20,
  order: { field: 'gmt_modify', is_asc: false },
  expressions: [],
  quick_query: { keyword: '' },
  include_deleted: false,
})

const pagination = ref({
  page: 1,
  rowsPerPage: 20,
  rowsNumber: 0,
  sortBy: 'gmt_modify',
  descending: true,
})

const rows = ref<Report[]>([])
const activeCategory = ref('')
const statusFilter = ref<'all' | ReportStatus>('all')
const dataSourceCount = ref(0)
const runtimeVisible = ref(false)
const runtimeReport = ref<Report | null>(null)
const runtimeData = ref<ReportPreviewRes>({ columns: [], rows: [] })
const runtimeLoading = ref(false)
const runtimeKeyword = ref('')
const runtimeFilterValues = ref<Record<string, string | number | Array<string | number> | null | undefined>>({})
const runtimePagination = ref({
  page: 1,
  rowsPerPage: 20,
  rowsNumber: 0,
})

type TableRequest = {
  pagination: {
    page: number
    rowsPerPage: number
    rowsNumber?: number
    sortBy?: string
    descending?: boolean
  }
}

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '已发布', value: 'published' },
  { label: '草稿', value: 'draft' },
  { label: '已归档', value: 'archived' },
]

const columns = computed<QTableProps['columns']>(() => [
  { name: 'report_name', field: 'report_name', label: '报表名称', align: 'left' },
  { name: 'report_kind', field: 'report_kind', label: '类型', align: 'center' },
  { name: 'category', field: 'category', label: '分类', align: 'left' },
  { name: 'data_source_name', field: 'data_source_name', label: '数据集', align: 'left' },
  { name: 'permission', field: 'permission_table_code', label: '权限', align: 'center' },
  { name: 'status', field: 'status', label: '状态', align: 'center' },
  { name: 'updated_at', field: (row) => row.updated_at || row.gmt_modify || '-', label: '最近更新', align: 'left' },
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

const filteredRows = computed(() => {
  const keyword = query.value.quick_query?.keyword?.trim().toLowerCase() || ''
  return rows.value.filter((item) => {
    const matchCategory = !activeCategory.value || (item.category || '未分类') === activeCategory.value
    const matchStatus = statusFilter.value === 'all' || item.status === statusFilter.value
    const matchKeyword =
      !keyword ||
      [item.report_name, item.report_code, item.category, item.data_source_name]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(keyword))
    return matchCategory && matchStatus && matchKeyword
  })
})

const publishedCount = computed(() => rows.value.filter((item) => item.status === 'published').length)
const permissionCount = computed(() => rows.value.filter((item) => item.permission_table_code).length)
const runtimeColumns = computed<QTableProps['columns']>(() =>
  runtimeData.value.columns.map((field) => ({
    name: field.code,
    field: field.code,
    label: field.name,
    align: 'left',
  })),
)
const runtimeRows = computed(() => runtimeData.value.rows)
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
  pagination.value.page = 1
  void fetchData()
}

async function fetchData() {
  try {
    query.value.page = pagination.value.page
    query.value.num = pagination.value.rowsPerPage
    const res = await reportApi.queryReports(query.value)
    rows.value = res.data || []
    pagination.value.rowsNumber = res.total ?? rows.value.length
  } catch {
    rows.value = []
    pagination.value.rowsNumber = 0
    $q.notify({ type: 'negative', message: '报表列表加载失败，请检查后端服务或接口权限' })
  }
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

function onTableRequest(props: TableRequest) {
  pagination.value = { ...pagination.value, ...props.pagination }
  void fetchData()
}

function openDesigner(row?: Report) {
  void router.push({ name: 'report_design', query: row?.id ? { id: row.id } : {} })
}

async function openRuntime(row: Report) {
  runtimeReport.value = row
  runtimeKeyword.value = ''
  runtimeFilterValues.value = {}
  runtimePagination.value.page = 1
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
      page: runtimePagination.value.page,
      num: runtimePagination.value.rowsPerPage,
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

function onRuntimeTableRequest(props: TableRequest) {
  runtimePagination.value = { ...runtimePagination.value, ...props.pagination }
  void loadRuntimePreview()
}

function resetRuntimeFilters() {
  runtimeKeyword.value = ''
  runtimeFilterValues.value = {}
  runtimePagination.value.page = 1
  void loadRuntimePreview()
}

function buildRuntimeParameterValues() {
  const values: Record<string, unknown> = {}
  runtimeParameters.value.forEach((param) => {
    const value = runtimeFilterValues.value[param.id]
    if (value === '' || value === null || value === undefined) return
    if (Array.isArray(value) && !value.some((item) => item !== '' && item !== null && item !== undefined)) return
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
  const current = Array.isArray(runtimeFilterValues.value[id]) ? [...(runtimeFilterValues.value[id] as Array<string | number>)] : ['', '']
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

async function copyReport(row: Report) {
  try {
    const fields = row.query_config?.fields || []
    const layout = row.layout_config
    const payload = {
      report_name: `${row.report_name} 副本`,
      report_code: `${row.report_code}_copy_${Date.now().toString().slice(-4)}`,
      report_kind: row.report_kind,
      fields,
      datasets: layout?.datasets || row.query_config?.datasets || [],
      dataset_joins: layout?.dataset_joins || row.query_config?.dataset_joins || [],
      parameters: layout?.parameters || row.query_config?.parameters || [],
      sheet: layout?.sheet || defaultReportSheet(),
    }
    const res = await reportApi.createReport({
      ...payload,
      ...(row.category ? { category: row.category } : {}),
      ...(row.description ? { description: row.description } : {}),
      ...(row.data_source_id ? { data_source_id: row.data_source_id } : {}),
      ...(row.permission_menu_id ? { permission_menu_id: row.permission_menu_id } : {}),
      ...(row.permission_table_code ? { permission_table_code: row.permission_table_code } : {}),
    })
    $q.notify({ type: 'positive', message: `已复制报表 #${res.data}` })
    await fetchData()
  } catch {
    $q.notify({ type: 'negative', message: '复制失败' })
  }
}

async function deleteReport(row: Report) {
  const confirmed = await new Promise<boolean>((resolve) => {
    $q.dialog({
      title: '删除报表',
      message: `确认删除「${row.report_name}」吗？删除后不可恢复。`,
      cancel: true,
      persistent: true,
    })
      .onOk(() => resolve(true))
      .onCancel(() => resolve(false))
      .onDismiss(() => resolve(false))
  })
  if (!confirmed) return
  try {
    await reportApi.deleteReport(row.id)
    $q.notify({ type: 'positive', message: '报表已删除' })
    await fetchData()
  } catch {
    $q.notify({ type: 'negative', message: '删除失败' })
  }
}

function kindLabel(kind: ReportKind) {
  const map: Record<ReportKind, string> = {
    detail: '明细表',
    summary: '汇总表',
    chart: '图表',
    pivot: '交叉表',
  }
  return map[kind] || '明细表'
}

function kindIcon(kind: ReportKind) {
  const map: Record<ReportKind, string> = {
    detail: 'table_rows',
    summary: 'summarize',
    chart: 'bar_chart',
    pivot: 'pivot_table_chart',
  }
  return map[kind] || 'table_rows'
}

function statusLabel(status: ReportStatus) {
  const map: Record<ReportStatus, string> = {
    draft: '草稿',
    published: '已发布',
    archived: '已归档',
  }
  return map[status] || '草稿'
}

function statusColor(status: ReportStatus) {
  const map: Record<ReportStatus, string> = {
    draft: 'grey-7',
    published: 'positive',
    archived: 'warning',
  }
  return map[status] || 'grey-7'
}

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

.status-filter {
  width: 150px;
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
