import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DetailFieldGrid from './DetailFieldGrid.vue'

const ChipStub = defineComponent({
  props: {
    color: String,
    textColor: String,
    outline: Boolean,
  },
  setup(props, { slots }) {
    return () =>
      h(
        'span',
        {
          'data-color': props.color,
          'data-text-color': props.textColor,
          'data-outline': String(props.outline),
        },
        slots.default?.(),
      )
  },
})

describe('DetailFieldGrid', () => {
  it('keeps status chips readable when an explicit text color is provided', () => {
    const wrapper = mount(DetailFieldGrid, {
      props: {
        items: [
          {
            label: '会话状态',
            value: '已强制下线',
            chip: true,
            color: 'grey-3',
            textColor: 'grey-9',
            outline: true,
          },
        ],
      },
      global: { stubs: { QChip: ChipStub } },
    })

    const chip = wrapper.get('[data-color="grey-3"]')
    expect(chip.attributes('data-text-color')).toBe('grey-9')
    expect(chip.attributes('data-outline')).toBe('true')
    expect(chip.text()).toBe('已强制下线')
  })

  it('uses solid white status chips in card layouts by default', () => {
    const wrapper = mount(DetailFieldGrid, {
      props: {
        variant: 'card',
        items: [{ label: '状态', value: '成功', chip: true, color: 'positive' }],
      },
      global: { stubs: { QChip: ChipStub } },
    })

    const chip = wrapper.get('[data-color="positive"]')
    expect(chip.attributes('data-text-color')).toBe('white')
    expect(chip.attributes('data-outline')).toBe('false')
  })
})
