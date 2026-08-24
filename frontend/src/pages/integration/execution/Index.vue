<template>
  <base-content class="q-pa-sm column no-wrap execution-page">
    <div class="runtime-status-strip row items-center no-wrap q-gutter-sm q-px-sm q-py-xs">
      <q-icon name="monitor_heart" size="22px" color="primary" />
      <div>
        <div class="text-weight-medium">Worker运行状态</div>
        <div class="text-caption text-grey-7">
          {{ workerStatus.worker_id || '未启用 Worker' }}
        </div>
      </div>
      <q-separator vertical inset />
      <status-chip
        :color="workerStatus.running ? 'positive' : 'grey-6'"
        :label="workerStatus.running ? '运行中' : workerStatus.enabled ? '已停止' : '未启用'"
      />
      <div class="text-caption">活动 {{ workerStatus.active_execution_count }}</div>
      <div class="text-caption">已完成 {{ workerStatus.completed_total }}</div>
      <div class="text-caption text-grey-7">
        最近轮询 {{ formatDate(workerStatus.last_poll_at) }}
      </div>
      <q-space />
      <q-btn
        flat
        round
        dense
        icon="refresh"
        color="primary"
        :loading="loading"
        @click="refresh"
        aria-label="刷新执行状态"
      >
        <q-tooltip>刷新执行状态</q-tooltip>
      </q-btn>
    </div>

    <q-table
      v-model:pagination="pagination"
      class="col sticky-header-table execution-table"
      color="primary"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :loading="loading"
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="refresh">
          <template #query-controls>
            <query-scheme-controls
              :controller="schemePage"
              :query-state="queryState"
              :fields="advancedFields"
              :advanced-enabled="advancedFields.length > 0"
            >
              <template #quick-search>
                <q-input
                  v-model="keyword"
                  dense
                  outlined
                  debounce="300"
                  :placeholder="quickSearchPlaceholder"
                  @keyup.enter="search"
                >
                  <template #append><q-icon name="search" /></template>
                </q-input>
                <q-btn
                  color="primary"
                  icon="search"
                  label="查询"
                  :disable="loading"
                  @click="search"
                />
              </template>
            </query-scheme-controls>
          </template>

          <template #right-actions>
            <q-btn
              v-for="button in top_buttons"
              :key="button.id"
              v-bind="menuButtonDisplayProps(button)"
              :color="button.color || 'primary'"
              :disable="loading"
              @click="handleButton(button)"
            />
          </template>
        </standard-table-toolbar>
      </template>

      <template #body-cell-execution_no="props">
        <q-td :props="props">
          <q-btn
            v-if="canViewDetail"
            flat
            dense
            color="primary"
            :label="props.row.execution_no"
            @click="openDetail(props.row)"
          />
          <span v-else>{{ props.row.execution_no }}</span>
        </q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props"
          ><status-chip
            :color="statusMeta[props.row.status]?.color || 'grey'"
            :label="statusMeta[props.row.status]?.label || props.row.status"
        /></q-td>
      </template>
      <template #body-cell-external_system="props">
        <q-td :props="props"
          ><div class="text-weight-medium">{{ props.row.external_system.name }}</div>
          <div class="text-caption text-grey-7">
            {{ props.row.external_system.system_code }}
          </div></q-td
        >
      </template>
      <template #body-cell-interface="props">
        <q-td :props="props"
          ><div>{{ props.row.interface.interface_code }}</div>
          <div class="text-caption text-grey-7">
            v{{ props.row.interface.version }} · {{ props.row.interface.name }}
          </div></q-td
        >
      </template>
      <template #body-cell-current_attempt="props">
        <q-td :props="props">{{ props.row.current_attempt }} / {{ props.row.max_attempts }}</q-td>
      </template>
      <template #body-cell-next_run_at="props">
        <q-td :props="props">{{ formatDate(props.row.next_run_at) }}</q-td>
      </template>
      <template #body-cell-started_at="props">
        <q-td :props="props">{{ formatDate(props.row.started_at) }}</q-td>
      </template>
      <template #body-cell-completed_at="props">
        <q-td :props="props">{{ formatDate(props.row.completed_at) }}</q-td>
      </template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs no-wrap">
          <q-btn
            v-for="button in availableButtons(props.row)"
            :key="button.id"
            flat
            dense
            size="sm"
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            @click="handleButton(button, props.row)"
            ><q-tooltip>{{ button.name }}</q-tooltip></q-btn
          >
        </q-td>
      </template>
      <template #bottom
        ><q-space /><table-pagination
          v-model:page="query.page"
          v-model:page-size="query.num"
          :total="total"
      /></template>
      <template #no-data
        ><div class="full-width row flex-center q-pa-xl text-grey-7">
          {{ emptyMessage }}
        </div></template
      >
    </q-table>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_execution' })

import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import {
  useIntegrationApi,
  type IntegrationExecutionListItem,
  type IntegrationExecutionQuery,
  type IntegrationWorkerStatus,
} from 'src/api/services/integration'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useTableQueryState } from 'src/composables/table-query-state'
import { useRuntimeTableMetadata } from 'src/composables/runtime-table-metadata'
import { useQuerySchemePage } from 'src/composables/query-scheme-page'
import type { MenuButton } from 'src/api/services/sys-menu'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { formatRetryReason, formatRuntimeDateTime } from 'src/pages/integration/runtime-display'
import { dispatchPageAction, type PageActionHandlers } from 'src/utils/button-handlers'
import { resolveTableEmptyMessage } from 'src/utils/table-state'
import { countEffectiveQueryRules } from 'src/utils/query-state'

