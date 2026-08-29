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
      <q-select
        v-model="form.credential_id"
        outlined dense emit-value map-options
        :disable="!form.external_system_id"
        :options="credentialOptions"
        label="认证凭证"
        hint="不需要认证时选择“不使用认证凭证”"
      />
      <q-select
        v-model="form.retry_policy_id"
        outlined dense emit-value map-options
        :options="retryPolicyOptions"
        label="重试策略"
        hint="无需自动重试时选择“不自动重试”"
      />
      <q-select v-model="form.protocol" outlined dense emit-value map-options :options="protocolOptions" label="协议 *" />
      <q-select v-model="form.http_method" outlined dense emit-value map-options :options="methodOptions" label="HTTP Method *" />
      <q-input
        v-model="form.relative_path"
        outlined dense
        label="相对路径 *"
        hint="仅填写以 / 开头的相对路径"
        :rules="[(value) => /^\/(?!\/)(?!.*(?:^|\/)\.\.?(?:\/|$))[^?#\s]*$/.test(value || '') || '请输入安全的相对路径']"
      />
      <q-input
        v-model.number="form.timeout_seconds"
        outlined dense type="number" min="1" :max="MAX_TIMEOUT_SECONDS"
        label="请求超时（秒） *"
        hint="平台允许 1 至 120 秒"
        :rules="[(value) => value >= 1 && value <= MAX_TIMEOUT_SECONDS || '请求超时必须在 1 至 120 秒之间']"
      />
      <q-input
        outlined dense type="number" min="1" :max="MAX_RESPONSE_KIB"
        label="响应大小限制（KiB） *"
        :model-value="responseLimitKiB"
        hint="1 KiB 至 65,536 KiB（64 MiB）"
        :rules="[() => form.response_limit >= MIN_RESPONSE_BYTES && form.response_limit <= MAX_RESPONSE_BYTES || '响应大小必须在 1 KiB 至 64 MiB 之间']"
        @update:model-value="updateResponseLimitKiB"
      />
      <section class="interface-form__wide input-contract-section">
        <div class="row items-center justify-between q-col-gutter-sm">
          <div>
            <div class="text-subtitle2">请求参数</div>
            <div class="text-caption text-grey-7">声明 Path、Query、Header 或 JSON Body 中允许填写的参数。</div>
          </div>
          <q-btn flat dense color="primary" icon="add" label="添加参数" @click="addParameter" />
        </div>

        <q-banner v-if="!form.input_contract.parameters.length" dense rounded class="input-contract-section__empty">
          当前接口没有业务入参。认证信息由“认证凭证”注入，不要在参数中声明 Token 或密码。
        </q-banner>

        <div
          v-for="(parameter, index) in form.input_contract.parameters"
          :key="index"
          class="input-parameter-row"
        >
          <q-select
            v-if="parameter.location === 'header'"
            v-model="parameter.code"
            outlined dense emit-value map-options
            :options="headerCodeOptions"
            label="参数编码 *"
            :rules="[requiredRule]"
          />
          <q-input
            v-else
            v-model="parameter.code"
            outlined dense
            label="参数编码 *"
            hint="Path参数需与相对路径中的 {占位符} 同名"
            :rules="[parameterCodeRule]"
          />
          <q-select
            v-model="parameter.location"
            outlined dense emit-value map-options
            :options="locationOptions"
            label="位置 *"
            :rules="[parameterLocationRule]"
            @update:model-value="onParameterLocationChanged(parameter)"
          />
          <q-select
            v-model="parameter.data_type"
            outlined dense emit-value map-options
            :options="dataTypeOptions(parameter.location)"
            label="数据类型 *"
          />
          <q-input
            v-model.number="parameter.max_length"
            outlined dense type="number" min="1" :max="maxParameterLength(parameter.location)"
            label="最大长度 *"
            :rules="[(value) => Number(value) >= 1 && Number(value) <= maxParameterLength(parameter.location) || `长度必须在 1 至 ${maxParameterLength(parameter.location)} 之间`]"
          />
          <div class="input-parameter-row__flags">
            <q-toggle v-model="parameter.required" label="必填" :disable="parameter.location === 'path'" />
            <q-toggle
              v-model="parameter.allow_multiple"
              label="允许多值"
              :disable="parameter.location !== 'query'"
            />
            <q-btn flat round dense color="negative" icon="delete_outline" aria-label="删除参数" @click="removeParameter(index)">
              <q-tooltip>删除参数</q-tooltip>
            </q-btn>
          </div>
        </div>
      </section>
      <q-banner class="interface-form__wide interface-form__response-note" dense rounded>
        接口定义只约束请求方法、路径和入参。HTTP 响应由同步任务选择的 Consumer 解析并写入对应业务模块。
      </q-banner>
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
  CredentialListItem,
  InterfaceDefinitionDetail,
  InterfaceHTTPMethod,
  InterfaceInputContract,
  InterfaceInputDataType,
  InterfaceInputLocation,
  InterfaceInputParameter,
  InterfaceProtocol,
  RetryPolicyListItem,
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
  credential_id: number | null
  retry_policy_id: number | null
  input_contract: InterfaceInputContract
}

