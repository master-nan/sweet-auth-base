import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import QueryQuickPresets from './QueryQuickPresets.vue'
import { QuerySchemeBindingKind, type QuerySchemePayloadV1 } from 'src/modules/query-scheme/types'

const ButtonStub = defineComponent({ props: { label: String }, emits: ['click'], setup(props, { emit }) { return () => h('button', { onClick: () => emit('click') }, props.label) } })

describe('QueryQuickPresets', () => {
  it('only exposes date presets with an explicit quick date field and emits bindings', async () => {
    const wrapper = mount(QueryQuickPresets, { props: { config: { menu_id: 1, scope_code: 'demo', scope_label: '示例', table_code: 'demo', quick_date_field: 'created_at', quick_presets: [], virtual_sort_fields: [], dynamic_binding_kinds: Object.values(QuerySchemeBindingKind) } }, global: { stubs: { QBtn: ButtonStub } } })
    expect(wrapper.text()).toContain('本月')
    await wrapper.findAll('button').find((button) => button.text() === '本月')!.trigger('click')
    const payload = wrapper.emitted('apply')?.[0]?.[0] as QuerySchemePayloadV1
    expect(payload.bindings.map((binding: { kind: string }) => binding.kind)).toEqual(['START_OF_MONTH', 'END_OF_MONTH'])
    expect(payload.expressions[0]?.rules[0]?.field).toBe('created_at')
  })

  it('does not guess a date field', () => {
    const wrapper = mount(QueryQuickPresets, { props: { config: { menu_id: 1, scope_code: 'demo', scope_label: '示例', table_code: 'demo', quick_presets: [], virtual_sort_fields: [], dynamic_binding_kinds: [] } }, global: { stubs: { QBtn: ButtonStub } } })
    expect(wrapper.text()).toBe('')
  })
})
