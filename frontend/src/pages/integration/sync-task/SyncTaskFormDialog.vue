<template>
  <form-dialog-shell
    v-model="visible"
    :title="editData ? t('ui.editSyncTasks') : t('ui.createSyncTaskTitle')"
    :subtitle="
      editData ? `${editData.task_code} · v${editData.version}` : t('ui.createDraftVersion1')
    "
    icon="sync_alt"
    :submit-text="editData ? t('ui.save') : t('ui.createRecord')"
    :loading="loading || false"
    width="min(1080px, calc(100vw - 40px))"
    @submit="submit"
  >
    <q-form ref="formRef" class="sync-task-form">
      <q-input
        v-model="form.task_code"
        outlined
        dense
        :disable="Boolean(editData)"
        :label="t('ui.taskCode')"
        :rules="[taskCodeRule]"
      />
      <q-input
        v-model="form.task_name"
        outlined
        dense
        :label="t('ui.taskName')"
        :rules="[requiredRule]"
      />
      <q-select
        v-model="form.external_system_id"
        outlined
        dense
        emit-value
        map-options
        :options="systemOptions"
        :label="t('ui.externalSystemRequiredLabel')"
        :rules="[positiveRule]"
        @update:model-value="onSystemChanged"
      />
      <q-select
        v-model="form.interface_definition_id"
        outlined
        dense
        emit-value
        map-options
        :options="interfaceOptions"
        :label="t('ui.interfaceDefinitionVersion')"
        :rules="[positiveRule]"
        @update:model-value="onInterfaceChanged"
      />
      <q-select
        v-model="consumerKey"
        outlined
        dense
        emit-value
        map-options
        :options="consumerOptions"
        label="Consumer *"
        :rules="[requiredRule]"
      >
        <template #no-option>
          <q-item>
            <q-item-section class="text-grey-7">
              {{ t('ui.consumerPleaseCheckBackendSyncConfigurationFirst') }}
            </q-item-section>
          </q-item>
        </template>
      </q-select>
      <q-select
        v-model="form.schedule_type"
        outlined
        dense
        emit-value
        map-options
        :options="scheduleOptions"
        :label="t('ui.scheduleMode')"
      />
      <q-input
        v-if="form.schedule_type === 'cron'"
        v-model="form.cron_expression"
        outlined
        dense
        :label="t('ui.cronFiveFields')"
        :hint="t('ui.formatMinutesHoursDaysAndMonthsEG02')"
        :rules="[cronRule]"
      >
        <template #append>
          <q-btn flat round dense icon="help_outline" :aria-label="t('ui.viewCronExample')">
            <q-tooltip>{{ t('ui.viewCronExample') }}</q-tooltip>
            <q-menu anchor="bottom right" self="top right">
              <q-list dense style="min-width: 320px">
                <q-item-label header>{{ t('ui.cronExamples') }}</q-item-label>
                <q-item
                  v-for="example in cronExamples"
                  :key="example.value"
                  v-close-popup
                  clickable
                  @click="form.cron_expression = example.value"
                >
                  <q-item-section>
                    <q-item-label class="text-mono">{{ example.value }}</q-item-label>
                    <q-item-label caption>{{ example.label }}</q-item-label>
                  </q-item-section>
                </q-item>
              </q-list>
            </q-menu>
          </q-btn>
        </template>
      </q-input>
      <q-input
        v-model="form.timezone"
        outlined
        dense
        :label="t('ui.ianaTimeZone')"
        :hint="t('ui.asiaShanghaiOrUtc')"
        :rules="[requiredRule]"
      />
      <q-select
        v-model="form.checkpoint_mode"
        outlined
        dense
        emit-value
        map-options
        :options="checkpointOptions"
        :label="t('ui.checkpointMode')"
      />
      <sweet-date-time-picker
        v-if="form.checkpoint_mode === 'timestamp'"
        v-model="initialCheckpointLocal"
        type="datetime"
        :label="t('ui.initialCheckpoint')"
        :rules="[requiredRule]"
      />
      <q-input
        v-if="form.checkpoint_mode === 'timestamp'"
        v-model.number="form.lookback_seconds"
        outlined
        dense
        type="number"
        min="0"
        max="604800"
        :label="t('ui.lookbackSeconds')"
        :rules="[lookbackRule]"
      />
      <q-input
        v-if="form.checkpoint_mode === 'timestamp'"
        v-model.number="form.window_slice_seconds"
        outlined
        dense
        type="number"
        min="60"
        max="604800"
        :label="t('ui.slicesOfWindowsSeconds')"
        :rules="[sliceRule]"
      />

      <q-select
        v-if="form.checkpoint_mode === 'timestamp'"
        v-model="windowMode"
        outlined
        dense
        emit-value
        map-options
        :options="windowModeOptions"
        :label="t('ui.sourceWindowContract')"
      />

      <div class="sync-task-form__wide text-subtitle2">{{ t('ui.controlledInputSchedule') }}</div>
      <q-select
        v-model="selectedStaticKeys"
        class="sync-task-form__wide"
        outlined
        dense
        multiple
        emit-value
        map-options
        use-chips
        :options="staticParameterOptions"
        :label="t('ui.staticParameters')"
        :hint="t('ui.onlyNonSensitiveParametersDeclaredInTheApiContractCan')"
      />
      <template v-for="parameter in selectedStaticParameters" :key="parameterKey(parameter)">
        <q-toggle
          v-if="parameter.data_type === 'boolean'"
          v-model="staticValues[parameterKey(parameter)]"
          :label="parameterLabel(parameter)"
        />
        <q-input
          v-else
          :model-value="String(staticValues[parameterKey(parameter)] ?? '')"
          outlined
          dense
          :label="parameterLabel(parameter)"
          :hint="parameter.allow_multiple ? t('ui.multipleValuesSeparatedByCommas') : undefined"
          @update:model-value="setStaticTextValue(parameter, $event)"
        />
      </template>

      <q-select
        v-if="form.checkpoint_mode === 'timestamp'"
        v-model="windowStartKey"
        outlined
        dense
        emit-value
        map-options
        :options="windowParameterOptions"
        :label="t('ui.windowStartParameters')"
        :hint="t('ui.optionsFromTheRequestedParameterAsDefinedByTheCurrent')"
        :rules="[requiredRule]"
      >
        <template #no-option>
          <q-item>
            <q-item-section class="text-grey-7">{{
              t('ui.currentInterfaceDoesNotDeclareBindingTimeParameters')
            }}</q-item-section>
          </q-item>
        </template>
      </q-select>
      <q-select
        v-if="form.checkpoint_mode === 'timestamp'"
        v-model="windowStartFormat"
        outlined
        dense
        emit-value
        map-options
        :options="formatOptions(windowStartKey)"
        :label="t('ui.windowStartFormat')"
        :hint="t('ui.formatIsThePlatformSFixedRulePleaseSelectThe')"
        :rules="[requiredRule]"
      >
        <template #no-option>
          <q-item>
            <q-item-section class="text-grey-7">{{
              t('ui.pleaseSelectTheStartingParametersForTheWindow')
            }}</q-item-section>
          </q-item>
        </template>
      </q-select>
      <q-select
        v-if="form.checkpoint_mode === 'timestamp' && windowMode === 'bounded_window'"
        v-model="windowEndKey"
        outlined
        dense
        emit-value
        map-options
        :options="windowParameterOptions"
        :label="t('ui.windowEndParameters')"
        :rules="[requiredRule]"
      />
      <q-select
        v-if="form.checkpoint_mode === 'timestamp' && windowMode === 'bounded_window'"
        v-model="windowEndFormat"
        outlined
        dense
        emit-value
        map-options
        :options="formatOptions(windowEndKey)"
        :label="t('ui.windowEndFormat')"
        :rules="[requiredRule]"
      />
      <q-input
        v-model="form.description"
        class="sync-task-form__wide"
        outlined
        dense
        type="textarea"
        autogrow
        maxlength="512"
        :label="t('ui.description')"
      />
    </q-form>
    <template #footer-status
      ><span class="text-caption text-grey-7">{{
        t('ui.theConfigurationOfTheTechnologyCannotBeModifiedDirectlyAfter')
      }}</span></template
    >
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { primitiveText } from 'src/utils/primitive-text'

