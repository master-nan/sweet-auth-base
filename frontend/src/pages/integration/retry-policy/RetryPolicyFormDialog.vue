<template>
  <form-dialog-shell
    v-model="visible"
    :title="editData ? t('ui.editRetryPolicy') : t('ui.addRetryPolicy')"
    :subtitle="
      editData ? `${editData.policy_code} · v${editData.version}` : t('ui.createDraftVersion1')
    "
    icon="autorenew"
    :submit-text="editData ? t('ui.save') : t('ui.createRecord')"
    :loading="loading || false"
    width="min(980px, calc(100vw - 48px))"
    @submit="submit"
  >
    <q-form ref="formRef" class="retry-policy-form">
      <q-input
        v-model="form.policy_code"
        outlined
        dense
        :disable="Boolean(editData)"
        :label="t('ui.policyEncoding')"
        :hint="t('ui.unmodifiableAfterStableEncoding')"
        :rules="[
          (value) =>
            /^[a-z][a-z0-9_]{0,63}$/.test(value || '') || t('ui.pleaseEnterAValidPolicyCode'),
        ]"
      />
      <q-input
        v-model="form.policy_name"
        outlined
        dense
        :label="t('ui.policyName')"
        :rules="[requiredName]"
      />
      <q-input
        v-model.number="form.max_attempts"
        outlined
        dense
        type="number"
        min="1"
        max="10"
        :label="t('ui.maximumAttempts')"
        :hint="t('ui.includeFirstCallRange1To10')"
        :rules="[attemptRule]"
      />
      <q-input
        v-model.number="initialDelaySeconds"
        outlined
        dense
        type="number"
        min="1"
        max="3600"
        :label="t('ui.initialDelaySeconds')"
        :rules="[initialDelayRule]"
      />
      <q-input
        v-model.number="maxDelaySeconds"
        outlined
        dense
        type="number"
        :min="initialDelaySeconds"
        max="86400"
        :label="t('ui.maximumDelaySeconds')"
        :rules="[maxDelayRule]"
      />
      <q-input
        v-model.number="retryWindowSeconds"
        outlined
        dense
        type="number"
        min="60"
        max="604800"
        :label="t('ui.retryWindowSec')"
        :hint="t('ui.retryWindowRangeHint')"
        :rules="[retryWindowRule]"
      />
      <q-select
        v-model="form.backoff_type"
        outlined
        dense
        emit-value
        map-options
        :options="backoffOptions"
        :label="t('ui.theWayOut')"
      />
      <q-input
        v-if="form.backoff_type === 'exponential'"
        v-model.number="form.backoff_multiplier"
        outlined
        dense
        type="number"
        min="1.1"
        max="4"
        step="0.1"
        :label="t('ui.numberOfRetreats')"
        :rules="[multiplierRule]"
      />
      <q-select
        v-model="form.jitter_type"
        outlined
        dense
        emit-value
        map-options
        :options="jitterOptions"
        :label="t('ui.dithering')"
      />
      <q-input
        v-if="form.jitter_type === 'full'"
        v-model.number="form.jitter_ratio"
        outlined
        dense
        type="number"
        min="1"
        max="1"
        :label="t('ui.dither1')"
        :hint="t('ui.v1FullJitterFixedTo1')"
        :rules="[(value) => value === 1 || t('ui.fullJitterRatioMustBe1')]"
      />
      <q-select
        v-model="form.retryable_error_categories"
        outlined
        dense
        multiple
        emit-value
        map-options
        use-chips
        :options="errorCategoryOptions"
        :label="t('ui.retryableErrorCategory')"
      />
      <q-select
        v-model="form.retryable_http_statuses"
        outlined
        dense
        multiple
        emit-value
        map-options
        use-chips
        :options="httpStatusOptions"
        :label="t('ui.retryableHttpState')"
      />
      <q-toggle
        v-model="form.respect_retry_after"
        color="primary"
        :label="t('ui.followRetryAfter')"
      />
      <q-input
        v-model="form.description"
        outlined
        dense
        type="textarea"
        autogrow
        class="retry-policy-form__wide"
        :label="t('ui.description')"
        maxlength="512"
      />
    </q-form>
    <template #footer-status>
      <span class="text-caption text-grey-7">{{
        t('ui.policyEnabledTechnicalFieldsCannotBeModifiedDirectlyCreateA')
      }}</span>
    </template>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, reactive, ref, watch } from 'vue'
import type { QForm } from 'quasar'
import FormDialogShell from '@/components/FormDialog/FormDialogShell.vue'
import type {
  RetryBackoffType,
  RetryErrorCategory,
  RetryHTTPStatus,
  RetryJitterType,
  RetryPolicyCreateRequest,
  RetryPolicyDetail,
} from '@/api/services/integration'

const { t } = useI18n({ useScope: 'global' })

export type RetryPolicyFormValue = RetryPolicyCreateRequest

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    editData: RetryPolicyDetail | null
    loading?: boolean
  }>(),
  { loading: false },
)
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: RetryPolicyFormValue): void
}>()

