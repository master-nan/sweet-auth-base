<template>
  <section class="canvas-panel">
    <div class="canvas-toolbar">
      <div class="toolbar-group">
        <q-btn flat dense icon="undo" :disable="!canUndo" @click="$emit('undo')">
          <q-tooltip>{{ t('ui.undo') }}</q-tooltip>
        </q-btn>
        <q-btn flat dense icon="redo" :disable="!canRedo" @click="$emit('redo')">
          <q-tooltip>{{ t('ui.redo') }}</q-tooltip>
        </q-btn>
        <q-separator vertical />
        <q-btn
          flat
          dense
          icon="format_bold"
          :color="activeBold ? 'primary' : 'dark'"
          @click="$emit('toggleBold')"
        >
          <q-tooltip>{{ t('ui.bold') }}</q-tooltip>
        </q-btn>
        <q-btn
          flat
          dense
          icon="format_italic"
          :color="activeItalic ? 'primary' : 'dark'"
          @click="$emit('toggleItalic')"
        >
          <q-tooltip>{{ t('ui.italic') }}</q-tooltip>
        </q-btn>
        <q-btn flat dense icon="format_color_text">
          <span class="color-swatch" :style="{ background: activeTextColor }" />
          <q-tooltip>{{ t('ui.textColor') }}</q-tooltip>
          <q-popup-proxy>
            <q-color
              :model-value="activeTextColor"
              no-header
              no-footer
              default-view="palette"
              @update:model-value="setTextColor"
            />
          </q-popup-proxy>
        </q-btn>
        <q-btn flat dense icon="format_color_fill">
          <span class="color-swatch" :style="{ background: activeBackgroundColor }" />
          <q-tooltip>{{ t('ui.cellBackground') }}</q-tooltip>
          <q-popup-proxy>
            <q-color
              :model-value="activeBackgroundColor"
              no-header
              no-footer
              default-view="palette"
              @update:model-value="setBackgroundColor"
            />
          </q-popup-proxy>
        </q-btn>
        <q-btn flat dense icon="format_align_left" @click="$emit('setAlign', 'left')">
          <q-tooltip>{{ t('ui.alignLeft') }}</q-tooltip>
        </q-btn>
        <q-btn flat dense icon="format_align_center" @click="$emit('setAlign', 'center')">
          <q-tooltip>{{ t('ui.alignCenter') }}</q-tooltip>
        </q-btn>
        <q-btn flat dense icon="format_align_right" @click="$emit('setAlign', 'right')">
          <q-tooltip>{{ t('ui.alignRight') }}</q-tooltip>
        </q-btn>
        <q-separator vertical />
        <q-btn
          flat
          dense
          icon="call_merge"
          :label="t('ui.mergeSelections')"
          :disable="!hasRangeSelection"
          @click="$emit('mergeSelection')"
        />
        <q-btn
          flat
          dense
          icon="splitscreen"
          :label="t('ui.unmerger')"
          @click="$emit('unmergeActiveCell')"
        />
        <q-btn
          flat
          dense
          icon="backspace"
          :label="t('ui.clearSelection')"
          @click="$emit('clearSelection')"
        />
      </div>
      <div class="toolbar-group">
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
        <q-separator v-if="datasetJoins.length" vertical />
        <q-btn
          dense
          outline
          size="sm"
          color="primary"
          icon="add"
          :label="t('ui.okay')"
          @click="$emit('addRow')"
        />
        <q-btn
          dense
          outline
          size="sm"
          color="primary"
          icon="add"
          :label="t('ui.column')"
          @click="$emit('addCol')"
        />
        <q-btn flat round dense icon="zoom_out" @click="$emit('zoomOut')">
          <q-tooltip>{{ t('ui.zoomOut') }}</q-tooltip>
        </q-btn>
        <q-chip square outline color="primary">{{ Math.round(scale * 100) }}%</q-chip>
        <q-btn flat round dense icon="zoom_in" @click="$emit('zoomIn')">
          <q-tooltip>{{ t('ui.zoomIn') }}</q-tooltip>
        </q-btn>
      </div>
    </div>

    <div
      ref="sheetScrollRef"
      class="sheet-scroll"
      @scroll="closeContextMenu"
      @copy="handleCopy"
      @cut="handleCut"
      @paste="handlePaste"
    >
      <div class="sheet-grid" :style="gridStyle">
        <div class="sheet-corner" />
        <div
          v-for="header in columnHeaders"
          :key="header.key"
          class="sheet-col-head"
          :style="header.style"
        >
          {{ header.label }}
          <span
            class="sheet-col-resizer"
            :title="t('ui.resizeColumn')"
            @mousedown.stop.prevent="startColumnResize($event, header.col)"
          />
        </div>
        <template v-for="renderRow in renderRows" :key="renderRow.key">
          <div
            class="sheet-row-head"
            :class="{
              detail: renderRow.detail,
              summary: renderRow.summary,
              'group-summary': renderRow.groupSummary,
            }"
            :style="renderRow.headerStyle"
          >
            {{ renderRow.row }}
            <span
              class="sheet-row-resizer"
              :title="t('ui.resizeRow')"
              @mousedown.stop.prevent="startRowResize($event, renderRow.row)"
            />
            <q-tooltip v-if="renderRow.detail">{{ t('ui.linesRunOnALineByLineBasis') }}</q-tooltip>
            <q-tooltip v-else-if="renderRow.groupSummary">{{
              t('ui.groupSummaryRowDescription')
            }}</q-tooltip>
            <q-tooltip v-else-if="renderRow.summary">{{
              t('ui.summarizeRowsAggregatingCurrentDataOnRunningTime')
            }}</q-tooltip>
          </div>
          <template v-for="renderCell in renderRow.cells" :key="renderCell.key">
            <div
              class="sheet-cell"
              :class="{
                active: renderCell.active,
                selected: renderCell.selected,
                bound: renderCell.bound,
                detail: renderCell.detail,
                summary: renderCell.summary,
                'group-summary': renderCell.groupSummary,
                'drop-target': dragOverCellId === renderCell.id,
                'fill-target': renderCell.fillTarget,
              }"
              :style="renderCell.style"
              :data-cell-id="renderCell.id"
              :data-drop-label="t('ui.dropHere')"
              role="button"
              tabindex="0"
              @click="handleCellClick(renderCell.row, renderCell.col, $event)"
              @dblclick.stop="startEdit(renderCell.row, renderCell.col)"
              @keydown="handleCellKeydown($event, renderCell.row, renderCell.col)"
              @dragenter.prevent="handleDragEnter(renderCell.id)"
              @dragover.prevent="handleDragEnter(renderCell.id)"
              @dragleave="handleDragLeave(renderCell.id)"
              @drop.prevent="handleDrop(renderCell.row, renderCell.col)"
              @contextmenu.prevent.stop="openContextMenu(renderCell.row, renderCell.col, $event)"
            >
              <input
                v-if="editingCellId === renderCell.id"
                v-model="editingValue"
                autofocus
                class="sheet-cell__editor"
                @click.stop
                @keydown.enter.stop.prevent="commitEdit(renderCell.row, renderCell.col)"
                @keydown.esc.stop.prevent="cancelEdit"
                @blur="commitEdit(renderCell.row, renderCell.col)"
              />
              <span v-else>{{ renderCell.value }}</span>
              <span
                v-if="renderCell.active && renderCell.fillable && editingCellId !== renderCell.id"
                class="sheet-fill-handle"
                :title="t('ui.dragFillCells')"
                @mousedown.stop.prevent="startFillDrag(renderCell.row, renderCell.col)"
              />
            </div>
          </template>
        </template>
      </div>

      <div
        v-if="contextMenu.visible"
        class="sheet-context-menu"
        :style="{ left: `${contextMenu.left}px`, top: `${contextMenu.top}px` }"
        @mousedown.stop
        @click.stop
      >
        <button type="button" @click="runContextAction('edit')">
          <q-icon name="edit" /> {{ t('ui.editText') }}
        </button>
        <button type="button" @click="runContextAction('clear')">
          <q-icon name="backspace" /> {{ t('ui.clearCells') }}
        </button>
        <button
          type="button"
          :disabled="!hasRangeSelection"
          @click="runContextAction('mergeSelection')"
        >
          <q-icon name="call_merge" /> {{ t('ui.mergeSelections') }}
        </button>
        <button type="button" @click="runContextAction('mergeRight')">
          <q-icon name="call_merge" /> {{ t('ui.mergeRight') }}
        </button>
        <button type="button" @click="runContextAction('unmerge')">
          <q-icon name="splitscreen" /> {{ t('ui.unmerger') }}
        </button>
        <div class="sheet-context-menu__separator" />
        <button type="button" @click="runContextAction('insertRow')">
          <q-icon name="table_rows" /> {{ t('ui.insertRowBelow') }}
        </button>
        <button type="button" @click="runContextAction('insertCol')">
          <q-icon name="view_column" /> {{ t('ui.insertColumnsRight') }}
        </button>
        <button type="button" @click="runContextAction('deleteRow')">
          <q-icon name="delete_sweep" /> {{ t('ui.deleteCurrentRow') }}
        </button>
        <button type="button" @click="runContextAction('deleteCol')">
          <q-icon name="delete_sweep" /> {{ t('ui.deleteCurrentColumn') }}
        </button>
        <div class="sheet-context-menu__separator" />
        <button type="button" @click="runContextAction('summary')">
          <q-icon name="functions" />
          {{ isSummaryRow(contextMenu.row) ? t('ui.ungroupRows') : t('ui.setAsSummaryRows') }}
        </button>
        <button type="button" @click="runContextAction('groupSummary')">
          <q-icon name="functions" />
          {{
            isGroupSummaryRow(contextMenu.row)
              ? t('ui.changeToGlobalSummaryRow')
              : t('ui.setAsGroupSummaryRow')
          }}
        </button>
        <button type="button" @click="runContextAction('detail')">
          <q-icon name="view_stream" />
          {{ isDetailRow(contextMenu.row) ? t('ui.cancelLines') : t('ui.setLines') }}
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import type {
  ReportDataset,
  ReportDatasetJoin,
  ReportSheetCell,
  ReportSheetConfig,
} from 'src/api/services/report'
import {
  reportCellStyle,
  reportColumnName,
  reportNormalizeSheetRange,
  reportCellId,
  reportSheetClipboardMatrix,
  reportSheetCellSpan,
  type ReportSheetClipboardCell,
  type ReportSheetRange,
} from 'src/modules/report/sheet'

