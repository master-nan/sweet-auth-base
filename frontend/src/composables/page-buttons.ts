import { computed } from 'vue'
import { useUserStore } from 'src/stores/user'
import type { Menu, MenuButton } from 'src/api/services/sys-menu'
import { SysMenuButtonPosition } from 'src/types/enum'
import { isPageButton } from 'src/utils/menu-button'

function findMenuByName(menus: Menu[], name: string): Menu | null {
  for (const menu of menus) {
    if (menu.name === name) return menu
    if (menu.children?.length) {
      const found = findMenuByName(menu.children, name)
      if (found) return found
    }
  }
  return null
}

export function usePageButtons(route_name: string) {
  const userStore = useUserStore()

  const all_buttons = computed(() => {
    const menu = findMenuByName(userStore.menus, route_name)
    if (!menu?.menu_buttons) return []
    return menu.menu_buttons
      .filter(isPageButton)
      .sort((a, b) => (a.sequence || 0) - (b.sequence || 0))
  })

  const line_buttons = computed(() =>
    all_buttons.value.filter((btn) => btn.position === SysMenuButtonPosition.LINE),
  )

  const top_buttons = computed(() =>
    all_buttons.value.filter((btn) => btn.position === SysMenuButtonPosition.TOP),
  )

  const bottom_buttons = computed(() =>
    all_buttons.value.filter((btn) => btn.position === SysMenuButtonPosition.BOTTOM),
  )

  const form_top_buttons = computed(() =>
    all_buttons.value.filter((btn) => btn.position === SysMenuButtonPosition.FORM_TOP),
  )

  const form_bottom_buttons = computed(() =>
    all_buttons.value.filter((btn) => btn.position === SysMenuButtonPosition.FORM_BOTTOM),
  )

  const record_detail_top_buttons = computed(() =>
    all_buttons.value.filter((btn) => btn.position === SysMenuButtonPosition.DETAIL_TOP),
  )

  const record_detail_bottom_buttons = computed(() =>
    all_buttons.value.filter((btn) => btn.position === SysMenuButtonPosition.DETAIL_BOTTOM),
  )

  const has_line_buttons = computed(() => line_buttons.value.length > 0)

  return {
    all_buttons,
    line_buttons,
    top_buttons,
    bottom_buttons,
    form_top_buttons,
    form_bottom_buttons,
    record_detail_top_buttons,
    record_detail_bottom_buttons,
    has_line_buttons,
  }
}

export function useMasterDetailPageButtons(
  route_name: string,
  isDetailButton: (btn: MenuButton) => boolean = () => false,
) {
  const pageButtons = usePageButtons(route_name)

  const master_buttons = computed(() =>
    pageButtons.all_buttons.value.filter((btn) => !isDetailButton(btn)),
  )
  const detail_buttons = computed(() => pageButtons.all_buttons.value.filter(isDetailButton))

  const byPosition = (buttons: typeof master_buttons, position: SysMenuButtonPosition) =>
    computed(() => buttons.value.filter((btn) => btn.position === position))

  const master_line_buttons = byPosition(master_buttons, SysMenuButtonPosition.LINE)
  const master_top_buttons = byPosition(master_buttons, SysMenuButtonPosition.TOP)
  const master_bottom_buttons = byPosition(master_buttons, SysMenuButtonPosition.BOTTOM)
  const master_form_top_buttons = byPosition(master_buttons, SysMenuButtonPosition.FORM_TOP)
  const master_form_bottom_buttons = byPosition(master_buttons, SysMenuButtonPosition.FORM_BOTTOM)

  const detail_line_buttons = byPosition(detail_buttons, SysMenuButtonPosition.LINE)
  const detail_top_buttons = byPosition(detail_buttons, SysMenuButtonPosition.TOP)
  const detail_bottom_buttons = byPosition(detail_buttons, SysMenuButtonPosition.BOTTOM)
  const detail_form_top_buttons = byPosition(detail_buttons, SysMenuButtonPosition.FORM_TOP)
  const detail_form_bottom_buttons = byPosition(detail_buttons, SysMenuButtonPosition.FORM_BOTTOM)

  const master_has_line_buttons = computed(() => master_line_buttons.value.length > 0)
  const detail_has_line_buttons = computed(() => detail_line_buttons.value.length > 0)

  return {
    ...pageButtons,
    master_buttons,
    detail_buttons,
    master_line_buttons,
    master_top_buttons,
    master_bottom_buttons,
    master_form_top_buttons,
    master_form_bottom_buttons,
    detail_line_buttons,
    detail_top_buttons,
    detail_bottom_buttons,
    detail_form_top_buttons,
    detail_form_bottom_buttons,
    master_has_line_buttons,
    detail_has_line_buttons,
  }
}

export type ButtonClickHandler = (btn: MenuButton, row?: any) => void
