<template>
  <aside class="inspector-panel">
    <q-tabs
      :model-value="tab"
      class="inspector-nav"
      dense
      inline-label
      narrow-indicator
      no-caps
      align="justify"
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
      <q-tab-panel name="cell" class="inspector-pane">
        <div class="context-head">
          <div>
            <span>{{ t('ui.cells') }}</span>
            <strong>{{ activeCellLabel }}</strong>
          </div>
          <q-badge v-if="hasActiveCell" outline color="primary">
            {{ bindingTypeLabel }}
          </q-badge>
        </div>
        <div v-if="hasActiveCell" class="panel-stack">
          <section class="setting-section">
          <q-input
            v-if="bindingType !== 'formula'"
            :model-value="cellValue"
            dense
            outlined
            :label="t('ui.titleAlien')"
            @update:model-value="$emit('update:cellValue', String($event || ''))"
          />
          <div v-if="bindingPreview" class="binding-preview">
            <q-icon name="data_object" />
            <span>{{ bindingPreview }}</span>
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
          <div
            v-if="bindingType !== 'static' && bindingType !== 'formula'"
            class="two-column-grid"
          >
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
          </div>
          <q-input
            v-if="bindingType === 'formula'"
            :model-value="formula"
            dense
            outlined
            :label="t('ui.formulaExpression')"
            :hint="t('ui.formulaSyntaxHint')"
            :error="!!formulaError"
            :error-message="formulaError"
            @update:model-value="$emit('update:formula', String($event || ''))"
          />
          </section>
          <section class="setting-section style-row">
            <q-toggle
              :model-value="cellBold"
              dense
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
          </section>
        </div>
        <div v-else class="empty-note">{{ t('ui.pleaseSelectACell') }}</div>
      </q-tab-panel>

      <q-tab-panel name="data" class="inspector-pane">
        <div class="context-head">
          <div>
            <span>{{ t('ui.datasetSettings') }}</span>
            <strong>{{ selectedDataset?.name || t('ui.unbound') }}</strong>
          </div>
          <q-badge v-if="selectedDataset" :color="selectedDataset.primary ? 'primary' : 'grey-7'">
            {{ selectedDataset.primary ? t('ui.mainDataSet') : t('ui.secondaryDataSet') }}
          </q-badge>
        </div>
        <div v-if="selectedDataset" class="panel-stack">
          <section class="setting-section dataset-settings">
            <q-input
              :model-value="selectedDataset.name"
              dense
              outlined
              :label="t('ui.datasetName')"
              @update:model-value="
                $emit('updateDatasetName', selectedDataset.id, String($event || ''))
              "
            />
            <div class="definition-row">
              <q-icon :name="selectedDataset.type === 'sql' ? 'data_object' : 'table_chart'" />
              <div>
                <span>{{ selectedDataset.type === 'sql' ? 'SQL' : t('ui.sourceTable') }}</span>
                <strong>{{
                  selectedDataset.type === 'sql' ? selectedDataset.sql : selectedDataset.source_code
                }}</strong>
              </div>
              <q-btn
                v-if="!selectedDataset.primary && selectedDataset.type === 'table'"
                flat
                dense
                round
                color="primary"
                icon="stars"
                @click="$emit('setPrimaryDataset', selectedDataset.id)"
              >
                <q-tooltip>{{ t('ui.changeToRunMainDataSet') }}</q-tooltip>
              </q-btn>
            </div>
          </section>
          <section class="setting-section">
            <div class="section-head">
              <div>
                <strong>{{ t('ui.dataSetAssociation') }}</strong>
                <span>{{ datasetJoins.length }}</span>
              </div>
              <q-btn
                outline
                dense
                color="primary"
                icon="add_link"
                :label="t('ui.addRelation')"
                :disable="datasets.length < 2"
                @click="$emit('addJoin')"
              />
            </div>
            <div v-if="datasets.length > 1 && !datasetJoins.length" class="join-warning">
              <q-icon name="warning_amber" />
              <span>{{ t('ui.multipleDataSetsHaveBeenAddedAndTheLayoutOf') }}</span>
            </div>
            <div class="join-list">
              <div v-for="join in datasetJoins" :key="join.id" class="join-row">
                <div class="join-row__body">
                  <span>{{ joinEndpoint(join.left_dataset_id, join.left_field) }}</span>
                  <q-badge outline color="primary">{{ joinTypeLabel(join) }}</q-badge>
                  <span>{{ joinEndpoint(join.right_dataset_id, join.right_field) }}</span>
                </div>
                <div class="row-actions">
                  <q-btn
                    flat
                    dense
                    round
                    color="primary"
                    icon="edit"
                    @click="$emit('editJoin', join.id)"
                  >
                    <q-tooltip>{{ t('ui.edit') }}</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    dense
                    round
                    color="negative"
                    icon="delete"
                    @click="$emit('removeJoin', join.id)"
                  >
                    <q-tooltip>{{ t('ui.delete') }}</q-tooltip>
                  </q-btn>
                </div>
              </div>
              <div v-if="!datasetJoins.length" class="empty-note compact">
                {{ t('ui.thereIsNoCorrelationAndMultipleDataSetsCannotBe') }}
              </div>
            </div>
          </section>
        </div>
        <div v-else class="empty-note">{{ t('ui.selectALeftDataSet') }}</div>
      </q-tab-panel>

      <q-tab-panel name="report" class="inspector-pane">
        <div class="context-head">
          <div>
            <span>{{ t('ui.report') }}</span>
            <strong>{{ t('ui.reportingCapacity') }}</strong>
          </div>
          <q-icon name="settings" color="primary" />
        </div>
        <div class="panel-stack">
          <section class="setting-section report-settings">
            <div class="two-column-grid">
              <q-input
                :model-value="category"
                dense
                outlined
                :label="t('ui.category')"
                @update:model-value="$emit('update:category', String($event || ''))"
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
                <q-tooltip>{{ t('ui.theLineIsThenCarriedOutByLineByLine') }}</q-tooltip>
              </q-select>
            </div>
            <q-input
              :model-value="description"
              dense
              outlined
              type="textarea"
              :rows="2"
              :label="t('ui.descriptionLabel')"
              @update:model-value="$emit('update:description', String($event || ''))"
            />
            <div class="definition-row">
              <q-icon name="table_chart" />
              <div>
                <span>{{ t('ui.runMasterTable') }}</span>
                <strong>{{ primaryDataset?.source_code || '-' }}</strong>
              </div>
            </div>
          </section>
          <section class="setting-section">
            <div class="section-head single-line">
              <div><strong>{{ t('ui.runTimeConfiguration') }}</strong></div>
            </div>
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
          </section>
        </div>
      </q-tab-panel>
    </q-tab-panels>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
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
  formulaError: string
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
  editJoin: [id: string]
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
    value === 'avg' ||
    value === 'max' ||
    value === 'min' ||
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

