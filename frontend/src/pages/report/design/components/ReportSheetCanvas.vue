<template>
  <section class="canvas-panel">
    <div class="canvas-toolbar">
      <div class="toolbar-group">
        <q-btn flat dense icon="undo" disable />
        <q-btn flat dense icon="redo" disable />
        <q-separator vertical />
        <q-btn
          flat
          dense
          icon="format_bold"
          :color="activeBold ? 'primary' : 'dark'"
          @click="$emit('toggleBold')"
        />
        <q-btn flat dense icon="format_align_left" @click="$emit('setAlign', 'left')" />
        <q-btn flat dense icon="format_align_center" @click="$emit('setAlign', 'center')" />
        <q-btn flat dense icon="format_align_right" @click="$emit('setAlign', 'right')" />
        <q-separator vertical />
        <q-btn flat dense icon="call_merge" label="合并选区" :disable="!hasRangeSelection" @click="$emit('mergeSelection')" />
        <q-btn flat dense icon="splitscreen" label="取消合并" @click="$emit('unmergeActiveCell')" />
        <q-btn flat dense icon="backspace" label="清除选区" @click="$emit('clearSelection')" />
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
        <q-btn outline dense color="primary" icon="add" label="行" @click="$emit('addRow')" />
        <q-btn outline dense color="primary" icon="add" label="列" @click="$emit('addCol')" />
        <q-btn flat dense icon="zoom_out" @click="$emit('zoomOut')" />
        <q-chip dense square outline color="primary">{{ Math.round(scale * 100) }}%</q-chip>
        <q-btn flat dense icon="zoom_in" @click="$emit('zoomIn')" />
      </div>
    </div>

    <div ref="sheetScrollRef" class="sheet-scroll" @scroll="closeContextMenu">
      <div
        class="sheet-grid"
        :style="{
          gridTemplateColumns: `42px repeat(${sheet.cols}, 118px)`,
          zoom: scale,
        }"
      >
        <div class="sheet-corner" />
        <div
          v-for="header in columnHeaders"
          :key="header.key"
          class="sheet-col-head"
          :style="header.style"
        >
          {{ header.label }}
        </div>
        <template v-for="renderRow in renderRows" :key="renderRow.key">
          <div
            class="sheet-row-head"
            :class="{ summary: renderRow.summary }"
            :style="renderRow.headerStyle"
          >
            {{ renderRow.row }}
          </div>
          <template v-for="renderCell in renderRow.cells" :key="renderCell.key">
            <div
              class="sheet-cell"
              :class="{
                active: renderCell.active,
                selected: renderCell.selected,
                bound: renderCell.bound,
                summary: renderCell.summary,
              }"
              :style="renderCell.style"
              :data-cell-id="renderCell.id"
              role="button"
              tabindex="0"
              :draggable="editingCellId !== renderCell.id"
              @click="handleCellClick(renderCell.row, renderCell.col, $event)"
              @dblclick.stop="startEdit(renderCell.row, renderCell.col)"
              @keydown.enter.prevent="startEdit(renderCell.row, renderCell.col)"
              @mousedown.left="$emit('startDragCell', renderCell.row, renderCell.col)"
              @mouseup.left="$emit('dropField', renderCell.row, renderCell.col)"
              @dragstart="$emit('startDragCell', renderCell.row, renderCell.col)"
              @dragover.prevent
              @drop.prevent="$emit('dropField', renderCell.row, renderCell.col)"
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
              >
              <span v-else>{{ renderCell.value }}</span>
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
        <button type="button" @click="runContextAction('edit')"><q-icon name="edit" /> 编辑文本</button>
        <button type="button" @click="runContextAction('clear')"><q-icon name="backspace" /> 清除单元格</button>
        <button type="button" :disabled="!hasRangeSelection" @click="runContextAction('mergeSelection')">
          <q-icon name="call_merge" /> 合并选区
        </button>
        <button type="button" @click="runContextAction('mergeRight')"><q-icon name="call_merge" /> 合并右侧</button>
        <button type="button" @click="runContextAction('unmerge')"><q-icon name="splitscreen" /> 取消合并</button>
        <div class="sheet-context-menu__separator" />
        <button type="button" @click="runContextAction('insertRow')"><q-icon name="table_rows" /> 下方插入行</button>
        <button type="button" @click="runContextAction('insertCol')"><q-icon name="view_column" /> 右侧插入列</button>
        <div class="sheet-context-menu__separator" />
        <button type="button" @click="runContextAction('summary')">
          <q-icon name="functions" /> {{ isSummaryRow(contextMenu.row) ? '取消汇总行' : '设为汇总行' }}
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
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
  reportSheetCellSpan,
  type ReportSheetRange,
} from 'src/modules/report/sheet'

