<template>
  <div class="report-designer-fullscreen">
    <report-designer-topbar
      :report-name="form.report_name"
      :report-code="form.report_code"
      :report-kind="form.report_kind"
      :primary-source-code="primaryDataset?.source_code"
      :saving="saving"
      :kind-options="reportKindOptions"
      @update:report-name="form.report_name = $event"
      @update:report-code="form.report_code = $event"
      @update:report-kind="applyReportKind"
      @back="goBack"
      @add-parameter="addParameter"
      @preview="preview"
      @validate="validateAndNotify"
      @save-draft="saveReport('draft')"
      @publish="publishReport"
    />

    <main class="designer-workbench">
      <report-resource-panel
        :datasets="datasets"
        :parameters="parameters"
        :selected-dataset-id="selectedDatasetId"
        :selected-parameter-id="selectedParameterId"
        @open-dataset="openDatasetDialog"
        @select-dataset="selectedDatasetId = $event"
        @edit-dataset="openDatasetDialog"
        @remove-dataset="removeDataset"
        @start-drag-field="startDragField"
        @end-drag-field="draggingField = null"
        @bind-field="bindFieldToActiveCell"
        @add-parameter="addParameter"
        @select-parameter="selectedParameterId = $event"
        @edit-parameter="openParameterDialog"
        @remove-parameter="removeParameter"
      />

      <report-sheet-canvas
        :sheet="sheet"
        :selected-cell-id="selectedCellId"
        :selection-range="selectionRange"
        :active-bold="!!activeCell?.style?.bold"
        :scale="sheet.scale || 0.85"
        :datasets="datasets"
        :dataset-joins="datasetJoins"
        :field-dragging="!!draggingField"
        @toggle-bold="toggleBold"
        @set-align="setAlign"
        @merge-right="mergeRight"
        @merge-selection="mergeSelection"
        @unmerge-active-cell="unmergeActiveCell"
        @clear-active-cell="clearActiveCell"
        @clear-selection="clearSelection"
        @add-row="addRow"
        @add-col="addCol"
        @select-cell="selectCell"
        @select-range="selectRange"
        @drop-field="dropField"
        @update-cell-value="updateCellValue"
        @clear-cell="clearCellAt"
        @merge-cell-right="mergeCellRightAt"
        @unmerge-cell="unmergeCellAt"
        @insert-row-after="insertRowAfter"
        @insert-col-after="insertColAfter"
        @toggle-summary-row="toggleSummaryRow"
        @toggle-detail-row="toggleDetailRow"
        @zoom-in="zoomIn"
        @zoom-out="zoomOut"
      />

      <report-inspector-panel
        v-model:tab="inspectorTab"
        v-model:cell-value="activeCellValue"
        v-model:binding-type="activeBindingType"
        v-model:binding-dataset-id="activeBindingDatasetId"
        v-model:binding-field="activeBindingField"
        v-model:formula="activeFormula"
        v-model:cell-bold="activeCellBold"
        v-model:cell-align="activeCellAlign"
        :active-cell-label="activeCellLabel"
        :has-active-cell="!!activeCell"
        :binding-preview="activeBindingPreview"
        :binding-type-options="reportBindingTypeOptions"
        :dataset-options="datasetOptions"
        :active-dataset-field-options="activeDatasetFieldOptions"
        :align-options="reportAlignOptions"
        :selected-dataset="selectedDataset"
        :primary-dataset="primaryDataset"
        :datasets="datasets"
        :dataset-joins="datasetJoins"
        :category="form.category || ''"
        :description="form.description || ''"
        :runtime-display="form.runtime_display || 'paged'"
        :runtime-page-size="form.runtime_page_size || 20"
        :runtime-display-options="reportRuntimeDisplayOptions"
        @update-dataset-name="updateDatasetName"
        @set-primary-dataset="setPrimaryDataset"
        @add-join="openJoinDialog"
        @remove-join="removeJoin"
        @update:category="form.category = $event"
        @update:description="form.description = $event"
        @update:runtime-display="form.runtime_display = $event"
        @update:runtime-page-size="form.runtime_page_size = $event"
      />
    </main>

    <footer class="designer-statusbar">
      <span>当前单元格：{{ activeCellLabel }}</span>
      <span>数据集：{{ datasets.length }}</span>
      <span>已绑定字段：{{ usedFields.length }}</span>
      <span>{{ reportId ? `报表ID ${reportId}` : '新建报表' }}</span>
    </footer>

    <report-dataset-dialog
      v-model="datasetDialogVisible"
      :editing-dataset="editingDataset"
      :draft="datasetDraft"
      :dataset-type-options="reportDatasetTypeOptions"
      :data-sources="dataSources"
      :preview-fields="datasetDraftPreviewFields"
      :sql-fields-loading="sqlFieldsLoading"
      @update:type="handleDraftTypeChange"
      @update:name="datasetDraft.name = $event"
      @update:source-code="handleDraftSourceChange"
      @update:sql="handleDatasetDraftSqlChange"
      @update:fields-text="datasetDraft.fieldsText = $event"
      @infer-sql-fields="inferSqlDatasetFields"
      @confirm="confirmDataset"
    />

    <report-parameter-dialog
      v-model="parameterDialogVisible"
      :editing="!!editingParameterId"
      :draft="parameterDraft"
      :dataset-options="datasetOptions"
      :field-options="parameterFieldOptions"
      :type-options="reportParameterTypeOptions"
      :operator-options="reportParameterOperatorOptions"
      @update:label="parameterDraft.label = $event"
      @update:dataset-id="handleParameterDatasetChange"
      @update:field="handleParameterFieldChange"
      @update:type="parameterDraft.type = $event"
      @update:operator="parameterDraft.operator = $event"
      @update:placeholder="parameterDraft.placeholder = $event"
      @update:default-value="parameterDraft.default_value = $event"
      @confirm="confirmParameter"
    />

    <report-join-dialog
      v-model="joinDialogVisible"
      :draft="joinDraft"
      :dataset-options="datasetOptions"
      :left-field-options="joinLeftFieldOptions"
      :right-field-options="joinRightFieldOptions"
      :join-type-options="reportDatasetJoinTypeOptions"
      @update:left-dataset-id="handleJoinLeftDatasetChange"
      @update:left-field="joinDraft.left_field = $event"
      @update:right-dataset-id="handleJoinRightDatasetChange"
      @update:right-field="joinDraft.right_field = $event"
      @update:join-type="joinDraft.join_type = $event"
      @confirm="confirmJoin"
    />

    <q-dialog v-model="previewDialogVisible">
      <q-card class="preview-dialog">
        <q-card-section class="dialog-head">
          <div>
            <div class="dialog-title">运行预览</div>
            <div class="dialog-caption">
              {{ form.id ? '真实数据预览，已经过后端数据权限' : '未保存报表的本地结构预览' }}
            </div>
          </div>
          <q-btn flat round dense icon="close" v-close-popup />
        </q-card-section>
        <q-card-section>
          <div class="preview-meta">
            <q-chip dense square color="primary" text-color="white">
              {{ previewData.total }} 行
            </q-chip>
            <q-chip
              v-if="previewData.meta?.dataset_id"
              dense
              square
              outline
              color="primary"
            >
              主数据集 {{ previewData.meta.dataset_id }}
            </q-chip>
            <q-chip
              v-for="join in previewData.joins || []"
              :key="join.id"
              dense
              square
              outline
              color="primary"
              icon="account_tree"
            >
              {{ joinLabel(join) }}
            </q-chip>
          </div>
          <report-sheet-preview
            :sheet="sheet"
            :datasets="datasets"
            :preview-data="previewData"
            :loading="previewLoading"
            :report-kind="form.report_kind"
          />
        </q-card-section>
      </q-card>
    </q-dialog>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_design' })

