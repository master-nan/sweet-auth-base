<template>
  <detail-page-shell
    :title="t('ui.implementationDetails')"
    :subtitle="detail?.execution_no || '-'"
    icon="play_circle"
    :loading="loading"
    :error="loadError"
    retryable
    @retry="loadDetail"
  >
    <template #actions>
      <status-chip
        v-if="detail"
        :color="statusMeta[detail.status]?.color || 'grey'"
        :label="statusMeta[detail.status]?.label || detail.status"
      />
      <q-btn
        flat
        color="primary"
        icon="arrow_back"
        :label="t('ui.backToList')"
        @click="router.back()"
      />
      <q-btn
        outline
        color="primary"
        icon="refresh"
        :label="t('ui.refresh')"
        :loading="loading"
        @click="loadDetail"
      />
    </template>

    <template v-if="detail">
      <section class="detail-page-section">
        <div class="detail-page-section__head">
          <h3>{{ t('ui.basicInfo') }}</h3>
        </div>
        <detail-field-grid :items="basicItems" variant="card" />
      </section>
      <section class="detail-page-section">
        <div class="detail-page-section__head">
          <h3>{{ t('ui.autoRetrySummary') }}</h3>
        </div>
        <detail-field-grid :items="retryItems" variant="card" />
      </section>
      <section class="detail-page-section">
        <div class="detail-page-section__head">
          <h3>{{ t('ui.enterASnapshotSummary') }}</h3>
        </div>
        <detail-field-grid :items="inputItems" variant="card" />
        <div class="execution-detail__note">
          {{ t('ui.thePageDisplaysOnlyTheNumberOfParametersTheSize') }}
        </div>
      </section>
      <section class="detail-page-section">
        <div class="detail-page-section__head">
          <h3>{{ t('ui.summaryOfStatusAndResults') }}</h3>
        </div>
        <detail-field-grid :items="resultItems" variant="card" />
        <div class="execution-detail__note">
          {{ t('ui.originalResponderIsNotSavedAsExecutionDetailsSecuritySummaries') }}
        </div>
      </section>
      <section v-if="detail.sync_business" class="detail-page-section">
        <div class="detail-page-section__head">
          <h3>{{ t('ui.synchronizeBusinessResults') }}</h3>
        </div>
        <detail-field-grid :items="syncItems" variant="card" />
      </section>
      <section class="detail-page-section">
        <div class="detail-page-section__head">
          <h3>{{ t('ui.attempts') }}</h3>
        </div>
        <div v-if="!canQueryLogs" class="text-body2 text-grey-7 q-py-md">
          {{ t('ui.noCallLogViewingPermission') }}
        </div>
        <q-table
          v-else
          flat
          bordered
          dense
          :loading="attemptsLoading"
          :rows="attempts"
          :columns="attemptColumns"
          row-key="id"
          ><template #body-cell-attempt_no="props"
            ><q-td :props="props"
              ><q-btn
                v-if="canViewLogDetail"
                flat
                dense
                color="primary"
                :label="`#${props.row.attempt_no}`"
                @click="openLog(props.row.id)"
              />
              <span v-else>#{{ props.row.attempt_no }}</span></q-td
            ></template
          ><template #body-cell-status="props"
            ><q-td :props="props"
              ><status-chip
                :color="logStatusMeta[props.row.status]?.color || 'grey'"
                :outline="false"
                :label="
                  logStatusMeta[props.row.status]?.label || props.row.status
                " /></q-td></template
        ></q-table>
      </section>
    </template>
  </detail-page-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'integration_execution_detail_page' })
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailFieldGrid from '@/components/Detail/DetailFieldGrid.vue'
import DetailPageShell from '@/components/Detail/DetailPageShell.vue'
import type { DetailFieldItem } from '@/components/Detail/types'
import StatusChip from '@/components/Display/StatusChip.vue'
import {
  useIntegrationApi,
  type IntegrationExecutionDetail,
  type IntegrationLogListItem,
} from '@/api/services/integration'
import { usePageButtons } from '@/composables/page-buttons'
import type { QTableProps } from 'quasar'
import { formatRetryReason, formatRuntimeDateTime } from '@/pages/integration/runtime-display'

