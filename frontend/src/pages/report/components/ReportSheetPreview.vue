<template>
  <div class="report-sheet-preview">
    <div v-if="loading" class="report-sheet-preview__state">
      <q-spinner color="primary" size="32px" />
      <span>正在加载报表数据</span>
    </div>

    <div v-else-if="!renderRows.length" class="report-sheet-preview__state">
      <q-icon name="dataset_off" size="32px" />
      <span>暂无可预览的数据或单元格配置</span>
    </div>

    <div v-else class="report-sheet-preview__scroll">
      <table class="report-sheet-preview__table">
        <tbody>
          <tr v-for="row in renderRows" :key="row.key">
            <td
              v-for="cell in row.cells"
              :key="cell.key"
              :colspan="cell.colspan"
              :rowspan="cell.rowspan"
              :class="{
                'is-bound': cell.bound,
                'is-summary': cell.summary,
                'is-empty': !cell.value,
              }"
              :style="cell.style"
            >
              {{ cell.value }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  ReportDataset,
  ReportKind,
  ReportPreviewRes,
  ReportSheetCell,
  ReportSheetConfig,
} from 'src/api/services/report'
import {
  reportRuntimeCellValue,
  reportSheetCellAt,
  reportSheetCellSpan,
  reportSheetIsCoveredCell,
  reportSheetUsedBounds,
} from 'src/modules/report/sheet'

const props = withDefaults(defineProps<{
  sheet: ReportSheetConfig
  datasets: ReportDataset[]
  previewData: ReportPreviewRes
  loading?: boolean
  reportKind?: ReportKind
}>(), {
  loading: false,
  reportKind: 'detail',
})

type RenderCell = {
  key: string
  value: string
  bound: boolean
  summary: boolean
  colspan: number
  rowspan: number
  style: Record<string, string | number | undefined>
}

type RenderRow = {
  key: string
  cells: RenderCell[]
}

const usedBounds = computed(() => reportSheetUsedBounds(props.sheet))

const templateRows = computed(() => {
  const rows = new Set<number>()
  props.sheet.cells.forEach((cell) => {
    if (
      cell.binding?.field &&
      cell.binding.type !== 'static' &&
      !props.sheet.summary_rows?.includes(cell.row)
    ) {
      rows.add(cell.row)
    }
  })
  return [...rows].sort((a, b) => a - b)
})

const templateRowGroups = computed(() => {
  const groups: number[][] = []
  templateRows.value.forEach((row) => {
    const lastGroup = groups[groups.length - 1]
    const lastRow = lastGroup?.[lastGroup.length - 1]
    if (lastGroup && lastRow !== undefined && row === lastRow + 1) {
      lastGroup.push(row)
      return
    }
    groups.push([row])
  })
  return groups
})

const renderRows = computed<RenderRow[]>(() => {
  const rows: RenderRow[] = []
  const templateGroupByStart = new Map(templateRowGroups.value.map((group) => [group[0], group]))
  const templateRowsInGroup = new Set(templateRowGroups.value.flat())
  const dataRows = props.previewData.rows || []
  const minRow = usedBounds.value.minRow
  const maxRow = usedBounds.value.maxRow
  const minCol = usedBounds.value.minCol
  const maxCol = usedBounds.value.maxCol

  for (let rowIndex = minRow; rowIndex <= maxRow; rowIndex += 1) {
    const templateGroup = templateGroupByStart.get(rowIndex)
    if (templateGroup && props.reportKind === 'detail') {
      if (!dataRows.length) {
        pushVisibleRow(rows, buildDetailRow(templateGroup, undefined, 'template-empty', minCol, maxCol, maxRow))
      } else {
        dataRows.forEach((dataRow, dataIndex) => {
          pushVisibleRow(rows, buildDetailRow(templateGroup, dataRow, `data-${dataIndex}`, minCol, maxCol, maxRow))
        })
      }
      rowIndex = templateGroup[templateGroup.length - 1] || rowIndex
      continue
    }
    if (templateRowsInGroup.has(rowIndex) && props.reportKind === 'detail') continue
    pushVisibleRow(rows, buildRow(rowIndex, undefined, 'static', minCol, maxCol, maxRow))
  }

  return trimTrailingEmptyRows(rows)
})

function pushVisibleRow(rows: RenderRow[], row: RenderRow) {
  if (rowHasContent(row)) rows.push(row)
}

function rowHasContent(row: RenderRow) {
  return row.cells.some((cell) => String(cell.value || '').trim() !== '')
}

