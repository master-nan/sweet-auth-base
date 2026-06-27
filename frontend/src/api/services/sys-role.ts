import type { Basic, Query, ResponseData } from 'src/types/global'
import { instance } from 'boot/axios'
import type { MenuButton, Menu } from './sys-menu'
import type { User } from './sys-user'
import type { RoleDataPermissionSaveItem } from './data-permission'

export interface Role extends Basic {
  name: string
  memo?: string
  menus?: Array<Menu>
  buttons?: Array<MenuButton>
  users?: Array<User>
}

export interface RoleCreateReq {
  name: string
  memo?: string
}

export interface RoleUpdateReq extends RoleCreateReq {
  id: number
}

export interface RolePermissionReq {
  role_id: number
  menu_ids: Array<number>
  button_ids: Array<number>
  data_permissions?: RoleDataPermissionSaveItem[]
}

export const useRoleApi = () => {
  const queryRole = async (params: Query) => {
    return instance
      .post<ResponseData<Array<Role>>>('/admin/role/query', params)
      .then((res) => {
        return res.data
      })
  }

  const queryRoleById = async (id: number) => {
    return instance.get<ResponseData<Role>>(`/admin/role/${id}`).then((res) => {
      return res.data
    })
  }

  const createRole = async (req: RoleCreateReq) => {
    return instance.post<ResponseData<number>>('/admin/role', req).then((res) => {
      return res.data
    })
  }

  const updateRole = async (req: RoleUpdateReq) => {
    return instance.put<ResponseData<number>>(`/admin/role/${req.id}`, req).then((res) => {
      return res.data
    })
  }

  const deleteRole = async (id: number) => {
    return instance.delete<ResponseData<number>>(`/admin/role/${id}`).then((res) => {
      return res.data
    })
  }

  const assignPermissions = async (
    roleId: number,
    menuIds: number[],
    buttonIds: number[],
    dataPermissions?: RoleDataPermissionSaveItem[],
  ) => {
    const req: RolePermissionReq = {
      role_id: roleId,
      menu_ids: menuIds,
      button_ids: buttonIds,
    }
    if (dataPermissions) req.data_permissions = dataPermissions
    return instance.post<ResponseData<boolean>>('/admin/role/assign-permissions', req).then((res) => {
      return res.data
    })
  }

  return {
    queryRole,
    queryRoleById,
    createRole,
    updateRole,
    deleteRole,
    assignPermissions
  }
}
