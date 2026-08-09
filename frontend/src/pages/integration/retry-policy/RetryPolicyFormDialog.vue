<template>
  <form-dialog-shell
    v-model="visible"
    :title="editData ? '编辑重试策略' : '新增重试策略'"
    :subtitle="editData ? `${editData.policy_code} · v${editData.version}` : '创建版本 1 草稿'"
    icon="autorenew"
    :submit-text="editData ? '保存' : '创建'"
    :loading="loading || false"
    width="min(980px, calc(100vw - 48px))"
    @submit="submit"
  >
    <q-form ref="formRef" class="retry-policy-form">
      <q-input
        v-model="form.policy_code"
        outlined dense
        :disable="Boolean(editData)"
        label="策略编码 *"
        hint="稳定编码创建后不可修改"
        :rules="[(value) => /^[a-z][a-z0-9_]{0,63}$/.test(value || '') || '请输入合法策略编码']"
      />
      <q-input v-model="form.policy_name" outlined dense label="策略名称 *" :rules="[requiredName]" />
      <q-input v-model.number="form.max_attempts" outlined dense type="number" min="1" max="10" label="最大尝试次数 *" hint="包含首次调用，范围 1 至 10" :rules="[attemptRule]" />
      <q-input v-model.number="initialDelaySeconds" outlined dense type="number" min="1" max="3600" label="初始延迟（秒） *" :rules="[initialDelayRule]" />
      <q-input v-model.number="maxDelaySeconds" outlined dense type="number" :min="initialDelaySeconds" max="86400" label="最大延迟（秒） *" :rules="[maxDelayRule]" />
      <q-input v-model.number="retryWindowSeconds" outlined dense type="number" min="60" max="604800" label="重试窗口（秒） *" hint="1 分钟至 7 天" :rules="[retryWindowRule]" />
      <q-select v-model="form.backoff_type" outlined dense emit-value map-options :options="backoffOptions" label="退避方式 *" />
      <q-input
        v-if="form.backoff_type === 'exponential'"
        v-model.number="form.backoff_multiplier"
        outlined dense type="number" min="1.1" max="4" step="0.1"
        label="退避倍数 *"
        :rules="[multiplierRule]"
      />
      <q-select v-model="form.jitter_type" outlined dense emit-value map-options :options="jitterOptions" label="抖动方式 *" />
      <q-input
        v-if="form.jitter_type === 'full'"
        v-model.number="form.jitter_ratio"
        outlined dense type="number" min="1" max="1"
        label="抖动比例 *"
        hint="V1 full jitter 固定为 1"
        :rules="[(value) => value === 1 || 'full jitter 比例必须为 1']"
      />
      <q-select
        v-model="form.retryable_error_categories"
        outlined dense multiple emit-value map-options use-chips
        :options="errorCategoryOptions"
        label="可重试错误分类"
      />
      <q-select
        v-model="form.retryable_http_statuses"
        outlined dense multiple emit-value map-options use-chips
        :options="httpStatusOptions"
        label="可重试 HTTP 状态"
      />
      <q-toggle v-model="form.respect_retry_after" color="primary" label="遵循 Retry-After" />
      <q-input v-model="form.description" outlined dense type="textarea" autogrow class="retry-policy-form__wide" label="描述" maxlength="512" />
    </q-form>
    <template #footer-status>
      <span class="text-caption text-grey-7">策略启用后技术字段不可直接修改，请创建新版本</span>
    </template>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { QForm } from 'quasar'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import type {
  RetryBackoffType,
  RetryErrorCategory,
  RetryHTTPStatus,
  RetryJitterType,
  RetryPolicyCreateRequest,
  RetryPolicyDetail,
} from 'src/api/services/integration'

export type RetryPolicyFormValue = RetryPolicyCreateRequest

const props = withDefaults(defineProps<{
  modelValue: boolean
  editData: RetryPolicyDetail | null
  loading?: boolean
}>(), { loading: false })
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: RetryPolicyFormValue): void
}>()

const formRef = ref<QForm | null>(null)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const form = reactive<RetryPolicyFormValue>(emptyForm())
const initialDelaySeconds = durationSeconds('initial_delay_ms')
const maxDelaySeconds = durationSeconds('max_delay_ms')
const retryWindowSeconds = durationSeconds('retry_window_ms')

