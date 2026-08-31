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
          :target="popupTarget"
          no-parent-event
          anchor="bottom middle"
          self="top middle"
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
              <div class="sweet-date-time-clock__selection">
                <span>{{ t('ui.selected') }}</span>
                <strong>{{ selectedValueSummary }}</strong>
              </div>
              <div class="sweet-date-time-clock__head">
                <span>{{ t('ui.time') }}</span>
                <q-btn flat dense color="primary" :label="t('ui.now')" @click="setNow" />
              </div>
              <div class="sweet-date-time-clock__editor">
                <template v-for="(part, index) in timeParts" :key="part.key">
                  <div class="sweet-date-time-clock__part">
                    <q-input
                      :model-value="timeDraft[part.key]"
                      :aria-label="part.inputLabel"
                      :data-time-part="part.key"
                      outlined
                      dense
                      hide-bottom-space
                      inputmode="numeric"
                      maxlength="2"
                      autocomplete="off"
                      @focus="selectTimePart"
                      @blur="normalizeTimePart(part.key)"
                      @update:model-value="handleTimePartInput(part.key, $event)"
                      @keydown.up.prevent="adjustTimePart(part.key, 1)"
                      @keydown.down.prevent="adjustTimePart(part.key, -1)"
                      @wheel.prevent="adjustTimePart(part.key, $event.deltaY < 0 ? 1 : -1)"
                    />
                    <span>{{ part.label }}</span>
                  </div>
                  <span v-if="index < timeParts.length - 1" class="sweet-date-time-clock__colon"
                    >:</span
                  >
                </template>
              </div>
              <div class="sweet-date-time-clock__hint">
                {{ t('ui.youCanEnterDirectlyOrUseAArrowKeyAnd') }}
              </div>
              <div class="sweet-date-time-clock__quick-title">{{ t('ui.commonTime') }}</div>
              <div class="sweet-date-time-clock__quick-list">
                <q-btn
                  v-for="preset in quickTimes"
                  :key="preset"
                  flat
                  dense
                  no-caps
                  :aria-label="t('ui.setPreset', { preset })"
                  :class="{ 'is-active': normalizedTimeValue === preset }"
                  :label="preset"
                  @click="applyQuickTime(preset)"
                />
              </div>
            </div>
            <div class="sweet-date-time-actions">
              <q-btn
                v-if="showDate"
                flat
                dense
                color="primary"
                :label="t('ui.today')"
                @click="setToday"
              />
              <q-btn flat dense color="grey-7" :label="t('ui.clear')" @click="clearValue" />
              <q-btn unelevated dense color="primary" :label="t('ui.sure')" @click="confirmValue" />
            </div>
          </div>
        </q-popup-proxy>
      </q-icon>
    </template>
  </q-input>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, reactive, ref, watch } from 'vue'

const { t } = useI18n({ useScope: 'global' })

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
const timeDraft = reactive<Record<TimePartKey, string>>({
  hour: '00',
  minute: '00',
  second: '00',
})

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
const popupTarget = computed(() => inputRef.value?.$el || false)
const timeParts = [
  {
    key: 'hour' as const,
    get label() {
      return t('ui.hourUnitShort')
    },
    get inputLabel() {
      return t('ui.hours')
    },
  },
  {
    key: 'minute' as const,
    get label() {
      return t('ui.minuteUnitShort')
    },
    get inputLabel() {
      return t('ui.min')
    },
  },
  {
    key: 'second' as const,
    get label() {
      return t('ui.sec')
    },
    get inputLabel() {
      return t('ui.seconds')
    },
  },
]
const quickTimes = ['00:00:00', '08:30:00', '12:00:00', '18:00:00']

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

const selectedValueSummary = computed(() => {
  if (props.type === 'time') return normalizedTimeValue.value
  if (!dateValue.value) return t('ui.pleaseSelectADate')
  return `${dateValue.value} ${normalizedTimeValue.value}`
})

const syncTimeDraft = () => {
  const parts = parseTime(timeValue.value)
  timeDraft.hour = pad2(parts.hour)
  timeDraft.minute = pad2(parts.minute)
  timeDraft.second = pad2(parts.second)
}

