import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('@/boot/axios', () => ({ instance: mocks }))

import { useQuerySchemeApi } from './query-scheme'

describe('query scheme API service', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset().mockResolvedValue({ data: { success: true, data: [] } }))
  })

  it('keeps runtime summary and resolve calls behind the service boundary', async () => {
    const api = useQuerySchemeApi()
    await api.available('system.user.list')
    await api.resolve(12, 'system.user.list', 3)

    expect(mocks.get).toHaveBeenCalledWith('/admin/runtime/query-schemes/available', expect.objectContaining({ params: { scope_code: 'system.user.list' } }))
    expect(mocks.post).toHaveBeenCalledWith('/admin/runtime/query-schemes/12/resolve', { scope_code: 'system.user.list', expected_revision: 3 }, expect.any(Object))
  })

  it('uses separate personal and shared management endpoints', async () => {
    const api = useQuerySchemeApi()
    const payload = { expressions: [], quick_query: { keyword: '' }, order: { field: '', is_asc: false }, bindings: [] }
    await api.createPersonal({ name: '我的方案', scope_code: 'system.user.list', query_payload: payload, is_default: false })
    await api.setSharedEnabled(9, false, 4)

    expect(mocks.post).toHaveBeenCalledWith('/admin/query-schemes/personal', expect.any(Object))
    expect(mocks.put).toHaveBeenCalledWith('/admin/query-schemes/shared/9/enabled', { enabled: false, revision: 4 })
  })
})
