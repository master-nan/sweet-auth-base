import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StatusChip from './StatusChip.vue'

const QChipStub = defineComponent({
  props: { color: String, icon: String },
  setup(props, { slots }) {
    return () => h('span', { 'data-color': props.color, 'data-icon': props.icon }, slots.default?.())
  },
})

describe('StatusChip', () => {
  it('renders only the domain-provided display mapping', () => {
    const wrapper = mount(StatusChip, {
      props: { label: '已启用', color: 'positive', icon: 'check' },
      global: { stubs: { QChip: QChipStub } },
    })
    expect(wrapper.text()).toContain('已启用')
    expect(wrapper.find('span').attributes()).toMatchObject({
      'data-color': 'positive',
      'data-icon': 'check',
    })
  })
})
