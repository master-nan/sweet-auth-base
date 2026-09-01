<template>
  <base-content class="q-pa-sm report-manage-page">
    <div class="report-workspace">
      <section class="report-main-grid">
        <aside class="category-panel">
          <div class="category-head">
            <q-icon name="folder_open" color="primary" />
            <div>
              <div class="section-title">{{ t('ui.reportCategory') }}</div>
              <span>{{ categoryTotal }} {{ t('ui.okay') }}</span>
            </div>
          </div>
          <button
            class="category-item"
            :class="{ active: activeCategory === '' }"
            @click="selectCategory('')"
          >
            <span>{{ t('ui.allReports') }}</span>
            <q-badge color="primary" outline>{{ categoryTotal }}</q-badge>
          </button>
          <button
            v-for="category in categories"
            :key="category.key"
            class="category-item"
            :class="{ active: activeCategory === category.key }"
            @click="selectCategory(category.key)"
          >
            <span>{{ category.label }}</span>
            <q-badge color="primary" outline>{{ category.count }}</q-badge>
          </button>
        </aside>

        <section class="report-list-panel">
          <div class="list-head">
            <div>
              <div class="section-title">{{ t('ui.reportManagement') }}</div>
            </div>
            <div class="report-head-actions">
              <q-input
                v-model="query.quick_query!.keyword"
                dense
                outlined
                clearable
                debounce="300"
                class="report-search"
                :placeholder="t('ui.searchReportNameCodeCategory')"
                @keyup.enter="handleSearch"
              >
                <template #prepend>
                  <q-icon name="search" />
                </template>
              </q-input>
              <q-select
                v-model="statusFilter"
                dense
                outlined
                emit-value
                map-options
                class="status-filter"
                :options="statusOptions"
              />
              <q-btn
                v-if="canCreateReport"
                color="primary"
                icon="add"
                :label="t('ui.newReport')"
                @click="openDesigner()"
              />
              <q-btn flat round color="primary" icon="refresh" @click="fetchData">
                <q-tooltip>{{ t('ui.refresh') }}</q-tooltip>
              </q-btn>
            </div>
          </div>

          <q-table
            flat
            bordered
            separator="cell"
            row-key="id"
            class="report-table"
            dense
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
                  {{
                    props.row.permission_table_code
                      ? t('ui.inheritDataPermissions')
                      : t('ui.noPermissionTableBound')
                  }}
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
                    v-if="canRunReport"
                    :disable="props.row.status !== 'published'"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="play_arrow"
                    @click="openRuntime(props.row)"
                  >
                    <q-tooltip>{{ t('ui.run') }}</q-tooltip>
                  </q-btn>
                  <q-btn
                    v-if="canExportReport"
                    :disable="props.row.status !== 'published'"
                    :loading="exportingReportId === props.row.id"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="download"
                    @click="exportReportRow(props.row)"
                  >
                    <q-tooltip>{{ t('ui.export') }}</q-tooltip>
                  </q-btn>
                  <q-btn
                    v-if="canDesignReport"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="design_services"
                    @click="openDesigner(props.row)"
                  >
                    <q-tooltip>{{ t('ui.design') }}</q-tooltip>
                  </q-btn>
                  <q-btn
                    v-if="canCopyReport && props.row.status === 'published'"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="content_copy"
                    @click="copyReport(props.row)"
                  >
                    <q-tooltip>{{ t('ui.copy') }}</q-tooltip>
                  </q-btn>
                  <q-btn
                    v-if="canPublishReport"
                    :disable="props.row.status === 'disabled'"
                    flat
                    size="sm"
                    round
                    color="positive"
                    icon="publish"
                    @click="publishReport(props.row)"
                  >
                    <q-tooltip>{{ t('ui.publishAction') }}</q-tooltip>
                  </q-btn>
                  <q-btn
                    v-if="canViewReportVersions"
                    :disable="!canViewVersions(props.row)"
                    flat
                    size="sm"
                    round
                    color="primary"
                    icon="history"
                    @click="openVersionDialog(props.row)"
                  >
                    <q-tooltip>{{ t('ui.version') }}</q-tooltip>
                  </q-btn>
                  <q-btn
                    v-if="canChangeReportStatus && props.row.status === 'published'"
                    flat
                    size="sm"
                    round
                    color="warning"
                    icon="pause_circle"
                    @click="changeReportStatus(props.row, 'disabled')"
                  >
                    <q-tooltip>{{ t('ui.disabled') }}</q-tooltip>
                  </q-btn>
                  <q-btn
                    v-if="canDeleteReport"
                    flat
                    size="sm"
                    round
                    color="negative"
                    icon="delete"
                    @click="deleteReport(props.row)"
                  >
                    <q-tooltip>{{ t('ui.delete') }}</q-tooltip>
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
              <table-pagination
                v-model:page="query.page"
                v-model:pageSize="query.num"
                :total="total"
              />
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
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'report_manage' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import { useRouter } from 'vue-router'
import type { Query } from 'src/types/global'
import { ExpressionLogic, ExpressionType, SysTableFieldType } from 'src/types/enum'
import {
  defaultReportSheet,
  useReportApi,
  type Report,
  type ReportStatus,
  type ReportKind,
} from 'src/api/services/report'
import { useLoadingStore } from 'src/stores/loading'
import { usePageButtons } from 'src/composables/page-buttons'
import { storeToRefs } from 'pinia'
import ReportRuntimeDialog from '../components/ReportRuntimeDialog.vue'
import ReportVersionDialog from '../components/ReportVersionDialog.vue'
import { useReportExport } from '../composables/useReportExport'

