<template>
  <aside class="inspector-panel">
    <q-tabs
      :model-value="tab"
      dense
      active-color="primary"
      indicator-color="primary"
      @update:model-value="emitTab"
    >
      <q-tab name="cell" icon="grid_on" label="单元格" />
      <q-tab name="data" icon="hub" label="数据" />
      <q-tab name="report" icon="settings" label="报表" />
    </q-tabs>

    <q-separator />

    <q-tab-panels :model-value="tab" animated class="inspector-tabs">
      <q-tab-panel name="cell">
        <div class="inspector-title">单元格 {{ activeCellLabel }}</div>
        <div v-if="hasActiveCell" class="inspector-form">
          <q-input
            :model-value="cellValue"
            dense
            outlined
            label="标题 / 别名"
            @update:model-value="$emit('update:cellValue', String($event || ''))"
          />
          <div v-if="bindingPreview" class="binding-preview">
            <q-icon name="data_object" />
            <span>绑定表达式：{{ bindingPreview }}</span>
          </div>
          <q-select
            :model-value="bindingType"
            dense
            outlined
            emit-value
              map-options
              label="绑定类型"
              :options="bindingTypeOptions"
            @update:model-value="emitBindingType"
          />
          <q-select
            :model-value="bindingDatasetId"
            dense
            outlined
            emit-value
            map-options
            label="数据集"
            :options="datasetOptions"
            @update:model-value="$emit('update:bindingDatasetId', String($event || ''))"
          />
          <q-select
            :model-value="bindingField"
            dense
            outlined
            emit-value
            map-options
            label="数据字段"
            :options="activeDatasetFieldOptions"
            @update:model-value="$emit('update:bindingField', String($event || ''))"
          />
          <q-input
            :model-value="formula"
            dense
            outlined
            label="公式 / 表达式"
            @update:model-value="$emit('update:formula', String($event || ''))"
          />
          <div class="style-grid">
            <q-toggle
              :model-value="cellBold"
              label="加粗"
              @update:model-value="$emit('update:cellBold', !!$event)"
            />
            <q-select
              :model-value="cellAlign"
              dense
              outlined
              emit-value
              map-options
              label="对齐"
              :options="alignOptions"
              @update:model-value="emitCellAlign"
            />
          </div>
        </div>
        <div v-else class="empty-note">请选择一个单元格</div>
      </q-tab-panel>

      <q-tab-panel name="data">
        <div class="inspector-title">数据集设置</div>
        <div class="inspector-form" v-if="selectedDataset">
          <q-input
            :model-value="selectedDataset.name"
            dense
            outlined
            label="数据集名称"
            @update:model-value="$emit('updateDatasetName', selectedDataset.id, String($event || ''))"
          />
          <q-input
            :model-value="selectedDataset.type === 'sql' ? selectedDataset.sql : selectedDataset.source_code"
            dense
            outlined
            readonly
            autogrow
            :label="selectedDataset.type === 'sql' ? 'SQL' : '来源表'"
          />
          <div class="dataset-role">
            <q-icon :name="selectedDataset.primary ? 'stars' : 'dataset_linked'" />
            <div>
              <strong>{{ selectedDataset.primary ? '运行主数据集' : '辅助数据集' }}</strong>
              <span>
                {{ selectedDataset.primary ? '作为默认预览、数据权限和参数推荐来源' : '可通过关联配置参与报表取数' }}
              </span>
            </div>
          </div>
          <q-btn
            v-if="!selectedDataset.primary && selectedDataset.type === 'table'"
            outline
            color="primary"
            icon="stars"
            label="改为运行主数据集"
            @click="$emit('setPrimaryDataset', selectedDataset.id)"
          />
        </div>
        <div v-else class="empty-note">请选择左侧数据集</div>

        <q-separator class="q-my-md" />
        <div class="inspector-title">数据集关联</div>
        <div class="join-list">
          <div v-for="join in datasetJoins" :key="join.id" class="join-row">
            <span>{{ joinLabel(join) }}</span>
            <q-btn flat dense round color="negative" icon="delete" @click="$emit('removeJoin', join.id)" />
          </div>
          <div v-if="!datasetJoins.length" class="empty-note">暂无关联，多个数据集无法互相取值。</div>
        </div>
        <q-btn
          outline
          color="primary"
          icon="add_link"
          label="新增关联"
          :disable="datasets.length < 2"
          @click="$emit('addJoin')"
        />
      </q-tab-panel>

      <q-tab-panel name="report">
        <div class="inspector-title">报表能力</div>
        <div class="inspector-form">
          <q-input
            :model-value="category"
            dense
            outlined
            label="分类"
            @update:model-value="$emit('update:category', String($event || ''))"
          />
          <q-input
            :model-value="description"
            dense
            outlined
            autogrow
            label="说明"
            @update:model-value="$emit('update:description', String($event || ''))"
          />
          <q-input
            :model-value="primaryDataset?.source_code || ''"
            dense
            outlined
            readonly
            label="运行主表"
          />
          <q-select
            :model-value="runtimeDisplay"
            dense
            outlined
            emit-value
            map-options
            label="运行展示"
            :options="runtimeDisplayOptions"
            @update:model-value="emitRuntimeDisplay"
          />
          <q-input
            v-if="runtimeDisplay === 'paged'"
            :model-value="runtimePageSize"
            dense
            outlined
            type="number"
            label="每页条数"
            min="1"
            max="500"
            @update:model-value="emitRuntimePageSize"
          />
          <div class="capability-list">
            <div><q-icon name="security" /> 预览和运行继承主表数据权限</div>
            <div><q-icon name="table_view" /> 当前版本保存类 Excel 单元格设计</div>
            <div><q-icon name="data_object" /> SQL 数据集先保存设计，执行会在安全校验后开放</div>
            <div><q-icon name="ios_share" /> Excel 导出结构已预留</div>
          </div>
        </div>
      </q-tab-panel>
    </q-tab-panels>
  </aside>
