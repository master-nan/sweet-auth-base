import { instance } from 'boot/axios'
import { localLoadingRequestConfig } from 'src/api/request-config'
import type { ResponseData } from 'src/types/global'

export type NotificationCategory =
  | 'SYSTEM'
  | 'BUSINESS'
  | 'TASK'
  | 'REMINDER'
  | 'SECURITY'
  | 'INTEGRATION'

export type NotificationLevel = 'INFO' | 'SUCCESS' | 'WARNING' | 'ERROR'
export type NotificationReadStatus = 'ALL' | 'UNREAD' | 'READ'

export interface NotificationAction {
  available: boolean
  path?: string
}

export interface NotificationSummary {
  id: number
  category: NotificationCategory
  level: NotificationLevel
  title: string
  content_preview: string
  read: boolean
  read_at?: string
  created_at: string
  action?: NotificationAction
}

export interface NotificationDetail extends NotificationSummary {
  content: string
  source: {
    module: string
    type: string
    id?: string
  }
}

export interface NotificationQuery {
  page: number
  num: number
  keyword: string
  read_status: NotificationReadStatus
  category?: NotificationCategory
}

export const NOTIFICATION_CATEGORY_LABELS: Record<NotificationCategory, string> = {
  SYSTEM: '系统',
  BUSINESS: '业务',
  TASK: '任务',
  REMINDER: '提醒',
  SECURITY: '安全',
  INTEGRATION: '集成',
}

export const NOTIFICATION_LEVEL_LABELS: Record<NotificationLevel, string> = {
  INFO: '信息',
  SUCCESS: '成功',
  WARNING: '警告',
  ERROR: '错误',
}

export const NOTIFICATION_LEVEL_COLORS: Record<NotificationLevel, string> = {
  INFO: 'primary',
  SUCCESS: 'positive',
  WARNING: 'warning',
  ERROR: 'negative',
}

export const useNotificationApi = () => ({
  unreadCount: () =>
    instance
      .get<
        ResponseData<{ unread_count: number }>
      >('/admin/runtime/notifications/unread-count', localLoadingRequestConfig)
      .then((response) => response.data),
  recent: (limit = 8) =>
    instance
      .get<ResponseData<NotificationSummary[]>>('/admin/runtime/notifications/recent', {
        ...localLoadingRequestConfig,
        params: { limit },
      })
      .then((response) => response.data),
  query: (query: NotificationQuery) =>
    instance
      .post<
        ResponseData<NotificationSummary[]>
      >('/admin/runtime/notifications/query', query, localLoadingRequestConfig)
      .then((response) => response.data),
  detail: (id: number) =>
    instance
      .get<
        ResponseData<NotificationDetail>
      >(`/admin/runtime/notifications/${id}`, localLoadingRequestConfig)
      .then((response) => response.data),
  markRead: (id: number) =>
    instance
      .post<
        ResponseData<NotificationDetail>
      >(`/admin/runtime/notifications/${id}/read`, undefined, localLoadingRequestConfig)
      .then((response) => response.data),
  markAllRead: () =>
    instance
      .post<
        ResponseData<{ updated_count: number }>
      >('/admin/runtime/notifications/read-all', undefined, localLoadingRequestConfig)
      .then((response) => response.data),
})
