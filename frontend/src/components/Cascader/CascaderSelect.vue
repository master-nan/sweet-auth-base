<template>
  <div
    class="cascader-select"
    :style="containerStyle"
    @mousedown.capture="handleOpen"
    @click="handleOpen"
  >
    <q-input
      ref="inputRef"
      v-model="inputValue"
      :label="label"
      outlined
      dense
      readonly
      :clearable="clearable"
      clear-icon="close"
      :rules="rules"
      :disable="disable"
      class="cascader-input"
      input-class="cursor-pointer"
      @clear.stop="clearSelection"
      @click.stop="handleOpen"
      @focus="handleOpen"
    >
      <template v-slot:append>
        <q-icon name="arrow_drop_down" class="cursor-pointer" @click.stop="handleOpen" />
      </template>
    </q-input>
    <q-menu v-model="menu" anchor="bottom left" self="top left" :fit="false" no-refocus>
      <div class="cascader-menu" :style="menuStyle">
        <div
          v-for="(levelOptions, index) in levels"
          :key="index"
          class="cascader-column"
          :style="columnStyle"
        >
          <q-list separator>
            <q-item
              v-for="(option, optionIndex) in levelOptions"
              :key="getOptionValue(option) ?? `option-${index}-${optionIndex}`"
              clickable
              v-ripple
              :active="innerValue[index] === getOptionValue(option)"
              active-class="cascader-active"
              :disable="isOptionDisabled(option)"
              @click="() => handleOptionClick(index, option)"
            >
              <q-item-section>{{ getOptionLabel(option) }}</q-item-section>
              <q-item-section v-if="hasChildren(option)" side>
                <q-icon name="chevron_right" size="18px" />
              </q-item-section>
            </q-item>
          </q-list>
        </div>
      </div>
    </q-menu>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

export interface CascaderOption {
  label?: string
  value?: string | number
  children?: CascaderOption[]
  disabled?: boolean
  [key: string]: any
}

