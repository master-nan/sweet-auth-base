<template>
  <sweet-select
    ref="selectRef"
    :model-value="normalizedModel"
    :options="filteredOptions"
    class="scope-value-select"
    dense
    outlined
    multiple
    emit-value
    map-options
    options-dense
    use-input
    input-debounce="120"
    clearable
    clear-icon="close"
    :label="label"
    :disable="disable"
    :loading="loading"
    :hint="hint"
    :max-values="maxValues || undefined"
    :new-value-mode="freeInput ? 'add-unique' : undefined"
    :display-value="selectionDisplay"
    @update:model-value="emitNormalized"
    @filter="filterOptions"
    @new-value="addFreeValues"
    @focus="emit('focus')"
  >
    <template #no-option>
      <q-item>
        <q-item-section class="text-grey">
          {{ freeInput ? '输入后回车添加' : '暂无可选数据' }}
        </q-item-section>
      </q-item>
    </template>
    <q-tooltip v-if="selectionTooltip">
      {{ selectionTooltip }}
    </q-tooltip>
  </sweet-select>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { DataPermissionOption } from 'src/api/services/data-permission'
import SweetSelect from 'src/components/Select/SweetSelect.vue'

const props = withDefaults(
  defineProps<{
    modelValue?: Array<string | number | null | undefined>
    options?: DataPermissionOption[]
    label?: string
    hint?: string
    disable?: boolean
    loading?: boolean
    freeInput?: boolean
    maxValues?: number
  }>(),
  {
    modelValue: () => [],
    options: () => [],
    label: '',
    hint: '',
    disable: false,
    loading: false,
    freeInput: false,
    maxValues: 0,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: string[]): void
  (event: 'focus'): void
}>()

const selectRef = ref()
const filterKeyword = ref('')

const normalizeValue = (value: unknown) => String(value ?? '').trim()

const normalizedModel = computed(() =>
  (props.modelValue || [])
    .map(normalizeValue)
    .filter(Boolean)
    .filter((value, index, values) => values.indexOf(value) === index),
)

const normalizedOptions = computed<DataPermissionOption[]>(() =>
  (props.options || [])
    .map((option) => {
      const parent = option.parent === undefined || option.parent === null ? '' : normalizeValue(option.parent)
      return {
        ...option,
        label: String(option.label ?? option.value ?? ''),
        value: normalizeValue(option.value),
        ...(parent ? { parent } : {}),
      }
    })
    .filter((option) => option.value),
)

const optionMap = computed(() => new Map(normalizedOptions.value.map((option) => [option.value, option.label])))

const mergedOptions = computed<DataPermissionOption[]>(() => {
  const seen = new Set<string>()
  const merged: DataPermissionOption[] = []
  normalizedOptions.value.forEach((option) => {
    if (seen.has(option.value)) return
    seen.add(option.value)
    merged.push(option)
  })
  normalizedModel.value.forEach((value) => {
    if (seen.has(value)) return
    seen.add(value)
    merged.push({ label: value, value })
  })
  return merged
})

const filteredOptions = computed(() => {
  const keyword = filterKeyword.value.trim().toLowerCase()
  if (!keyword) return mergedOptions.value
  return mergedOptions.value.filter((option) =>
    String(props.freeInput ? `${option.label} ${option.value}` : option.label).toLowerCase().includes(keyword),
  )
})

const labelForValue = (value: string) => optionMap.value.get(value) || value

const selectedLabels = computed(() => normalizedModel.value.map(labelForValue))

const selectionDisplay = computed(() => {
  const labels = selectedLabels.value
  if (labels.length === 0) return ''
  const visibleLabels = labels.slice(0, 2).join('、')
  return labels.length > 2 ? `${visibleLabels} 等 ${labels.length} 个` : visibleLabels
})

const selectionTooltip = computed(() => selectedLabels.value.join('、'))

const emitNormalized = (value: unknown) => {
  const next = Array.isArray(value) ? value : value === null || value === undefined || value === '' ? [] : [value]
  emit(
    'update:modelValue',
    next
      .map(normalizeValue)
      .filter(Boolean)
      .filter((item, index, values) => values.indexOf(item) === index),
  )
}

const addFreeValues = (value: string, done: (value?: string | string[], mode?: 'toggle' | 'add' | 'add-unique') => void) => {
  if (!props.freeInput) {
    done()
    return
  }
  const values = String(value || '')
    .split(/[,，;；\n]+/)
    .map((item) => item.trim())
    .filter(Boolean)
  if (values.length === 0) {
    done()
    return
  }
  const next = [...normalizedModel.value]
  values.forEach((item) => {
    if (!next.includes(item)) next.push(item)
  })
  emit('update:modelValue', next)
  done()
}

const filterOptions = (value: string, update: (callback: () => void) => void) => {
  update(() => {
    filterKeyword.value = value
  })
}

defineExpose({
  focus: () => selectRef.value?.focus?.(),
})
</script>

<style scoped lang="scss">
.scope-value-select :deep(.q-field__native) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
