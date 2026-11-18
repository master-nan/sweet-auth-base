import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SysTableRelationType, SysTableFieldType, SysTableFieldInputType } from '@/types/enum'
import type { TableRelation } from '@/api/services/sys-table'
import DocumentDetails from './DocumentDetails.vue'

const mocks = vi.hoisted(() => ({ rows: vi.fn(), schema: vi.fn() }))
vi.mock('@/api/services/generalization', () => ({
  useGeneralizationApi: () => ({ getDocumentRows: mocks.rows }),
}))
vi.mock('@/api/services/sys-table', () => ({
  useTableApi: () => ({ queryRuntimeTableByCode: mocks.schema }),
}))
vi.mock('@/stores/user', () => ({
  useUserStore: () => ({
    menus: ['goods', 'fees'].map((table_code, i) => ({
      table_code,
      id: i + 100,
      menu_buttons: ['create', 'update', 'delete', 'detail'].map((event_action) => ({
        is_button: true,
        event_action,
      })),
    })),
  }),
}))
vi.mock('@/api/services/sys-dict', () => ({ useDictApi: () => ({}) }))
vi.mock('@/boot/axios', () => ({ instance: {} }))
const tabs = ['goods', 'fees'].map((related_table_code, i) => ({
  id: i + 1,
  related_table_code,
  relation_type: SysTableRelationType.ONE_TO_MANY,
  document_tab: i ? '费用明细' : '商品明细',
  foreign_key: 'order_id',
  amount_field: 'amount',
})) as TableRelation[]
const field = {
  id: 3,
  field_code: 'amount',
  field_name: '金额',
  field_type: SysTableFieldType.DECIMAL,
  input_type: SysTableFieldInputType.INPUT_NUMBER,
  is_insert_show: true,
  is_update_show: true,
  detail_visible: true,
  numeric_precision: 16,
  numeric_scale: 2,
  binding: '',
  linkage_config: '',
  dict_code: '',
}

