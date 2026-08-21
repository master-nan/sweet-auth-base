import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const storage = vi.hoisted(() => new Map<string, unknown>())
const navigation = vi.hoisted(() => ({ push: vi.fn() }))
const tagView = vi.hoisted(() => ({ removeAllTagView: vi.fn() }))
const configure = vi.hoisted(() => ({ $reset: vi.fn() }))

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

import { LocalStorage } from 'quasar'
import { UI_PREFERENCES_KEY } from 'src/utils/ui-preferences'
import { useUserStore } from './user'

describe('user session state', () => {
  beforeEach(() => {
    storage.clear()
    navigation.push.mockReset()
    tagView.removeAllTagView.mockReset()
    configure.$reset.mockReset()
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
    expect(navigation.push).toHaveBeenCalledWith({ name: 'Login' })
  })
})
