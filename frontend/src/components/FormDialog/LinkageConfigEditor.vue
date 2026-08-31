<template>
  <div class="linkage-editor" :class="{ 'linkage-editor--dark': isDarkMode }">
    <div class="row items-center q-mb-sm">
      <q-toggle
        v-model="state.enabled"
        color="primary"
        :label="t('ui.enableConnection')"
        :dark="isDarkMode"
        :disable="disable"
      />
      <q-space />
      <q-chip v-if="state.enabled" dense square color="primary" text-color="white">
        {{ state.mode === 'cascader' ? t('ui.cascadeField') : t('ui.associatedFields') }}
      </q-chip>
    </div>

    <div v-if="state.enabled" class="linkage-editor__body">
      <div class="row q-col-gutter-md">
        <div class="col-12 col-sm-4">
          <q-select
            v-model="state.mode"
            :options="modeOptions"
            :label="t('ui.mode')"
            outlined
            dense
            :dark="isDarkMode"
            emit-value
            map-options
            :disable="disable"
          />
        </div>
        <div class="col-12 col-sm-8">
          <q-input
            v-model.trim="state.tableCode"
            :label="t('ui.relatedTableCode')"
            outlined
            dense
            :dark="isDarkMode"
            :disable="disable"
          />
        </div>

        <div class="col-12 col-sm-4">
          <q-input
            v-model.trim="state.labelKey"
            :label="t('ui.showFields')"
            outlined
            dense
            :dark="isDarkMode"
            :disable="disable"
          />
        </div>
        <div class="col-12 col-sm-4">
          <q-input
            v-model.trim="state.valueKey"
            :label="t('ui.valueField')"
            outlined
            dense
            :dark="isDarkMode"
            :disable="disable"
          />
        </div>
        <div class="col-12 col-sm-4">
          <q-input
            v-model.number="state.pageSize"
            :label="t('ui.loaded')"
            outlined
            dense
            :dark="isDarkMode"
            type="number"
            :disable="disable"
          />
        </div>

        <template v-if="state.mode === 'cascader'">
          <div class="col-12 col-sm-4">
            <q-input
              v-model.trim="state.parentKey"
              :label="t('ui.parentFields')"
              outlined
              dense
              :dark="isDarkMode"
              :disable="disable"
            />
          </div>
          <div class="col-12 col-sm-4">
            <q-select
              v-model="state.selectable"
              :options="selectableOptions"
              :label="t('ui.optionalNodes')"
              outlined
              dense
              :dark="isDarkMode"
              emit-value
              map-options
              :disable="disable"
            />
          </div>
          <div class="col-12 col-sm-4 flex items-center">
            <q-toggle
              v-model="state.showPath"
              color="primary"
              :label="t('ui.showFullPath')"
              :dark="isDarkMode"
              :disable="disable"
            />
          </div>
        </template>

        <div class="col-12">
          <div class="mapping-box">
            <div class="row items-center q-mb-sm">
              <div class="text-subtitle2 text-weight-medium">{{ t('ui.filterMap') }}</div>
              <q-space />
              <q-btn
                flat
                dense
                color="primary"
                icon="add"
                :label="t('ui.add')"
                :disable="disable"
                @click="() => addMappingRow()"
              />
            </div>
            <div v-if="mappingRows.length === 0" class="text-caption text-grey-6 q-py-xs">
              {{ t('ui.noFilterFieldConfigured') }}
            </div>
            <div
              v-for="row in mappingRows"
              :key="row.id"
              class="row q-col-gutter-sm items-center q-mb-sm"
            >
              <div class="col-12 col-sm-5">
                <q-input
                  v-model.trim="row.target"
                  :label="t('ui.relatedTableField')"
                  outlined
                  dense
                  :dark="isDarkMode"
                  :disable="disable"
                />
              </div>
              <div class="col-12 col-sm-5">
                <q-input
                  v-model.trim="row.source"
                  :label="t('ui.currentTableFields')"
                  outlined
                  dense
                  :dark="isDarkMode"
                  :disable="disable"
                />
              </div>
              <div class="col-12 col-sm-2 text-right">
                <q-btn
                  flat
                  round
                  dense
                  color="negative"
                  icon="delete"
                  :disable="disable"
                  @click="removeMappingRow(row.id)"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, nextTick, reactive, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { decodeHtmlEntities, parseJsonSafe } from 'src/utils/field-metadata'

const { t } = useI18n({ useScope: 'global' })

defineOptions({ name: 'LinkageConfigEditor' })

type LinkageMode = 'relation' | 'cascader'
type MappingRow = {
  id: number
  target: string
  source: string
}