import { computed, reactive, ref, watch } from 'vue'
import type { QForm } from 'quasar'
import SweetDateTimePicker from 'src/components/DateTime/SweetDateTimePicker.vue'
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
  type SyncWindowMode,
  useIntegrationApi,
} from 'src/api/services/integration'

const { t } = useI18n({ useScope: 'global' })

export type SyncTaskFormValue = SyncTaskCreateRequest
type SyncTaskEditableForm = Omit<
  SyncTaskCreateRequest,
  'external_system_id' | 'interface_definition_id'
> & {
  external_system_id: number | null
  interface_definition_id: number | null
}
const props = withDefaults(
  defineProps<{
    modelValue: boolean
    editData: SyncTaskEdit | null
    systems: ExternalSystemListItem[]
    interfaces: InterfaceDefinitionListItem[]
    consumers: SyncConsumerMetadata[]
    loading?: boolean
  }>(),
  { loading: false },
)
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: SyncTaskFormValue): void
}>()
const api = useIntegrationApi()
const formRef = ref<QForm | null>(null)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const definition = ref<InterfaceDefinitionDetail | null>(null)
const selectedStaticKeys = ref<string[]>([])
const staticValues = reactive<Record<string, string | boolean>>({})
const consumerKey = ref('')
const windowStartKey = ref('')
const windowEndKey = ref('')
const windowStartFormat = ref<SyncTimeFormat | ''>('')
const windowEndFormat = ref<SyncTimeFormat | ''>('')
const windowMode = ref<SyncWindowMode>('bounded_window')
const initialCheckpointLocal = ref('')
const form = reactive<SyncTaskEditableForm>(emptyForm())

