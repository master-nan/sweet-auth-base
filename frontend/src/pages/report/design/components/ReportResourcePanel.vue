<template>
  <aside class="resource-panel">
    <section class="panel-section">
      <div class="section-head">
        <div>
          <strong>{{ t('ui.dataset') }}</strong>
          <span>{{ datasets.length }} {{ t('ui.dataSets') }}</span>
        </div>
        <div class="section-actions">
          <q-btn
            v-if="datasets.length > 1"
            flat
            round
            dense
            size="sm"
            color="primary"
            icon="add_link"
            @click="$emit('addJoin')"
          >
            <q-tooltip>{{ t('ui.addRelation') }}</q-tooltip>
          </q-btn>
          <q-btn
            size="sm"
            color="primary"
            icon="add"
            :label="t('ui.create')"
            @click="$emit('openDataset')"
          />
        </div>
      </div>
      <div class="dataset-tools">
        <q-input
          v-model="fieldKeyword"
          dense
          outlined
          clearable
          :placeholder="t('ui.searchFieldNameEncoding')"
        >
          <template #prepend>
            <q-icon name="search" />
          </template>
        </q-input>
        <q-btn
          flat
          dense
          color="primary"
          :icon="allExpanded ? 'unfold_less' : 'unfold_more'"
          @click="toggleAllDatasets"
        >
          <q-tooltip>{{
            allExpanded ? t('ui.collectAllDataSets') : t('ui.expandAllDataSets')
          }}</q-tooltip>
        </q-btn>
      </div>

      <div class="dataset-list">
        <article
          v-for="dataset in datasets"
          :key="dataset.id"
          class="dataset-card"
          :class="{ active: dataset.id === selectedDatasetId }"
          @click="$emit('selectDataset', dataset.id)"
        >
          <div class="dataset-card__head">
            <q-btn
              flat
              dense
              round
              size="sm"
              :icon="isDatasetExpanded(dataset.id) ? 'expand_more' : 'chevron_right'"
              @click.stop="toggleDataset(dataset.id)"
            />
            <q-icon :name="dataset.type === 'sql' ? 'data_object' : 'table_chart'" />
            <div>
              <strong>{{ dataset.name }}</strong>
              <span>
                {{ dataset.type === 'sql' ? t('ui.sqlDataset') : dataset.source_code }}
                · {{ filteredFields(dataset).length }}/{{ dataset.fields.length }}
                {{ t('ui.field') }}
              </span>
            </div>
            <q-space />
            <q-badge v-if="dataset.primary" color="primary">{{ t('ui.primary') }}</q-badge>
            <q-btn
              flat
              size="sm"
              round
              color="primary"
              icon="edit"
              @click.stop="$emit('editDataset', dataset.id)"
            />
            <q-btn
              v-if="datasets.length > 1"
              flat
              size="sm"
              round
              color="negative"
              icon="delete"
              @click.stop="$emit('removeDataset', dataset.id)"
            />
          </div>

          <div v-show="isDatasetExpanded(dataset.id)" class="field-tree">
            <div v-if="dataset.fields.length" class="field-tree__hint">
              {{ t('ui.dragFieldsToCanvasCellsOrClickToBindThe') }}
            </div>
            <button
              v-for="field in filteredFields(dataset)"
              :key="`${dataset.id}-${field.code}`"
              class="field-row"
              draggable="true"
              @dragstart="$emit('startDragField', dataset, field)"
              @dragend="$emit('endDragField')"
              @click.stop="$emit('bindField', dataset, field)"
            >
              <q-icon :name="reportFieldIcon(field)" />
              <span>{{ field.name }}</span>
              <em>{{ field.code }}</em>
              <q-badge outline color="primary">{{ fieldRoleLabel(field) }}</q-badge>
            </button>
            <div v-if="dataset.fields.length && !filteredFields(dataset).length" class="empty-note">
              {{ t('ui.noMatchField') }}
            </div>
          </div>
        </article>
      </div>
    </section>

    <section class="panel-section">
      <div class="section-head">
        <div>
          <strong>{{ t('ui.parameters') }}</strong>
          <span>{{ parameters.length }} {{ t('ui.queryParameterCountSuffix') }}</span>
        </div>
        <q-btn
          size="sm"
          color="primary"
          icon="add"
          :label="t('ui.create')"
          @click="$emit('addParameter')"
        />
      </div>
      <div v-if="parameters.length" class="param-list">
        <div
          v-for="param in parameters"
          :key="param.id"
          class="param-row"
          :class="{ active: selectedParameterId === param.id }"
          @click="$emit('selectParameter', param.id)"
        >
          <q-icon name="tune" />
          <span>{{ param.label }}</span>
          <em>{{ parameterTargetLabel(param) }}</em>
          <q-btn
            flat
            size="sm"
            round
            color="primary"
            icon="edit"
            @click.stop="$emit('editParameter', param.id)"
          />
          <q-btn
            flat
            size="sm"
            round
            color="negative"
            icon="delete"
            @click.stop="$emit('removeParameter', param.id)"
          />
        </div>
      </div>
      <div v-else class="empty-note">{{ t('ui.noParametersForNow') }}</div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import type { ReportDataset, ReportField, ReportParameter } from '@/api/services/report'
