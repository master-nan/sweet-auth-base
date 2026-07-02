<template>
  <aside class="resource-panel">
    <section class="panel-section">
      <div class="section-head">
        <div>
          <strong>数据集</strong>
          <span>{{ datasets.length }} 个数据集</span>
        </div>
        <q-btn size="sm" color="primary" icon="add" label="新增" @click="$emit('openDataset')" />
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
              <span>{{ dataset.type === 'sql' ? 'SQL 数据集' : dataset.source_code }}</span>
            </div>
            <q-space />
            <q-badge v-if="dataset.primary" color="primary">主</q-badge>
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
            <button
              v-for="field in dataset.fields"
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
          </div>
        </article>
      </div>
    </section>

    <section class="panel-section">
      <div class="section-head">
        <div>
          <strong>参数</strong>
          <span>{{ parameters.length }} 个查询参数</span>
        </div>
        <q-btn size="sm" color="primary" icon="add" label="新增" @click="$emit('addParameter')" />
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
      <div v-else class="empty-note">暂无参数</div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { ReportDataset, ReportField, ReportParameter } from 'src/api/services/report'
import { reportFieldIcon } from 'src/modules/report/sheet'

const props = defineProps<{
  datasets: ReportDataset[]
  parameters: ReportParameter[]
  selectedDatasetId: string
  selectedParameterId: string
}>()

defineEmits<{
  openDataset: []
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

function fieldRoleLabel(field: ReportField) {
  if (field.role === 'metric') return '指标'
  if (field.role === 'dimension') return '维度'
  if (field.role === 'time') return '时间'
  return '文本'
}

function parameterTargetLabel(param: ReportParameter) {
  const dataset = props.datasets.find((item) => item.id === param.dataset_id)
  const field = dataset?.fields.find((item) => item.code === param.field)
  return `${dataset?.name || '未绑定'} · ${field?.name || param.field} · ${param.operator}`
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

.section-head {
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
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
