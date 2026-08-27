import type { MenuButton } from 'src/api/services/sys-menu'
import { SysMenuButtonDisplayMode, SysMenuButtonPosition } from 'src/types/enum'

interface MenuButtonDisplayOptions {
  label?: string
  icon?: string | undefined
  position?: SysMenuButtonPosition
}

const normalizeDisplayMode = (mode?: string) => {
  const normalized = (mode || SysMenuButtonDisplayMode.AUTO).trim().toLowerCase()
  if (normalized === 'icon') return SysMenuButtonDisplayMode.ICON
  if (normalized === 'text') return SysMenuButtonDisplayMode.TEXT
  if (normalized === 'icon_text') return SysMenuButtonDisplayMode.ICON_TEXT
  return SysMenuButtonDisplayMode.AUTO
}

const defaultDisplayMode = (position?: SysMenuButtonPosition) => {
  if (position === SysMenuButtonPosition.LINE) return SysMenuButtonDisplayMode.ICON
  if (position === SysMenuButtonPosition.FORM_BOTTOM) return SysMenuButtonDisplayMode.TEXT
  return SysMenuButtonDisplayMode.ICON_TEXT
}

export const menuButtonDisplayProps = (btn: MenuButton, options: MenuButtonDisplayOptions = {}) => {
  const label = options.label || btn.name
  const icon = (options.icon ?? btn.icon) || undefined
  const position = options.position || btn.position
  const configuredMode = normalizeDisplayMode(String(btn.display_mode || ''))
  const mode =
    configuredMode === SysMenuButtonDisplayMode.AUTO ? defaultDisplayMode(position) : configuredMode

  const showIcon = !!icon && mode !== SysMenuButtonDisplayMode.TEXT
  const showText = mode !== SysMenuButtonDisplayMode.ICON || !icon

  return {
    icon: showIcon ? icon : undefined,
    label: showText ? label : undefined,
    round: showIcon && !showText,
    'aria-label': label,
  }
}
