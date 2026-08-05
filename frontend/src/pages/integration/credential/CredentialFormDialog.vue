<template>
  <form-dialog-shell
    v-model="visible"
    :title="rotateMode ? '轮换凭证' : editData ? '编辑集成凭证' : '新增集成凭证'"
    :subtitle="editData ? editData.credential_code : '秘密提交后不会再次展示'"
    icon="key"
    :submit-text="rotateMode ? '确认轮换' : editData ? '保存' : '创建'"
    :loading="loading || false"
    width="min(900px, calc(100vw - 48px))"
    @submit="submit"
  >
    <q-form ref="formRef" class="credential-form">
      <template v-if="!rotateMode">
        <q-select v-model="form.external_system_id" outlined dense emit-value map-options :disable="Boolean(editData)" :options="systemOptions" label="所属外部系统 *" :rules="[(value) => Boolean(value) || '请选择所属外部系统']" />
        <q-input v-model="form.credential_code" outlined dense :disable="Boolean(editData)" label="凭证编码 *" hint="小写字母开头，可使用数字和下划线" :rules="[(value) => /^[a-z][a-z0-9_]{1,63}$/.test(value || '') || '请输入合法凭证编码']" />
        <q-input v-model="form.name" outlined dense label="凭证名称 *" :rules="[(value) => Boolean(value?.trim()) || '请输入凭证名称']" />
        <q-select v-model="form.credential_type" outlined dense emit-value map-options :disable="Boolean(editData)" :options="typeOptions" label="凭证类型 *" />
        <q-input v-model="form.expires_at" outlined dense type="datetime-local" label="有效期" clearable />
        <q-input v-model="form.description" outlined dense type="textarea" autogrow label="描述" />
      </template>

      <template v-if="!editData || rotateMode">
        <q-banner class="credential-form__notice" rounded>
          <template #avatar><q-icon name="shield" color="primary" /></template>
          密钥只用于本次{{ rotateMode ? '轮换' : '创建' }}，提交后页面不会显示、复制或恢复原值。
        </q-banner>
        <template v-if="form.credential_type === 'basic'">
          <q-input v-model="form.secret.username" outlined dense autocomplete="off" label="用户名 *" :rules="[requiredSecret]" />
          <q-input v-model="form.secret.password" outlined dense type="password" autocomplete="new-password" label="密码 *" :rules="[requiredSecret]" />
        </template>
        <q-input v-else-if="form.credential_type === 'api_key'" v-model="form.secret.api_key" class="credential-form__wide" outlined dense type="password" autocomplete="new-password" label="API Key *" :rules="[requiredSecret]" />
        <q-input v-else-if="form.credential_type === 'bearer_token'" v-model="form.secret.token" class="credential-form__wide" outlined dense type="password" autocomplete="new-password" label="Bearer Token *" :rules="[requiredSecret]" />
        <template v-else>
          <q-input v-model="form.secret.client_id" outlined dense autocomplete="off" label="Client ID *" :rules="[requiredSecret]" />
          <q-input v-model="form.secret.client_secret" outlined dense type="password" autocomplete="new-password" label="Client Secret *" :rules="[requiredSecret]" />
        </template>
      </template>
    </q-form>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { QForm } from 'quasar'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import type { CredentialDetail, CredentialSecret, CredentialType, ExternalSystemListItem } from 'src/api/services/integration'

type CredentialFormValue = {
  external_system_id: number | null
  credential_code: string
  name: string
  credential_type: CredentialType
  expires_at: string
  description: string
  secret: CredentialSecret
}

const props = withDefaults(defineProps<{
  modelValue: boolean
  editData: CredentialDetail | null
  systems: ExternalSystemListItem[]
  rotateMode?: boolean
  loading?: boolean
}>(), { rotateMode: false, loading: false })
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: CredentialFormValue): void
}>()
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const formRef = ref<QForm | null>(null)
const form = reactive<CredentialFormValue>(emptyForm())
const systemOptions = computed(() => props.systems.map((item) => ({ label: `${item.name}（${item.system_code}）`, value: item.id })))
const typeOptions = [
  { label: 'Basic', value: 'basic' }, { label: 'API Key', value: 'api_key' },
  { label: 'Bearer Token', value: 'bearer_token' }, { label: 'OAuth Client', value: 'oauth_client' },
]
const requiredSecret = (value: string) => Boolean(value?.trim()) || '请输入秘密内容'

function emptyForm(): CredentialFormValue {
  return { external_system_id: null, credential_code: '', name: '', credential_type: 'basic', expires_at: '', description: '', secret: {} }
}

watch(() => [props.modelValue, props.editData, props.rotateMode] as const, ([open, detail, rotate]) => {
  if (!open) {
    form.secret = {}
    return
  }
  Object.assign(form, detail ? {
    external_system_id: detail.external_system.id,
    credential_code: detail.credential_code,
    name: detail.name,
    credential_type: detail.credential_type,
    expires_at: detail.expires_at ? detail.expires_at.slice(0, 16) : '',
    description: detail.description || '',
    secret: {},
  } : emptyForm())
  if (rotate) form.secret = {}
}, { immediate: true })

watch(() => form.credential_type, () => { form.secret = {} })

const submit = async () => {
  if (!(await formRef.value?.validate())) return
  emit('submit', { ...form, secret: { ...form.secret } })
}
</script>

<style scoped lang="scss">
.credential-form { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px 22px; padding: 4px 4px 18px; }
.credential-form__notice, .credential-form__wide { grid-column: 1 / -1; }
.credential-form__notice { background: var(--app-primary-soft); color: inherit; }
@media (max-width: 700px) { .credential-form { grid-template-columns: 1fr; } .credential-form__notice, .credential-form__wide { grid-column: auto; } }
</style>
