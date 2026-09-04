<template>
  <div class="report-designer-fullscreen">
    <report-designer-topbar
      :report-name="form.report_name"
      :report-code="form.report_code"
      :primary-source-code="primaryDataset?.source_code"
      :report-status="form.status || 'draft'"
      :published-version-no="publishedVersionNo"
      :saving="saving"
      :previewing="saving || previewLoading"
      :publishing="saving || publishing"
      :preview-disabled="form.status === 'disabled'"
      :publish-disabled="form.status === 'disabled'"
      :version-disabled="!form.id"
      @update:report-name="form.report_name = $event"
      @update:report-code="form.report_code = $event"
      @back="goBack"
      @add-parameter="addParameter"
      @preview="preview"
      @validate="validateAndNotify"
      @save-draft="saveReport('draft')"
      @publish="publishReport"
      @versions="openVersionDialog"
    />

    <main class="designer-workbench">
      <nav class="designer-palette">
        <q-btn
          flat
          dense
          icon="tune"
          :aria-label="t('ui.properties')"
          :class="{ active: sidePanelTab === 'properties' }"
          @click="showProperties('cell')"
        >
          <q-tooltip>{{ t('ui.cells') }}</q-tooltip>
        </q-btn>
        <q-btn
          flat
          dense
          icon="storage"
          :aria-label="t('ui.dataSourceTab')"
          :class="{ active: sidePanelTab === 'dataSource' }"
          @click="sidePanelTab = 'dataSource'"
        >
          <q-tooltip>{{ t('ui.dataset') }}</q-tooltip>
        </q-btn>
        <q-separator />
        <q-btn flat dense icon="filter_alt" :aria-label="t('ui.parameters')" @click="addParameter">
          <q-tooltip>{{ t('ui.parameters') }}</q-tooltip>
        </q-btn>
        <q-btn
          flat
          dense
          icon="account_tree"
          :aria-label="t('ui.dataSetAssociation')"
          :disable="datasets.length < 2"
          @click="openJoinDialog()"
        >
          <q-tooltip>{{ t('ui.dataSetAssociation') }}</q-tooltip>
        </q-btn>
        <q-btn
          flat
          dense
          icon="functions"
          :aria-label="t('ui.summaryRow')"
          @click="toggleActiveSummaryRow"
        >
          <q-tooltip>{{ t('ui.summaryRow') }}</q-tooltip>
        </q-btn>
      </nav>

      <report-sheet-canvas
        v-if="designerReady"
        :sheet="sheet"
        :selected-cell-id="selectedCellId"
        :datasets="datasets"
        :dataset-joins="datasetJoins"
        :field-dragging="!!draggingField"
        @update:sheet="applyUniverSheet"
        @select-cell="selectCell"
        @select-range="selectRange"
        @drop-field="dropField"
        @move-bound-cell="moveBoundCell"
        @toggle-summary-row="toggleSummaryRow"
        @toggle-group-summary-row="toggleGroupSummaryRow"
        @toggle-detail-row="toggleDetailRow"
      />
      <div v-else class="designer-canvas-loading">
        <q-spinner color="primary" size="36px" />
      </div>

      <aside class="designer-side-panel">
        <q-tabs
          v-model="sidePanelTab"
          class="side-panel-tabs"
          dense
          inline-label
          narrow-indicator
          no-caps
          align="justify"
          active-color="primary"
          indicator-color="primary"
        >
          <q-tab name="properties" icon="tune" :label="t('ui.properties')" />
          <q-tab name="dataSource" icon="storage" :label="t('ui.dataSourceTab')" />
        </q-tabs>
        <q-separator />
        <q-tab-panels v-model="sidePanelTab" class="designer-side-panels">
          <q-tab-panel name="properties" class="side-panel-page">
            <report-inspector-panel
              v-model:tab="inspectorTab"
              v-model:cell-value="activeCellValue"
              v-model:binding-type="activeBindingType"
              v-model:binding-dataset-id="activeBindingDatasetId"
              v-model:binding-field="activeBindingField"
              v-model:formula="activeFormula"
              :formula-error="activeFormulaError"
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
              :report-kind="form.report_kind"
              :report-kind-options="reportKindOptions"
              :runtime-display="form.runtime_display || 'paged'"
              :runtime-page-size="form.runtime_page_size || 20"
              :runtime-display-options="reportRuntimeDisplayOptions"
              @update-dataset-name="updateDatasetName"
              @set-primary-dataset="setPrimaryDataset"
              @add-join="openJoinDialog()"
              @edit-join="openJoinDialog"
              @remove-join="removeJoin"
              @update:category="form.category = $event"
              @update:description="form.description = $event"
              @update:report-kind="applyReportKind"
              @update:runtime-display="form.runtime_display = $event"
              @update:runtime-page-size="form.runtime_page_size = $event"
            />
          </q-tab-panel>
          <q-tab-panel name="dataSource" class="side-panel-page">
            <report-resource-panel
              :datasets="datasets"
              :parameters="parameters"
              :selected-dataset-id="selectedDatasetId"
              :selected-parameter-id="selectedParameterId"
              @open-dataset="openDatasetDialog"
              @add-join="openJoinDialog()"
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
          </q-tab-panel>
        </q-tab-panels>
      </aside>
    </main>

    <footer class="designer-statusbar">
      <span>{{ t('ui.currentCell') }}{{ activeCellLabel }}</span>
      <span>{{ t('ui.dataSet') }}{{ datasets.length }}</span>
      <span>{{ t('ui.boundFieldsPrefix') }}{{ usedFields.length }}</span>
      <span>{{ reportId ? t('ui.reportId', { id: reportId }) : t('ui.newReport') }}</span>
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
      :editing="!!editingJoinId"
      :draft="joinDraft"
      :dataset-options="datasetOptions"
      :left-field-options="joinLeftFieldOptions"
      :right-field-options="joinRightFieldOptions"
      :join-type-options="reportDatasetJoinTypeOptions"
      @update:left-dataset-id="handleJoinLeftDatasetChange"
      @update:right-dataset-id="handleJoinRightDatasetChange"
      @update:join-type="joinDraft.join_type = $event"
      @update:conditions="joinDraft.conditions = $event"
      @confirm="confirmJoin"
    />

    <q-dialog v-model="previewDialogVisible">
      <q-card class="preview-dialog">
        <q-card-section class="dialog-head">
          <div>
            <div class="dialog-title">{{ t('ui.runPreview') }}</div>
            <div class="dialog-caption">
              {{
                previewUsingBackend
                  ? t('ui.realDataPreviewBackendDataPermissions')
                  : t('ui.previewOfLocalStructuresWithoutSavingReports')
              }}
            </div>
          </div>
          <q-btn flat round dense icon="close" v-close-popup />
        </q-card-section>
        <q-card-section>
          <div class="preview-meta">
            <q-chip v-if="previewUsingBackend" dense square color="primary" text-color="white">
              {{ previewData.total }} {{ t('ui.okay') }}
            </q-chip>
            <q-chip v-if="previewData.meta?.version_no" dense square outline color="primary">
              {{ t('ui.previewV') }}{{ previewData.meta.version_no }}
            </q-chip>
            <q-chip v-if="previewData.meta?.dataset_id" dense square outline color="primary">
              {{ t('ui.mainDataSet') }} {{ previewData.meta.dataset_id }}
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

    <report-version-dialog
      v-model="versionDialogVisible"
      :report-id="form.id"
      :current-version-id="publishedVersionId"
      :current-version-no="publishedVersionNo"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

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
  type ReportDatasetJoinCondition,
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
} from '@/api/services/report'
import ReportDesignerTopbar from './components/ReportDesignerTopbar.vue'
import ReportDatasetDialog from './components/ReportDatasetDialog.vue'
import ReportInspectorPanel from './components/ReportInspectorPanel.vue'
import ReportJoinDialog from './components/ReportJoinDialog.vue'
import ReportParameterDialog from './components/ReportParameterDialog.vue'
import ReportResourcePanel from './components/ReportResourcePanel.vue'
import ReportSheetCanvas from './components/ReportSheetCanvas.vue'
import ReportSheetPreview from '../components/ReportSheetPreview.vue'
import ReportVersionDialog from '../components/ReportVersionDialog.vue'
import {
  reportAlignOptions,
  reportDatasetJoinTypeOptions,
  reportBindingTypeOptions,
  reportDatasetTypeOptions,
  reportKindOptions,
  reportParameterOperatorOptions,
  reportParameterTypeOptions,
  reportRuntimeDisplayOptions,
} from '@/modules/report/options'
import {
  buildReportLocalPreview,
  collectReportUsedFields,
  defaultReportBindingType,
  reportBindingText,
  reportCellId,
  reportColumnName,
  reportNormalizeSheetRange,
  reportSheetCellAt,
  reportValidateFormula,
  type ReportSheetRange,
} from '@/modules/report/sheet'
import {
  hasReportCellConfig,
  normalizeReportSheet,
  reportParameterDefaultsForField,
} from '@/modules/report/schema'
import { useTagViewStore } from '@/stores/tagView'

