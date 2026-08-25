import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import TableColumnSelector from './TableColumnSelector.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const ButtonStub = defineComponent({
  props: { label: String },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        { type: 'button', onClick: () => emit('click') },
        props.label || slots.default?.(),
      )
  },
})

const ItemStub = defineComponent({
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, slots.default?.())
  },
})

const mountSelector = () =>
  mount(TableColumnSelector, {
    props: {
      modelValue: ['status', 'name'],
      columns: [
        { name: 'status', label: '状态', field: 'status' },
        { name: 'name', label: '应用名称', field: 'name' },
        { name: 'app_key', label: '应用Key', field: 'app_key' },
      ],
    },
    global: {
      stubs: {
        QBtn: ButtonStub,
        QBadge: defineComponent({
          props: { label: [String, Number] },
          setup(props) {
            return () => h('span', { 'data-testid': 'count' }, String(props.label))
          },
        }),
        QTooltip: true,
        QMenu: SlotStub,
        QInput: SlotStub,
        QIcon: true,
        QList: SlotStub,
        QItem: ItemStub,
        QItemSection: SlotStub,
        QItemLabel: SlotStub,
        QCheckbox: true,
      },
    },
  })

describe('TableColumnSelector', () => {
  it('shows the visible column count and all available columns', () => {
    const wrapper = mountSelector()

    expect(wrapper.get('[data-testid="count"]').text()).toBe('2')
    expect(wrapper.text()).toContain('状态')
    expect(wrapper.text()).toContain('应用名称')
    expect(wrapper.text()).toContain('应用Key')
  })

  it('toggles columns while preserving table column order', async () => {
    const wrapper = mountSelector()
    const statusItem = wrapper
      .findAllComponents(ItemStub)
      .find((item) => item.text().includes('状态'))

    await statusItem!.trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual(['name'])
  })

  it('supports select all and restoring the initial visible columns', async () => {
    const wrapper = mountSelector()

    await wrapper
      .findAllComponents(ButtonStub)
      .find((button) => button.text() === '全选')!
      .trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual(['status', 'name', 'app_key'])

    await wrapper.setProps({ modelValue: ['app_key'] })
    await wrapper
      .findAllComponents(ButtonStub)
      .find((button) => button.text() === '恢复默认')!
      .trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual(['status', 'name'])
  })
})