import { computed, onMounted, reactive, ref } from 'vue'
import { useQuasar } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import {
  defaultReportSheet,
  useReportApi,
  type Report,
  type ReportCellBindingType,
  type ReportDataSource,
  type ReportDatasetJoin,
  type ReportDatasetJoinType,
  type ReportDataset,
  type ReportDatasetType,
  type ReportField,
  type ReportKind,
  type ReportParameter,
  type ReportParameterOperator,
  type ReportParameterType,
  type ReportPreviewRes,
  type ReportSaveReq,
  type ReportSheetCell,
  type ReportSheetConfig,
} from 'src/api/services/report'
import ReportDesignerTopbar from './components/ReportDesignerTopbar.vue'
import ReportDatasetDialog from './components/ReportDatasetDialog.vue'
import ReportInspectorPanel from './components/ReportInspectorPanel.vue'
import ReportJoinDialog from './components/ReportJoinDialog.vue'
import ReportParameterDialog from './components/ReportParameterDialog.vue'
import ReportResourcePanel from './components/ReportResourcePanel.vue'
import ReportSheetCanvas from './components/ReportSheetCanvas.vue'
import ReportSheetPreview from '../components/ReportSheetPreview.vue'
import {
  reportAlignOptions,
  reportDatasetJoinTypeOptions,
  reportBindingTypeOptions,
  reportDatasetTypeOptions,
  reportKindOptions,
  reportParameterOperatorOptions,
  reportParameterTypeOptions,
  reportRuntimeDisplayOptions,
} from 'src/modules/report/options'
import {
  buildReportLocalPreview,
  collectReportUsedFields,
  defaultReportBindingType,
  reportBindingText,
  reportCellId,
  reportColumnName,
  reportNormalizeSheetRange,
  reportSheetCellAt,
  reportSheetCellSpan,
  type ReportSheetRange,
} from 'src/modules/report/sheet'
import { hasReportCellConfig, normalizeReportSheet, reportParameterDefaultsForField } from 'src/modules/report/schema'

const $q = useQuasar()
const route = useRoute()
const router = useRouter()
const reportApi = useReportApi()

const reportId = computed(() => Number(route.query.id || 0))
const saving = ref(false)
const previewLoading = ref(false)
const dataSources = ref<ReportDataSource[]>([])
const datasets = ref<ReportDataset[]>([])
const datasetJoins = ref<ReportDatasetJoin[]>([])
const parameters = ref<ReportParameter[]>([])
const sheet = ref<ReportSheetConfig>(defaultReportSheet())
const selectedDatasetId = ref('')
const selectedCellId = ref('2:2')
const selectionRange = ref<ReportSheetRange | null>(null)
const selectedParameterId = ref('')
const inspectorTab = ref<'cell' | 'data' | 'report'>('cell')
const draggingField = ref<{ datasetId: string; fieldCode: string } | null>(null)
const datasetDialogVisible = ref(false)
const editingDatasetId = ref('')
const sqlFieldsLoading = ref(false)
const parameterDialogVisible = ref(false)
const editingParameterId = ref('')
const joinDialogVisible = ref(false)
const previewDialogVisible = ref(false)
const previewData = ref<ReportPreviewRes>({ columns: [], rows: [], total: 0 })

const form = reactive<ReportSaveReq>({
  report_name: '',
  report_code: '',
  report_kind: 'detail',
  category: '',
  description: '',
  data_source_id: undefined,
  permission_menu_id: 0,
  permission_table_code: '',
  fields: [],
  datasets: [],
  dataset_joins: [],
  parameters: [],
  sheet: defaultReportSheet(),
  runtime_display: 'paged',
  runtime_page_size: 20,
  status: 'draft',
})

const datasetDraft = reactive<{
  type: ReportDatasetType
  name: string
  source_code: string
  sql: string
  fieldsText: string
}>({
  type: 'table',
  name: '',
  source_code: '',
  sql: '',
  fieldsText: '',
})

const parameterDraft = reactive<{
  id: string
  label: string
  dataset_id: string
  field: string
  type: ReportParameterType
  operator: ReportParameterOperator
  placeholder: string
  default_value: string
}>({
  id: '',
  label: '',
  dataset_id: '',
  field: '',
  type: 'text',
  operator: 'like',
  placeholder: '',
  default_value: '',
})

const joinDraft = reactive<{
  left_dataset_id: string
  left_field: string
  right_dataset_id: string
  right_field: string
  join_type: ReportDatasetJoinType
}>({
  left_dataset_id: '',
  left_field: '',
  right_dataset_id: '',
  right_field: '',
  join_type: 'left',
})

