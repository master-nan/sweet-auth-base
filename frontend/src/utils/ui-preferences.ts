import { LocalStorage } from 'quasar'

export const UI_PREFERENCES_KEY = 'sweet-admin.ui-preferences.v1'
export const UI_PREFERENCES_VERSION = 1
export const DEFAULT_PRIMARY_COLOR = '#7367f0'

export type LayoutMode = 'split' | 'full'
export type DisplayDensity = 'comfortable' | 'compact'
export type SupportedLocale = 'en-US' | 'zh-CN'

export interface UIPreferences {
  version: typeof UI_PREFERENCES_VERSION
  layoutMode: LayoutMode
  density: DisplayDensity
  primaryColor: string
  dark: boolean
  locale: SupportedLocale
  drawerMini: boolean
}

const hexColorPattern = /^#[0-9a-f]{6}$/i

const isLayoutMode = (value: unknown): value is LayoutMode =>
  value === 'split' || value === 'full'

const isDisplayDensity = (value: unknown): value is DisplayDensity =>
  value === 'comfortable' || value === 'compact'

const isSupportedLocale = (value: unknown): value is SupportedLocale =>
  value === 'en-US' || value === 'zh-CN'

const isBoolean = (value: unknown): value is boolean => typeof value === 'boolean'

const isHexColor = (value: unknown): value is string =>
  typeof value === 'string' && hexColorPattern.test(value)

export const defaultUIPreferences = (): UIPreferences => ({
  version: UI_PREFERENCES_VERSION,
  layoutMode: 'split',
  density: 'comfortable',
  primaryColor: DEFAULT_PRIMARY_COLOR,
  dark: (LocalStorage.getItem('dark') as boolean | null) ?? false,
  locale: (LocalStorage.getItem('lang') as SupportedLocale | null) ?? 'zh-CN',
  drawerMini: false,
})

export const readUIPreferences = (): UIPreferences => {
  const defaults = defaultUIPreferences()
  const stored = LocalStorage.getItem<Partial<UIPreferences>>(UI_PREFERENCES_KEY)

  if (!stored || typeof stored !== 'object') return defaults

  return {
    version: UI_PREFERENCES_VERSION,
    layoutMode: isLayoutMode(stored.layoutMode) ? stored.layoutMode : defaults.layoutMode,
    density: isDisplayDensity(stored.density) ? stored.density : defaults.density,
    primaryColor: isHexColor(stored.primaryColor)
      ? stored.primaryColor
      : defaults.primaryColor,
    dark: isBoolean(stored.dark) ? stored.dark : defaults.dark,
    locale: isSupportedLocale(stored.locale) ? stored.locale : defaults.locale,
    drawerMini: isBoolean(stored.drawerMini) ? stored.drawerMini : defaults.drawerMini,
  }
}

export const writeUIPreferences = (patch: Partial<Omit<UIPreferences, 'version'>>) => {
  const next: UIPreferences = {
    ...readUIPreferences(),
    ...patch,
    version: UI_PREFERENCES_VERSION,
  }

  LocalStorage.set(UI_PREFERENCES_KEY, next)

  // 同步写入Theme和语言Store仍在读取的独立键，保持当前多个入口使用同一偏好值。
  LocalStorage.set('dark', next.dark)
  LocalStorage.set('lang', next.locale)

  return next
}
