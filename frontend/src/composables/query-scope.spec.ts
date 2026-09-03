import { beforeEach, describe, expect, it, vi } from 'vitest'
const route = vi.hoisted(() => ({ name: '', meta: {} as Record<string, unknown> }))
const user = vi.hoisted(() => ({ menus: [] as never[] }))
vi.mock('@/api/services/query-scheme', () => ({ useQuerySchemeApi: () => ({}) }))
vi.mock('@/stores/user', () => ({ useUserStore: () => user }))
vi.mock('vue-router', () => ({ useRoute: () => route }))
import { collectQueryScopes, useQueryScope } from './query-scope'

describe('query scope menu projection', () => {
  beforeEach(() => {
    route.name = ''
    route.meta = {}
    user.menus = []
  })

  it('uses backend menu scope identities and labels without a frontend mapping', () => {
    const scopes = collectQueryScopes([{ id: 1, pid: 0, name: 'system', path: '', component: '', title: '系统', is_hidden: false, sequence: 1, option: '', query_scope_code: undefined, children: [{ id: 2, pid: 1, name: 'system_user', path: 'user', component: '', title: '用户管理', is_hidden: false, sequence: 1, option: '', query_scope_code: 'system.user.list' }] }] as never)
    expect(scopes).toEqual([{ scope_code: 'system.user.list', scope_label: '用户管理', route_name: 'system_user' }])
  })

  it('does not accept a route meta value as a scope identity', () => {
    route.name = 'system_user'
    route.meta = { queryScopeCode: 'forged.scope' }
    user.menus = [{
      id: 2,
      name: 'system_user',
      title: '用户管理',
      children: [],
    }] as never[]

    expect(useQueryScope().scopeCode.value).toBe('')
  })
})