const { t } = useI18n({ useScope: 'global' })

const $q = useQuasar()
const router = useRouter()
const reportApi = useReportApi()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const { hasGrantedCapability } = usePageButtons('report_manage')

const canCreateReport = computed(() => hasGrantedCapability('report_manage_create'))
const canDesignReport = computed(() => hasGrantedCapability('report_manage_design'))
const canCopyReport = computed(() => hasGrantedCapability('report_manage_copy'))
const canChangeReportStatus = computed(() => hasGrantedCapability('report_manage_status'))
const canDeleteReport = computed(() => hasGrantedCapability('report_manage_delete'))
const canPublishReport = computed(() => hasGrantedCapability('report_manage_publish'))
const canRunReport = computed(
  () => hasGrantedCapability('report_manage_run') || hasGrantedCapability('report_manage_preview'),
)
const canExportReport = computed(() => hasGrantedCapability('report_manage_export'))
const canViewReportVersions = computed(() => hasGrantedCapability('report_manage_versions'))

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
  rowsPerPage: 0,
  rowsNumber: 0,
  sortBy: 'gmt_modify',
  descending: true,
})

const rows = ref<Report[]>([])
const total = ref(0)
const categoryRows = ref<Report[]>([])
const categoryTotal = ref(0)
const activeCategory = ref('')
const statusFilter = ref<'all' | ReportStatus>('all')
const runtimeVisible = ref(false)
const runtimeReport = ref<Report | null>(null)
const versionDialogVisible = ref(false)
const versionDialogReport = ref<Report | null>(null)
const { exportingReportId, exportReportRow } = useReportExport()

const statusOptions = [
  {
    get label() {
      return t('ui.allStatus')
    },
    value: 'all',
  },
  {
    get label() {
      return t('ui.published')
    },
    value: 'published',
  },
  {
    get label() {
      return t('ui.draft')
    },
    value: 'draft',
  },
  {
    get label() {
      return t('ui.deactivatedStatus')
    },
    value: 'disabled',
  },
]

const columns = computed<QTableProps['columns']>(() => [
  {
    name: 'report_name',
    field: 'report_name',
    get label() {
      return t('ui.reportName')
    },
    align: 'left',
  },
  {
    name: 'report_kind',
    field: 'report_kind',
    get label() {
      return t('ui.expansionMode')
    },
    align: 'center',
  },
  {
    name: 'category',
    field: 'category',
    get label() {
      return t('ui.category')
    },
    align: 'left',
  },
  {
    name: 'data_source_name',
    field: 'data_source_name',
    get label() {
      return t('ui.dataset')
    },
    align: 'left',
  },
  {
    name: 'permission',
    field: 'permission_table_code',
    get label() {
      return t('ui.permissions')
    },
    align: 'center',
  },
  {
    name: 'status',
    field: 'status',
    get label() {
      return t('ui.status')
    },
    align: 'center',
  },
  {
    name: 'updated_at',
    field: (row) => row.updated_at || row.gmt_modify || '-',
    get label() {
      return t('ui.recentlyUpdated')
    },
    align: 'left',
  },
  {
    name: 'actions',
    field: 'actions',
    get label() {
      return t('ui.actions')
    },
    align: 'center',
  },
])

