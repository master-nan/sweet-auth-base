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
      <q-card flat bordered
        ><q-card-section
          ><div class="text-subtitle1 text-weight-bold q-mb-sm">Attempt 记录</div>
          <q-table
            flat
            bordered
            dense
            :rows="detail.attempts"
            :columns="attemptColumns"
            row-key="id"
            ><template #body-cell-attempt_no="props"
              ><q-td :props="props"
                ><q-btn
                  flat
                  dense
                  color="primary"
                  :label="`#${props.row.attempt_no}`"
                  @click="openLog(props.row.id)" /></q-td></template
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
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import { useIntegrationApi, type IntegrationExecutionDetail } from 'src/api/services/integration'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import type { QTableProps } from 'quasar'
import { formatRuntimeDateTime } from 'src/pages/integration/runtime-display'

const route = useRoute()
const router = useRouter()
const api = useIntegrationApi()
const { loading } = storeToRefs(useLoadingStore())
const detail = ref<IntegrationExecutionDetail | null>(null)
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
  const id = Number(route.params.id)
  if (id > 0) {
    const response = await api.getExecution(id)
    detail.value = response.data || null
  }
})
</script>