const $q = useQuasar()
const router = useRouter()
const api = useIntegrationApi()
const loading = ref(false)
const loadError = ref('')
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, hasGrantedCapability } = usePageButtons('integration_execution')
const canQueryExecutions = computed(() => hasGrantedCapability('integration_execution_query'))
const canViewWorkerStatus = computed(() => hasGrantedCapability('integration_worker_status'))
const canViewDetail = computed(() => hasGrantedCapability('integration_execution_detail'))
const rows = ref<IntegrationExecutionListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const queryState = useTableQueryState<IntegrationExecutionQuery>({
  createInitialQuery: () => ({
    page: 1,
    num: 15,
    order: { field: 'gmt_create', is_asc: false },
    quick_query: { keyword: '' },
    expressions: emptyExpressions(),
  }),
})
const { query, keyword, appliedAdvanced: appliedAdvancedQuery } = queryState
const { quickSearchPlaceholder, advancedSearchFields: advancedFields, loadMetadata } =
  useRuntimeTableMetadata('integration_execution')
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: canQueryExecutions.value,
    error: loadError.value,
    hasQuery: !!keyword.value || activeFilterCount.value > 0,
  }),
)
const workerStatus = ref<IntegrationWorkerStatus>({
  enabled: false,
  running: false,
  worker_id: '',
  started_at: '',
  last_poll_at: '',
  last_success_at: '',
  last_error_category: '',
  active_execution_count: 0,
  claimed_total: 0,
  completed_total: 0,
  failed_total: 0,
  recovered_total: 0,
})
const statusMeta: Record<string, { label: string; color: string }> = {
  created: { label: '待执行', color: 'grey-7' },
  running: { label: '执行中', color: 'primary' },
  retry_waiting: { label: '等待重试', color: 'warning' },
  succeeded: { label: '成功', color: 'positive' },
  failed: { label: '失败', color: 'negative' },
  cancelled: { label: '已取消', color: 'grey-6' },
}
const columns: QTableProps['columns'] = [
  { name: 'execution_no', label: '执行编号', field: 'execution_no', align: 'left', sortable: true },
  { name: 'external_system', label: '外部系统', field: 'external_system', align: 'left' },
  { name: 'interface', label: '接口', field: 'interface', align: 'left' },
  { name: 'trigger_source', label: '触发来源', field: 'trigger_source', align: 'center' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
  { name: 'current_attempt', label: 'Attempt', field: 'current_attempt', align: 'center' },
  { name: 'next_run_at', label: '下次重试', field: 'next_run_at', align: 'left' },
  {
    name: 'retry_reason_code',
    label: '重试原因',
    field: (row) => formatRetryReason(row.retry_reason_code),
    align: 'left',
  },
  { name: 'started_at', label: '开始时间', field: 'started_at', align: 'left' },
  { name: 'completed_at', label: '结束时间', field: 'completed_at', align: 'left' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]

const formatDate = formatRuntimeDateTime
const fetchData = async () => {
  if (!canQueryExecutions.value) return
  loading.value = true
  loadError.value = ''
  try {
    const response = await api.queryExecutions(query.value)
    rows.value = response.data || []
    total.value = response.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = 'Execution 加载失败'
  } finally {
    loading.value = false
  }
}
const fetchWorker = async () => {
  if (!canViewWorkerStatus.value) return
  const response = await api.getWorkerStatus()
  if (response.data) workerStatus.value = response.data
}
const refresh = async () => {
  await fetchData()
  await fetchWorker().catch(() => undefined)
}
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('integration_execution', queryState, resetAndFetch)
const search = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
}
const openDetail = (row: IntegrationExecutionListItem) => {
  void router.push({ name: 'integration_execution_detail_page', params: { id: row.id } })
}
const availableButtons = (row: IntegrationExecutionListItem) =>
  line_buttons.value.filter((button) =>
    button.event_action === 'cancel'
      ? row.status === 'created' || row.status === 'retry_waiting'
      : true,
  )
const cancel = (row: IntegrationExecutionListItem) => {
  confirmAction({
    title: '取消执行',
    message: `确认取消执行“${row.execution_no}”？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await api.cancelExecution(row.id, row.revision)
      if (result.success) await fetchData()
    })()
  })
}
const handlers: PageActionHandlers<IntegrationExecutionListItem> = {
  detail: (row) => row && openDetail(row),
  cancel: (row) => row && cancel(row),
}
const handleButton = (button: MenuButton, row?: IntegrationExecutionListItem) => {
  dispatchPageAction(button, handlers, row)
}
onMounted(async () => {
  await loadMetadata()
  await schemePage.initialize()
  await refresh()
  initialized.value = true
})
watch(
  () => [query.value.page, query.value.num] as const,
  () => {
    if (initialized.value) void fetchData()
  },
)
watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([field, descending]) => {
    if (
      !initialized.value ||
      !queryState.applySorting(field || '', descending, new Set(['execution_no']))
    )
      return
    void fetchData()
  },
)
</script>

<style scoped>
.execution-page {
  gap: 8px;
}

.runtime-status-strip {
  min-height: 48px;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  background: var(--app-surface);
}

.execution-table {
  min-height: 0;
}
</style>
