<template>
  <form-dialog-shell
    v-model="visible"
    :title="t('ui.retryPolicyDetails')"
    :subtitle="
      detail ? `${detail.policy_code} · v${detail.version}` : t('ui.readingPolicyConfiguration')
    "
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
        <div class="text-caption text-grey-7">{{ t('ui.description') }}</div>
        <div class="text-body1">{{ detail.description || '-' }}</div>
      </div>
    </div>
    <div v-else class="row justify-center q-pa-xl">
      <q-spinner-dots color="primary" size="36px" />
    </div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { type RetryPolicyDetail, useIntegrationApi } from 'src/api/services/integration'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const detail = ref<RetryPolicyDetail | null>(null)
const loading = ref(false)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const statusLabels = {
  get draft() {
    return t('ui.draft')
  },
  get enabled() {
    return t('ui.activatedStatus')
  },
  get disabled() {
    return t('ui.deactivatedStatus')
  },
}
const backoffLabels = {
  get fixed() {
    return t('ui.fixedInterval')
  },
  get exponential() {
    return t('ui.exponentialBackoff')
  },
}
const jitterLabels = {
  get none() {
    return t('ui.noShake')
  },
  full: 'Full Jitter',
}
const errorLabels: Record<string, string> = {
  get network() {
    return t('ui.temporaryNetworkError')
  },
  get timeout() {
    return t('ui.timeout')
  },
  get remote() {
    return t('ui.remoteServiceError')
  },
}
const duration = (milliseconds: number) =>
  milliseconds % 1000 === 0
    ? t('ui.secondsValue', { value1: milliseconds / 1000 })
    : t('ui.millisecond', { milliseconds: milliseconds })
const items = computed(() =>
  detail.value
    ? [
        {
          get label() {
            return t('ui.policyNameLabel')
          },
          value: detail.value.policy_name,
        },
        {
          get label() {
            return t('ui.policyCode')
          },
          value: detail.value.policy_code,
          mono: true,
        },
        {
          get label() {
            return t('ui.version')
          },
          value: `v${detail.value.version}`,
        },
        {
          get label() {
            return t('ui.status')
          },
          value: statusLabels[detail.value.status],
        },
        {
          get label() {
            return t('ui.maximumNumberOfAttempts')
          },
          value: t('ui.includingFirstTime', { value1: detail.value.max_attempts }),
        },
        {
          get label() {
            return t('ui.howToRetreat')
          },
          value: backoffLabels[detail.value.backoff_type],
        },
        {
          get label() {
            return t('ui.initialMaximumDelay')
          },
          value: `${duration(detail.value.initial_delay_ms)} / ${duration(detail.value.max_delay_ms)}`,
        },
        {
          get label() {
            return t('ui.backoffMultiplier')
          },
          value: String(detail.value.backoff_multiplier),
        },
        {
          get label() {
            return t('ui.dither')
          },
          value: `${jitterLabels[detail.value.jitter_type]} · ${detail.value.jitter_ratio}`,
        },
        {
          get label() {
            return t('ui.retryWindow')
          },
          value: duration(detail.value.retry_window_ms),
        },
        {
          get label() {
            return t('ui.errorCategory')
          },
          value:
            detail.value.retryable_error_categories
              .map((item) => errorLabels[item] || item)
              .join('、') || '-',
        },
        {
          get label() {
            return t('ui.httpStatus')
          },
          value: detail.value.retryable_http_statuses.join('、') || '-',
        },
        {
          label: 'Retry-After',
          value: detail.value.respect_retry_after ? t('ui.follow') : t('ui.ignore'),
        },
        {
          get label() {
            return t('ui.updatedAt')
          },
          value: detail.value.gmt_modify,
        },
      ]
    : [],
)

watch(
  () => [props.modelValue, props.id] as const,
  async ([open, id]) => {
    if (!open || !id) return
    loading.value = true
    try {
      detail.value = (await api.getRetryPolicy(id)).data || null
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.retry-policy-detail {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 20px 24px;
  padding: 8px 4px 20px;
}
.retry-policy-detail__item {
  min-width: 0;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--app-border);
}
.retry-policy-detail__wide {
  grid-column: 1 / -1;
}
@media (max-width: 760px) {
  .retry-policy-detail {
    grid-template-columns: 1fr;
  }
  .retry-policy-detail__wide {
    grid-column: auto;
  }
}
</style>