const { t } = useI18n({ useScope: 'global' })
const reportClipboardMime = 'application/x-sweet-report-cells'

const props = defineProps<{
  sheet: ReportSheetConfig
  selectedCellId: string
  selectionRange: ReportSheetRange | null
  canUndo: boolean
  canRedo: boolean
  activeBold: boolean
  activeItalic: boolean
  activeTextColor: string
  activeBackgroundColor: string
  scale: number
  datasets: ReportDataset[]
  datasetJoins: ReportDatasetJoin[]
  fieldDragging: boolean
}>()

const emit = defineEmits<{
  undo: []
  redo: []
  toggleBold: []
  toggleItalic: []
  setTextColor: [value: string]
  setBackgroundColor: [value: string]
  setAlign: [value: 'left' | 'center' | 'right']
  mergeRight: []
  mergeSelection: []
  unmergeActiveCell: []
  clearActiveCell: []
  clearSelection: []
  addRow: []
  addCol: []
  selectCell: [row: number, col: number]
  selectRange: [row: number, col: number]
  dropField: [row: number, col: number]
  updateCellValue: [row: number, col: number, value: string]
  clearCell: [row: number, col: number]
  mergeCellRight: [row: number, col: number]
  unmergeCell: [row: number, col: number]
  insertRowAfter: [row: number]
  insertColAfter: [col: number]
  deleteRow: [row: number]
  deleteCol: [col: number]
  pasteCells: [matrix: ReportSheetClipboardCell[][]]
  fillCells: [sourceRow: number, sourceCol: number, targetRow: number, targetCol: number]
  resizeColumn: [col: number, width: number]
  resizeRow: [row: number, height: number]
  toggleSummaryRow: [row: number]
  toggleGroupSummaryRow: [row: number]
  toggleDetailRow: [row: number]
  zoomIn: []
  zoomOut: []
}>()

