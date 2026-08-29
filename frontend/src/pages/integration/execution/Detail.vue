<template>
  <base-content scrollable class="q-pa-sm record-detail-page">
    <div class="record-detail">
      <header class="record-detail-header">
        <div class="record-detail-title-wrap">
          <q-icon name="play_circle" class="record-detail-icon" />
          <div>
            <div class="record-detail-title">执行详情</div>
            <div class="record-detail-subtitle">{{ detail?.execution_no || '-' }}</div>
          </div>
        </div>
        <q-space />
        <div class="record-detail-actions">
          <status-chip
            v-if="detail"
            :color="statusMeta[detail.status]?.color || 'grey'"
            :label="statusMeta[detail.status]?.label || detail.status"
          />
          <q-btn flat color="primary" icon="arrow_back" label="返回列表" @click="router.back()" />
          <q-btn
            outline
            color="primary"
            icon="refresh"
            label="刷新"
            :loading="loading"
            @click="loadDetail"
          />
        </div>
      </header>

      <q-inner-loading :showing="loading">
        <q-spinner color="primary" size="42px" />
      </q-inner-loading>

      <q-banner v-if="loadError" rounded class="record-detail-error">
        <template #avatar>
          <q-icon name="error_outline" color="negative" />
        </template>
        {{ loadError }}
        <q-btn
          flat
          color="negative"
          label="重新加载"
          @click="loadDetail"
        />
      </q-banner>

      <template v-if="detail">
        <section class="record-detail-panel">
          <div class="record-detail-panel-head"><h3>基础信息</h3></div>
          <detail-field-grid :items="basicItems" variant="card" />
        </section>
        <section class="record-detail-panel">
          <div class="record-detail-panel-head"><h3>自动重试摘要</h3></div>
          <detail-field-grid :items="retryItems" variant="card" />
        </section>
        <section class="record-detail-panel">
          <div class="record-detail-panel-head"><h3>输入快照摘要</h3></div>
          <detail-field-grid :items="inputItems" variant="card" />
          <div class="execution-detail__note">
            页面只展示参数数量、快照大小和
            Hash。真实请求值可能包含身份标识或凭证，不会返回管理页面。
          </div>
        </section>
        <section class="record-detail-panel">
          <div class="record-detail-panel-head"><h3>状态与结果摘要</h3></div>
          <detail-field-grid :items="resultItems" variant="card" />
          <div class="execution-detail__note">
            原始响应体不作为执行详情保存；排查时使用安全摘要、Hash、HTTP 状态和下方 Attempt 记录。
          </div>
        </section>
        <section v-if="detail.sync_business" class="record-detail-panel">
          <div class="record-detail-panel-head"><h3>同步业务结果</h3></div>
          <detail-field-grid :items="syncItems" variant="card" />
        </section>
        <section class="record-detail-panel">
          <div class="record-detail-panel-head"><h3>Attempt 记录</h3></div>
          <div v-if="!canQueryLogs" class="text-body2 text-grey-7 q-py-md">无调用日志查看权限</div>
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
    </div>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_execution_detail_page' })
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import DetailFieldGrid from 'src/components/Detail/DetailFieldGrid.vue'
import type { DetailFieldItem } from 'src/components/Detail/types'
import StatusChip from 'src/components/Display/StatusChip.vue'
import {
  useIntegrationApi,
  type IntegrationExecutionDetail,
  type IntegrationLogListItem,
} from 'src/api/services/integration'
import { usePageButtons } from 'src/composables/page-buttons'
import type { QTableProps } from 'quasar'
import { formatRetryReason, formatRuntimeDateTime } from 'src/pages/integration/runtime-display'

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
  created: { label: '待执行', color: 'grey-7' },
  running: { label: '执行中', color: 'primary' },
  retry_waiting: { label: '等待重试', color: 'warning' },
  succeeded: { label: '成功', color: 'positive' },
  failed: { label: '失败', color: 'negative' },
  cancelled: { label: '已取消', color: 'grey-6' },
}
const logStatusMeta: Record<string, { label: string; color: string }> = {
  running: { label: '执行中', color: 'primary' },
  succeeded: { label: '成功', color: 'positive' },
  failed: { label: '失败', color: 'negative' },
  cancelled: { label: '已取消', color: 'grey-6' },
}
const formatDate = formatRuntimeDateTime
const basicItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value) return []
  return [
    {
      label: '外部系统',
      value: `${detail.value.external_system.name}（${detail.value.external_system.system_code}）`,
    },
    {
      label: '接口',
      value: `${detail.value.interface.name} · v${detail.value.interface.version}`,
    },
    { label: '触发来源', value: detail.value.trigger_source },
    { label: '执行编号', value: detail.value.execution_no },
  ]
})
const retryItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value) return []
  return [
    {
      label: '重试策略',
      value: detail.value.retry_policy
        ? `${detail.value.retry_policy.policy_code} · v${detail.value.retry_policy.policy_version}`
        : '未配置',
    },
    {
      label: 'Attempt',
      value: `${detail.value.current_attempt} / ${detail.value.max_attempts}`,
    },
    { label: '剩余次数', value: detail.value.attempts_remaining },
    { label: '下次重试', value: formatDate(detail.value.next_run_at) },
    {
      label: '重试原因',
      value: formatRetryReason(detail.value.retry_reason_code),
      fullWidth: true,
    },
  ]
})
const inputItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value) return []
  return [
    { label: '快照版本', value: `v${detail.value.input_summary.snapshot_version}` },
    { label: '快照大小', value: `${detail.value.input_summary.size_bytes} 字节` },
    { label: 'Path 参数', value: detail.value.input_summary.path_count },
    { label: 'Query 参数', value: detail.value.input_summary.query_count },
    { label: 'Header 参数', value: detail.value.input_summary.header_count },
    { label: 'JSON Body', value: detail.value.input_summary.has_body ? '有' : '无' },
    { label: '输入 Hash', value: detail.value.input_hash, fullWidth: true },
  ]
})
const resultItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value) return []
  return [
    { label: '当前 Attempt', value: detail.value.current_attempt },
    { label: '开始时间', value: formatDate(detail.value.started_at) },
    { label: '完成时间', value: formatDate(detail.value.completed_at) },
    { label: '耗时', value: `${detail.value.duration_ms} 毫秒` },
    { label: 'HTTP 状态', value: detail.value.result_http_status || '-' },
    { label: '响应大小', value: `${detail.value.result_size_bytes} 字节` },
    { label: '错误分类', value: detail.value.error_category || '-' },
    { label: '租约持有者', value: detail.value.lease_owner_summary || '-' },
    { label: '租约到期', value: formatDate(detail.value.lease_expires_at) },
    { label: '结果 Hash', value: detail.value.result_hash || '-', fullWidth: true },
    { label: '安全结果摘要', value: detail.value.result_summary || '-', fullWidth: true },
  ]
})
const syncItems = computed<DetailFieldItem[]>(() => {
  if (!detail.value?.sync_business) return []
  return [
    { label: '状态', value: detail.value.sync_business.status },
    {
      label: '成功 / 失败',
      value: `${detail.value.sync_business.success_count} / ${detail.value.sync_business.failed_count}`,
    },
    { label: '原因', value: detail.value.sync_business.reason_code || '-' },
    { label: '业务引用', value: detail.value.sync_business.reference || '-' },
  ]
})
const attemptColumns: QTableProps['columns'] = [
  { name: 'attempt_no', label: 'Attempt', field: 'attempt_no', align: 'left' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
  { name: 'http_status', label: 'HTTP状态', field: 'http_status', align: 'center' },
  { name: 'duration_ms', label: '耗时（毫秒）', field: 'duration_ms', align: 'right' },
  { name: 'error_category', label: '错误分类', field: 'error_category', align: 'left' },
  { name: 'result_certainty', label: '结果确定性', field: 'result_certainty', align: 'left' },
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
      loadError.value = error instanceof Error && error.message ? error.message : '执行详情加载失败'
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
