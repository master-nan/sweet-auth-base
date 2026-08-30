import { instance } from 'boot/axios'
import type { ResponseData } from 'src/types/global'

export interface UserSession {
  id: number
  user_id: number
  user_name: string
  status: string
  online: boolean
  current: boolean
  login_at: string
  last_seen_at: string
  expires_at: string
  logout_at?: string | null
  logout_reason: string
  login_channel: string
  ip_address: string
  device_type: string
  browser: string
  operating_system: string
}

export interface UserSessionList {
  items: UserSession[]
  total: number
  online_users: number
  online_devices: number
}

export type UserSessionStatusFilter = 'online' | 'active' | 'closed' | 'all'

export const useUserSessionApi = () => ({
  query: (params: {
    keyword: string
    status: UserSessionStatusFilter
    page: number
    num: number
  }) =>
    instance
      .post<ResponseData<UserSessionList>>('/admin/session/query', params)
      .then((response) => response.data),
  revoke: (id: number) =>
    instance
      .post<ResponseData<boolean>>(`/admin/session/${id}/revoke`)
      .then((response) => response.data),
  revokeUser: (userId: number) =>
    instance
      .post<ResponseData<boolean>>(`/admin/session/user/${userId}/revoke`)
      .then((response) => response.data),
})
