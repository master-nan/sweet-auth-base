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
      global: { stubs: { QBtn: QBtnStub, QTooltip: true, QSpace: true, QSeparator: true } },
    })

    expect(wrapper.find('[data-testid="quick"]').exists()).toBe(true)
    await wrapper.find('[data-testid="refresh"]').trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })

  it('lays out query scheme slots without owning their API behavior', () => {
    const wrapper = mount(StandardTableToolbar, {
      slots: {
        'scheme-selector': '<span data-testid="scheme">方案</span>',
        'quick-presets': '<span data-testid="preset">本月</span>',
        'save-scheme': '<span data-testid="save">保存方案</span>',
      },
      global: { stubs: { QBtn: QBtnStub, QTooltip: true, QSpace: true, QSeparator: true } },
    })
    expect(wrapper.find('[data-testid="scheme"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="preset"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="save"]').exists()).toBe(true)
  })

  it('keeps query, business, and platform actions in a stable visual order', () => {
    const wrapper = mount(StandardTableToolbar, {
      slots: {
        'scheme-selector': '<span>方案</span>',
        'quick-search': '<span>查询</span>',
        'save-scheme': '<span>保存方案</span>',
        'right-actions': '<span>业务动作</span>',
        'column-selector': '<span>列设置</span>',
      },
      global: { stubs: { QBtn: QBtnStub, QTooltip: true, QSpace: true, QSeparator: true } },
    })
    const text = wrapper.text()
    expect(text.indexOf('方案')).toBeLessThan(text.indexOf('查询'))
    expect(text.indexOf('查询')).toBeLessThan(text.indexOf('保存方案'))
    expect(text.indexOf('保存方案')).toBeLessThan(text.indexOf('业务动作'))
    expect(text.indexOf('业务动作')).toBeLessThan(text.indexOf('列设置'))
  })
})
