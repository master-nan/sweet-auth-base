import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const routerPush = vi.hoisted(() => vi.fn())
const routerHasRoute = vi.hoisted(() => vi.fn())
const notify = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush, hasRoute: routerHasRoute }),
}))
vi.mock('quasar', async (importOriginal) => {
  const original = await importOriginal<Record<string, unknown>>()
  return { ...original, useQuasar: () => ({ notify }) }
})
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
    routerHasRoute.mockReturnValue(true)
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
})
