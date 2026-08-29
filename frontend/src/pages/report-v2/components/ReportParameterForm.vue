<template>
  <div class="report-parameter-form">
    <div class="parameter-grid">
      <q-input
        :model-value="keyword"
        dense
        outlined
        clearable
        label="关键词"
        :disable="disabled"
        @update:model-value="$emit('update:keyword', String($event || ''))"
        @keyup.enter="$emit('search')"
      />

      <template v-for="param in parameters" :key="param.id">
        <q-select
          v-if="controlMeta(param).controlType === 'select'"
          :model-value="valueOf(param.id)"
          dense
          outlined
          emit-value
          map-options
          clearable
          :label="controlMeta(param).label"
          :options="controlMeta(param).options"
          :loading="loading"
          :disable="Boolean(disabled)"
          @update:model-value="updateValue(param.id, $event)"
        />

        <q-select
          v-else-if="controlMeta(param).controlType === 'boolean'"
          :model-value="valueOf(param.id)"
          dense
          outlined
          emit-value
          map-options
          clearable
          :label="controlMeta(param).label"
          :options="controlMeta(param).options"
          :disable="disabled"
          @update:model-value="updateValue(param.id, $event)"
        />

        <sweet-date-time-picker
          v-else-if="['date', 'datetime'].includes(controlMeta(param).controlType)"
          :model-value="String(inputValueOf(param.id) || '')"
          :type="controlMeta(param).controlType === 'datetime' ? 'datetime' : 'date'"
          :label="controlMeta(param).label"
          :disable="Boolean(disabled)"
          @update:model-value="updateValue(param.id, $event)"
        />

        <q-input
          v-else
          :model-value="inputValueOf(param.id)"
          dense
          outlined
          clearable
          :type="controlMeta(param).htmlInputType"
          :label="controlMeta(param).label"
          :placeholder="controlMeta(param).placeholder"
          :disable="disabled"
          @update:model-value="updateValue(param.id, normalizeInputValue(param.id, $event))"
          @keyup.enter="$emit('search')"
        />
      </template>
    </div>

    <div class="parameter-actions">
      <div class="parameter-hint">
        <q-spinner v-if="loading" size="14px" color="primary" />
        <span v-if="parameters.length">参数控件优先复用低代码字段元数据和字典。</span>
        <span v-else>当前报表未配置参数，可直接查询运行。</span>
      </div>
      <div class="row q-gutter-sm">
        <q-btn outline color="primary" icon="restart_alt" label="重置" :disable="disabled" @click="$emit('reset')" />
        <q-btn color="primary" unelevated icon="search" label="查询" :loading="disabled" @click="$emit('search')" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import SweetDateTimePicker from 'src/components/DateTime/SweetDateTimePicker.vue'
import type { ReportParameter } from 'src/api/services/report'
import type {
  ReportParameterControlMeta,
  ReportParameterControlType,
} from '../composables/useReportParameterControls'

type ParameterValue = string | number | boolean | Array<string | number> | null | undefined

const props = withDefaults(defineProps<{
  parameters: ReportParameter[]
  modelValue: Record<string, ParameterValue>
  keyword?: string
  loading?: boolean
  controlMetas?: Record<string, ReportParameterControlMeta>
  disabled?: boolean
}>(), {
  keyword: '',
  loading: false,
  controlMetas: () => ({}),
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, ParameterValue>]
  'update:keyword': [value: string]
  search: []
  reset: []
}>()

function controlMeta(param: ReportParameter): ReportParameterControlMeta {
  return props.controlMetas[param.id] || {
    id: param.id,
    label: param.label,
    field: param.field,
    controlType: fallbackControlType(String(param.type || 'text')),
    htmlInputType: fallbackHtmlInputType(String(param.type || 'text')),
    options: [],
    required: false,
    placeholder: param.placeholder || `请输入${param.label}`,
    source: 'fallback',
  }
}

function valueOf(id: string) {
  return props.modelValue[id] ?? null
}

function inputValueOf(id: string): string | number | null {
  const value = props.modelValue[id]
  if (typeof value === 'string' || typeof value === 'number') return value
  return null
}

function updateValue(id: string, value: ParameterValue) {
  emit('update:modelValue', {
    ...props.modelValue,
    [id]: value,
  })
}

function normalizeInputValue(id: string, value: unknown): ParameterValue {
  const meta = props.controlMetas[id]
  if (value === '' || value === null || value === undefined) return null
  if (meta?.controlType === 'number') return Number(value)
  return String(value)
}

function fallbackControlType(type: string): ReportParameterControlType {
  if (type === 'select') return 'select'
  if (type === 'date') return 'date'
  if (type === 'datetime') return 'datetime'
  if (type === 'number') return 'number'
  if (type === 'boolean' || type === 'bool') return 'boolean'
  return 'text'
}

function fallbackHtmlInputType(type: string) {
  if (type === 'number') return 'number'
  if (type === 'date') return 'date'
  if (type === 'datetime') return 'datetime-local'
  return 'text'
}
</script>

<style scoped lang="scss">
.report-parameter-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.parameter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
  align-items: start;
}

.parameter-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.parameter-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #7a8699;
  font-size: 12px;
}
</style>
