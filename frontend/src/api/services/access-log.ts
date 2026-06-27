import type { Basic, Query, ResponseData } from 'src/types/global'
import { instance } from 'boot/axios'

export interface AccessLog extends Basic {
  user_id: number
  user_name: string
  method: string
  ip: string
  locality: string
  url: string
  action: string
  resource_type: string
  resource_code: string
  resource_id: string
  menu_id: number
  status_code: number
  success: boolean
  duration_ms: number
  body?: string
  query?: string
  response?: string
}

export const useAccessLogApi = () => {
  const queryAccessLogs = async (params: Query) => {
    return instance.post<ResponseData<AccessLog[]>>('/admin/log/access/query', params).then((res) => res.data)
  }

  const getAccessLogById = async (id: number) => {
    return instance.get<ResponseData<AccessLog>>(`/admin/log/access/${id}`).then((res) => res.data)
  }

  return {
    queryAccessLogs,
    getAccessLogById,
  }
}