const editingCellId = ref('')
const editingValue = ref('')
const sheetScrollRef = ref<HTMLElement | null>(null)
const dragOverCellId = ref('')
const contextMenu = reactive({
  visible: false,
  row: 1,
  col: 1,
  left: 0,
  top: 0,
})
const resizeState = reactive<{
  axis: 'column' | 'row' | null
  index: number
  startPointer: number
  startSize: number
  currentSize: number
}>({
  axis: null,
  index: 0,
  startPointer: 0,
  startSize: 0,
  currentSize: 0,
})
const fillState = reactive({
  active: false,
  sourceRow: 0,
  sourceCol: 0,
  targetRow: 0,
  targetCol: 0,
})
const hasRangeSelection = computed(() => {
  if (!props.selectionRange) return false
  const bounds = reportNormalizeSheetRange(props.selectionRange)
  return bounds.maxRow > bounds.minRow || bounds.maxCol > bounds.minCol
})

const summaryRows = computed(() => new Set(props.sheet.summary_rows || []))
const groupSummaryRows = computed(() => new Set(props.sheet.group_summary_rows || []))
const detailRows = computed(() => new Set(props.sheet.detail_rows || []))

const selectionBounds = computed(() =>
  props.selectionRange ? reportNormalizeSheetRange(props.selectionRange) : null,
)
const gridStyle = computed(() => ({
  gridTemplateColumns: [
    '42px',
    ...Array.from({ length: props.sheet.cols }, (_, index) => `${columnWidth(index + 1)}px`),
  ].join(' '),
  gridTemplateRows: [
    '32px',
    ...Array.from({ length: props.sheet.rows }, (_, index) => `${rowHeight(index + 1)}px`),
  ].join(' '),
  zoom: props.scale,
}))