const { t } = useI18n({ useScope: 'global' })

const $q = useQuasar()
const route = useRoute()
const router = useRouter()
const reportApi = useReportApi()
const tagViewStore = useTagViewStore()

const reportId = computed(() => Number(route.query.id || 0))
const saving = ref(false)
const publishing = ref(false)
const previewLoading = ref(false)
const previewUsingBackend = ref(false)
const designerReady = ref(false)
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
const sidePanelTab = ref<'properties' | 'dataSource'>('dataSource')
const draggingField = ref<{ datasetId: string; fieldCode: string } | null>(null)
const datasetDialogVisible = ref(false)
const editingDatasetId = ref('')
const sqlFieldsLoading = ref(false)
const parameterDialogVisible = ref(false)
const editingParameterId = ref('')
const joinDialogVisible = ref(false)
const editingJoinId = ref('')
const previewDialogVisible = ref(false)
const versionDialogVisible = ref(false)
const previewData = ref<ReportPreviewRes>({ columns: [], rows: [], total: 0 })
const publishedVersionId = ref<number | undefined>(undefined)
const publishedVersionNo = ref<number | undefined>(undefined)
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
  right_dataset_id: string
  join_type: ReportDatasetJoinType
  conditions: ReportDatasetJoinCondition[]
}>({
  left_dataset_id: '',
  right_dataset_id: '',
  join_type: 'left',
  conditions: [{ left_field: '', right_field: '' }],
})