const { t } = useI18n({ useScope: 'global' })

const route = useRoute()
const router = useRouter()
const api = useIntegrationApi()
const loading = ref(false)
const loadError = ref('')
const detail = ref<IntegrationExecutionDetail | null>(null)
const attempts = ref<IntegrationLogListItem[]>([])
const attemptsLoading = ref(false)
const { hasGrantedCapability } = usePageButtons('integration_execution')
const canViewExecutionDetail = computed(() => hasGrantedCapability('integration_execution_detail'))
const canQueryLogs = computed(() => hasGrantedCapability('integration_log_query'))
const canViewLogDetail = computed(() => hasGrantedCapability('integration_log_detail'))
const statusMeta: Record<string, { label: string; color: string }> = {
  created: {
    get label() {
      return t('ui.pending')
    },
    color: 'grey-7',
  },
  running: {
    get label() {
      return t('ui.executionRunningStatus')
    },
    color: 'primary',
  },
  retry_waiting: {
    get label() {
      return t('ui.waitingToRetry')
    },
    color: 'warning',
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
  cancelled: {
    get label() {
      return t('ui.cancelled')
    },
    color: 'grey-6',
  },
}
const logStatusMeta: Record<string, { label: string; color: string }> = {
  running: {
    get label() {
      return t('ui.executionRunningStatus')
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
  cancelled: {
    get label() {
      return t('ui.cancelled')
    },
    color: 'grey-6',
  },
}
const formatDate = formatRuntimeDateTime
const basicItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value) return []
  return [
    {
      get label() {
        return t('ui.externalSystemLabel')
      },
      value: `${detail.value.external_system.name}（${detail.value.external_system.system_code}）`,
    },
    {
      get label() {
        return t('ui.api')
      },
      value: `${detail.value.interface.name} · v${detail.value.interface.version}`,
    },
    {
      get label() {
        return t('ui.triggerSource')
      },
      value: detail.value.trigger_source,
    },
    {
      get label() {
        return t('ui.executionId')
      },
      value: detail.value.execution_no,
    },
  ]
})
const retryItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value) return []
  return [
    {
      get label() {
        return t('ui.retryPolicy')
      },
      value: detail.value.retry_policy
        ? `${detail.value.retry_policy.policy_code} · v${detail.value.retry_policy.policy_version}`
        : t('ui.notConfigured'),
    },
    {
      label: 'Attempt',
      value: `${detail.value.current_attempt} / ${detail.value.max_attempts}`,
    },
    {
      get label() {
        return t('ui.remaining')
      },
      value: detail.value.attempts_remaining,
    },
    {
      get label() {
        return t('ui.nextRetry')
      },
      value: formatDate(detail.value.next_run_at),
    },
    {
      get label() {
        return t('ui.retryReason')
      },
      value: formatRetryReason(detail.value.retry_reason_code),
      fullWidth: true,
    },
  ]
})
const inputItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value) return []
  return [
    {
      get label() {
        return t('ui.quickPageVersion')
      },
      value: `v${detail.value.input_summary.snapshot_version}`,
    },
    {
      get label() {
        return t('ui.quickSize')
      },
      value: t('ui.bytes', { value1: detail.value.input_summary.size_bytes }),
    },
    {
      get label() {
        return t('ui.pathParameter')
      },
      value: detail.value.input_summary.path_count,
    },
    {
      get label() {
        return t('ui.queryParameter')
      },
      value: detail.value.input_summary.query_count,
    },
    {
      get label() {
        return t('ui.headerParameter')
      },
      value: detail.value.input_summary.header_count,
    },
    {
      label: 'JSON Body',
      value: detail.value.input_summary.has_body ? t('ui.hasJsonBodyYes') : t('ui.none'),
    },
    {
      get label() {
        return t('ui.enterHash')
      },
      value: detail.value.input_hash,
      fullWidth: true,
    },
  ]
})
const resultItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value) return []
  return [
    {
      get label() {
        return t('ui.currentAttempt')
      },
      value: detail.value.current_attempt,
    },
    {
      get label() {
        return t('ui.startTime')
      },
      value: formatDate(detail.value.started_at),
    },
    {
      get label() {
        return t('ui.completedAt')
      },
      value: formatDate(detail.value.completed_at),
    },
    {
      get label() {
        return t('ui.duration')
      },
      value: t('ui.millisecondsValue', { value1: detail.value.duration_ms }),
    },
    {
      get label() {
        return t('ui.httpStatus')
      },
      value: detail.value.result_http_status || '-',
    },
    {
      get label() {
        return t('ui.responseSize')
      },
      value: t('ui.bytes', { value1: detail.value.result_size_bytes }),
    },
    {
      get label() {
        return t('ui.errorCategory')
      },
      value: detail.value.error_category || '-',
    },
    {
      get label() {
        return t('ui.leaseholders')
      },
      value: detail.value.lease_owner_summary || '-',
    },
    {
      get label() {
        return t('ui.leaseExpiration')
      },
      value: formatDate(detail.value.lease_expires_at),
    },
    {
      get label() {
        return t('ui.resultsHash')
      },
      value: detail.value.result_hash || '-',
      fullWidth: true,
    },
    {
      get label() {
        return t('ui.summaryOfSecurityResults')
      },
      value: detail.value.result_summary || '-',
      fullWidth: true,
    },
  ]
})
const syncItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value?.sync_business) return []
  return [
    {
      get label() {
        return t('ui.status')
      },
      value: detail.value.sync_business.status,
    },
    {
      get label() {
        return t('ui.successFailed')
      },
      value: `${detail.value.sync_business.success_count} / ${detail.value.sync_business.failed_count}`,
    },
    {
      get label() {
        return t('ui.reason')
      },
      value: detail.value.sync_business.reason_code || '-',
    },
    {
      get label() {
        return t('ui.businessReference')
      },
      value: detail.value.sync_business.reference || '-',
    },
  ]
})
const attemptColumns: QTableProps['columns'] = [
  { name: 'attempt_no', label: 'Attempt', field: 'attempt_no', align: 'left' },
  {
    name: 'status',
    get label() {
      return t('ui.status')
    },
    field: 'status',
    align: 'center',
  },
  {
    name: 'http_status',
    get label() {
      return t('ui.httpStatusLabel')
    },
    field: 'http_status',
    align: 'center',
  },
  {
    name: 'duration_ms',
    get label() {
      return t('ui.durationMs')
    },
    field: 'duration_ms',
    align: 'right',
  },
  {
    name: 'error_category',
    get label() {
      return t('ui.errorCategory')
    },
    field: 'error_category',
    align: 'left',
  },
  {
    name: 'result_certainty',
    get label() {
      return t('ui.resultCertainty')
    },
    field: 'result_certainty',
    align: 'left',
  },
]
const openLog = (logId: number) => {
  if (detail.value)
    void router.push({
      name: 'integration_log',
      query: { execution_id: String(detail.value.id), log_id: String(logId) },
    })
}
const loadDetail = async () => {
  if (!canViewExecutionDetail.value) return
  const id = Number(route.params.id)
  if (id > 0) {
    loading.value = true
    loadError.value = ''
    try {
      const response = await api.getExecution(id)
      detail.value = response.data || null
      attempts.value = []
      if (detail.value && canQueryLogs.value) {
        attemptsLoading.value = true
        const logs = await api.queryLogs({
          page: 1,
          num: 500,
          order: { field: 'attempt_no', is_asc: true },
          quick_query: { keyword: '' },
          expressions: [],
          execution_id: detail.value.id,
        })
        attempts.value = logs.data || []
      }
    } catch (error) {
      detail.value = null
      attempts.value = []
      loadError.value =
        error instanceof Error && error.message
          ? error.message
          : t('ui.failedToLoadExecutionDetails')
    } finally {
      attemptsLoading.value = false
      loading.value = false
    }
  }
}
onMounted(async () => {
  await loadDetail()
})
</script>

<style scoped lang="scss">
.execution-detail__note {
  margin-top: 18px;
  padding: 10px 12px;
  border-left: 3px solid var(--q-primary);
  background: var(--app-primary-soft);
  color: var(--app-text-muted);
  font-size: 13px;
  line-height: 1.6;
}
</style>