const props = defineProps<{
  sheet: ReportSheetConfig
  selectedCellId: string
  selectionRange: ReportSheetRange | null
  activeBold: boolean
  scale: number
  datasets: ReportDataset[]
  datasetJoins: ReportDatasetJoin[]
}>()

const emit = defineEmits<{
  toggleBold: []
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
  startDragCell: [row: number, col: number]
  dropField: [row: number, col: number]
  updateCellValue: [row: number, col: number, value: string]
  clearCell: [row: number, col: number]
  mergeCellRight: [row: number, col: number]
  unmergeCell: [row: number, col: number]
  insertRowAfter: [row: number]
  insertColAfter: [col: number]
  toggleSummaryRow: [row: number]
  zoomIn: []
  zoomOut: []
}>()

const editingCellId = ref('')
const editingValue = ref('')
const sheetScrollRef = ref<HTMLElement | null>(null)
const contextMenu = reactive({
  visible: false,
  row: 1,
  col: 1,
  left: 0,
  top: 0,
})
const hasRangeSelection = computed(() => {
  if (!props.selectionRange) return false
  const bounds = reportNormalizeSheetRange(props.selectionRange)
  return bounds.maxRow > bounds.minRow || bounds.maxCol > bounds.minCol
})

const summaryRows = computed(() => new Set(props.sheet.summary_rows || []))

const selectionBounds = computed(() =>
  props.selectionRange ? reportNormalizeSheetRange(props.selectionRange) : null,
)

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
    return {
      key: `row-${row}`,
      row,
      summary,
      headerStyle: {
        gridColumn: 1,
        gridRow: row + 1,
      },
      cells: renderCellsForRow(row, summary),
    }
  }),
)

function renderCellsForRow(row: number, summary: boolean) {
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
      value: cell.value || '',
      active: props.selectedCellId === cell.id,
      selected: isSelectedCell(row, col),
      bound: Boolean(cell.binding?.field),
      summary,
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
  return cellMap.value.get(cellKey(row, col)) || {
    id: reportCellId(row, col),
    row,
    col,
    value: '',
  }
}

function cellKey(row: number, col: number) {
  return `${row}:${col}`
}

function isSelectedCell(row: number, col: number) {
  const bounds = selectionBounds.value
  return Boolean(bounds && row >= bounds.minRow && row <= bounds.maxRow && col >= bounds.minCol && col <= bounds.maxCol)
}

function startEdit(row: number, col: number) {
  const cell = cellAt(row, col)
  editingCellId.value = cell.id
  editingValue.value = cell.value || ''
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

function commitEdit(row: number, col: number) {
  if (editingCellId.value !== cellAt(row, col).id) return
  emit('updateCellValue', row, col, editingValue.value)
  editingCellId.value = ''
}

function cancelEdit() {
  editingCellId.value = ''
}

function isSummaryRow(row: number) {
  return summaryRows.value.has(row)
}

function openContextMenu(row: number, col: number, event: MouseEvent) {
  const scrollEl = sheetScrollRef.value
  const rect = scrollEl?.getBoundingClientRect()
  contextMenu.row = row
  contextMenu.col = col
  contextMenu.left = rect && scrollEl ? event.clientX - rect.left + scrollEl.scrollLeft : event.offsetX
  contextMenu.top = rect && scrollEl ? event.clientY - rect.top + scrollEl.scrollTop : event.offsetY
  contextMenu.visible = true
  emit('selectCell', row, col)
}

function closeContextMenu() {
  contextMenu.visible = false
}

function runContextAction(
  action: 'edit' | 'clear' | 'mergeSelection' | 'mergeRight' | 'unmerge' | 'insertRow' | 'insertCol' | 'summary',
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
  else emit('toggleSummaryRow', row)
}

function joinLabel(join: ReportDatasetJoin) {
  const left = props.datasets.find((item) => item.id === join.left_dataset_id)
  const right = props.datasets.find((item) => item.id === join.right_dataset_id)
  return `${left?.name || join.left_dataset_id}.${join.left_field} ${join.join_type.toUpperCase()} ${right?.name || join.right_dataset_id}.${join.right_field}`
}
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
  grid-auto-rows: 42px;
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

.sheet-cell.bound {
  background: #fbfaff;
}

.sheet-cell.selected {
  background: #f1efff;
  box-shadow: inset 0 0 0 1px rgba(115, 103, 240, 0.34);
}

.sheet-row-head.summary,
.sheet-cell.summary {
  background: #fff8e8;
}

.sheet-row-head.summary {
  color: #b7791f;
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
