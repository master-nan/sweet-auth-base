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
    @click="openPopup"
    @keydown.prevent
    @paste.prevent
    @drop.prevent
    @clear.stop="clearValue"
  >
    <template #append>
      <q-icon :name="appendIcon" class="cursor-pointer" @click.stop="openPopup">
        <q-popup-proxy
          ref="popupRef"
          transition-show="scale"
          transition-hide="scale"
          :breakpoint="0"
        >
          <div class="sweet-date-time-panel" :class="`sweet-date-time-panel--${type}`">
            <div v-if="showDate" class="sweet-date-time-calendar">
              <q-date
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
            </div>
            <q-separator v-if="showDate && showTime" vertical />
            <div v-if="showTime" class="sweet-date-time-clock">
              <div class="sweet-date-time-clock__head">
                <span>时间</span>
                <q-btn flat dense color="primary" label="现在" @click="setNow" />
              </div>
              <div class="sweet-date-time-clock__value">
                {{ normalizedTimeValue }}
              </div>
              <div class="sweet-date-time-clock__grid">
                <div
                  v-for="part in timeParts"
                  :key="part.key"
                  class="sweet-date-time-clock__part"
                >
                  <q-btn
                    flat
                    dense
                    round
                    icon="keyboard_arrow_up"
                    :aria-label="`${part.label}加一`"
                    @click="adjustTimePart(part.key, 1)"
                  />
                  <div class="sweet-date-time-clock__number">{{ timePartValue(part.key) }}</div>
                  <q-btn
                    flat
                    dense
                    round
                    icon="keyboard_arrow_down"
                    :aria-label="`${part.label}减一`"
                    @click="adjustTimePart(part.key, -1)"
                  />
                  <div class="sweet-date-time-clock__label">{{ part.label }}</div>
                </div>
              </div>
            </div>
            <div class="sweet-date-time-actions">
              <q-btn flat dense color="primary" label="今天" @click="setToday" />
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
type TimePartKey = 'hour' | 'minute' | 'second'

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
const timeParts = [
  { key: 'hour' as const, label: '时', max: 23 },
  { key: 'minute' as const, label: '分', max: 59 },
  { key: 'second' as const, label: '秒', max: 59 },
]

const pad2 = (value: number) => String(value).padStart(2, '0')

const parseTime = (value: string) => {
  const [hourRaw, minuteRaw, secondRaw] = String(value || '00:00:00').split(':')
  const normalize = (input: string | undefined, max: number) => {
    const numberValue = Number(input)
    if (!Number.isFinite(numberValue)) return 0
    return Math.min(Math.max(Math.trunc(numberValue), 0), max)
  }
  return {
    hour: normalize(hourRaw, 23),
    minute: normalize(minuteRaw, 59),
    second: normalize(secondRaw, 59),
  }
}

const normalizedTimeValue = computed(() => {
  const parts = parseTime(timeValue.value)
  return `${pad2(parts.hour)}:${pad2(parts.minute)}:${pad2(parts.second)}`
})

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
  if (props.type === 'time') return normalizedTimeValue.value || ''
  if (props.type === 'datetime') {
    if (!dateValue.value) return ''
    return `${dateValue.value} ${normalizedTimeValue.value || '00:00:00'}`
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

const timePartValue = (part: TimePartKey) => {
  return pad2(parseTime(timeValue.value)[part])
}

const setTimePart = (part: TimePartKey, value: number) => {
  const current = parseTime(timeValue.value)
  const max = part === 'hour' ? 23 : 59
  const normalized = ((value % (max + 1)) + max + 1) % (max + 1)
  current[part] = normalized
  timeValue.value = `${pad2(current.hour)}:${pad2(current.minute)}:${pad2(current.second)}`
  emitValue()
}

const adjustTimePart = (part: TimePartKey, delta: number) => {
  const current = parseTime(timeValue.value)
  setTimePart(part, current[part] + delta)
}

const setToday = () => {
  const now = new Date()
  if (props.type === 'time') {
    timeValue.value = `${pad2(now.getHours())}:${pad2(now.getMinutes())}:${pad2(now.getSeconds())}`
  } else if (props.type === 'year') {
    dateValue.value = String(now.getFullYear())
  } else if (props.type === 'year-month') {
    dateValue.value = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}`
  } else {
    dateValue.value = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`
    if (props.type === 'datetime' && !timeValue.value) {
      timeValue.value = '00:00:00'
    }
  }
  emitValue()
}

const setNow = () => {
  const now = new Date()
  timeValue.value = `${pad2(now.getHours())}:${pad2(now.getMinutes())}:${pad2(now.getSeconds())}`
  if (props.type === 'datetime' && !dateValue.value) {
    dateValue.value = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`
  }
  emitValue()
}

const clearValue = () => {
  dateValue.value = ''
  timeValue.value = ''
  emit('update:modelValue', '')
  emit('change', '')
  popupRef.value?.hide?.()
}

const openPopup = () => {
  if (props.disable) return
  popupRef.value?.show?.()
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
  min-width: 292px;
  display: grid;
  grid-template-columns: 1fr;
  gap: 0;
  overflow: hidden;
  padding: 0;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 12px 34px rgba(33, 43, 72, 0.18);
}

.sweet-date-time-panel--datetime {
  grid-template-columns: 292px auto 180px;
  align-items: stretch;
}

.sweet-date-time-calendar {
  padding: 8px;
}

.sweet-date-time-panel :deep(.q-date) {
  width: 276px;
  box-shadow: none;
}

.sweet-date-time-panel :deep(.q-date__header) {
  display: none;
}

.sweet-date-time-panel :deep(.q-date__calendar-item button) {
  border-radius: 6px;
}

.sweet-date-time-clock {
  width: 180px;
  display: flex;
  flex-direction: column;
  padding: 14px 14px 10px;
  background: #fbfcff;
}

.sweet-date-time-clock__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #172033;
  font-size: 14px;
  font-weight: 700;
}

.sweet-date-time-clock__value {
  margin: 10px 0 12px;
  padding: 8px 10px;
  border: 1px solid #dfe5f1;
  border-radius: 6px;
  background: #fff;
  color: #172033;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 18px;
  text-align: center;
}

.sweet-date-time-clock__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
}

.sweet-date-time-clock__part {
  display: grid;
  justify-items: center;
  gap: 2px;
}

.sweet-date-time-clock__number {
  width: 42px;
  height: 36px;
  display: grid;
  place-items: center;
  border: 1px solid #e1e6f0;
  border-radius: 6px;
  background: #fff;
  color: #172033;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 18px;
  font-weight: 700;
}

.sweet-date-time-clock__label {
  color: #7a869f;
  font-size: 12px;
}

.sweet-date-time-actions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid #edf0f7;
  background: #fff;
}
</style>
