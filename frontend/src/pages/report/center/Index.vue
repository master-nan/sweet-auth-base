<template>
  <base-content class="q-pa-sm report-center-page">
    <div class="report-workspace">
      <section class="report-main-grid">
        <aside class="category-panel">
          <div class="category-head">
            <q-icon name="folder_open" color="primary" />
            <div>
              <div class="section-title">{{ t('ui.reportCategory') }}</div>
              <span>{{ total }} {{ t('ui.okay') }}</span>
            </div>
          </div>
          <button
            class="category-item"
            :class="{ active: activeCategory === '' }"
            @click="selectCategory('')"
          >
            <span>{{ t('ui.allReports') }}</span>
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
        </aside>

        <section class="report-list-panel">
          <div class="list-head">
            <div class="section-title">{{ t('ui.runableReport') }}</div>
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
                    <q-tooltip>{{ t('ui.run') }}</q-tooltip>
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
      mode="center"
      :allow-export="true"
    />
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'report_center' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import type { Query } from 'src/types/global'
import { useReportApi, type Report, type ReportKind } from 'src/api/services/report'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import ReportRuntimeDialog from '../components/ReportRuntimeDialog.vue'

const { t } = useI18n({ useScope: 'global' })

const $q = useQuasar()
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
  rowsPerPage: 0,
  rowsNumber: 0,
  sortBy: 'gmt_modify',
  descending: true,
})

const rows = ref<Report[]>([])
const total = ref(0)
const activeCategory = ref('')
const runtimeVisible = ref(false)
const runtimeReport = ref<Report | null>(null)

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
  const map = new Map<string, number>()
  rows.value.forEach((item) => {
    const name = item.category || t('ui.uncategorized')
    map.set(name, (map.get(name) || 0) + 1)
  })
  return Array.from(map.entries()).map(([name, count]) => ({ name, count }))
})

const filteredRows = computed(() => rows.value)

const emptyText = computed(() => t('ui.forTheTimeBeingThereAreNoOperationalStatementsAnd'))

onMounted(() => {
  void fetchData()
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
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.failedToLoadReportsCheckTheBackendServiceOrApi')
      },
    })
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

async function openRuntime(row: Report) {
  try {
    const res = await reportApi.queryReportById(row.id, row.permission_menu_id || 0)
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
