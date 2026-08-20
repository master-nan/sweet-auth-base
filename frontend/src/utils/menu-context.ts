import type { Menu } from 'src/api/services/sys-menu'

export const toPositiveMenuId = (raw: unknown) => {
  const value = Array.isArray(raw) ? raw[0] : raw
  const id = Number(value)
  return Number.isFinite(id) && id > 0 ? id : 0
}

export const findMenuById = (menus: Menu[], menuId: number): Menu | null => {
  for (const menu of menus) {
    if (menu.id === menuId) return menu
    if (menu.children?.length) {
      const child = findMenuById(menu.children, menuId)
      if (child) return child
    }
  }
  return null
}

export const findMenuByName = (menus: Menu[], name: string): Menu | null => {
  if (!name) return null
  for (const menu of menus) {
    if (menu.name === name) return menu
    if (menu.children?.length) {
      const child = findMenuByName(menu.children, name)
      if (child) return child
    }
  }
  return null
}

export const findMenuByTableCode = (menus: Menu[], tableCode: string): Menu | null => {
  const code = tableCode.trim()
  if (!code) return null
  for (const menu of menus) {
    if (menuMatchesTableCode(menu, code) && menu.id) return menu
    if (menu.children?.length) {
      const child = findMenuByTableCode(menu.children, code)
      if (child) return child
    }
  }
  return null
}

export type MenuTrailItem = {
  menu: Menu
  fullPath: string
}

export const findMenuTrailById = (
  menus: Menu[],
  menuId: number,
  basePath = '/admin',
): MenuTrailItem[] => {
  if (!menuId) return []
  return findMenuTrail(menus, menuId, normalizeMenuPath(basePath), [], '') || []
}

export const findMenuPathByTableCode = (
  menus: Menu[],
  tableCode: string,
  basePath = '/admin',
): string | null => {
  const code = tableCode.trim()
  if (!code) return null
  return findMenuPath(menus, code, normalizeMenuPath(basePath))
}

const findMenuTrail = (
  menus: Menu[],
  menuId: number,
  parentPath: string,
  trail: MenuTrailItem[],
  fallbackTableCode: string,
): MenuTrailItem[] | null => {
  for (const menu of menus) {
    const tableCode = String(menu.table_code || fallbackTableCode || '').trim()
    const currentPath = joinMenuPath(parentPath, resolveMenuPath(menu.path, tableCode))
    const nextTrail = [
      ...trail,
      {
        menu,
        fullPath: currentPath,
      },
    ]
    if (menu.id === menuId) return nextTrail
    if (menu.children?.length) {
      const childTrail = findMenuTrail(menu.children, menuId, currentPath, nextTrail, tableCode)
      if (childTrail) return childTrail
    }
  }
  return null
}

const findMenuPath = (menus: Menu[], tableCode: string, parentPath: string): string | null => {
  for (const menu of menus) {
    const currentPath = joinMenuPath(parentPath, resolveMenuPath(menu.path, tableCode))
    if (menuMatchesTableCode(menu, tableCode) && menu.id) return currentPath
    if (menu.children?.length) {
      const childPath = findMenuPath(menu.children, tableCode, currentPath)
      if (childPath) return childPath
    }
  }
  return null
}

const resolveMenuPath = (path: string, tableCode: string) => {
  return String(path || '').replace(':table_code', tableCode)
}

const joinMenuPath = (basePath: string, itemPath: string) => {
  if (!itemPath) return normalizeMenuPath(basePath)
  if (itemPath.includes('http')) return itemPath
  return normalizeMenuPath(`${basePath}/${itemPath}`)
}

const normalizeMenuPath = (value: string) => {
  const cleanValue = (value || '/').split(/[?#]/)[0] || '/'
  if (cleanValue.includes('http')) return cleanValue
  const withLeadingSlash = cleanValue.startsWith('/') ? cleanValue : `/${cleanValue}`
  return withLeadingSlash.replace(/\/+/g, '/').replace(/\/$/, '') || '/'
}

const menuMatchesTableCode = (menu: Menu, tableCode: string) => {
  return String(menu.table_code || '').trim() === tableCode
}
