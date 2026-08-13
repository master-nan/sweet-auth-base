<template>
  <base-content scrollable class="q-pa-sm">
    <organization-record-detail-content
      mode="page"
      :title="title"
      :subtitle="batchDetail ? dictLabel('org_sync_type', batchDetail.sync_type) : ''"
      :sections="detailSections"
      icon="sync"
      :status-label="
        batchDetail ? dictLabel('org_sync_record_status', batchDetail.status) : ''
      "
      :status-color="batchDetail ? organizationStatusColor(batchDetail.status) : 'positive'"
      :loading="loading"
      :error="loadError"
      :top-buttons="record_detail_top_buttons"
      :bottom-buttons="record_detail_bottom_buttons"
      :record-context="batchDetail"
      @close="emit('close')"
      @button-click="handleDetailAction"
    >
      <template #section="{ sectionKey }">
        <q-table
          v-if="sectionKey === 'records'"
          flat
          bordered
          separator="cell"
          :rows="batchRecords"
          :columns="recordColumns"
          row-key="id"
          :loading="loading"
          :pagination="{ rowsPerPage: 0 }"
          hide-bottom
        >
          <template #body-cell-action="slotProps">
            <q-td :props="slotProps">
              {{ dictLabel('org_sync_action', slotProps.row.action) }}
            </q-td>
          </template>
          <template #body-cell-status="slotProps">
            <q-td :props="slotProps">
              {{ dictLabel('org_sync_record_status', slotProps.row.status) }}
            </q-td>
          </template>
          <template #no-data>
            <div class="full-width text-center text-grey-7 q-pa-lg">
              {{ recordsLoadError || (canQueryRecords ? '暂无对象处理记录' : '无业务同步记录查看权限') }}
            </div>
          </template>
        </q-table>
      </template>
    </organization-record-detail-content>

    <organization-record-detail-dialog
      v-model="showErrorDialog"
      title="同步批次错误"
      :subtitle="batchDetail?.batch_no || ''"
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
import { computed, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import {
  getSyncBatchDetail,
  getSyncBatchError,
  querySyncRecords,
  type SyncBatchDetail,
  type SyncRecordListItem,
} from 'src/api/services/org'
import type { MenuButton } from 'src/api/services/sys-menu'
import { usePageButtons } from 'src/composables/page-buttons'
import OrganizationRecordDetailContent from 'src/pages/organization/components/OrganizationRecordDetailContent.vue'
import OrganizationRecordDetailDialog from 'src/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type {
  OrganizationDetailItem,
  OrganizationDetailSection,
} from 'src/pages/organization/components/organization-record-detail'
import {
  createOrganizationQuery,
  formatOrganizationDateTime,
  organizationStatusColor,
} from 'src/pages/organization/organization-list-page'
import { useDictStore } from 'src/stores/dict'
import { useUserStore } from 'src/stores/user'

const props = defineProps<{
  recordId: number
}>()

const emit = defineEmits<{
  close: []
  'title-change': [title: string]
}>()

const dictStore = useDictStore()
const userStore = useUserStore()
const { record_detail_top_buttons, record_detail_bottom_buttons } = usePageButtons(
  'organization_sync_batch',
)

const loading = ref(false)
const loadError = ref('')
const batchDetail = ref<SyncBatchDetail | null>(null)
const batchRecords = ref<SyncRecordListItem[]>([])
const recordsLoadError = ref('')
const showErrorDialog = ref(false)
const errorLoading = ref(false)
const errorLoadError = ref('')
const errorSummary = ref('')
const canQueryRecords = computed(() => userStore.buttons.includes('organization_sync_error_query'))

const title = computed(() =>
  batchDetail.value
    ? `同步批次详情：${batchDetail.value.batch_no}`
    : '同步批次详情',
)

const errorItems = computed<OrganizationDetailItem[]>(() => [
  { label: '错误摘要', value: errorSummary.value, fullWidth: true },
])

const detailSections = computed<OrganizationDetailSection[]>(() => {
  const detail = batchDetail.value
  if (!detail) return []
  return [
    {
      key: 'basic',
      label: '基础信息',
      caption: '批次范围与执行结果',
      items: [
        { label: '批次号', value: detail.batch_no },
        { label: '同步类型', value: dictLabel('org_sync_type', detail.sync_type) },
        { label: '对象范围', value: detail.object_scope },
        {
          label: '状态',
          value: dictLabel('org_sync_record_status', detail.status),
          chip: true,
          color: organizationStatusColor(detail.status),
        },
        { label: '集成执行ID', value: detail.execution_id ?? null },
        { label: '开始时间', value: formatOrganizationDateTime(detail.started_at) },
        { label: '完成时间', value: formatOrganizationDateTime(detail.completed_at) },
        {
          label: '成功 / 失败 / 跳过 / 总数',
          value: `${detail.success_count} / ${detail.failed_count} / ${detail.skipped_count} / ${detail.total_count}`,
        },
      ],
    },
    {
      key: 'records',
      label: '对象处理记录',
      caption: '本批次对象处理明细',
      count: batchRecords.value.length,
      items: [],
    },
  ]
})

const recordColumns: QTableProps['columns'] = [
  { name: 'object_type', field: 'object_type', label: '对象类型', align: 'left' },
  { name: 'source_summary', field: 'source_summary', label: '来源摘要', align: 'left' },
  { name: 'action', field: 'action', label: '动作', align: 'center' },
  { name: 'status', field: 'status', label: '状态', align: 'center' },
  { name: 'error_code', field: 'error_code', label: '错误码', align: 'left' },
]

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || String(value || '-')

const loadDetail = async () => {
  if (!props.recordId) {
    loadError.value = '同步批次详情缺少记录ID'
    return
  }

  loading.value = true
  loadError.value = ''
  recordsLoadError.value = ''
  batchDetail.value = null
  batchRecords.value = []
  try {
    await dictStore.loadDicts([
      'org_sync_type',
      'org_sync_action',
      'org_sync_record_status',
    ])
    batchDetail.value = await getSyncBatchDetail(props.recordId)
    if (canQueryRecords.value) {
      try {
        const records = await querySyncRecords({
        ...createOrganizationQuery('org_sync_record'),
        batch_id: props.recordId,
        order: { field: 'gmt_create', is_asc: true },
        num: 100,
        })
        batchRecords.value = records.items
      } catch {
        recordsLoadError.value = '对象处理记录加载失败'
      }
    }
  } catch {
    loadError.value = '同步批次详情加载失败'
  } finally {
    loading.value = false
  }
}

const openError = async () => {
  if (!batchDetail.value) return
  errorSummary.value = ''
  errorLoadError.value = ''
  errorLoading.value = true
  showErrorDialog.value = true
  try {
    const result = await getSyncBatchError(batchDetail.value.id)
    errorSummary.value = result.error_summary
  } catch {
    errorLoadError.value = '同步批次错误加载失败'
  } finally {
    errorLoading.value = false
  }
}

const handleDetailAction = (button: MenuButton) => {
  if (button.event_action === 'view_error') void openError()
}

watch(
  () => props.recordId,
  () => void loadDetail(),
  { immediate: true },
)

watch(
  title,
  (value) => emit('title-change', value),
  { immediate: true },
)
</script>