const categories = computed(() => {
  const map = new Map<string, { label: string; count: number }>()
  categoryRows.value.forEach((item) => {
    const category = item.category?.trim()
    const key = category || uncategorizedCategoryKey
    const current = map.get(key)
    map.set(key, {
      label: category || t('ui.uncategorized'),
      count: (current?.count || 0) + 1,
    })
  })
  return Array.from(map.entries()).map(([key, value]) => ({ key, ...value }))
})

const filteredRows = computed(() => rows.value)

const emptyText = computed(() =>
  canCreateReport.value
    ? t('ui.thereAreNoReportsClickOnTheTopRightCorner')
    : t('ui.noStatementsAreAvailable'),
)

onMounted(() => {
  void fetchData()
})

function handleSearch() {
  resetToFirstPageOrFetch()
}

async function fetchData() {
  try {
    const [res, categoryRes] = await Promise.all([
      reportApi.queryReports(buildListQuery()),
      reportApi.queryReports(buildCategoryQuery()),
    ])
    rows.value = res.data || []
    total.value = res.total ?? rows.value.length
    categoryRows.value = categoryRes.data || []
    categoryTotal.value = categoryRes.total ?? categoryRows.value.length
    pagination.value.page = query.value.page
    pagination.value.rowsNumber = total.value
  } catch {
    rows.value = []
    total.value = 0
    categoryRows.value = []
    categoryTotal.value = 0
    pagination.value.rowsNumber = 0
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.failedToLoadReportsCheckTheBackendServiceOrApi')
      },
    })
  }
}

const uncategorizedCategoryKey = '__report_uncategorized__'

function buildListQuery(): Query {
  return {
    ...query.value,
    expressions:
      activeCategory.value === uncategorizedCategoryKey
        ? [
            {
              logic: ExpressionLogic.OR,
              rules: [
                {
                  field: 'category',
                  expression_type: ExpressionType.EQ,
                  value: '',
                  type: SysTableFieldType.VARCHAR,
                },
                {
                  field: 'category',
                  expression_type: ExpressionType.IS_NULL,
                  value: null,
                  type: SysTableFieldType.VARCHAR,
                },
              ],
            },
          ]
        : [],
    filters: buildListFilters(),
  }
}

function buildCategoryQuery(): Query {
  return {
    ...query.value,
    page: 1,
    num: 5000,
    expressions: [],
    filters: buildCategoryFilters(),
  }
}

function buildListFilters() {
  const filters = buildCategoryFilters()
  if (activeCategory.value && activeCategory.value !== uncategorizedCategoryKey) {
    filters.category = activeCategory.value
  }
  return filters
}

function buildCategoryFilters() {
  const filters: Record<string, string> = {}
  if (statusFilter.value !== 'all') filters.status = statusFilter.value
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

function openDesigner(row?: Report) {
  void router.push({ name: 'report_design', query: row?.id ? { id: row.id } : {} })
}

async function openRuntime(row: Report) {
  try {
    const res = await reportApi.queryReportById(row.id)
    runtimeReport.value = res.data
    runtimeVisible.value = true
  } catch {
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.failedToLoadReportDetails')
      },
    })
  }
}

