import { describe, expect, it, vi } from 'vitest'
import type { TableField } from 'src/api/services/sys-table'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { resolveRuntimeColumns } from 'src/utils/column-format'

vi.mock('src/api/services/generalization', () => ({
  useGeneralizationApi: () => ({ queryGeneralizationByCode: vi.fn() }),
}))
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ menus: [] }) }))

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
})
