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
        <div class="row q-col-gutter-sm items-center full-width">
          <div class="col-12 col-md">
            <q-input
              v-model="query.quick_query!.keyword"
              dense
              outlined
              debounce="300"
              placeholder="搜索批次号或对象范围"
              @keyup.enter="search"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
          </div>
          <div class="col-6 col-sm-auto">
            <q-select
              v-model="query.sync_type"
              dense
              outlined
              clearable
              emit-value
              map-options
              :options="syncTypeOptions"
              label="同步类型"
            />
          </div>
          <div class="col-6 col-sm-auto">
            <q-select
              v-model="query.status"
              dense
              outlined
              clearable
              emit-value
              map-options
              :options="syncStatusOptions"
              label="批次状态"
            />
          </div>
          <div class="col-auto">
            <q-btn color="primary" icon="search" label="查询" :disable="loading" @click="search" />
          </div>
          <div class="col-auto">
            <q-btn flat round color="primary" icon="tune" @click="openAdvancedQuery">
              <q-tooltip>高级查询</q-tooltip>
            </q-btn>
          </div>
          <q-space />
          <div class="col-auto">
            <q-btn
              v-for="button in refreshButtons"
              :key="button.id || button.code"
              flat
              round
              :icon="button.icon || 'refresh'"
              :color="button.color || 'primary'"
              :loading="loading"
              @click="fetchData"
            >
              <q-tooltip>{{ button.name }}</q-tooltip>
            </q-btn>
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
            dense
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
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

    <q-dialog v-model="showDetailDialog">
      <q-card style="width: 1000px; max-width: 95vw">
        <q-card-section class="row items-start no-wrap">
          <div>
            <div class="text-h6">{{ batchDetail?.batch_no || '同步批次详情' }}</div>
            <div class="text-caption text-grey-7">
              {{ batchDetail ? dictLabel('org_sync_type', batchDetail.sync_type) : '' }}
            </div>
          </div>
          <q-space />
          <q-btn v-close-popup flat round dense icon="close"><q-tooltip>关闭</q-tooltip></q-btn>
        </q-card-section>
        <q-separator />
        <q-card-section v-if="detailLoading" class="row justify-center q-pa-xl">
          <q-spinner color="primary" size="32px" />
        </q-card-section>
        <q-banner v-else-if="detailError" class="text-negative">{{ detailError }}</q-banner>
        <template v-else-if="batchDetail">
          <q-list separator>
            <q-item>
              <q-item-section>
                <q-item-label caption>对象范围</q-item-label>
                <q-item-label>{{ batchDetail.object_scope }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>状态</q-item-label>
                <q-item-label>{{
                  dictLabel('org_sync_record_status', batchDetail.status)
                }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>集成执行ID</q-item-label>
                <q-item-label>{{ batchDetail.execution_id || '-' }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>开始时间</q-item-label>
                <q-item-label>{{
                  formatOrganizationDateTime(batchDetail.started_at)
                }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>完成时间</q-item-label>
                <q-item-label>{{
                  formatOrganizationDateTime(batchDetail.completed_at)
                }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>成功 / 失败 / 跳过 / 总数</q-item-label>
                <q-item-label>
                  {{ batchDetail.success_count }} / {{ batchDetail.failed_count }} /
                  {{ batchDetail.skipped_count }} / {{ batchDetail.total_count }}
                </q-item-label>
              </q-item-section>
            </q-item>
          </q-list>
          <q-separator />
          <q-card-section class="text-subtitle1 text-weight-medium">对象处理记录</q-card-section>
          <q-table
            flat
            :rows="batchRecords"
            :columns="recordColumns"
            row-key="id"
            :loading="recordLoading"
            :pagination="{ rowsPerPage: 0 }"
            hide-bottom
          >
            <template #body-cell-action="props">
              <q-td :props="props">{{ dictLabel('org_sync_action', props.row.action) }}</q-td>
            </template>
            <template #body-cell-status="props">
              <q-td :props="props">{{
                dictLabel('org_sync_record_status', props.row.status)
              }}</q-td>
            </template>
            <template #no-data>
              <div class="full-width text-center text-grey-7 q-pa-lg">暂无对象处理记录</div>
            </template>
          </q-table>
        </template>
        <q-card-actions align="right">
          <q-btn v-close-popup flat color="primary" label="关闭" />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <organization-record-detail-dialog
      v-model="showErrorDialog"
      title="同步批次错误"
      :subtitle="currentBatch?.batch_no || ''"
      :items="errorItems"
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
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import {
  getSyncBatchDetail,
  getSyncBatchError,
  querySyncBatches,
  querySyncRecords,
  type SyncBatchDetail,
  type SyncBatchListItem,
  type SyncBatchQueryRequest,
  type SyncRecordListItem,
} from 'src/api/services/org'
import type { MenuButton } from 'src/api/services/sys-menu'
import { usePageButtons } from 'src/composables/page-buttons'
import OrganizationRecordDetailDialog from 'src/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type { OrganizationDetailItem } from 'src/pages/organization/components/OrganizationRecordDetailDialog.vue'
import {
  createOrganizationField,
  createOrganizationQuery,
  formatOrganizationDateTime,
  organizationStatusColor,
} from 'src/pages/organization/organization-list-page'
import { useDictStore } from 'src/stores/dict'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const dictStore = useDictStore()
const { line_buttons, top_buttons } = usePageButtons('organization_sync_batch')

const rows = ref<SyncBatchListItem[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const query = ref<SyncBatchQueryRequest>(createOrganizationQuery('org_sync_batch'))
const tempAdvancedQuery = ref<SyncBatchQueryRequest>(cloneDeep(query.value))
const showAdvancedQuery = ref(false)

const showDetailDialog = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const batchDetail = ref<SyncBatchDetail | null>(null)
const batchRecords = ref<SyncRecordListItem[]>([])
const recordLoading = ref(false)
const currentBatch = ref<SyncBatchListItem | null>(null)

const showErrorDialog = ref(false)
const errorLoading = ref(false)
const errorLoadError = ref('')
const errorSummary = ref('')

const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
const syncTypeOptions = computed(() => dictStore.getDictOptions('org_sync_type'))
const syncStatusOptions = computed(() => dictStore.getDictOptions('org_sync_record_status'))
const errorItems = computed<OrganizationDetailItem[]>(() => [
  { label: '错误摘要', value: errorSummary.value },
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

const recordColumns: QTableProps['columns'] = [
  { name: 'object_type', field: 'object_type', label: '对象类型', align: 'left' },
  { name: 'source_code', field: 'source_code', label: '源对象编码', align: 'left' },
  { name: 'action', field: 'action', label: '动作', align: 'center' },
  { name: 'status', field: 'status', label: '状态', align: 'center' },
  { name: 'error_code', field: 'error_code', label: '错误码', align: 'left' },
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

const visibleRowButtons = (row: SyncBatchListItem) =>
  line_buttons.value.filter(
    (button) =>
      button.event_action === 'detail' ||
      (button.event_action === 'view_error' && row.has_error),
  )

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
  currentBatch.value = row
  batchDetail.value = null
  batchRecords.value = []
  detailError.value = ''
  detailLoading.value = true
  recordLoading.value = true
  showDetailDialog.value = true
  try {
    const [detail, records] = await Promise.all([
      getSyncBatchDetail(row.id),
      querySyncRecords({
        ...createOrganizationQuery('org_sync_record'),
        batch_id: row.id,
        order: { field: 'gmt_create', is_asc: true },
        num: 100,
      }),
    ])
    batchDetail.value = detail
    batchRecords.value = records.items
  } catch {
    detailError.value = '同步批次详情加载失败'
  } finally {
    detailLoading.value = false
    recordLoading.value = false
  }
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
