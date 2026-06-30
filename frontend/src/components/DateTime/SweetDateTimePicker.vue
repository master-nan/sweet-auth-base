<template>
  <q-input
    ref="inputRef"
    :model-value="displayValue"
    :label="label"
    outlined
    dense
    readonly
    :clearable="clearable"
    clear-icon="close"
    :disable="disable"
    :rules="rules"
    :hint="hint"
    :lazy-rules="lazyRules"
    class="sweet-date-time-picker"
    @clear.stop="clearValue"
  >
    <template #append>
      <q-icon :name="appendIcon" class="cursor-pointer">
        <q-popup-proxy
          ref="popupRef"
          transition-show="scale"
          transition-hide="scale"
          :breakpoint="0"
        >
          <div class="sweet-date-time-panel" :class="`sweet-date-time-panel--${type}`">
            <q-date
              v-if="showDate"
              v-model="dateValue"
              :mask="dateMask"
              :default-view="dateDefaultView"
              emit-immediately
              flat
              minimal
              color="primary"
              text-color="primary"
              @update:model-value="handleDateChange"
            />
            <q-separator v-if="showDate && showTime" vertical />
            <q-time
              v-if="showTime"
              v-model="timeValue"
              mask="HH:mm:ss"
              format24h
              flat
              color="primary"
              text-color="primary"
              @update:model-value="emitValue"
            />
            <div class="sweet-date-time-actions">
              <q-btn flat dense color="grey-7" label="清空" @click="clearValue" />
              <q-btn unelevated dense color="primary" label="确定" @click="confirmValue" />
            </div>
          </div>
        </q-popup-proxy>
      </q-icon>
    </template>
  </q-input>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

type PickerType = 'date' | 'time' | 'datetime' | 'year' | 'year-month'

const props = withDefaults(
  defineProps<{
    modelValue?: string | null
    type?: PickerType
    label?: string
    clearable?: boolean
    disable?: boolean
    rules?: Array<(value: unknown) => boolean | string>
    hint?: string
    lazyRules?: boolean | 'ondemand'
  }>(),
  {
    modelValue: '',
    type: 'date',
    label: '',
    clearable: true,
    disable: false,
    rules: () => [],
    hint: '',
    lazyRules: 'ondemand',
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'change', value: string): void
}>()

const inputRef = ref<any>(null)
const popupRef = ref<any>(null)
const dateValue = ref('')
const timeValue = ref('')

const showDate = computed(() => props.type !== 'time')
const showTime = computed(() => props.type === 'time' || props.type === 'datetime')
const appendIcon = computed(() => (props.type === 'time' ? 'access_time' : 'event'))
const dateMask = computed(() => {
  if (props.type === 'year') return 'YYYY'
  if (props.type === 'year-month') return 'YYYY-MM'
  return 'YYYY-MM-DD'
})
const dateDefaultView = computed(() => {
  if (props.type === 'year') return 'Years'
  if (props.type === 'year-month') return 'Months'
  return 'Calendar'
})
const displayValue = computed(() => props.modelValue || '')

const splitValue = (value: string | null | undefined) => {
  const raw = String(value || '').trim()
  if (!raw) {
    dateValue.value = ''
    timeValue.value = ''
    return
  }
  if (props.type === 'time') {
    timeValue.value = raw
    return
  }
  if (props.type === 'datetime') {
    const [datePart, timePart] = raw.split(' ')
    dateValue.value = datePart || ''
    timeValue.value = timePart || '00:00:00'
    return
  }
  dateValue.value = raw
}

const buildValue = () => {
  if (props.type === 'time') return timeValue.value || ''
  if (props.type === 'datetime') {
    if (!dateValue.value) return ''
    return `${dateValue.value} ${timeValue.value || '00:00:00'}`
  }
  return dateValue.value || ''
}

const emitValue = () => {
  const value = buildValue()
  emit('update:modelValue', value)
  emit('change', value)
}

const handleDateChange = () => {
  if (props.type === 'year' && dateValue.value.length > 4) {
    dateValue.value = dateValue.value.slice(0, 4)
  }
  if (props.type === 'year-month' && dateValue.value.length > 7) {
    dateValue.value = dateValue.value.slice(0, 7)
  }
  emitValue()
}

const clearValue = () => {
  dateValue.value = ''
  timeValue.value = ''
  emit('update:modelValue', '')
  emit('change', '')
}

const confirmValue = () => {
  emitValue()
  popupRef.value?.hide?.()
}

watch(
  () => [props.modelValue, props.type],
  () => splitValue(props.modelValue),
  { immediate: true },
)

defineExpose({
  validate: () => inputRef.value?.validate?.(),
})
</script>

<style scoped lang="scss">
.sweet-date-time-picker :deep(.q-field__native) {
  cursor: pointer;
}

.sweet-date-time-panel {
  min-width: 290px;
  display: grid;
  grid-template-columns: 1fr;
  gap: 0;
  padding: 10px;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 10px 32px rgba(33, 43, 72, 0.18);
}

.sweet-date-time-panel--datetime {
  grid-template-columns: auto auto auto;
  align-items: stretch;
}

.sweet-date-time-panel :deep(.q-date),
.sweet-date-time-panel :deep(.q-time) {
  box-shadow: none;
}

.sweet-date-time-panel :deep(.q-date__calendar-item button),
.sweet-date-time-panel :deep(.q-time__clock-position) {
  border-radius: 6px;
}

.sweet-date-time-actions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 8px 4px 0;
  border-top: 1px solid #edf0f7;
}
</style>