const sqlDraftFields = ref<ReportField[]>([])
const datasetOptions = computed(() =>
  datasets.value.map((item) => ({
    label: `${item.name} (${item.type === 'sql' ? 'SQL' : item.source_code})`,
    value: item.id,
  })),
)
const primaryDataset = computed(() => datasets.value.find((item) => item.primary) || datasets.value[0])
const selectedDataset = computed(() => datasets.value.find((item) => item.id === selectedDatasetId.value))
const editingDataset = computed(() => datasets.value.find((item) => item.id === editingDatasetId.value))
const activeCell = computed(() => {
  const parts = selectedCellId.value.split(':')
  const row = Number(parts[0])
  const col = Number(parts[1])
  if (!Number.isInteger(row) || !Number.isInteger(col)) return undefined
  return reportSheetCellAt(sheet.value, row, col)
})
const activeCellLabel = computed(() => {
  const cell = activeCell.value
  if (!cell) return '-'
  return `${reportColumnName(cell.col)}${cell.row}`
})
const activeBindingDataset = computed(() =>
  datasets.value.find((item) => item.id === activeCell.value?.binding?.dataset_id),
)
const activeDatasetFieldOptions = computed(() =>
  (activeBindingDataset.value?.fields || []).map((field) => ({
    label: `${field.name} (${field.code})`,
    value: field.code,
  })),
)
const activeBindingPreview = computed(() => {
  const binding = activeCell.value?.binding
  if (!binding?.dataset_id || !binding.field) return ''
  const dataset = datasets.value.find((item) => item.id === binding.dataset_id)
  const field = dataset?.fields.find((item) => item.code === binding.field)
  if (!dataset || !field) return ''
  return reportBindingText(binding.type, dataset, field)
})
const usedFields = computed(() => collectReportUsedFields(datasets.value, sheet.value))
const datasetDraftPreviewFields = computed(() => {
  if (datasetDraft.type === 'table') {
    return dataSources.value.find((item) => item.code === datasetDraft.source_code)?.fields || []
  }
  return sqlDraftFields.value.length ? sqlDraftFields.value : parseSqlDatasetFields(datasetDraft.fieldsText)
})
const parameterFieldOptions = computed(() => {
  const dataset = datasets.value.find((item) => item.id === parameterDraft.dataset_id)
  return (dataset?.fields || []).map((field) => ({
    label: `${field.name} (${field.code})`,
    value: field.code,
  }))
})
const joinLeftFieldOptions = computed(() => datasetFieldOptions(joinDraft.left_dataset_id))
const joinRightFieldOptions = computed(() => datasetFieldOptions(joinDraft.right_dataset_id))

const activeCellValue = computed({
  get: () => activeCell.value?.value || '',
  set: (value: string) => patchActiveCell({ value }),
})
const activeBindingType = computed<ReportCellBindingType>({
  get: () => activeCell.value?.binding?.type || 'static',
  set: (value) => {
    patchBinding({ type: value })
    refreshActiveCellBindingText()
  },
})
const activeBindingDatasetId = computed({
  get: () => activeCell.value?.binding?.dataset_id || primaryDataset.value?.id || '',
  set: (value: string) => {
    const dataset = datasets.value.find((item) => item.id === value)
    const currentField = activeCell.value?.binding?.field
    const field = dataset?.fields.find((item) => item.code === currentField) || dataset?.fields[0]
    patchBinding({ dataset_id: value, field: field?.code || '' })
    refreshActiveCellBindingText()
  },
})
const activeBindingField = computed({
  get: () => activeCell.value?.binding?.field || '',
  set: (value: string) => {
    const dataset = datasets.value.find((item) => item.id === activeBindingDatasetId.value)
    const field = dataset?.fields.find((item) => item.code === value)
    patchBinding({ field: value })
    if (field) patchActiveCell({ value: reportBindingText(activeBindingType.value, dataset!, field) })
    buildLocalPreview()
  },
})
const activeFormula = computed({
  get: () => activeCell.value?.binding?.formula || '',
  set: (value: string) => patchBinding({ formula: value }),
})
const activeCellBold = computed({
  get: () => !!activeCell.value?.style?.bold,
  set: (value: boolean) => patchCellStyle({ bold: value }),
})
const activeCellAlign = computed<'left' | 'center' | 'right'>({
  get: () => activeCell.value?.style?.align || 'left',
  set: (value) => patchCellStyle({ align: value }),
})

onMounted(async () => {
  await loadDataSources()
  await loadReport()
  buildLocalPreview()
})

async function loadDataSources() {
  try {
    const res = await reportApi.queryDataSources()
    dataSources.value = res.data || []
  } catch {
    dataSources.value = []
    $q.notify({ type: 'negative', message: '数据源加载失败' })
  }
}

async function loadReport() {
  if (!reportId.value) {
    initNewReport()
    return
  }
  try {
    const res = await reportApi.queryReportById(reportId.value)
    applyReport(res.data)
  } catch {
    $q.notify({ type: 'negative', message: '报表加载失败' })
    goBack()
  }
}

function applyReport(report: Report) {
  form.id = report.id
  form.report_name = report.report_name
  form.report_code = report.report_code
  form.report_kind = report.report_kind
  form.category = report.category || ''
  form.description = report.description || ''
  form.permission_menu_id = report.permission_menu_id || 0
  form.permission_table_code = report.permission_table_code || report.source_code || ''
  form.runtime_display = report.layout_config?.runtime_display || 'paged'
  form.runtime_page_size = Number(report.layout_config?.runtime_page_size || 20)
  datasets.value = enrichReportDatasets(
    report.layout_config?.datasets?.length
      ? report.layout_config.datasets
      : createInitialDatasets(report.source_code),
  )
  datasetJoins.value = report.layout_config?.dataset_joins || report.query_config?.dataset_joins || []
  sheet.value = normalizeReportSheet(report.layout_config?.sheet || defaultReportSheet())
  parameters.value = report.layout_config?.parameters || report.query_config?.parameters || []
  selectedDatasetId.value = datasets.value[0]?.id || ''
  selectedCellId.value = sheet.value.active_cell || '1:1'
  syncForm()
}

