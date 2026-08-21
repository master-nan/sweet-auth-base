import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import type { TableField } from 'src/api/services/sys-table'
import { SysTableFieldInputType, SysTableFieldType, SysTableFieldTypeMap } from 'src/types/enum'
import {
  getFieldControlType,
  coerceFieldValue,
  compareExactDecimal,
  isNumericFieldType,
  parseLinkageConfig,
  resolveOrganizationSelectorConfig,
} from 'src/utils/field-metadata'
import type { OrganizationSelectorType } from 'src/types/organization-selector'

const canonicalFieldTypes = [
  ['BigIntFieldType', SysTableFieldType.BIGINT],
  ['DecimalFieldType', SysTableFieldType.DECIMAL],
  ['VarcharFieldType', SysTableFieldType.VARCHAR],
  ['TextFieldType', SysTableFieldType.TEXT],
  ['BooleanFieldType', SysTableFieldType.BOOLEAN],
  ['DateFieldType', SysTableFieldType.DATE],
  ['DatetimeFieldType', SysTableFieldType.DATETIME],
  ['TimeFieldType', SysTableFieldType.TIME],
  ['SmallIntFieldType', SysTableFieldType.SMALLINT],
  ['JsonFieldType', SysTableFieldType.JSON],
  ['IntFieldType', SysTableFieldType.INT],
] as const

describe('canonical metadata contract', () => {
  it('keeps Backend and Frontend storage type ids identical', () => {
    const backendSource = readFileSync(
      resolve(process.cwd(), '../backend/enum/enum.go'),
      'utf8',
    )
    const block = backendSource.match(/type SysTableFieldType uint8[\s\S]*?const \(\n([\s\S]*?)\n\)/)?.[1]
    const backendFieldTypes = Object.fromEntries(
      [...(block?.matchAll(/^\s*(\w+FieldType)\s+SysTableFieldType\s*=\s*(\d+)$/gm) || [])].map(
        (match) => [match[1], Number(match[2])],
      ),
    )

    expect(backendFieldTypes).toEqual(Object.fromEntries(canonicalFieldTypes))
  })
})

describe('organization selector metadata resolver', () => {
  it('exposes only canonical storage type ids', () => {
    expect(Object.values(SysTableFieldType).filter((value) => typeof value === 'number')).toEqual([
      1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
    ])
    expect(SysTableFieldType.DECIMAL).toBe(2)
    expect(SysTableFieldType.SMALLINT).toBe(9)
    expect(SysTableFieldTypeMap).not.toHaveProperty('12')
    expect(SysTableFieldTypeMap).not.toHaveProperty('13')
  })

  it('keeps Decimal values as text and recognizes canonical SmallInt', () => {
    expect(coerceFieldValue('12345678901234567890.123400', SysTableFieldType.DECIMAL)).toBe(
      '12345678901234567890.123400',
    )
    expect(isNumericFieldType(SysTableFieldType.SMALLINT)).toBe(true)
  })

  it('compares large Decimal values without JavaScript Number conversion', () => {
    expect(compareExactDecimal('99999999999999999999.99', '99999999999999999999.98')).toBe(1)
    expect(compareExactDecimal('-12345678901234567890.1', '-12345678901234567890.01')).toBe(-1)
    expect(compareExactDecimal('1.2300', '1.23')).toBe(0)
  })
  it.each<[OrganizationSelectorType, OrganizationSelectorType]>([
    ['legal_entity', 'legal_entity'],
    ['org_unit', 'org_unit'],
    ['employee', 'employee'],
    ['position', 'position'],
  ])('resolves the %s selector protocol', (selectorType, expected) => {
    expect(
      resolveOrganizationSelectorConfig({
        input_type: 'selector',
        selector_type: selectorType,
      }),
    ).toEqual({
      selectorType: expected,
      multiple: false,
      includeHistory: false,
      disabled: false,
    })
  })

  it('reads flags and canonical selector metadata from linkage_config', () => {
    expect(
      resolveOrganizationSelectorConfig({
        input_type: SysTableFieldInputType.SELECT,
        linkage_config: JSON.stringify({
          selector_type: 'position',
          multiple: true,
          include_history: 'true',
          disabled: 1,
        }),
      }),
    ).toEqual({
      selectorType: 'position',
      multiple: true,
      includeHistory: true,
      disabled: true,
    })
  })

  it('rejects removed selector aliases at runtime', () => {
    expect(
      resolveOrganizationSelectorConfig({
        input_type: SysTableFieldInputType.SELECT,
        linkage_config: JSON.stringify({ selector_type: 'position_select' }),
      }),
    ).toBeNull()
  })

  it('gives direct selector metadata priority over linkage and dictionary metadata', () => {
    expect(
      resolveOrganizationSelectorConfig({
        input_type: 'selector',
        selector_type: 'employee',
        dict_code: 'org_unit_type',
        multiple: false,
        linkage_config: {
          selector_type: 'org_unit',
          multiple: true,
        },
      }),
    ).toEqual({
      selectorType: 'employee',
      multiple: false,
      includeHistory: false,
      disabled: false,
    })
  })

  it('returns null when selector metadata is absent', () => {
    expect(
      resolveOrganizationSelectorConfig({
        input_type: SysTableFieldInputType.INPUT,
        field_type: SysTableFieldType.VARCHAR,
      }),
    ).toBeNull()
  })

  it('keeps existing dictionary and linkage metadata on their original fallback path', () => {
    const dictionaryField = {
      input_type: SysTableFieldInputType.SELECT,
      field_type: SysTableFieldType.VARCHAR,
      dict_code: 'whether',
    } as TableField
    const linkageField = {
      input_type: SysTableFieldInputType.INPUT,
      field_type: SysTableFieldType.BIGINT,
      linkage_config: JSON.stringify({
        linkage: {
          enabled: true,
          mode: 'cascader',
        },
      }),
    } as TableField

    expect(resolveOrganizationSelectorConfig(dictionaryField)).toBeNull()
    expect(getFieldControlType(dictionaryField)).toBe('select')
    expect(resolveOrganizationSelectorConfig(linkageField)).toBeNull()
    expect(parseLinkageConfig(linkageField)).toEqual({
      enabled: true,
      mode: 'cascader',
    })
    expect(getFieldControlType(linkageField)).toBe('cascader')
  })

  it('rejects an unknown explicit selector instead of changing legacy rendering', () => {
    expect(
      resolveOrganizationSelectorConfig({
        input_type: 'selector',
        selector_type: 'organization',
        linkage_config: {
          selector_type: 'employee',
        },
      }),
    ).toBeNull()
  })
})
