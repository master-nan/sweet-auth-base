<template>
  <section class="canvas-panel">
    <div class="report-sheet-bar">
      <div class="report-sheet-bar__context">
        <q-chip dense square outline color="primary" icon="grid_on">
          {{ activeCellLabel }}
        </q-chip>
        <q-chip
          v-if="activeBoundCell"
          dense
          square
          color="primary"
          text-color="white"
          icon="drag_indicator"
          draggable="true"
          class="bound-cell-drag-handle"
          @dragstart="handleBoundCellDragStart"
          @dragend="handleBoundCellDragEnd"
        >
          {{ activeBoundCell.value }}
          <q-tooltip>{{ t('ui.dragTheBoundFieldToAnotherCell') }}</q-tooltip>
        </q-chip>
        <q-btn-dropdown
          flat
          dense
          no-caps
          class="row-type-control"
          :icon="activeRowTypeMeta.icon"
          :label="activeRowTypeMeta.label"
          :color="activeRowType === 'normal' ? 'dark' : 'primary'"
        >
          <q-list dense class="row-type-menu">
            <q-item
              v-for="option in rowTypeOptions"
              :key="option.value"
              v-close-popup
              clickable
              :active="activeRowType === option.value"
              active-class="text-primary bg-grey-2"
              @click="setActiveRowType(option.value)"
            >
              <q-item-section avatar>
                <q-icon :name="option.icon" />
              </q-item-section>
              <q-item-section>{{ option.label }}</q-item-section>
              <q-item-section v-if="activeRowType === option.value" side>
                <q-icon name="check" color="primary" />
              </q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>
        <span class="report-sheet-bar__hint">{{ t('ui.reportRowTypeHint') }}</span>
      </div>

      <div v-if="datasetJoins.length" class="report-sheet-bar__joins">
        <q-chip
          v-for="join in datasetJoins"
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
    </div>

    <div
      class="univer-shell"
      @dragover.prevent="handleFieldDragOver"
      @dragleave="handleFieldDragLeave"
      @drop.prevent="handleFieldDrop"
    >
      <div ref="univerContainer" class="univer-container" />
      <div v-if="dropTarget" class="field-drop-tip" :style="dropTarget.style">
        {{ reportColumnName(dropTarget.col) }}{{ dropTarget.row }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import '@univerjs/preset-sheets-core/lib/index.css'

import {
  AUTO_FILL_APPLY_TYPE,
  IAutoFillService,
  UniverSheetsCorePreset,
  type FWorkbook,
} from '@univerjs/preset-sheets-core'
import enUS from '@univerjs/preset-sheets-core/locales/en-US'
import zhCN from '@univerjs/preset-sheets-core/locales/zh-CN'
import {
  createUniver,
  Direction,
  LifecycleStages,
  LocaleType,
  mergeLocales,
  type FUniver,
  type IDisposable,
  type IRange,
  type Univer,
} from '@univerjs/presets'
import { IRenderManagerService } from '@univerjs/engine-render'
import { ISheetSelectionRenderService } from '@univerjs/sheets-ui'
import { useQuasar } from 'quasar'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import type { ReportDataset, ReportDatasetJoin, ReportSheetConfig } from 'src/api/services/report'
import { reportColumnName, type ReportSheetRange } from 'src/modules/report/sheet'
import {
  copyReportUniverCellMetadata,
  getReportUniverFillSourceIndex,
  reportSheetToUniverSnapshot,
  univerSnapshotToReportSheet,
} from 'src/modules/report/univer'

type ReportRowType = 'normal' | 'detail' | 'groupSummary' | 'summary'

const props = defineProps<{
  sheet: ReportSheetConfig
  selectedCellId: string
  datasets: ReportDataset[]
  datasetJoins: ReportDatasetJoin[]
  fieldDragging: boolean
}>()

const emit = defineEmits<{
  'update:sheet': [value: ReportSheetConfig]
  selectCell: [row: number, col: number]
  selectRange: [value: ReportSheetRange]
  dropField: [row: number, col: number]
  moveBoundCell: [sourceRow: number, sourceCol: number, targetRow: number, targetCol: number]
  toggleSummaryRow: [row: number]
  toggleGroupSummaryRow: [row: number]
  toggleDetailRow: [row: number]
}>()

const { locale, t } = useI18n({ useScope: 'global' })
const $q = useQuasar()
const univerContainer = ref<HTMLElement | null>(null)
let univer: Univer | undefined
let univerAPI: FUniver | undefined
let workbook: FWorkbook | undefined
let selectionDisposable: IDisposable | undefined
let commandDisposable: IDisposable | undefined
let autoFillDisposable: IDisposable | undefined
let lifecycleDisposable: IDisposable | undefined
let syncTimer: number | undefined
let rebuilding = false
let applyingExternalSheet = false
let lastEmittedSignature = ''
let movingBindingCell: { row: number; col: number } | null = null

const dropTarget = ref<{
  row: number
  col: number
  style: { left: string; top: string }
} | null>(null)

const activeRow = computed(() => {
  const [row] = props.selectedCellId.split(':').map(Number)
  return Math.min(Math.max(row || 1, 1), props.sheet.rows)
})
const activeCellLabel = computed(() => {
  const [, rawColumn] = props.selectedCellId.split(':').map(Number)
  return `${reportColumnName(Math.max(rawColumn || 1, 1))}${activeRow.value}`
})
const activeBoundCell = computed(() => {
  const [row, col] = props.selectedCellId.split(':').map(Number)
  return props.sheet.cells.find(
    (cell) =>
      cell.row === row && cell.col === col && cell.binding && cell.binding.type !== 'static',
  )
})
const activeRowType = computed<ReportRowType>(() => {
  if (props.sheet.group_summary_rows?.includes(activeRow.value)) return 'groupSummary'
  if (props.sheet.summary_rows?.includes(activeRow.value)) return 'summary'
  if (props.sheet.detail_rows?.includes(activeRow.value)) return 'detail'
  return 'normal'
})
const rowTypeOptions = computed<Array<{ value: ReportRowType; label: string; icon: string }>>(
  () => [
    { value: 'normal', label: t('ui.normalRow'), icon: 'horizontal_rule' },
    { value: 'detail', label: t('ui.detailRow'), icon: 'south' },
    { value: 'groupSummary', label: t('ui.groupSummaryRow'), icon: 'functions' },
    { value: 'summary', label: t('ui.summaryRow'), icon: 'calculate' },
  ],
)
const activeRowTypeMeta = computed(
  () => rowTypeOptions.value.find((option) => option.value === activeRowType.value)!,
)

const sheetSignature = (value: ReportSheetConfig) => JSON.stringify(value)

function publishSelection(selections: IRange[]) {
  const selection = selections.at(-1)
  if (!selection) return
  const row = selection.startRow + 1
  const col = selection.startColumn + 1
  emit('selectCell', row, col)
  emit('selectRange', {
    startRow: row,
    startCol: col,
    endRow: selection.endRow + 1,
    endCol: selection.endColumn + 1,
  })
}

function createWorkbook() {
  if (!univerAPI) return
  if (workbook) univerAPI.disposeUnit(workbook.getId())

  rebuilding = true
  workbook = univerAPI.createWorkbook(reportSheetToUniverSnapshot(props.sheet))

  const [row, col] = props.selectedCellId.split(':').map(Number)
  if (Number.isInteger(row) && Number.isInteger(col)) {
    const activeSheet = workbook.getActiveSheet()
    activeSheet.setActiveRange(activeSheet.getRange(row! - 1, col! - 1))
  }
  window.setTimeout(() => {
    rebuilding = false
  }, 0)
}

function scheduleSheetSync() {
  if (rebuilding || applyingExternalSheet || !workbook) return
  if (syncTimer !== undefined) window.clearTimeout(syncTimer)
  syncTimer = window.setTimeout(() => {
    if (!workbook || rebuilding) return
    const next = univerSnapshotToReportSheet(workbook.save())
    next.active_cell = props.selectedCellId
    lastEmittedSignature = sheetSignature(next)
    emit('update:sheet', next)
  }, 0)
}

function applyExternalSheet(value: ReportSheetConfig) {
  if (!workbook || !univerAPI) return
  const current = workbook.save()
  const target = reportSheetToUniverSnapshot(value)
  const currentWorksheet = current.sheets[current.sheetOrder[0] || 'report-sheet'] || {}
  const targetWorksheet = target.sheets[target.sheetOrder[0] || 'report-sheet'] || {}
  const rowHeightSignature = (rows: typeof currentWorksheet.rowData) =>
    JSON.stringify(
      Object.fromEntries(Object.entries(rows || {}).map(([row, data]) => [row, data?.h ?? null])),
    )
  const structureChanged =
    currentWorksheet.rowCount !== targetWorksheet.rowCount ||
    currentWorksheet.columnCount !== targetWorksheet.columnCount ||
    JSON.stringify(currentWorksheet.mergeData || []) !==
      JSON.stringify(targetWorksheet.mergeData || []) ||
    rowHeightSignature(currentWorksheet.rowData) !== rowHeightSignature(targetWorksheet.rowData) ||
    JSON.stringify(currentWorksheet.columnData || {}) !==
      JSON.stringify(targetWorksheet.columnData || {})
  if (structureChanged) {
    createWorkbook()
    return
  }

  applyingExternalSheet = true
  try {
    const worksheet = workbook.getActiveSheet()
    const cellKeys = new Set<string>()
    Object.entries(currentWorksheet.cellData || {}).forEach(([row, columns]) => {
      Object.keys(columns || {}).forEach((column) => cellKeys.add(`${row}:${column}`))
    })
    Object.entries(targetWorksheet.cellData || {}).forEach(([row, columns]) => {
      Object.keys(columns || {}).forEach((column) => cellKeys.add(`${row}:${column}`))
    })
    cellKeys.forEach((key) => {
      const [row, column] = key.split(':').map(Number)
      const currentCell = currentWorksheet.cellData?.[row!]?.[column!]
      const targetCell = targetWorksheet.cellData?.[row!]?.[column!]
      if (JSON.stringify(currentCell || null) === JSON.stringify(targetCell || null)) return
      worksheet
        .getRange(row!, column!)
        .setValueForCell(targetCell || { v: null, f: null, s: null, custom: null })
    })
    for (let row = 0; row < Number(targetWorksheet.rowCount || 0); row += 1) {
      const currentCustom = currentWorksheet.rowData?.[row]?.custom
      const targetCustom = targetWorksheet.rowData?.[row]?.custom
      if (JSON.stringify(currentCustom || null) !== JSON.stringify(targetCustom || null)) {
        worksheet.setRowCustomMetadata(row, targetCustom)
      }
    }
    worksheet.setCustomMetadata(targetWorksheet.custom)
  } finally {
    applyingExternalSheet = false
  }
}

function registerReportMetadataHooks() {
  if (!univer) return
  const injector = univer.__getInjector()
  autoFillDisposable = injector.get(IAutoFillService).addHook({
    id: 'sweet-report-binding-fill',
    onBeforeSubmit(location, direction, applyType, cellValue) {
      if (applyType === AUTO_FILL_APPLY_TYPE.ONLY_FORMAT || !univerAPI) return
      const target = univerAPI.getSheetTarget(location.unitId, location.subUnitId)
      if (!target) return
      const sourceRows = location.source.rows
      const sourceColumns = location.source.cols
      const reverseRows = direction === Direction.UP
      const reverseColumns = direction === Direction.LEFT
      location.target.rows.forEach((row, rowIndex) => {
        location.target.cols.forEach((column, columnIndex) => {
          const targetCell = cellValue[row]?.[column]
          if (!targetCell) return
          const sourceRow = getReportUniverFillSourceIndex(sourceRows, rowIndex, reverseRows)
          const sourceColumn = getReportUniverFillSourceIndex(
            sourceColumns,
            columnIndex,
            reverseColumns,
          )
          if (sourceRow === undefined || sourceColumn === undefined) return
          const sourceCell = target.worksheet.getRange(sourceRow, sourceColumn).getCellData()
          cellValue[row]![column] = copyReportUniverCellMetadata(sourceCell, targetCell)
        })
      })
    },
  })
}

function initializeWorkbook() {
  if (!univerAPI || workbook) return
  createWorkbook()
  commandDisposable = univerAPI.addEvent(univerAPI.Event.CommandExecuted, scheduleSheetSync)
  univerAPI.toggleDarkMode($q.dark.isActive)
}

function initializeRenderedFeatures() {
  if (!univerAPI) return
  if (!autoFillDisposable) registerReportMetadataHooks()
  if (!selectionDisposable) {
    selectionDisposable = univerAPI.addEvent(
      univerAPI.Event.SelectionChanged,
      ({ workbook: selectedWorkbook, selections }) => {
        if (selectedWorkbook.getId() !== workbook?.getId()) return
        publishSelection(selections)
      },
    )
  }
}

function setActiveRowType(type: ReportRowType) {
  const current = activeRowType.value
  if (current === type) return
  if (type === 'normal') {
    if (current === 'detail') emit('toggleDetailRow', activeRow.value)
    else if (current === 'groupSummary') emit('toggleGroupSummaryRow', activeRow.value)
    else if (current === 'summary') emit('toggleSummaryRow', activeRow.value)
    return
  }
  if (type === 'detail') emit('toggleDetailRow', activeRow.value)
  else if (type === 'groupSummary') emit('toggleGroupSummaryRow', activeRow.value)
  else emit('toggleSummaryRow', activeRow.value)
}

function eventCell(event: DragEvent) {
  if (!univer || !workbook || !univerContainer.value) return null
  const canvases = [...univerContainer.value.querySelectorAll('canvas')]
    .map((canvas) => ({ canvas, rect: canvas.getBoundingClientRect() }))
    .filter(
      ({ rect }) =>
        rect.width > 0 &&
        rect.height > 0 &&
        event.clientX >= rect.left &&
        event.clientX <= rect.right &&
        event.clientY >= rect.top &&
        event.clientY <= rect.bottom,
    )
    .sort(
      (left, right) => right.rect.width * right.rect.height - left.rect.width * left.rect.height,
    )
  const canvasRect = canvases[0]?.rect
  if (!canvasRect) return null
  const render = univer.__getInjector().get(IRenderManagerService).getRenderById(workbook.getId())
  const cell = render
    ?.with(ISheetSelectionRenderService)
    .getCellWithCoordByOffset(event.clientX - canvasRect.left, event.clientY - canvasRect.top)
  if (
    !cell ||
    cell.actualRow < 0 ||
    cell.actualColumn < 0 ||
    cell.actualRow >= props.sheet.rows ||
    cell.actualColumn >= props.sheet.cols
  ) {
    return null
  }
  return { row: cell.actualRow + 1, col: cell.actualColumn + 1 }
}

function selectDropTarget(row: number, col: number) {
  const worksheet = workbook?.getActiveSheet()
  worksheet?.setActiveRange(worksheet.getRange(row - 1, col - 1))
  emit('selectCell', row, col)
  emit('selectRange', { startRow: row, startCol: col, endRow: row, endCol: col })
}

function handleFieldDragOver(event: DragEvent) {
  if (!props.fieldDragging && !movingBindingCell) return
  const cell = eventCell(event)
  if (!cell) {
    dropTarget.value = null
    return
  }
  const shellRect = univerContainer.value?.getBoundingClientRect()
  dropTarget.value = {
    ...cell,
    style: {
      left: `${Math.max(8, event.clientX - (shellRect?.left || 0) + 12)}px`,
      top: `${Math.max(8, event.clientY - (shellRect?.top || 0) + 12)}px`,
    },
  }
  if (props.selectedCellId !== `${cell.row}:${cell.col}`) selectDropTarget(cell.row, cell.col)
}

function handleFieldDragLeave(event: DragEvent) {
  const nextTarget = event.relatedTarget
  if (nextTarget instanceof Node && univerContainer.value?.contains(nextTarget)) return
  dropTarget.value = null
}

function handleFieldDrop(event: DragEvent) {
  const target = eventCell(event) || dropTarget.value
  if (!target) return
  if (movingBindingCell) {
    emit('moveBoundCell', movingBindingCell.row, movingBindingCell.col, target.row, target.col)
  } else {
    emit('dropField', target.row, target.col)
  }
  movingBindingCell = null
  dropTarget.value = null
}

function handleBoundCellDragStart(event: DragEvent) {
  const cell = activeBoundCell.value
  if (!cell) return
  movingBindingCell = { row: cell.row, col: cell.col }
  event.dataTransfer?.setData('application/x-sweet-report-cell', `${cell.row}:${cell.col}`)
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
}

function handleBoundCellDragEnd() {
  movingBindingCell = null
  dropTarget.value = null
}

function joinLabel(join: ReportDatasetJoin) {
  const left = props.datasets.find((item) => item.id === join.left_dataset_id)
  const right = props.datasets.find((item) => item.id === join.right_dataset_id)
  const condition = join.conditions[0]
  if (!condition)
    return `${left?.name || join.left_dataset_id} ${join.join_type.toUpperCase()} ${right?.name || join.right_dataset_id}`
  const suffix = join.conditions.length > 1 ? ` +${join.conditions.length - 1}` : ''
  return `${left?.name || join.left_dataset_id}.${condition.left_field} ${join.join_type.toUpperCase()} ${right?.name || join.right_dataset_id}.${condition.right_field}${suffix}`
}

onMounted(async () => {
  await nextTick()
  if (!univerContainer.value) return
  const created = createUniver({
    locale: locale.value === 'en-US' ? LocaleType.EN_US : LocaleType.ZH_CN,
    locales: {
      [LocaleType.ZH_CN]: mergeLocales(zhCN),
      [LocaleType.EN_US]: mergeLocales(enUS),
    },
    presets: [
      UniverSheetsCorePreset({
        container: univerContainer.value,
        header: true,
        toolbar: true,
        formulaBar: true,
        footer: {
          sheetBar: false,
          statisticBar: true,
          menus: true,
          zoomSlider: true,
        },
      }),
    ],
  })
  univer = created.univer
  univerAPI = created.univerAPI
  initializeWorkbook()
  if (univerAPI.getCurrentLifecycleStage() >= LifecycleStages.Rendered) {
    initializeRenderedFeatures()
    return
  }
  lifecycleDisposable = univerAPI.addEvent(univerAPI.Event.LifeCycleChanged, ({ stage }) => {
    if (stage < LifecycleStages.Rendered) return
    lifecycleDisposable?.dispose()
    lifecycleDisposable = undefined
    initializeRenderedFeatures()
  })
})

watch(
  () => props.sheet,
  (value) => {
    const signature = sheetSignature(value)
    if (signature === lastEmittedSignature) return
    applyExternalSheet(value)
  },
  { deep: true },
)

watch(
  () => $q.dark.isActive,
  (dark) => univerAPI?.toggleDarkMode(dark),
)

watch(locale, (value) => {
  univerAPI?.setLocale(value === 'en-US' ? LocaleType.EN_US : LocaleType.ZH_CN)
})

onBeforeUnmount(() => {
  if (syncTimer !== undefined) window.clearTimeout(syncTimer)
  selectionDisposable?.dispose()
  commandDisposable?.dispose()
  autoFillDisposable?.dispose()
  lifecycleDisposable?.dispose()
  univer?.dispose()
  workbook = undefined
  univerAPI = undefined
  univer = undefined
})
</script>

<style scoped lang="scss">
.canvas-panel {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 42px minmax(0, 1fr);
  background: #fff;
}

.report-sheet-bar {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 5px 10px;
  border-bottom: 1px solid #dfe5f2;
  background: #fbfcff;
  overflow: hidden;
}

.report-sheet-bar__context,
.report-sheet-bar__joins {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.report-sheet-bar__joins {
  overflow-x: auto;
}

.report-sheet-bar__joins .q-chip {
  flex: 0 0 auto;
  max-width: 260px;
}

.report-sheet-bar__hint {
  color: #8290a8;
  font-size: 12px;
  white-space: nowrap;
}

.row-type-control {
  min-width: 118px;
}

.row-type-menu {
  min-width: 180px;
}

.univer-shell,
.univer-container {
  position: relative;
  min-width: 0;
  min-height: 0;
  width: 100%;
  height: 100%;
}

.field-drop-tip {
  position: absolute;
  z-index: 21;
  padding: 3px 7px;
  border: 1px solid var(--q-primary);
  border-radius: 4px;
  background: var(--q-primary);
  color: #fff;
  box-shadow: 0 4px 12px rgba(23, 32, 51, 0.16);
  font-size: 11px;
  font-weight: 700;
  pointer-events: none;
}

.bound-cell-drag-handle {
  max-width: 220px;
  cursor: grab;
}

.bound-cell-drag-handle:active {
  cursor: grabbing;
}

:deep(.univer-app) {
  width: 100%;
  height: 100%;
}

body.body--dark .report-sheet-bar {
  border-color: #30384a;
  background: #1f2533;
}
</style>