function joinEndpoint(datasetId: string, fieldCode: string) {
  return `${datasetName(datasetId)}.${fieldName(datasetId, fieldCode)}`
}

function joinTypeLabel(join: ReportDatasetJoin) {
  return join.join_type === 'inner' ? t('ui.inline') : t('ui.leftAssociation')
}

const bindingTypeLabel = computed(
  () => props.bindingTypeOptions.find((option) => option.value === props.bindingType)?.label || '',
)
</script>

<style scoped lang="scss">
.inspector-panel {
  min-height: 0;
  display: grid;
  grid-template-rows: 42px 1px minmax(0, 1fr);
  overflow: hidden;
  background: #fbfcff;
}

.inspector-nav {
  min-height: 42px;
  background: #fff;
}

.inspector-nav :deep(.q-tab) {
  min-height: 42px;
  padding: 0 8px;
}

.inspector-nav :deep(.q-tab__content) {
  min-width: 0;
  gap: 5px;
}

.inspector-nav :deep(.q-icon) {
  font-size: 18px;
}

.inspector-nav :deep(.q-tab__label) {
  font-size: 12px;
  font-weight: 700;
}

.inspector-tabs {
  min-height: 0;
  overflow: auto;
  background: transparent;
}

.inspector-tabs :deep(.q-panel),
.inspector-tabs :deep(.q-tab-panel) {
  height: auto !important;
}