function initNewReport() {
  form.report_name = ''
  form.report_code = `report_${Date.now().toString().slice(-6)}`
  form.report_kind = 'detail'
  form.category = ''
  form.description = ''
  form.permission_menu_id = 0
  form.permission_table_code = ''
  form.runtime_display = 'paged'
  form.runtime_page_size = 20
  datasets.value = []
  datasetJoins.value = []
  sheet.value = defaultReportSheet()
  parameters.value = []
  selectedDatasetId.value = ''
  selectedCellId.value = '1:1'
  syncForm()
}

function createInitialDatasets(sourceCode?: string): ReportDataset[] {
  const source = dataSources.value.find((item) => item.code === sourceCode) || dataSources.value[0]
  if (!source) return []
  return [
    {
      id: 'main',
      name: source.name || '主数据',
      type: 'table',
      source_code: source.code,
      fields: source.fields,
      primary: true,
    },
  ]
}

function enrichReportDatasets(sourceDatasets: ReportDataset[]): ReportDataset[] {
  return ensureDesignerDatasets(sourceDatasets)
}

function ensureDesignerDatasets(sourceDatasets: ReportDataset[]): ReportDataset[] {
  return sourceDatasets.map((dataset) => {
    if (dataset.type !== 'table' || !dataset.source_code) {
      return {
        ...dataset,
        fields: (dataset.fields || []).map((field) => ({ ...field })),
      }
    }
    const source = dataSources.value.find((item) => item.code === dataset.source_code)
    if (!source) {
      return {
        ...dataset,
        fields: (dataset.fields || []).map((field) => ({ ...field })),
      }
    }
    const existingByCode = new Map((dataset.fields || []).map((field) => [field.code, field]))
    const sourceCodes = new Set(source.fields.map((field) => field.code))
    return {
      ...dataset,
      name: dataset.name || source.name,
      fields: [
        ...source.fields.map((field) => ({ ...(existingByCode.get(field.code) || field) })),
        ...(dataset.fields || [])
          .filter((field) => !sourceCodes.has(field.code))
          .map((field) => ({ ...field })),
      ],
    }
  })
}

function openDatasetDialog(id = '') {
  editingDatasetId.value = id
  const current = datasets.value.find((item) => item.id === id)
  if (current) {
    datasetDraft.type = current.type
    datasetDraft.source_code = current.source_code || ''
    datasetDraft.name = current.name || ''
    datasetDraft.sql = current.sql || ''
    datasetDraft.fieldsText = current.fields.map((field) => field.code).join(',')
    sqlDraftFields.value = current.type === 'sql' ? [...current.fields] : []
    datasetDialogVisible.value = true
    return
  }
  datasetDraft.type = 'table'
  datasetDraft.source_code = dataSources.value[0]?.code || ''
  datasetDraft.name = dataSources.value[0]?.name || ''
  datasetDraft.sql = ''
  datasetDraft.fieldsText = ''
  sqlDraftFields.value = []
  datasetDialogVisible.value = true
}

function handleDraftTypeChange(value: ReportDatasetType) {
  datasetDraft.type = value
  if (datasetDraft.type === 'table') {
    sqlDraftFields.value = []
    handleDraftSourceChange(datasetDraft.source_code || dataSources.value[0]?.code || '')
  } else {
    datasetDraft.name = 'SQL 数据集'
    datasetDraft.source_code = ''
    sqlDraftFields.value = parseSqlDatasetFields(datasetDraft.fieldsText)
  }
}

function handleDraftSourceChange(value: string) {
  const source = dataSources.value.find((item) => item.code === value)
  datasetDraft.source_code = value
  datasetDraft.name = source?.name || value
}

function handleDatasetDraftSqlChange(value: string) {
  datasetDraft.sql = value
  sqlDraftFields.value = []
  datasetDraft.fieldsText = ''
}

async function inferSqlDatasetFields() {
  const sql = datasetDraft.sql.trim()
  if (!sql) {
    $q.notify({ type: 'warning', message: '请先填写 SQL' })
    return false
  }
  sqlFieldsLoading.value = true
  try {
    const res = await reportApi.inferSqlFields(sql)
    sqlDraftFields.value = res.data || []
    datasetDraft.fieldsText = sqlDraftFields.value.map((field) => field.code).join(',')
    if (!sqlDraftFields.value.length) {
      $q.notify({ type: 'warning', message: 'SQL 未解析出字段' })
      return false
    }
    $q.notify({ type: 'positive', message: `已解析 ${sqlDraftFields.value.length} 个字段` })
    return true
  } catch (error) {
    sqlDraftFields.value = []
    $q.notify({ type: 'negative', message: sqlFieldInferErrorMessage(error) })
    return false
  } finally {
    sqlFieldsLoading.value = false
  }
}

async function confirmDataset() {
  const id = editingDatasetId.value || `dataset_${Date.now()}`
  if (datasetDraft.type === 'table') {
    const source = dataSources.value.find((item) => item.code === datasetDraft.source_code)
    if (!source) {
      $q.notify({ type: 'warning', message: '请选择来源表' })
      return
    }
    upsertDataset({
      id,
      name: datasetDraft.name || source.name,
      type: 'table',
      source_code: source.code,
      fields: source.fields,
      primary: editingDataset.value?.primary || datasets.value.length === 0,
    })
  } else {
    if (!datasetDraft.sql.trim()) {
      $q.notify({ type: 'warning', message: '请填写 SQL' })
      return
    }
    let fields = datasetDraftPreviewFields.value
    if (!datasetDraft.sql.trim() || fields.length === 0) {
      const ok = await inferSqlDatasetFields()
      fields = datasetDraftPreviewFields.value
      if (!ok || fields.length === 0) {
        $q.notify({ type: 'warning', message: '请先解析 SQL 字段' })
        return
      }
    }
    upsertDataset({
      id,
      name: datasetDraft.name || 'SQL 数据集',
      type: 'sql',
      sql: datasetDraft.sql,
      fields,
      primary: editingDataset.value?.primary || false,
    })
  }
  selectedDatasetId.value = id
  editingDatasetId.value = ''
  datasetDialogVisible.value = false
  buildLocalPreview()
}

function parseSqlDatasetFields(fieldsText: string): ReportField[] {
  return fieldsText
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .map((code) => ({ code, name: code, type: 'string', role: 'text' as const }))
}

