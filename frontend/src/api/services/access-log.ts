import type { Basic, Query, ResponseData } from 'src/types/global'
import { instance } from 'boot/axios'

export interface AccessLog extends Basic {
  user_name: string
  request_id: string
  trace_id: string
  method: string
  ip: string
  locality: string
  url: string
  action: string
  resource_type: string
  resource_code: string
  resource_id: string
  status_code: number
  success: boolean
  duration_ms: number
  result: string
  error_code: string
  error_message: string
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
