import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const routerPush = vi.hoisted(() => vi.fn())
const routerHasRoute = vi.hoisted(() => vi.fn())
const notify = vi.hoisted(() => vi.fn())
const dialog = vi.hoisted(() => vi.fn(() => ({ onOk: vi.fn() })))
const statusesRequest = vi.hoisted(() => vi.fn())
const prepareRequest = vi.hoisted(() => vi.fn())
const cleanupRequest = vi.hoisted(() => vi.fn())
const queryExternalSystems = vi.hoisted(() => vi.fn())
const queryInterfaceDefinitions = vi.hoisted(() => vi.fn())
const createExecution = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush, hasRoute: routerHasRoute }),
}))
vi.mock('quasar', async (importOriginal) => {
  const original = await importOriginal<Record<string, unknown>>()
  return { ...original, useQuasar: () => ({ notify, dialog }) }
})
vi.mock('@/api/services/development-verification', () => ({
  useDevelopmentVerificationApi: () => ({
    statuses: statusesRequest,
    prepare: prepareRequest,
    cleanup: cleanupRequest,
  }),
}))
vi.mock('@/api/services/integration', () => ({
  useIntegrationApi: () => ({
    queryExternalSystems,
    queryInterfaceDefinitions,
    createExecution,
  }),
}))
vi.mock('@/components/BaseContent/BaseContent.vue', () => ({
  default: { name: 'BaseContent', template: '<div><slot /></div>' },
}))

import VerificationPage from './Index.vue'

const mountPage = () =>
  shallowMount(VerificationPage, {
    global: { renderStubDefaultSlot: true },
  })

describe('develop verification page', () => {
  beforeEach(() => {
    routerPush.mockReset()
    routerHasRoute.mockReset()
    notify.mockReset()
    dialog.mockClear()
    statusesRequest.mockReset()
    prepareRequest.mockReset()
    cleanupRequest.mockReset()
    queryExternalSystems.mockReset()
    queryInterfaceDefinitions.mockReset()
    createExecution.mockReset()
    routerHasRoute.mockReturnValue(true)
    statusesRequest.mockResolvedValue({ data: [] })
  })

  it('provides practical scenarios for permission, TMS, file, integration and report checks', () => {
    const wrapper = mountPage()
    const scenarios = (
      wrapper.vm as unknown as {
        scenarios: Array<{ id: string; sampleId?: string; status: string; sampleFiles?: unknown[] }>
      }
    ).scenarios
    expect(scenarios.map((item) => item.id)).toEqual(
      expect.arrayContaining([
        'permission-page',
        'data-permission',
        'tms-company-scope',
        'file-upload',
        'video-preview',
        'integration-call',
        'report-runtime',
      ]),
    )
    expect(
      scenarios
        .filter((item) =>
          ['organization-sync', 'integration-call', 'file-upload', 'video-preview'].includes(
            item.id,
          ),
        )
        .map((item) => item.sampleId),
    ).toEqual(['organization-sync', 'integration-call', 'file-upload', 'video-preview'])
    expect(scenarios.find((item) => item.id === 'file-upload')?.sampleFiles).toHaveLength(2)
    expect(scenarios.find((item) => item.id === 'video-preview')?.sampleFiles).toHaveLength(1)
    expect(scenarios.find((item) => item.id === 'report-runtime')?.status).toBe('ready')
  })

  it('filters scenarios by category and keyword', async () => {
    const wrapper = mountPage()
    const vm = wrapper.vm as unknown as {
      category: string
      keyword: string
      filteredScenarios: Array<{ id: string }>
    }
    vm.category = 'file'
    vm.keyword = '视频'
    await wrapper.vm.$nextTick()
    expect(vm.filteredScenarios.map((item) => item.id)).toEqual(['video-preview'])
  })

  it('opens the related page only when the current router exposes it', async () => {
    const wrapper = mountPage()
    const vm = wrapper.vm as unknown as {
      openScenario: (scenario: { routeName: string }) => Promise<void>
    }
    await vm.openScenario({ routeName: 'system_role' })
    expect(routerPush).toHaveBeenCalledWith({ name: 'system_role' })

    routerHasRoute.mockReturnValue(false)
    await vm.openScenario({ routeName: 'system_role' })
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'warning', message: '当前账号没有可打开的相关页面' }),
    )
  })

  it('prepares a business sample and shows its one-time accounts', async () => {
    prepareRequest.mockResolvedValue({
      data: {
        status: {
          scenario_id: 'data-permission',
          state: 'ready',
          available: true,
          item_count: 3,
          summary: '样例已准备',
          details: [],
        },
        accounts: [
          {
            user_name: 'verify_permission_east',
            password: 'temporary-password',
            role: '仅华东订单',
            expected: '只能看到 EAST 的两条订单',
          },
        ],
      },
    })
    const wrapper = mountPage()
    await flushPromises()
    const vm = wrapper.vm as unknown as {
      scenarios: Array<{ id: string }>
      prepareSample: (scenario: { id: string }) => Promise<void>
      accountDialog: boolean
      preparedAccounts: Array<{ user_name: string }>
    }
    const scenario = vm.scenarios.find((item) => item.id === 'data-permission')
    expect(scenario).toBeDefined()

    await vm.prepareSample(scenario!)

    expect(prepareRequest).toHaveBeenCalledWith('data-permission')
    expect(vm.accountDialog).toBe(true)
    expect(vm.preparedAccounts[0]?.user_name).toBe('verify_permission_east')
    expect(notify).toHaveBeenCalledWith(expect.objectContaining({ type: 'positive' }))
  })

  it('runs the prepared integration fixture through the real execution API', async () => {
    queryExternalSystems.mockResolvedValue({
      data: [{ id: 11, system_code: 'verify_integration_source' }],
    })
    queryInterfaceDefinitions.mockResolvedValue({
      data: [{ id: 22, interface_code: 'verify_ping' }],
    })
    createExecution.mockResolvedValue({
      success: true,
      data: { id: 33, execution_no: 'INT-33' },
    })
    const wrapper = mountPage()
    await flushPromises()
    const vm = wrapper.vm as unknown as { runIntegrationSample: () => Promise<void> }

    await vm.runIntegrationSample()

    expect(createExecution).toHaveBeenCalledWith(
      expect.objectContaining({
        external_system_id: 11,
        interface_definition_id: 22,
        trigger_source: 'manual',
      }),
    )
    expect(routerPush).toHaveBeenCalledWith({
      name: 'integration_execution_detail_page',
      params: { id: 33 },
    })
  })
})