const props = withDefaults(defineProps<{
  modelValue: boolean
  editData: InterfaceDefinitionDetail | null
  systems: ExternalSystemListItem[]
  credentials: CredentialListItem[]
  retryPolicies?: RetryPolicyListItem[]
  loading?: boolean
}>(), { retryPolicies: () => [], loading: false })
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: InterfaceFormValue): void
}>()
const formRef = ref<QForm | null>(null)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const form = reactive<InterfaceFormValue>(emptyForm())
const systemOptions = computed(() => props.systems.map((item) => ({ label: `${item.name}（${item.system_code}）`, value: item.id })))
const credentialOptions = computed(() => [
  { label: '不使用认证凭证', value: null },
  ...props.credentials
    .filter((item) => item.external_system.id === form.external_system_id && item.effective_status === 'active')
    .map((item) => ({ label: `${item.name}（${item.credential_code}）`, value: item.id })),
])
const retryPolicyOptions = computed(() => [
  { label: '不自动重试', value: null },
  ...props.retryPolicies
    .filter((item) => item.status === 'enabled')
    .map((item) => ({ label: `${item.policy_name}（${item.policy_code} · v${item.version}）`, value: item.id })),
])
const protocolOptions = [{ label: 'HTTPS', value: 'https' }, { label: 'HTTP', value: 'http' }]
const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((value) => ({ label: value, value }))
const locationOptions = [
  { label: 'Path 路径', value: 'path' },
  { label: 'Query 查询参数', value: 'query' },
  { label: 'Header 请求头', value: 'header' },
  { label: 'JSON Body', value: 'body' },
]
const headerCodeOptions = ['Accept', 'Accept-Language', 'User-Agent', 'X-Correlation-ID'].map((value) => ({ label: value, value }))
const inputDataTypeLabels: Record<InterfaceInputDataType, string> = {
  string: '字符串', integer: '整数', number: '数字', boolean: '布尔', object: '对象', array: '数组',
}
const MAX_TIMEOUT_SECONDS = 120
const MIN_RESPONSE_BYTES = 1024
const MAX_RESPONSE_BYTES = 64 * 1024 * 1024
const KIBIBYTE = 1024
const MAX_RESPONSE_KIB = MAX_RESPONSE_BYTES / KIBIBYTE
const responseLimitKiB = computed(() => form.response_limit / KIBIBYTE)

function updateResponseLimitKiB(value: string | number | null) {
  const numeric = Number(value)
  form.response_limit = Number.isFinite(numeric) ? Math.round(numeric * KIBIBYTE) : 0
}

function emptyForm(): InterfaceFormValue {
  return { external_system_id: null, interface_code: '', name: '', protocol: 'https', http_method: 'GET', relative_path: '/', credential_id: null, retry_policy_id: null, timeout_seconds: 30, response_limit: 10485760, description: '', input_contract: { version: 1, parameters: [] } }
}

const requiredRule = (value: unknown) => Boolean(value) || '此项必填'
const parameterCodeRule = (value: string) => /^[A-Za-z][A-Za-z0-9_]{0,63}$/.test(value || '') || '请输入合法参数编码'
const parameterLocationRule = (value: InterfaceInputLocation) => value !== 'body' || form.http_method !== 'GET' || 'GET接口不能声明JSON Body参数'
const maxParameterLength = (location: InterfaceInputLocation) => location === 'path' ? 256 : location === 'query' ? 2048 : 4096
const dataTypeOptions = (location: InterfaceInputLocation) => {
  const values: InterfaceInputDataType[] = location === 'path' || location === 'header'
    ? ['string']
    : location === 'query'
      ? ['string', 'integer', 'number', 'boolean']
      : ['string', 'integer', 'number', 'boolean', 'object', 'array']
  return values.map((value) => ({ label: inputDataTypeLabels[value], value }))
}
const defaultParameterMaxLength = (location: InterfaceInputLocation) => maxParameterLength(location)
const emptyParameter = (): InterfaceInputParameter => ({
  code: '', location: 'query', data_type: 'string', required: false, allow_multiple: false, sensitive: false, max_length: 2048,
})
const addParameter = () => form.input_contract.parameters.push(emptyParameter())
const removeParameter = (index: number) => form.input_contract.parameters.splice(index, 1)
const onParameterLocationChanged = (parameter: InterfaceInputParameter) => {
  const availableTypes = dataTypeOptions(parameter.location).map((item) => item.value)
  if (!availableTypes.includes(parameter.data_type)) parameter.data_type = availableTypes[0] || 'string'
  parameter.max_length = defaultParameterMaxLength(parameter.location)
  if (parameter.location === 'path') {
    parameter.required = true
    parameter.allow_multiple = false
  } else if (parameter.location !== 'query') parameter.allow_multiple = false
  if (parameter.location === 'header' && !headerCodeOptions.some((item) => item.value === parameter.code)) parameter.code = ''
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
      credential_id: detail.credential_id || null,
      retry_policy_id: detail.retry_policy_id || null,
      timeout_seconds: detail.timeout_seconds,
      response_limit: detail.response_limit,
      description: detail.description || '',
      input_contract: {
        version: 1,
        parameters: (detail.input_contract?.parameters || []).map((parameter) => ({ ...parameter })),
      },
    } : emptyForm())
  },
  { immediate: true },
)

watch(() => form.external_system_id, (systemID, previous) => {
  if (previous != null && systemID !== previous) form.credential_id = null
})

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
.input-contract-section {
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface-muted);
}
.input-contract-section__empty, .interface-form__response-note { background: var(--app-primary-soft); }
.input-parameter-row {
  display: grid;
  grid-template-columns: minmax(180px, 1.4fr) repeat(3, minmax(130px, 1fr)) minmax(250px, auto);
  gap: 10px;
  align-items: start;
}
.input-parameter-row__flags { min-height: 40px; display: flex; align-items: center; gap: 4px; }
@media (max-width: 700px) {
  .interface-form { grid-template-columns: 1fr; }
  .interface-form__wide { grid-column: auto; }
  .input-parameter-row { grid-template-columns: 1fr; }
}
</style>
