<template>
  <aside class="inspector-panel">
    <q-tabs
      :model-value="tab"
      dense
      active-color="primary"
      indicator-color="primary"
      @update:model-value="emitTab"
    >
      <q-tab name="cell" icon="grid_on" :label="t('ui.cells')" />
      <q-tab name="data" icon="hub" :label="t('ui.data')" />
      <q-tab name="report" icon="settings" :label="t('ui.report')" />
    </q-tabs>

    <q-separator />

    <q-tab-panels :model-value="tab" animated class="inspector-tabs">
      <q-tab-panel name="cell">
        <div class="inspector-title">{{ t('ui.cells') }} {{ activeCellLabel }}</div>
        <div v-if="hasActiveCell" class="inspector-form">
          <q-input
            :model-value="cellValue"
            dense
            outlined
            :label="t('ui.titleAlien')"
            @update:model-value="$emit('update:cellValue', String($event || ''))"
          />
          <div v-if="bindingPreview" class="binding-preview">
            <q-icon name="data_object" />
            <span>{{ t('ui.tieExpression') }}{{ bindingPreview }}</span>
          </div>
          <q-select
            :model-value="bindingType"
            dense
            outlined
            emit-value
            map-options
            :label="t('ui.bindingType')"
            :options="bindingTypeOptions"
            @update:model-value="emitBindingType"
          />
          <q-select
            :model-value="bindingDatasetId"
            dense
            outlined
            emit-value
            map-options
            :label="t('ui.dataset')"
            :options="datasetOptions"
            @update:model-value="$emit('update:bindingDatasetId', String($event || ''))"
          />
          <q-select
            :model-value="bindingField"
            dense
            outlined
            emit-value
            map-options
            :label="t('ui.dataField')"
            :options="activeDatasetFieldOptions"
            @update:model-value="$emit('update:bindingField', String($event || ''))"
          />
          <q-input
            :model-value="formula"
            dense
            outlined
            :label="t('ui.formulaExpression')"
            @update:model-value="$emit('update:formula', String($event || ''))"
          />
          <div class="style-grid">
            <q-toggle
              :model-value="cellBold"
              :label="t('ui.putItOn')"
              @update:model-value="$emit('update:cellBold', !!$event)"
            />
            <q-select
              :model-value="cellAlign"
              dense
              outlined
              emit-value
              map-options
              :label="t('ui.alignment')"
              :options="alignOptions"
              @update:model-value="emitCellAlign"
            />
          </div>
        </div>
        <div v-else class="empty-note">{{ t('ui.pleaseSelectACell') }}</div>
      </q-tab-panel>

      <q-tab-panel name="data">
        <div class="inspector-title">{{ t('ui.datasetSettings') }}</div>
        <div class="inspector-form" v-if="selectedDataset">
          <q-input
            :model-value="selectedDataset.name"
            dense
            outlined
            :label="t('ui.datasetName')"
            @update:model-value="
              $emit('updateDatasetName', selectedDataset.id, String($event || ''))
            "
          />
          <q-input
            :model-value="
              selectedDataset.type === 'sql' ? selectedDataset.sql : selectedDataset.source_code
            "
            dense
            outlined
            readonly
            autogrow
            :label="selectedDataset.type === 'sql' ? 'SQL' : t('ui.sourceTable')"
          />
          <div class="dataset-role">
            <q-icon :name="selectedDataset.primary ? 'stars' : 'dataset_linked'" />
            <div>
              <strong>{{
                selectedDataset.primary ? t('ui.runMainDataSet') : t('ui.secondaryDataSet')
              }}</strong>
              <span>
                {{
                  selectedDataset.primary
                    ? t('ui.recommendedSourceAsDefaultPreviewDataPrivilegesAndParameters')
                    : t('ui.numbersCanBeObtainedFromAssociatedConfigurationParticipationReports')
                }}
              </span>
            </div>
          </div>
          <q-btn
            v-if="!selectedDataset.primary && selectedDataset.type === 'table'"
            outline
            color="primary"
            icon="stars"
            :label="t('ui.changeToRunMainDataSet')"
            @click="$emit('setPrimaryDataset', selectedDataset.id)"
          />
        </div>
        <div v-else class="empty-note">{{ t('ui.selectALeftDataSet') }}</div>

        <q-separator class="q-my-md" />
        <div class="inspector-title">{{ t('ui.dataSetAssociation') }}</div>
        <q-banner
          v-if="datasets.length > 1 && !datasetJoins.length"
          dense
          rounded
          class="join-warning"
        >
          <template #avatar>
            <q-icon name="warning" color="warning" />
          </template>
          {{ t('ui.multipleDataSetsHaveBeenAddedAndTheLayoutOf') }}
        </q-banner>
        <div class="join-list">
          <div v-for="join in datasetJoins" :key="join.id" class="join-row">
            <span>{{ joinLabel(join) }}</span>
            <q-btn
              flat
              dense
              round
              color="negative"
              icon="delete"
              @click="$emit('removeJoin', join.id)"
            />
          </div>
          <div v-if="!datasetJoins.length" class="empty-note">
            {{ t('ui.thereIsNoCorrelationAndMultipleDataSetsCannotBe') }}
          </div>
        </div>
        <q-btn
          outline
          color="primary"
          icon="add_link"
          :label="t('ui.addRelation')"
          :disable="datasets.length < 2"
          @click="$emit('addJoin')"
        />
      </q-tab-panel>

      <q-tab-panel name="report">
        <div class="inspector-title">{{ t('ui.reportingCapacity') }}</div>
        <div class="inspector-form">
          <q-input
            :model-value="category"
            dense
            outlined
            :label="t('ui.category')"
            @update:model-value="$emit('update:category', String($event || ''))"
          />
          <q-input
            :model-value="description"
            dense
            outlined
            autogrow
            :label="t('ui.descriptionLabel')"
            @update:model-value="$emit('update:description', String($event || ''))"
          />
          <q-input
            :model-value="primaryDataset?.source_code || ''"
            dense
            outlined
            readonly
            :label="t('ui.runMasterTable')"
          />
          <q-select
            :model-value="reportKind"
            dense
            outlined
            emit-value
            map-options
            :label="t('ui.dataDevelopment')"
            :options="reportKindOptions"
            @update:model-value="emitReportKind"
          >
            <q-tooltip> {{ t('ui.theLineIsThenCarriedOutByLineByLine') }} </q-tooltip>
          </q-select>
          <div class="runtime-grid">
            <q-select
              :model-value="runtimeDisplay"
              dense
              outlined
              emit-value
              map-options
              :label="t('ui.runPageBreak')"
              :options="runtimeDisplayOptions"
              @update:model-value="emitRuntimeDisplay"
            />
            <q-input
              :model-value="runtimePageSize"
              dense
              outlined
              type="number"
              :label="t('ui.itemsPerPage')"
              min="1"
              max="500"
              :disable="runtimeDisplay !== 'paged'"
              @update:model-value="emitRuntimePageSize"
            />
          </div>
          <div class="capability-list">
            <div>
              <q-icon name="security" />
              {{ t('ui.previewAndRunSuccessionMasterTableDataPrivileges') }}
            </div>
            <div><q-icon name="table_view" /> {{ t('ui.lineByLineFixedSumbarsOnly') }}</div>
            <div>
              <q-icon name="data_object" />
              {{ t('ui.saveTheSqlDatasetDesignFirstExecutionIsEnabledAfter') }}
            </div>
            <div><q-icon name="ios_share" /> {{ t('ui.excelExportStructureReserved') }}</div>
          </div>
        </div>
      </q-tab-panel>
    </q-tab-panels>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import type {
  ReportCellBindingType,
  ReportCellStyle,
  ReportDataset,
  ReportDatasetJoin,
  ReportKind,
  ReportRuntimeDisplayMode,
} from 'src/api/services/report'

const { t } = useI18n({ useScope: 'global' })

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
  reportKind: ReportKind
  reportKindOptions: Array<Option<ReportKind>>
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
  'update:reportKind': [value: ReportKind]
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

function emitReportKind(value: unknown) {
  if (value === 'detail' || value === 'summary') {
    emit('update:reportKind', value)
  }
}

function emitRuntimeDisplay(value: unknown) {
  if (value === 'paged' || value === 'all') {
    emit('update:runtimeDisplay', value)
  }
}

function emitRuntimePageSize(value: unknown) {
  const numeric = Number(value)
  emit(
    'update:runtimePageSize',
    Number.isFinite(numeric) ? Math.min(Math.max(numeric, 1), 500) : 20,
  )
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

.runtime-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 120px;
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

.join-warning {
  margin-bottom: 10px;
  color: #806000;
  background: #fff8e1;
  border: 1px solid #ffe6a3;
}

.join-row {
  justify-content: space-between;
  color: #172033;
}
</style>
