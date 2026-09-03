<template>
  <base-content :scrollable="showDetailDialog && detailMode === 'page'" class="q-pa-sm">
    <q-table
      v-if="!showDetailDialog || detailMode === 'dialog'"
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
              :advanced-title="t('ui.synchronizationOfAtypicalAdvancedQuery')"
              :show-filter-count="false"
            >
              <template #quick-search>
                <q-input
                  v-model="keyword"
                  dense
                  outlined
                  debounce="300"
                  :placeholder="t('ui.searchSourceCodeErrorCode')"
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

      <template #body-cell-action="props">
        <q-td :props="props">{{ dictLabel('org_sync_action', props.row.action) }}</q-td>
      </template>
      <template #body-cell-object_type="props">
        <q-td :props="props">{{ organizationSyncObjectLabel(props.row.object_type) }}</q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props">
          <status-chip
            :color="organizationStatusColor(props.row.status)"
            :label="dictLabel('org_sync_record_status', props.row.status)"
          />
        </q-td>
      </template>
      <template #body-cell-last_retry_at="props">
        <q-td :props="props">{{ formatOrganizationDateTime(props.row.last_retry_at) }}</q-td>
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
      v-model="showDetailDialog"
      :title="t('ui.synchroniseRecordDetails')"
      :subtitle="recordDetail?.source_summary || ''"
      :sections="detailSections"
      icon="sync_problem"
      :status-label="recordDetail ? dictLabel('org_sync_record_status', recordDetail.status) : ''"
      :status-color="recordDetail ? organizationStatusColor(recordDetail.status) : 'negative'"
      :loading="detailLoading"
      :error="detailError"
      :mode="detailMode"
      :top-buttons="record_detail_top_buttons"
      :bottom-buttons="record_detail_bottom_buttons"
      :record-context="currentRecord"
      @button-click="handleDetailAction"
    />

    <organization-record-detail-dialog
      v-model="showErrorDialog"
      :title="t('ui.syncErrorDetails')"
      :subtitle="currentRecord?.source_summary || ''"
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

defineOptions({ name: 'organization_sync_error' })

import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useRoute } from 'vue-router'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import QuerySchemeControls from '@/components/QueryScheme/QuerySchemeControls.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import StatusChip from '@/components/Display/StatusChip.vue'
import {
  getSyncRecordDetail,
  getSyncRecordError,
  querySyncRecords,
  type SyncRecordDetail,
  type SyncRecordError,
  type SyncRecordListItem,
  type SyncRecordQueryRequest,
} from '@/api/services/org'
import type { MenuButton } from '@/api/services/sys-menu'
import { usePageButtons } from '@/composables/page-buttons'
import { useTableQueryState } from '@/composables/table-query-state'
import { useQuerySchemePage } from '@/composables/query-scheme-page'
import OrganizationRecordDetailDialog from '@/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type {
  OrganizationDetailItem,
  OrganizationDetailSection,
} from '@/pages/organization/components/organization-record-detail'
import { useOrganizationDetailMode } from '@/pages/organization/use-organization-detail-mode'
import {
  createOrganizationField,
  createOrganizationQuery,
  formatOrganizationDateTime,
  formatOrganizationValue,
  organizationSyncObjectLabel,
  organizationStatusColor,
} from '@/pages/organization/organization-list-page'
import { useDictStore } from '@/stores/dict'
import { ExpressionType, SysTableFieldInputType, SysTableFieldType } from '@/types/enum'
import { menuButtonDisplayProps } from '@/utils/menu-button-display'
import { resolveTableEmptyMessage } from '@/utils/table-state'
import { countEffectiveQueryRules } from '@/utils/query-state'

const { t } = useI18n({ useScope: 'global' })

const route = useRoute()
const dictStore = useDictStore()
const {
  line_buttons,
  record_detail_top_buttons,
  record_detail_bottom_buttons,
  hasGrantedCapability,
} = usePageButtons('organization_sync_error')
const detailMode = useOrganizationDetailMode('organization_sync_error', 'dialog')

