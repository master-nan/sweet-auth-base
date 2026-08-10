<template>
  <form-dialog-shell
    v-model="visible"
    :title="editData ? '编辑同步任务' : '新增同步任务'"
    :subtitle="editData ? `${editData.task_code} · v${editData.version}` : '创建版本 1 草稿'"
    icon="sync_alt"
    :submit-text="editData ? '保存' : '创建'"
    :loading="loading || false"
    width="min(1080px, calc(100vw - 40px))"
    @submit="submit"
  >
    <q-form ref="formRef" class="sync-task-form">
      <q-input v-model="form.task_code" outlined dense :disable="Boolean(editData)" label="任务编码 *" :rules="[taskCodeRule]" />
      <q-input v-model="form.task_name" outlined dense label="任务名称 *" :rules="[requiredRule]" />
      <q-select v-model="form.external_system_id" outlined dense emit-value map-options :options="systemOptions" label="外部系统 *" :rules="[positiveRule]" @update:model-value="onSystemChanged" />
      <q-select v-model="form.interface_definition_id" outlined dense emit-value map-options :options="interfaceOptions" label="接口定义版本 *" :rules="[positiveRule]" @update:model-value="onInterfaceChanged" />
      <q-select v-model="consumerKey" outlined dense emit-value map-options :options="consumerOptions" label="Consumer *" :rules="[requiredRule]" />
      <q-select v-model="form.schedule_type" outlined dense emit-value map-options :options="scheduleOptions" label="调度方式 *" />
      <q-input v-if="form.schedule_type === 'cron'" v-model="form.cron_expression" outlined dense label="Cron（五段式） *" hint="分钟 小时 日 月 星期" :rules="[cronRule]" />
      <q-input v-model="form.timezone" outlined dense label="IANA 时区 *" hint="例如 Asia/Shanghai 或 UTC" :rules="[requiredRule]" />
      <q-select v-model="form.checkpoint_mode" outlined dense emit-value map-options :options="checkpointOptions" label="Checkpoint 模式 *" />
      <q-input v-if="form.checkpoint_mode === 'timestamp'" v-model="initialCheckpointLocal" outlined dense type="datetime-local" label="初始 Checkpoint *" :rules="[requiredRule]" />
      <q-input v-if="form.checkpoint_mode === 'timestamp'" v-model.number="form.lookback_seconds" outlined dense type="number" min="0" max="604800" label="Lookback（秒）" :rules="[lookbackRule]" />
      <q-input v-if="form.checkpoint_mode === 'timestamp'" v-model.number="form.window_slice_seconds" outlined dense type="number" min="60" max="604800" label="窗口切片（秒） *" :rules="[sliceRule]" />

      <div class="sync-task-form__wide text-subtitle2">受控输入计划</div>
      <q-select
        v-model="selectedStaticKeys"
        class="sync-task-form__wide"
        outlined dense multiple emit-value map-options use-chips
        :options="staticParameterOptions"
        label="静态参数"
        hint="仅可选择接口契约中声明的非敏感参数"
      />
      <template v-for="parameter in selectedStaticParameters" :key="parameterKey(parameter)">
        <q-toggle v-if="parameter.data_type === 'boolean'" v-model="staticValues[parameterKey(parameter)]" :label="parameterLabel(parameter)" />
        <q-input
          v-else
          :model-value="String(staticValues[parameterKey(parameter)] ?? '')"
          outlined
          dense
          :label="parameterLabel(parameter)"
          :hint="parameter.allow_multiple ? '多个值使用英文逗号分隔' : undefined"
          @update:model-value="setStaticTextValue(parameter, $event)"
        />
      </template>

      <q-select v-if="form.checkpoint_mode === 'timestamp'" v-model="windowStartKey" outlined dense emit-value map-options :options="windowParameterOptions" label="窗口开始参数 *" :rules="[requiredRule]" />
      <q-select v-if="form.checkpoint_mode === 'timestamp'" v-model="windowStartFormat" outlined dense emit-value map-options :options="formatOptions(windowStartKey)" label="窗口开始格式 *" />
      <q-select v-if="form.checkpoint_mode === 'timestamp'" v-model="windowEndKey" outlined dense emit-value map-options :options="windowParameterOptions" label="窗口结束参数 *" :rules="[requiredRule]" />
      <q-select v-if="form.checkpoint_mode === 'timestamp'" v-model="windowEndFormat" outlined dense emit-value map-options :options="formatOptions(windowEndKey)" label="窗口结束格式 *" />
      <q-input v-model="form.description" class="sync-task-form__wide" outlined dense type="textarea" autogrow maxlength="512" label="描述" />
    </q-form>
    <template #footer-status><span class="text-caption text-grey-7">任务启用后技术配置不可直接修改；Checkpoint 由服务端维护</span></template>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { QForm } from 'quasar'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import {
  type ExternalSystemListItem,
  type InterfaceDefinitionDetail,
  type InterfaceDefinitionListItem,
  type InterfaceInputParameter,
  type SyncCheckpointMode,
  type SyncConsumerMetadata,
  type SyncExecutionInputPlan,
  type SyncScheduleType,
  type SyncTaskCreateRequest,
  type SyncTaskEdit,
  type SyncTimeFormat,
  useIntegrationApi,
} from 'src/api/services/integration'

