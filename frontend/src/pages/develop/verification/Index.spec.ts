import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const routerPush = vi.hoisted(() => vi.fn())
const routerHasRoute = vi.hoisted(() => vi.fn())
const notify = vi.hoisted(() => vi.fn())
const dialog = vi.hoisted(() => vi.fn(() => ({ onOk: vi.fn() })))
const statusesRequest = vi.hoisted(() => vi.fn())
const prepareRequest = vi.hoisted(() => vi.fn())
const cleanupRequest = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush, hasRoute: routerHasRoute }),
}))
vi.mock('quasar', async (importOriginal) => {
  const original = await importOriginal<Record<string, unknown>>()
  return { ...original, useQuasar: () => ({ notify, dialog }) }
})
vi.mock('src/api/services/development-verification', () => ({
  useDevelopmentVerificationApi: () => ({
    statuses: statusesRequest,
    prepare: prepareRequest,
    cleanup: cleanupRequest,
  }),
}))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
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
    routerHasRoute.mockReturnValue(true)
    statusesRequest.mockResolvedValue({ data: [] })
  })

  it('provides practical scenarios for permission, TMS, file, integration and report checks', () => {
    const wrapper = mountPage()
    const scenarios = (wrapper.vm as unknown as { scenarios: Array<{ id: string }> }).scenarios
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
})
