import { describe, expect, it } from 'vitest'

import { createBlankReportSheet, makeReportCellId } from './schema'
import {
  reportDeleteSheetColumn,
  reportDeleteSheetRow,
  reportInsertSheetColumn,
  reportInsertSheetRow,
  reportPasteSheetCells,
  reportSheetClipboardMatrix,
} from './sheet'

describe('report sheet structure editing', () => {
  it('keeps cells, merged spans, row markers and heights aligned when rows change', () => {
    const sheet = createBlankReportSheet(10, 8)
    sheet.cells = [
      { id: makeReportCellId(2, 2), row: 2, col: 2, value: '标题', rowspan: 2 },
      { id: makeReportCellId(3, 4), row: 3, col: 4, value: '金额' },
    ]
    sheet.detail_rows = [2, 4]
    sheet.summary_rows = [3]
    sheet.row_heights = { '2': 50, '3': 60 }

    const inserted = reportInsertSheetRow(sheet, 2)
    expect(inserted.rows).toBe(11)
    expect(inserted.cells).toEqual([
      { id: makeReportCellId(2, 2), row: 2, col: 2, value: '标题', rowspan: 3 },
      { id: makeReportCellId(4, 4), row: 4, col: 4, value: '金额' },
    ])
    expect(inserted.detail_rows).toEqual([2, 5])
    expect(inserted.summary_rows).toEqual([4])
    expect(inserted.row_heights).toEqual({ '2': 50, '4': 60 })

    const deleted = reportDeleteSheetRow(inserted, 3)
    expect(deleted.rows).toBe(10)
    expect(deleted.cells).toEqual(sheet.cells)
    expect(deleted.detail_rows).toEqual([2, 4])
    expect(deleted.summary_rows).toEqual([3])
    expect(deleted.row_heights).toEqual({ '2': 50, '3': 60 })
  })

  it('keeps cells, merged spans and widths aligned when columns change', () => {
    const sheet = createBlankReportSheet(10, 8)
    sheet.cells = [
      { id: makeReportCellId(2, 2), row: 2, col: 2, value: '标题', colspan: 2 },
      { id: makeReportCellId(4, 3), row: 4, col: 3, value: '金额' },
    ]
    sheet.column_widths = { '2': 150, '3': 180 }

    const inserted = reportInsertSheetColumn(sheet, 2)
    expect(inserted.cols).toBe(9)
    expect(inserted.cells).toEqual([
      { id: makeReportCellId(2, 2), row: 2, col: 2, value: '标题', colspan: 3 },
      { id: makeReportCellId(4, 4), row: 4, col: 4, value: '金额' },
    ])
    expect(inserted.column_widths).toEqual({ '2': 150, '4': 180 })

    const deleted = reportDeleteSheetColumn(inserted, 3)
    expect(deleted.cols).toBe(8)
    expect(deleted.cells).toEqual(sheet.cells)
    expect(deleted.column_widths).toEqual({ '2': 150, '3': 180 })
  })

  it('copies and pastes values, bindings and styles as a matrix', () => {
    const sheet = createBlankReportSheet(10, 8)
    sheet.cells = [
      {
        id: makeReportCellId(2, 2),
        row: 2,
        col: 2,
        value: '[订单.金额]',
        binding: { type: 'sum', dataset_id: 'orders', field: 'amount' },
        style: { bold: true, color: '#6d5dfc' },
      },
      { id: makeReportCellId(2, 3), row: 2, col: 3, value: '元' },
    ]

    const matrix = reportSheetClipboardMatrix(sheet, {
      minRow: 2,
      maxRow: 2,
      minCol: 2,
      maxCol: 3,
    })
    const pasted = reportPasteSheetCells(sheet, 5, 4, matrix)

    expect(pasted.cells).toContainEqual({
      id: makeReportCellId(5, 4),
      row: 5,
      col: 4,
      value: '[订单.金额]',
      binding: { type: 'sum', dataset_id: 'orders', field: 'amount' },
      style: { bold: true, color: '#6d5dfc' },
    })
    expect(pasted.cells).toContainEqual({
      id: makeReportCellId(5, 5),
      row: 5,
      col: 5,
      value: '元',
    })
  })
})