function buildRow(
  sourceRow: number,
  dataRow: Record<string, unknown> | undefined,
  suffix: string,
  minCol: number,
  maxCol: number,
  maxRow: number,
): RenderRow {
  const cells: RenderCell[] = []
  for (let col = minCol; col <= maxCol; col += 1) {
    if (reportSheetIsCoveredCell(props.sheet, sourceRow, col)) continue
    const cell = cellAt(sourceRow, col)
    const { rowspan, colspan } = reportSheetCellSpan(cell, { maxRow, maxCol })
    cells.push({
      key: `${sourceRow}:${col}:${suffix}`,
      value: displayCellValue(cell, dataRow),
      bound: Boolean(cell.binding?.field),
      summary: props.sheet.summary_rows?.includes(sourceRow) || false,
      colspan,
      rowspan,
      style: cellStyle(cell),
    })
  }
  return { key: `${sourceRow}:${suffix}`, cells }
}

function buildDetailRow(
  sourceRows: number[],
  dataRow: Record<string, unknown> | undefined,
  suffix: string,
  minCol: number,
  maxCol: number,
  maxRow: number,
): RenderRow {
  const cells: RenderCell[] = []
  for (let col = minCol; col <= maxCol; col += 1) {
    const sourceCell = firstDetailCellAt(sourceRows, col)
    if (!sourceCell) {
      cells.push(emptyRenderCell(`empty:${col}:${suffix}`))
      continue
    }
    const { colspan } = reportSheetCellSpan(sourceCell, { maxRow, maxCol })
    cells.push({
      key: `${sourceCell.row}:${sourceCell.col}:${suffix}`,
      value: displayCellValue(sourceCell, dataRow),
      bound: Boolean(sourceCell.binding?.field),
      summary: false,
      colspan,
      rowspan: 1,
      style: cellStyle(sourceCell),
    })
    col += colspan - 1
  }
  return { key: `detail:${sourceRows.join('-')}:${suffix}`, cells }
}

function firstDetailCellAt(sourceRows: number[], col: number) {
  for (const row of sourceRows) {
    if (reportSheetIsCoveredCell(props.sheet, row, col)) continue
    const cell = cellAt(row, col)
    if (cell.binding?.field || String(cell.value || '').trim() !== '') return cell
  }
  return null
}

function emptyRenderCell(key: string): RenderCell {
  return {
    key,
    value: '',
    bound: false,
    summary: false,
    colspan: 1,
    rowspan: 1,
    style: {},
  }
}

function cellAt(row: number, col: number): ReportSheetCell {
  return reportSheetCellAt(props.sheet, row, col)
}

function displayCellValue(cell: ReportSheetCell, dataRow: Record<string, unknown> | undefined) {
  if (cell.binding?.field && cell.binding.type !== 'static') {
    if (!dataRow && props.reportKind !== 'detail') {
      return aggregateCellValue(cell)
    }
    return reportRuntimeCellValue(dataRow, cell, props.datasets)
  }
  return cell.value || ''
}

function aggregateCellValue(cell: ReportSheetCell) {
  const dataRows = props.previewData.rows || []
  if (!dataRows.length) return cell.value || ''
  const values = dataRows
    .map((row) => reportRuntimeCellValue(row, cell, props.datasets))
    .filter((value) => value !== '')
  if (cell.binding?.type === 'count') return String(values.length)
  if (cell.binding?.type === 'sum') {
    const total = values.reduce((sum, value) => {
      const numeric = Number(value)
      return Number.isFinite(numeric) ? sum + numeric : sum
    }, 0)
    return String(total)
  }
  const uniqueValues = [...new Set(values)]
  if (!uniqueValues.length) return ''
  if (uniqueValues.length === 1) return uniqueValues[0] || ''
  return `${uniqueValues[0]} 等 ${uniqueValues.length} 个`
}

function cellStyle(cell: ReportSheetCell) {
  const style = cell.style || {}
  return {
    textAlign: style.align || 'left',
    fontWeight: style.bold ? 800 : 500,
    fontStyle: style.italic ? 'italic' : 'normal',
    background: style.background || undefined,
    color: style.color && !cell.binding?.field ? style.color : undefined,
  }
}

function trimTrailingEmptyRows(rows: RenderRow[]) {
  let end = rows.length
  while (end > 1) {
    const row = rows[end - 1]
    if (!row || rowHasContent(row)) break
    end -= 1
  }
  return rows.slice(0, end)
}
</script>

<style scoped lang="scss">
.report-sheet-preview {
  min-height: 260px;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fff;
  overflow: hidden;
}

.report-sheet-preview__state {
  min-height: 260px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  color: #71809a;
}

.report-sheet-preview__scroll {
  max-height: min(62vh, 720px);
  overflow: auto;
}

.report-sheet-preview__table {
  min-width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.report-sheet-preview__table td {
  min-width: 128px;
  height: 42px;
  padding: 8px 10px;
  border-right: 1px solid #dfe5f2;
  border-bottom: 1px solid #dfe5f2;
  color: #172033;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
}

.report-sheet-preview__table td.is-bound {
  background: #fbfaff;
}

.report-sheet-preview__table td.is-summary {
  background: #fff8e8;
  font-weight: 800;
}

.report-sheet-preview__table td.is-empty {
  color: transparent;
}
</style>