const sqlDraftFields = ref<ReportField[]>([])
const datasetOptions = computed(() =>
  datasets.value.map((item) => ({
    label: `${item.name} (${item.type === 'sql' ? 'SQL' : item.source_code})`,
    value: item.id,
  })),
)
const primaryDataset = computed(
  () => datasets.value.find((item) => item.primary) || datasets.value[0],
)
const selectedDataset = computed(() =>
  datasets.value.find((item) => item.id === selectedDatasetId.value),
)
const editingDataset = computed(() =>
  datasets.value.find((item) => item.id === editingDatasetId.value),
)
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
  if (binding?.type === 'formula') return normalizeFormulaDisplay(binding.formula || '')
  if (!binding?.dataset_id || !binding.field) return ''
  const dataset = datasets.value.find((item) => item.id === binding.dataset_id)
  const field = dataset?.fields.find((item) => item.code === binding.field)
  if (!dataset || !field) return ''
  return reportBindingText(binding.type, dataset, field)
})
const usedFields = computed(() => collectReportUsedFields(datasets.value, sheet.value))
const boundDatasetIds = computed(() => {
  const ids = new Set<string>()
  sheet.value.cells.forEach((cell) => {
    const binding = cell.binding
    if (!binding?.dataset_id || !binding.field || binding.type === 'static') return
    ids.add(binding.dataset_id)
  })
  return [...ids]
})
const datasetDraftPreviewFields = computed(() => {
  if (datasetDraft.type === 'table') {
    return dataSources.value.find((item) => item.code === datasetDraft.source_code)?.fields || []
  }
  return sqlDraftFields.value.length
    ? sqlDraftFields.value
    : parseSqlDatasetFields(datasetDraft.fieldsText)
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
  set: (value: string) => {
    const cell = activeCell.value
    if (cell?.binding?.type === 'formula') {
      patchActiveCell({
        value: normalizeFormulaDisplay(value),
        binding: { ...cell.binding, formula: value },
      })
    } else {
      patchActiveCell({ value })
    }
    buildLocalPreview()
  },
})
const activeBindingType = computed<ReportCellBindingType>({
  get: () => activeCell.value?.binding?.type || 'static',
  set: (value) => {
    const cell = activeCell.value
    if (!cell) return
    if (value === 'formula') {
      patchActiveCell({
        binding: {
          type: 'formula',
          formula: cell.binding?.formula || '',
        },
      })
    } else {
      patchBinding({ type: value })
    }
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
    if (field)
      patchActiveCell({ value: reportBindingText(activeBindingType.value, dataset!, field) })
    buildLocalPreview()
  },
})
const activeFormula = computed({
  get: () => activeCell.value?.binding?.formula || '',
  set: (value: string) => {
    const cell = activeCell.value
    if (!cell) return
    patchActiveCell({
      value: normalizeFormulaDisplay(value),
      binding: {
        ...(cell.binding || {}),
        type: 'formula',
        formula: value,
      },
    })
    buildLocalPreview()
  },
})
const activeFormulaError = computed(() => {
  const formula = activeFormula.value.trim()
  if (!formula) return ''
  return formulaErrorMessage(reportValidateFormula(formula))
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
  tagViewStore.forgetTagView(route.fullPath)
  try {
    await loadDataSources()
    await loadReport()
    buildLocalPreview()
  } finally {
    designerReady.value = true
  }
})

async function loadDataSources() {
  try {
    const res = await reportApi.queryDataSources()
    dataSources.value = res.data || []
  } catch {
    dataSources.value = []
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.dataSourceLoadFailed')
      },
    })
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
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.failedToLoadReport')
      },
    })
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
  form.status = report.status || 'draft'
  form.permission_menu_id = report.permission_menu_id || 0
  form.permission_table_code = report.permission_table_code || report.source_code || ''
  form.runtime_display = report.layout_config?.runtime_display || 'paged'
  form.runtime_page_size = Number(report.layout_config?.runtime_page_size || 20)
  publishedVersionId.value = report.published_version_id
  publishedVersionNo.value = report.published_version_no
  datasets.value = enrichReportDatasets(
    report.layout_config?.datasets?.length
      ? report.layout_config.datasets
      : createInitialDatasets(report.source_code),
  )
  datasetJoins.value =
    report.layout_config?.dataset_joins || report.query_config?.dataset_joins || []
  sheet.value = normalizeReportSheet(report.layout_config?.sheet || defaultReportSheet())
  parameters.value = report.layout_config?.parameters || report.query_config?.parameters || []
  selectedDatasetId.value = datasets.value[0]?.id || ''
  selectedCellId.value = sheet.value.active_cell || '1:1'
  syncForm()
}

function initNewReport() {
  form.id = undefined
  form.report_name = ''
  form.report_code = `report_${Date.now().toString().slice(-6)}`
  form.report_kind = 'detail'
  form.category = ''
  form.description = ''
  form.status = 'draft'
  form.permission_menu_id = 0
  form.permission_table_code = ''
  form.runtime_display = 'paged'
  form.runtime_page_size = 20
  publishedVersionId.value = undefined
  publishedVersionNo.value = undefined
  datasets.value = []
  datasetJoins.value = []
  sheet.value = defaultReportSheet()
  parameters.value = []
  selectedDatasetId.value = ''
  selectedCellId.value = '1:1'
  sidePanelTab.value = 'dataSource'
  syncForm()
  datasetDialogVisible.value = true
}

function createInitialDatasets(sourceCode?: string): ReportDataset[] {
  const source = dataSources.value.find((item) => item.code === sourceCode) || dataSources.value[0]
  if (!source) return []
  return [
    {
      id: 'main',
      name: source.name || t('ui.primaryData'),
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
  sidePanelTab.value = 'dataSource'
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
  datasetDraft.source_code = ''
  datasetDraft.name = ''
  datasetDraft.sql = ''
  datasetDraft.fieldsText = ''
  sqlDraftFields.value = []
  datasetDialogVisible.value = true
}

function handleDraftTypeChange(value: ReportDatasetType) {
  datasetDraft.type = value
  if (datasetDraft.type === 'table') {
    sqlDraftFields.value = []
    handleDraftSourceChange(datasetDraft.source_code || '')
  } else {
    datasetDraft.name = t('ui.sqlDataset')
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
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.pleaseFillInSql')
      },
    })
    return false
  }
  sqlFieldsLoading.value = true
  try {
    const res = await reportApi.inferSqlFields(sql)
    sqlDraftFields.value = res.data || []
    datasetDraft.fieldsText = sqlDraftFields.value.map((field) => field.code).join(',')
    if (!sqlDraftFields.value.length) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.noFieldsParsedFromSql')
        },
      })
      return false
    }
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.parsedFields', { value1: sqlDraftFields.value.length })
      },
    })
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
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.selectSourceTable')
        },
      })
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
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.sqlRequired')
        },
      })
      return
    }
    let fields = datasetDraftPreviewFields.value
    if (!datasetDraft.sql.trim() || fields.length === 0) {
      const ok = await inferSqlDatasetFields()
      fields = datasetDraftPreviewFields.value
      if (!ok || fields.length === 0) {
        $q.notify({
          type: 'warning',
          get message() {
            return t('ui.pleaseParseSqlFieldsFirst')
          },
        })
        return
      }
    }
    upsertDataset({
      id,
      name: datasetDraft.name || t('ui.sqlDataset'),
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
  sheet.value.cells = sheet.value.cells
    .map((cell) => {
      if (cell.binding?.dataset_id !== id) return cell
      const style = { ...(cell.style || {}) }
      delete style.color
      return { ...cell, value: '', binding: undefined, style }
    })
    .filter(hasReportCellConfig)
  if (removed?.primary && datasets.value[0]) datasets.value[0].primary = true
  selectedDatasetId.value = datasets.value[0]?.id || ''
  buildLocalPreview()
}

function setPrimaryDataset(id: string) {
  const dataset = datasets.value.find((item) => item.id === id)
  if (!dataset || dataset.type !== 'table') {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.onlyExistingTableDataSetsCanBeUsedAsThe')
      },
    })
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

