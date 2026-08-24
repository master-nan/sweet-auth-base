import { describe, expect, it, vi } from 'vitest'
import { useRuntimeTableMetadata } from './runtime-table-metadata'
import type { RuntimeTableMetadata, TableField } from 'src/api/services/sys-table'
import {
  SysDetailOpenMode,
  SysFormOpenMode,
  SysMasterDetailMode,
  SysTableType,
} from 'src/types/enum'

vi.mock('src/api/services/sys-table', () => ({
  useTableApi: () => ({ queryRuntimeTableByCode: vi.fn() }),
}))

describe('useRuntimeTableMetadata', () => {
  it('loads once and derives stable capability field groups', async () => {
    const fields = [
      {
        field_code: 'name', field_name: '名称', sequence: 2, is_list_show: true,
        is_quick_search: true, is_advanced_search: false, is_insert_show: true,
        is_update_show: true, detail_span: 2,
      },
      {
        field_code: 'code', field_name: '编码', sequence: 3, is_list_show: true,
        is_quick_search: true, is_advanced_search: false, is_insert_show: true,
        is_update_show: true, detail_span: 1,
      },
      {
        field_code: 'status', field_name: '状态', sequence: 1, is_list_show: true,
        is_quick_search: false, is_advanced_search: true, is_insert_show: false,
        is_update_show: false, detail_span: 1,
      },
    ] as TableField[]
    const metadata: RuntimeTableMetadata = {
      id: 1,
      table_name: '演示表',
      table_code: 'demo',
      table_type: SysTableType.SYSTEM,
      master_detail_mode: SysMasterDetailMode.AUTO,
      form_open_mode: SysFormOpenMode.AUTO,
      detail_open_mode: SysDetailOpenMode.AUTO,
      table_fields: fields,
      table_relations: [],
    }
    const loader = vi.fn().mockResolvedValue({ success: true, data: metadata })
    const state = useRuntimeTableMetadata('demo', { loader })

    await state.loadMetadata()

    expect(loader).toHaveBeenCalledWith('demo')
    expect(state.fields.value.map((field) => field.field_code)).toEqual(['status', 'name', 'code'])
    expect(state.quickSearchFields.value.map((field) => field.field_code)).toEqual(['name', 'code'])
    expect(state.quickSearchPlaceholder.value).toBe('搜索名称、编码')
    expect(state.advancedSearchFields.value.map((field) => field.field_code)).toEqual(['status'])
    expect(state.formFields.value.map((field) => field.field_code)).toEqual(['name', 'code'])
    expect(state.detailFields.value.map((field) => field.field_code)).toEqual(['status', 'name', 'code'])
  })
})
