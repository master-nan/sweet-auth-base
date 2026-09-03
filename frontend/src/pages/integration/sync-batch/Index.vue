<template>
  <base-content class="q-pa-sm">
    <q-table
      v-model:pagination="pagination"
      class="fit sticky-header-table"
      color="primary"
      :dense="$q.screen.lt.md"
      separator="cell"
      flat
      bordered
      :rows="rows"
      :columns="columns"
      row-key="id"
      :loading="loading"
      :no-data-label="emptyMessage"
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
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
                  @keyup.enter="handleBasicSearch"
                >
                  <template #append><q-icon name="search" /></template>
                </q-input>
                <q-btn color="primary" :label="t('ui.search')" @click="handleBasicSearch" />
              </template>
            </query-scheme-controls>
          </template>
        </standard-table-toolbar>
      </template>
      <template #body-cell-batch_no="props">
        <q-td :props="props">
          <q-btn
            v-if="canDetail"
            flat
            dense
            no-caps
            color="primary"
            :label="props.row.batch_no"
            @click="openDetail(props.row.id)"
          />
          <span v-else class="text-mono">{{ props.row.batch_no }}</span>
          <div class="text-caption text-grey-7">
            {{ props.row.task_name }} · v{{ props.row.task_version }}
          </div>
        </q-td>
      </template>
      <template #body-cell-trigger_type="props"
        ><q-td :props="props">{{ triggerLabel(props.row.trigger_type) }}</q-td></template
      >
      <template #body-cell-status="props"
        ><q-td :props="props"
          ><status-chip
            :color="statusFor(props.row).color"
            :label="statusFor(props.row).label" /></q-td
      ></template>
      <template #body-cell-window="props"
        ><q-td :props="props"
          >{{ formatDate(props.row.window_start) }}
          <div class="text-caption text-grey-7">
            {{ t('ui.to') }} {{ formatDate(props.row.window_end) }}
          </div></q-td
        ></template
      >
      <template #body-cell-checkpoint="props"
        ><q-td :props="props"
          >{{ formatDate(props.row.checkpoint_before) }}
          <div class="text-caption text-grey-7">
            {{ t('ui.to') }} {{ formatDate(props.row.checkpoint_after) }}
          </div></q-td
        ></template
      >
      <template #body-cell-progress="props"
        ><q-td :props="props"
          >{{ props.row.current_slice_no }} / {{ props.row.planned_slice_count }}
          <div class="text-caption text-grey-7">
            Execution {{ props.row.execution_count }}
          </div></q-td
        ></template
      >
      <template #body-cell-result="props"
        ><q-td :props="props"
          >{{ t('ui.technology') }} {{ props.row.technical_success_count }} /
          {{ props.row.technical_failed_count }}
          <div class="text-caption text-grey-7">
            {{ t('ui.operations') }} {{ props.row.business_success_count }} /
            {{ props.row.business_failed_count }}
          </div></q-td
        ></template
      >
      <template #body-cell-reason="props"
        ><q-td :props="props">{{ props.row.reason_code || '-' }}</q-td></template
      >
      <template #body-cell-started_at="props"
        ><q-td :props="props">{{ formatDate(props.row.started_at) }}</q-td></template
      >
      <template #body-cell-completed_at="props"
        ><q-td :props="props">{{ formatDate(props.row.completed_at) }}</q-td></template
      >
      <template #bottom
        ><q-space /><table-pagination
          v-model:page="query.page"
          v-model:page-size="query.num"
          :total="total"
      /></template>
    </q-table>

    <form-dialog-shell
      v-model="showDetail"
      :title="t('ui.syncBatchDetails')"
      :subtitle="detail?.batch_no || t('ui.readingBatches')"
      icon="view_timeline"
      readonly
      :loading="detailLoading"
      width="min(1040px, calc(100vw - 48px))"
    >
      <template v-if="detail">
        <div class="row q-col-gutter-lg">
          <div v-for="item in detailItems" :key="item.label" class="col-12 col-sm-6 col-md-4">
            <div class="text-caption text-grey-7">{{ item.label }}</div>
            <div class="text-body1">{{ item.value }}</div>
          </div>
        </div>
        <q-separator class="q-my-md" />
        <div class="text-subtitle2 q-mb-sm">{{ t('ui.executionDetails') }}</div>
        <div v-if="!canQueryExecutions" class="text-body2 text-grey-7">
          {{ t('ui.noExecutionRecordViewingPermission') }}
        </div>
        <q-list v-else bordered separator>
          <q-item v-for="execution in executions" :key="execution.id">
            <q-item-section>
              <q-item-label>
                <q-btn
                  v-if="canViewExecutionDetail"
                  flat
                  dense
                  no-caps
                  color="primary"
                  :label="execution.execution_no"
                  @click="openExecution(execution.id)"
                />
                <span v-else class="text-mono">{{ execution.execution_no }}</span>
              </q-item-label>
              <q-item-label caption>
                Slice {{ execution.sync_source?.slice_no || '-' }} ·
                {{ formatDate(execution.sync_source?.window_start) }} {{ t('ui.to') }}
                {{ formatDate(execution.sync_source?.window_end) }}
              </q-item-label>
            </q-item-section>
            <q-item-section side>{{ execution.status }}</q-item-section>
          </q-item>
          <q-item v-if="executions.length === 0"
            ><q-item-section class="text-grey-7">{{
              t('ui.cannotInitialiseEvolutionSMailComponent')
            }}</q-item-section></q-item
          >
        </q-list>
      </template>
    </form-dialog-shell>
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'integration_sync_batch' })
import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import StatusChip from '@/components/Display/StatusChip.vue'
import FormDialogShell from '@/components/FormDialog/FormDialogShell.vue'
import QuerySchemeControls from '@/components/QueryScheme/QuerySchemeControls.vue'
import {
  type IntegrationExecutionListItem,
  type SyncBatchDetail,
  type SyncBatchListItem,
  type SyncBatchQuery,
  useIntegrationApi,
} from '@/api/services/integration'
import { usePageButtons } from '@/composables/page-buttons'
import { useTableQueryState } from '@/composables/table-query-state'
import { useRuntimeTableMetadata } from '@/composables/runtime-table-metadata'
import { useQuerySchemePage } from '@/composables/query-scheme-page'
import { formatRuntimeDateTime } from '@/pages/integration/runtime-display'
import { resolveTableEmptyMessage } from '@/utils/table-state'
import { countEffectiveQueryRules } from '@/utils/query-state'

