import { instance } from '@/boot/axios'
import type { ResponseData } from '@/types/global'

export interface UserSession {
  id: number
  user_id: number
  user_name: string
  user_deleted: boolean
  status: string
  online: boolean
  current: boolean
  login_at: string
  last_seen_at: string
  expires_at: string
  logout_at?: string | null
  logout_reason: string
  closed_by_user_id: number
  closed_by_user_name: string
  login_channel: string
  ip_address: string
  user_agent: string
  device_type: string
  browser: string
  operating_system: string
}

export interface UserSessionList {
  items: UserSession[]
  total: number
  online_users: number
  online_sessions: number
  online_devices: number
}

export type UserSessionStatusFilter = 'online' | 'active' | 'closed' | 'all'

export interface UserSessionQuery {
  keyword: string
  status: UserSessionStatusFilter
  login_started_at?: string
  login_ended_at?: string
  page: number
  num: number
}

export const useUserSessionApi = () => ({
  query: (params: UserSessionQuery) =>
    instance
      .post<ResponseData<UserSessionList>>('/admin/session/query', params)
      .then((response) => response.data),
  export: (params: UserSessionQuery) =>
    instance.post<Blob>('/admin/session/export', params, { responseType: 'blob' }),
  revoke: (id: number, reason: string) =>
    instance
      .post<ResponseData<boolean>>(`/admin/session/${id}/revoke`, { reason })
      .then((response) => response.data),
  revokeUser: (userId: number, reason: string) =>
    instance
      .post<ResponseData<boolean>>(`/admin/session/user/${userId}/revoke`, { reason })
      .then((response) => response.data),
})
