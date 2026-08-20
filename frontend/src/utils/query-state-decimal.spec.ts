import { describe, expect, it } from 'vitest'
import { SysTableFieldType } from 'src/types/enum'
import { normalizeQueryValueByFieldType } from 'src/utils/query-state'

describe('query numeric value normalization', () => {
  it('does not convert exact Decimal conditions to JavaScript Number', () => {
    const value = '12345678901234567890.1234567890'
    expect(normalizeQueryValueByFieldType(value, SysTableFieldType.DECIMAL)).toBe(value)
  })

  it('normalizes canonical SmallInt conditions as integers', () => {
    expect(normalizeQueryValueByFieldType('32767', SysTableFieldType.SMALLINT)).toBe(32767)
  })
})
