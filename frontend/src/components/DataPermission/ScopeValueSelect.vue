<template>
  <q-select
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
    @update:model-value="emitNormalized"
    @filter="filterOptions"
    @new-value="addFreeValues"
    @focus="emit('focus')"
  >
    <template #selected>
      <div v-if="normalizedModel.length" class="scope-value-select__chips">
        <q-chip
          v-for="value in normalizedModel"
          :key="value"
          dense
          square
          removable
          class="scope-value-select__chip"
          @remove.stop="removeValue(value)"
        >
          <span class="scope-value-select__chip-label">{{ labelForValue(value) }}</span>
        </q-chip>
      </div>
    </template>

    <template #no-option>
      <q-item>
        <q-item-section class="text-grey">
          {{ freeInput ? '输入后回车添加' : '暂无可选数据' }}
        </q-item-section>
      </q-item>
    </template>
  </q-select>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { DataPermissionOption } from 'src/api/services/data-permission'

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
    [option.label, option.value].join(' ').toLowerCase().includes(keyword),
  )
})

const labelForValue = (value: string) => optionMap.value.get(value) || value

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

const removeValue = (value: string) => {
  emit('update:modelValue', normalizedModel.value.filter((item) => item !== value))
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
}

.scope-value-select__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  max-height: 72px;
  overflow: auto;
  padding: 2px 0;
}

.scope-value-select__chip {
  max-width: 190px;
  margin: 0;
  border-radius: 6px;
  background: #eef2ff;
  color: #4f46e5;
}

.scope-value-select__chip-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
