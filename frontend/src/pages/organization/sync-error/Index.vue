<template>
  <base-content :scrollable="showDetailDialog && detailMode === 'page'" class="q-pa-sm">
    <scrollable-table
      v-if="!showDetailDialog || detailMode === 'dialog'"
      class="fit"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :loading="loading"
      :pagination="{ rowsPerPage: 0 }"
      hide-pagination
    >
      <template #top>
        <div class="row q-gutter-xs full-width">
          <div class="col-grow row q-gutter-xs">
            <q-input
              v-model="query.quick_query!.keyword"
              dense
              outlined
              debounce="300"
              placeholder="搜索关键词"
              @keyup.enter="search"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-btn color="primary" label="搜索" :disable="loading" @click="search" />
            <q-btn outline color="primary" icon="tune" @click="openAdvancedQuery">
              <q-tooltip>高级查询</q-tooltip>
            </q-btn>
          </div>
          <q-space />
          <div class="row q-gutter-xs">
            <q-btn
              v-for="button in refreshButtons"
              :key="button.id || button.code"
              v-bind="menuButtonDisplayProps(button)"
              :color="button.color || 'primary'"
              :disable="loading"
              @click="fetchData"
            />
          </div>
        </div>
      </template>

      <template #body-cell-action="props">
        <q-td :props="props">{{ dictLabel('org_sync_action', props.row.action) }}</q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props">
          <q-chip dense square outline :color="organizationStatusColor(props.row.status)">
            {{ dictLabel('org_sync_record_status', props.row.status) }}
          </q-chip>
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
          {{ loadError || '暂无同步异常' }}
        </div>
      </template>
      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </scrollable-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      :fields="advancedFields"
      title="同步异常高级查询"
      @search="applyAdvancedQuery"
    />

    <organization-record-detail-dialog
      v-model="showDetailDialog"
      title="同步记录详情"
      :subtitle="recordDetail?.source_code || ''"
      :sections="detailSections"
      icon="sync_problem"
      :status-label="
        recordDetail ? dictLabel('org_sync_record_status', recordDetail.status) : ''
      "
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
      :subtitle="currentRecord?.source_code || ''"
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

import cloneDeep from 'lodash/cloneDeep'
import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useRoute } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import ScrollableTable from 'src/components/Table/ScrollableTable.vue'
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
  organizationStatusColor,
} from 'src/pages/organization/organization-list-page'
import { useDictStore } from 'src/stores/dict'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const route = useRoute()
const dictStore = useDictStore()
const {
  line_buttons,
  top_buttons,
  record_detail_top_buttons,
  record_detail_bottom_buttons,
} = usePageButtons('organization_sync_error')
const detailMode = useOrganizationDetailMode('organization_sync_error', 'dialog')

const rows = ref<SyncRecordListItem[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const query = ref<SyncRecordQueryRequest>({
  ...createOrganizationQuery('org_sync_record'),
  status: 'failed',
})
const tempAdvancedQuery = ref<SyncRecordQueryRequest>(cloneDeep(query.value))
const showAdvancedQuery = ref(false)

const currentRecord = ref<SyncRecordListItem | null>(null)
const recordDetail = ref<SyncRecordDetail | null>(null)
const showDetailDialog = ref(false)
const detailLoading = ref(false)
const detailError = ref('')

const recordError = ref<SyncRecordError | null>(null)
const showErrorDialog = ref(false)
const errorLoading = ref(false)
const errorLoadError = ref('')

const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
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
  { name: 'source_code', field: 'source_code', label: '源对象编码', align: 'left' },
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
  createOrganizationField('源对象编码', 'source_code'),
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
  dictStore.getDictLabel(code, value) || String(value || '-')

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
        { label: '源对象编码', value: detail.source_code },
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
    { label: '错误信息', value: error.error_message, fullWidth: true },
    { label: '依赖类型', value: dictLabel('org_dependency_type', error.dependency_type) },
    { label: '依赖对象', value: error.dependency_key },
  ]
})

const objectTypeLabel = (value: string) =>
  objectTypeOptions.find((item) => item.value === value)?.label || value || '-'

const applyRouteFilters = () => {
  const objectType = String(route.query.object_type || '')
  const localId = Number(route.query.local_id)
  if (objectType) query.value.object_type = objectType
  if (Number.isInteger(localId) && localId > 0) query.value.local_id = localId
}

const search = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}

const openAdvancedQuery = () => {
  tempAdvancedQuery.value = cloneDeep(query.value)
  showAdvancedQuery.value = true
}

const applyAdvancedQuery = () => {
  query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions)
  showAdvancedQuery.value = false
  search()
}

const fetchData = async () => {
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
  () => void fetchData(),
)

onMounted(async () => {
  applyRouteFilters()
  await dictStore.loadDicts([
    'org_sync_action',
    'org_sync_record_status',
    'org_dependency_type',
  ])
  await fetchData()
})
</script>
