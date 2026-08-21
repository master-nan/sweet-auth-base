import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import QuerySchemeSelector from './QuerySchemeSelector.vue'
import { QuerySchemeType, QuerySchemeValidationStatus } from 'src/modules/query-scheme/types'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const ItemStub = defineComponent({
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, slots.default?.())
  },
})

describe('QuerySchemeSelector', () => {
  it('groups runtime summaries and opens management without loading payloads', async () => {
    const scheme = {
      id: 1,
      name: '本月异常',
      type: QuerySchemeType.PERSONAL,
      is_default: true,
      status: QuerySchemeValidationStatus.VALID,
    }
    const wrapper = mount(QuerySchemeSelector, {
      props: { schemes: [scheme], currentLabel: '本月异常（已修改）' },
      global: {
        stubs: {
          QBtnDropdown: SlotStub,
          QList: SlotStub,
          QItem: ItemStub,
          QItemLabel: SlotStub,
          QItemSection: SlotStub,
          QIcon: true,
          QTooltip: true,
          QSeparator: true,
        },
      },
    })
    expect(wrapper.text()).toContain('本月异常')
    expect(wrapper.text()).toContain('我的方案')
    expect(wrapper.text()).not.toContain('公共方案')
    expect(wrapper.text()).not.toContain('角色方案')
    const items = wrapper.findAll('button')
    await items[0]!.trigger('click')
    await items.at(-1)!.trigger('click')
    expect(wrapper.emitted('select')?.[0]?.[0]).toEqual(scheme)
    expect(wrapper.emitted('manage')).toHaveLength(1)
  })

  it('distinguishes an empty selector from a failed runtime request', async () => {
    const emptyWrapper = mount(QuerySchemeSelector, {
      props: { schemes: [] },
      global: {
        stubs: {
          QBtnDropdown: SlotStub,
          QList: SlotStub,
          QItem: ItemStub,
          QItemLabel: SlotStub,
          QItemSection: SlotStub,
          QIcon: true,
          QTooltip: true,
          QSeparator: true,
        },
      },
    })
    expect(emptyWrapper.text()).toContain('暂无已保存方案')

    const failedWrapper = mount(QuerySchemeSelector, {
      props: { schemes: [], loadError: 'network error' },
      global: {
        stubs: {
          QBtnDropdown: SlotStub,
          QList: SlotStub,
          QItem: ItemStub,
          QItemLabel: SlotStub,
          QItemSection: SlotStub,
          QIcon: true,
          QTooltip: true,
          QSeparator: true,
        },
      },
    })
    const retry = failedWrapper.findAll('button').find((item) => item.text().includes('点击重试'))
    await retry!.trigger('click')
    expect(failedWrapper.text()).toContain('查询方案加载失败')
    expect(failedWrapper.emitted('retry')).toHaveLength(1)
  })

  it('moves save actions into the selector and reflects the current source', async () => {
    const wrapper = mount(QuerySchemeSelector, {
      props: {
        schemes: [],
        dirty: true,
        source: {
          id: 3,
          name: '个人方案',
          type: QuerySchemeType.PERSONAL,
          revision: 2,
          is_default: false,
        },
      },
      global: {
        stubs: {
          QBtnDropdown: SlotStub,
          QList: SlotStub,
          QItem: ItemStub,
          QItemLabel: SlotStub,
          QItemSection: SlotStub,
          QIcon: true,
          QTooltip: true,
          QSeparator: true,
        },
      },
    })

    expect(wrapper.text()).toContain('保存当前方案修改')
    const save = wrapper.findAll('button').find((item) => item.text().includes('保存当前方案修改'))
    await save!.trigger('click')
    expect(wrapper.emitted('save-current')).toHaveLength(1)
  })

  it('keeps a 64-character current name bounded and discoverable', () => {
    const currentLabel = '长'.repeat(64)
    const wrapper = mount(QuerySchemeSelector, {
      props: { schemes: [], currentLabel },
      global: {
        stubs: {
          QBtnDropdown: SlotStub,
          QList: SlotStub,
          QItem: ItemStub,
          QItemLabel: SlotStub,
          QItemSection: SlotStub,
          QIcon: true,
          QTooltip: true,
          QSeparator: true,
        },
      },
    })

    expect(wrapper.find('.query-scheme-selector').attributes('title')).toBe(currentLabel)
  })
})