import { reportFieldIcon } from '@/modules/report/sheet'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{
  datasets: ReportDataset[]
  parameters: ReportParameter[]
  selectedDatasetId: string
  selectedParameterId: string
}>()

defineEmits<{
  openDataset: []
  addJoin: []
  selectDataset: [id: string]
  editDataset: [id: string]
  removeDataset: [id: string]
  startDragField: [dataset: ReportDataset, field: ReportField]
  endDragField: []
  bindField: [dataset: ReportDataset, field: ReportField]
  addParameter: []
  selectParameter: [id: string]
  editParameter: [id: string]
  removeParameter: [id: string]
}>()

const expandedDatasetIds = ref<string[]>([])
const fieldKeyword = ref('')
const allExpanded = computed(
  () => props.datasets.length > 0 && expandedDatasetIds.value.length === props.datasets.length,
)

watch(
  () => props.datasets.map((item) => item.id),
  (ids) => {
    if (!expandedDatasetIds.value.length) {
      expandedDatasetIds.value = ids.slice(0, 1)
      return
    }
    expandedDatasetIds.value = expandedDatasetIds.value.filter((id) => ids.includes(id))
  },
  { immediate: true },
)

function isDatasetExpanded(id: string) {
  return expandedDatasetIds.value.includes(id)
}

function toggleDataset(id: string) {
  if (expandedDatasetIds.value.includes(id)) {
    expandedDatasetIds.value = expandedDatasetIds.value.filter((item) => item !== id)
  } else {
    expandedDatasetIds.value = [...expandedDatasetIds.value, id]
  }
}

function toggleAllDatasets() {
  expandedDatasetIds.value = allExpanded.value ? [] : props.datasets.map((item) => item.id)
}

function filteredFields(dataset: ReportDataset) {
  const keyword = fieldKeyword.value.trim().toLowerCase()
  if (!keyword) return dataset.fields
  return dataset.fields.filter((field) =>
    `${field.name} ${field.code}`.toLowerCase().includes(keyword),
  )
}

function fieldRoleLabel(field: ReportField) {
  if (field.role === 'metric') return t('ui.indicators')
  if (field.role === 'dimension') return t('ui.dimensions')
  if (field.role === 'time') return t('ui.time')
  return t('ui.text')
}

function parameterTargetLabel(param: ReportParameter) {
  const dataset = props.datasets.find((item) => item.id === param.dataset_id)
  const field = dataset?.fields.find((item) => item.code === param.field)
  return t('ui.resourceSummaryParts', {
    value1: dataset?.name || t('ui.unbound'),
    value2: field?.name || param.field,
    value3: param.operator,
  })
}
</script>

<style scoped lang="scss">
.resource-panel {
  min-height: 0;
  overflow: auto;
  border-right: 1px solid #dfe5f2;
  background: #fbfcff;
}

.panel-section {
  padding: 10px;
  border-bottom: 1px solid #e7ecf6;
}

.section-head,
.dataset-card__head {
  display: flex;
  align-items: center;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.section-head {
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.dataset-tools {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 32px;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.section-head strong,
.section-head span {
  display: block;
}

.section-head strong {
  font-size: 16px;
  font-weight: 900;
}

.section-head span,
.dataset-card__head span,
.field-row em,
.param-row em {
  color: #71809a;
}

.section-head span {
  margin-top: 2px;
  font-size: 12px;
}

.dataset-list,
.field-tree,
.param-list {
  display: grid;
  gap: 6px;
}

.field-tree__hint {
  padding: 2px 8px 4px;
  color: #71809a;
  font-size: 12px;
}

.dataset-card {
  padding: 2px;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fff;
  cursor: pointer;
}

.dataset-card.active {
  border-color: var(--q-primary);
  box-shadow: 0 0 0 2px rgba(115, 103, 240, 0.12);
}

.dataset-card__head {
  gap: 8px;
  margin-bottom: 6px;
}

.dataset-card__head strong,
.dataset-card__head span {
  display: block;
}

.field-row,
.param-row {
  width: 100%;
  min-height: 34px;
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 8px;
  border: 1px solid #e7ecf6;
  border-radius: 6px;
  background: #fff;
  text-align: left;
  cursor: grab;
}

.field-row {
  padding: 6px 8px;
}

.param-row {
  padding: 9px 10px;
  grid-template-columns: 20px minmax(0, 1fr) auto 30px 30px;
  cursor: pointer;
}

.field-row span,
.param-row span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.param-row.active,
.field-row:hover {
  border-color: var(--q-primary);
  background: #f6f4ff;
}

.empty-note {
  padding: 14px;
  color: #71809a;
  text-align: center;
}
</style>