function upsertDataset(dataset: ReportDataset) {
  const normalizedDataset = ensureDesignerDatasets([dataset])[0] || dataset
  const index = datasets.value.findIndex((item) => item.id === dataset.id)
  if (index === -1) datasets.value.push(normalizedDataset)
  else datasets.value[index] = normalizedDataset
}

function removeDataset(id: string) {
  const removed = datasets.value.find((item) => item.id === id)
  datasets.value = datasets.value.filter((item) => item.id !== id)
  datasetJoins.value = datasetJoins.value.filter(
    (join) => join.left_dataset_id !== id && join.right_dataset_id !== id,
  )
  parameters.value = parameters.value.filter((param) => param.dataset_id !== id)
  sheet.value.cells = sheet.value.cells.map((cell) => {
    if (cell.binding?.dataset_id !== id) return cell
    const style = { ...(cell.style || {}) }
    delete style.color
    return { ...cell, value: '', binding: undefined, style }
  }).filter(hasReportCellConfig)
  if (removed?.primary && datasets.value[0]) datasets.value[0].primary = true
  selectedDatasetId.value = datasets.value[0]?.id || ''
  buildLocalPreview()
}

function setPrimaryDataset(id: string) {
  const dataset = datasets.value.find((item) => item.id === id)
  if (!dataset || dataset.type !== 'table') {
    $q.notify({ type: 'warning', message: '只有现有表数据集可以作为第一版运行主表' })
    return
  }
  datasets.value.forEach((item) => {
    item.primary = item.id === id
  })
  form.permission_table_code = dataset.source_code || ''
}

function updateDatasetName(id: string, name: string) {
  const dataset = datasets.value.find((item) => item.id === id)
  if (dataset) dataset.name = name
}

function startDragField(dataset: ReportDataset, field: ReportField) {
  draggingField.value = { datasetId: dataset.id, fieldCode: field.code }
}

function dropField(row: number, col: number) {
  const dragged = draggingField.value
  if (!dragged) return
  const dataset = datasets.value.find((item) => item.id === dragged.datasetId)
  const field = dataset?.fields.find((item) => item.code === dragged.fieldCode)
  if (dataset && field) bindCell(row, col, dataset, field, defaultReportBindingType(field))
  draggingField.value = null
}

function bindFieldToActiveCell(dataset: ReportDataset, field: ReportField) {
  const cell = activeCell.value
  if (!cell) return
  bindCell(cell.row, cell.col, dataset, field, defaultReportBindingType(field))
}

function bindCell(
  row: number,
  col: number,
  dataset: ReportDataset,
  field: ReportField,
  type: ReportCellBindingType,
) {
  const value = reportBindingText(type, dataset, field)
  patchCell(row, col, {
    value,
    binding: {
      type,
      dataset_id: dataset.id,
      field: field.code,
    },
    style: { ...cellAt(row, col).style, color: '#6d5dfc', bold: true },
  })
  markDetailRow(row)
  selectedCellId.value = reportCellId(row, col)
  inspectorTab.value = 'cell'
  buildLocalPreview()
}

function cellAt(row: number, col: number) {
  return reportSheetCellAt(sheet.value, row, col)
}

function patchCell(row: number, col: number, patch: Partial<ReportSheetCell>) {
  const index = sheet.value.cells.findIndex((item) => item.row === row && item.col === col)
  const current = index === -1 ? { id: reportCellId(row, col), row, col, value: '' } : sheet.value.cells[index]!
  const next = { ...current, ...patch }
  if (!hasReportCellConfig(next)) {
    if (index !== -1) sheet.value.cells.splice(index, 1)
    return
  }
  if (index === -1) sheet.value.cells.push(next)
  else sheet.value.cells[index] = next
  sheet.value.cells.sort((a, b) => a.row - b.row || a.col - b.col)
}

function patchActiveCell(patch: Partial<ReportSheetCell>) {
  const cell = activeCell.value
  if (!cell) return
  patchCell(cell.row, cell.col, patch)
}

function updateCellValue(row: number, col: number, value: string) {
  patchCell(row, col, { value })
  buildLocalPreview()
}

function patchBinding(patch: Partial<NonNullable<ReportSheetCell['binding']>>) {
  const cell = activeCell.value
  if (!cell) return
  patchActiveCell({
    binding: {
      type: cell.binding?.type || 'static',
      ...(cell.binding || {}),
      ...patch,
    },
  })
}

function patchCellStyle(patch: Partial<NonNullable<ReportSheetCell['style']>>) {
  const cell = activeCell.value
  if (!cell) return
  patchActiveCell({ style: { ...(cell.style || {}), ...patch } })
}

function refreshActiveCellBindingText() {
  const cell = activeCell.value
  const binding = cell?.binding
  if (!cell || !binding?.dataset_id || !binding.field) return
  const dataset = datasets.value.find((item) => item.id === binding.dataset_id)
  const field = dataset?.fields.find((item) => item.code === binding.field)
  if (!dataset || !field) return
  const nextValue = reportBindingText(binding.type, dataset, field)
  if (!cell.value || isGeneratedBindingValue(cell.value)) {
    patchActiveCell({ value: nextValue })
  }
  buildLocalPreview()
}

function isGeneratedBindingValue(value: string) {
  return datasets.value.some((dataset) =>
    dataset.fields.some((field) =>
      ['field', 'group', 'sum', 'count', 'formula'].some((type) =>
        reportBindingText(type as ReportCellBindingType, dataset, field) === value,
      ),
    ),
  )
}

function selectCell(row: number, col: number) {
  selectedCellId.value = reportCellId(row, col)
  selectionRange.value = { startRow: row, startCol: col, endRow: row, endCol: col }
  sheet.value.active_cell = selectedCellId.value
  selectedParameterId.value = ''
}

function selectRange(row: number, col: number) {
  const active = activeCell.value || cellAt(row, col)
  selectionRange.value = {
    startRow: active.row,
    startCol: active.col,
    endRow: row,
    endCol: col,
  }
  selectedCellId.value = reportCellId(row, col)
  sheet.value.active_cell = selectedCellId.value
  selectedParameterId.value = ''
}

function clearActiveCell() {
  const cell = activeCell.value
  if (!cell) return
  patchActiveCell({ value: '', binding: undefined, style: {}, rowspan: 1, colspan: 1 })
  buildLocalPreview()
}

