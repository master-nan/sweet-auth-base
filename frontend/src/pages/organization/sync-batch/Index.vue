<template>
  <base-content class="q-pa-sm">
    <q-table
      class="fit sticky-header-table"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :loading="loading"
      :no-data-label="emptyMessage"
      v-model:pagination="pagination"
      hide-pagination
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #query-controls>
            <query-scheme-controls
              :controller="schemePage"
              :query-state="queryState"
              :fields="advancedFields"
              :advanced-title="t('ui.syncBatchAdvancedQueries')"
              :show-filter-count="false"
            >
              <template #quick-search>
                <q-input
                  v-model="keyword"
                  dense
                  outlined
                  debounce="300"
                  :placeholder="t('ui.searchBatchNumbers')"
                  @keyup.enter="search"
                >
                  <template #append><q-icon name="search" /></template>
                </q-input>
                <q-btn color="primary" :label="t('ui.search')" :disable="loading" @click="search" />
              </template>
            </query-scheme-controls>
          </template>
        </standard-table-toolbar>
      </template>

      <template #body-cell-sync_type="props">
        <q-td :props="props">{{ dictLabel('org_sync_type', props.row.sync_type) }}</q-td>
      </template>
      <template #body-cell-object_scope="props">
        <q-td :props="props">{{ organizationSyncObjectLabel(props.row.object_scope) }}</q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props">
          <status-chip
            :color="organizationStatusColor(props.row.status)"
            :label="dictLabel('org_sync_record_status', props.row.status)"
          />
        </q-td>
      </template>
      <template #body-cell-started_at="props">
        <q-td :props="props">{{ formatOrganizationDateTime(props.row.started_at) }}</q-td>
      </template>
      <template #body-cell-duration="props">
        <q-td :props="props">
          {{ formatDuration(props.row.started_at, props.row.completed_at) }}
        </q-td>
      </template>
      <template #body-cell-progress="props">
        <q-td :props="props">
          <span class="text-positive">{{ props.row.success_count }}</span>
          <span class="text-grey-6"> / </span>
          <span :class="props.row.failed_count ? 'text-negative' : ''">{{
            props.row.failed_count
          }}</span>
          <span class="text-grey-6"> / {{ props.row.total_count }}</span>
        </q-td>
      </template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            v-for="button in visibleRowButtons(props.row)"
            :key="button.id || button.code"
            flat
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            size="sm"
            @click="handleRowAction(button, props.row)"
          >
            <q-tooltip>{{ button.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>
      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <organization-record-detail-dialog
      v-model="showErrorDialog"
      :title="t('ui.syncBatchErrors')"
      :subtitle="currentBatch?.batch_no || ''"
      :items="errorItems"
      icon="error_outline"
      :status-label="t('ui.failed')"
      status-color="negative"
      :loading="errorLoading"
      :error="errorLoadError"
    />
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'organization_sync_batch' })

import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import QuerySchemeControls from '@/components/QueryScheme/QuerySchemeControls.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import StatusChip from '@/components/Display/StatusChip.vue'
import {
  getSyncBatchError,
  querySyncBatches,
  type SyncBatchListItem,
  type SyncBatchQueryRequest,
} from '@/api/services/org'
import type { MenuButton } from '@/api/services/sys-menu'
import { usePageButtons } from '@/composables/page-buttons'
import { useTableQueryState } from '@/composables/table-query-state'
import { useQuerySchemePage } from '@/composables/query-scheme-page'
import OrganizationRecordDetailDialog from '@/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type { OrganizationDetailItem } from '@/pages/organization/components/organization-record-detail'
import {
  createOrganizationField,
  createOrganizationQuery,
  formatOrganizationDateTime,
  formatOrganizationValue,
  organizationSyncObjectLabel,
  organizationStatusColor,
} from '@/pages/organization/organization-list-page'
import { buildOrganizationDetailRoute } from '@/pages/organization/organization-detail-route'
import { useDictStore } from '@/stores/dict'
import { SysTableFieldInputType, SysTableFieldType } from '@/types/enum'
import { menuButtonDisplayProps } from '@/utils/menu-button-display'
import { resolveTableEmptyMessage } from '@/utils/table-state'

const { t } = useI18n({ useScope: 'global' })

const dictStore = useDictStore()
const router = useRouter()
const { line_buttons, hasGrantedCapability } = usePageButtons('organization_sync_batch')

