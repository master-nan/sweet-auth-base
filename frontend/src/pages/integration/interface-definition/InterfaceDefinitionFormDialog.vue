<template>
  <form-dialog-shell
    v-model="visible"
    :title="editData ? '编辑接口定义' : '新增接口定义'"
    :subtitle="editData ? `${editData.interface_code} · v${editData.version}` : '创建接口草稿'"
    icon="api"
    submit-text="保存"
    :loading="loading || false"
    width="min(980px, calc(100vw - 48px))"
    @submit="submit"
  >
    <q-form ref="formRef" class="interface-form">
      <q-select
        v-model="form.external_system_id"
        outlined dense emit-value map-options
        :disable="Boolean(editData)"
        :options="systemOptions"
        label="所属外部系统 *"
        :rules="[(value) => Boolean(value) || '请选择所属外部系统']"
      />
      <q-input
        v-model="form.interface_code"
        outlined dense
        :disable="Boolean(editData)"
        label="接口编码 *"
        hint="小写字母开头，可使用数字和下划线"
        :rules="[(value) => /^[a-z][a-z0-9_]{1,63}$/.test(value || '') || '请输入合法接口编码']"
      />
      <q-input v-model="form.name" outlined dense label="接口名称 *" :rules="[(value) => Boolean(value?.trim()) || '请输入接口名称']" />
      <q-select v-model="form.protocol" outlined dense emit-value map-options :options="protocolOptions" label="协议 *" />
      <q-select v-model="form.http_method" outlined dense emit-value map-options :options="methodOptions" label="HTTP Method *" />
      <q-input
        v-model="form.relative_path"
        outlined dense
        label="相对路径 *"
        hint="仅填写以 / 开头的相对路径"
        :rules="[(value) => /^\/(?!\/)(?!.*(?:^|\/)\.\.?(?:\/|$))[^?#\s]*$/.test(value || '') || '请输入安全的相对路径']"
      />
      <q-input v-model.number="form.timeout_seconds" outlined dense type="number" min="1" max="300" label="超时（秒） *" />
      <q-input v-model.number="form.response_limit" outlined dense type="number" min="1024" max="104857600" label="响应大小限制（字节） *" />
      <q-input v-model="form.description" outlined dense type="textarea" autogrow class="interface-form__wide" label="描述" />
    </q-form>
    <template #footer-status>
      <span class="text-caption text-grey-7">技术契约启用后不可直接修改</span>
    </template>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { QForm } from 'quasar'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import type {
  ExternalSystemListItem,
  InterfaceDefinitionDetail,
  InterfaceHTTPMethod,
  InterfaceProtocol,
} from 'src/api/services/integration'

type InterfaceFormValue = {
  external_system_id: number | null
  interface_code: string
  name: string
  protocol: InterfaceProtocol
  http_method: InterfaceHTTPMethod
  relative_path: string
  timeout_seconds: number
  response_limit: number
  description: string
}

const props = withDefaults(defineProps<{
  modelValue: boolean
  editData: InterfaceDefinitionDetail | null
  systems: ExternalSystemListItem[]
  loading?: boolean
}>(), { loading: false })
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: InterfaceFormValue): void
}>()
const formRef = ref<QForm | null>(null)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const form = reactive<InterfaceFormValue>(emptyForm())
const systemOptions = computed(() => props.systems.map((item) => ({ label: `${item.name}（${item.system_code}）`, value: item.id })))
const protocolOptions = [{ label: 'HTTPS', value: 'https' }, { label: 'HTTP', value: 'http' }]
const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((value) => ({ label: value, value }))

function emptyForm(): InterfaceFormValue {
  return { external_system_id: null, interface_code: '', name: '', protocol: 'https', http_method: 'GET', relative_path: '/', timeout_seconds: 30, response_limit: 10485760, description: '' }
}

watch(
  () => [props.modelValue, props.editData] as const,
  ([open, detail]) => {
    if (!open) return
    Object.assign(form, detail ? {
      external_system_id: detail.external_system.id,
      interface_code: detail.interface_code,
      name: detail.name,
      protocol: detail.protocol,
      http_method: detail.http_method,
      relative_path: detail.relative_path,
      timeout_seconds: detail.timeout_seconds,
      response_limit: detail.response_limit,
      description: detail.description || '',
    } : emptyForm())
  },
  { immediate: true },
)

const submit = async () => {
  if (!(await formRef.value?.validate())) return
  emit('submit', { ...form })
}
</script>

<style scoped lang="scss">
.interface-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px 22px;
  padding: 4px 4px 18px;
}
.interface-form__wide { grid-column: 1 / -1; }
@media (max-width: 700px) {
  .interface-form { grid-template-columns: 1fr; }
  .interface-form__wide { grid-column: auto; }
}
</style>
