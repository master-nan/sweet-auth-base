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

      <template #body-cell-sync_type="props">
        <q-td :props="props">{{ dictLabel('org_sync_type', props.row.sync_type) }}</q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props">
          <q-chip dense square outline :color="organizationStatusColor(props.row.status)">
            {{ dictLabel('org_sync_record_status', props.row.status) }}
          </q-chip>
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
      <template #no-data>
        <div class="full-width row flex-center q-pa-xl text-grey-7">
          {{ loadError || '暂无同步批次' }}
        </div>
      </template>
      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      :fields="advancedFields"
      title="同步批次高级查询"
      @search="applyAdvancedQuery"
    />

    <organization-record-detail-dialog
      v-model="showErrorDialog"
      title="同步批次错误"
      :subtitle="currentBatch?.batch_no || ''"
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
defineOptions({ name: 'organization_sync_batch' })

import cloneDeep from 'lodash/cloneDeep'
import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import {
  getSyncBatchError,
  querySyncBatches,
  type SyncBatchListItem,
  type SyncBatchQueryRequest,
} from 'src/api/services/org'
import type { MenuButton } from 'src/api/services/sys-menu'
import { usePageButtons } from 'src/composables/page-buttons'
import OrganizationRecordDetailDialog from 'src/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type { OrganizationDetailItem } from 'src/pages/organization/components/organization-record-detail'
import {
  createOrganizationField,
  createOrganizationQuery,
  formatOrganizationDateTime,
  organizationStatusColor,
} from 'src/pages/organization/organization-list-page'
import { buildOrganizationDetailRoute } from 'src/pages/organization/organization-detail-route'
import { useDictStore } from 'src/stores/dict'
import { useUserStore } from 'src/stores/user'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const dictStore = useDictStore()
const userStore = useUserStore()
const router = useRouter()
const { line_buttons, top_buttons } = usePageButtons('organization_sync_batch')

const rows = ref<SyncBatchListItem[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const query = ref<SyncBatchQueryRequest>(createOrganizationQuery('org_sync_batch'))
const tempAdvancedQuery = ref<SyncBatchQueryRequest>(cloneDeep(query.value))
const showAdvancedQuery = ref(false)
const canQueryBatches = computed(() => userStore.buttons.includes('organization_sync_batch_query'))

const currentBatch = ref<SyncBatchListItem | null>(null)

const showErrorDialog = ref(false)
const errorLoading = ref(false)
const errorLoadError = ref('')
const errorSummary = ref('')

const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
const errorItems = computed<OrganizationDetailItem[]>(() => [
  { label: '错误摘要', value: errorSummary.value, fullWidth: true },
])
const columns: QTableProps['columns'] = [
  { name: 'batch_no', field: 'batch_no', label: '批次号', align: 'left', sortable: true },
  { name: 'sync_type', field: 'sync_type', label: '同步类型', align: 'center' },
  { name: 'object_scope', field: 'object_scope', label: '对象范围', align: 'left' },
  { name: 'started_at', field: 'started_at', label: '开始时间', align: 'left', sortable: true },
  { name: 'duration', field: 'duration', label: '耗时', align: 'right' },
  { name: 'progress', field: 'progress', label: '成功 / 失败 / 总数', align: 'right' },
  { name: 'status', field: 'status', label: '状态', align: 'center' },
  { name: 'actions', field: 'actions', label: '操作', align: 'center' },
]

const advancedFields = [
  createOrganizationField('批次号', 'batch_no'),
  createOrganizationField('同步类型', 'sync_type', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_sync_type',
  }),
  createOrganizationField('对象范围', 'object_scope'),
  createOrganizationField('批次状态', 'status', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_sync_record_status',
  }),
  createOrganizationField('开始时间', 'started_at', SysTableFieldType.DATETIME, {
    inputType: SysTableFieldInputType.DATETIME_PICKER,
  }),
]

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || String(value || '-')

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
    loadError.value = '同步批次加载失败'
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
    errorLoadError.value = '同步批次错误加载失败'
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
  () => void fetchData(),
)

onMounted(async () => {
  await dictStore.loadDicts([
    'org_sync_type',
    'org_sync_action',
    'org_sync_record_status',
  ])
  await fetchData()
})
</script>
