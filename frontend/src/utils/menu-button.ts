import type { Menu, MenuButton } from 'src/api/services/sys-menu'

export function isPageButton(button: MenuButton): boolean {
  return button.is_button
}

export function isApiPermission(button: MenuButton): boolean {
  return !button.is_button
}

export function isAvailablePageButton(button: MenuButton): boolean {
  return (
    isPageButton(button) &&
    button.state !== false &&
    button.is_hidden !== true &&
    button.is_disabled !== true
  )
}

export function findMenuByName(menus: Menu[], name: string): Menu | null {
  for (const menu of menus) {
    if (menu.name === name) return menu
    if (menu.children?.length) {
      const found = findMenuByName(menu.children, name)
      if (found) return found
    }
  }
  return null
}

export function resolvePageButtons(menus: Menu[], routeName: string): MenuButton[] {
  const menu = findMenuByName(menus, routeName)
  if (!menu?.menu_buttons) return []
  return menu.menu_buttons
    .filter(isAvailablePageButton)
    .sort((a, b) => (a.sequence || 0) - (b.sequence || 0))
}