const Button = defineComponent({
  props: ['label', 'icon'],
  template: '<button type="button">{{ label }}<slot /></button>',
})
const Input = defineComponent({
  props: ['modelValue'],
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        value: props.modelValue,
        onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
      })
  },
})
const Tabs = defineComponent({
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: '<nav><slot /></nav>',
})
function render(readonly = false, masterId = 7) {
  return mount(DocumentDetails, {
    props: { relations: tabs, tableCode: 'orders', menuId: 44, masterId, readonly },
    global: {
      plugins: [createPinia()],
      stubs: {
        QTabs: Tabs,
        QTab: { props: ['label'], template: '<span>{{ label }}</span>' },
        QMarkupTable: { template: '<div><table><slot /></table></div>' },
        QBtn: Button,
        QInput: Input,
        QTooltip: true,
        QLinearProgress: true,
        QBanner: { template: '<aside><slot /></aside>' },
        QForm: true,
        QSpinner: true,
        QIcon: true,
      },
    },
  })
}
beforeEach(() => {
  mocks.schema.mockResolvedValue({ success: true, data: { table_fields: [field] } })
  mocks.rows.mockImplementation((_code, _id, relation) =>
    Promise.resolve({
      success: true,
      data: [{ id: relation + 10, amount: relation === 1 ? '2.10' : '0.20' }],
    }),
  )
})
describe('通用单据明细', () => {
  function mixedSchema() {
    const flag = (code: string, insert: boolean, update: boolean, detail?: boolean, extra = {}) => ({
      ...field, field_code: code, field_name: code, sequence: 1, is_list_show: false,
      is_insert_show: insert, is_update_show: update, detail_visible: detail, ...extra,
    })
    mocks.schema.mockResolvedValue({ success: true, data: { table_fields: [
      flag('listOnly', false, false, undefined, { is_list_show: true }),
      flag('onInsert', true, false, false),
      flag('onUpdate', false, true, false),
      flag('locked', true, true, true, { is_update_readonly: true }),
      flag('detailOnly', false, false, true),
    ] } })
    mocks.rows.mockResolvedValue({ success: true, data: [{
      id: 81, listOnly: 'LIST-VALUE', onInsert: 'INSERT-SECRET', onUpdate: '2',
      locked: '79', detailOnly: 'DETAIL-VALUE',
    }] })
  }
  it('同一运单中已存行和新增行分别选列，隐藏单元格不回显，只读不参与修改', async () => {
    mixedSchema()
    const wrapper = render()
    await flushPromises()
    const first = wrapper.get('.document-section')
    const heads = () => first.get('thead').text()
    expect(heads()).not.toContain('listOnly')
    expect(heads()).not.toContain('onInsert')
    expect(heads()).not.toContain('detailOnly')
    expect(first.get('tbody').text()).toContain('79')
    expect(first.get('tbody').findAll('input')).toHaveLength(1)
    await first.get('tbody input').setValue('4')
    await wrapper.get('.document-heading button').trigger('click')
    expect(heads()).toContain('onInsert')
    const [oldRow, newRow] = first.findAll('tbody tr')
    expect(oldRow!.text()).not.toContain('INSERT-SECRET')
    expect(oldRow!.findAll('input')).toHaveLength(1)
    expect(newRow!.findAll('input')).toHaveLength(2)
    await newRow!.get('input').setValue('7')
    expect(wrapper.vm.changes()[0]).toMatchObject({
      added: [expect.objectContaining({ onInsert: '7' })],
      updated: [{ id: 81, data: { onUpdate: '4' } }],
    })
    expect(wrapper.vm.changes()[0]!.added[0]).not.toHaveProperty('onUpdate')
    await newRow!.get('[aria-label="删除明细"]').trigger('click')
    expect(heads()).not.toContain('onInsert')
    wrapper.unmount()
  })
  it('新增单据只出新增列；详情严格按详情可见设置，不以列表勾选兜底', async () => {
    mixedSchema()
    const create = render(false, 0)
    await flushPromises()
    const header = create.get('thead').text()
    expect(header).toContain('onInsert')
    expect(header).not.toContain('onUpdate')
    expect(header).not.toContain('listOnly')
    create.unmount()
    const detail = render(true)
    await flushPromises()
    expect(detail.get('thead').text()).toContain('detailOnly')
    expect(detail.get('thead').text()).not.toContain('listOnly')
    expect(detail.text()).not.toContain('LIST-VALUE')
    expect(detail.text()).not.toContain('INSERT-SECRET')
    expect(detail.text()).toContain('DETAIL-VALUE')
    expect(detail.find('tbody input').exists()).toBe(false)
    detail.unmount()
  })
  it('分开汇总各页签，编辑一项不改变另一项金额', async () => {
    const wrapper = render()
    await flushPromises()
    expect(wrapper.findAll('.document-totals')).toHaveLength(1)
    expect(wrapper.get('.document-totals b').text()).toBe('2.10')
    await wrapper.get('.document-section input').setValue('7.99')
    expect(wrapper.get('.document-totals b').text()).toBe('7.99')
    wrapper.findComponent(Tabs).vm.$emit('update:modelValue', 2)
    await flushPromises()
    expect(wrapper.get('.document-totals b').text()).toBe('0.20')
    wrapper.findComponent(Tabs).vm.$emit('update:modelValue', 1)
    await flushPromises()
    expect(wrapper.get('.document-totals b').text()).toBe('7.99')
    wrapper.unmount()
  })
  it('混合了金额隐藏行时不展示不完整的小计，也不泄露隐藏金额', async () => {
    mocks.schema.mockResolvedValue({ success: true, data: { table_fields: [
      { ...field, is_insert_show: false },
      { ...field, field_code: 'qty', field_name: '数量' },
    ] } })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('tfoot').text()).toContain('2.10')
    await wrapper.get('.document-heading button').trigger('click')
    expect(wrapper.get('tfoot').text()).not.toContain('金额合计')
    expect(wrapper.findAll('tbody tr')[1]!.get('[data-field="amount"]').text()).toBe('')
    wrapper.unmount()
  })
  it('编辑页签保留草稿，删除已有行和新增行只进入整单变更', async () => {
    const wrapper = render()
    await flushPromises()
    const sections = wrapper.findAll('.document-section')
    expect(wrapper.find('nav').text()).toBe('商品明细费用明细')
    expect(wrapper.get('.document-totals b').text()).toBe('2.10')
    await sections[0]!.get('input').setValue('3.10')
    wrapper.findComponent(Tabs).vm.$emit('update:modelValue', 2)
    await flushPromises()
    expect((sections[0]!.element as HTMLElement).style.display).toBe('none')
    await sections[1]!.get('[aria-label="删除明细"]').trigger('click')
    expect(wrapper.get('.document-totals b').text()).toBe('0.00')
    await wrapper.get('.document-heading button').trigger('click')
    await sections[1]!.get('input').setValue('0.08')
    expect(wrapper.get('.document-totals b').text()).toBe('0.08')
    expect(wrapper.vm.changes()).toEqual([
      { relation_id: 1, added: [], deleted: [], updated: [{ id: 11, data: { amount: '3.10' } }] },
      { relation_id: 2, added: [{ amount: '0.08' }], deleted: [12], updated: [] },
    ])
    wrapper.unmount()
  })
  it('新建不读取历史明细，详情只展示持久化的明细', async () => {
    const create = render(false, 0)
    await flushPromises()
    expect(mocks.rows).not.toHaveBeenCalled()
    expect(create.vm.changes()).toEqual([])
    expect(create.get('.document-totals b').text()).toBe('0.00')
    create.unmount()
    const detail = render(true)
    await flushPromises()
    expect(mocks.rows).toHaveBeenCalledWith('orders', 7, 2, 44, 'detail')
    expect(detail.get('.document-scroll tbody').text()).toContain('2.10')
    expect(detail.find('input').exists()).toBe(false)
    expect(detail.find('.document-heading button').exists()).toBe(false)
    detail.findComponent(Tabs).vm.$emit('update:modelValue', 2)
    await flushPromises()
    expect(detail.get('.document-totals b').text()).toBe('0.20')
    detail.unmount()
  })
  it('明细读取失败不将空行当成可提交的草稿', async () => {
    mocks.rows.mockRejectedValue(new Error('无法读取明细'))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('无法读取明细')
    expect(() => wrapper.vm.changes()).toThrow('无法读取明细')
    wrapper.unmount()
  })
})