function moveBoundCell(sourceRow: number, sourceCol: number, targetRow: number, targetCol: number) {
  if (sourceRow === targetRow && sourceCol === targetCol) return
  const source = cellAt(sourceRow, sourceCol)
  if (!source.binding || source.binding.type === 'static') return
  const target = cellAt(targetRow, targetCol)
  patchCell(targetRow, targetCol, {
    value: source.value,
    binding: { ...source.binding },
    style: { ...(target.style || {}), ...(source.style || {}) },
  })
  const sourceStyle = { ...(source.style || {}) }
  delete sourceStyle.color
  delete sourceStyle.bold
  patchCell(sourceRow, sourceCol, { value: '', binding: undefined, style: sourceStyle })
  const sourceStillBound = sheet.value.cells.some(
    (cell) => cell.row === sourceRow && cell.binding && cell.binding.type !== 'static',
  )
  if (!sourceStillBound) {
    sheet.value.detail_rows = (sheet.value.detail_rows || []).filter((row) => row !== sourceRow)
  }
  markDetailRow(targetRow)
  selectCell(targetRow, targetCol)
  buildLocalPreview()
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
  })
  markDetailRow(row)
  selectedCellId.value = reportCellId(row, col)
  inspectorTab.value = 'cell'
  sidePanelTab.value = 'properties'
  buildLocalPreview()
}

function cellAt(row: number, col: number) {
  return reportSheetCellAt(sheet.value, row, col)
}

function patchCell(row: number, col: number, patch: Partial<ReportSheetCell>) {
  const index = sheet.value.cells.findIndex((item) => item.row === row && item.col === col)
  const current =
    index === -1 ? { id: reportCellId(row, col), row, col, value: '' } : sheet.value.cells[index]!
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

function applyUniverSheet(value: ReportSheetConfig) {
  sheet.value = value
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
  const range = selectionRange.value
  if (!range) {
    const cell = activeCell.value
    if (!cell) return
    patchActiveCell({ style: { ...(cell.style || {}), ...patch } })
    buildLocalPreview()
    return
  }
  const bounds = reportNormalizeSheetRange(range)
  cellsInBounds(bounds).forEach((cell) => {
    patchCell(cell.row, cell.col, { style: { ...(cell.style || {}), ...patch } })
  })
  buildLocalPreview()
}

function refreshActiveCellBindingText() {
  const cell = activeCell.value
  const binding = cell?.binding
  if (cell && binding?.type === 'formula') {
    const nextValue = normalizeFormulaDisplay(binding.formula || '')
    if (!cell.value || isGeneratedBindingValue(cell.value) || cell.value.startsWith('=')) {
      patchActiveCell({ value: nextValue })
    }
    buildLocalPreview()
    return
  }
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

function normalizeFormulaDisplay(value: string) {
  const formula = value.trim()
  if (!formula) return ''
  return formula.startsWith('=') ? formula : `=${formula}`
}

function formulaErrorMessage(error: ReturnType<typeof reportValidateFormula>) {
  if (!error) return ''
  if (error === 'division_by_zero') return t('ui.formulaDivisionByZero')
  if (error === 'circular_reference') return t('ui.formulaCircularReference')
  return t('ui.formulaSyntaxError')
}

function isGeneratedBindingValue(value: string) {
  return datasets.value.some((dataset) =>
    dataset.fields.some((field) =>
      ['field', 'group', 'sum', 'count', 'avg', 'max', 'min', 'formula'].some(
        (type) => reportBindingText(type as ReportCellBindingType, dataset, field) === value,
      ),
    ),
  )
}

function selectCell(row: number, col: number) {
  selectedCellId.value = reportCellId(row, col)
  selectionRange.value = { startRow: row, startCol: col, endRow: row, endCol: col }
  sheet.value.active_cell = selectedCellId.value
  selectedParameterId.value = ''
  showProperties('cell')
}

function selectRange(range: ReportSheetRange) {
  selectionRange.value = range
  selectedParameterId.value = ''
  showProperties('cell')
}

function showProperties(tab: 'cell' | 'data' | 'report') {
  sidePanelTab.value = 'properties'
  inspectorTab.value = tab
}

function toggleActiveSummaryRow() {
  const row = activeCell.value?.row
  if (!row) return
  toggleSummaryRow(row)
  showProperties('cell')
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

function toggleSummaryRow(row: number) {
  const rows = new Set(sheet.value.summary_rows || [])
  if (rows.has(row)) {
    rows.delete(row)
    sheet.value.group_summary_rows = (sheet.value.group_summary_rows || []).filter(
      (item) => item !== row,
    )
  } else {
    rows.add(row)
  }
  sheet.value.summary_rows = [...rows].sort((a, b) => a - b)
  if (rows.has(row)) {
    sheet.value.detail_rows = (sheet.value.detail_rows || []).filter((item) => item !== row)
  }
  buildLocalPreview()
}

function toggleGroupSummaryRow(row: number) {
  const groupRows = new Set(sheet.value.group_summary_rows || [])
  const summaryRows = new Set(sheet.value.summary_rows || [])
  if (groupRows.has(row)) {
    groupRows.delete(row)
  } else {
    groupRows.add(row)
    summaryRows.add(row)
    sheet.value.detail_rows = (sheet.value.detail_rows || []).filter((item) => item !== row)
  }
  sheet.value.group_summary_rows = [...groupRows].sort((a, b) => a - b)
  sheet.value.summary_rows = [...summaryRows].sort((a, b) => a - b)
  buildLocalPreview()
}

function toggleDetailRow(row: number) {
  const rows = new Set(sheet.value.detail_rows || [])
  if (rows.has(row)) rows.delete(row)
  else rows.add(row)
  sheet.value.detail_rows = [...rows].sort((a, b) => a - b)
  if (rows.has(row)) {
    sheet.value.summary_rows = (sheet.value.summary_rows || []).filter((item) => item !== row)
    sheet.value.group_summary_rows = (sheet.value.group_summary_rows || []).filter(
      (item) => item !== row,
    )
  }
  buildLocalPreview()
}

function markDetailRow(row: number) {
  const rows = new Set(sheet.value.detail_rows || [])
  rows.add(row)
  sheet.value.detail_rows = [...rows].sort((a, b) => a - b)
  sheet.value.summary_rows = (sheet.value.summary_rows || []).filter((item) => item !== row)
  sheet.value.group_summary_rows = (sheet.value.group_summary_rows || []).filter(
    (item) => item !== row,
  )
}

function addParameter() {
  sidePanelTab.value = 'dataSource'
  openParameterDialog('')
}

function openParameterDialog(id = '') {
  sidePanelTab.value = 'dataSource'
  const dataset = primaryDataset.value
  const field = dataset?.fields[0]
  if (!dataset || !field) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.addTableDataSetFirst')
      },
    })
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
  parameterDraft.placeholder = current?.placeholder || t('ui.pleaseEnter', { value1: field.name })
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
    parameterDraft.placeholder = t('ui.pleaseEnter', { value1: field.name })
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
  parameterDraft.placeholder = t('ui.pleaseEnter', { value1: field.name })
  parameterDraft.default_value = ''
  const defaults = reportParameterDefaultsForField(field)
  parameterDraft.type = defaults.type
  parameterDraft.operator = defaults.operator
}