export type SyncTaskFormValue = SyncTaskCreateRequest
const props = withDefaults(defineProps<{
  modelValue: boolean
  editData: SyncTaskEdit | null
  systems: ExternalSystemListItem[]
  interfaces: InterfaceDefinitionListItem[]
  consumers: SyncConsumerMetadata[]
  loading?: boolean
}>(), { loading: false })
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void; (event: 'submit', value: SyncTaskFormValue): void }>()
const api = useIntegrationApi()
const formRef = ref<QForm | null>(null)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const definition = ref<InterfaceDefinitionDetail | null>(null)
const selectedStaticKeys = ref<string[]>([])
const staticValues = reactive<Record<string, string | boolean>>({})
const consumerKey = ref('')
const windowStartKey = ref('')
const windowEndKey = ref('')
const windowStartFormat = ref<SyncTimeFormat>('rfc3339')
const windowEndFormat = ref<SyncTimeFormat>('rfc3339')
const initialCheckpointLocal = ref('')
const form = reactive<SyncTaskCreateRequest>(emptyForm())

const scheduleOptions: { label: string; value: SyncScheduleType }[] = [{ label: '仅手工触发', value: 'none' }, { label: 'Cron 定时', value: 'cron' }]
const checkpointOptions: { label: string; value: SyncCheckpointMode }[] = [{ label: '无 Checkpoint', value: 'none' }, { label: '时间戳', value: 'timestamp' }]
const systemOptions = computed(() => props.systems.map((item) => ({ label: `${item.name} (${item.system_code})`, value: item.id })))
const interfaceOptions = computed(() => props.interfaces.filter((item) => item.external_system.id === form.external_system_id && item.status === 'enabled').map((item) => ({ label: `${item.name} (${item.interface_code} · v${item.version})`, value: item.id })))
const consumerOptions = computed(() => props.consumers.map((item) => ({ label: `${item.name || item.code} (${item.code} · v${item.version})`, value: `${item.code}@${item.version}` })))
const parameters = computed(() => definition.value?.input_contract?.parameters || [])
const eligibleParameters = computed(() => parameters.value.filter((item) => !item.sensitive && !['authorization', 'cookie', 'idempotency-key'].includes(item.code.toLowerCase())))
const parameterKey = (parameter: InterfaceInputParameter) => `${parameter.location}:${parameter.code}`
const parameterLabel = (parameter: InterfaceInputParameter) => `${parameter.name || parameter.code} · ${parameter.location}/${parameter.data_type}${parameter.required ? ' *' : ''}`
const staticParameterOptions = computed(() => eligibleParameters.value.filter((item) => !['object', 'array'].includes(item.data_type)).map((item) => ({ label: parameterLabel(item), value: parameterKey(item) })))
const windowParameterOptions = computed(() => eligibleParameters.value.filter((item) => !item.allow_multiple && ['string', 'integer', 'number'].includes(item.data_type)).map((item) => ({ label: parameterLabel(item), value: parameterKey(item) })))
const selectedStaticParameters = computed(() => eligibleParameters.value.filter((item) => selectedStaticKeys.value.includes(parameterKey(item))))
const formatOptions = (key: string) => {
  const parameter = eligibleParameters.value.find((item) => parameterKey(item) === key)
  const values: SyncTimeFormat[] = parameter?.data_type === 'string' ? ['rfc3339', 'unix_seconds', 'unix_milliseconds'] : ['unix_seconds', 'unix_milliseconds']
  return values.map((value) => ({ label: value === 'rfc3339' ? 'RFC 3339' : value === 'unix_seconds' ? 'Unix 秒' : 'Unix 毫秒', value }))
}