const props = defineProps({
  modelValue: {
    type: [Array, String, Number, Object, Boolean] as any,
    default: () => [],
  },
  options: {
    type: Array as () => CascaderOption[],
    default: () => [],
  },
  label: {
    type: String,
    default: '',
  },
  separator: {
    type: String,
    default: ' / ',
  },
  rules: {
    type: Array as () => Array<(val: any) => boolean | string>,
    default: () => [],
  },
  disable: {
    type: Boolean,
    default: false,
  },
  readonly: {
    type: Boolean,
    default: false,
  },
  valueMode: {
    type: String as () => 'value' | 'path' | 'object' | 'pathObjects',
    default: 'path',
  },
  selectable: {
    type: String as () => 'leaf' | 'any' | 'level',
    default: 'leaf',
  },
  selectableLevel: {
    type: Number,
    default: 0,
  },
  optionValueKey: {
    type: String,
    default: 'value',
  },
  optionLabelKey: {
    type: String,
    default: 'label',
  },
  optionChildrenKey: {
    type: String,
    default: 'children',
  },
  menuMinWidth: {
    type: Number,
    default: 360,
  },
  menuMaxWidth: {
    type: Number,
    default: 720,
  },
  columnMinWidth: {
    type: Number,
    default: 180,
  },
  maxHeight: {
    type: Number,
    default: 320,
  },
  showPath: {
    type: Boolean,
    default: true,
  },
  clearable: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits(['update:modelValue', 'change'])

const innerValue = ref<Array<string | number>>([])
// 已确认选中的值（仅在 canSelect 通过时更新）
const confirmedValue = ref<Array<string | number>>([])
const inputRef = ref<any>(null)

const menu = ref(false)

const getOptionValue = (option: CascaderOption) => option?.[props.optionValueKey]
const getOptionLabel = (option: CascaderOption) => option?.[props.optionLabelKey]
const getOptionChildren = (option: CascaderOption) =>
  option?.[props.optionChildrenKey] as CascaderOption[] | undefined

const hasChildren = (option: CascaderOption) => {
  const children = getOptionChildren(option)
  return Array.isArray(children) && children.length > 0
}

const isOptionDisabled = (option: CascaderOption) => !!option.disabled

const findPathByValue = (
  options: CascaderOption[],
  value: string | number,
  path: CascaderOption[] = [],
): CascaderOption[] => {
  for (const option of options) {
    const currentValue = getOptionValue(option)
    const nextPath = [...path, option]
    if (currentValue === value) {
      return nextPath
    }
    const children = getOptionChildren(option)
    if (children && children.length) {
      const found = findPathByValue(children, value, nextPath)
      if (found.length) {
        return found
      }
    }
  }
  return []
}

const normalizeValue = (value: any) => {
  if (props.valueMode === 'path') {
    return Array.isArray(value) ? value : []
  }
  if (props.valueMode === 'pathObjects') {
    return Array.isArray(value) ? value.map((item) => getOptionValue(item)) : []
  }
  if (props.valueMode === 'object') {
    const optionValue = value ? getOptionValue(value) : undefined
    if (optionValue === undefined || optionValue === null || optionValue === '') return []
    const pathOptions = findPathByValue(props.options, optionValue)
    return pathOptions.map((item) => getOptionValue(item))
  }
  if (value === undefined || value === null || value === '') return []
  const pathOptions = findPathByValue(props.options, value)
  return pathOptions.map((item) => getOptionValue(item))
}

watch(
  () => props.modelValue,
  (value) => {
    const normalized = normalizeValue(value)
    innerValue.value = normalized
    confirmedValue.value = [...normalized]
  },
  { immediate: true },
)

watch(
  () => [props.options, props.valueMode],
  () => {
    const normalized = normalizeValue(props.modelValue)
    innerValue.value = normalized
    confirmedValue.value = [...normalized]
  },
)

const findOption = (options: CascaderOption[], value: string | number) =>
  options.find((item) => getOptionValue(item) === value)

const levels = computed(() => {
  const result: CascaderOption[][] = []
  let currentOptions = props.options
  if (!currentOptions || currentOptions.length === 0) {
    return [[]]
  }
  result.push(currentOptions)

  for (const value of innerValue.value) {
    const matched = findOption(currentOptions, value)
    const children = matched ? getOptionChildren(matched) : undefined
    if (!children || children.length === 0) {
      break
    }
    currentOptions = children
    result.push(currentOptions)
  }

  return result
})

const resolvePathOptions = (values: Array<string | number>) => {
  const resolved: CascaderOption[] = []
  let currentOptions = props.options
  for (const value of values) {
    const matched = findOption(currentOptions, value)
    if (!matched) break
    resolved.push(matched)
    currentOptions = getOptionChildren(matched) || []
  }
  return resolved
}

const displayLabel = computed(() => {
  const labels = resolvePathOptions(confirmedValue.value).map((item) => getOptionLabel(item))
  if (!props.showPath) {
    return labels.length ? labels[labels.length - 1] : ''
  }
  return labels.join(props.separator)
})

const inputValue = computed({
  get: () => displayLabel.value,
  set: (val) => {
    if (!val) {
      clearSelection()
    }
  },
})

const canSelect = (index: number, option: CascaderOption) => {
  if (props.selectable === 'any') return true
  if (props.selectable === 'level') {
    return props.selectableLevel > 0 && index + 1 === props.selectableLevel
  }
  return !hasChildren(option)
}

const buildEmitValue = (values: Array<string | number>) => {
  const pathOptions = resolvePathOptions(values)
  if (props.valueMode === 'value') {
    return values[values.length - 1]
  }
  if (props.valueMode === 'object') {
    return pathOptions[pathOptions.length - 1]
  }
  if (props.valueMode === 'pathObjects') {
    return pathOptions
  }
  return values
}

const handleOptionClick = (index: number, option: CascaderOption) => {
  const nextValue = innerValue.value.slice(0, index)
  nextValue[index] = getOptionValue(option)
  innerValue.value = nextValue

  const isLeaf = !hasChildren(option)
  const selectableNow = canSelect(index, option)

  if (selectableNow) {
    confirmedValue.value = [...nextValue]
    const emitValue = buildEmitValue(nextValue)
    emit('update:modelValue', emitValue)
    emit('change', emitValue)
  }

  if (!isLeaf) {
    // 有子节点，展开下一级
    menu.value = true
    return
  }

  // 叶子节点且可选，关闭菜单
  if (selectableNow) {
    menu.value = false
  }
}

const clearSelection = () => {
  innerValue.value = []
  confirmedValue.value = []
  const emitValue = buildEmitValue([])
  emit('update:modelValue', emitValue)
  emit('change', emitValue)
}

const handleOpen = () => {
  if (props.disable || props.readonly) return
  menu.value = true
}

const validate = () => {
  return inputRef.value?.validate?.()
}

const containerStyle = computed(() => ({
  width: '100%',
}))

const menuStyle = computed(() => ({
  minWidth: `${props.menuMinWidth}px`,
  maxWidth: `${props.menuMaxWidth}px`,
}))

const columnStyle = computed(() => ({
  minWidth: `${props.columnMinWidth}px`,
  maxHeight: `${props.maxHeight}px`,
}))

defineExpose({
  validate,
})
</script>

<style scoped lang="scss">
.cascader-select {
  display: flex;
  flex-direction: column;
}

.cascader-menu {
  display: flex;
  min-width: 360px;
  max-width: 720px;
  background: white;
}

.cascader-column {
  min-width: 180px;
  max-height: 320px;
  overflow-y: auto;
  border-right: 1px solid rgba(0, 0, 0, 0.08);
}

.cascader-column:last-child {
  border-right: none;
}

.cascader-active {
  color: $primary;
  background: rgba($primary, 0.08);
}

.cascader-input :deep(.q-field__native) {
  cursor: pointer;
  caret-color: transparent;
}

// 覆盖 Quasar readonly 模式的虚线边框为实线
.cascader-input :deep(.q-field__control::before) {
  border-style: solid !important;
}
</style>
