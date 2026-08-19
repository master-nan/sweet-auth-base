import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { QuerySchemeType } from 'src/modules/query-scheme/types'

vi.mock('quasar', () => ({ useQuasar: () => ({ screen: { lt: { sm: false } } }) }))
import QuerySchemeSaveDialog from './QuerySchemeSaveDialog.vue'

const SlotStub = defineComponent({ setup(_, { slots }) { return () => h('div', slots.default?.()) } })
const InputStub = defineComponent({ props: { modelValue: String }, emits: ['update:modelValue'], setup(props) { return () => h('input', { value: props.modelValue }) } })
const ButtonStub = defineComponent({ props: { label: String }, emits: ['click'], setup(props, { emit }) { return () => h('button', { onClick: () => emit('click') }, props.label) } })

describe('QuerySchemeSaveDialog', () => {
  it('offers update and save-as only for an owned personal source', async () => {
    const wrapper = mount(QuerySchemeSaveDialog, { props: { modelValue: true, source: { id: 1, name: '本人方案', type: QuerySchemeType.PERSONAL, revision: 2, is_default: false } }, global: { stubs: { QDialog: SlotStub, QCard: SlotStub, QCardSection: SlotStub, QCardActions: SlotStub, QSeparator: true, QSpace: true, QBtn: ButtonStub, QInput: InputStub, QCheckbox: true, QTooltip: true } } })
    expect(wrapper.text()).toContain('保存修改')
    expect(wrapper.text()).toContain('另存为')
    await wrapper.findAll('button').find((button) => button.text() === '保存修改')!.trigger('click')
    expect(wrapper.emitted('save')?.[0]?.[0]).toMatchObject({ name: '本人方案', saveAs: false })
  })

  it('does not allow direct shared scheme updates', () => {
    const wrapper = mount(QuerySchemeSaveDialog, { props: { modelValue: true, source: { id: 2, name: '公共方案', type: QuerySchemeType.PUBLIC, revision: 1, is_default: false } }, global: { stubs: { QDialog: SlotStub, QCard: SlotStub, QCardSection: SlotStub, QCardActions: SlotStub, QSeparator: true, QSpace: true, QBtn: ButtonStub, QInput: InputStub, QCheckbox: true, QTooltip: true } } })
    expect(wrapper.text()).not.toContain('保存修改')
    expect(wrapper.text()).toContain('另存为我的方案')
  })
})