function emptyForm(): SyncTaskCreateRequest {
  return { task_code: '', task_name: '', description: '', external_system_id: 0, interface_definition_id: 0, consumer_code: '', consumer_version: 0, schedule_type: 'none', cron_expression: '', timezone: 'UTC', checkpoint_mode: 'none', lookback_seconds: 0, window_slice_seconds: 0, input_plan: emptyPlan() }
}
function emptyPlan(): SyncExecutionInputPlan { return { version: 1, static_input: { path_params: {}, query_params: {}, headers: {} } } }
const taskCodeRule = (value: string) => /^[a-z][a-z0-9_]{0,63}$/.test(value || '') || '请输入合法任务编码'
const requiredRule = (value: unknown) => Boolean(value) || '此项必填'
const positiveRule = (value: number) => value > 0 || '请选择有效项目'
const cronRule = (value: string) => value.trim().split(/\s+/).length === 5 || 'Cron 必须为五段式'
const lookbackRule = (value: number) => value >= 0 && value <= 604800 || 'Lookback 必须在 0 至 604800 秒之间'
const sliceRule = (value: number) => value >= 60 && value <= 604800 || '切片必须在 60 至 604800 秒之间'

async function loadDefinition(id: number) {
  definition.value = id ? (await api.getInterfaceDefinition(id)).data || null : null
}
function onSystemChanged() { form.interface_definition_id = 0; resetPlan(); definition.value = null }
async function onInterfaceChanged(value: number | null) { resetPlan(); await loadDefinition(Number(value) || 0) }
function resetPlan() { selectedStaticKeys.value = []; Object.keys(staticValues).forEach((key) => delete staticValues[key]); windowStartKey.value = ''; windowEndKey.value = '' }
function setStaticTextValue(parameter: InterfaceInputParameter, value: string | number | null) { staticValues[parameterKey(parameter)] = String(value ?? '') }
function parseStaticValue(parameter: InterfaceInputParameter, raw: string | boolean): unknown {
  if (parameter.data_type === 'boolean') return Boolean(raw)
  const text = String(raw ?? '')
  if (parameter.data_type === 'integer') return Number.parseInt(text, 10)
  if (parameter.data_type === 'number') return Number(text)
  return text
}
function buildInputPlan(): SyncExecutionInputPlan {
  const plan = emptyPlan()
  for (const parameter of selectedStaticParameters.value) {
    const raw = staticValues[parameterKey(parameter)] ?? ''
    const values = parameter.allow_multiple ? String(raw).split(',').map((item) => item.trim()) : [String(raw)]
    if (parameter.location === 'path') plan.static_input.path_params[parameter.code] = String(raw)
    if (parameter.location === 'query') plan.static_input.query_params[parameter.code] = values
    if (parameter.location === 'header') plan.static_input.headers[parameter.code] = values
    if (parameter.location === 'body') (plan.static_input.json_body ||= {})[parameter.code] = parseStaticValue(parameter, raw)
  }
  if (form.checkpoint_mode === 'timestamp') {
    const [startLocation, ...startCode] = windowStartKey.value.split(':')
    const [endLocation, ...endCode] = windowEndKey.value.split(':')
    plan.window_start_binding = { location: startLocation as InterfaceInputParameter['location'], code: startCode.join(':'), format: windowStartFormat.value }
    plan.window_end_binding = { location: endLocation as InterfaceInputParameter['location'], code: endCode.join(':'), format: windowEndFormat.value }
  }
  return plan
}
function applyPlan(plan: SyncExecutionInputPlan) {
  resetPlan()
  for (const parameter of eligibleParameters.value) {
    const key = parameterKey(parameter)
    let value: unknown
    if (parameter.location === 'path') value = plan.static_input.path_params?.[parameter.code]
    if (parameter.location === 'query') value = plan.static_input.query_params?.[parameter.code]
    if (parameter.location === 'header') value = plan.static_input.headers?.[parameter.code]
    if (parameter.location === 'body') value = plan.static_input.json_body?.[parameter.code]
    if (value !== undefined) { selectedStaticKeys.value.push(key); staticValues[key] = Array.isArray(value) ? value.join(',') : typeof value === 'boolean' ? value : String(value) }
  }
  if (plan.window_start_binding) { windowStartKey.value = `${plan.window_start_binding.location}:${plan.window_start_binding.code}`; windowStartFormat.value = plan.window_start_binding.format }
  if (plan.window_end_binding) { windowEndKey.value = `${plan.window_end_binding.location}:${plan.window_end_binding.code}`; windowEndFormat.value = plan.window_end_binding.format }
}

