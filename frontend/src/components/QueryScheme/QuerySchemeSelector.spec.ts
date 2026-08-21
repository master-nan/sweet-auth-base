import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import QuerySchemeSelector from './QuerySchemeSelector.vue'
import {
  QuerySchemeType,
  QuerySchemeValidationStatus,
  type QuerySchemeSource,
  type QuerySchemeSummary,
} from 'src/modules/query-scheme/types'

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
const ButtonStub = defineComponent({
  props: { label: String },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, props.label || slots.default?.())
  },
})
const DialogStub = defineComponent({
  props: { modelValue: Boolean },
  setup(props, { slots }) {
    return () => (props.modelValue ? h('div', { 'data-testid': 'confirm' }, slots.default?.()) : null)
  },
})
const IconStub = defineComponent({
  props: { name: String, color: String },
  setup(props, { slots }) {
    return () =>
      h('span', { 'data-icon': props.name, 'data-color': props.color }, slots.default?.())
  },
})

type SelectorProps = {
  schemes: QuerySchemeSummary[]
  currentLabel?: string
  dirty?: boolean
  loadError?: string
  source?: QuerySchemeSource | null
}

const mountSelector = (props: SelectorProps) =>
  mount(QuerySchemeSelector, {
    props,
    global: {
      stubs: {
        QBtnDropdown: SlotStub,
        QList: SlotStub,
        QItem: ItemStub,
        QItemLabel: SlotStub,
        QItemSection: SlotStub,
        QDialog: DialogStub,
        QCard: SlotStub,
        QCardSection: SlotStub,
        QCardActions: SlotStub,
        QBtn: ButtonStub,
        QIcon: IconStub,
        QTooltip: SlotStub,
        QSeparator: true,
      },
    },
  })

describe('QuerySchemeSelector', () => {
  it('groups only populated sections and opens management without loading payloads', async () => {
    const scheme = {
      id: 1,
      name: '本月异常',
      type: QuerySchemeType.PERSONAL,
      is_default: true,
      status: QuerySchemeValidationStatus.VALID,
    }
    const wrapper = mountSelector({ schemes: [scheme], currentLabel: '本月异常（已修改）' })

    expect(wrapper.text()).toContain('我的方案')
    expect(wrapper.text()).not.toContain('公共方案')
    expect(wrapper.text()).not.toContain('角色方案')
    const items = wrapper.findAll('button')
    await items[0]!.trigger('click')
    await items.at(-1)!.trigger('click')
    expect(wrapper.emitted('select')?.[0]?.[0]).toEqual(scheme)
    expect(wrapper.emitted('manage')).toHaveLength(1)
  })

  it('distinguishes no schemes from a failed runtime request', async () => {
    const emptyWrapper = mountSelector({ schemes: [] })
    expect(emptyWrapper.text()).toContain('暂无已保存方案')

    const failedWrapper = mountSelector({ schemes: [], loadError: 'network error' })
    const retry = failedWrapper.findAll('button').find((item) => item.text().includes('点击重试'))
    await retry!.trigger('click')
    expect(failedWrapper.text()).toContain('查询方案加载失败')
    expect(failedWrapper.emitted('retry')).toHaveLength(1)
  })

  it('moves save actions into the selector and reflects the current source', async () => {
    const wrapper = mountSelector({
      schemes: [],
      dirty: true,
      source: {
        id: 3,
        name: '个人方案',
        type: QuerySchemeType.PERSONAL,
        revision: 2,
        is_default: false,
      },
    })

    const save = wrapper.findAll('button').find((item) => item.text().includes('保存当前方案修改'))
    await save!.trigger('click')
    expect(wrapper.emitted('save-current')).toHaveLength(1)
  })

  it('requires confirmation before switching away from dirty conditions', async () => {
    const scheme = {
      id: 4,
      name: '待审核',
      type: QuerySchemeType.PERSONAL,
      is_default: false,
      status: QuerySchemeValidationStatus.VALID,
    }
    const wrapper = mountSelector({ schemes: [scheme], dirty: true })

    await wrapper.findAll('button')[0]!.trigger('click')
    expect(wrapper.emitted('select')).toBeUndefined()
    expect(wrapper.find('[data-testid="confirm"]').text()).toContain('未保存修改')
    await wrapper.findAll('button').find((item) => item.text() === '继续切换')!.trigger('click')
    expect(wrapper.emitted('select')?.[0]?.[0]).toEqual(scheme)
  })

  it('marks degraded and invalid schemes without exposing technical details', () => {
    const wrapper = mountSelector({
      schemes: [
        {
          id: 5,
          name: '需修复',
          type: QuerySchemeType.PUBLIC,
          is_default: false,
          status: QuerySchemeValidationStatus.DEGRADED,
        },
        {
          id: 6,
          name: '不可用',
          type: QuerySchemeType.ROLE,
          is_default: false,
          status: QuerySchemeValidationStatus.INVALID,
        },
      ],
    })

    expect(wrapper.find('[data-icon="warning_amber"][data-color="warning"]').exists()).toBe(true)
    expect(wrapper.find('[data-icon="warning_amber"][data-color="negative"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('field_code')
  })

  it('keeps a 64-character current name bounded and discoverable', () => {
    const currentLabel = '长'.repeat(64)
    const wrapper = mountSelector({ schemes: [], currentLabel })

    expect(wrapper.find('.query-scheme-selector').attributes('title')).toBe(currentLabel)
  })
})