function confirmParameter() {
  if (!parameterDraft.label.trim() || !parameterDraft.dataset_id || !parameterDraft.field) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.pleaseCompleteTheParameterNameDataSetAndFieldConfiguration')
      },
    })
    return
  }
  const dataset = datasets.value.find((item) => item.id === parameterDraft.dataset_id)
  const field = dataset?.fields.find((item) => item.code === parameterDraft.field)
  const compatibilityError = parameterCompatibilityError(
    parameterDraft.type,
    parameterDraft.operator,
    field,
  )
  if (compatibilityError) {
    $q.notify({ type: 'warning', message: compatibilityError })
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
  if (Array.isArray(value))
    return value
      .map((item) => String(item ?? ''))
      .filter(Boolean)
      .join(',')
  if (value === null || value === undefined) return ''
  return String(value)
}

function parseParameterDefault(
  value: string,
  type: ReportParameterType,
): ReportParameter['default_value'] {
  const text = value.trim()
  if (!text) return undefined
  if (type === 'date_range') {
    const parts = text
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
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
    return t('ui.sqlFieldParsingInterfaceIsNotValidPleaseAssignSql')
  }
  return t('ui.failedToParseSqlFieldsCheckTheSqlOrBackend')
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
  const condition = join.conditions[0]
  if (!condition)
    return `${datasetName(join.left_dataset_id)} ${relation} ${datasetName(join.right_dataset_id)}`
  const suffix = join.conditions.length > 1 ? ` +${join.conditions.length - 1}` : ''
  return `${datasetName(join.left_dataset_id)}.${fieldName(join.left_dataset_id, condition.left_field)} ${relation} ${datasetName(join.right_dataset_id)}.${fieldName(join.right_dataset_id, condition.right_field)}${suffix}`
}

function openJoinDialog(id = '') {
  sidePanelTab.value = 'properties'
  inspectorTab.value = 'data'
  editingJoinId.value = id
  const current = datasetJoins.value.find((item) => item.id === id)
  if (current) {
    joinDraft.left_dataset_id = current.left_dataset_id
    joinDraft.right_dataset_id = current.right_dataset_id
    joinDraft.join_type = current.join_type
    joinDraft.conditions = current.conditions.map((condition) => ({ ...condition }))
    joinDialogVisible.value = true
    return
  }
  const left = primaryDataset.value || datasets.value[0]
  const connectedDatasetIds = new Set<string>([left?.id || ''])
  datasetJoins.value.forEach((join) => {
    connectedDatasetIds.add(join.left_dataset_id)
    connectedDatasetIds.add(join.right_dataset_id)
  })
  const right =
    datasets.value.find((item) => item.id !== left?.id && !connectedDatasetIds.has(item.id)) ||
    datasets.value.find((item) => item.id !== left?.id) ||
    datasets.value[1]
  const suggested = suggestJoinFields(left, right)
  joinDraft.left_dataset_id = left?.id || ''
  joinDraft.right_dataset_id = right?.id || ''
  joinDraft.join_type = 'left'
  joinDraft.conditions = [
    {
      left_field: suggested.leftField || left?.fields[0]?.code || '',
      right_field: suggested.rightField || right?.fields[0]?.code || '',
    },
  ]
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
  const firstField = datasets.value.find((item) => item.id === id)?.fields[0]?.code || ''
  joinDraft.conditions = joinDraft.conditions.map((condition) => ({
    ...condition,
    left_field: firstField,
  }))
}

