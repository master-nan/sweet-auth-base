import { describe, expect, it } from 'vitest'

import { createBlankReportSheet, makeReportCellId, normalizeReportSheet } from './schema'
import {
  copyReportUniverCellMetadata,
  getReportUniverFillSourceIndex,
  reportSheetToUniverSnapshot,
  univerSnapshotToReportSheet,
} from './univer'

describe('report Univer snapshot adapter', () => {
  it('round trips report bindings, formulas, styles, merges, sizes and row roles', () => {
    const sheet = createBlankReportSheet(18, 9)
    sheet.active_cell = makeReportCellId(3, 2)
    sheet.scale = 1.15
    sheet.detail_rows = [3]
    sheet.summary_rows = [8, 9]
    sheet.group_summary_rows = [8]
    sheet.row_heights = { '3': 52 }
    sheet.column_widths = { '2': 180 }
    sheet.cells = [
      {
        id: makeReportCellId(1, 1),
        row: 1,
        col: 1,
        value: '销售明细',
        colspan: 3,
        style: {
          bold: true,
          italic: true,
          align: 'center',
          background: '#f1f0ff',
          color: '#5547d9',
        },
      },
      {
        id: makeReportCellId(3, 2),
        row: 3,
        col: 2,
        value: '订单.S(金额)',
        binding: { type: 'field', dataset_id: 'orders', field: 'amount' },
        style: { align: 'right' },
      },
      {
        id: makeReportCellId(9, 2),
        row: 9,
        col: 2,
        value: '=SUM(B3:B8)',
        binding: { type: 'formula', formula: '=SUM(B3:B8)' },
        style: { univer: { fs: 14, n: { pattern: '#,##0.00' } } },
      },
    ]

    const snapshot = reportSheetToUniverSnapshot(sheet, '销售报表')
    const restored = univerSnapshotToReportSheet(snapshot)

    expect(snapshot.name).toBe('销售报表')
    expect(snapshot.sheets['report-sheet']?.zoomRatio).toBe(1.15)
    expect(restored).toEqual(sheet)
  })

  it('keeps business metadata independent from the visible cell value', () => {
    const sheet = createBlankReportSheet()
    sheet.cells = [
      {
        id: makeReportCellId(2, 1),
        row: 2,
        col: 1,
        value: '客户名称',
        binding: { type: 'group', dataset_id: 'customer', field: 'name' },
      },
    ]

    const snapshot = reportSheetToUniverSnapshot(sheet)
    const cell = snapshot.sheets['report-sheet']?.cellData?.[1]?.[0]
    expect(cell?.v).toBe('客户名称')
    expect(cell?.custom?.sweetReport.binding).toEqual({
      type: 'group',
      dataset_id: 'customer',
      field: 'name',
    })
    expect(univerSnapshotToReportSheet(snapshot).cells[0]?.binding?.type).toBe('group')
  })

  it('removes the obsolete binding highlight without changing user formatting', () => {
    const sheet = createBlankReportSheet()
    sheet.cells = [
      {
        id: makeReportCellId(2, 1),
        row: 2,
        col: 1,
        value: '客户.S(名称)',
        binding: { type: 'field', dataset_id: 'customer', field: 'name' },
        style: { bold: true, color: '#6d5dfc', align: 'center' },
      },
      {
        id: makeReportCellId(2, 2),
        row: 2,
        col: 2,
        value: '重点客户',
        style: { bold: true, color: '#6d5dfc' },
      },
    ]

    const normalized = normalizeReportSheet(sheet)

    expect(normalized.cells[0]?.style).toEqual({ align: 'center' })
    expect(normalized.cells[1]?.style).toEqual({ bold: true, color: '#6d5dfc' })
  })

  it('keeps row roles attached to rows when Univer shifts row data', () => {
    const sheet = createBlankReportSheet(10, 6)
    sheet.detail_rows = [3]
    sheet.summary_rows = [6]
    const snapshot = reportSheetToUniverSnapshot(sheet)
    const worksheet = snapshot.sheets[snapshot.sheetOrder[0]!]!

    worksheet.rowData = Object.fromEntries(
      Object.entries(worksheet.rowData || {}).map(([index, row]) => [
        Number(index) >= 2 ? Number(index) + 1 : Number(index),
        row,
      ]),
    )
    worksheet.rowCount = 11

    const restored = univerSnapshotToReportSheet(snapshot)
    expect(restored.detail_rows).toEqual([4])
    expect(restored.summary_rows).toEqual([7])
  })

  it('copies binding metadata and keeps the shifted formula from native fill data', () => {
    const source = {
      f: '=A1*2',
      custom: {
        sweetReport: {
          binding: { type: 'formula', formula: '=A1*2' },
        },
      },
    }
    const result = copyReportUniverCellMetadata(source, { f: '=A2*2', v: 20 })

    expect(result).toEqual({
      f: '=A2*2',
      v: 20,
      custom: {
        sweetReport: {
          binding: { type: 'formula', formula: '=A2*2' },
        },
      },
    })
    expect(source.custom.sweetReport.binding.formula).toBe('=A1*2')
  })

  it('reverses the source pattern when filling upward or leftward', () => {
    const source = [3, 4, 5]

    expect([0, 1, 2, 3].map((offset) => getReportUniverFillSourceIndex(source, offset))).toEqual([
      3, 4, 5, 3,
    ])
    expect(
      [0, 1, 2, 3].map((offset) => getReportUniverFillSourceIndex(source, offset, true)),
    ).toEqual([5, 4, 3, 5])
  })
})
