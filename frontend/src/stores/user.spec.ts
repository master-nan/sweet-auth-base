import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const storage = vi.hoisted(() => new Map<string, unknown>())
const navigation = vi.hoisted(() => ({
  push: vi.fn(),
  hasRoute: vi.fn(() => true),
  removeRoute: vi.fn(),
}))
const tagView = vi.hoisted(() => ({ removeAllTagView: vi.fn() }))
const configure = vi.hoisted(() => ({ $reset: vi.fn() }))
const permission = vi.hoisted(() => ({ $reset: vi.fn() }))
const keepAlive = vi.hoisted(() => ({ $reset: vi.fn() }))
const breadcrumbs = vi.hoisted(() => ({ $reset: vi.fn() }))

vi.mock('quasar', () => ({
  LocalStorage: {
    getItem: (key: string) => storage.get(key) ?? null,
    remove: (key: string) => storage.delete(key),
    set: (key: string, value: unknown) => storage.set(key, value),
  },
}))
vi.mock('src/router/index', () => ({ Router: navigation }))
vi.mock('./tagView', () => ({ useTagViewStore: () => tagView }))
vi.mock('./configure', () => ({ useConfigureStore: () => configure }))
vi.mock('./permission', () => ({ useRouterStore: () => permission }))
vi.mock('./keep-alive', () => ({ useKeepAliveStore: () => keepAlive }))
vi.mock('./breadcrumbs', () => ({ useBreadcrumbsStore: () => breadcrumbs }))

import { LocalStorage } from 'quasar'
import { UI_PREFERENCES_KEY } from 'src/utils/ui-preferences'
import { isStaleSessionSnapshot, useUserStore } from './user'

const unsignedToken = (subject: string, sessionID: string, tokenID: string) => {
  const header = btoa(JSON.stringify({ alg: 'none', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ sub: subject, sid: sessionID, jti: tokenID }))
  return `${header}.${payload}.signature`
}

describe('user session state', () => {
  beforeEach(() => {
    storage.clear()
    navigation.push.mockReset()
    tagView.removeAllTagView.mockReset()
    configure.$reset.mockReset()
    permission.$reset.mockReset()
    keepAlive.$reset.mockReset()
    breadcrumbs.$reset.mockReset()
    navigation.hasRoute.mockClear()
    navigation.removeRoute.mockReset()
    setActivePinia(createPinia())
  })

  it('clears authentication state on logout without removing persistent UI preferences', () => {
    LocalStorage.set('access_token', 'token')
    LocalStorage.set('must_change_password', true)
    LocalStorage.set('password_change_reason', 'initial_reset')
    LocalStorage.set(UI_PREFERENCES_KEY, {
      primaryColor: '#0f766e',
      dark: true,
      layoutMode: 'full',
    })
    LocalStorage.set('dark', true)
    LocalStorage.set('lang', 'en-US')
    LocalStorage.set('unrelated-setting', 'keep')

    const user = useUserStore()
    user.setLogout()

    expect(LocalStorage.getItem('access_token')).toBeNull()
    expect(LocalStorage.getItem('must_change_password')).toBeNull()
    expect(LocalStorage.getItem('password_change_reason')).toBeNull()
    expect(LocalStorage.getItem(UI_PREFERENCES_KEY)).toEqual({
      primaryColor: '#0f766e',
      dark: true,
      layoutMode: 'full',
    })
    expect(LocalStorage.getItem('dark')).toBe(true)
    expect(LocalStorage.getItem('lang')).toBe('en-US')
    expect(LocalStorage.getItem('unrelated-setting')).toBe('keep')
    expect(user.access_token).toBe('')
    expect(tagView.removeAllTagView).toHaveBeenCalledOnce()
    expect(configure.$reset).toHaveBeenCalledOnce()
    expect(permission.$reset).toHaveBeenCalledOnce()
    expect(keepAlive.$reset).toHaveBeenCalledOnce()
    expect(breadcrumbs.$reset).toHaveBeenCalledOnce()
    expect(navigation.removeRoute).toHaveBeenCalledWith('admin')
    expect(navigation.push).toHaveBeenCalledWith({ name: 'Login' })
  })

  it('atomically clears account-bound state before switching to another token', () => {
    LocalStorage.set('access_token', 'token-a')
    const user = useUserStore()
    user.user_name = 'account-a'
    user.buttons = ['system_user_query']
    user.menu_names = ['system_user']

    user.setLoginToken('token-b')

    expect(user.access_token).toBe('token-b')
    expect(LocalStorage.getItem('access_token')).toBe('token-b')
    expect(user.user_name).toBe('')
    expect(user.buttons).toEqual([])
    expect(user.menu_names).toEqual([])
    expect(permission.$reset).toHaveBeenCalledOnce()
    expect(tagView.removeAllTagView).toHaveBeenCalledOnce()
    expect(navigation.removeRoute).toHaveBeenCalledWith('admin')
  })

  it('classifies a late response from session A as stale after session B starts', () => {
    expect(isStaleSessionSnapshot('token-a', 1, 'token-b', 2, 'token-b')).toBe(true)
    expect(isStaleSessionSnapshot('token-b', 2, 'token-b', 2, 'token-b')).toBe(false)
  })

  it('keeps account-bound state when the same login session rotates its access token', () => {
    const firstToken = unsignedToken('42', 'session-a', 'first')
    const refreshedToken = unsignedToken('42', 'session-a', 'refreshed')
    LocalStorage.set('access_token', firstToken)
    const user = useUserStore()
    user.user_name = 'admin'
    user.buttons = ['system_user_query']

    user.syncPersistedSession(refreshedToken)

    expect(user.access_token).toBe(refreshedToken)
    expect(user.user_name).toBe('admin')
    expect(user.buttons).toEqual(['system_user_query'])
    expect(permission.$reset).not.toHaveBeenCalled()
    expect(tagView.removeAllTagView).not.toHaveBeenCalled()
  })

  it('resets account-bound state when the same user starts a different login session', () => {
    const firstToken = unsignedToken('42', 'session-a', 'first')
    const nextSessionToken = unsignedToken('42', 'session-b', 'second')
    LocalStorage.set('access_token', firstToken)
    const user = useUserStore()
    user.user_name = 'admin'
    user.buttons = ['system_user_query']

    user.syncPersistedSession(nextSessionToken)

    expect(user.access_token).toBe(nextSessionToken)
    expect(user.user_name).toBe('')
    expect(user.buttons).toEqual([])
    expect(permission.$reset).toHaveBeenCalledOnce()
    expect(tagView.removeAllTagView).toHaveBeenCalledOnce()
  })
})
