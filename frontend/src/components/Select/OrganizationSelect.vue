<template>
  <sweet-select
    ref="selectRef"
    class="organization-select"
    v-bind="$attrs"
    :model-value="modelValue"
    :options="options"
    :multiple="multiple"
    :disable="disabled"
    :clearable="clearable"
    :loading="loading"
    emit-value
    map-options
    use-input
    input-debounce="300"
    option-value="value"
    option-label="label"
    option-disable="disabled"
    @update:model-value="handleModelValue"
    @filter="handleFilter"
  >
    <template v-for="(_, slotName) in forwardedSlots" #[slotName]="slotProps">
      <slot :name="slotName" v-bind="slotProps || {}" />
    </template>

    <template #no-option>
      <slot name="no-option" :failed="loadFailed">
        <q-item>
          <q-item-section class="text-grey-7">
            {{ loadFailed ? t('ui.failedToLoadOption') : t('ui.dataNotSelected') }}
          </q-item-section>
        </q-item>
      </slot>
    </template>
  </sweet-select>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, useSlots, watch } from 'vue'
import {
  queryOrganizationOptions,
  type OrganizationSelectorOption,
  type OrganizationSelectorType,
} from '@/api/services/org'
import SweetSelect from '@/components/Select/SweetSelect.vue'

const { t } = useI18n({ useScope: 'global' })

type OrganizationModelValue = number | number[] | null | undefined
type FilterUpdate = (callback: () => void) => void
type FilterAbort = () => void
const maxReplayIds = 100

defineOptions({
  inheritAttrs: false,
})

const props = withDefaults(
  defineProps<{
    modelValue?: OrganizationModelValue
    selectorType: OrganizationSelectorType
    multiple?: boolean
    disabled?: boolean
    clearable?: boolean
    includeHistory?: boolean
    selectedIds?: number[]
    keyword?: string
  }>(),
  {
    modelValue: null,
    multiple: false,
    disabled: false,
    clearable: true,
    includeHistory: false,
    selectedIds: () => [],
    keyword: '',
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: OrganizationModelValue): void
  (event: 'change', value: OrganizationModelValue): void
}>()

const slots = useSlots()
const selectRef = ref<InstanceType<typeof SweetSelect> | null>(null)
const options = ref<OrganizationSelectorOption[]>([])
const loading = ref(false)
const loadFailed = ref(false)
let requestSequence = 0

const forwardedSlots = computed(() =>
  Object.fromEntries(Object.entries(slots).filter(([slotName]) => slotName !== 'no-option')),
)

const normalizeIds = (values: unknown[]): number[] =>
  values
    .filter((value): value is number => Number.isInteger(value) && Number(value) > 0)
    .filter((value, index, allValues) => allValues.indexOf(value) === index)

const modelIds = computed(() => {
  if (Array.isArray(props.modelValue)) return normalizeIds(props.modelValue)
  return normalizeIds([props.modelValue])
})

const replayIds = computed(() =>
  normalizeIds([...modelIds.value, ...props.selectedIds]).slice(0, maxReplayIds),
)
const replayIdsKey = computed(() => replayIds.value.join(','))

const requestOptions = async (searchKeyword: string) => {
  const sequence = ++requestSequence
  loading.value = true
  loadFailed.value = false

  try {
    const normalizedKeyword = searchKeyword.trim()
    const result = await queryOrganizationOptions(props.selectorType, {
      page: 1,
      num: 50,
      only_effective: true,
      include_history: props.includeHistory,
      ...(normalizedKeyword ? { keyword: normalizedKeyword } : {}),
      ...(replayIds.value.length ? { selected_ids: replayIds.value } : {}),
    })
    if (sequence !== requestSequence) return undefined
    return result.items
  } catch {
    if (sequence === requestSequence) loadFailed.value = true
    return null
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

const refresh = async (searchKeyword = props.keyword) => {
  const items = await requestOptions(searchKeyword)
  if (items !== null && items !== undefined) options.value = items
}

const handleFilter = (searchKeyword: string, update: FilterUpdate, abort: FilterAbort) => {
  void requestOptions(searchKeyword).then((items) => {
    if (items === undefined) {
      abort()
      return
    }
    update(() => {
      options.value = items || []
    })
  })
}

const handleModelValue = (value: OrganizationModelValue) => {
  emit('update:modelValue', value)
  emit('change', value)
}

watch(
  [() => props.selectorType, () => props.includeHistory, () => props.keyword, replayIdsKey],
  () => {
    void refresh()
  },
  { immediate: true },
)

defineExpose({
  focus: () => selectRef.value?.focus?.(),
  blur: () => selectRef.value?.blur?.(),
  refresh,
})
</script>

<style scoped lang="scss">
.organization-select {
  width: 100%;
}
</style>
