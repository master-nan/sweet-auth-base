import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useTableQueryState } from './table-query-state'
import { QuerySchemeType, QuerySchemeValidationStatus } from 'src/modules/query-scheme/types'
import type { Query } from 'src/types/global'

const api = vi.hoisted(() => ({ available: vi.fn(), resolve: vi.fn(), createPersonal: vi.fn(), updatePersonal: vi.fn() }))
const scope = vi.hoisted(() => ({
  scopeCode: { value: 'system.user.list' },
  config: { value: null },
  loading: { value: false },
  error: { value: '' },
  available: { value: true },
  loadScope: vi.fn(),
}))
vi.mock('src/api/services/query-scheme', () => ({ useQuerySchemeApi: () => api }))
vi.mock('src/composables/query-scope', () => ({ useQueryScope: () => scope }))

import { useQuerySchemes } from './query-schemes'

const createState = () => useTableQueryState<Query>({ createInitialQuery: () => ({ page: 1, num: 15, expressions: [], quick_query: { keyword: '' }, order: { field: '', is_asc: false } }) })

describe('useQuerySchemes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    scope.loadScope.mockResolvedValue({ scope_code: 'system.user.list' })
    api.available.mockResolvedValue({ data: [] })
  })

  it('applies personal default before page default only during initialization', async () => {
    const personal = { id: 1, name: '个人默认', type: QuerySchemeType.PERSONAL, is_default: true, status: QuerySchemeValidationStatus.VALID }
    const page = { id: 2, name: '页面默认', type: QuerySchemeType.PAGE_DEFAULT, is_default: true, status: QuerySchemeValidationStatus.VALID }
    api.available.mockResolvedValue({ data: [page, personal] })
    api.resolve.mockResolvedValue({ data: { scheme: { ...personal, revision: 1 }, validation_status: QuerySchemeValidationStatus.VALID, issues: [], bindings: [], binding_kinds: [], resolved_query: { expressions: [], quick_query: { keyword: 'mine' }, order: { field: '', is_asc: false } } } })
    const runtime = useQuerySchemes('system_user', createState())

    expect(await runtime.initialize()).toBe(true)
    expect(api.resolve).toHaveBeenCalledWith(1, 'system.user.list')
    await runtime.initialize()
    expect(api.resolve).toHaveBeenCalledTimes(1)
  })

  it('does not apply degraded schemes or alter the current query', async () => {
    const state = createState()
    const scheme = { id: 3, name: '旧字段方案', type: QuerySchemeType.PUBLIC, is_default: false, status: QuerySchemeValidationStatus.DEGRADED }
    api.resolve.mockResolvedValue({ data: { scheme: { ...scheme, revision: 1 }, validation_status: QuerySchemeValidationStatus.DEGRADED, issues: [{ code: 'field_unavailable', message: '字段已不可用' }], bindings: [], binding_kinds: [] } })
    const runtime = useQuerySchemes('system_user', state)

    expect(await runtime.applyScheme(scheme)).toBe(false)
    expect(runtime.issues.value[0]?.message).toBe('字段已不可用')
    expect(state.schemeSource.value).toBeNull()
  })

  it('does not overwrite the local scheme baseline when an update conflicts', async () => {
    const state = createState()
    const scheme = {
      id: 9,
      name: '我的查询',
      type: QuerySchemeType.PERSONAL,
      revision: 3,
      is_default: false,
      status: QuerySchemeValidationStatus.VALID,
    }
    state.applyResolvedScheme(scheme, {
      expressions: [],
      quick_query: { keyword: 'before' },
      order: { field: '', is_asc: false },
    }, [])
    state.query.value.quick_query = { keyword: 'after' }
    api.updatePersonal.mockRejectedValue(new Error('方案已被其他操作更新，请刷新后重试'))
    const runtime = useQuerySchemes('system_user', state)

    await expect(runtime.savePersonal('我的查询', false, false)).rejects.toThrow('方案已被其他操作更新，请刷新后重试')
    expect(state.schemeSource.value?.revision).toBe(3)
    expect(state.dirty.value).toBe(true)
  })
})
