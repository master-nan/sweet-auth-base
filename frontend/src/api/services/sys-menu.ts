import type { Basic, Query, ResponseData } from 'src/types/global'
import type {
  SysDetailOpenMode,
  SysMenuButtonDisplayMode,
  SysMenuButtonPosition,
} from 'src/types/enum'
import { type Role } from 'src/api/services/sys-role'
import { instance } from 'boot/axios'

export interface MenuButton extends Basic {
  menu_id: number
  name: string
  code: string
  icon: string
  color: string
  display_mode?: SysMenuButtonDisplayMode | string
  sequence: number
  memo: string
  position: SysMenuButtonPosition
  event_type: string
  event_action: string
  api_path: string
  http_method: string
  disable_when: string
  params_schema: string
  confirm_text: string
  is_button: boolean
  is_hidden?: boolean
  is_disabled: boolean
  before_hooks?: string
  after_hooks?: string
  roles?: Array<Role>
  menus?: Array<Menu>
}

export interface Menu extends Basic {
  pid: number
  name: string
  path: string
  component: string
  title: string
  is_hidden: boolean
  sequence: number
  page_type?: 'directory' | 'fixed' | 'low_code' | string
  table_code?: string
  detail_open_mode?: SysDetailOpenMode
  option: string
  icon?: string
  redirect?: string
  is_unfold?: boolean
  menu_buttons?: Array<MenuButton>
  children?: Array<Menu>
}

export interface MenuCreateReq {
  pid: number
  name: string
  path: string
  component: string
  title: string
  is_hidden: boolean
  sequence: number
  page_type?: 'directory' | 'fixed' | 'low_code' | string
  table_code?: string
  option: string
  icon?: string
  redirect?: string
}

export interface MenuUpdateReq extends MenuCreateReq {
  id: number
}

export interface MenuButtonCreateReq {
  menu_id: number
  name: string
  code: string
  icon: string
  color: string
  display_mode?: SysMenuButtonDisplayMode | string
  sequence: number
  memo: string
  position: SysMenuButtonPosition
  event_type: string
  event_action: string
  api_path: string
  http_method: string
  disable_when: string
  params_schema: string
  confirm_text: string
  is_button?: boolean
  is_hidden?: boolean
  is_disabled: boolean
  before_hooks?: string
  after_hooks?: string
  roles?: Array<Role>
  menus?: Array<Menu>
}

export interface MenuButtonUpdateReq extends MenuButtonCreateReq {
  id?: number
}

// 添加菜单排序接口
export interface MenuOrderUpdateReq {
  menus: Array<{
    id: number
    sequence: number
  }>
}

export const useMenuApi = () => {
  const queryMenu = async (params: Query) => {
    return instance.post<ResponseData<Array<Menu>>>('/admin/menu/query', params).then((res) => {
      return res.data
    })
  }
  const queryMenuById = async (id: number) => {
    return instance.get<ResponseData<Array<Menu>>>(`/admin/menu/${id}`).then((res) => {
      return res.data
    })
  }

  const queryMyMenu = async () => {
    return instance.get<ResponseData<Array<Menu>>>('/admin/menu/my').then((res) => {
      return res.data
    })
  }

  const queryUserMenus = async (userId: number) => {
    return instance.get<ResponseData<Array<Menu>>>(`/admin/menu/user/${userId}`).then((res) => {
      return res.data
    })
  }

  const createMenu = async (req: MenuCreateReq) => {
    return instance.post<ResponseData<number>>('/admin/menu', req).then((res) => {
      return res.data
    })
  }
  const updateMenu = async (req: MenuUpdateReq) => {
    return instance.put<ResponseData<number>>(`/admin/menu/${req.id}`, req).then((res) => {
      return res.data
    })
  }
  const deleteMenu = async (id: number) => {
    return instance.delete<ResponseData<number>>(`/admin/menu/${id}`).then((res) => {
      return res.data
    })
  }

  // 菜单按钮相关API
  const queryMenuButtons = async (menuId: number) => {
    return instance
      .get<ResponseData<Array<MenuButton>>>(`/admin/menu/buttons/${menuId}`)
      .then((res) => {
        return res.data
      })
  }

  const createMenuButton = async (req: MenuButtonCreateReq) => {
    return instance.post<ResponseData<number>>('/admin/menu/button', req).then((res) => {
      return res.data
    })
  }

  const updateMenuButton = async (req: MenuButtonUpdateReq) => {
    return instance.put<ResponseData<number>>(`/admin/menu/button/${req.id}`, req).then((res) => {
      return res.data
    })
  }

  const deleteMenuButton = async (id: number) => {
    return instance.delete<ResponseData<number>>(`/admin/menu/button/${id}`).then((res) => {
      return res.data
    })
  }

  // 添加菜单排序API
  const updateMenuOrder = async (menus: Menu[]) => {
    // 构建排序请求
    const orderRequest: MenuOrderUpdateReq = {
      menus: menus.map((menu, index) => ({
        id: menu.id,
        sequence: index + 1, // 设置排序序号
      })),
    }

    return instance.put<ResponseData<boolean>>('/admin/menu/order', orderRequest).then((res) => {
      return res.data
    })
  }

  const refreshMenuCache = async () => {
    return instance.post<ResponseData<boolean>>('/admin/menu/refresh-cache').then((res) => {
      return res.data
    })
  }

  return {
    queryMenu,
    queryMenuById,
    queryMyMenu,
    queryUserMenus,
    createMenu,
    updateMenu,
    deleteMenu,
    queryMenuButtons,
    createMenuButton,
    updateMenuButton,
    deleteMenuButton,
    updateMenuOrder,
    refreshMenuCache,
  }
}
