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
        <q-btn flat dense icon="call_merge" label="合并右侧" @click="$emit('mergeRight')" />
        <q-btn flat dense icon="backspace" label="清除" @click="$emit('clearActiveCell')" />
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

    <div class="sheet-scroll">
      <div
        class="sheet-grid"
        :style="{
          gridTemplateColumns: `42px repeat(${sheet.cols}, 118px)`,
          zoom: scale,
        }"
      >
        <div class="sheet-corner" />
        <div v-for="col in sheet.cols" :key="`head-${col}`" class="sheet-col-head">
          {{ reportColumnName(col) }}
        </div>
        <template v-for="row in sheet.rows" :key="`row-${row}`">
          <div class="sheet-row-head" :class="{ summary: isSummaryRow(row) }">{{ row }}</div>
          <div
            v-for="col in sheet.cols"
            :key="cellAt(row, col).id"
            class="sheet-cell"
            :class="{
              active: selectedCellId === cellAt(row, col).id,
              bound: !!cellAt(row, col).binding?.field,
              summary: isSummaryRow(row),
            }"
            :style="reportCellStyle(cellAt(row, col))"
            :data-cell-id="cellAt(row, col).id"
            role="button"
            tabindex="0"
            :draggable="editingCellId !== cellAt(row, col).id"
            @click="handleCellClick(row, col, $event)"
            @dblclick.stop="startEdit(row, col)"
            @keydown.enter.prevent="startEdit(row, col)"
            @mousedown.left="$emit('startDragCell', row, col)"
            @mouseup.left="$emit('dropField', row, col)"
            @dragstart="$emit('startDragCell', row, col)"
            @dragover.prevent
            @drop.prevent="$emit('dropField', row, col)"
          >
            <input
              v-if="editingCellId === cellAt(row, col).id"
              v-model="editingValue"
              autofocus
              class="sheet-cell__editor"
              @click.stop
              @keydown.enter.stop.prevent="commitEdit(row, col)"
              @keydown.esc.stop.prevent="cancelEdit"
              @blur="commitEdit(row, col)"
            >
            <span v-else>{{ cellAt(row, col).value || '' }}</span>
            <q-menu context-menu @before-show="$emit('selectCell', row, col)">
              <q-list dense style="min-width: 150px">
                <q-item clickable v-close-popup @click="startEdit(row, col)">
                  <q-item-section avatar><q-icon name="edit" /></q-item-section>
                  <q-item-section>编辑文本</q-item-section>
                </q-item>
                <q-item clickable v-close-popup @click="$emit('clearCell', row, col)">
                  <q-item-section avatar><q-icon name="backspace" /></q-item-section>
                  <q-item-section>清除单元格</q-item-section>
                </q-item>
                <q-item clickable v-close-popup @click="$emit('mergeCellRight', row, col)">
                  <q-item-section avatar><q-icon name="call_merge" /></q-item-section>
                  <q-item-section>合并右侧</q-item-section>
                </q-item>
                <q-separator />
                <q-item clickable v-close-popup @click="$emit('insertRowAfter', row)">
                  <q-item-section avatar><q-icon name="table_rows" /></q-item-section>
                  <q-item-section>下方插入行</q-item-section>
                </q-item>
                <q-item clickable v-close-popup @click="$emit('insertColAfter', col)">
                  <q-item-section avatar><q-icon name="view_column" /></q-item-section>
                  <q-item-section>右侧插入列</q-item-section>
                </q-item>
                <q-separator />
                <q-item clickable v-close-popup @click="$emit('toggleSummaryRow', row)">
                  <q-item-section avatar><q-icon name="functions" /></q-item-section>
                  <q-item-section>{{ isSummaryRow(row) ? '取消汇总行' : '设为汇总行' }}</q-item-section>
                </q-item>
              </q-list>
            </q-menu>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type {
  ReportDataset,
  ReportDatasetJoin,
  ReportSheetCell,
  ReportSheetConfig,
} from 'src/api/services/report'
import { reportCellId, reportCellStyle, reportColumnName } from 'src/modules/report/sheet'

const props = defineProps<{
  sheet: ReportSheetConfig
  selectedCellId: string
  activeBold: boolean
  scale: number
  datasets: ReportDataset[]
  datasetJoins: ReportDatasetJoin[]
}>()

const emit = defineEmits<{
  toggleBold: []
  setAlign: [value: 'left' | 'center' | 'right']
  mergeRight: []
  clearActiveCell: []
  addRow: []
  addCol: []
  selectCell: [row: number, col: number]
  startDragCell: [row: number, col: number]
  dropField: [row: number, col: number]
  updateCellValue: [row: number, col: number, value: string]
  clearCell: [row: number, col: number]
  mergeCellRight: [row: number, col: number]
  insertRowAfter: [row: number]
  insertColAfter: [col: number]
  toggleSummaryRow: [row: number]
  zoomIn: []
  zoomOut: []
}>()

const editingCellId = ref('')
const editingValue = ref('')

function cellAt(row: number, col: number): ReportSheetCell {
  return (
    props.sheet.cells.find((item) => item.row === row && item.col === col) || {
      id: reportCellId(row, col),
      row,
      col,
      value: '',
    }
  )
}

function startEdit(row: number, col: number) {
  const cell = cellAt(row, col)
  editingCellId.value = cell.id
  editingValue.value = cell.value || ''
  emit('selectCell', row, col)
}

function handleCellClick(row: number, col: number, event: MouseEvent) {
  emit('selectCell', row, col)
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
  return props.sheet.summary_rows?.includes(row) || false
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

.sheet-row-head.summary,
.sheet-cell.summary {
  background: #fff8e8;
}

.sheet-row-head.summary {
  color: #b7791f;
}
</style>
