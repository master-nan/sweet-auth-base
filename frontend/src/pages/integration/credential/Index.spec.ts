import { computed, defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  queryCredentials: vi.fn(),
  queryExternalSystems: vi.fn(),
  getCredential: vi.fn(),
  createCredential: vi.fn(),
  updateCredential: vi.fn(),
  rotateCredential: vi.fn(),
  enableCredential: vi.fn(),
  disableCredential: vi.fn(),
  revokeCredential: vi.fn(),
}))
const tableApiMocks = vi.hoisted(() => ({ queryRuntimeTableByCode: vi.fn() }))
const schemeMocks = vi.hoisted(() => ({ initialize: vi.fn() }))
const buttons = vi.hoisted(() => ({
  top: [{ id: 1, name: '新增', event_action: 'create', icon: 'add', color: 'primary' }],
  line: [
    { id: 2, name: '详情', event_action: 'detail' },
    { id: 3, name: '编辑', event_action: 'update' },
    { id: 4, name: '轮换', event_action: 'rotate' },
    { id: 5, name: '启用', event_action: 'enable' },
    { id: 6, name: '停用', event_action: 'disable' },
    { id: 7, name: '吊销', event_action: 'revoke' },
  ],
}))
vi.mock('quasar', () => ({ useQuasar: () => ({ screen: { lt: { md: false } } }) }))
vi.mock('boot/axios', () => ({ instance: {} }))
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }) }))
vi.mock('src/api/services/integration', () => ({ useIntegrationApi: () => apiMocks }))
vi.mock('src/composables/query-scheme-page', async () => {
  const { ref } = await import('vue')
  return {
    useQuerySchemePage: () => ({
      runtime: {
        schemes: ref([]),
        currentLabel: ref('查询方案'),
        loading: ref(false),
        error: ref(''),
        scope: { config: ref(null) },
        loadAvailable: vi.fn(),
      },
      showSaveDialog: ref(false),
      saving: ref(false),
      initialize: schemeMocks.initialize,
      selectScheme: vi.fn(),
      applyPreset: vi.fn(),
      restoreCurrent: vi.fn(),
      resetDefault: vi.fn(),
      openManager: vi.fn(),
      savePersonal: vi.fn(),
    }),
  }
})
vi.mock('src/api/services/sys-table', () => ({ useTableApi: () => tableApiMocks }))
vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => ({
    top_buttons: computed(() => buttons.top),
    line_buttons: computed(() => buttons.line),
    has_line_buttons: computed(() => true),
  }),
}))
vi.mock('src/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({ confirmAction: vi.fn(() => ({ onOk: vi.fn() })) }),
}))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))
vi.mock('src/components/Table/TablePagination.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/Query/AdvancedQuery.vue', () => ({ default: { template: '<div />' } }))
vi.mock('./CredentialFormDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('./CredentialDetailDialog.vue', () => ({ default: { template: '<div />' } }))

import CredentialPage from './Index.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const TableStub = defineComponent({
  name: 'QTable',
  props: { rows: { type: Array, default: () => [] } },
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-testid': 'table', 'data-row-count': props.rows.length }, [
        slots.top?.(),
        slots.bottom?.(),
      ])
  },
})
const ButtonStub = defineComponent({
  name: 'QBtn',
  props: { label: { type: String, default: '' } },
  emits: ['click'],
  setup(props, { emit }) {
    return () => h('button', { onClick: () => emit('click') }, props.label)
  },
})
const mountPage = () =>
  shallowMount(CredentialPage, {
    global: {
      plugins: [createPinia()],
      stubs: {
        BaseContent: SlotStub,
        QTable: TableStub,
        QInput: true,
        QSelect: true,
        QBtn: ButtonStub,
        QIcon: true,
        QSpace: true,
        QBadge: true,
        QTooltip: true,
        QChip: true,
        QTd: SlotStub,
        TablePagination: true,
        AdvancedQuery: true,
        CredentialFormDialog: true,
        CredentialDetailDialog: true,
      },
    },
  })

const system = {
  id: 10,
  system_code: 'demo_erp',
  name: 'Demo ERP',
  system_type: 'erp',
  base_url_summary: 'https://erp.example.com',
  owner_identifier: 'owner',
  owner_name: '负责人',
  status: 'enabled',
  revision: 1,
  gmt_modify: '',
}
const row = {
  id: 41,
  external_system: { id: 10, system_code: 'demo_erp', name: 'Demo ERP' },
  credential_code: 'erp_api',
  name: 'ERP API Key',
  credential_type: 'api_key',
  status: 'draft',
  effective_status: 'draft',
  fingerprint_summary: 'abcdef123456',
  version: 1,
  revision: 1,
  gmt_modify: '',
}

describe('credential management page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
    schemeMocks.initialize.mockReset()
    schemeMocks.initialize.mockResolvedValue(false)
    tableApiMocks.queryRuntimeTableByCode.mockReset()
    apiMocks.queryCredentials.mockResolvedValue({ data: [row], total: 1 })
    apiMocks.queryExternalSystems.mockResolvedValue({ data: [system], total: 1 })
    tableApiMocks.queryRuntimeTableByCode.mockResolvedValue({ data: { table_fields: [] } })
  })

  it('loads metadata, systems and the credential page without exposing secret fields', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(tableApiMocks.queryRuntimeTableByCode).toHaveBeenCalledWith('integration_credential')
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
    expect(schemeMocks.initialize.mock.invocationCallOrder[0]).toBeLessThan(
      apiMocks.queryCredentials.mock.invocationCallOrder[0]!,
    )
    expect(apiMocks.queryCredentials).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, num: 15 }),
    )
    expect(wrapper.find('[data-testid="table"]').attributes('data-row-count')).toBe('1')
    expect(wrapper.text()).not.toContain('secret')
    await (wrapper.vm as unknown as { fetchData: () => Promise<void> }).fetchData()
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
  })

  it('hides every mutating action after revocation', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const vm = wrapper.vm as unknown as {
      availableLineButtons: (item: typeof row) => Array<{ event_action: string }>
    }
    expect(vm.availableLineButtons(row).map((item) => item.event_action)).toEqual([
      'detail',
      'update',
      'rotate',
      'enable',
      'revoke',
    ])
    expect(
      vm
        .availableLineButtons({ ...row, status: 'revoked', effective_status: 'revoked' })
        .map((item) => item.event_action),
    ).toEqual(['detail'])
  })
})
