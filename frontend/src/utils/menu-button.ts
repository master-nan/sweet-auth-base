import type { Menu, MenuButton } from '@/api/services/sys-menu'

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

export const findButtonCapability = (buttons: MenuButton[], code: string) =>
  buttons.find((button) => isAvailablePageButton(button) && button.code === code)

export const hasButtonCapability = (buttons: MenuButton[], code: string) =>
  !!findButtonCapability(buttons, code)

export const findButtonActionCapability = (buttons: MenuButton[], action: string) =>
  buttons.find((button) => isAvailablePageButton(button) && button.event_action === action)

export const hasButtonActionCapability = (buttons: MenuButton[], action: string) =>
  !!findButtonActionCapability(buttons, action)

export function hasGrantedActionCapability(menus: Menu[], action: string): boolean {
  for (const menu of menus) {
    if (menu.menu_buttons?.some((button) =>
      button.state !== false &&
      button.is_disabled !== true &&
      button.event_action === action,
    )) {
      return true
    }
    if (menu.children?.length && hasGrantedActionCapability(menu.children, action)) {
      return true
    }
  }
  return false
}
