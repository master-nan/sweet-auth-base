import type { Menu } from '@/api/services/sys-menu'
import { SysDetailOpenMode } from '@/types/enum'
import { findMenuByName } from '@/utils/menu-button'

export type OrganizationDetailMode = 'dialog' | 'page'

export const resolveOrganizationDetailMode = (
  menus: Menu[],
  routeName: string,
  autoMode: OrganizationDetailMode,
): OrganizationDetailMode => {
  const configuredMode = findMenuByName(menus, routeName)?.detail_open_mode
  if (configuredMode === SysDetailOpenMode.DIALOG) return 'dialog'
  if (configuredMode === SysDetailOpenMode.PAGE) return 'page'
  return autoMode
}
