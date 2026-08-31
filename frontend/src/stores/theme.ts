import { defineStore } from 'pinia'
import { colors, setCssVar, Dark } from 'quasar'
import {
  DEFAULT_PRIMARY_COLOR,
  readUIPreferences,
  writeUIPreferences,
  type DisplayDensity,
  type LayoutMode,
} from 'src/utils/ui-preferences'

const { changeAlpha, getPaletteColor, lighten, luminosity } = colors
const preferences = readUIPreferences()
const primaryColor = preferences.primaryColor || getPaletteColor('primary')
const darkColor = '#202638'
const darkPageColor = '#161b28'

interface ThemeColor {
  primary: string
  layoutMode: LayoutMode
  density: DisplayDensity
  dark: boolean
}

export const useThemeStore = defineStore('theme', {
  state: (): ThemeColor => ({
    primary: primaryColor,
    layoutMode: preferences.layoutMode,
    density: preferences.density,
    dark: preferences.dark,
  }),

  getters: {
    primaryColor(state): string {
      return state.primary
    },
    baseBgColor(): string {
      return Dark.isActive ? darkPageColor : '#f7f9fa'
    },

    activeBgColor(state): string {
      if (Dark.isActive) {
        return lighten(darkColor, 15)
      }
      return luminosity(state.primary) > 0.4
        ? lighten(state.primary, -40)
        : lighten(state.primary, 90)
    },
    activeTextColor(state): string {
      if (Dark.isActive) {
        return '#ffffff'
      }
      return luminosity(state.primary) > 0.4 ? '#ffffff' : state.primary
    },

    drawerBgColor(state): string {
      return Dark.isActive ? darkPageColor : state.primary
    },

    drawerTextColor(state): string {
      return luminosity(state.primary) > 0.4 ? '#000000' : '#ffffff'
    },

    headerBgColor(): string {
      return Dark.isActive ? darkColor : '#ffffff'
    },
    headerTextColor(): string {
      return Dark.isActive ? '#ffffff' : '#000000'
    },
  },

  actions: {
    applyPreferences() {
      this.applyThemeColor(this.primary)
      this.applyDensity(this.density)
      Dark.set(this.dark)
    },
    applyThemeColor(color: string) {
      this.primary = color
      setCssVar('primary', color)
      document.documentElement.style.setProperty('--app-primary-soft', changeAlpha(color, 0.08))
      document.documentElement.style.setProperty(
        '--app-primary-soft-strong',
        changeAlpha(color, 0.16),
      )
      document.documentElement.style.setProperty('--app-primary-border', changeAlpha(color, 0.28))
      document.documentElement.style.setProperty('--app-primary-shadow', changeAlpha(color, 0.2))
    },
    setThemeColor(color: string) {
      this.applyThemeColor(color)
      writeUIPreferences({ primaryColor: color })
    },
    resetThemeColor() {
      this.setThemeColor(DEFAULT_PRIMARY_COLOR)
    },
    setLayoutMode(mode: LayoutMode) {
      this.layoutMode = mode
      writeUIPreferences({ layoutMode: mode })
    },
    applyDensity(density: DisplayDensity) {
      this.density = density
      document.body.classList.toggle('app-density--compact', density === 'compact')
    },
    setDensity(density: DisplayDensity) {
      this.applyDensity(density)
      writeUIPreferences({ density })
    },
    setDarkMode(dark: boolean) {
      this.dark = dark
      Dark.set(dark)
      writeUIPreferences({ dark })
    },
  },
})