const scheduleOptions: { label: string; value: SyncScheduleType }[] = [
  {
    get label() {
      return t('ui.manualTriggerOnly')
    },
    value: 'none',
  },
  {
    get label() {
      return t('ui.cronSchedule')
    },
    value: 'cron',
  },
]
const cronExamples = [
  {
    value: '0 * * * *',
    get label() {
      return t('ui.hourlyFullPoint')
    },
  },
  {
    value: '0 */2 * * *',
    get label() {
      return t('ui.every2Hours')
    },
  },
  {
    value: '0 2 * * *',
    get label() {
      return t('ui.daily0200')
    },
  },
  {
    value: '0 2 * * 1-5',
    get label() {
      return t('ui.workday0200')
    },
  },
]
const checkpointOptions: { label: string; value: SyncCheckpointMode }[] = [
  {
    get label() {
      return t('ui.noCheckpoint')
    },
    value: 'none',
  },
  {
    get label() {
      return t('ui.timetamp')
    },
    value: 'timestamp',
  },
]
const windowModeOptions: { label: string; value: SyncWindowMode }[] = [
  {
    get label() {
      return t('ui.fullStartWindow')
    },
    value: 'bounded_window',
  },
  {
    get label() {
      return t('ui.onlyBelowTheBoundsOfTimeResponsesAreNotSubject')
    },
    value: 'lower_bound_only',
  },
]
const systemOptions = computed(() =>
  props.systems.map((item) => ({ label: `${item.name} (${item.system_code})`, value: item.id })),
)
const interfaceOptions = computed(() =>
  props.interfaces
    .filter(
      (item) => item.external_system.id === form.external_system_id && item.status === 'enabled',
    )
    .map((item) => ({
      label: `${item.name} (${item.interface_code} · v${item.version})`,
      value: item.id,
    })),
)
const consumerOptions = computed(() =>
  props.consumers.map((item) => ({
    label: `${item.name || item.code} (${item.code} · v${item.version})`,
    value: `${item.code}@${item.version}`,
  })),
)
const parameters = computed(() => definition.value?.input_contract?.parameters || [])
const eligibleParameters = computed(() =>
  parameters.value.filter(
    (item) =>
      !item.sensitive &&
      !['authorization', 'cookie', 'idempotency-key'].includes(item.code.toLowerCase()),
  ),
)
const parameterKey = (parameter: InterfaceInputParameter) =>
  `${parameter.location}:${parameter.code}`
const parameterLabel = (parameter: InterfaceInputParameter) =>
  `${parameter.name || parameter.code} · ${parameter.location}/${parameter.data_type}${parameter.required ? ' *' : ''}`
const staticParameterOptions = computed(() =>
  eligibleParameters.value
    .filter((item) => !['object', 'array'].includes(item.data_type))
    .map((item) => ({ label: parameterLabel(item), value: parameterKey(item) })),
)
const windowParameterOptions = computed(() =>
  eligibleParameters.value
    .filter(
      (item) => !item.allow_multiple && ['string', 'integer', 'number'].includes(item.data_type),
    )
    .map((item) => ({ label: parameterLabel(item), value: parameterKey(item) })),
)
const selectedStaticParameters = computed(() =>
  eligibleParameters.value.filter((item) => selectedStaticKeys.value.includes(parameterKey(item))),
)
const formatOptions = (key: string) => {
  const parameter = eligibleParameters.value.find((item) => parameterKey(item) === key)
  if (!parameter) return []
  const values: SyncTimeFormat[] =
    parameter.data_type === 'string'
      ? ['rfc3339', 'local_datetime_seconds', 'unix_seconds', 'unix_milliseconds']
      : ['unix_seconds', 'unix_milliseconds']
  return values.map((value) => ({
    get label() {
      return value === 'rfc3339'
        ? 'RFC 3339'
        : value === 'local_datetime_seconds'
          ? t('ui.localDateTimeSec')
          : value === 'unix_seconds'
            ? t('ui.unixSeconds')
            : t('ui.unixMilliseconds')
    },
    value,
  }))
}
const normalizeWindowFormat = (key: string, current: SyncTimeFormat | ''): SyncTimeFormat | '' => {
  const values = formatOptions(key).map((item) => item.value)
  return current && values.includes(current) ? current : values[0] || ''
}