function handleJoinRightDatasetChange(id: string) {
  joinDraft.right_dataset_id = id
  const firstField = datasets.value.find((item) => item.id === id)?.fields[0]?.code || ''
  joinDraft.conditions = joinDraft.conditions.map((condition) => ({
    ...condition,
    right_field: firstField,
  }))
}

function confirmJoin() {
  const conditions = joinDraft.conditions
    .map((condition) => ({
      left_field: condition.left_field.trim(),
      right_field: condition.right_field.trim(),
    }))
    .filter((condition) => condition.left_field || condition.right_field)
  if (
    !joinDraft.left_dataset_id ||
    !joinDraft.right_dataset_id ||
    joinDraft.left_dataset_id === joinDraft.right_dataset_id ||
    !conditions.length ||
    conditions.some((condition) => !condition.left_field || !condition.right_field)
  ) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.pleaseConfigureTheRelevantFieldsOfTheTwoDifferentData')
      },
    })
    return
  }
  const conditionKeys = new Set(
    conditions.map((condition) => `${condition.left_field}:${condition.right_field}`),
  )
  if (conditionKeys.size !== conditions.length) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.checkTheDatasetCorrelationFieldWhichMustBeFromThe')
      },
    })
    return
  }
  const duplicate = datasetJoins.value.find(
    (join) =>
      join.id !== editingJoinId.value &&
      join.left_dataset_id === joinDraft.left_dataset_id &&
      join.right_dataset_id === joinDraft.right_dataset_id &&
      JSON.stringify(join.conditions) === JSON.stringify(conditions),
  )
  if (duplicate) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.datasetAssociationAlreadyExists')
      },
    })
    return
  }
  const join: ReportDatasetJoin = {
    id: editingJoinId.value || `join_${Date.now()}`,
    left_dataset_id: joinDraft.left_dataset_id,
    right_dataset_id: joinDraft.right_dataset_id,
    join_type: joinDraft.join_type,
    conditions,
  }
  const index = datasetJoins.value.findIndex((item) => item.id === join.id)
  if (index === -1) datasetJoins.value.push(join)
  else datasetJoins.value[index] = join
  editingJoinId.value = ''
  joinDialogVisible.value = false
}

function removeJoin(id: string) {
  datasetJoins.value = datasetJoins.value.filter((item) => item.id !== id)
  if (editingJoinId.value === id) editingJoinId.value = ''
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
  if (form.status === 'disabled') {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.previewWhenADisabledReportCannotBeDesigned')
      },
    })
    return
  }
  if (!usedFields.value.length) {
    syncForm()
    if (!validateReport(false)) return
    previewUsingBackend.value = false
    previewLoading.value = false
    buildLocalPreview()
    previewDialogVisible.value = true
    return
  }
  const id = await saveReport('draft', { strict: true, notify: false })
  if (!id) return
  previewUsingBackend.value = true
  previewDialogVisible.value = true
  previewLoading.value = true
  try {
    const res = await reportApi.designPreviewReport(id, {
      report_id: id,
      dataset_id: datasetJoins.value.length ? undefined : primaryDataset.value?.id,
      data_source_id: primaryDataset.value?.source_code,
      page: 1,
      num: form.runtime_display === 'all' ? 10000 : Number(form.runtime_page_size || 20),
    })
    previewData.value = res
  } catch (error) {
    previewUsingBackend.value = false
    buildLocalPreview()
    const message =
      error instanceof Error && error.message
        ? error.message
        : t('ui.previewFailedAtDesignLocalStructurePreviewShown')
    $q.notify({ type: 'negative', message })
  } finally {
    previewLoading.value = false
  }
}

async function saveReport(
  status: 'draft' = 'draft',
  options: { strict?: boolean; notify?: boolean } = {},
): Promise<number | null> {
  syncForm()
  const strict = options.strict ?? false
  const shouldNotify = options.notify ?? true
  if (!validateReport(strict)) return null
  form.status = form.id ? form.status || status : status
  saving.value = true
  try {
    if (form.id) {
      await reportApi.updateReport(form)
    } else {
      const res = await reportApi.createReport(form)
      form.id = res.data
      await router.replace({ name: 'report_design', query: { ...route.query, id: form.id } })
    }
    if (shouldNotify) {
      $q.notify({
        type: 'positive',
        get message() {
          return t('ui.reportDesignSaved')
        },
      })
    }
    return form.id || null
  } catch (error) {
    const message =
      error instanceof Error && error.message ? error.message : t('ui.reportSaveFailed')
    $q.notify({ type: 'negative', message })
    return null
  } finally {
    saving.value = false
  }
}

async function publishReport() {
  if (form.status === 'disabled') {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.disableFromReleaseOfDisabledReport')
      },
    })
    return
  }
  const changeLog = await confirmPublishReport()
  if (changeLog === null) return
  const id = await saveReport('draft', { strict: true, notify: false })
  if (!id) return
  publishing.value = true
  try {
    const res = await reportApi.publishReport(id, changeLog ? { change_log: changeLog } : {})
    form.status = res.status || 'published'
    publishedVersionId.value = res.version_id
    publishedVersionNo.value = res.version_no
    await refreshCurrentReport(id)
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.theReportIsPublishedAndCanBeRunAtThe')
      },
    })
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : t('ui.publishFailed')
    $q.notify({ type: 'negative', message })
  } finally {
    publishing.value = false
  }
}