const cellMap = computed(() => {
  const map = new Map<string, ReportSheetCell>()
  props.sheet.cells.forEach((cell) => {
    map.set(cellKey(cell.row, cell.col), cell)
  })
  return map
})

const coveredCells = computed(() => {
  const covered = new Set<string>()
  props.sheet.cells.forEach((cell) => {
    const { rowspan, colspan } = reportSheetCellSpan(cell, {
      maxRow: props.sheet.rows,
      maxCol: props.sheet.cols,
    })
    if (rowspan === 1 && colspan === 1) return
    for (let row = cell.row; row < cell.row + rowspan; row += 1) {
      for (let col = cell.col; col < cell.col + colspan; col += 1) {
        if (row === cell.row && col === cell.col) continue
        covered.add(cellKey(row, col))
      }
    }
  })
  return covered
})

const columnHeaders = computed(() =>
  Array.from({ length: props.sheet.cols }, (_, index) => {
    const col = index + 1
    return {
      key: `head-${col}`,
      col,
      label: reportColumnName(col),
      style: {
        gridColumn: col + 1,
        gridRow: 1,
      },
    }
  }),
)

const renderRows = computed(() =>
  Array.from({ length: props.sheet.rows }, (_, index) => {
    const row = index + 1
    const summary = summaryRows.value.has(row)
    const groupSummary = groupSummaryRows.value.has(row)
    const detail = detailRows.value.has(row)
    return {
      key: `row-${row}`,
      row,
      detail,
      summary,
      groupSummary,
      headerStyle: {
        gridColumn: 1,
        gridRow: row + 1,
      },
      cells: renderCellsForRow(row, detail, summary, groupSummary),
    }
  }),
)

function renderCellsForRow(row: number, detail: boolean, summary: boolean, groupSummary: boolean) {
  const cells = []
  for (let col = 1; col <= props.sheet.cols; col += 1) {
    if (coveredCells.value.has(cellKey(row, col))) continue
    const cell = cellAt(row, col)
    const { rowspan, colspan } = reportSheetCellSpan(cell, {
      maxRow: props.sheet.rows,
      maxCol: props.sheet.cols,
    })
    cells.push({
      key: cell.id,
      id: cell.id,
      row,
      col,
      value:
        cell.binding?.type === 'formula'
          ? normalizeFormulaDisplay(cell.binding.formula || cell.value)
          : cell.value || '',
      active: props.selectedCellId === cell.id,
      selected: isSelectedCell(row, col),
      bound: Boolean(cell.binding?.field),
      fillable: rowspan === 1 && colspan === 1,
      fillTarget: fillState.active && row === fillState.targetRow && col === fillState.targetCol,
      detail,
      summary,
      groupSummary,
      style: {
        ...reportCellStyle(cell),
        gridColumn: `${cell.col + 1} / span ${colspan}`,
        gridRow: `${cell.row + 1} / span ${rowspan}`,
      },
    })
  }
  return cells
}