watch(() => [props.modelValue, props.editData] as const, async ([open, edit]) => {
  if (!open) return
  Object.assign(form, edit ? { task_code: edit.task_code, task_name: edit.task_name, description: edit.description, external_system_id: edit.external_system.id, interface_definition_id: edit.interface_definition.id, consumer_code: edit.consumer.code, consumer_version: edit.consumer.version, schedule_type: edit.schedule_type, cron_expression: edit.cron_summary || '', timezone: edit.timezone, checkpoint_mode: edit.checkpoint_mode, initial_checkpoint_at: edit.initial_checkpoint_at, lookback_seconds: edit.lookback_seconds, window_slice_seconds: edit.window_slice_seconds, input_plan: edit.input_plan } : emptyForm())
  consumerKey.value = form.consumer_code ? `${form.consumer_code}@${form.consumer_version}` : ''
  initialCheckpointLocal.value = form.initial_checkpoint_at ? form.initial_checkpoint_at.slice(0, 16) : ''
  await loadDefinition(form.interface_definition_id)
  applyPlan(edit?.input_plan || emptyPlan())
}, { immediate: true })
watch(() => form.schedule_type, (value) => { if (value === 'none') form.cron_expression = '' })
watch(() => form.checkpoint_mode, (value) => { if (value === 'none') { delete form.initial_checkpoint_at; form.lookback_seconds = 0; form.window_slice_seconds = 0; windowStartKey.value = ''; windowEndKey.value = '' } else if (!form.window_slice_seconds) form.window_slice_seconds = 3600 })
watch(consumerKey, (value) => { const [code, version] = value.split('@'); form.consumer_code = code || ''; form.consumer_version = Number(version) || 0 })

async function submit() {
  if (!(await formRef.value?.validate())) return
  if (form.checkpoint_mode === 'timestamp' && initialCheckpointLocal.value) form.initial_checkpoint_at = new Date(initialCheckpointLocal.value).toISOString()
  else delete form.initial_checkpoint_at
  emit('submit', { ...form, input_plan: buildInputPlan() })
}
</script>

<style scoped lang="scss">
.sync-task-form { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px 22px; padding: 4px 4px 18px; }
.sync-task-form__wide { grid-column: 1 / -1; }
@media (max-width: 760px) { .sync-task-form { grid-template-columns: 1fr; } .sync-task-form__wide { grid-column: auto; } }
</style>
