<template>
  <base-content class="q-pa-md">
    <div class="row items-center q-mb-md q-gutter-sm">
      <q-btn flat round icon="arrow_back" color="primary" @click="router.back()" />
      <div>
        <div class="text-h5">执行详情</div>
        <div class="text-caption text-grey-7">{{ detail?.execution_no || '-' }}</div>
      </div>
      <q-space />
      <q-chip
        v-if="detail"
        dense
        square
        outline
        :color="statusMeta[detail.status]?.color || 'grey'"
        :label="statusMeta[detail.status]?.label || detail.status"
      />
    </div>
    <q-inner-loading :showing="loading" />
    <template v-if="detail">
      <q-card flat bordered class="q-mb-md"
        ><q-card-section
          ><div class="text-subtitle1 text-weight-bold q-mb-md">基础信息</div>
          <div class="row q-col-gutter-lg">
            <div class="col-12 col-md-4">
              <div class="text-caption text-grey-7">外部系统</div>
              <div>
                {{ detail.external_system.name }}（{{ detail.external_system.system_code }}）
              </div>
            </div>
            <div class="col-12 col-md-4">
              <div class="text-caption text-grey-7">接口</div>
              <div>{{ detail.interface.name }} · v{{ detail.interface.version }}</div>
            </div>
            <div class="col-12 col-md-4">
              <div class="text-caption text-grey-7">触发来源</div>
              <div>{{ detail.trigger_source }}</div>
            </div>
          </div></q-card-section
        ></q-card
      >
      <q-card flat bordered class="q-mb-md">
        <q-card-section>
          <div class="text-subtitle1 text-weight-bold q-mb-md">自动重试摘要</div>
          <div class="row q-col-gutter-lg">
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">重试策略</div>
              <div>
                {{ detail.retry_policy?.policy_code || '未配置' }}
                <span v-if="detail.retry_policy"> · v{{ detail.retry_policy.policy_version }}</span>
              </div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">Attempt</div>
              <div>{{ detail.current_attempt }} / {{ detail.max_attempts }}</div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">剩余次数</div>
              <div>{{ detail.attempts_remaining }}</div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">下次重试</div>
              <div>{{ formatDate(detail.next_run_at) }}</div>
            </div>
            <div class="col-12">
              <div class="text-caption text-grey-7">重试原因</div>
              <div>{{ formatRetryReason(detail.retry_reason_code) }}</div>
            </div>
          </div>
        </q-card-section>
      </q-card>
      <q-card flat bordered class="q-mb-md"
        ><q-card-section
          ><div class="text-subtitle1 text-weight-bold q-mb-md">输入快照摘要</div>
          <div class="row q-col-gutter-lg">
            <div class="col-6 col-md-2">
              <div class="text-caption text-grey-7">快照版本</div>
              <div>v{{ detail.input_summary.snapshot_version }}</div>
            </div>
            <div class="col-6 col-md-2">
              <div class="text-caption text-grey-7">快照大小</div>
              <div>{{ detail.input_summary.size_bytes }} 字节</div>
            </div>
            <div class="col-6 col-md-2">
              <div class="text-caption text-grey-7">Path 参数</div>
              <div>{{ detail.input_summary.path_count }}</div>
            </div>
            <div class="col-6 col-md-2">
              <div class="text-caption text-grey-7">Query 参数</div>
              <div>{{ detail.input_summary.query_count }}</div>
            </div>
            <div class="col-6 col-md-2">
              <div class="text-caption text-grey-7">Header 参数</div>
              <div>{{ detail.input_summary.header_count }}</div>
            </div>
            <div class="col-6 col-md-2">
              <div class="text-caption text-grey-7">JSON Body</div>
              <div>{{ detail.input_summary.has_body ? '有' : '无' }}</div>
            </div>
          </div>
          <div class="q-mt-md">
            <div class="text-caption text-grey-7">输入 Hash</div>
            <div class="text-mono text-break">{{ detail.input_hash }}</div>
          </div></q-card-section
        ></q-card
      >
      <q-card flat bordered class="q-mb-md"
        ><q-card-section
          ><div class="text-subtitle1 text-weight-bold q-mb-md">状态与结果摘要</div>
          <div class="row q-col-gutter-lg">
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">当前 Attempt</div>
              <div>{{ detail.current_attempt }}</div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">开始时间</div>
              <div>{{ formatDate(detail.started_at) }}</div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">完成时间</div>
              <div>{{ formatDate(detail.completed_at) }}</div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">耗时</div>
              <div>{{ detail.duration_ms }} 毫秒</div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">租约持有者</div>
              <div>{{ detail.lease_owner_summary || '-' }}</div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">租约到期</div>
              <div>{{ formatDate(detail.lease_expires_at) }}</div>
            </div>
            <div class="col-6 col-md-3">
              <div class="text-caption text-grey-7">错误分类</div>
              <div>{{ detail.error_category || '-' }}</div>
            </div>
          </div>
          <div v-if="detail.result_summary" class="q-mt-md text-body2">
            {{ detail.result_summary }}
          </div></q-card-section
        ></q-card
      >
      <q-card v-if="detail.sync_business" flat bordered class="q-mb-md">
        <q-card-section>
          <div class="text-subtitle1 text-weight-bold q-mb-md">同步业务结果</div>
          <div class="row q-col-gutter-lg">
            <div class="col-6 col-md-3"><div class="text-caption text-grey-7">状态</div><div>{{ detail.sync_business.status }}</div></div>
            <div class="col-6 col-md-3"><div class="text-caption text-grey-7">成功 / 失败</div><div>{{ detail.sync_business.success_count }} / {{ detail.sync_business.failed_count }}</div></div>
            <div class="col-6 col-md-3"><div class="text-caption text-grey-7">原因</div><div>{{ detail.sync_business.reason_code || '-' }}</div></div>
            <div class="col-6 col-md-3"><div class="text-caption text-grey-7">业务引用</div><div>{{ detail.sync_business.reference || '-' }}</div></div>
          </div>
        </q-card-section>
      </q-card>
      <q-card flat bordered
        ><q-card-section
          ><div class="text-subtitle1 text-weight-bold q-mb-sm">Attempt 记录</div>
          <div v-if="!canQueryLogs" class="text-body2 text-grey-7 q-py-md">
            无调用日志查看权限
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
                  @click="openLog(props.row.id)" />
                <span v-else>#{{ props.row.attempt_no }}</span></q-td
              ></template
            ><template #body-cell-status="props"
              ><q-td :props="props"
                ><q-chip
                  dense
                  square
                  :color="logStatusMeta[props.row.status]?.color || 'grey'"
                  text-color="white"
                  :label="
                    logStatusMeta[props.row.status]?.label || props.row.status
                  " /></q-td></template></q-table></q-card-section
      ></q-card>
    </template>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_execution_detail_page' })
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import {
  useIntegrationApi,
  type IntegrationExecutionDetail,
  type IntegrationLogListItem,
} from 'src/api/services/integration'
import { useLoadingStore } from 'src/stores/loading'
import { useUserStore } from 'src/stores/user'
import { storeToRefs } from 'pinia'
import type { QTableProps } from 'quasar'
import { formatRetryReason, formatRuntimeDateTime } from 'src/pages/integration/runtime-display'

