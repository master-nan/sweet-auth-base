import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getExecution: vi.fn(),
  queryLogs: vi.fn(),
}))
const permissionCodes = vi.hoisted(() => [] as string[])
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('src/api/services/integration', () => ({ useIntegrationApi: () => apiMocks }))
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ buttons: permissionCodes }) }))
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '51' } }),
  useRouter: () => ({ push: routerPush, back: vi.fn() }),
}))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))

import ExecutionDetailPage from './Detail.vue'

const detail = {
  id: 51,
  execution_no: 'INT-51',
  external_system: { id: 10, system_code: 'hr_demo', name: 'HR Demo' },
  interface: { id: 20, system_code: 'org_list', name: '组织列表', version: 1 },
  trigger_source: 'manual',
  status: 'succeeded',
  current_attempt: 1,
  max_attempts: 3,
  attempts_remaining: 2,
  retry_policy: { policy_code: 'safe_retry', policy_version: 2, max_attempts: 3 },
  retry_reason_code: 'retry_allowed',
  revision: 3,
  gmt_create: '2026-08-06 10:00:00',
  duration_ms: 20,
  idempotency_scope: 'manual',
  idempotency_key: 'req-51',
  input_hash: 'a'.repeat(64),
  input_summary: {
    snapshot_version: 1,
    size_bytes: 128,
    path_count: 1,
    query_count: 2,
    header_count: 1,
    has_body: true,
  },
  result_size_bytes: 0,
}

const mountPage = () =>
  shallowMount(ExecutionDetailPage, {
    global: {
      plugins: [createPinia()],
      renderStubDefaultSlot: true,
    },
  })

describe('integration execution detail permissions', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    permissionCodes.splice(0, permissionCodes.length, 'integration_execution_detail')
    routerPush.mockReset()
    apiMocks.getExecution.mockReset()
    apiMocks.queryLogs.mockReset()
    apiMocks.getExecution.mockResolvedValue({ data: detail })
    apiMocks.queryLogs.mockResolvedValue({
      data: [
        {
          id: 91,
          execution_no: 'INT-51',
          attempt_no: 1,
          system_code: 'hr_demo',
          interface_code: 'org_list',
          status: 'succeeded',
          started_at: '2026-08-06 10:00:00',
          duration_ms: 20,
          result_certainty: 'confirmed',
        },
      ],
      total: 1,
    })
  })

  it('does not request the execution when direct detail permission is absent', async () => {
    permissionCodes.splice(0)
    mountPage()
    await flushPromises()

    expect(apiMocks.getExecution).not.toHaveBeenCalled()
    expect(apiMocks.queryLogs).not.toHaveBeenCalled()
  })

  it('does not request Attempt data without log query permission', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(apiMocks.getExecution).toHaveBeenCalledWith(51)
    expect(apiMocks.queryLogs).not.toHaveBeenCalled()
    expect((wrapper.vm as unknown as { canQueryLogs: boolean }).canQueryLogs).toBe(false)
  })

  it('renders only the safe execution input summary', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('输入快照摘要')
    expect(wrapper.text()).toContain('128 字节')
    expect(wrapper.text()).toContain('JSON Body')
    expect(wrapper.text()).not.toContain('Authorization')
    expect(wrapper.text()).not.toContain('Payload')
  })

  it('renders the safe retry summary without exposing the policy snapshot', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('自动重试摘要')
    expect(wrapper.text()).toContain('safe_retry')
    expect(wrapper.text()).toContain('1 / 3')
    expect(wrapper.text()).toContain('符合自动重试条件')
    expect(wrapper.text()).not.toContain('retry_allowed')
    expect(wrapper.text()).not.toContain('RetryPolicySnapshot')
    expect(wrapper.text()).not.toContain('retryable_error_categories')
  })

  it('loads Attempt data through the independent log query API', async () => {
    permissionCodes.push('integration_log_query')
    const wrapper = mountPage()
    await flushPromises()

    expect(apiMocks.queryLogs).toHaveBeenCalledWith(
      expect.objectContaining({ execution_id: 51, page: 1, num: 500 }),
    )
    expect((wrapper.vm as unknown as { attempts: unknown[] }).attempts).toHaveLength(1)
  })

  it('only enables Attempt detail navigation with log detail permission', async () => {
    permissionCodes.push('integration_log_query', 'integration_log_detail')
    const wrapper = mountPage()
    await flushPromises()

    const vm = wrapper.vm as unknown as {
      canViewLogDetail: boolean
      openLog: (id: number) => void
    }
    expect(vm.canViewLogDetail).toBe(true)
    vm.openLog(91)
    expect(routerPush).toHaveBeenCalledWith({
      name: 'integration_log',
      query: { execution_no: 'INT-51', log_id: '91' },
    })
  })
})