</template>

<script setup lang="ts">
import type {
  ReportCellBindingType,
  ReportCellStyle,
  ReportDataset,
  ReportDatasetJoin,
  ReportRuntimeDisplayMode,
} from 'src/api/services/report'

type InspectorTab = 'cell' | 'data' | 'report'
type Option<T = string> = { label: string; value: T }

const props = defineProps<{
  tab: InspectorTab
  activeCellLabel: string
  hasActiveCell: boolean
  cellValue: string
  bindingPreview: string
  bindingType: ReportCellBindingType
  bindingDatasetId: string
  bindingField: string
  formula: string
  cellBold: boolean
  cellAlign: NonNullable<ReportCellStyle['align']>
  bindingTypeOptions: Array<Option<ReportCellBindingType>>
  datasetOptions: Array<Option>
  activeDatasetFieldOptions: Array<Option>
  alignOptions: Array<Option<NonNullable<ReportCellStyle['align']>>>
  selectedDataset?: ReportDataset | undefined
  primaryDataset?: ReportDataset | undefined
  datasets: ReportDataset[]
  datasetJoins: ReportDatasetJoin[]
  category: string
  description: string
  runtimeDisplay: ReportRuntimeDisplayMode
  runtimePageSize: number
  runtimeDisplayOptions: Array<Option<ReportRuntimeDisplayMode>>
}>()

const emit = defineEmits<{
  'update:tab': [value: InspectorTab]
  'update:cellValue': [value: string]
  'update:bindingType': [value: ReportCellBindingType]
  'update:bindingDatasetId': [value: string]
  'update:bindingField': [value: string]
  'update:formula': [value: string]
  'update:cellBold': [value: boolean]
  'update:cellAlign': [value: NonNullable<ReportCellStyle['align']>]
  updateDatasetName: [datasetId: string, name: string]
  setPrimaryDataset: [datasetId: string]
  addJoin: []
  removeJoin: [id: string]
  'update:category': [value: string]
  'update:description': [value: string]
  'update:runtimeDisplay': [value: ReportRuntimeDisplayMode]
  'update:runtimePageSize': [value: number]
}>()

function emitTab(value: unknown) {
  if (value === 'cell' || value === 'data' || value === 'report') emit('update:tab', value)
}

function emitBindingType(value: unknown) {
  if (
    value === 'static' ||
    value === 'field' ||
    value === 'group' ||
    value === 'sum' ||
    value === 'count' ||
    value === 'formula'
  ) {
    emit('update:bindingType', value)
  }
}

function emitCellAlign(value: unknown) {
  if (value === 'left' || value === 'center' || value === 'right') {
    emit('update:cellAlign', value)
  }
}

function emitRuntimeDisplay(value: unknown) {
  if (value === 'paged' || value === 'all') {
    emit('update:runtimeDisplay', value)
  }
}

function emitRuntimePageSize(value: unknown) {
  const numeric = Number(value)
  emit('update:runtimePageSize', Number.isFinite(numeric) ? Math.min(Math.max(numeric, 1), 500) : 20)
}

function datasetName(id: string) {
  return props.datasets.find((item) => item.id === id)?.name || id
}

function fieldName(datasetId: string, fieldCode: string) {
  const dataset = props.datasets.find((item) => item.id === datasetId)
  return dataset?.fields.find((item) => item.code === fieldCode)?.name || fieldCode
}

function joinLabel(join: ReportDatasetJoin) {
  return `${datasetName(join.left_dataset_id)}.${fieldName(join.left_dataset_id, join.left_field)} ${join.join_type === 'inner' ? '=' : '⇐'} ${datasetName(join.right_dataset_id)}.${fieldName(join.right_dataset_id, join.right_field)}`
}
</script>

<style scoped lang="scss">
.inspector-panel {
  min-height: 0;
  overflow: auto;
  border-left: 1px solid #dfe5f2;
  background: #fbfcff;
}

.inspector-tabs {
  background: transparent;
}

.inspector-title {
  margin-bottom: 14px;
  font-size: 16px;
  font-weight: 900;
}

.inspector-form,
.capability-list {
  display: grid;
  gap: 10px;
}

.empty-note {
  padding: 14px;
  color: #71809a;
  text-align: center;
}

.style-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 10px;
}

.capability-list div {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid #e7ecf6;
  border-radius: 8px;
  background: #fff;
  color: #5f6f88;
}

.binding-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px dashed #cfd6e6;
  border-radius: 8px;
  color: #5f6f88;
  background: #fff;
  font-size: 12px;
}

.dataset-role,
.join-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border: 1px solid #e7ecf6;
  border-radius: 8px;
  background: #fff;
}

.dataset-role strong,
.dataset-role span {
  display: block;
}

.dataset-role span {
  margin-top: 2px;
  color: #71809a;
  font-size: 12px;
}

.join-list {
  display: grid;
  gap: 8px;
  margin-bottom: 10px;
}

.join-row {
  justify-content: space-between;
  color: #172033;
}
</style>
