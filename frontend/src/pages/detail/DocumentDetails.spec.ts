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
  it('编辑页签保留草稿，删除已有行和新增行只进入整单变更', async () => {
    const wrapper = render()
    await flushPromises()
    const sections = wrapper.findAll('.document-section')
    expect(wrapper.find('nav').text()).toBe('商品明细费用明细')
    expect(wrapper.get('.document-totals').text()).toContain('2.30')
    await sections[0]!.get('input').setValue('3.10')
    wrapper.findComponent(Tabs).vm.$emit('update:modelValue', 2)
    await flushPromises()
    expect((sections[0]!.element as HTMLElement).style.display).toBe('none')
    await sections[1]!.get('[aria-label="删除明细"]').trigger('click')
    await sections[1]!.get('.lines-footer button').trigger('click')
    await sections[1]!.get('input').setValue('0.08')
    expect(wrapper.get('.document-totals').text()).toContain('3.18')
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
    create.unmount()
    const detail = render(true)
    await flushPromises()
    expect(mocks.rows).toHaveBeenCalledWith('orders', 7, 2, 44)
    expect(detail.get('.document-scroll tbody').text()).toContain('2.10')
    expect(detail.find('input').exists()).toBe(false)
    expect(detail.find('.lines-footer button').exists()).toBe(false)
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
