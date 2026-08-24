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
              advanced-title="同步异常高级查询"
              :show-filter-count="false"
            >
              <template #quick-search>
                <q-input
                  v-model="keyword"
                  dense
                  outlined
                  debounce="300"
                  placeholder="搜索来源编码、错误编码"
                  @keyup.enter="search"
                >
                  <template #append><q-icon name="search" /></template>
                </q-input>
                <q-btn color="primary" label="搜索" :disable="loading" @click="search" />
              </template>
            </query-scheme-controls>
          </template>
        </standard-table-toolbar>
      </template>

      <template #body-cell-action="props">
        <q-td :props="props">{{ dictLabel('org_sync_action', props.row.action) }}</q-td>
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
      <template #no-data>
        <div class="full-width row flex-center q-pa-xl text-grey-7">
          {{ emptyMessage }}
        </div>
      </template>
      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <organization-record-detail-dialog
      v-model="showDetailDialog"
      title="同步记录详情"
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
      title="同步错误详情"
      :subtitle="currentRecord?.source_summary || ''"
      :items="errorItems"
      icon="error_outline"
      status-label="失败"
      status-color="negative"
      :loading="errorLoading"
      :error="errorLoadError"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'organization_sync_error' })

import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useRoute } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import {
  getSyncRecordDetail,
  getSyncRecordError,
  querySyncRecords,
  type SyncRecordDetail,
  type SyncRecordError,
  type SyncRecordListItem,
  type SyncRecordQueryRequest,
} from 'src/api/services/org'
import type { MenuButton } from 'src/api/services/sys-menu'
import { usePageButtons } from 'src/composables/page-buttons'
import { useTableQueryState } from 'src/composables/table-query-state'
import { useQuerySchemePage } from 'src/composables/query-scheme-page'
import OrganizationRecordDetailDialog from 'src/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type {
  OrganizationDetailItem,
  OrganizationDetailSection,
} from 'src/pages/organization/components/organization-record-detail'
import { useOrganizationDetailMode } from 'src/pages/organization/use-organization-detail-mode'
import {
  createOrganizationField,
  createOrganizationQuery,
  formatOrganizationDateTime,
  formatOrganizationValue,
  organizationStatusColor,
} from 'src/pages/organization/organization-list-page'
import { useDictStore } from 'src/stores/dict'
import { ExpressionType, SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { resolveTableEmptyMessage } from 'src/utils/table-state'
import { countEffectiveQueryRules } from 'src/utils/query-state'

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

const objectTypeOptions = [
  { label: '法人主体', value: 'legal_entity' },
  { label: '组织单元', value: 'org_unit' },
  { label: '管理架构节点', value: 'structure_node' },
  { label: '人员', value: 'employee' },
  { label: '岗位', value: 'position' },
  { label: '任职', value: 'assignment' },
]

const columns: QTableProps['columns'] = [
  { name: 'batch_id', field: 'batch_id', label: '批次ID', align: 'right', sortable: true },
  { name: 'object_type', field: 'object_type', label: '对象类型', align: 'left' },
  { name: 'source_summary', field: 'source_summary', label: '来源摘要', align: 'left' },
  { name: 'action', field: 'action', label: '动作', align: 'center' },
  { name: 'status', field: 'status', label: '处理状态', align: 'center' },
  { name: 'error_code', field: 'error_code', label: '错误码', align: 'left' },
  { name: 'retry_count', field: 'retry_count', label: '重试次数', align: 'right' },
  { name: 'last_retry_at', field: 'last_retry_at', label: '最近重试', align: 'left' },
  { name: 'actions', field: 'actions', label: '操作', align: 'center' },
]

const advancedFields = [
  createOrganizationField('批次ID', 'batch_id', SysTableFieldType.BIGINT, {
    inputType: SysTableFieldInputType.INPUT_NUMBER,
  }),
  createOrganizationField('对象类型', 'object_type'),
  createOrganizationField('本地对象ID', 'local_id', SysTableFieldType.BIGINT, {
    inputType: SysTableFieldInputType.INPUT_NUMBER,
  }),
  createOrganizationField('同步动作', 'action', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_sync_action',
  }),
  createOrganizationField('处理状态', 'status', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_sync_record_status',
  }),
  createOrganizationField('错误码', 'error_code'),
  createOrganizationField('重试次数', 'retry_count', SysTableFieldType.INT, {
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
      label: '基本信息',
      caption: '对象与处理状态',
      icon: 'info',
      items: [
        { label: '批次ID', value: detail.batch_id },
        { label: '集成执行ID', value: detail.execution_id ?? null },
        { label: '对象类型', value: objectTypeLabel(detail.object_type) },
        { label: '来源摘要', value: detail.source_summary },
        { label: '本地对象ID', value: detail.local_id ?? null },
        { label: '同步动作', value: dictLabel('org_sync_action', detail.action) },
        {
          label: '处理状态',
          value: dictLabel('org_sync_record_status', detail.status),
          chip: true,
          color: organizationStatusColor(detail.status),
        },
      ],
    },
    {
      key: 'retry',
      label: '重试与依赖',
      caption: '依赖和重试信息',
      icon: 'replay',
      items: [
        { label: '依赖类型', value: dictLabel('org_dependency_type', detail.dependency_type) },
        { label: '重试次数', value: detail.retry_count },
        { label: '最近重试时间', value: formatOrganizationDateTime(detail.last_retry_at) },
      ],
    },
  ]
})

const errorItems = computed<OrganizationDetailItem[]>(() => {
  const error = recordError.value
  if (!error) return []
  return [
    { label: '错误码', value: error.error_code },
    { label: '依赖类型', value: dictLabel('org_dependency_type', error.dependency_type) },
    { label: '依赖摘要', value: error.dependency_summary },
  ]
})

const objectTypeLabel = (value: string) =>
  objectTypeOptions.find((item) => item.value === value)?.label || value || '-'

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
    loadError.value = '同步异常加载失败'
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
    detailError.value = '同步记录详情加载失败'
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
    errorLoadError.value = '同步错误详情加载失败'
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
