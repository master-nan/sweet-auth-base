import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ExpressionLogic, ExpressionType, SysTableFieldType } from 'src/types/enum'
import { QuerySchemeBindingKind } from 'src/modules/query-scheme/types'

vi.mock('src/stores/dict', () => ({ useDictStore: () => ({ getDictLabel: () => '异常' }) }))
vi.mock('src/api/services/runtime-relation', () => ({ queryRuntimeRelationOptions: vi.fn() }))
import { queryRuntimeRelationOptions } from 'src/api/services/runtime-relation'
import QuerySchemePreview from './QuerySchemePreview.vue'

const BadgeStub = defineComponent({ props: { label: String }, setup(props) { return () => h('span', props.label) } })

describe('QuerySchemePreview', () => {
  it('uses metadata, operator and binding labels instead of field codes', () => {
    const wrapper = mount(QuerySchemePreview, { props: { fields: [{ field_code: 'created_at', field_name: '创建时间', field_type: SysTableFieldType.DATE, dict_code: '' }] as never, payload: { expressions: [{ logic: ExpressionLogic.AND, rules: [{ field: 'created_at', expression_type: ExpressionType.BETWEEN, value: [null, null] }], nested: [] }], quick_query: { keyword: '' }, order: { field: '', is_asc: false }, bindings: [{ pointer: '/expressions/0/rules/0/value/0', kind: QuerySchemeBindingKind.START_OF_MONTH }, { pointer: '/expressions/0/rules/0/value/1', kind: QuerySchemeBindingKind.END_OF_MONTH }] } }, global: { stubs: { QBadge: BadgeStub } } })
    expect(wrapper.text()).toContain('创建时间 区间 本月开始 至 本月结束')
    expect(wrapper.text()).not.toContain('created_at')
  })

  it('renders the empty state while an editable scheme payload is initializing', () => {
    const wrapper = mount(QuerySchemePreview, {
      props: {
        fields: [],
        payload: {
          expressions: null,
          quick_query: { keyword: '' },
          order: { field: '', is_asc: false },
          bindings: null,
        } as never,
      },
      global: { stubs: { QBadge: BadgeStub } },
    })

    expect(wrapper.text()).toContain('未设置高级查询条件')
  })

  it('resolves relation values through selected_values for the preview', async () => {
    vi.mocked(queryRuntimeRelationOptions).mockResolvedValue({
      items: [{ value: '9527', label: '华东客户' }],
      total: 1,
    })
    const wrapper = mount(QuerySchemePreview, {
      props: {
        menuId: 205,
        fields: [
          {
            id: 9,
            field_code: 'customer_id',
            field_name: '客户',
            field_type: SysTableFieldType.BIGINT,
            relation: { target_table_code: 'customer', value_field: 'id', display_field: 'name' },
          },
        ] as never,
        payload: {
          expressions: [
            {
              logic: ExpressionLogic.AND,
              rules: [
                { field: 'customer_id', expression_type: ExpressionType.EQ, value: '9527' },
              ],
              nested: [],
            },
          ],
          quick_query: { keyword: '' },
          order: { field: '', is_asc: false },
          bindings: [],
        },
      },
      global: { stubs: { QBadge: BadgeStub } },
    })
    await flushPromises()

    expect(queryRuntimeRelationOptions).toHaveBeenCalledWith(
      9,
      expect.objectContaining({ menu_id: 205, selected_values: ['9527'] }),
    )
    expect(wrapper.text()).toContain('客户 等于 华东客户')
    expect(wrapper.text()).not.toContain('9527')
  })

  it('renders nested boolean expressions as group and condition tree nodes', () => {
    const wrapper = mount(QuerySchemePreview, {
      props: {
        fields: [
          { field_code: 'username', field_name: '用户名' },
          { field_code: 'language', field_name: '语言' },
          { field_code: 'status', field_name: '状态' },
        ] as never,
        payload: {
          expressions: [
            {
              logic: ExpressionLogic.OR,
              rules: [
                { field: 'username', expression_type: ExpressionType.LIKE, value: 'admin' },
                { field: 'language', expression_type: ExpressionType.EQ, value: 'zh-CN' },
              ],
              nested: [
                {
                  logic: ExpressionLogic.AND,
                  rules: [
                    { field: 'status', expression_type: ExpressionType.EQ, value: 'enabled' },
                  ],
                  nested: [],
                },
              ],
            },
          ],
          quick_query: { keyword: '' },
          order: { field: '', is_asc: false },
          bindings: [],
        },
      },
      global: { stubs: { QBadge: BadgeStub } },
    })

    expect(wrapper.findAll('.query-preview-tree__line--group')).toHaveLength(2)
    expect(wrapper.findAll('.query-preview-tree__line--rule')).toHaveLength(3)
    expect(wrapper.text()).toContain('OR满足任一条件')
    expect(wrapper.text()).toContain('AND满足全部条件')
    expect(wrapper.text()).toContain('用户名 包含 admin')
    expect(wrapper.findAll('.query-preview-tree__branch').some((branch) => branch.text().includes('└─'))).toBe(true)
  })
})
