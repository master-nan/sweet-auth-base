import { describe, expect, it } from 'vitest'
import { SysTableFieldType } from 'src/types/enum'
import { parseParamsSchema } from 'src/utils/params-schema'

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

  it('does not accept removed storage type ids', () => {
    const fields = parseParamsSchema(
      JSON.stringify([
        { field_code: 'old_float', field_type: 2 },
        { field_code: 'old_tiny', field_type: 9 },
      ]),
    )

    expect(fields.map((field) => field.field_type)).toEqual([
      SysTableFieldType.VARCHAR,
      SysTableFieldType.VARCHAR,
    ])
  })
})