function cellAt(row: number, col: number): ReportSheetCell {
  return (
    cellMap.value.get(cellKey(row, col)) || {
      id: reportCellId(row, col),
      row,
      col,
      value: '',
    }
  )
}

function cellKey(row: number, col: number) {
  return `${row}:${col}`
}

function normalizeFormulaDisplay(value: string) {
  const formula = value.trim()
  if (!formula) return ''
  return formula.startsWith('=') ? formula : `=${formula}`
}

function isSelectedCell(row: number, col: number) {
  const bounds = selectionBounds.value
  return Boolean(
    bounds &&
      row >= bounds.minRow &&
      row <= bounds.maxRow &&
      col >= bounds.minCol &&
      col <= bounds.maxCol,
  )
}

function setTextColor(value: string | null) {
  if (value) emit('setTextColor', value)
}

function setBackgroundColor(value: string | null) {
  if (value) emit('setBackgroundColor', value)
}

function columnWidth(col: number) {
  if (resizeState.axis === 'column' && resizeState.index === col) return resizeState.currentSize
  return props.sheet.column_widths?.[String(col)] || 118
}

function rowHeight(row: number) {
  if (resizeState.axis === 'row' && resizeState.index === row) return resizeState.currentSize
  return props.sheet.row_heights?.[String(row)] || 42
}

function startColumnResize(event: MouseEvent, col: number) {
  startResize('column', col, event.clientX, columnWidth(col))
}

function startRowResize(event: MouseEvent, row: number) {
  startResize('row', row, event.clientY, rowHeight(row))
}

function startResize(
  axis: 'column' | 'row',
  index: number,
  startPointer: number,
  startSize: number,
) {
  resizeState.axis = axis
  resizeState.index = index
  resizeState.startPointer = startPointer
  resizeState.startSize = startSize
  resizeState.currentSize = startSize
  window.addEventListener('mousemove', handleResizeMove)
  window.addEventListener('mouseup', finishResize, { once: true })
}

function handleResizeMove(event: MouseEvent) {
  if (!resizeState.axis) return
  const pointer = resizeState.axis === 'column' ? event.clientX : event.clientY
  const delta = (pointer - resizeState.startPointer) / Math.max(props.scale, 0.1)
  const min = resizeState.axis === 'column' ? 64 : 28
  const max = resizeState.axis === 'column' ? 360 : 160
  resizeState.currentSize = Math.min(Math.max(Math.round(resizeState.startSize + delta), min), max)
}

function finishResize() {
  window.removeEventListener('mousemove', handleResizeMove)
  if (resizeState.axis === 'column') {
    emit('resizeColumn', resizeState.index, resizeState.currentSize)
  } else if (resizeState.axis === 'row') {
    emit('resizeRow', resizeState.index, resizeState.currentSize)
  }
  resizeState.axis = null
}

function handleCopy(event: ClipboardEvent) {
  if (editingCellId.value || !event.clipboardData || !selectionBounds.value) return
  const matrix = reportSheetClipboardMatrix(props.sheet, selectionBounds.value)
  event.clipboardData.setData(reportClipboardMime, JSON.stringify(matrix))
  event.clipboardData.setData(
    'text/plain',
    matrix.map((row) => row.map((cell) => cell.value || '').join('\t')).join('\n'),
  )
  event.preventDefault()
}

function handleCut(event: ClipboardEvent) {
  if (editingCellId.value) return
  handleCopy(event)
  if (event.defaultPrevented) emit('clearSelection')
}

function handlePaste(event: ClipboardEvent) {
  if (editingCellId.value || !event.clipboardData) return
  const custom = event.clipboardData.getData(reportClipboardMime)
  const matrix = custom
    ? parseClipboardMatrix(custom)
    : textClipboardMatrix(event.clipboardData.getData('text/plain'))
  if (!matrix.length) return
  event.preventDefault()
  emit('pasteCells', matrix)
}

