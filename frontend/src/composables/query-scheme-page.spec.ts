import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Query } from '@/types/global'

const notify = vi.hoisted(() => vi.fn())
const push = vi.hoisted(() => vi.fn())
const route = vi.hoisted(() => ({ query: {} as Record<string, unknown> }))
const runtime = vi.hoisted(() => ({
  issues: { value: [] as Array<{ message: string }> },
  error: { value: '' },
  initialize: vi.fn(),
  applyScheme: vi.fn(),
  applyPreset: vi.fn(),
  restoreCurrentScheme: vi.fn(),
  resetToDefault: vi.fn(),
  savePersonal: vi.fn(),
}))

vi.mock('quasar', () => ({ useQuasar: () => ({ notify }) }))
vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({ push }),
}))
vi.mock('@/composables/query-schemes', () => ({ useQuerySchemes: () => runtime }))

import { useQuerySchemePage } from './query-scheme-page'

const createState = () => ({ query: { value: { page: 1 } } }) as never

describe('useQuerySchemePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    route.query = { query_scheme_id: '17' }
    window.history.replaceState({}, '')
    runtime.issues.value = []
    runtime.error.value = ''
  })

  it('initializes a requested scheme and refreshes after a valid selection', async () => {
    const refresh = vi.fn()
    runtime.initialize.mockResolvedValue(true)
    runtime.applyScheme.mockResolvedValue(true)
    const page = useQuerySchemePage('system_user', createState(), refresh)

    await page.initialize()
    await page.selectScheme({ id: 2 } as never)

    expect(runtime.initialize).toHaveBeenCalledWith(17, {})
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('passes explicit route-context preservation to runtime initialization', async () => {
    const page = useQuerySchemePage('system_user', createState(), vi.fn())

    await page.initialize({ preserveInitialQuery: true })

    expect(runtime.initialize).toHaveBeenCalledWith(17, { preserveInitialQuery: true })
  })

  it('consumes a transient navigation state without leaving it in the URL contract', async () => {
    route.query = {}
    window.history.replaceState({ query_scheme_id: '23' }, '')
    const page = useQuerySchemePage('system_user', createState(), vi.fn())

    await page.initialize()

    expect(runtime.initialize).toHaveBeenCalledWith(23, {})
    expect(window.history.state.query_scheme_id).toBeUndefined()
  })

  it('reports a blocked scheme without refreshing business data', async () => {
    const refresh = vi.fn()
    runtime.applyScheme.mockResolvedValue(false)
    runtime.issues.value = [{ message: '字段已不可用' }]
    const page = useQuerySchemePage('system_user', createState(), refresh)

    await page.selectScheme({ id: 3 } as never)

    expect(notify).toHaveBeenCalledWith({ type: 'warning', message: '字段已不可用' })
    expect(refresh).not.toHaveBeenCalled()
  })

  it('keeps preset, restore, default and save behavior in one page boundary', async () => {
    const refresh = vi.fn()
    runtime.restoreCurrentScheme.mockReturnValue(true)
    runtime.resetToDefault.mockResolvedValue(true)
    runtime.savePersonal.mockResolvedValue({ id: 1 })
    const page = useQuerySchemePage<Query>('system_user', createState(), refresh)
    page.showSaveDialog.value = true

    page.applyPreset({
      expressions: [],
      quick_query: { keyword: '' },
      order: { field: '', is_asc: false },
      bindings: [],
    })
    page.restoreCurrent()
    await page.resetDefault()
    await page.savePersonal({ name: '我的方案', isDefault: true, saveAs: false })

    expect(refresh).toHaveBeenCalledTimes(3)
    expect(page.showSaveDialog.value).toBe(false)
    expect(page.saving.value).toBe(false)
  })

  it('lets the pagination watcher own refresh when applying a scheme resets page', async () => {
    const refresh = vi.fn()
    const query = { value: { page: 3 } }
    runtime.applyScheme.mockImplementation(() => {
      query.value.page = 1
      return Promise.resolve(true)
    })
    const page = useQuerySchemePage('system_user', { query } as never, refresh)

    await page.selectScheme({ id: 4 } as never)

    expect(refresh).not.toHaveBeenCalled()
  })

  it('runs a query change once whether or not it resets pagination', () => {
    const refresh = vi.fn()
    const state = {
      query: { value: { page: 1 } },
      submitQuickSearch: vi.fn(function (this: void) {}),
    }
    const page = useQuerySchemePage('system_user', state as never, refresh)

    page.runQueryChange(() => undefined)
    expect(refresh).toHaveBeenCalledOnce()

    state.query.value.page = 3
    page.runQueryChange(() => {
      state.query.value.page = 1
    })
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('keeps the save dialog open and absorbs a reported save conflict', async () => {
    runtime.savePersonal.mockRejectedValueOnce(new Error('方案已被其他操作更新，请刷新后重试'))
    const page = useQuerySchemePage('system_user', createState(), vi.fn())
    page.showSaveDialog.value = true

    await page.savePersonal({ name: '我的方案', isDefault: false, saveAs: false })

    expect(page.showSaveDialog.value).toBe(true)
    expect(page.saving.value).toBe(false)
  })
})