async function copyReport(row: Report) {
  try {
    const detail = await reportApi.queryReportById(row.id).then((res) => res.data)
    const fields = detail.query_config?.fields || []
    const layout = detail.layout_config
    const payload = {
      get report_name() {
        return t('ui.reportCopyName', { value1: detail.report_name })
      },
      report_code: `${detail.report_code}_copy_${Date.now().toString().slice(-4)}`,
      report_kind: detail.report_kind,
      fields,
      datasets: layout?.datasets || detail.query_config?.datasets || [],
      dataset_joins: layout?.dataset_joins || detail.query_config?.dataset_joins || [],
      parameters: layout?.parameters || detail.query_config?.parameters || [],
      sheet: layout?.sheet || defaultReportSheet(),
    }
    const res = await reportApi.createReport({
      ...payload,
      ...(detail.category ? { category: detail.category } : {}),
      ...(detail.description ? { description: detail.description } : {}),
      ...(detail.data_source_id ? { data_source_id: detail.data_source_id } : {}),
      ...(detail.permission_menu_id ? { permission_menu_id: detail.permission_menu_id } : {}),
      ...(detail.permission_table_code
        ? { permission_table_code: detail.permission_table_code }
        : {}),
    })
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.copyedReport', { value1: res.data })
      },
    })
    await fetchData()
  } catch {
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.copyFailed')
      },
    })
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
      get title() {
        return t('ui.publishReport')
      },
      get message() {
        return t('ui.confirmTheReleaseOfTheNewReleaseVersionWillBeRunBy', {
          value1: row.report_name,
        })
      },
      prompt: {
        model: '',
        type: 'textarea',
        get label() {
          return t('ui.releaseNotesOptional')
        },
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
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.theReportWasPublished')
      },
    })
    await fetchData()
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : t('ui.publishFailed')
    $q.notify({ type: 'negative', message })
  }
}

async function changeReportStatus(row: Report, status: ReportStatus) {
  if (status === 'published') {
    await publishReport(row)
    return
  }
  const actionText = status === 'draft' ? t('ui.forAsDraft') : t('ui.disabled')
  const confirmed = await new Promise<boolean>((resolve) => {
    $q.dialog({
      get title() {
        return t('ui.reportActionLabel', { actionText: actionText })
      },
      get message() {
        return t('ui.areYouSure', { actionText: actionText, value2: row.report_name })
      },
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
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.reportActionResult', { actionText: actionText })
      },
    })
    await fetchData()
  } catch {
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.namedActionFailed', { actionText: actionText })
      },
    })
  }
}

async function deleteReport(row: Report) {
  const confirmed = await new Promise<boolean>((resolve) => {
    $q.dialog({
      get title() {
        return t('ui.deleteReport')
      },
      get message() {
        return t('ui.areYouSureYouWantToDelete', { value1: row.report_name })
      },
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
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.reportDeleted')
      },
    })
    await fetchData()
  } catch {
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.failedToDelete')
      },
    })
  }
}

function kindLabel(kind: ReportKind) {
  const map: Record<ReportKind, string> = {
    get detail() {
      return t('ui.detailRow')
    },
    get summary() {
      return t('ui.summaryRow')
    },
  }
  return map[kind] || t('ui.detailRow')
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
    get draft() {
      return t('ui.draft')
    },
    get published() {
      return t('ui.published')
    },
    get disabled() {
      return t('ui.deactivatedStatus')
    },
  }
  return map[status] || t('ui.draft')
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
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fff;
  overflow: hidden;
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

.report-main-grid {
  min-height: calc(100vh - 176px);
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
}

.category-panel {
  padding: 14px 10px;
  border-right: 1px solid #dfe5f2;
  background: #fbfcff;
}

.category-head {
  min-height: 44px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 8px 10px;
}

.category-head .q-icon {
  font-size: 24px;
}

.category-head span {
  color: #71809a;
  font-size: 12px;
}

.section-title {
  font-size: 17px;
  font-weight: 800;
  color: #172033;
}

.category-item {
  width: 100%;
  min-height: 38px;
  margin-top: 4px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #172033;
  padding: 0 10px;
  cursor: pointer;
}

.category-item.active {
  background: rgba(115, 103, 240, 0.1);
  color: var(--q-primary);
  font-weight: 800;
}

.report-list-panel {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.list-head {
  min-height: 66px;
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

.report-table :deep(thead tr) {
  height: 42px;
}

.report-table :deep(tbody tr) {
  height: 48px;
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
  .report-main-grid {
    grid-template-columns: 1fr;
  }

  .category-panel {
    border-right: 0;
    border-bottom: 1px solid #dfe5f2;
  }

  .list-head {
    align-items: stretch;
    flex-direction: column;
  }

  .report-head-actions,
  .report-search {
    width: 100%;
  }

  .report-head-actions {
    justify-content: flex-start;
  }
}
</style>
