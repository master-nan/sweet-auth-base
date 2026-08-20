import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ExpressionLogic, ExpressionType, SysTableFieldType } from 'src/types/enum'
import { QuerySchemeBindingKind } from 'src/modules/query-scheme/types'

vi.mock('src/stores/dict', () => ({ useDictStore: () => ({ getDictLabel: () => '异常' }) }))
vi.mock('src/api/services/runtime-relation', () => ({ queryRuntimeRelationOptions: vi.fn() }))
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
})
