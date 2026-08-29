import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SweetDateTimePicker from './SweetDateTimePicker.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const QInputStub = defineComponent({
  props: { modelValue: String, label: String },
  setup(props, { slots }) {
    return () => h('div', { 'data-value': props.modelValue }, slots.append?.())
  },
})

const QDateStub = defineComponent({
  name: 'QDate',
  props: { mask: String, defaultView: String, modelValue: String },
  template: '<div />',
})

const QBtnStub = defineComponent({
  props: { label: String, ariaLabel: String },
  emits: ['click'],
  setup(props, { emit }) {
    return () => h('button', { 'aria-label': props.ariaLabel, onClick: () => emit('click') }, props.label)
  },
})

const mountPicker = (type: 'date' | 'time' | 'datetime' | 'year' | 'year-month', modelValue = '') =>
  mount(SweetDateTimePicker, {
    props: { type, modelValue },
    global: {
      stubs: {
        QInput: QInputStub,
        QIcon: SlotStub,
        QPopupProxy: SlotStub,
        QDate: QDateStub,
        QSeparator: true,
        QBtn: QBtnStub,
      },
    },
  })

describe('SweetDateTimePicker', () => {
  it.each([
    ['date', 'YYYY-MM-DD', 'Calendar'],
    ['year-month', 'YYYY-MM', 'Months'],
    ['year', 'YYYY', 'Years'],
  ] as const)('uses the controlled Quasar calendar for %s values', (type, mask, defaultView) => {
    const date = mountPicker(type).findComponent(QDateStub)
    expect(date.props('mask')).toBe(mask)
    expect(date.props('defaultView')).toBe(defaultView)
  })

  it('changes datetime values through explicit hour controls', async () => {
    const wrapper = mountPicker('datetime', '2026-08-20 04:18:00')
    await wrapper.get('button[aria-label="时加一"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['2026-08-20 05:18:00'])
  })

  it('renders time-only values without a calendar', () => {
    const wrapper = mountPicker('time', '12:30:45')
    expect(wrapper.findComponent(QDateStub).exists()).toBe(false)
    expect(wrapper.find('[data-value="12:30:45"]').exists()).toBe(true)
  })
})
