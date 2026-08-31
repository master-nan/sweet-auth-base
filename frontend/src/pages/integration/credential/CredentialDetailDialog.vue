<template>
  <form-dialog-shell
    v-model="visible"
    :title="t('ui.integratedVoucherDetails')"
    :subtitle="
      detail?.credential_code ||
      (loadFailed ? t('ui.documentDetailsFailedToRead') : t('ui.readingVoucherMetadata'))
    "
    icon="key"
    readonly
    :loading="loading"
    width="min(900px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="credential-detail">
      <section>
        <div class="credential-detail__title">{{ t('ui.basicInfo') }}</div>
        <detail-field-grid :items="basicItems" />
      </section>
      <q-separator />
      <section>
        <div class="credential-detail__title">{{ t('ui.securityAndRotation') }}</div>
        <detail-field-grid :items="securityItems" />
        <q-banner class="credential-detail__notice q-mt-md" rounded>{{
          t('ui.theRotationHistoryIsWrittenInThePlatformSAudit')
        }}</q-banner>
      </section>
    </div>
    <div v-else-if="loading" class="credential-detail__loading">
      <q-spinner-dots color="primary" size="36px" />
    </div>
    <div v-else class="credential-detail__error">
      <q-icon name="error_outline" color="negative" size="42px" />
      <div class="text-subtitle1 text-weight-bold">{{ t('ui.couldNotCloseTemporaryFolderS') }}</div>
      <div class="text-body2 text-grey-7">
        {{ t('ui.theCredentialMayHaveExpiredOrItsExternalSystemMay') }}
      </div>
      <q-btn outline color="primary" icon="refresh" :label="t('ui.reload')" @click="load" />
    </div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import DetailFieldGrid from 'src/components/Detail/DetailFieldGrid.vue'
import type { DetailFieldItem } from 'src/components/Detail/types'
import { type CredentialDetail, useIntegrationApi } from 'src/api/services/integration'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const detail = ref<CredentialDetail | null>(null)
const loading = ref(false)
const loadFailed = ref(false)
const typeLabels: Record<string, string> = {
  basic: 'Basic',
  api_key: 'API Key',
  bearer_token: 'Bearer Token',
  oauth_client: 'OAuth Client',
}
const statusLabels: Record<string, string> = {
  get draft() {
    return t('ui.draft')
  },
  get active() {
    return t('ui.activatedStatus')
  },
  get disabled() {
    return t('ui.deactivatedStatus')
  },
  get revoked() {
    return t('ui.revoked')
  },
  get expired() {
    return t('ui.expired')
  },
}
const basicItems = computed<DetailFieldItem[]>(() =>
  detail.value
    ? [
        {
          get label() {
            return t('ui.owningSystem')
          },
          value: `${detail.value.external_system.name}（${detail.value.external_system.system_code}）`,
        },
        {
          get label() {
            return t('ui.credentialCodeLabel')
          },
          value: detail.value.credential_code,
        },
        {
          get label() {
            return t('ui.certificateName')
          },
          value: detail.value.name,
        },
        {
          get label() {
            return t('ui.type')
          },
          value: typeLabels[detail.value.credential_type] || detail.value.credential_type,
        },
        {
          get label() {
            return t('ui.status')
          },
          value: statusLabels[detail.value.effective_status] || detail.value.effective_status,
        },
        {
          get label() {
            return t('ui.created')
          },
          value: detail.value.gmt_create,
        },
      ]
    : [],
)
const securityItems = computed<DetailFieldItem[]>(() =>
  detail.value
    ? [
        {
          get label() {
            return t('ui.secretVersion')
          },
          value: `v${detail.value.version}`,
        },
        {
          get label() {
            return t('ui.fingerprintSummary')
          },
          value: detail.value.fingerprint_summary || '-',
        },
        {
          get label() {
            return t('ui.validityPeriod')
          },
          value: detail.value.expires_at || t('ui.noExpiration'),
        },
        {
          get label() {
            return t('ui.recentRotation')
          },
          value: detail.value.rotated_at || t('ui.notYetRotated'),
        },
        {
          get label() {
            return t('ui.description')
          },
          value: detail.value.description || '-',
          fullWidth: true,
        },
      ]
    : [],
)
const load = async () => {
  if (!props.id) return
  loading.value = true
  loadFailed.value = false
  detail.value = null
  try {
    detail.value = (await api.getCredential(props.id)).data
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}
watch(
  () => [props.modelValue, props.id] as const,
  ([open]) => {
    if (!open || !props.id) {
      detail.value = null
      loadFailed.value = false
      return
    }
    void load()
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.credential-detail {
  display: grid;
  gap: 24px;
  padding: 4px 6px 20px;
}
.credential-detail__title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 700;
}
.credential-detail__notice {
  background: var(--app-primary-soft);
  color: inherit;
}
.credential-detail__loading,
.credential-detail__error {
  min-height: 240px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  text-align: center;
}
</style>