const rows = ref<SyncRecordListItem[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const canQueryRecords = computed(() => hasGrantedCapability('organization_sync_error_query'))
const routeObjectType = String(route.query.object_type || '').trim()
const routeLocalId = Number(route.query.local_id)
const hasRouteContext = !!routeObjectType || (Number.isInteger(routeLocalId) && routeLocalId > 0)
const queryState = useTableQueryState<SyncRecordQueryRequest>({
  createInitialQuery: () => {
    const initial = { ...createOrganizationQuery('org_sync_record'), status: 'failed' }
    const rules: NonNullable<SyncRecordQueryRequest['expressions']>[number]['rules'] = []
    if (routeObjectType) {
      rules.push({
        field: 'object_type',
        expression_type: ExpressionType.EQ,
        value: routeObjectType,
      })
    }
    if (Number.isInteger(routeLocalId) && routeLocalId > 0) {
      rules.push({ field: 'local_id', expression_type: ExpressionType.EQ, value: routeLocalId })
    }
    if (rules.length) initial.expressions = [{ rules, nested: [] }]
    return initial
  },
})
const { query, keyword, appliedAdvanced } = queryState
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvanced.value))
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('organization_sync_error', queryState, resetAndFetch)
const initialized = ref(false)
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: canQueryRecords.value,
    error: loadError.value,
    hasQuery: !!keyword.value || activeFilterCount.value > 0,
  }),
)

const currentRecord = ref<SyncRecordListItem | null>(null)
const recordDetail = ref<SyncRecordDetail | null>(null)
const showDetailDialog = ref(false)
const detailLoading = ref(false)
const detailError = ref('')

const recordError = ref<SyncRecordError | null>(null)
const showErrorDialog = ref(false)
const errorLoading = ref(false)
const errorLoadError = ref('')

const columns: QTableProps['columns'] = [
  {
    name: 'batch_id',
    field: 'batch_id',
    get label() {
      return t('ui.batchId')
    },
    align: 'right',
    sortable: true,
  },
  {
    name: 'object_type',
    field: 'object_type',
    get label() {
      return t('ui.objectType')
    },
    align: 'left',
  },
  {
    name: 'source_summary',
    field: 'source_summary',
    get label() {
      return t('ui.sourceSummary')
    },
    align: 'left',
  },
  {
    name: 'action',
    field: 'action',
    get label() {
      return t('ui.action')
    },
    align: 'center',
  },
  {
    name: 'status',
    field: 'status',
    get label() {
      return t('ui.processingStatus')
    },
    align: 'center',
  },
  {
    name: 'error_code',
    field: 'error_code',
    get label() {
      return t('ui.errorCode')
    },
    align: 'left',
  },
  {
    name: 'retry_count',
    field: 'retry_count',
    get label() {
      return t('ui.retryCount')
    },
    align: 'right',
  },
  {
    name: 'last_retry_at',
    field: 'last_retry_at',
    get label() {
      return t('ui.recentTry')
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
]

const advancedFields = [
  createOrganizationField(t('ui.batchId'), 'batch_id', SysTableFieldType.BIGINT, {
    inputType: SysTableFieldInputType.INPUT_NUMBER,
  }),
  createOrganizationField(t('ui.objectType'), 'object_type'),
  createOrganizationField(t('ui.localObjectId'), 'local_id', SysTableFieldType.BIGINT, {
    inputType: SysTableFieldInputType.INPUT_NUMBER,
  }),
  createOrganizationField(t('ui.syncAction'), 'action', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_sync_action',
  }),
  createOrganizationField(t('ui.processingStatus'), 'status', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_sync_record_status',
  }),
  createOrganizationField(t('ui.errorCode'), 'error_code'),
  createOrganizationField(t('ui.retryCount'), 'retry_count', SysTableFieldType.INT, {
    inputType: SysTableFieldInputType.INPUT_NUMBER,
  }),
]

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || formatOrganizationValue(value)

const visibleRowButtons = (row?: SyncRecordListItem) => {
  void row
  return line_buttons.value.filter((button) => button.event_action === 'detail')
}

