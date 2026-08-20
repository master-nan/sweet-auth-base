import { computed } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({ queryLogs: vi.fn(), getLog: vi.fn() }))
const detailButtons = vi.hoisted(() => [] as Array<Record<string, unknown>>)
const permissionCodes = vi.hoisted(() => [] as string[])
const tableApiMocks = vi.hoisted(() => ({ queryRuntimeTableByCode: vi.fn() }))
const schemeMocks = vi.hoisted(() => ({ initialize: vi.fn() }))

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
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { execution_no: 'INT-51', log_id: '91' } }),
}))
vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => ({
    line_buttons: computed(() => detailButtons),
    top_buttons: computed(() => []),
    hasGrantedCapability: (code: string) => permissionCodes.includes(code),
  }),
}))
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ buttons: permissionCodes }) }))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))
vi.mock('src/components/Table/TablePagination.vue', () => ({
  default: { template: '<div />' },
}))

import IntegrationLogPage from './Index.vue'

const mountPage = () =>
  shallowMount(IntegrationLogPage, {
    global: { plugins: [createPinia()], renderStubDefaultSlot: true },
  })

describe('integration log detail permission', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    detailButtons.splice(0)
    permissionCodes.splice(0)
    apiMocks.queryLogs.mockReset()
    apiMocks.getLog.mockReset()
    schemeMocks.initialize.mockReset()
    schemeMocks.initialize.mockResolvedValue(false)
    tableApiMocks.queryRuntimeTableByCode.mockReset()
    tableApiMocks.queryRuntimeTableByCode.mockResolvedValue({
      success: true,
      data: {
        table_fields: [
          {
            field_code: 'attempt_no',
            field_name: 'Attempt',
            field_type: 'int',
            is_advanced_search: true,
            sequence: 1,
          },
        ],
      },
    })
    apiMocks.queryLogs.mockResolvedValue({ data: [], total: 0 })
    apiMocks.getLog.mockResolvedValue({
      data: {
        id: 91,
        execution_no: 'INT-51',
        attempt_no: 2,
        retryable: true,
        retry_reason_code: 'retry_allowed',
        retry_delay_ms: 2000,
        retry_scheduled_at: '2026-08-09T10:00:02Z',
        retry_after_source: 'http_delta',
        result_size_bytes: 0,
      },
    })
  })

  it('does not request logs without the log query permission', async () => {
    mountPage()
    await flushPromises()

    expect(apiMocks.queryLogs).not.toHaveBeenCalled()
    expect(apiMocks.getLog).not.toHaveBeenCalled()
  })

  it('queries logs but does not request routed detail without detail permission', async () => {
    permissionCodes.push('integration_log_query')
    mountPage()
    await flushPromises()

    expect(apiMocks.queryLogs).toHaveBeenCalled()
    expect(apiMocks.getLog).not.toHaveBeenCalled()
    expect(tableApiMocks.queryRuntimeTableByCode).toHaveBeenCalledWith('integration_log')
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
    expect(schemeMocks.initialize.mock.invocationCallOrder[0]).toBeLessThan(
      apiMocks.queryLogs.mock.invocationCallOrder[0]!,
    )
  })

  it('loads routed log detail when the detail permission is present', async () => {
    permissionCodes.push('integration_log_query')
    detailButtons.push({ id: 1, name: '详情', event_action: 'detail' })
    const wrapper = mountPage()
    await flushPromises()

    expect(apiMocks.getLog).toHaveBeenCalledWith(91)
    expect(wrapper.text()).toContain('自动重试')
    expect(wrapper.text()).toContain('符合自动重试条件')
    expect(wrapper.text()).not.toContain('retry_allowed')
    expect(wrapper.text()).toContain('Retry-After 秒数')
    expect(wrapper.text()).not.toContain('Authorization')
    expect(wrapper.text()).not.toContain('Payload')
    await (wrapper.vm as unknown as { fetchData: () => Promise<void> }).fetchData()
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
  })
})
