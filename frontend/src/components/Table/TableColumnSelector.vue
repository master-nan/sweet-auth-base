<template>
  <q-btn
    outline
    dense
    color="primary"
    icon="view_column"
    class="table-column-selector__trigger"
    :aria-label="t('ui.setShowbar')"
    :disable="disabled || !columnOptions.length"
  >
    <q-badge floating rounded color="primary" :label="modelValue.length" />
    <q-tooltip>{{ t('ui.setShowbar') }}</q-tooltip>

    <q-menu anchor="bottom right" self="top right" :offset="[0, 8]">
      <div class="table-column-selector__panel q-pa-sm">
        <div class="row items-center justify-between q-px-xs q-pb-xs">
          <div class="text-subtitle2 text-weight-medium">{{ t('ui.showColumns') }}</div>
          <div class="row items-center q-gutter-xs">
            <q-btn flat dense color="primary" :label="t('ui.selectAll')" @click="selectAll" />
            <q-btn
              flat
              dense
              color="primary"
              :label="t('ui.restoreDefault')"
              @click="restoreDefault"
            />
          </div>
        </div>

        <q-input
          v-model="keyword"
          dense
          outlined
          clearable
          debounce="150"
          :placeholder="t('ui.searchColumns')"
          class="q-mb-xs"
        >
          <template #prepend><q-icon name="search" /></template>
        </q-input>

        <q-list dense class="table-column-selector__list">
          <q-item
            v-for="column in filteredColumns"
            :key="column.value"
            clickable
            @click="toggleColumn(column.value)"
          >
            <q-item-section avatar>
              <q-checkbox
                dense
                color="primary"
                :model-value="selectedColumnNames.has(column.value)"
                :disable="isLastSelected(column.value)"
                @click.stop
                @update:model-value="toggleColumn(column.value)"
              />
            </q-item-section>
            <q-item-section>
              <q-item-label>{{ column.label }}</q-item-label>
            </q-item-section>
          </q-item>

          <q-item v-if="!filteredColumns.length">
            <q-item-section class="text-grey-6 text-center q-py-sm">{{
              t('ui.noMatchingColumnsFound')
            }}</q-item-section>
          </q-item>
        </q-list>
      </div>
    </q-menu>
  </q-btn>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'

const { t } = useI18n({ useScope: 'global' })

type TableColumn = NonNullable<QTableProps['columns']>[number]

const props = withDefaults(
  defineProps<{
    modelValue: string[]
    columns?: readonly TableColumn[] | undefined
    disabled?: boolean
  }>(),
  {
    disabled: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const keyword = ref('')
const defaultVisibleColumns = ref<string[]>([])

const columnOptions = computed(() =>
  (props.columns ?? []).map((column) => ({
    label: String(column.label || column.name),
    value: column.name,
  })),
)

const selectedColumnNames = computed(() => new Set(props.modelValue))
const filteredColumns = computed(() => {
  const term = keyword.value.trim().toLocaleLowerCase()
  if (!term) return columnOptions.value
  return columnOptions.value.filter(
    (column) =>
      column.label.toLocaleLowerCase().includes(term) ||
      column.value.toLocaleLowerCase().includes(term),
  )
})

watch(
  [columnOptions, () => props.modelValue],
  ([options, selected]) => {
    if (defaultVisibleColumns.value.length || !options.length || !selected.length) return
    const available = new Set(options.map((option) => option.value))
    defaultVisibleColumns.value = selected.filter((name) => available.has(name))
  },
  { immediate: true },
)

const normalizedSelection = (selected: ReadonlySet<string>) =>
  columnOptions.value.filter((column) => selected.has(column.value)).map((column) => column.value)

const toggleColumn = (name: string) => {
  const selected = new Set(props.modelValue)
  if (selected.has(name)) {
    if (selected.size <= 1) return
    selected.delete(name)
  } else {
    selected.add(name)
  }
  emit('update:modelValue', normalizedSelection(selected))
}

const selectAll = () =>
  emit(
    'update:modelValue',
    columnOptions.value.map((column) => column.value),
  )

const restoreDefault = () => {
  const available = new Set(columnOptions.value.map((column) => column.value))
  const defaults = defaultVisibleColumns.value.filter((name) => available.has(name))
  emit('update:modelValue', defaults.length ? defaults : [...available])
}

const isLastSelected = (name: string) =>
  props.modelValue.length === 1 && selectedColumnNames.value.has(name)
</script>

<style scoped>
.table-column-selector__trigger {
  width: 40px;
  min-width: 40px;
  height: 40px;
}

.table-column-selector__panel {
  width: 300px;
  max-width: calc(100vw - 24px);
}

.table-column-selector__list {
  max-height: 280px;
  overflow-y: auto;
}
</style>