const backoffOptions: { label: string; value: RetryBackoffType }[] = [
  { label: '固定间隔', value: 'fixed' },
  { label: '指数退避', value: 'exponential' },
]
const jitterOptions: { label: string; value: RetryJitterType }[] = [
  { label: '不使用抖动', value: 'none' },
  { label: 'Full Jitter', value: 'full' },
]
const errorCategoryOptions: { label: string; value: RetryErrorCategory }[] = [
  { label: '临时网络错误', value: 'network' },
  { label: '超时', value: 'timeout' },
  { label: '远端服务错误', value: 'remote' },
]
const httpStatusOptions: { label: string; value: RetryHTTPStatus }[] = [429, 502, 503, 504].map((value) => ({ label: String(value), value: value as RetryHTTPStatus }))

function emptyForm(): RetryPolicyFormValue {
  return {
    policy_code: '', policy_name: '', description: '', max_attempts: 3,
    initial_delay_ms: 5000, max_delay_ms: 300000,
    backoff_type: 'exponential', backoff_multiplier: 2,
    jitter_type: 'full', jitter_ratio: 1, retry_window_ms: 86400000,
    retryable_error_categories: ['network', 'timeout', 'remote'],
    retryable_http_statuses: [429, 502, 503, 504], respect_retry_after: true,
  }
}

function durationSeconds(field: 'initial_delay_ms' | 'max_delay_ms' | 'retry_window_ms') {
  return computed({
    get: () => form[field] / 1000,
    set: (value: number) => { form[field] = Number.isFinite(Number(value)) ? Math.round(Number(value) * 1000) : 0 },
  })
}

const requiredName = (value: string) => Boolean(value?.trim()) || '请输入策略名称'
const attemptRule = (value: number) => Number.isInteger(value) && value >= 1 && value <= 10 || '最大尝试次数必须在 1 至 10 之间'
const initialDelayRule = (value: number) => value >= 1 && value <= 3600 || '初始延迟必须在 1 至 3,600 秒之间'
const maxDelayRule = (value: number) => value >= initialDelaySeconds.value && value <= 86400 || '最大延迟不能小于初始延迟，且不能超过 86,400 秒'
const requiredRetryWindowSeconds = () => {
  let delayMs = form.initial_delay_ms
  let totalMs = 0
  for (let attempt = 1; attempt < form.max_attempts; attempt += 1) {
    totalMs += delayMs
    if (form.backoff_type === 'exponential') {
      delayMs = Math.min(form.max_delay_ms, Math.ceil(delayMs * form.backoff_multiplier))
    }
  }
  return Math.ceil(totalMs / 1000)
}
const retryWindowRule = (value: number) => value >= 60 && value <= 604800 && value >= requiredRetryWindowSeconds() || '重试窗口必须在 60 至 604,800 秒之间，并覆盖完整重试计划'
const multiplierRule = (value: number) => value >= 1.1 && value <= 4 || '指数退避倍数必须在 1.1 至 4 之间'

watch(() => [props.modelValue, props.editData] as const, ([open, detail]) => {
  if (!open) return
  Object.assign(form, detail ? {
    policy_code: detail.policy_code, policy_name: detail.policy_name, description: detail.description,
    max_attempts: detail.max_attempts, initial_delay_ms: detail.initial_delay_ms,
    max_delay_ms: detail.max_delay_ms, backoff_type: detail.backoff_type,
    backoff_multiplier: detail.backoff_multiplier, jitter_type: detail.jitter_type,
    jitter_ratio: detail.jitter_ratio, retry_window_ms: detail.retry_window_ms,
    retryable_error_categories: [...detail.retryable_error_categories],
    retryable_http_statuses: [...detail.retryable_http_statuses],
    respect_retry_after: detail.respect_retry_after,
  } : emptyForm())
}, { immediate: true })

watch(() => form.backoff_type, (value) => {
  if (value === 'fixed') form.backoff_multiplier = 1
  else if (form.backoff_multiplier < 1.1 || form.backoff_multiplier > 4) form.backoff_multiplier = 2
})
watch(() => form.jitter_type, (value) => {
  if (value === 'none') form.jitter_ratio = 0
  else if (form.jitter_ratio !== 1) form.jitter_ratio = 1
})

const submit = async () => {
  if (!(await formRef.value?.validate())) return
  emit('submit', {
    ...form,
    retryable_error_categories: [...form.retryable_error_categories],
    retryable_http_statuses: [...form.retryable_http_statuses],
  })
}
</script>

<style scoped lang="scss">
.retry-policy-form { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px 22px; padding: 4px 4px 18px; }
.retry-policy-form__wide { grid-column: 1 / -1; }
@media (max-width: 700px) { .retry-policy-form { grid-template-columns: 1fr; } .retry-policy-form__wide { grid-column: auto; } }
</style>