function parseClipboardMatrix(value: string): ReportSheetClipboardCell[][] {
  try {
    const parsed: unknown = JSON.parse(value)
    if (!Array.isArray(parsed)) return []
    return parsed.map((row) => {
      if (!Array.isArray(row)) return []
      return row.map((cell) => normalizeClipboardCell(cell))
    })
  } catch {
    return []
  }
}

function normalizeClipboardCell(value: unknown): ReportSheetClipboardCell {
  if (!value || typeof value !== 'object') return { value: '' }
  const cell = value as Partial<ReportSheetClipboardCell>
  return {
    value: typeof cell.value === 'string' ? cell.value : String(cell.value || ''),
    ...(cell.binding ? { binding: { ...cell.binding } } : {}),
    ...(cell.style ? { style: { ...cell.style } } : {}),
    ...(cell.colspan ? { colspan: Number(cell.colspan) } : {}),
    ...(cell.rowspan ? { rowspan: Number(cell.rowspan) } : {}),
  }
}

function textClipboardMatrix(value: string): ReportSheetClipboardCell[][] {
  if (!value) return []
  return value
    .replace(/\r/g, '')
    .split('\n')
    .map((row) => row.split('\t').map((cell) => ({ value: cell })))
}

function startEdit(row: number, col: number) {
  const cell = cellAt(row, col)
  editingCellId.value = cell.id
  editingValue.value =
    cell.binding?.type === 'formula'
      ? normalizeFormulaDisplay(cell.binding.formula || cell.value)
      : cell.value || ''
  emit('selectCell', row, col)
}

function handleCellClick(row: number, col: number, event: MouseEvent) {
  closeContextMenu()
  if (event.shiftKey) emit('selectRange', row, col)
  else emit('selectCell', row, col)
  if (event.detail >= 2) {
    startEdit(row, col)
  }
}

function handleCellKeydown(event: KeyboardEvent, row: number, col: number) {
  if (event.target instanceof HTMLInputElement) return
  if (event.key === 'Enter') {
    event.preventDefault()
    startEdit(row, col)
    return
  }
  if (event.key === 'Delete' || event.key === 'Backspace') {
    event.preventDefault()
    emit('clearSelection')
    return
  }
  if (event.metaKey || event.ctrlKey) {
    const key = event.key.toLowerCase()
    if (key === 'z') {
      event.preventDefault()
      if (event.shiftKey) emit('redo')
      else emit('undo')
    } else if (key === 'y') {
      event.preventDefault()
      emit('redo')
    } else if (key === 'b') {
      event.preventDefault()
      emit('toggleBold')
    } else if (key === 'i') {
      event.preventDefault()
      emit('toggleItalic')
    }
    return
  }
  const next = nextCellByKey(event.key, row, col)
  if (!next) return
  event.preventDefault()
  if (event.shiftKey) emit('selectRange', next.row, next.col)
  else emit('selectCell', next.row, next.col)
}

function nextCellByKey(key: string, row: number, col: number) {
  if (key === 'ArrowUp') return { row: Math.max(row - 1, 1), col }
  if (key === 'ArrowDown') return { row: Math.min(row + 1, props.sheet.rows), col }
  if (key === 'ArrowLeft') return { row, col: Math.max(col - 1, 1) }
  if (key === 'ArrowRight') return { row, col: Math.min(col + 1, props.sheet.cols) }
  return null
}

function commitEdit(row: number, col: number) {
  if (editingCellId.value !== cellAt(row, col).id) return
  emit('updateCellValue', row, col, editingValue.value)
  editingCellId.value = ''
}

function cancelEdit() {
  editingCellId.value = ''
}

function handleDragEnter(cellId: string) {
  if (!props.fieldDragging) return
  dragOverCellId.value = cellId
}

function handleDragLeave(cellId: string) {
  if (dragOverCellId.value === cellId) dragOverCellId.value = ''
}

function handleDrop(row: number, col: number) {
  dragOverCellId.value = ''
  emit('dropField', row, col)
}

function startFillDrag(row: number, col: number) {
  fillState.active = true
  fillState.sourceRow = row
  fillState.sourceCol = col
  fillState.targetRow = row
  fillState.targetCol = col
  window.addEventListener('mousemove', handleFillMove)
  window.addEventListener('mouseup', finishFillDrag, { once: true })
}