const detailSections = computed<OrganizationDetailSection[]>(() => {
  const detail = recordDetail.value
  if (!detail) return []
  return [
    {
      key: 'basic',
      get label() {
        return t('ui.basicInformation')
      },
      get caption() {
        return t('ui.objectAndProcessStatus')
      },
      icon: 'info',
      items: [
        {
          get label() {
            return t('ui.batchId')
          },
          value: detail.batch_id,
        },
        {
          get label() {
            return t('ui.integrationExecutionId')
          },
          value: detail.execution_id ?? null,
        },
        {
          get label() {
            return t('ui.objectType')
          },
          value: objectTypeLabel(detail.object_type),
        },
        {
          get label() {
            return t('ui.sourceSummary')
          },
          value: detail.source_summary,
        },
        {
          get label() {
            return t('ui.localObjectId')
          },
          value: detail.local_id ?? null,
        },
        {
          get label() {
            return t('ui.syncAction')
          },
          value: dictLabel('org_sync_action', detail.action),
        },
        {
          get label() {
            return t('ui.processingStatus')
          },
          value: dictLabel('org_sync_record_status', detail.status),
          chip: true,
          color: organizationStatusColor(detail.status),
        },
      ],
    },
    {
      key: 'retry',
      get label() {
        return t('ui.retryAndDependence')
      },
      get caption() {
        return t('ui.relianceAndRetryInformation')
      },
      icon: 'replay',
      items: [
        {
          get label() {
            return t('ui.dependencyType')
          },
          value: dictLabel('org_dependency_type', detail.dependency_type),
        },
        {
          get label() {
            return t('ui.retryCount')
          },
          value: detail.retry_count,
        },
        {
          get label() {
            return t('ui.lastRetryTime')
          },
          value: formatOrganizationDateTime(detail.last_retry_at),
        },
      ],
    },
  ]
})

const errorItems = computed<OrganizationDetailItem[]>(() => {
  const error = recordError.value
  if (!error) return []
  return [
    {
      get label() {
        return t('ui.errorCode')
      },
      value: error.error_code,
    },
    {
      get label() {
        return t('ui.dependencyType')
      },
      value: dictLabel('org_dependency_type', error.dependency_type),
    },
    {
      get label() {
        return t('ui.relianceOnSummary')
      },
      value: error.dependency_summary,
    },
  ]
})

const objectTypeLabel = (value: string) => organizationSyncObjectLabel(value)

const search = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

const fetchData = async () => {
  if (!canQueryRecords.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await querySyncRecords(query.value)
    rows.value = result.items
    total.value = result.total
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = t('ui.synchronisingAbnormalLoadFailed')
  } finally {
    loading.value = false
  }
}

const openDetail = async (row: SyncRecordListItem) => {
  currentRecord.value = row
  recordDetail.value = null
  detailError.value = ''
  detailLoading.value = true
  showDetailDialog.value = true
  try {
    recordDetail.value = await getSyncRecordDetail(row.id)
  } catch {
    detailError.value = t('ui.synchronisingRecordingDetailsLoadedFailed')
  } finally {
    detailLoading.value = false
  }
}

const openError = async (row: SyncRecordListItem) => {
  currentRecord.value = row
  recordError.value = null
  errorLoadError.value = ''
  errorLoading.value = true
  showErrorDialog.value = true
  try {
    recordError.value = await getSyncRecordError(row.id)
  } catch {
    errorLoadError.value = t('ui.synchronisingErrorDetailsLoadedFailed')
  } finally {
    errorLoading.value = false
  }
}

const handleRowAction = (button: MenuButton, row: SyncRecordListItem) => {
  if (button.event_action === 'detail') void openDetail(row)
  if (button.event_action === 'view_error') void openError(row)
}

const handleDetailAction = (button: MenuButton) => {
  if (!currentRecord.value) return
  if (button.event_action === 'view_error') void openError(currentRecord.value)
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
    if (!queryState.applySorting(field || '', descending, new Set(['batch_id']))) return
    void fetchData()
  },
)

onMounted(async () => {
  await dictStore.loadDicts(['org_sync_action', 'org_sync_record_status', 'org_dependency_type'])
  await schemePage.initialize({ preserveInitialQuery: hasRouteContext })
  await fetchData()
  initialized.value = true
})
</script>