function toggleBold() {
  activeCellBold.value = !activeCellBold.value
}

function setAlign(value: 'left' | 'center' | 'right') {
  activeCellAlign.value = value
}

function mergeRight() {
  const cell = activeCell.value
  if (!cell || cell.col >= sheet.value.cols) return
  const nextCol = cell.col + (cell.colspan || 1)
  if (nextCol <= sheet.value.cols && cellHasContent(cellAt(cell.row, nextCol))) {
    $q.notify({ type: 'warning', message: '右侧单元格已有内容，请先选择区域合并或清空后再合并' })
    return
  }
  patchActiveCell({ colspan: Math.min((cell.colspan || 1) + 1, sheet.value.cols - cell.col + 1) })
  buildLocalPreview()
}

function mergeSelection() {
  const range = selectionRange.value
  if (!range) return
  const bounds = reportNormalizeSheetRange(range)
  if (bounds.maxRow === bounds.minRow && bounds.maxCol === bounds.minCol) {
    $q.notify({ type: 'warning', message: '请先按住 Shift 选择要合并的单元格区域' })
    return
  }
  const anchor = cellAt(bounds.minRow, bounds.minCol)
  const blocked = cellsInBounds(bounds).some((cell) => {
    if (cell.row === anchor.row && cell.col === anchor.col) return false
    return cellHasContent(cell)
  })
  if (blocked) {
    $q.notify({ type: 'warning', message: '合并区域内已有内容，请先清理后再合并' })
    return
  }
  cellsInBounds(bounds).forEach((cell) => {
    if (cell.row === anchor.row && cell.col === anchor.col) return
    patchCell(cell.row, cell.col, {
      value: '',
      binding: undefined,
      style: {},
      rowspan: 1,
      colspan: 1,
    })
  })
  patchCell(anchor.row, anchor.col, {
    rowspan: bounds.maxRow - bounds.minRow + 1,
    colspan: bounds.maxCol - bounds.minCol + 1,
  })
  selectedCellId.value = anchor.id
  selectionRange.value = {
    startRow: anchor.row,
    startCol: anchor.col,
    endRow: anchor.row,
    endCol: anchor.col,
  }
  buildLocalPreview()
}

function unmergeActiveCell() {
  const cell = activeCell.value
  if (!cell) return
  unmergeCellAt(cell.row, cell.col)
}

function clearSelection() {
  const range = selectionRange.value
  if (!range) {
    clearActiveCell()
    return
  }
  const bounds = reportNormalizeSheetRange(range)
  cellsInBounds(bounds).forEach((cell) => {
    patchCell(cell.row, cell.col, {
      value: '',
      binding: undefined,
      style: {},
      rowspan: 1,
      colspan: 1,
    })
  })
  buildLocalPreview()
}

function clearCellAt(row: number, col: number) {
  patchCell(row, col, { value: '', binding: undefined, style: {}, rowspan: 1, colspan: 1 })
  buildLocalPreview()
}

function mergeCellRightAt(row: number, col: number) {
  selectCell(row, col)
  mergeRight()
}

function unmergeCellAt(row: number, col: number) {
  const cell = cellAt(row, col)
  const span = reportSheetCellSpan(cell, { maxRow: sheet.value.rows, maxCol: sheet.value.cols })
  if (span.rowspan === 1 && span.colspan === 1) return
  patchCell(row, col, { rowspan: 1, colspan: 1 })
  buildLocalPreview()
}

function cellHasContent(cell: ReportSheetCell) {
  return Boolean(cell.value || cell.binding?.field || cell.binding?.formula)
}

function cellsInBounds(bounds: ReturnType<typeof reportNormalizeSheetRange>) {
  const cells: ReportSheetCell[] = []
  for (let row = bounds.minRow; row <= bounds.maxRow; row += 1) {
    for (let col = bounds.minCol; col <= bounds.maxCol; col += 1) {
      cells.push(cellAt(row, col))
    }
  }
  return cells
}

function addRow() {
  sheet.value.rows += 1
  buildLocalPreview()
}

function insertRowAfter(row: number) {
  sheet.value.cells = sheet.value.cells.map((cell) =>
    cell.row > row
      ? { ...cell, row: cell.row + 1, id: reportCellId(cell.row + 1, cell.col) }
      : cell,
  )
  sheet.value.rows += 1
  sheet.value.summary_rows = (sheet.value.summary_rows || []).map((item) => (item > row ? item + 1 : item))
  sheet.value.detail_rows = (sheet.value.detail_rows || []).map((item) => (item > row ? item + 1 : item))
  buildLocalPreview()
}

function addCol() {
  sheet.value.cols += 1
  buildLocalPreview()
}

function insertColAfter(col: number) {
  sheet.value.cells = sheet.value.cells.map((cell) =>
    cell.col > col
      ? { ...cell, col: cell.col + 1, id: reportCellId(cell.row, cell.col + 1) }
      : cell,
  )
  sheet.value.cols += 1
  buildLocalPreview()
}

function toggleSummaryRow(row: number) {
  const rows = new Set(sheet.value.summary_rows || [])
  if (rows.has(row)) rows.delete(row)
  else rows.add(row)
  sheet.value.summary_rows = [...rows].sort((a, b) => a - b)
  if (rows.has(row)) {
    sheet.value.detail_rows = (sheet.value.detail_rows || []).filter((item) => item !== row)
  }
  buildLocalPreview()
}

function toggleDetailRow(row: number) {
  const rows = new Set(sheet.value.detail_rows || [])
  if (rows.has(row)) rows.delete(row)
  else rows.add(row)
  sheet.value.detail_rows = [...rows].sort((a, b) => a - b)
  if (rows.has(row)) {
    sheet.value.summary_rows = (sheet.value.summary_rows || []).filter((item) => item !== row)
  }
  buildLocalPreview()
}

function markDetailRow(row: number) {
  const rows = new Set(sheet.value.detail_rows || [])
  rows.add(row)
  sheet.value.detail_rows = [...rows].sort((a, b) => a - b)
  sheet.value.summary_rows = (sheet.value.summary_rows || []).filter((item) => item !== row)
}

