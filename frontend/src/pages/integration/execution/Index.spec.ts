import { computed, defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  queryExecutions: vi.fn(),
  getWorkerStatus: vi.fn(),
  cancelExecution: vi.fn(),
}))
const permissionCodes = vi.hoisted(() => [
  'integration_execution_query',
  'integration_execution_detail',
  'integration_worker_status',
])
const lineButtons = vi.hoisted(() => [
  { id: 1, name: '详情', event_action: 'detail' },
  { id: 2, name: '取消', event_action: 'cancel' },
])
const tableApiMocks = vi.hoisted(() => ({ queryRuntimeTableByCode: vi.fn() }))
const schemeMocks = vi.hoisted(() => ({ initialize: vi.fn() }))

vi.mock('quasar', () => ({ useQuasar: () => ({}) }))
vi.mock('boot/axios', () => ({ instance: {} }))
vi.mock('src/api/services/integration', () => ({ useIntegrationApi: () => apiMocks }))
vi.mock('src/api/services/sys-table', () => ({ useTableApi: () => tableApiMocks }))
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
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ buttons: permissionCodes }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => ({
    line_buttons: computed(() => lineButtons),
    top_buttons: computed(() => []),
    hasGrantedCapability: (code: string) => permissionCodes.includes(code),
  }),
}))
vi.mock('src/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({ confirmAction: vi.fn(() => ({ onOk: vi.fn() })) }),
}))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))
vi.mock('src/components/Table/TablePagination.vue', () => ({ default: { template: '<div />' } }))

import ExecutionPage from './Index.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const TableStub = defineComponent({
  props: { rows: { type: Array, default: () => [] } },
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-testid': 'table', 'data-row-count': props.rows.length }, [
        slots.top?.(),
        slots.bottom?.(),
      ])
  },
})
const mountPage = () =>
  shallowMount(ExecutionPage, {
    global: {
      plugins: [createPinia()],
      stubs: {
        BaseContent: SlotStub,
        QTable: TableStub,
        QCard: SlotStub,
        QCardSection: SlotStub,
        QInput: true,
        QSelect: true,
        QBtn: true,
        QIcon: true,
        QSpace: true,
        QChip: true,
        QTooltip: true,
        QTd: SlotStub,
        TablePagination: true,
      },
    },
  })

describe('integration execution retry summary', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    permissionCodes.splice(
      0,
      permissionCodes.length,
      'integration_execution_query',
      'integration_execution_detail',
      'integration_worker_status',
    )
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
    schemeMocks.initialize.mockReset()
    schemeMocks.initialize.mockResolvedValue(false)
    tableApiMocks.queryRuntimeTableByCode.mockReset()
    tableApiMocks.queryRuntimeTableByCode.mockResolvedValue({
      success: true,
      data: {
        table_fields: [
          {
            field_code: 'status',
            field_name: '状态',
            field_type: 'varchar',
            is_advanced_search: true,
            sequence: 1,
          },
        ],
      },
    })
    apiMocks.queryExecutions.mockResolvedValue({
      data: [
        {
          id: 51,
          execution_no: 'INT-51',
          external_system: { name: 'HR', system_code: 'hr' },
          interface: { name: '组织', interface_code: 'org', version: 1 },
          trigger_source: 'manual',
          status: 'retry_waiting',
          current_attempt: 1,
          max_attempts: 3,
          next_run_at: '2026-08-09T10:00:02Z',
          retry_reason_code: 'retry_allowed',
          revision: 2,
          started_at: '2026-08-09T10:00:00Z',
          completed_at: '',
        },
      ],
      total: 1,
    })
    apiMocks.getWorkerStatus.mockResolvedValue({ data: { enabled: false, running: false } })
  })

  it('does not preload executions or worker status without their query permissions', async () => {
    permissionCodes.splice(0)
    mountPage()
    await flushPromises()

    expect(apiMocks.queryExecutions).not.toHaveBeenCalled()
    expect(apiMocks.getWorkerStatus).not.toHaveBeenCalled()
  })

  it('loads safe retry list summaries and exposes no runtime mutation actions', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(tableApiMocks.queryRuntimeTableByCode).toHaveBeenCalledWith('integration_execution')
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
    expect(schemeMocks.initialize.mock.invocationCallOrder[0]).toBeLessThan(
      apiMocks.queryExecutions.mock.invocationCallOrder[0]!,
    )
    const vm = wrapper.vm as unknown as {
      columns: Array<{ name: string }>
      availableButtons: (row: { status: string }) => Array<{ event_action: string }>
      fetchData: () => Promise<void>
    }
    expect(wrapper.find('[data-testid="table"]').attributes('data-row-count')).toBe('1')
    expect(wrapper.find('advanced-query-stub').exists()).toBe(true)
    expect(vm.columns.map((column) => column.name)).toEqual(
      expect.arrayContaining([
        'current_attempt',
        'next_run_at',
        'retry_reason_code',
        'started_at',
        'completed_at',
      ]),
    )
    expect(
      vm.availableButtons({ status: 'retry_waiting' }).map((button) => button.event_action),
    ).toEqual(['detail', 'cancel'])
    expect(lineButtons.map((button) => button.event_action)).not.toEqual(
      expect.arrayContaining(['retry_now', 'replay', 'start', 'complete', 'fail']),
    )
    await vm.fetchData()
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
  })
})
