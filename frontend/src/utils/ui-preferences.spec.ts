import { beforeEach, describe, expect, it, vi } from 'vitest'

const storage = vi.hoisted(() => new Map<string, unknown>())

vi.mock('quasar', () => ({
  LocalStorage: {
    clear: () => storage.clear(),
    getItem: (key: string) => storage.get(key) ?? null,
    set: (key: string, value: unknown) => storage.set(key, value),
  },
}))

import { LocalStorage } from 'quasar'
import {
  UI_PREFERENCES_KEY,
  defaultUIPreferences,
  readUIPreferences,
  writeUIPreferences,
} from './ui-preferences'

describe('ui preferences', () => {
  beforeEach(() => {
    LocalStorage.clear()
  })

  it('returns stable defaults', () => {
    expect(defaultUIPreferences()).toEqual({
      version: 1,
      layoutMode: 'split',
      primaryColor: '#7367f0',
      dark: false,
      locale: 'zh-CN',
      drawerMini: false,
    })
  })

  it('migrates legacy dark mode and locale values', () => {
    LocalStorage.set('dark', true)
    LocalStorage.set('lang', 'en-US')

    expect(readUIPreferences()).toMatchObject({
      dark: true,
      locale: 'en-US',
    })
  })

  it('persists a versioned preference object and keeps legacy keys compatible', () => {
    const saved = writeUIPreferences({
      layoutMode: 'full',
      primaryColor: '#0f766e',
      dark: true,
      locale: 'en-US',
      drawerMini: true,
    })

    expect(LocalStorage.getItem(UI_PREFERENCES_KEY)).toEqual(saved)
    expect(LocalStorage.getItem('dark')).toBe(true)
    expect(LocalStorage.getItem('lang')).toBe('en-US')
    expect(readUIPreferences()).toEqual(saved)
  })

  it('falls back safely when stored values are invalid', () => {
    LocalStorage.set(UI_PREFERENCES_KEY, {
      layoutMode: 'unknown',
      primaryColor: 'purple',
      dark: 'yes',
      locale: 'fr-FR',
      drawerMini: 1,
    })

    expect(readUIPreferences()).toEqual(defaultUIPreferences())
  })
})
