import { describe, expect, it } from 'vitest'
import { vi } from 'vitest'
vi.mock('src/api/services/query-scheme', () => ({ useQuerySchemeApi: () => ({}) }))
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ menus: [] }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ name: '', meta: {} }) }))
import { collectQueryScopes } from './query-scope'

describe('query scope menu projection', () => {
  it('uses backend menu scope identities and labels without a frontend mapping', () => {
    const scopes = collectQueryScopes([{ id: 1, pid: 0, name: 'system', path: '', component: '', title: '系统', is_hidden: false, sequence: 1, option: '', query_scope_code: undefined, children: [{ id: 2, pid: 1, name: 'system_user', path: 'user', component: '', title: '用户管理', is_hidden: false, sequence: 1, option: '', query_scope_code: 'system.user.list' }] }] as never)
    expect(scopes).toEqual([{ scope_code: 'system.user.list', scope_label: '用户管理', route_name: 'system_user' }])
  })
})
