import { describe, expect, it, vi } from 'vitest'
import type { TableField } from 'src/api/services/sys-table'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { queryRuntimeRelationOptions } from 'src/api/services/runtime-relation'
import {
  buildColumnFormat,
  buildRelationLookups,
  formatDateTime,
  hydrateRelationLookups,
  resolveRuntimeColumns,
} from 'src/utils/column-format'

vi.mock('src/api/services/runtime-relation', () => ({ queryRuntimeRelationOptions: vi.fn() }))
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ menus: [] }) }))
vi.mock('src/router', () => ({ Router: { currentRoute: { value: { name: '' } } } }))

const field = (overrides: Partial<TableField>): TableField =>
  ({
    id: 1,
    table_id: 1,
    field_name: '名称',
    field_code: 'name',
    field_type: SysTableFieldType.VARCHAR,
    field_length: 64,
    field_decimal_length: 0,
    input_type: SysTableFieldInputType.INPUT,
    default_value: '',
    dict_code: '',
    is_primary_key: false,
    is_index: false,
    is_quick_search: false,
    is_advanced_search: false,
    is_sort: false,
    is_null: true,
    is_list_show: true,
    is_insert_show: false,
    is_update_show: false,
    sequence: 1,
    original_field_id: 0,
    binding: '',
    ...overrides,
  }) as TableField

describe('resolveRuntimeColumns', () => {
  it('renders only the Go zero timestamp as never occurred', () => {
    expect(formatDateTime(null)).toBe('-')
    expect(formatDateTime('0001-01-01T00:00:00Z')).toBe('-')
    expect(formatDateTime('1970-01-01T00:00:00Z')).not.toBe('-')
  })

  it('combines metadata, typed overrides and virtual columns in stable order', () => {
    const result = resolveRuntimeColumns<{ name: string; status: string }>(
      [
        field({ field_code: 'status', field_name: '状态', sequence: 2, is_sort: true }),
        field({ field_code: 'name', field_name: '名称', sequence: 1 }),
      ],
      {
        context: { getDictLabel: () => '' },
        overrides: [
          { fieldCode: 'name', label: '业务名称', visible: false },
          { fieldCode: 'status', align: 'center' },
        ],
        virtualColumns: [
          {
            name: 'actions',
            label: '操作',
            field: 'actions',
            order: 100,
            defaultVisible: true,
          },
        ],
      },
    )

    expect(result.columns.map((column) => column.name)).toEqual(['name', 'status', 'actions'])
    expect(result.columns[0]?.label).toBe('业务名称')
    expect(result.columns[1]?.align).toBe('center')
    expect(result.visibleColumns).toEqual(['status', 'actions'])
    expect(result.sortableFields).toEqual(new Set(['status']))
  })

  it('does not promote an unsortable metadata field through a page override', () => {
    const result = resolveRuntimeColumns([field({ field_code: 'name', is_sort: false })], {
      context: { getDictLabel: () => '' },
      overrides: [{ fieldCode: 'name', align: 'right' }],
      virtualColumns: [
        {
          name: 'computed_total',
          label: '合计',
          field: 'computed_total',
          serverSortField: 'total_amount',
        },
      ],
    })

    expect(result.columns[0]?.sortable).toBe(false)
    expect(result.columns[1]?.sortable).toBe(true)
    expect(result.sortableFields).toEqual(new Set(['total_amount']))
  })

  it('applies list_width as a bounded default column width', () => {
    const result = resolveRuntimeColumns([field({ list_width: 180 })], {
      context: { getDictLabel: () => '' },
    })
    expect(result.columns[0]?.style).toBe('width: 180px; max-width: 180px')
    expect(result.columns[0]?.headerStyle).toBe('width: 180px; max-width: 180px')
  })

  it('does not expose a raw foreign key when a relation label is unavailable', () => {
    const relationField = field({
      field_code: 'customer_id',
      linkage_config: JSON.stringify({
        linkage: {
          enabled: true,
          mode: 'relation',
          tableCode: 'customer',
          valueKey: 'id',
          labelKey: 'name',
        },
      }),
    })
    const format = buildColumnFormat(relationField, {
      getDictLabel: () => '',
      relationLookups: { customer_id: {} },
    })
    expect(format?.(9527)).toBe('关联值未解析')
  })

  it('hydrates a relation value that is outside the first options page', async () => {
    const relationField = field({
      field_code: 'customer_id',
      linkage_config: JSON.stringify({
        linkage: {
          enabled: true,
          mode: 'relation',
          tableCode: 'customer',
          valueKey: 'id',
          labelKey: 'name',
        },
      }),
    })
    const queryOptions = vi.mocked(queryRuntimeRelationOptions)
    queryOptions
      .mockResolvedValueOnce({ items: [], total: 80 })
      .mockResolvedValueOnce({ items: [{ value: '9527', label: '华东客户' }], total: 1 })

    const lookups = await buildRelationLookups([relationField], 205)
    const hydrated = await hydrateRelationLookups(
      [relationField],
      [{ customer_id: 9527 }],
      lookups,
      205,
    )

    expect(queryOptions).toHaveBeenLastCalledWith(
      relationField.id,
      expect.objectContaining({ menu_id: 205, selected_values: ['9527'] }),
    )
    const format = buildColumnFormat(relationField, {
      getDictLabel: () => '',
      relationLookups: hydrated,
    })
    expect(hydrated).not.toBe(lookups)
    expect(hydrated.customer_id).not.toBe(lookups.customer_id)
    expect(format?.(9527)).toBe('华东客户')
  })

  it('hydrates a relation used only by detail without loading an unrelated first page', async () => {
    const relationField = field({
      field_code: 'customer_id',
      is_list_show: false,
      linkage_config: JSON.stringify({
        linkage: {
          enabled: true,
          mode: 'relation',
          tableCode: 'customer',
          valueKey: 'id',
          labelKey: 'name',
        },
      }),
    })
    const queryOptions = vi.mocked(queryRuntimeRelationOptions)
    queryOptions.mockResolvedValueOnce({
      items: [{ value: '9527', label: '华东客户' }],
      total: 1,
    })

    const lookups = await buildRelationLookups([relationField], 205)
    expect(queryOptions).not.toHaveBeenCalled()
    await hydrateRelationLookups([relationField], [{ customer_id: 9527 }], lookups, 205)

    expect(queryOptions).toHaveBeenCalledWith(
      relationField.id,
      expect.objectContaining({ selected_values: ['9527'] }),
    )
  })
})