function confirmPublishReport() {
  return new Promise<string | null>((resolve) => {
    $q.dialog({
      get title() {
        return t('ui.publishReport')
      },
      get message() {
        return t('ui.confirmsThatTheCurrentStatementIsBeingIssuedTheNew')
      },
      prompt: {
        model: '',
        type: 'textarea',
        get label() {
          return t('ui.releaseNotesOptional')
        },
      },
      cancel: true,
      persistent: true,
    })
      .onOk((value) => resolve(String(value || '').trim()))
      .onCancel(() => resolve(null))
      .onDismiss(() => resolve(null))
  })
}

async function refreshCurrentReport(id: number) {
  try {
    const res = await reportApi.queryReportById(id)
    applyReport(res.data)
  } catch {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.releaseSuccessfullyButFailureToRefreshTheDetailsOfThe')
      },
    })
  }
}

function openVersionDialog() {
  if (!form.id) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.pleaseSaveTheReportAndSeeTheVersion')
      },
    })
    return
  }
  versionDialogVisible.value = true
}

function syncForm() {
  datasets.value = ensureDesignerDatasets(datasets.value)
  const primary = primaryDataset.value
  form.data_source_id = primary?.source_code || ''
  form.permission_table_code = primary?.source_code || form.permission_table_code || ''
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
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.pleaseFillInTheNameOfTheReport')
      },
    })
    return false
  }
  if (!form.report_code.trim()) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.pleaseCompleteTheReportCode')
      },
    })
    return false
  }
  if (!strict) return true
  if (!primaryDataset.value || primaryDataset.value.type !== 'table') {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.theFirstEditionMustBeRunWithAnExistingTable')
      },
    })
    return false
  }
  if (usedFields.value.length === 0) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.pleaseBindAtLeastOnePrimaryDataSetField')
      },
    })
    return false
  }
  if (!validateSqlDatasets()) {
    return false
  }
  if (!validateBoundCells()) {
    return false
  }
  if (!validateParameters()) {
    return false
  }
  if (!validateRuntimeSettings()) {
    return false
  }
  if (!validateDatasetJoins()) {
    return false
  }
  return true
}

function validateAndNotify() {
  syncForm()
  if (validateReport(true)) {
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.configureCheckPassed')
      },
    })
  }
}

function validateDatasetJoins() {
  for (const join of datasetJoins.value) {
    const leftDataset = datasets.value.find((item) => item.id === join.left_dataset_id)
    const rightDataset = datasets.value.find((item) => item.id === join.right_dataset_id)
    if (!leftDataset || !rightDataset || leftDataset.id === rightDataset.id) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.checkTheDataSetAssociationBothEndsMustBeDifferent')
        },
      })
      return false
    }
    const conditionsValid =
      join.conditions.length > 0 &&
      join.conditions.every(
        (condition) =>
          leftDataset.fields.some((field) => field.code === condition.left_field) &&
          rightDataset.fields.some((field) => field.code === condition.right_field),
      )
    const conditionKeys = new Set(
      join.conditions.map((condition) => `${condition.left_field}:${condition.right_field}`),
    )
    if (!conditionsValid || conditionKeys.size !== join.conditions.length) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.checkTheDatasetCorrelationFieldWhichMustBeFromThe')
        },
      })
      return false
    }
  }

  const primaryId = primaryDataset.value?.id || ''
  const connectedDatasetIds = orderableDatasetJoinIds(datasetJoins.value, primaryId)
  if (datasetJoins.value.length && !connectedDatasetIds) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.datasetAssociationsMustFormATreeFromThePrimaryDataset')
      },
    })
    return false
  }

  if (boundDatasetIds.value.length <= 1) return true
  if (!datasetJoins.value.length) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.theCurrentReportUsesMultipleDatasetFieldsPleaseConfigureThe')
      },
    })
    return false
  }
  const unjoinedDataset = boundDatasetIds.value.find((id) => !connectedDatasetIds?.has(id))
  if (unjoinedDataset) {
    const datasetName =
      datasets.value.find((item) => item.id === unjoinedDataset)?.name || unjoinedDataset
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.theDataSetHasBeenBoundToCellsButIsNotConfigured', { datasetName: datasetName })
      },
    })
    return false
  }
  return true
}

function orderableDatasetJoinIds(joins: ReportDatasetJoin[], primaryId: string) {
  if (!primaryId) return null
  const connected = new Set<string>([primaryId])
  const remaining = [...joins]
  while (remaining.length) {
    const index = remaining.findIndex((join) => {
      const leftConnected = connected.has(join.left_dataset_id)
      const rightConnected = connected.has(join.right_dataset_id)
      return leftConnected !== rightConnected
    })
    if (index < 0) return null
    const [join] = remaining.splice(index, 1)
    if (!join) return null
    connected.add(join.left_dataset_id)
    connected.add(join.right_dataset_id)
  }
  return connected
}

function validateSqlDatasets() {
  const invalid = datasets.value.find(
    (dataset) => dataset.type === 'sql' && (!dataset.sql?.trim() || !dataset.fields?.length),
  )
  if (!invalid) return true
  $q.notify({
    type: 'warning',
    get message() {
      return t('ui.sqlDataSetNeedsToFillInSqlAndParseFieldsFirst', { value1: invalid.name })
    },
  })
  return false
}