const { t } = useI18n({ useScope: 'global' })

const router = useRouter()
const $q = useQuasar()
const api = useIntegrationApi()
const loading = ref(false)
const loadError = ref('')
const { hasGrantedCapability } = usePageButtons('integration_sync_batch')
const rows = ref<SyncBatchListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showDetail = ref(false)
const detailLoading = ref(false)
const detail = ref<SyncBatchDetail | null>(null)
const executions = ref<IntegrationExecutionListItem[]>([])
const canQueryBatches = computed(() => hasGrantedCapability('integration_sync_batch_query'))
const canDetail = computed(() => hasGrantedCapability('integration_sync_batch_detail'))
const canQueryExecutions = computed(() => hasGrantedCapability('integration_execution_query'))
const canViewExecutionDetail = computed(() => hasGrantedCapability('integration_execution_detail'))
const {
  quickSearchPlaceholder,
  advancedSearchFields: advancedFields,
  loadMetadata,
} = useRuntimeTableMetadata('integration_sync_batch')
const statusMeta = {
  created: {
    get label() {
      return t('ui.toBeRun')
    },
    color: 'grey-7',
  },
  running: {
    get label() {
      return t('ui.running')
    },
    color: 'primary',
  },
  succeeded: {
    get label() {
      return t('ui.success')
    },
    color: 'positive',
  },
  failed: {
    get label() {
      return t('ui.failed')
    },
    color: 'negative',
  },
}
const statusFor = (row: SyncBatchListItem) => statusMeta[row.status]
const triggerOptions = [
  {
    get label() {
      return t('ui.manual')
    },
    value: 'manual',
  },
  {
    get label() {
      return t('ui.scheduledTrigger')
    },
    value: 'scheduled',
  },
]
const triggerLabel = (value: string) =>
  triggerOptions.find((item) => item.value === value)?.label || value
