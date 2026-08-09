<template>
  <form-dialog-shell
    v-model="visible"
    title="重试策略详情"
    :subtitle="detail ? `${detail.policy_code} · v${detail.version}` : '正在读取策略配置'"
    icon="autorenew"
    readonly
    :loading="loading"
    width="min(960px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="retry-policy-detail">
      <div v-for="item in items" :key="item.label" class="retry-policy-detail__item">
        <div class="text-caption text-grey-7">{{ item.label }}</div>
        <div class="text-body1" :class="{ 'text-mono': item.mono }">{{ item.value }}</div>
      </div>
      <div class="retry-policy-detail__item retry-policy-detail__wide">
        <div class="text-caption text-grey-7">描述</div>
        <div class="text-body1">{{ detail.description || '-' }}</div>
      </div>
    </div>
    <div v-else class="row justify-center q-pa-xl"><q-spinner-dots color="primary" size="36px" /></div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { type RetryPolicyDetail, useIntegrationApi } from 'src/api/services/integration'

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const detail = ref<RetryPolicyDetail | null>(null)
const loading = ref(false)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const statusLabels = { draft: '草稿', enabled: '已启用', disabled: '已停用' }
const backoffLabels = { fixed: '固定间隔', exponential: '指数退避' }
const jitterLabels = { none: '无抖动', full: 'Full Jitter' }
const errorLabels: Record<string, string> = { network: '临时网络错误', timeout: '超时', remote: '远端服务错误' }
const duration = (milliseconds: number) => milliseconds % 1000 === 0 ? `${milliseconds / 1000} 秒` : `${milliseconds} 毫秒`
const items = computed(() => detail.value ? [
  { label: '策略名称', value: detail.value.policy_name },
  { label: '策略编码', value: detail.value.policy_code, mono: true },
  { label: '版本', value: `v${detail.value.version}` },
  { label: '状态', value: statusLabels[detail.value.status] },
  { label: '最大尝试次数', value: `${detail.value.max_attempts} 次（含首次）` },
  { label: '退避方式', value: backoffLabels[detail.value.backoff_type] },
  { label: '初始 / 最大延迟', value: `${duration(detail.value.initial_delay_ms)} / ${duration(detail.value.max_delay_ms)}` },
  { label: '退避倍数', value: String(detail.value.backoff_multiplier) },
  { label: '抖动', value: `${jitterLabels[detail.value.jitter_type]} · ${detail.value.jitter_ratio}` },
  { label: '重试窗口', value: duration(detail.value.retry_window_ms) },
  { label: '错误分类', value: detail.value.retryable_error_categories.map((item) => errorLabels[item] || item).join('、') || '-' },
  { label: 'HTTP 状态', value: detail.value.retryable_http_statuses.join('、') || '-' },
  { label: 'Retry-After', value: detail.value.respect_retry_after ? '遵循' : '忽略' },
  { label: '更新时间', value: detail.value.gmt_modify },
] : [])

watch(() => [props.modelValue, props.id] as const, async ([open, id]) => {
  if (!open || !id) return
  loading.value = true
  try { detail.value = (await api.getRetryPolicy(id)).data || null } finally { loading.value = false }
}, { immediate: true })
</script>

<style scoped lang="scss">
.retry-policy-detail { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 20px 24px; padding: 8px 4px 20px; }
.retry-policy-detail__item { min-width: 0; padding-bottom: 12px; border-bottom: 1px solid var(--app-border-color); }
.retry-policy-detail__wide { grid-column: 1 / -1; }
@media (max-width: 760px) { .retry-policy-detail { grid-template-columns: 1fr; } .retry-policy-detail__wide { grid-column: auto; } }
</style>