const formRef = ref<QForm | null>(null)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const form = reactive<RetryPolicyFormValue>(emptyForm())
const initialDelaySeconds = durationSeconds('initial_delay_ms')
const maxDelaySeconds = durationSeconds('max_delay_ms')
const retryWindowSeconds = durationSeconds('retry_window_ms')

const backoffOptions: { label: string; value: RetryBackoffType }[] = [
  {
    get label() {
      return t('ui.fixedInterval')
    },
    value: 'fixed',
  },
  {
    get label() {
      return t('ui.exponentialBackoff')
    },
    value: 'exponential',
  },
]
const jitterOptions: { label: string; value: RetryJitterType }[] = [
  {
    get label() {
      return t('ui.noJitter')
    },
    value: 'none',
  },
  { label: 'Full Jitter', value: 'full' },
]
const errorCategoryOptions: { label: string; value: RetryErrorCategory }[] = [
  {
    get label() {
      return t('ui.temporaryNetworkError')
    },
    value: 'network',
  },
  {
    get label() {
      return t('ui.timeout')
    },
    value: 'timeout',
  },
  {
    get label() {
      return t('ui.remoteServiceError')
    },
    value: 'remote',
  },
]
const httpStatusOptions: { label: string; value: RetryHTTPStatus }[] = [429, 502, 503, 504].map(
  (value) => ({ label: String(value), value: value as RetryHTTPStatus }),
)

function emptyForm(): RetryPolicyFormValue {
  return {
    policy_code: '',
    policy_name: '',
    description: '',
    max_attempts: 3,
    initial_delay_ms: 5000,
    max_delay_ms: 300000,
    backoff_type: 'exponential',
    backoff_multiplier: 2,
    jitter_type: 'full',
    jitter_ratio: 1,
    retry_window_ms: 86400000,
    retryable_error_categories: ['network', 'timeout', 'remote'],
    retryable_http_statuses: [429, 502, 503, 504],
    respect_retry_after: true,
  }
}

function durationSeconds(field: 'initial_delay_ms' | 'max_delay_ms' | 'retry_window_ms') {
  return computed({
    get: () => form[field] / 1000,
    set: (value: number) => {
      form[field] = Number.isFinite(Number(value)) ? Math.round(Number(value) * 1000) : 0
    },
  })
}

const requiredName = (value: string) => Boolean(value?.trim()) || t('ui.pleaseEnterAPolicyName')
const attemptRule = (value: number) =>
  (Number.isInteger(value) && value >= 1 && value <= 10) ||
  t('ui.maximumNumberOfAttemptsMustBeBetween1And10')
const initialDelayRule = (value: number) =>
  (value >= 1 && value <= 3600) || t('ui.initialDelayMustBeBetween1And3600Seconds')
const maxDelayRule = (value: number) =>
  (value >= initialDelaySeconds.value && value <= 86400) ||
  t('ui.maximumDelayCannotBeLessThanTheInitialDelayAnd')
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
const retryWindowRule = (value: number) =>
  (value >= 60 && value <= 604800 && value >= requiredRetryWindowSeconds()) ||
  t('ui.retryWindowMustBeBetween60And604800Seconds')
const multiplierRule = (value: number) =>
  (value >= 1.1 && value <= 4) || t('ui.indexEvasiveMultiplesMustBeBetween11And4')

watch(
  () => [props.modelValue, props.editData] as const,
  ([open, detail]) => {
    if (!open) return
    Object.assign(
      form,
      detail
        ? {
            policy_code: detail.policy_code,
            policy_name: detail.policy_name,
            description: detail.description,
            max_attempts: detail.max_attempts,
            initial_delay_ms: detail.initial_delay_ms,
            max_delay_ms: detail.max_delay_ms,
            backoff_type: detail.backoff_type,
            backoff_multiplier: detail.backoff_multiplier,
            jitter_type: detail.jitter_type,
            jitter_ratio: detail.jitter_ratio,
            retry_window_ms: detail.retry_window_ms,
            retryable_error_categories: [...detail.retryable_error_categories],
            retryable_http_statuses: [...detail.retryable_http_statuses],
            respect_retry_after: detail.respect_retry_after,
          }
        : emptyForm(),
    )
  },
  { immediate: true },
)

watch(
  () => form.backoff_type,
  (value) => {
    if (value === 'fixed') form.backoff_multiplier = 1
    else if (form.backoff_multiplier < 1.1 || form.backoff_multiplier > 4)
      form.backoff_multiplier = 2
  },
)
watch(
  () => form.jitter_type,
  (value) => {
    if (value === 'none') form.jitter_ratio = 0
    else if (form.jitter_ratio !== 1) form.jitter_ratio = 1
  },
)

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
.retry-policy-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px 22px;
  padding: 4px 4px 18px;
}
.retry-policy-form__wide {
  grid-column: 1 / -1;
}
@media (max-width: 700px) {
  .retry-policy-form {
    grid-template-columns: 1fr;
  }
  .retry-policy-form__wide {
    grid-column: auto;
  }
}
</style>
