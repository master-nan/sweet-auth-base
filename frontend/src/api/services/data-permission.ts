import { instance } from 'boot/axios'
import type { Basic, Query, ResponseData } from 'src/types/global'
import type { Menu } from 'src/api/services/sys-menu'

export type DataPermissionValueType = 'string' | 'number'
export type DataPermissionSourceType = 'none' | 'table'
export type DataPermissionMatchType = 'in' | 'eq'
export type DataPermissionAction =
  | 'query'
  | 'detail'
  | 'create'
  | 'update'
  | 'delete'
  | 'export'
  | 'batch_delete'
export type DataPermissionStrategy = 'all' | 'none' | 'specified' | 'tree' | 'self' | 'user_dimension'
export type DataPermissionOverrideMode = 'replace' | 'union' | 'intersect' | 'deny'

export interface DataPermissionOption {
  label: string
  value: string
  parent?: string
}

export interface DataPermissionDimension extends Basic {
  code: string
  name: string
  value_type: DataPermissionValueType | string
  source_type: DataPermissionSourceType | string
  source_code: string
  label_field: string
  value_field: string
  parent_field: string
  memo: string
}

export interface DataPermissionDimensionSaveReq {
  id?: number
  code: string
  name: string
  value_type: DataPermissionValueType | string
  source_type: DataPermissionSourceType | string
  source_code: string
  label_field: string
  value_field: string
  parent_field: string
  memo: string
  state?: boolean
}

export interface DataPermissionBinding extends Basic {
  menu_id: number
  table_code: string
  dimension_code: string
  field_code: string
  match_type: DataPermissionMatchType | string
  required: boolean
  actions: DataPermissionAction[] | string[]
  dimension?: DataPermissionDimension
  menu?: Menu
}

export interface DataPermissionBindingSaveItem {
  id?: number
  dimension_code: string
  field_code: string
  match_type: DataPermissionMatchType | string
  required?: boolean
  actions: DataPermissionAction[] | string[]
  state?: boolean
}

export interface RoleDataPermission extends Basic {
  role_id: number
  menu_id: number
  table_code: string
  dimension_code: string
  strategy: DataPermissionStrategy | string
  scope_values: string[]
  menu?: Menu
  dimension?: DataPermissionDimension
}

export interface RoleDataPermissionSaveItem {
  menu_id: number
  table_code?: string
  dimension_code: string
  strategy: DataPermissionStrategy | string
  scope_values: string[]
  state?: boolean
}

export interface UserDataPermissionOverride extends Basic {
  user_id: number
  menu_id: number
  table_code: string
  dimension_code: string
  strategy: DataPermissionStrategy | string
  scope_values: string[]
  override_mode: DataPermissionOverrideMode | string
  expire_at?: string | null
  menu?: Menu
  dimension?: DataPermissionDimension
}

export interface UserDimensionValue extends Basic {
  user_id: number
  dimension_code: string
  scope_values: string[]
  dimension?: DataPermissionDimension
}

export interface UserDataPermissionOverrideSaveItem {
  menu_id: number
  table_code?: string
  dimension_code: string
  strategy: DataPermissionStrategy | string
  scope_values: string[]
  override_mode: DataPermissionOverrideMode | string
  expire_at?: string
  state?: boolean
}

export interface UserDimensionValueSaveItem {
  dimension_code: string
  scope_values: string[]
  state?: boolean
}

export interface DataPermissionScopeCondition {
  DimensionCode?: string
  Field?: string
  MatchType?: string
  ValueType?: string
  Values?: string[]
  dimension_code: string
  field: string
  match_type: string
  value_type: string
  values: string[]
}

export interface DataPermissionResolvedScope {
  AllowAll?: boolean
  DenyAll?: boolean
  Conditions?: DataPermissionScopeCondition[]
  allow_all?: boolean
  deny_all?: boolean
  conditions?: DataPermissionScopeCondition[]
}

export interface DataPermissionDebugResult {
  user_id: number
  user_name: string
  menu_id: number
  table_code: string
  action: DataPermissionAction | string
  role_ids: number[]
  scope?: DataPermissionResolvedScope
  bindings: DataPermissionBinding[]
  role_scopes: RoleDataPermission[]
  user_overrides: UserDataPermissionOverride[]
  user_dimensions: UserDimensionValue[]
  notes: string[]
}

export const dataPermissionActionOptions = [
  { label: '查询', value: 'query' },
  { label: '详情', value: 'detail' },
  { label: '新增', value: 'create' },
  { label: '编辑', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '导出', value: 'export' },
  { label: '批量删除', value: 'batch_delete' },
]

export const dataPermissionStrategyOptions = [
  { label: '全部', value: 'all' },
  { label: '无权限', value: 'none' },
  { label: '指定值', value: 'specified' },
  { label: '树范围', value: 'tree' },
  { label: '用户归属', value: 'user_dimension' },
  { label: '本人ID', value: 'self' },
]

