import { beforeEach, describe, expect, it, vi } from 'vitest'

const storage = vi.hoisted(() => new Map<string, unknown>())

vi.mock('quasar', () => ({
  LocalStorage: {
    getItem: (key: string) => storage.get(key) ?? null,
    remove: (key: string) => storage.delete(key),
    set: (key: string, value: unknown) => storage.set(key, value),
  },
}))

import { LocalStorage } from 'quasar'
import { UI_PREFERENCES_KEY } from './ui-preferences'
import { clearAuthSessionStorage } from './auth-session-storage'

describe('auth session storage', () => {
  beforeEach(() => {
    storage.clear()
  })

  it('clears authentication state without removing persistent UI preferences', () => {
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

    clearAuthSessionStorage()

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
  })
})
