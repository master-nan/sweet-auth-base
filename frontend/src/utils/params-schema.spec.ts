import { describe, expect, it } from 'vitest'
import { SysTableFieldInputType, SysTableFieldType } from '@/types/enum'
import { parseParamsSchema } from '@/utils/params-schema'

describe('parameter schema storage types', () => {
  it('maps JSON Schema numbers to exact Decimal metadata', () => {
    const fields = parseParamsSchema(
      JSON.stringify({
        type: 'object',
        properties: { amount: { type: 'number', title: '金额', default: 12.34 } },
      }),
    )

    expect(fields).toHaveLength(1)
    expect(fields[0]?.field_type).toBe(SysTableFieldType.DECIMAL)
    expect(fields[0]?.numeric_precision).toBe(38)
    expect(fields[0]?.numeric_scale).toBe(18)
    expect(fields[0]?.default_value).toBe('12.34')
  })

  it('accepts canonical Decimal and SmallInt ids', () => {
    const fields = parseParamsSchema(
      JSON.stringify([
        { field_code: 'amount', field_type: 2 },
        { field_code: 'quantity', field_type: 9 },
      ]),
    )

    expect(fields.map((field) => field.field_type)).toEqual([
      SysTableFieldType.DECIMAL,
      SysTableFieldType.SMALLINT,
    ])
  })

  it('rejects historical storage type ids after canonical migration', () => {
    const fields = parseParamsSchema(
      JSON.stringify([
        { field_code: 'historical_smallint', field_type: 12 },
        { field_code: 'historical_decimal', field_type: 13 },
      ]),
    )

    expect(fields.map((field) => field.field_type)).toEqual([
      SysTableFieldType.VARCHAR,
      SysTableFieldType.VARCHAR,
    ])
  })

  it('maps year_month to the canonical input type', () => {
    const fields = parseParamsSchema(
      JSON.stringify([
        { field_code: 'billing_month', field_type: 'date', input_type: 'year_month' },
      ]),
    )

    expect(fields[0]?.input_type).toBe(SysTableFieldInputType.YEAR_MONTH_PICKER)
  })
})