export const dataPermissionOverrideModeOptions = [
  { label: '替换角色范围', value: 'replace' },
  { label: '并集追加', value: 'union' },
  { label: '交集收窄', value: 'intersect' },
  { label: '拒绝访问', value: 'deny' },
]

export const splitScopeValues = (value: string | string[] | undefined | null) => {
  if (Array.isArray(value)) return value.map(String).map((item) => item.trim()).filter(Boolean)
  return String(value || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

export const joinScopeValues = (values: string[] | undefined | null) => (values || []).join(',')

export const useDataPermissionApi = () => {
  const queryDimensions = async (params: Query) => {
    return instance
      .post<ResponseData<DataPermissionDimension[]>>('/admin/data-permission/dimension/query', params)
      .then((res) => res.data)
  }

  const getDimensionById = async (id: number) => {
    return instance
      .get<ResponseData<DataPermissionDimension>>(`/admin/data-permission/dimension/${id}`)
      .then((res) => res.data)
  }

  const createDimension = async (req: DataPermissionDimensionSaveReq) => {
    return instance.post<ResponseData<boolean>>('/admin/data-permission/dimension', req).then((res) => res.data)
  }

  const updateDimension = async (req: DataPermissionDimensionSaveReq) => {
    return instance
      .put<ResponseData<boolean>>(`/admin/data-permission/dimension/${req.id}`, req)
      .then((res) => res.data)
  }

  const deleteDimension = async (id: number) => {
    return instance
      .delete<ResponseData<boolean>>(`/admin/data-permission/dimension/${id}`)
      .then((res) => res.data)
  }

  const getDimensionOptions = async (code: string) => {
    return instance
      .get<ResponseData<DataPermissionOption[]>>(`/admin/data-permission/dimension-options/${code}`)
      .then((res) => res.data)
  }

  const getMenuBindings = async (menuId: number) => {
    return instance
      .get<ResponseData<DataPermissionBinding[]>>(`/admin/data-permission/bindings/menu/${menuId}`)
      .then((res) => res.data)
  }

  const saveMenuBindings = async (menuId: number, bindings: DataPermissionBindingSaveItem[]) => {
    return instance
      .put<ResponseData<boolean>>(`/admin/data-permission/bindings/menu/${menuId}`, {
        menu_id: menuId,
        bindings,
      })
      .then((res) => res.data)
  }

  const getRoleDataPermissions = async (roleId: number) => {
    return instance
      .get<ResponseData<RoleDataPermission[]>>(`/admin/role/${roleId}/data-permissions`)
      .then((res) => res.data)
  }

  const saveRoleDataPermissions = async (roleId: number, permissions: RoleDataPermissionSaveItem[]) => {
    return instance
      .put<ResponseData<boolean>>(`/admin/role/${roleId}/data-permissions`, {
        role_id: roleId,
        permissions,
      })
      .then((res) => res.data)
  }

  const getUserDataPermissionOverrides = async (userId: number) => {
    return instance
      .get<ResponseData<UserDataPermissionOverride[]>>(`/admin/user/${userId}/data-permissions`)
      .then((res) => res.data)
  }

  const saveUserDataPermissionOverrides = async (
    userId: number,
    overrides: UserDataPermissionOverrideSaveItem[],
  ) => {
    return instance
      .put<ResponseData<boolean>>(`/admin/user/${userId}/data-permissions`, {
        user_id: userId,
        overrides,
      })
      .then((res) => res.data)
  }

  const getUserDimensionValues = async (userId: number) => {
    return instance
      .get<ResponseData<UserDimensionValue[]>>(`/admin/user/${userId}/dimension-values`)
      .then((res) => res.data)
  }

  const saveUserDimensionValues = async (userId: number, items: UserDimensionValueSaveItem[]) => {
    return instance
      .put<ResponseData<boolean>>(`/admin/user/${userId}/dimension-values`, {
        user_id: userId,
        items,
      })
      .then((res) => res.data)
  }

  const debugDataScope = async (params: {
    menu_id?: number
    table_code: string
    action: DataPermissionAction | string
  }) => {
    return instance
      .get<ResponseData<DataPermissionDebugResult>>('/admin/data-permission/debug', { params })
      .then((res) => res.data)
  }

  return {
    queryDimensions,
    getDimensionById,
    createDimension,
    updateDimension,
    deleteDimension,
    getDimensionOptions,
    getMenuBindings,
    saveMenuBindings,
    getRoleDataPermissions,
    saveRoleDataPermissions,
    getUserDataPermissionOverrides,
    saveUserDataPermissionOverrides,
    getUserDimensionValues,
    saveUserDimensionValues,
    debugDataScope,
  }
}