const toLocalDateTime = (value?: string) => {
  if (!value) return ''
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return ''
  const pad2 = (part: number) => String(part).padStart(2, '0')
  return `${parsed.getFullYear()}-${pad2(parsed.getMonth() + 1)}-${pad2(parsed.getDate())} ${pad2(parsed.getHours())}:${pad2(parsed.getMinutes())}:${pad2(parsed.getSeconds())}`
}

const toAPIDateTime = (value: string) => new Date(value.replace(' ', 'T')).toISOString()

function emptyForm(): SyncTaskEditableForm {
  return {
    task_code: '',
    task_name: '',
    description: '',
    external_system_id: null,
    interface_definition_id: null,
    consumer_code: '',
    consumer_version: 0,
    schedule_type: 'none',
    cron_expression: '',
    timezone: 'UTC',
    checkpoint_mode: 'none',
    lookback_seconds: 0,
    window_slice_seconds: 0,
    input_plan: emptyPlan(),
  }
}
function emptyPlan(): SyncExecutionInputPlan {
  return { version: 1, static_input: { path_params: {}, query_params: {}, headers: {} } }
}
const taskCodeRule = (value: string) =>
  /^[a-z][a-z0-9_]{0,63}$/.test(value || '') || t('ui.pleaseEnterAValidTaskCode')
const requiredRule = (value: unknown) => Boolean(value) || t('ui.thisFieldIsRequired')
const positiveRule = (value: number | null) => Number(value) > 0 || t('ui.selectAValidItem')
const cronRule = (value: string) =>
  value.trim().split(/\s+/).length === 5 || t('ui.cronMustUseFiveFields')
const lookbackRule = (value: number) =>
  (value >= 0 && value <= 604800) || t('ui.lookbackMustBeBetween0And604800Seconds')
const sliceRule = (value: number) =>
  (value >= 60 && value <= 604800) || t('ui.theSliceMustBeBetween60And604800Seconds')

async function loadDefinition(id: number | null) {
  definition.value = id ? (await api.getInterfaceDefinition(id)).data || null : null
}
function onSystemChanged() {
  form.interface_definition_id = null
  resetPlan()
  definition.value = null
}
async function onInterfaceChanged(value: number | null) {
  resetPlan()
  await loadDefinition(Number(value) || 0)
}
function resetPlan() {
  selectedStaticKeys.value = []
  Object.keys(staticValues).forEach((key) => delete staticValues[key])
  windowStartKey.value = ''
  windowEndKey.value = ''
  windowStartFormat.value = ''
  windowEndFormat.value = ''
}
function setStaticTextValue(parameter: InterfaceInputParameter, value: string | number | null) {
  staticValues[parameterKey(parameter)] = String(value ?? '')
}
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
    const values = parameter.allow_multiple
      ? String(raw)
          .split(',')
          .map((item) => item.trim())
      : [String(raw)]
    if (parameter.location === 'path') plan.static_input.path_params[parameter.code] = String(raw)
    if (parameter.location === 'query') plan.static_input.query_params[parameter.code] = values
    if (parameter.location === 'header') plan.static_input.headers[parameter.code] = values
    if (parameter.location === 'body')
      (plan.static_input.json_body ||= {})[parameter.code] = parseStaticValue(parameter, raw)
  }
  if (form.checkpoint_mode === 'timestamp') {
    plan.version = 2
    plan.window_mode = windowMode.value
    const [startLocation, ...startCode] = windowStartKey.value.split(':')
    plan.window_start_binding = {
      location: startLocation as InterfaceInputParameter['location'],
      code: startCode.join(':'),
      format: windowStartFormat.value as SyncTimeFormat,
    }
    if (windowMode.value === 'bounded_window') {
      const [endLocation, ...endCode] = windowEndKey.value.split(':')
      plan.window_end_binding = {
        location: endLocation as InterfaceInputParameter['location'],
        code: endCode.join(':'),
        format: windowEndFormat.value as SyncTimeFormat,
      }
    }
  }
  return plan
}
function applyPlan(plan: SyncExecutionInputPlan) {
  resetPlan()
  windowMode.value = plan.version === 2 ? plan.window_mode || 'bounded_window' : 'bounded_window'
  for (const parameter of eligibleParameters.value) {
    const key = parameterKey(parameter)
    let value: unknown
    if (parameter.location === 'path') value = plan.static_input.path_params?.[parameter.code]
    if (parameter.location === 'query') value = plan.static_input.query_params?.[parameter.code]
    if (parameter.location === 'header') value = plan.static_input.headers?.[parameter.code]
    if (parameter.location === 'body') value = plan.static_input.json_body?.[parameter.code]
    if (value !== undefined) {
      selectedStaticKeys.value.push(key)
      staticValues[key] = Array.isArray(value)
        ? value.map((item) => primitiveText(item)).join(',')
        : typeof value === 'boolean'
          ? value
          : primitiveText(value)
    }
  }
  if (plan.window_start_binding) {
    windowStartKey.value = `${plan.window_start_binding.location}:${plan.window_start_binding.code}`
    windowStartFormat.value = plan.window_start_binding.format
  }
  if (plan.window_end_binding) {
    windowEndKey.value = `${plan.window_end_binding.location}:${plan.window_end_binding.code}`
    windowEndFormat.value = plan.window_end_binding.format
  }
}

