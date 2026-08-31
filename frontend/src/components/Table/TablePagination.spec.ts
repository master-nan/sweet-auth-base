import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import TablePagination from './TablePagination.vue'
import { testI18n } from 'src/test/setup'

const ButtonStub = defineComponent({
  inheritAttrs: false,
  props: { icon: String, disable: Boolean },
  emits: ['click'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          type: 'button',
          disabled: props.disable,
          onClick: () => emit('click'),
        },
        [props.icon, slots.default?.()],
      )
  },
})

const InputStub = defineComponent({
  inheritAttrs: false,
  props: { modelValue: String },
  emits: ['update:modelValue', 'blur'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h('label', [
        h('input', {
          ...attrs,
          value: props.modelValue,
          onInput: (event: Event) =>
            emit('update:modelValue', (event.target as HTMLInputElement).value),
          onBlur: () => emit('blur'),
        }),
        slots.append?.(),
      ])
  },
})

const SelectStub = defineComponent({
  inheritAttrs: false,
  props: { modelValue: Number, options: Array },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          type: 'button',
          onClick: () => emit('update:modelValue', 50),
        },
        [String(props.modelValue), slots.append?.()],
      )
  },
})

const mountPagination = (
  page = 1001,
  pageSize = 20,
  total = 25_000,
  locale: 'en-US' | 'zh-CN' = 'zh-CN',
) => {
  testI18n.global.locale.value = locale
  return mount(TablePagination, {
    props: { page, pageSize, total },
    global: {
      stubs: {
        QBtn: ButtonStub,
        QTooltip: true,
        QInput: InputStub,
        QSelect: SelectStub,
        QSeparator: true,
        QItemLabel: true,
        QItem: true,
        QItemSection: true,
      },
    },
  })
}

describe('TablePagination', () => {
  it('shows the total count and fixed current-page summary', () => {
    const wrapper = mountPagination()

    expect(wrapper.text()).toContain('共 25,000 条')
    expect(wrapper.text()).toContain('/ 1,250 页')
    expect(wrapper.get('input[aria-label="当前页"]').element).toHaveProperty('value', '1001')
    expect(wrapper.findComponent(SelectStub).props('options')).toEqual([20, 50, 100, 200])
  })

  it('moves to the first, previous, next and last pages', async () => {
    const wrapper = mountPagination()

    await wrapper.get('button[aria-label="首页"]').trigger('click')
    await wrapper.get('button[aria-label="下一页"]').trigger('click')
    await wrapper.get('button[aria-label="末页"]').trigger('click')
    await wrapper.get('button[aria-label="上一页"]').trigger('click')

    expect(wrapper.emitted('update:page')?.map(([page]) => page)).toEqual([1, 2, 1250, 1249])
  })

  it('updates totals and accessible labels when the locale changes', () => {
    const wrapper = mountPagination(1, 20, 25_000, 'en-US')

    expect(wrapper.text()).toContain('25,000 items')
    expect(wrapper.text()).toContain('/ 1,250 pages')
    expect(wrapper.get('input[aria-label="Current page"]').attributes('aria-label')).toBe(
      'Current page',
    )
    expect(wrapper.get('button[aria-label="Items per page"]').attributes('aria-label')).toBe(
      'Items per page',
    )
  })

  it('clamps an entered page and resets the page when page size changes', async () => {
    const wrapper = mountPagination()
    const input = wrapper.get('input[aria-label="当前页"]')

    await input.setValue('99999')
    await input.trigger('blur')
    expect(wrapper.emitted('update:page')?.at(-1)).toEqual([1250])

    await wrapper.get('button[aria-label="每页条数"]').trigger('click')
    expect(wrapper.emitted('update:pageSize')?.at(-1)).toEqual([50])
    expect(wrapper.emitted('update:page')?.at(-1)).toEqual([1])
  })
})