const splitValue = (value: string | null | undefined) => {
  const raw = String(value || '').trim()
  if (!raw) {
    dateValue.value = ''
    timeValue.value = ''
    syncTimeDraft()
    return
  }
  if (props.type === 'time') {
    timeValue.value = raw
    syncTimeDraft()
    return
  }
  if (props.type === 'datetime') {
    const [datePart, timePart] = raw.split(' ')
    dateValue.value = datePart || ''
    timeValue.value = timePart || '00:00:00'
    syncTimeDraft()
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

const setTimePart = (part: TimePartKey, value: number) => {
  const current = parseTime(timeValue.value)
  const max = part === 'hour' ? 23 : 59
  const normalized = ((value % (max + 1)) + max + 1) % (max + 1)
  current[part] = normalized
  timeValue.value = `${pad2(current.hour)}:${pad2(current.minute)}:${pad2(current.second)}`
  syncTimeDraft()
  emitValue()
}

const adjustTimePart = (part: TimePartKey, delta: number) => {
  const current = parseTime(timeValue.value)
  setTimePart(part, current[part] + delta)
}

const handleTimePartInput = (part: TimePartKey, value: string | number | null) => {
  const digits = String(value ?? '')
    .replace(/\D/g, '')
    .slice(0, 2)
  timeDraft[part] = digits
  if (!digits) return

  const current = parseTime(timeValue.value)
  const max = part === 'hour' ? 23 : 59
  current[part] = Math.min(Number(digits), max)
  timeValue.value = `${pad2(current.hour)}:${pad2(current.minute)}:${pad2(current.second)}`
  emitValue()
}

const normalizeTimePart = (part: TimePartKey) => {
  const current = parseTime(timeValue.value)
  const draftValue = Number(timeDraft[part])
  setTimePart(part, Number.isFinite(draftValue) ? draftValue : current[part])
}

const selectTimePart = (event: Event) => {
  const target = event.target
  if (target instanceof HTMLInputElement) target.select()
}

const applyQuickTime = (value: string) => {
  timeValue.value = value
  syncTimeDraft()
  emitValue()
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
  syncTimeDraft()
  emitValue()
}

const setNow = () => {
  const now = new Date()
  timeValue.value = `${pad2(now.getHours())}:${pad2(now.getMinutes())}:${pad2(now.getSeconds())}`
  if (props.type === 'datetime' && !dateValue.value) {
    dateValue.value = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`
  }
  syncTimeDraft()
  emitValue()
}

const clearValue = () => {
  dateValue.value = ''
  timeValue.value = ''
  syncTimeDraft()
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
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
  box-shadow: 0 12px 34px rgba(33, 43, 72, 0.18);
}

.sweet-date-time-panel--datetime {
  grid-template-columns: 292px auto 196px;
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

.sweet-date-time-panel :deep(.q-date__calendar-item .bg-primary),
.sweet-date-time-panel :deep(.q-date__calendar-item .bg-primary .q-btn__content),
.sweet-date-time-panel :deep(.q-date__calendar-item .q-btn--active),
.sweet-date-time-panel :deep(.q-date__calendar-item .q-btn--active .q-btn__content) {
  color: #fff !important;
}

.sweet-date-time-panel :deep(.q-date__calendar-item .text-primary:not(.bg-primary)) {
  color: var(--q-primary) !important;
}

.sweet-date-time-clock {
  width: 196px;
  display: flex;
  flex-direction: column;
  padding: 10px 12px;
  background: var(--app-surface-muted);
}

.sweet-date-time-clock__selection {
  min-height: 44px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 3px;
  margin-bottom: 8px;
  padding: 6px 8px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface);
}

.sweet-date-time-clock__selection span {
  color: var(--app-text-muted);
  font-size: 12px;
}

.sweet-date-time-clock__selection strong {
  overflow: hidden;
  color: var(--app-text-strong);
  font-size: 13px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sweet-date-time-clock__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--app-text-strong);
  font-size: 14px;
  font-weight: 700;
}

.sweet-date-time-clock__editor {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 6px minmax(0, 1fr) 6px minmax(0, 1fr);
  align-items: start;
  gap: 2px;
  margin-top: 8px;
}

.sweet-date-time-clock__part {
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.sweet-date-time-clock__part :deep(.q-field) {
  width: 100%;
}

.sweet-date-time-clock__part :deep(.q-field__control) {
  height: 40px;
  background: var(--app-surface);
}

.sweet-date-time-clock__part :deep(.q-field__native) {
  color: var(--app-text-strong);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 15px;
  font-weight: 600;
  text-align: center;
}

.sweet-date-time-clock__part > span {
  color: var(--app-text-muted);
  font-size: 11px;
}

.sweet-date-time-clock__colon {
  padding-top: 5px;
  color: var(--app-text-strong);
  font-size: 16px;
  font-weight: 600;
  text-align: center;
}

.sweet-date-time-clock__hint {
  margin-top: 4px;
  color: var(--app-text-muted);
  font-size: 11px;
  text-align: center;
}

.sweet-date-time-clock__quick-title {
  margin-top: 10px;
  color: var(--app-text-muted);
  font-size: 12px;
}

.sweet-date-time-clock__quick-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  margin-top: 6px;
}

.sweet-date-time-clock__quick-list :deep(.q-btn) {
  min-height: 28px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface);
  color: var(--app-text-strong);
  padding: 0 4px;
  font-size: 12px;
}

.sweet-date-time-clock__quick-list :deep(.q-btn.is-active) {
  border-color: var(--q-primary);
  color: var(--q-primary);
}

.sweet-date-time-actions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid var(--app-border);
  background: var(--app-surface);
}

@media (max-width: 560px) {
  .sweet-date-time-panel--datetime {
    grid-template-columns: 292px;
  }

  .sweet-date-time-panel--datetime :deep(.q-separator--vertical) {
    display: none;
  }

  .sweet-date-time-clock {
    width: 292px;
  }
}
</style>