const rows = ref<SyncBatchListItem[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const queryState = useTableQueryState<SyncBatchQueryRequest>({
  createInitialQuery: () => createOrganizationQuery('org_sync_batch'),
})
const { query, keyword } = queryState
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('organization_sync_batch', queryState, resetAndFetch)
const initialized = ref(false)
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const canQueryBatches = computed(() => hasGrantedCapability('organization_sync_batch_query'))
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: canQueryBatches.value,
    error: loadError.value,
    hasQuery: !!keyword.value,
  }),
)

const currentBatch = ref<SyncBatchListItem | null>(null)

const showErrorDialog = ref(false)
const errorLoading = ref(false)
const errorLoadError = ref('')
const errorSummary = ref('')

const errorItems = computed<OrganizationDetailItem[]>(() => [
  {
    get label() {
      return t('ui.errorSummary')
    },
    value: errorSummary.value,
    fullWidth: true,
  },
])
const columns: QTableProps['columns'] = [
  {
    name: 'batch_no',
    field: 'batch_no',
    get label() {
      return t('ui.batchNumber')
    },
    align: 'left',
    sortable: true,
  },
  {
    name: 'sync_type',
    field: 'sync_type',
    get label() {
      return t('ui.syncType')
    },
    align: 'center',
  },
  {
    name: 'object_scope',
    field: 'object_scope',
    get label() {
      return t('ui.objectScope')
    },
    align: 'left',
  },
  {
    name: 'started_at',
    field: 'started_at',
    get label() {
      return t('ui.startTime')
    },
    align: 'left',
    sortable: true,
  },
  {
    name: 'duration',
    field: 'duration',
    get label() {
      return t('ui.duration')
    },
    align: 'right',
  },
  {
    name: 'progress',
    field: 'progress',
    get label() {
      return t('ui.successFailureTotal')
    },
    align: 'right',
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
    name: 'actions',
    field: 'actions',
    get label() {
      return t('ui.actions')
    },
    align: 'center',
  },
]

const advancedFields = [
  createOrganizationField(t('ui.batchNumber'), 'batch_no'),
  createOrganizationField(t('ui.syncType'), 'sync_type', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_sync_type',
  }),
  createOrganizationField(t('ui.objectScope'), 'object_scope'),
  createOrganizationField(t('ui.batchStatus'), 'status', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_sync_record_status',
  }),
  createOrganizationField(t('ui.startTime'), 'started_at', SysTableFieldType.DATETIME, {
    inputType: SysTableFieldInputType.DATETIME_PICKER,
  }),
]

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || formatOrganizationValue(value)

const visibleRowButtons = (row?: SyncBatchListItem) => {
  void row
  return line_buttons.value.filter((button) => button.event_action === 'detail')
}

const formatDuration = (startedAt?: string | null, completedAt?: string | null) => {
  if (!startedAt || !completedAt) return '-'
  const milliseconds = new Date(completedAt).getTime() - new Date(startedAt).getTime()
  if (!Number.isFinite(milliseconds) || milliseconds < 0) return '-'
  if (milliseconds < 1000) return `${milliseconds} ms`
  return `${(milliseconds / 1000).toFixed(1)} s`
}

const search = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

const fetchData = async () => {
  if (!canQueryBatches.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await querySyncBatches(query.value)
    rows.value = result.items
    total.value = result.total
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = t('ui.failedToLoadSyncBatches')
  } finally {
    loading.value = false
  }
}

const openDetail = async (row: SyncBatchListItem) => {
  await router.push(buildOrganizationDetailRoute('org_sync_batch', row.id, row.batch_no))
}

const openError = async (row: SyncBatchListItem) => {
  currentBatch.value = row
  errorSummary.value = ''
  errorLoadError.value = ''
  errorLoading.value = true
  showErrorDialog.value = true
  try {
    const result = await getSyncBatchError(row.id)
    errorSummary.value = result.error_summary
  } catch {
    errorLoadError.value = t('ui.failedToLoadSyncBatchErrors')
  } finally {
    errorLoading.value = false
  }
}

const handleRowAction = (button: MenuButton, row: SyncBatchListItem) => {
  if (button.event_action === 'detail') void openDetail(row)
  if (button.event_action === 'view_error') void openError(row)
}

watch(
  () => [query.value.page, query.value.num],
  () => {
    if (initialized.value) void fetchData()
  },
)

watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([field, descending]) => {
    if (!initialized.value) return
    if (!queryState.applySorting(field || '', descending, new Set(['batch_no', 'started_at'])))
      return
    void fetchData()
  },
)

onMounted(async () => {
  await dictStore.loadDicts(['org_sync_type', 'org_sync_action', 'org_sync_record_status'])
  await schemePage.initialize()
  await fetchData()
  initialized.value = true
})
</script>