watch(
  () => [props.modelValue, props.editData] as const,
  async ([open, edit]) => {
    if (!open) return
    Object.assign(
      form,
      edit
        ? {
            task_code: edit.task_code,
            task_name: edit.task_name,
            description: edit.description,
            external_system_id: edit.external_system.id,
            interface_definition_id: edit.interface_definition.id,
            consumer_code: edit.consumer.code,
            consumer_version: edit.consumer.version,
            schedule_type: edit.schedule_type,
            cron_expression: edit.cron_summary || '',
            timezone: edit.timezone,
            checkpoint_mode: edit.checkpoint_mode,
            initial_checkpoint_at: edit.initial_checkpoint_at,
            lookback_seconds: edit.lookback_seconds,
            window_slice_seconds: edit.window_slice_seconds,
            input_plan: edit.input_plan,
          }
        : emptyForm(),
    )
    consumerKey.value = form.consumer_code ? `${form.consumer_code}@${form.consumer_version}` : ''
    initialCheckpointLocal.value = toLocalDateTime(form.initial_checkpoint_at)
    await loadDefinition(form.interface_definition_id)
    applyPlan(edit?.input_plan || emptyPlan())
  },
  { immediate: true },
)
watch(
  () => form.schedule_type,
  (value) => {
    if (value === 'none') form.cron_expression = ''
  },
)
watch(
  () => form.checkpoint_mode,
  (value) => {
    if (value === 'none') {
      delete form.initial_checkpoint_at
      form.lookback_seconds = 0
      form.window_slice_seconds = 0
      windowStartKey.value = ''
      windowEndKey.value = ''
    } else if (!form.window_slice_seconds) form.window_slice_seconds = 3600
  },
)
watch(windowMode, (value) => {
  if (value === 'lower_bound_only') windowEndKey.value = ''
})
watch(windowStartKey, (key) => {
  windowStartFormat.value = normalizeWindowFormat(key, windowStartFormat.value)
})
watch(windowEndKey, (key) => {
  windowEndFormat.value = normalizeWindowFormat(key, windowEndFormat.value)
})
watch(consumerKey, (value) => {
  const [code, version] = value.split('@')
  form.consumer_code = code || ''
  form.consumer_version = Number(version) || 0
})

async function submit() {
  if (!(await formRef.value?.validate())) return
  if (form.external_system_id === null || form.interface_definition_id === null) return
  if (form.checkpoint_mode === 'timestamp' && initialCheckpointLocal.value)
    form.initial_checkpoint_at = toAPIDateTime(initialCheckpointLocal.value)
  else delete form.initial_checkpoint_at
  emit('submit', {
    ...form,
    external_system_id: form.external_system_id,
    interface_definition_id: form.interface_definition_id,
    input_plan: buildInputPlan(),
  })
}
</script>

<style scoped lang="scss">
.sync-task-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px 22px;
  padding: 4px 4px 18px;
}
.sync-task-form__wide {
  grid-column: 1 / -1;
}
@media (max-width: 760px) {
  .sync-task-form {
    grid-template-columns: 1fr;
  }
  .sync-task-form__wide {
    grid-column: auto;
  }
}
</style>