const props = withDefaults(
  defineProps<{
    modelValue?: string | Record<string, unknown> | null
    disable?: boolean
  }>(),
  {
    modelValue: '',
    disable: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
}>()

const $q = useQuasar()
const isDarkMode = computed(() => Boolean($q?.dark?.isActive))

const modeOptions = [
  {
    get label() {
      return t('ui.associationSelection')
    },
    value: 'relation',
  },
  {
    get label() {
      return t('ui.cascadeSelection')
    },
    value: 'cascader',
  },
]

const selectableOptions = [
  {
    get label() {
      return t('ui.anyNode')
    },
    value: 'any',
  },
  {
    get label() {
      return t('ui.leafNode')
    },
    value: 'leaf',
  },
  {
    get label() {
      return t('ui.specifyLevel')
    },
    value: 'level',
  },
]

const state = reactive({
  enabled: false,
  mode: 'relation' as LinkageMode,
  tableCode: '',
  labelKey: 'name',
  valueKey: 'id',
  parentKey: 'parent_id',
  pageSize: 200,
  targetMenuId: null as number | null,
  selectable: 'any',
  showPath: true,
})

const mappingRows = ref<MappingRow[]>([])
let rowId = 1
let hydrating = false
let lastEmitted = ''

const addMappingRow = (target = '', source = '') => {
  mappingRows.value.push({ id: rowId++, target, source })
}

const removeMappingRow = (id: number) => {
  mappingRows.value = mappingRows.value.filter((row) => row.id !== id)
}

const resetState = () => {
  state.enabled = false
  state.mode = 'relation'
  state.tableCode = ''
  state.labelKey = 'name'
  state.valueKey = 'id'
  state.parentKey = 'parent_id'
  state.pageSize = 200
  state.targetMenuId = null
  state.selectable = 'any'
  state.showPath = true
  mappingRows.value = []
}

const parseModelValue = (value: typeof props.modelValue) => {
  if (!value) return null
  if (typeof value === 'object') return value
  const raw = decodeHtmlEntities(String(value).trim())
  if (!raw) return null
  return parseJsonSafe(raw)
}

const hydrate = (value: typeof props.modelValue) => {
  hydrating = true
  resetState()
  const parsed = parseModelValue(value)
  const linkage = parsed && typeof parsed === 'object' ? (parsed as any).linkage : null
  if (linkage) {
    state.enabled = !!linkage.enabled
    state.mode = linkage.mode === 'cascader' ? 'cascader' : 'relation'
    state.tableCode = linkage.tableCode || ''
    state.labelKey = linkage.labelKey || 'name'
    state.valueKey = linkage.valueKey || 'id'
    state.parentKey = linkage.parentKey || 'parent_id'
    state.pageSize = Number(linkage.pageSize || linkage.searchPageSize || 200)
    state.targetMenuId = Number(linkage.targetMenuId) > 0 ? Number(linkage.targetMenuId) : null
    state.selectable = linkage.selectable || 'any'
    state.showPath = linkage.showPath !== false

    const mapping = linkage.filterMapping || {}
    Object.entries(mapping).forEach(([target, source]) => {
      if (typeof source === 'string') addMappingRow(target, source)
    })
  }
  void nextTick(() => {
    hydrating = false
  })
}

const emitConfig = () => {
  if (hydrating) return
  if (!state.enabled) {
    lastEmitted = ''
    emit('update:modelValue', '')
    return
  }

  const filterMapping = mappingRows.value.reduce<Record<string, string>>((result, row) => {
    if (row.target && row.source) {
      result[row.target] = row.source
    }
    return result
  }, {})

  const linkage: Record<string, unknown> = {
    enabled: true,
    mode: state.mode,
    labelKey: state.labelKey || 'name',
    valueKey: state.valueKey || 'id',
  }
  if (state.tableCode) linkage.tableCode = state.tableCode
  if (state.pageSize && state.pageSize > 0) linkage.pageSize = state.pageSize
  if (state.targetMenuId && state.targetMenuId > 0) linkage.targetMenuId = state.targetMenuId
  if (Object.keys(filterMapping).length > 0) linkage.filterMapping = filterMapping
  if (state.mode === 'cascader') {
    linkage.parentKey = state.parentKey || 'parent_id'
    linkage.selectable = state.selectable || 'any'
    linkage.showPath = state.showPath
  }

  lastEmitted = JSON.stringify({ linkage })
  emit('update:modelValue', lastEmitted)
}

watch(
  () => props.modelValue,
  (value) => {
    if (typeof value === 'string' && value === lastEmitted) return
    hydrate(value)
  },
  { immediate: true },
)

watch([state, mappingRows], emitConfig, { deep: true })
</script>

<style scoped lang="scss">
.linkage-editor {
  border: 1px solid rgba(25, 118, 210, 0.16);
  border-radius: 8px;
  padding: 12px;
  background: #fbfdff;
}

.linkage-editor__body {
  padding-top: 8px;
  border-top: 1px solid rgba(15, 23, 42, 0.08);
}

.mapping-box {
  border: 1px dashed rgba(15, 23, 42, 0.16);
  border-radius: 8px;
  padding: 12px;
  background: #ffffff;
}

.linkage-editor--dark {
  border-color: rgba(255, 255, 255, 0.12);
  background: var(--app-dark-surface-soft);
  color: var(--app-dark-text);
}

.linkage-editor--dark .linkage-editor__body {
  border-top-color: rgba(255, 255, 255, 0.1);
}

.linkage-editor--dark .mapping-box {
  border-color: rgba(255, 255, 255, 0.14);
  background: var(--app-dark-surface);
}
</style>
