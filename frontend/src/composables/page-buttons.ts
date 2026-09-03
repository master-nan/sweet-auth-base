import { computed } from 'vue'
import { translate } from '@/boot/i18n'
import { useUserStore } from '@/stores/user'
import type { MenuButton } from '@/api/services/sys-menu'
import { SysMenuButtonPosition } from '@/types/enum'
import {
  findButtonActionCapability,
  findButtonCapability,
  resolvePageButtons,
} from '@/utils/menu-button'
import { resolveMenuButtonLabel } from '@/utils/menu-button-display'

export function usePageButtons(route_name: string) {
  const userStore = useUserStore()

  const grantedCapabilityCodes = computed(() => new Set(userStore.buttons))
  const hasGrantedCapability = (code: string) => grantedCapabilityCodes.value.has(code)

  const all_buttons = computed(() =>
    resolvePageButtons(userStore.menus, route_name).map((button) => ({
      ...button,
      name: resolveMenuButtonLabel(button, translate),
    })),
  )

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

  const findCapability = (code: string) =>
    findButtonCapability(all_buttons.value, code)
  const hasCapability = (code: string) => !!findCapability(code)
  const findActionCapability = (action: string) =>
    findButtonActionCapability(all_buttons.value, action)
  const hasActionCapability = (action: string) => !!findActionCapability(action)

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
    findCapability,
    hasCapability,
    findActionCapability,
    hasActionCapability,
    hasGrantedCapability,
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
