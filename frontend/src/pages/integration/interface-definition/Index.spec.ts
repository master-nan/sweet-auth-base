import { computed, defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  queryInterfaceDefinitions: vi.fn(), queryExternalSystems: vi.fn(), getInterfaceDefinition: vi.fn(),
  createInterfaceDefinition: vi.fn(), updateInterfaceDefinition: vi.fn(), createInterfaceDefinitionVersion: vi.fn(),
  enableInterfaceDefinition: vi.fn(), disableInterfaceDefinition: vi.fn(),
}))
const tableApiMocks = vi.hoisted(() => ({ queryTableByCode: vi.fn() }))
const buttons = vi.hoisted(() => ({
  top: [{ id: 1, name: '新增', event_action: 'create', icon: 'add', color: 'primary' }],
  line: [
    { id: 2, name: '详情', event_action: 'detail' }, { id: 3, name: '编辑', event_action: 'update' },
    { id: 4, name: '创建版本', event_action: 'create_version' }, { id: 5, name: '启用', event_action: 'enable' },
    { id: 6, name: '停用', event_action: 'disable' },
  ],
}))
vi.mock('quasar', () => ({ useQuasar: () => ({ screen: { lt: { md: false } } }) }))
vi.mock('boot/axios', () => ({ instance: {} }))
vi.mock('src/api/services/integration', () => ({ useIntegrationApi: () => apiMocks }))
vi.mock('src/api/services/sys-table', () => ({ useTableApi: () => tableApiMocks }))
vi.mock('src/composables/page-buttons', () => ({ usePageButtons: () => ({ top_buttons: computed(() => buttons.top), line_buttons: computed(() => buttons.line), has_line_buttons: computed(() => true) }) }))
vi.mock('src/composables/confirm-dialog', () => ({ useConfirmDialog: () => ({ confirmAction: vi.fn(() => ({ onOk: vi.fn() })) }) }))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({ default: { template: '<div><slot /></div>' } }))
vi.mock('src/components/Table/TablePagination.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/Query/AdvancedQuery.vue', () => ({ default: { template: '<div />' } }))
vi.mock('./InterfaceDefinitionFormDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('./InterfaceDefinitionDetailDialog.vue', () => ({ default: { template: '<div />' } }))

import InterfaceDefinitionPage from './Index.vue'

const SlotStub = defineComponent({ setup(_, { slots }) { return () => h('div', slots.default?.()) } })
const TableStub = defineComponent({
  name: 'QTable', props: { rows: { type: Array, default: () => [] } },
  setup(props, { slots }) { return () => h('section', { 'data-testid': 'table', 'data-row-count': props.rows.length }, [slots.top?.(), slots.bottom?.()]) },
})
const ButtonStub = defineComponent({
  name: 'QBtn', props: { label: { type: String, default: '' } }, emits: ['click'],
  setup(props, { emit }) { return () => h('button', { onClick: () => emit('click') }, props.label) },
})
const mountPage = () => shallowMount(InterfaceDefinitionPage, { global: { plugins: [createPinia()], stubs: {
  BaseContent: SlotStub, QTable: TableStub, QInput: true, QSelect: true, QBtn: ButtonStub, QIcon: true, QSpace: true,
  QBadge: true, QTooltip: true, QChip: true, QTd: SlotStub, TablePagination: true, AdvancedQuery: true,
  InterfaceDefinitionFormDialog: true, InterfaceDefinitionDetailDialog: true,
} } })

const system = { id: 10, system_code: 'demo_erp', name: 'Demo ERP', system_type: 'erp', base_url_summary: 'https://erp.example.com', owner_identifier: 'owner', owner_name: '负责人', status: 'enabled', revision: 1, gmt_modify: '' }
const row = { id: 31, external_system: { id: 10, system_code: 'demo_erp', name: 'Demo ERP' }, interface_code: 'order_query', name: '订单查询', version: 1, protocol: 'https', http_method: 'GET', path_summary: '/api/orders', status: 'draft', revision: 1, gmt_modify: '' }

describe('interface definition management page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
    tableApiMocks.queryTableByCode.mockReset()
    apiMocks.queryInterfaceDefinitions.mockResolvedValue({ data: [row], total: 1 })
    apiMocks.queryExternalSystems.mockResolvedValue({ data: [system], total: 1 })
    tableApiMocks.queryTableByCode.mockResolvedValue({ data: { table_fields: [] } })
  })

  it('loads metadata, systems and the paged interface list', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(tableApiMocks.queryTableByCode).toHaveBeenCalledWith('integration_interface_definition')
    expect(apiMocks.queryExternalSystems).toHaveBeenCalled()
    expect(apiMocks.queryInterfaceDefinitions).toHaveBeenCalledWith(expect.objectContaining({ page: 1, num: 15 }))
    expect(wrapper.find('[data-testid="table"]').attributes('data-row-count')).toBe('1')
    expect(wrapper.text()).toContain('新增')
  })

  it('shows only lifecycle-compatible dynamic actions', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const vm = wrapper.vm as unknown as { availableLineButtons: (item: typeof row) => Array<{ event_action: string }> }
    expect(vm.availableLineButtons(row).map((item) => item.event_action)).toEqual(['detail', 'update', 'enable'])
    expect(vm.availableLineButtons({ ...row, status: 'enabled' }).map((item) => item.event_action)).toEqual(['detail', 'create_version', 'disable'])
  })
})