function zoomIn() {
  sheet.value.scale = Math.min((sheet.value.scale || 0.85) + 0.1, 1.5)
}

function zoomOut() {
  sheet.value.scale = Math.max((sheet.value.scale || 0.85) - 0.1, 0.5)
}

function addParameter() {
  openParameterDialog('')
}

function openParameterDialog(id = '') {
  const dataset = primaryDataset.value
  const field = dataset?.fields[0]
  if (!dataset || !field) {
    $q.notify({ type: 'warning', message: '请先添加表数据集' })
    return
  }
  const current = parameters.value.find((item) => item.id === id)
  editingParameterId.value = id
  parameterDraft.id = current?.id || ''
  parameterDraft.label = current?.label || field.name
  parameterDraft.dataset_id = current?.dataset_id || dataset.id
  parameterDraft.field = current?.field || field.code
  const defaults = reportParameterDefaultsForField(field)
  parameterDraft.type = current?.type || defaults.type
  parameterDraft.operator = current?.operator || defaults.operator
  parameterDraft.placeholder = current?.placeholder || `请输入${field.name}`
  parameterDraft.default_value = stringifyParameterDefault(current?.default_value)
  parameterDialogVisible.value = true
}

function handleParameterDatasetChange(id: string) {
  parameterDraft.dataset_id = id
  const field = datasets.value.find((item) => item.id === id)?.fields[0]
  parameterDraft.field = field?.code || ''
  parameterDraft.default_value = ''
  if (field) {
    const defaults = reportParameterDefaultsForField(field)
    parameterDraft.label = field.name
    parameterDraft.placeholder = `请输入${field.name}`
    parameterDraft.type = defaults.type
    parameterDraft.operator = defaults.operator
  }
}

function handleParameterFieldChange(fieldCode: string) {
  parameterDraft.field = fieldCode
  const dataset = datasets.value.find((item) => item.id === parameterDraft.dataset_id)
  const field = dataset?.fields.find((item) => item.code === fieldCode)
  if (!field) return
  parameterDraft.label = field.name
  parameterDraft.placeholder = `请输入${field.name}`
  parameterDraft.default_value = ''
  const defaults = reportParameterDefaultsForField(field)
  parameterDraft.type = defaults.type
  parameterDraft.operator = defaults.operator
}

function confirmParameter() {
  if (!parameterDraft.label.trim() || !parameterDraft.dataset_id || !parameterDraft.field) {
    $q.notify({ type: 'warning', message: '请完整配置参数名称、数据集和字段' })
    return
  }
  const id = editingParameterId.value || `param_${Date.now()}`
  const param: ReportParameter = {
    id,
    label: parameterDraft.label,
    dataset_id: parameterDraft.dataset_id,
    field: parameterDraft.field,
    type: parameterDraft.type,
    operator: parameterDraft.operator,
    placeholder: parameterDraft.placeholder,
    default_value: parseParameterDefault(parameterDraft.default_value, parameterDraft.type),
  }
  const index = parameters.value.findIndex((item) => item.id === id)
  if (index === -1) parameters.value.push(param)
  else parameters.value[index] = param
  selectedParameterId.value = param.id
  parameterDialogVisible.value = false
  editingParameterId.value = ''
}

function removeParameter(id: string) {
  parameters.value = parameters.value.filter((item) => item.id !== id)
  if (selectedParameterId.value === id) selectedParameterId.value = ''
}

function stringifyParameterDefault(value: ReportParameter['default_value']) {
  if (Array.isArray(value)) return value.map((item) => String(item ?? '')).filter(Boolean).join(',')
  if (value === null || value === undefined) return ''
  return String(value)
}

function parseParameterDefault(value: string, type: ReportParameterType): ReportParameter['default_value'] {
  const text = value.trim()
  if (!text) return undefined
  if (type === 'date_range') {
    const parts = text.split(',').map((item) => item.trim()).filter(Boolean)
    return parts.length ? parts.slice(0, 2) : undefined
  }
  if (type === 'number') {
    const num = Number(text)
    return Number.isFinite(num) ? num : text
  }
  return text
}

function sqlFieldInferErrorMessage(error: unknown) {
  const status = (error as { response?: { status?: number } })?.response?.status
  if (status === 401 || status === 403) {
    return 'SQL 字段解析接口无权限，请给当前角色分配“SQL字段解析”接口权限'
  }
  return 'SQL 字段解析失败，请检查 SQL 语句或后端接口'
}

function datasetFieldOptions(datasetId: string) {
  const dataset = datasets.value.find((item) => item.id === datasetId)
  return (dataset?.fields || []).map((field) => ({
    label: `${field.name} (${field.code})`,
    value: field.code,
  }))
}

function datasetName(datasetId: string) {
  return datasets.value.find((item) => item.id === datasetId)?.name || datasetId
}

function fieldName(datasetId: string, fieldCode: string) {
  const dataset = datasets.value.find((item) => item.id === datasetId)
  return dataset?.fields.find((item) => item.code === fieldCode)?.name || fieldCode
}

function joinLabel(join: ReportDatasetJoin) {
  const relation = join.join_type === 'inner' ? '=' : '⇐'
  return `${datasetName(join.left_dataset_id)}.${fieldName(join.left_dataset_id, join.left_field)} ${relation} ${datasetName(join.right_dataset_id)}.${fieldName(join.right_dataset_id, join.right_field)}`
}

function openJoinDialog() {
  const left = primaryDataset.value || datasets.value[0]
  const right = datasets.value.find((item) => item.id !== left?.id) || datasets.value[1]
  const suggested = suggestJoinFields(left, right)
  joinDraft.left_dataset_id = left?.id || ''
  joinDraft.left_field = suggested.leftField || left?.fields[0]?.code || ''
  joinDraft.right_dataset_id = right?.id || ''
  joinDraft.right_field = suggested.rightField || right?.fields[0]?.code || ''
  joinDraft.join_type = 'left'
  joinDialogVisible.value = true
}

