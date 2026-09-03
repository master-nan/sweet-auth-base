<template>
  <form-dialog-shell
    v-model="visible"
    :title="
      rotateMode
        ? t('ui.rotationVouchers')
        : editData
          ? t('ui.editIntegratedCertificates')
          : t('ui.addIntegratedCertificate')
    "
    :subtitle="editData ? editData.credential_code : t('ui.itWonTBeShownAgainWhenItSFiledInSecret')"
    icon="key"
    :submit-text="
      rotateMode ? t('ui.confirmRotation') : editData ? t('ui.save') : t('ui.createRecord')
    "
    :loading="loading || false"
    width="min(900px, calc(100vw - 48px))"
    @submit="submit"
  >
    <q-form ref="formRef" class="credential-form">
      <template v-if="!rotateMode">
        <q-select
          v-model="form.external_system_id"
          outlined
          dense
          emit-value
          map-options
          :disable="Boolean(editData)"
          :options="systemOptions"
          :label="t('ui.externalSystem')"
          :rules="[(value) => Boolean(value) || t('ui.selectTheExternalSystemToWhichYouBelong')]"
        />
        <q-input
          v-model="form.credential_code"
          outlined
          dense
          :disable="Boolean(editData)"
          :label="t('ui.credentialCode')"
          :hint="t('ui.startWithALowercaseLetterNumbersAndUnderscoresAreAllowed')"
          :rules="[
            (value) =>
              /^[a-z][a-z0-9_]{1,63}$/.test(value || '') || t('ui.pleaseEnterAValidVoucherCode'),
          ]"
        />
        <q-input
          v-model="form.name"
          outlined
          dense
          :label="t('ui.credentialName')"
          :rules="[(value) => Boolean(value?.trim()) || t('ui.pleaseEnterTheNameOfTheCertificate')]"
        />
        <q-select
          v-model="form.credential_type"
          outlined
          dense
          emit-value
          map-options
          :disable="Boolean(editData)"
          :options="typeOptions"
          :label="t('ui.credentialType')"
        />
        <sweet-date-time-picker
          v-model="form.expires_at"
          type="datetime"
          :label="t('ui.validityPeriod')"
        />
        <q-input
          v-model="form.description"
          outlined
          dense
          type="textarea"
          autogrow
          :label="t('ui.description')"
        />
      </template>

      <template v-if="!editData || rotateMode">
        <q-banner class="credential-form__notice" rounded>
          <template #avatar><q-icon name="shield" color="primary" /></template>
          {{ t('ui.keysOnlyForThisTime') }}{{ rotateMode ? t('ui.rotation') : t('ui.createRecord')
          }}{{ t('ui.theOriginalValueWillNotBeDisplayedCopiedOrRestoredOnThe') }}
        </q-banner>
        <template v-if="form.credential_type === 'basic'">
          <q-input
            v-model="form.secret.username"
            outlined
            dense
            autocomplete="off"
            :label="t('ui.username')"
            :rules="[requiredSecret]"
          />
          <q-input
            v-model="form.secret.password"
            outlined
            dense
            type="password"
            autocomplete="new-password"
            :label="t('ui.password')"
            :rules="[requiredSecret]"
          />
        </template>
        <q-input
          v-else-if="form.credential_type === 'api_key'"
          v-model="form.secret.api_key"
          class="credential-form__wide"
          outlined
          dense
          type="password"
          autocomplete="new-password"
          label="API Key *"
          :rules="[requiredSecret]"
        />
        <q-input
          v-else-if="form.credential_type === 'bearer_token'"
          v-model="form.secret.token"
          class="credential-form__wide"
          outlined
          dense
          type="password"
          autocomplete="new-password"
          label="Bearer Token *"
          :rules="[requiredSecret]"
        />
        <template v-else>
          <q-input
            v-model="form.secret.client_id"
            outlined
            dense
            autocomplete="off"
            label="Client ID *"
            :rules="[requiredSecret]"
          />
          <q-input
            v-model="form.secret.client_secret"
            outlined
            dense
            type="password"
            autocomplete="new-password"
            label="Client Secret *"
            :rules="[requiredSecret]"
          />
        </template>
      </template>
    </q-form>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, reactive, ref, watch } from 'vue'
import type { QForm } from 'quasar'
import SweetDateTimePicker from '@/components/DateTime/SweetDateTimePicker.vue'
import FormDialogShell from '@/components/FormDialog/FormDialogShell.vue'
import type {
  CredentialDetail,
  CredentialSecret,
  CredentialType,
  ExternalSystemListItem,
} from '@/api/services/integration'

const { t } = useI18n({ useScope: 'global' })

type CredentialFormValue = {
  external_system_id: number | null
  credential_code: string
  name: string
  credential_type: CredentialType
  expires_at: string
  description: string
  secret: CredentialSecret
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    editData: CredentialDetail | null
    systems: ExternalSystemListItem[]
    rotateMode?: boolean
    loading?: boolean
  }>(),
  { rotateMode: false, loading: false },
)
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: CredentialFormValue): void
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const formRef = ref<QForm | null>(null)
const form = reactive<CredentialFormValue>(emptyForm())
const systemOptions = computed(() =>
  props.systems.map((item) => ({ label: `${item.name}（${item.system_code}）`, value: item.id })),
)
const typeOptions = [
  { label: 'Basic', value: 'basic' },
  { label: 'API Key', value: 'api_key' },
  { label: 'Bearer Token', value: 'bearer_token' },
  { label: 'OAuth Client', value: 'oauth_client' },
]
const requiredSecret = (value: string) =>
  Boolean(value?.trim()) || t('ui.pleaseEnterTheSecretContents')

function emptyForm(): CredentialFormValue {
  return {
    external_system_id: null,
    credential_code: '',
    name: '',
    credential_type: 'basic',
    expires_at: '',
    description: '',
    secret: {},
  }
}

function toLocalDateTime(value?: string) {
  if (!value) return ''
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return ''
  const pad2 = (part: number) => String(part).padStart(2, '0')
  return `${parsed.getFullYear()}-${pad2(parsed.getMonth() + 1)}-${pad2(parsed.getDate())} ${pad2(parsed.getHours())}:${pad2(parsed.getMinutes())}:${pad2(parsed.getSeconds())}`
}

watch(
  () => [props.modelValue, props.editData, props.rotateMode] as const,
  ([open, detail, rotate]) => {
    if (!open) {
      form.secret = {}
      return
    }
    Object.assign(
      form,
      detail
        ? {
            external_system_id: detail.external_system.id,
            credential_code: detail.credential_code,
            name: detail.name,
            credential_type: detail.credential_type,
            expires_at: toLocalDateTime(detail.expires_at),
            description: detail.description || '',
            secret: {},
          }
        : emptyForm(),
    )
    if (rotate) form.secret = {}
  },
  { immediate: true },
)

watch(
  () => form.credential_type,
  () => {
    form.secret = {}
  },
)

const submit = async () => {
  if (!(await formRef.value?.validate())) return
  emit('submit', { ...form, secret: { ...form.secret } })
}
</script>

<style scoped lang="scss">
.credential-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px 22px;
  padding: 4px 4px 18px;
}
.credential-form__notice,
.credential-form__wide {
  grid-column: 1 / -1;
}
.credential-form__notice {
  background: var(--app-primary-soft);
  color: inherit;
}
@media (max-width: 700px) {
  .credential-form {
    grid-template-columns: 1fr;
  }
  .credential-form__notice,
  .credential-form__wide {
    grid-column: auto;
  }
}
</style>