function handleFillMove(event: MouseEvent) {
  if (!fillState.active) return
  const target = document
    .elementFromPoint(event.clientX, event.clientY)
    ?.closest<HTMLElement>('[data-cell-id]')
  if (!target) return
  const [rowText, colText] = (target.dataset.cellId || '').split(':')
  const row = Number(rowText)
  const col = Number(colText)
  if (!Number.isInteger(row) || !Number.isInteger(col)) return

  const rowDistance = Math.abs(row - fillState.sourceRow)
  const colDistance = Math.abs(col - fillState.sourceCol)
  fillState.targetRow = rowDistance >= colDistance ? row : fillState.sourceRow
  fillState.targetCol = rowDistance >= colDistance ? fillState.sourceCol : col
}

function finishFillDrag() {
  window.removeEventListener('mousemove', handleFillMove)
  if (!fillState.active) return
  const { sourceRow, sourceCol, targetRow, targetCol } = fillState
  fillState.active = false
  if (sourceRow === targetRow && sourceCol === targetCol) return
  emit('fillCells', sourceRow, sourceCol, targetRow, targetCol)
}

function isSummaryRow(row: number) {
  return summaryRows.value.has(row)
}

function isDetailRow(row: number) {
  return detailRows.value.has(row)
}

function isGroupSummaryRow(row: number) {
  return groupSummaryRows.value.has(row)
}

function openContextMenu(row: number, col: number, event: MouseEvent) {
  const scrollEl = sheetScrollRef.value
  const rect = scrollEl?.getBoundingClientRect()
  contextMenu.row = row
  contextMenu.col = col
  contextMenu.left =
    rect && scrollEl ? event.clientX - rect.left + scrollEl.scrollLeft : event.offsetX
  contextMenu.top = rect && scrollEl ? event.clientY - rect.top + scrollEl.scrollTop : event.offsetY
  contextMenu.visible = true
  if (!isSelectedCell(row, col)) emit('selectCell', row, col)
}

function closeContextMenu() {
  contextMenu.visible = false
}

function runContextAction(
  action:
    | 'edit'
    | 'clear'
    | 'mergeSelection'
    | 'mergeRight'
    | 'unmerge'
    | 'insertRow'
    | 'insertCol'
    | 'deleteRow'
    | 'deleteCol'
    | 'summary'
    | 'groupSummary'
    | 'detail',
) {
  const { row, col } = contextMenu
  closeContextMenu()
  if (action === 'edit') startEdit(row, col)
  else if (action === 'clear') emit('clearCell', row, col)
  else if (action === 'mergeSelection') emit('mergeSelection')
  else if (action === 'mergeRight') emit('mergeCellRight', row, col)
  else if (action === 'unmerge') emit('unmergeCell', row, col)
  else if (action === 'insertRow') emit('insertRowAfter', row)
  else if (action === 'insertCol') emit('insertColAfter', col)
  else if (action === 'deleteRow') emit('deleteRow', row)
  else if (action === 'deleteCol') emit('deleteCol', col)
  else if (action === 'summary') emit('toggleSummaryRow', row)
  else if (action === 'groupSummary') emit('toggleGroupSummaryRow', row)
  else emit('toggleDetailRow', row)
}

function joinLabel(join: ReportDatasetJoin) {
  const left = props.datasets.find((item) => item.id === join.left_dataset_id)
  const right = props.datasets.find((item) => item.id === join.right_dataset_id)
  return `${left?.name || join.left_dataset_id}.${join.left_field} ${join.join_type.toUpperCase()} ${right?.name || join.right_dataset_id}.${join.right_field}`
}

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', handleResizeMove)
  window.removeEventListener('mouseup', finishResize)
  window.removeEventListener('mousemove', handleFillMove)
  window.removeEventListener('mouseup', finishFillDrag)
})
</script>