function suggestJoinFields(left?: ReportDataset, right?: ReportDataset) {
  if (!left || !right) return { leftField: '', rightField: '' }
  const rightKey = reportSourceKey(right)
  const leftIdField =
    left.fields.find((field) => field.code === `${rightKey}_id`) ||
    left.fields.find((field) => field.code.endsWith('_id')) ||
    left.fields.find((field) => field.code === 'id')
  const rightIdField =
    right.fields.find((field) => field.code === 'id') ||
    right.fields.find((field) => field.code.endsWith('_id')) ||
    right.fields[0]
  return {
    leftField: leftIdField?.code || '',
    rightField: rightIdField?.code || '',
  }
}

function reportSourceKey(dataset: ReportDataset) {
  const raw = dataset.source_code || dataset.name || dataset.id
  const parts = raw.split('_').filter(Boolean)
  return parts[parts.length - 1] || raw
}

function handleJoinLeftDatasetChange(id: string) {
  joinDraft.left_dataset_id = id
  joinDraft.left_field = datasets.value.find((item) => item.id === id)?.fields[0]?.code || ''
}

function handleJoinRightDatasetChange(id: string) {
  joinDraft.right_dataset_id = id
  joinDraft.right_field = datasets.value.find((item) => item.id === id)?.fields[0]?.code || ''
}

function confirmJoin() {
  if (
    !joinDraft.left_dataset_id ||
    !joinDraft.left_field ||
    !joinDraft.right_dataset_id ||
    !joinDraft.right_field ||
    joinDraft.left_dataset_id === joinDraft.right_dataset_id
  ) {
    $q.notify({ type: 'warning', message: '请完整配置两个不同数据集的关联字段' })
    return
  }
  datasetJoins.value.push({
    id: `join_${Date.now()}`,
    left_dataset_id: joinDraft.left_dataset_id,
    left_field: joinDraft.left_field,
    right_dataset_id: joinDraft.right_dataset_id,
    right_field: joinDraft.right_field,
    join_type: joinDraft.join_type,
  })
  joinDialogVisible.value = false
}

function removeJoin(id: string) {
  datasetJoins.value = datasetJoins.value.filter((item) => item.id !== id)
}

function applyReportKind(kind: ReportKind) {
  form.report_kind = kind
  if (kind === 'detail') {
    return
  }
  if (kind === 'summary' && !(sheet.value.summary_rows || []).length) {
    sheet.value.summary_rows = [3]
  }
}

function buildLocalPreview() {
  previewData.value = buildReportLocalPreview(datasets.value, sheet.value)
}

async function preview() {
  syncForm()
  previewDialogVisible.value = true
  if (!form.id) {
    buildLocalPreview()
    return
  }
  previewLoading.value = true
  try {
    const res = await reportApi.previewReport({
      report_id: form.id,
      dataset_id: datasetJoins.value.length ? undefined : primaryDataset.value?.id,
      data_source_id: primaryDataset.value?.source_code,
      page: 1,
      num: form.runtime_display === 'all' ? 10000 : Number(form.runtime_page_size || 20),
    })
    previewData.value = res.data
  } catch {
    buildLocalPreview()
    $q.notify({ type: 'negative', message: '真实数据预览失败，已显示本地结构预览' })
  } finally {
    previewLoading.value = false
  }
}

async function saveReport(status: 'draft' | 'published' = 'draft') {
  syncForm()
  if (!validateReport(status === 'published')) return false
  form.status = status
  saving.value = true
  try {
    if (form.id) {
      await reportApi.updateReport(form)
    } else {
      const res = await reportApi.createReport(form)
      form.id = res.data
    }
    $q.notify({ type: 'positive', message: '报表设计已保存' })
    return true
  } catch {
    $q.notify({ type: 'negative', message: '报表保存失败' })
    return false
  } finally {
    saving.value = false
  }
}

async function publishReport() {
  const saved = await saveReport('published')
  if (saved && form.id) {
    $q.notify({ type: 'positive', message: '报表已发布，可在报表中心运行' })
  }
}

function syncForm() {
  datasets.value = ensureDesignerDatasets(datasets.value)
  const primary = primaryDataset.value
  form.data_source_id = primary?.source_code || ''
  form.permission_table_code = form.permission_table_code || primary?.source_code || ''
  form.fields = usedFields.value
  form.datasets = datasets.value.map((dataset) => ({
    ...dataset,
    fields: (dataset.fields || []).map((field) => ({ ...field })),
  }))
  form.dataset_joins = datasetJoins.value
  form.parameters = parameters.value
  form.sheet = normalizeReportSheet(sheet.value)
}

function validateReport(strict = true) {
  if (!form.report_name.trim()) {
    $q.notify({ type: 'warning', message: '请填写报表名称' })
    return false
  }
  if (!form.report_code.trim()) {
    $q.notify({ type: 'warning', message: '请填写报表编码' })
    return false
  }
  if (!strict) return true
  if (!primaryDataset.value || primaryDataset.value.type !== 'table') {
    $q.notify({ type: 'warning', message: '第一版运行必须设置一个现有表作为主数据集' })
    return false
  }
  if (usedFields.value.length === 0) {
    $q.notify({ type: 'warning', message: '请至少绑定一个主数据集字段' })
    return false
  }
  return true
}

function validateAndNotify() {
  syncForm()
  if (validateReport(true)) {
    $q.notify({ type: 'positive', message: '配置检查通过' })
  }
}

function goBack() {
  void router.push({ name: 'report_manage' })
}
</script>

<style scoped lang="scss">
.report-designer-fullscreen {
  position: fixed;
  inset: 0;
  z-index: 2500;
  display: grid;
  grid-template-rows: 72px minmax(0, 1fr) 30px;
  background: #f6f8fc;
  color: #172033;
}

.dialog-caption {
  color: #71809a;
}

.designer-workbench {
  min-height: 0;
  display: grid;
  grid-template-columns: 300px minmax(760px, 1fr) 330px;
  overflow: auto;
}

.designer-statusbar {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 0 14px;
  border-top: 1px solid #dfe5f2;
  background: #fff;
  color: #71809a;
  font-size: 12px;
}

.preview-dialog {
  width: min(1040px, 94vw);
}

.preview-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}

.dialog-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid #e7ecf6;
}

.dialog-title {
  font-size: 20px;
  font-weight: 900;
}

@media (max-width: 1360px) {
  .designer-workbench {
    grid-template-columns: 270px minmax(720px, 1fr) 300px;
  }
}
</style>