const formatDate = formatRuntimeDateTime
const columns: QTableProps['columns'] = [
  {
    name: 'batch_no',
    get label() {
      return t('ui.batch')
    },
    field: 'batch_no',
    align: 'left',
    sortable: true,
  },
  {
    name: 'trigger_type',
    get label() {
      return t('ui.triggerType')
    },
    field: 'trigger_type',
    align: 'center',
    sortable: true,
  },
  {
    name: 'status',
    get label() {
      return t('ui.status')
    },
    field: 'status',
    align: 'center',
    sortable: true,
  },
  {
    name: 'window',
    get label() {
      return t('ui.logicalWindow')
    },
    field: 'window_start',
    align: 'left',
    sortable: true,
  },
  { name: 'checkpoint', label: 'Checkpoint', field: 'checkpoint_before', align: 'left' },
  {
    name: 'progress',
    get label() {
      return t('ui.sliceProgress')
    },
    field: 'current_slice_no',
    align: 'center',
  },
  {
    name: 'result',
    get label() {
      return t('ui.successFailed')
    },
    field: 'technical_success_count',
    align: 'center',
  },
  {
    name: 'reason',
    get label() {
      return t('ui.reason')
    },
    field: 'reason_code',
    align: 'left',
  },
  {
    name: 'started_at',
    get label() {
      return t('ui.startTime')
    },
    field: 'started_at',
    align: 'left',
    sortable: true,
  },
  {
    name: 'completed_at',
    get label() {
      return t('ui.endTime')
    },
    field: 'completed_at',
    align: 'left',
    sortable: true,
  },
]
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const queryState = useTableQueryState<SyncBatchQuery>({
  createInitialQuery: () => ({
    page: 1,
    num: 20,
    order: { field: '', is_asc: false },
    quick_query: { keyword: '' },
    expressions: emptyExpressions(),
  }),
})
const { query, keyword, appliedAdvanced: appliedAdvancedQuery } = queryState
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: canQueryBatches.value,
    error: loadError.value,
    hasQuery: !!keyword.value || activeFilterCount.value > 0,
  }),
)
const fetchData = async () => {
  if (!canQueryBatches.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await api.querySyncBatches(query.value)
    rows.value = result.data || []
    total.value = result.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = t('ui.failedToLoadSyncBatches')
  } finally {
    loading.value = false
  }
}
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('integration_sync_batch', queryState, resetAndFetch)
const handleBasicSearch = () => schemePage.runQueryChange(queryState.submitQuickSearch)
const openDetail = async (id: number) => {
  if (!canDetail.value) return
  showDetail.value = true
  detailLoading.value = true
  executions.value = []
  try {
    const requests: Promise<unknown>[] = [
      api.getSyncBatch(id).then((result) => {
        detail.value = result.data || null
      }),
    ]
    if (canQueryExecutions.value) {
      requests.push(
        api
          .queryExecutions({
            page: 1,
            num: 100,
            order: { field: 'sync_slice_no', is_asc: true },
            expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
            sync_batch_id: id,
          })
          .then((result) => {
            executions.value = result.data || []
          }),
      )
    }
    await Promise.all(requests)
  } finally {
    detailLoading.value = false
  }
}
const openExecution = (id: number) => {
  void router.push({ name: 'integration_execution_detail_page', params: { id } })
}
const detailItems = computed(() =>
  detail.value
    ? [
        {
          get label() {
            return t('ui.taskVersion')
          },
          value: `${detail.value.task_code} · v${detail.value.task_version}`,
        },
        {
          get label() {
            return t('ui.triggerType')
          },
          value: triggerLabel(detail.value.trigger_type),
        },
        {
          get label() {
            return t('ui.status')
          },
          value: statusMeta[detail.value.status].label,
        },
        {
          get label() {
            return t('ui.apiVersion')
          },
          value: `${detail.value.interface_code} · v${detail.value.interface_version}`,
        },
        {
          label: 'Consumer',
          value: `${detail.value.consumer_code} · v${detail.value.consumer_version}`,
        },
        {
          get label() {
            return t('ui.logicalWindow')
          },
          value: t('ui.rangeFromTo', {
            value1: formatDate(detail.value.window_start),
            value2: formatDate(detail.value.window_end),
          }),
        },
        {
          label: 'Checkpoint',
          value: t('ui.rangeFromTo', {
            value1: formatDate(detail.value.checkpoint_before),
            value2: formatDate(detail.value.checkpoint_after),
          }),
        },
        {
          get label() {
            return t('ui.sliceProgress')
          },
          value: `${detail.value.current_slice_no} / ${detail.value.planned_slice_count}（Execution ${detail.value.execution_count}）`,
        },
        {
          get label() {
            return t('ui.technicalResults')
          },
          value: t('ui.successfulFailed', {
            value1: detail.value.technical_success_count,
            value2: detail.value.technical_failed_count,
          }),
        },
        {
          get label() {
            return t('ui.businessResult')
          },
          value: t('ui.successfulFailed', {
            value1: detail.value.business_success_count,
            value2: detail.value.business_failed_count,
          }),
        },
        {
          get label() {
            return t('ui.reason')
          },
          value: detail.value.reason_code || '-',
        },
        {
          get label() {
            return t('ui.startFinish')
          },
          value: `${formatDate(detail.value.started_at)} / ${formatDate(detail.value.completed_at)}`,
        },
        {
          get label() {
            return t('ui.summaryOfResults')
          },
          value: detail.value.result_summary || '-',
        },
      ]
    : [],
)
onMounted(async () => {
  await loadMetadata()
  await schemePage.initialize()
  await fetchData()
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
      !queryState.applySorting(
        field || '',
        descending,
        new Set([
          'batch_no',
          'trigger_type',
          'status',
          'window_start',
          'started_at',
          'completed_at',
        ]),
      )
    )
      return
    resetAndFetch()
  },
)
</script>
