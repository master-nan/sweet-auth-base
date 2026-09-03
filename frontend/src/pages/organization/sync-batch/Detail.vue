<template>
  <base-content scrollable class="q-pa-sm">
    <organization-record-detail-content
      mode="page"
      :title="title"
      :subtitle="batchDetail ? dictLabel('org_sync_type', batchDetail.sync_type) : ''"
      :sections="detailSections"
      icon="sync"
      :status-label="batchDetail ? dictLabel('org_sync_record_status', batchDetail.status) : ''"
      :status-color="batchDetail ? organizationStatusColor(batchDetail.status) : 'positive'"
      :loading="loading"
      :error="loadError"
      :top-buttons="record_detail_top_buttons"
      :bottom-buttons="record_detail_bottom_buttons"
      :record-context="batchDetail"
      @close="emit('close')"
      @refresh="loadDetail"
      @button-click="handleDetailAction"
    >
      <template #section="{ sectionKey }">
        <q-table
          v-if="sectionKey === 'records'"
          class="organization-sync-record-table"
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
          <template #body-cell-object_type="slotProps">
            <q-td :props="slotProps">
              {{ organizationSyncObjectLabel(slotProps.row.object_type) }}
            </q-td>
          </template>
          <template #body-cell-status="slotProps">
            <q-td :props="slotProps">
              {{ dictLabel('org_sync_record_status', slotProps.row.status) }}
            </q-td>
          </template>
          <template #no-data>
            <div class="full-width column flex-center q-gutter-sm q-pa-lg text-grey-7">
              <q-icon :name="recordsLoadError ? 'cloud_off' : 'inbox'" color="grey-5" size="48px" />
              <span>
                {{
                  recordsLoadError ||
                  (canQueryRecords
                    ? t('ui.noObjectProcessingRecords')
                    : t('ui.noBusinessSyncRecordView'))
                }}
              </span>
            </div>
          </template>
        </q-table>
      </template>
    </organization-record-detail-content>

    <organization-record-detail-dialog
      v-model="showErrorDialog"
      :title="t('ui.syncBatchErrors')"
      :subtitle="batchDetail?.batch_no || ''"
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

import { computed, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import {
  getSyncBatchDetail,
  getSyncBatchError,
  querySyncRecords,
  type SyncBatchDetail,
  type SyncRecordListItem,
} from '@/api/services/org'
import type { MenuButton } from '@/api/services/sys-menu'
import { usePageButtons } from '@/composables/page-buttons'
import OrganizationRecordDetailContent from '@/pages/organization/components/OrganizationRecordDetailContent.vue'
import OrganizationRecordDetailDialog from '@/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type {
  OrganizationDetailItem,
  OrganizationDetailSection,
} from '@/pages/organization/components/organization-record-detail'
import {
  createOrganizationQuery,
  formatOrganizationDateTime,
  formatOrganizationValue,
  organizationSyncObjectLabel,
  organizationStatusColor,
} from '@/pages/organization/organization-list-page'
import { useDictStore } from '@/stores/dict'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{
  recordId: number
}>()

const emit = defineEmits<{
  close: []
  'title-change': [title: string]
}>()

const dictStore = useDictStore()
const { record_detail_top_buttons, record_detail_bottom_buttons, hasGrantedCapability } =
  usePageButtons('organization_sync_batch')

const loading = ref(false)
const loadError = ref('')
const batchDetail = ref<SyncBatchDetail | null>(null)
const batchRecords = ref<SyncRecordListItem[]>([])
const recordsLoadError = ref('')
const showErrorDialog = ref(false)
const errorLoading = ref(false)
const errorLoadError = ref('')
const errorSummary = ref('')
const canQueryRecords = computed(() => hasGrantedCapability('organization_sync_error_query'))

const title = computed(() =>
  batchDetail.value
    ? t('ui.syncBatchDetailsTitle', { value1: batchDetail.value.batch_no })
    : t('ui.syncBatchDetails'),
)

const errorItems = computed<OrganizationDetailItem[]>(() => [
  {
    get label() {
      return t('ui.errorSummary')
    },
    value: errorSummary.value,
    fullWidth: true,
  },
])

const detailSections = computed<OrganizationDetailSection[]>(() => {
  const detail = batchDetail.value
  if (!detail) return []
  return [
    {
      key: 'basic',
      get label() {
        return t('ui.basicInfo')
      },
      get caption() {
        return t('ui.bScopeOfTheInstalmentAndResultsOfTheImplementation')
      },
      items: [
        {
          get label() {
            return t('ui.batchNumber')
          },
          value: detail.batch_no,
        },
        {
          get label() {
            return t('ui.syncType')
          },
          value: dictLabel('org_sync_type', detail.sync_type),
        },
        {
          get label() {
            return t('ui.objectScope')
          },
          value: organizationSyncObjectLabel(detail.object_scope),
        },
        {
          get label() {
            return t('ui.status')
          },
          value: dictLabel('org_sync_record_status', detail.status),
          chip: true,
          color: organizationStatusColor(detail.status),
        },
        {
          get label() {
            return t('ui.integrationExecutionId')
          },
          value: detail.execution_id ?? null,
        },
        {
          get label() {
            return t('ui.startTime')
          },
          value: formatOrganizationDateTime(detail.started_at),
        },
        {
          get label() {
            return t('ui.completedAt')
          },
          value: formatOrganizationDateTime(detail.completed_at),
        },
        {
          get label() {
            return t('ui.successFailedSkipTotal')
          },
          value: `${detail.success_count} / ${detail.failed_count} / ${detail.skipped_count} / ${detail.total_count}`,
        },
      ],
    },
    {
      key: 'records',
      get label() {
        return t('ui.objectProcessingRecords')
      },
      get caption() {
        return t('ui.thisBatchHandlesTheDetails')
      },
      count: batchRecords.value.length,
      items: [],
    },
  ]
})

const recordColumns: QTableProps['columns'] = [
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
      return t('ui.status')
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
]

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || formatOrganizationValue(value)

const loadDetail = async () => {
  if (!props.recordId) {
    loadError.value = t('ui.synchronisingBatchDetailsMissingId')
    return
  }

  loading.value = true
  loadError.value = ''
  recordsLoadError.value = ''
  batchDetail.value = null
  batchRecords.value = []
  try {
    await dictStore.loadDicts(['org_sync_type', 'org_sync_action', 'org_sync_record_status'])
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
        recordsLoadError.value = t('ui.objectProcessingRecordLoadedFailed')
      }
    }
  } catch {
    loadError.value = t('ui.synchronisingBatchDetailsLoadedFailed')
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
    errorLoadError.value = t('ui.failedToLoadSyncBatchErrors')
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

watch(title, (value) => emit('title-change', value), { immediate: true })
</script>

<style scoped>
.organization-sync-record-table :deep(.q-table__middle) {
  overflow: visible;
  overscroll-behavior: auto;
}
</style>
