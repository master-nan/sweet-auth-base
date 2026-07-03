<template>
  <base-content class="q-pa-sm report-manage-page">
    <div class="report-workspace">
      <section class="report-head">
        <div>
          <div class="report-title">报表管理</div>
          <div class="report-caption">
            管理报表草稿、发布、停用和设计入口，业务运行请到报表中心。
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
          <q-btn
            color="primary"
            icon="add"
            label="新建报表"
            @click="openDesigner()"
          />
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
          <div class="section-title">分类筛选</div>
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
              <div class="section-title">报表管理</div>
              <div class="report-caption">维护草稿、发布和停用状态，设计配置在设计器中处理。</div>
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
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_manage' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import { useRouter } from 'vue-router'
import type { Query } from 'src/types/global'
import {
  defaultReportSheet,
  useReportApi,
  type Report,
  type ReportStatus,
  type ReportKind,
} from 'src/api/services/report'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import ReportRuntimeDialog from '../components/ReportRuntimeDialog.vue'
import ReportVersionDialog from '../components/ReportVersionDialog.vue'
import { useReportExport } from '../composables/useReportExport'

const $q = useQuasar()
const router = useRouter()
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
const statusFilter = ref<'all' | ReportStatus>('all')
const dataSourceCount = ref(0)
const runtimeVisible = ref(false)
const runtimeReport = ref<Report | null>(null)
const versionDialogVisible = ref(false)
const versionDialogReport = ref<Report | null>(null)
const { exportingReportId, exportReportRow } = useReportExport()

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '已发布', value: 'published' },
  { label: '草稿', value: 'draft' },
  { label: '已停用', value: 'disabled' },
]

const columns = computed<QTableProps['columns']>(() => [
  { name: 'report_name', field: 'report_name', label: '报表名称', align: 'left' },
  { name: 'report_kind', field: 'report_kind', label: '展开方式', align: 'center' },
  { name: 'category', field: 'category', label: '分类', align: 'left' },
  { name: 'data_source_name', field: 'data_source_name', label: '数据集', align: 'left' },
  { name: 'permission', field: 'permission_table_code', label: '权限', align: 'center' },
  { name: 'status', field: 'status', label: '状态', align: 'center' },
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

const emptyText = computed(() => '暂无报表，点击右上角新建报表开始配置。')

const publishedCount = computed(
  () => rows.value.filter((item) => item.status === 'published').length,
)
const permissionCount = computed(
  () => rows.value.filter((item) => item.permission_table_code).length,
)

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
  const filters: Record<string, string> = {}
  if (statusFilter.value !== 'all') filters.status = statusFilter.value
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

function openDesigner(row?: Report) {
  void router.push({ name: 'report_design', query: row?.id ? { id: row.id } : {} })
}

function openRuntime(row: Report) {
  runtimeReport.value = row
  runtimeVisible.value = true
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
    const message = error instanceof Error && error.message ? error.message : '发布失败'
    $q.notify({ type: 'negative', message })
  }
}

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

function statusLabel(status: ReportStatus) {
  const map: Record<ReportStatus, string> = {
    draft: '草稿',
    published: '已发布',
    disabled: '已停用',
  }
  return map[status] || '草稿'
}

function statusColor(status: ReportStatus) {
  const map: Record<ReportStatus, string> = {
    draft: 'grey-7',
    published: 'positive',
    disabled: 'warning',
  }
  return map[status] || 'grey-7'
}

watch(
  () => [query.value.page, query.value.num] as const,
  ([page]) => {
    pagination.value.page = page
    void fetchData()
  },
)

watch(
  () => statusFilter.value,
  () => {
    activeCategory.value = ''
    resetToFirstPageOrFetch()
  },
)
</script>

<style scoped lang="scss">
.report-manage-page {
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