const route = useRoute()
const router = useRouter()
const api = useIntegrationApi()
const userStore = useUserStore()
const { loading } = storeToRefs(useLoadingStore())
const detail = ref<IntegrationExecutionDetail | null>(null)
const attempts = ref<IntegrationLogListItem[]>([])
const attemptsLoading = ref(false)
const canViewExecutionDetail = computed(() => userStore.buttons.includes('integration_execution_detail'))
const canQueryLogs = computed(() => userStore.buttons.includes('integration_log_query'))
const canViewLogDetail = computed(() => userStore.buttons.includes('integration_log_detail'))
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
const attemptColumns: QTableProps['columns'] = [
  { name: 'attempt_no', label: 'Attempt', field: 'attempt_no', align: 'left' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
  { name: 'http_status', label: 'HTTP状态', field: 'http_status', align: 'center' },
  { name: 'duration_ms', label: '耗时（毫秒）', field: 'duration_ms', align: 'right' },
  { name: 'error_category', label: '错误分类', field: 'error_category', align: 'left' },
  { name: 'result_certainty', label: '结果确定性', field: 'result_certainty', align: 'left' },
]
const formatDate = formatRuntimeDateTime
const openLog = (logId: number) => {
  if (detail.value)
    void router.push({
      name: 'integration_log',
      query: { execution_no: detail.value.execution_no, log_id: String(logId) },
    })
}
onMounted(async () => {
  if (!canViewExecutionDetail.value) return
  const id = Number(route.params.id)
  if (id > 0) {
    const response = await api.getExecution(id)
    detail.value = response.data || null
    if (detail.value && canQueryLogs.value) {
      attemptsLoading.value = true
      try {
        const logs = await api.queryLogs({
          page: 1,
          num: 500,
          order: { field: 'attempt_no', is_asc: true },
          quick_query: { keyword: '' },
          expressions: [],
          execution_id: detail.value.id,
        })
        attempts.value = logs.data || []
      } finally {
        attemptsLoading.value = false
      }
    }
  }
})
</script>