function validateBoundCells() {
  for (const cell of sheet.value.cells) {
    const binding = cell.binding
    if (!binding || binding.type === 'static') continue
    const label = `${reportColumnName(cell.col)}${cell.row}`
    if (binding.type === 'formula') {
      const formula = binding.formula?.trim() || cell.value?.trim() || ''
      if (!formula) {
        $q.notify({
          type: 'warning',
          get message() {
            return t('ui.theFormulaForCellIsEmpty', { label: label })
          },
        })
        return false
      }
      const formulaError = reportValidateFormula(formula)
      if (formulaError) {
        $q.notify({
          type: 'warning',
          message: t('ui.cellFormulaValidationFailed', {
            label,
            reason: formulaErrorMessage(formulaError),
          }),
        })
        return false
      }
      continue
    }
    if (!binding.dataset_id || !binding.field) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.cellIncompletelyBoundDataSetFields', { label: label })
        },
      })
      return false
    }
    const dataset = datasets.value.find((item) => item.id === binding.dataset_id)
    const field = dataset?.fields.find((item) => item.code === binding.field)
    if (!dataset || !field) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.cellBoundDataSetOrFieldNoLongerExists', { label: label })
        },
      })
      return false
    }
    if ((binding.type === 'sum' || binding.type === 'avg') && !isNumericReportField(field)) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.theNumericalAggregationCellShouldSelectANumericalField', {
            label: label,
          })
        },
      })
      return false
    }
  }
  return true
}

function validateParameters() {
  const seen = new Set<string>()
  for (const param of parameters.value) {
    if (!param.label?.trim() || !param.dataset_id || !param.field) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.checkTheReportParametersTheParameterNameTheDataSet')
        },
      })
      return false
    }
    const key = `${param.dataset_id}:${param.field}:${param.operator}`
    if (seen.has(key)) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.parameterRepeatedlyBindsTheSameFieldAndMatches', { value1: param.label })
        },
      })
      return false
    }
    seen.add(key)
    const dataset = datasets.value.find((item) => item.id === param.dataset_id)
    const field = dataset?.fields.find((item) => item.code === param.field)
    if (!dataset || !field) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.theDataSetOrFieldBoundByTheParameterNoLongerExists', { value1: param.label })
        },
      })
      return false
    }
    const compatibilityError = parameterCompatibilityError(param.type, param.operator, field)
    if (compatibilityError) {
      $q.notify({
        type: 'warning',
        get message() {
          return t('ui.parameter', { value1: param.label, compatibilityError: compatibilityError })
        },
      })
      return false
    }
  }
  return true
}

function validateRuntimeSettings() {
  if (form.runtime_display !== 'paged') return true
  const pageSize = Number(form.runtime_page_size || 0)
  if (!Number.isInteger(pageSize) || pageSize < 1 || pageSize > 500) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.pageBreakShowsTheNumberOfPagesPerPageMust')
      },
    })
    return false
  }
  return true
}

function parameterCompatibilityError(
  type: ReportParameterType,
  operator: ReportParameterOperator,
  field?: ReportField,
) {
  if (!field) return t('ui.pleaseSelectAValidField')
  if (type === 'date_range' && operator !== 'between') {
    return t('ui.dateRangeParametersMustBeMatchedByAnInterArea')
  }
  if (operator === 'between' && type !== 'date_range') {
    return t('ui.matchBetweenFieldsWithDateRangeParameters')
  }
  if (operator === 'like' && isNumericReportField(field)) {
    return t('ui.numericalFieldsCannotBeMatchedByIncludingThemReplaceThem')
  }
  if (type === 'number' && operator === 'like') {
    return t('ui.numericalParametersCannotBeMatchedWithInclusion')
  }
  return ''
}

function isNumericReportField(field: ReportField) {
  const type = String(field.type || '').toLowerCase()
  return (
    field.role === 'metric' ||
    [
      'number',
      'numeric',
      'decimal',
      'float',
      'double',
      'real',
      'int',
      'bigint',
      'smallint',
      'serial',
    ].some((item) => type.includes(item))
  )
}

function goBack() {
  void router.push({ name: 'report_manage' })
}
</script>

<style scoped lang="scss">
.report-designer-fullscreen {
  position: fixed;
  inset: 0;
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
  grid-template-columns: 52px minmax(760px, 1fr) 360px;
  overflow: auto;
}

.designer-palette {
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 6px;
  border-right: 1px solid #dfe5f2;
  background: #fff;
}

.designer-palette .q-btn {
  width: 40px;
  height: 40px;
  min-height: 40px;
  border-radius: 6px;
  color: #64748b;
}

.designer-palette .q-btn.active {
  color: var(--q-primary);
  background: rgba(115, 103, 240, 0.1);
  box-shadow: inset 3px 0 0 var(--q-primary);
}

.designer-palette .q-separator {
  width: 28px;
  margin: 2px 0;
}

.designer-side-panel {
  min-height: 0;
  display: grid;
  grid-template-rows: 44px 1px minmax(0, 1fr);
  border-left: 1px solid #dfe5f2;
  background: #fbfcff;
  overflow: hidden;
}

.designer-canvas-loading {
  display: grid;
  min-width: 0;
  min-height: 0;
  place-items: center;
  background: #fff;
}

.side-panel-tabs {
  min-height: 44px;
  background: #fff;
}

.side-panel-tabs :deep(.q-tab) {
  min-height: 44px;
  padding: 0 12px;
}

.side-panel-tabs :deep(.q-tab__content) {
  min-width: 0;
  gap: 6px;
}

.side-panel-tabs :deep(.q-icon) {
  font-size: 19px;
}

.side-panel-tabs :deep(.q-tab__label) {
  font-size: 13px;
  font-weight: 700;
}

.designer-side-panels {
  min-height: 0;
  background: transparent;
}

.designer-side-panels :deep(> .q-panel) {
  height: 100%;
  min-height: 0;
}

.designer-side-panels :deep(.side-panel-page) {
  height: 100%;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

.designer-side-panels :deep(.resource-panel),
.designer-side-panels :deep(.inspector-panel) {
  height: 100%;
  border: 0;
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
    grid-template-columns: 48px minmax(720px, 1fr) 320px;
  }
}
</style>