.inspector-pane {
  padding: 0 !important;
}

.context-head {
  min-height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid #e7ecf6;
  background: #fff;
}

.context-head > div,
.definition-row > div {
  min-width: 0;
}

.context-head span,
.context-head strong,
.definition-row span,
.definition-row strong {
  display: block;
}

.context-head span,
.definition-row span {
  color: #71809a;
  font-size: 11px;
}

.context-head strong {
  margin-top: 2px;
  font-size: 14px;
  font-weight: 800;
}

.context-head > .q-icon {
  font-size: 20px;
}

.panel-stack {
  display: grid;
  background: #fff;
}

.setting-section {
  min-width: 0;
  display: grid;
  gap: 9px;
  padding: 12px;
  border-bottom: 1px solid #e7ecf6;
}

.two-column-grid,
.runtime-grid,
.style-row {
  min-width: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: center;
  gap: 8px;
}

.runtime-grid {
  grid-template-columns: minmax(0, 1fr) 104px;
}

.empty-note {
  padding: 24px 14px;
  color: #71809a;
  font-size: 12px;
  text-align: center;
}

.empty-note.compact {
  padding: 10px 4px;
}

.binding-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px dashed #cfd6e6;
  border-radius: 6px;
  color: #5f6f88;
  background: #fff;
  font-size: 12px;
}

.definition-row {
  min-height: 44px;
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 8px 10px;
  border: 1px dashed #cfd6e6;
  border-radius: 6px;
  background: #fbfcff;
}

.definition-row > .q-icon {
  flex: 0 0 auto;
  color: var(--q-primary);
  font-size: 18px;
}

.definition-row strong {
  min-width: 0;
  margin-top: 1px;
  overflow: hidden;
  color: #263248;
  font-size: 12px;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.definition-row .q-btn {
  flex: 0 0 auto;
  margin-left: auto;
}

.section-head {
  min-height: 30px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.section-head > div {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 7px;
}

.section-head strong {
  font-size: 13px;
  font-weight: 800;
}

.section-head span {
  min-width: 22px;
  padding: 1px 6px;
  border-radius: 10px;
  color: var(--q-primary);
  background: rgba(115, 103, 240, 0.1);
  font-size: 11px;
  text-align: center;
}

.section-head.single-line {
  min-height: 22px;
}

.join-list {
  display: grid;
  gap: 6px;
}

.join-warning {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  padding: 8px 9px;
  color: #806000;
  background: #fff8e1;
  border: 1px solid #ffe6a3;
  border-radius: 6px;
  font-size: 11px;
  line-height: 1.5;
}

.join-row {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 8px 6px 8px 10px;
  border: 1px solid #e1e7f1;
  border-radius: 6px;
  background: #fff;
  color: #172033;
}

.join-row__body {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.join-row__body > span {
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.join-row__body .q-badge {
  width: fit-content;
  font-size: 10px;
}

.row-actions {
  display: flex;
  align-items: center;
}

.row-actions .q-btn {
  font-size: 11px;
}

.report-settings :deep(textarea) {
  resize: none;
}

.setting-section :deep(.q-field--dense .q-field__control),
.setting-section :deep(.q-field--dense .q-field__marginal) {
  min-height: 38px;
  height: 38px;
}

.setting-section :deep(.q-field--auto-height.q-field--dense .q-field__control),
.setting-section :deep(.q-field--auto-height.q-field--dense .q-field__native) {
  min-height: 38px;
  height: auto;
}

.setting-section :deep(.q-field__label) {
  font-size: 12px;
}
</style>
