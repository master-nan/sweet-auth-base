import type { MenuButton } from 'src/api/services/sys-menu'

export function isPageButton(button: MenuButton): boolean {
  return button.is_button
}

export function isApiPermission(button: MenuButton): boolean {
  return !isPageButton(button)
}
