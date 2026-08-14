import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StandardTableToolbar from './StandardTableToolbar.vue'

const QBtnStub = defineComponent({
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { 'data-testid': 'refresh', onClick: () => emit('click') }, slots.default?.())
  },
})

describe('StandardTableToolbar', () => {
  it('always exposes refresh as a platform view action without business buttons', async () => {
    const wrapper = mount(StandardTableToolbar, {
      slots: { 'quick-search': '<input data-testid="quick" />' },
      global: { stubs: { QBtn: QBtnStub, QTooltip: true, QSpace: true } },
    })

    expect(wrapper.find('[data-testid="quick"]').exists()).toBe(true)
    await wrapper.find('[data-testid="refresh"]').trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })
})
