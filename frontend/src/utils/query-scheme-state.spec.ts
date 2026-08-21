import { describe, expect, it } from 'vitest'
import { ExpressionLogic, ExpressionType, SysTableFieldType } from 'src/types/enum'
import {
  isSimpleQueryExpression,
  normalizeQuerySchemePayload,
  normalizeQueryValueByFieldType,
  queryExpressionDepth,
} from './query-state'
import { QuerySchemeBindingKind } from 'src/modules/query-scheme/types'

describe('query scheme expression modes', () => {
  it('allows lossless simple mode only for one AND group without nesting', () => {
    expect(isSimpleQueryExpression([{ logic: ExpressionLogic.AND, rules: [], nested: [] }])).toBe(true)
    expect(isSimpleQueryExpression([{ logic: ExpressionLogic.OR, rules: [], nested: [] }])).toBe(false)
    expect(isSimpleQueryExpression([{ logic: ExpressionLogic.AND, rules: [], nested: [{ logic: ExpressionLogic.AND, rules: [], nested: [] }] }])).toBe(false)
  })

  it('detects schema depth three for read-only UI handling', () => {
    const groups = [{ logic: ExpressionLogic.AND, rules: [], nested: [{ logic: ExpressionLogic.AND, rules: [], nested: [{ logic: ExpressionLogic.AND, rules: [], nested: [] }] }] }]
    expect(queryExpressionDepth(groups)).toBe(3)
  })

  it('preserves placeholder rules targeted by a controlled binding', () => {
    const payload = normalizeQuerySchemePayload(
      {
        expressions: [{ logic: ExpressionLogic.AND, rules: [{ field: 'created_at', expression_type: ExpressionType.EQ, value: null }], nested: [] }],
        quick_query: { keyword: '' },
        order: { field: '', is_asc: false },
      },
      [{ pointer: '/expressions/0/rules/0/value', kind: QuerySchemeBindingKind.TODAY }],
    )
    expect(payload.expressions[0]?.rules).toHaveLength(1)
    expect(payload.bindings[0]?.kind).toBe(QuerySchemeBindingKind.TODAY)
  })
})

describe('query scheme numeric values', () => {
  it('keeps exact Decimal conditions outside JavaScript Number conversion', () => {
    const value = '12345678901234567890.1234567890'
    expect(normalizeQueryValueByFieldType(value, SysTableFieldType.DECIMAL)).toBe(value)
  })

  it('normalizes canonical SmallInt conditions as integers', () => {
    expect(normalizeQueryValueByFieldType('32767', SysTableFieldType.SMALLINT)).toBe(32767)
  })
})