<style scoped lang="scss">
.canvas-panel {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 54px minmax(0, 1fr);
  background:
    linear-gradient(#e9edf7 1px, transparent 1px),
    linear-gradient(90deg, #e9edf7 1px, transparent 1px);
  background-size: 28px 28px;
}

.canvas-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid #dfe5f2;
  background: rgba(255, 255, 255, 0.96);
}

.toolbar-group {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.toolbar-group .q-chip {
  max-width: 260px;
}

.color-swatch {
  position: absolute;
  right: 6px;
  bottom: 4px;
  width: 15px;
  height: 4px;
  border: 1px solid rgba(23, 32, 51, 0.24);
}

.sheet-scroll {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 24px;
  position: relative;
}

.sheet-grid {
  width: max-content;
  min-width: 980px;
  display: grid;
  border: 1px solid #cfd6e6;
  background: #fff;
  box-shadow: 0 16px 36px rgba(24, 32, 51, 0.08);
}

.sheet-corner,
.sheet-col-head,
.sheet-row-head,
.sheet-cell {
  border-right: 1px solid #dfe5f2;
  border-bottom: 1px solid #dfe5f2;
}

.sheet-corner,
.sheet-col-head,
.sheet-row-head {
  display: grid;
  place-items: center;
  background: #f3f5fb;
  color: #71809a;
  font-weight: 800;
  position: sticky;
  z-index: 4;
}

.sheet-corner {
  top: 0;
  left: 0;
  z-index: 8;
}

.sheet-col-head {
  top: 0;
  z-index: 6;
}

.sheet-col-resizer {
  position: absolute;
  top: 0;
  right: -4px;
  width: 8px;
  height: 100%;
  cursor: col-resize;
  z-index: 2;
}

.sheet-row-head {
  left: 0;
  z-index: 5;
  box-shadow: 1px 0 0 #dfe5f2;
}

.sheet-row-resizer {
  position: absolute;
  right: 0;
  bottom: -4px;
  width: 100%;
  height: 8px;
  cursor: row-resize;
  z-index: 2;
}

.sheet-cell {
  min-width: 0;
  padding: 0 10px;
  background: #fff;
  cursor: cell;
  display: flex;
  align-items: center;
  overflow: hidden;
}

.sheet-cell span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sheet-cell__editor {
  width: 100%;
  height: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  font: inherit;
  color: inherit;
}

.sheet-cell.active {
  position: relative;
  z-index: 1;
  outline: 2px solid var(--q-primary);
  outline-offset: -2px;
}

.sheet-fill-handle {
  position: absolute;
  right: -1px;
  bottom: -1px;
  width: 9px;
  height: 9px;
  border: 1px solid #fff;
  background: var(--q-primary);
  cursor: crosshair;
  z-index: 3;
}

.sheet-cell.fill-target {
  position: relative;
  z-index: 2;
  outline: 2px dashed var(--q-primary);
  outline-offset: -3px;
}

.sheet-cell.bound {
  background: #fbfaff;
}

.sheet-cell.drop-target {
  position: relative;
  z-index: 2;
  outline: 2px dashed var(--q-primary);
  outline-offset: -4px;
  background: #f4f1ff;
}

.sheet-cell.drop-target::after {
  content: attr(data-drop-label);
  position: absolute;
  right: 8px;
  bottom: 5px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--q-primary);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  pointer-events: none;
}

.sheet-cell.selected {
  background: #f1efff;
  box-shadow: inset 0 0 0 1px rgba(115, 103, 240, 0.34);
}

.sheet-row-head.detail,
.sheet-cell.detail {
  background: #f8fbff;
}

.sheet-row-head.detail {
  color: #4b6b9b;
}

.sheet-row-head.summary,
.sheet-cell.summary {
  background: #fff8e8;
}

.sheet-row-head.summary {
  color: #b7791f;
}

.sheet-row-head.group-summary,
.sheet-cell.group-summary {
  background: #eefbf6;
}

.sheet-row-head.group-summary {
  color: #187a5b;
}

.sheet-context-menu {
  position: absolute;
  z-index: 20;
  width: 168px;
  padding: 6px;
  border: 1px solid #dfe5f2;
  border-radius: 6px;
  background: #fff;
  box-shadow: 0 14px 34px rgba(24, 32, 51, 0.16);
}

.sheet-context-menu button {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 9px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: #172033;
  text-align: left;
  cursor: pointer;
}

.sheet-context-menu button:hover:not(:disabled) {
  background: #f1efff;
  color: var(--q-primary);
}

.sheet-context-menu button:disabled {
  color: #a6afc0;
  cursor: not-allowed;
}

.sheet-context-menu__separator {
  height: 1px;
  margin: 5px 0;
  background: #e7ecf6;
}
</style>
