import { translate as t } from 'src/boot/i18n'
import { defineStore } from 'pinia'
import {
  useNotificationApi,
  type NotificationDetail,
  type NotificationSummary,
} from 'src/api/services/notification'
import { StaleSessionResponseError } from 'src/boot/axios'
import { useUserStore } from 'src/stores/user'

const pollingInterval = 60_000

type NotificationState = {
  unreadCount: number
  recentItems: NotificationSummary[]
  loading: boolean
  error: string
  pollingTimer: ReturnType<typeof setInterval> | null
  visibilityHandler: (() => void) | null
}

export const useNotificationStore = defineStore('notification', {
  state: (): NotificationState => ({
    unreadCount: 0,
    recentItems: [],
    loading: false,
    error: '',
    pollingTimer: null,
    visibilityHandler: null,
  }),

  actions: {
    isCurrentSession(generation: number) {
      return generation === useUserStore().session_generation
    },
    isStale(error: unknown, generation: number) {
      return error instanceof StaleSessionResponseError || !this.isCurrentSession(generation)
    },
    async refreshUnreadCount() {
      const userStore = useUserStore()
      if (!userStore.isLogin) return false
      const generation = userStore.session_generation
      try {
        const response = await useNotificationApi().unreadCount()
        if (!this.isCurrentSession(generation)) return false
        this.unreadCount = Math.max(0, Number(response.data?.unread_count || 0))
        return true
      } catch (error) {
        if (this.isStale(error, generation)) return false
        this.error = error instanceof Error ? error.message : t('ui.failedToLoadUnreadNotification')
        return false
      }
    },
    async loadRecent() {
      const userStore = useUserStore()
      if (!userStore.isLogin) return false
      const generation = userStore.session_generation
      this.loading = true
      this.error = ''
      try {
        const response = await useNotificationApi().recent(8)
        if (!this.isCurrentSession(generation)) return false
        this.recentItems = response.data || []
        return true
      } catch (error) {
        if (this.isStale(error, generation)) return false
        this.error = error instanceof Error ? error.message : t('ui.recentNotificationLoadFailed')
        return false
      } finally {
        if (this.isCurrentSession(generation)) this.loading = false
      }
    },
    async markRead(id: number): Promise<NotificationDetail | null> {
      const userStore = useUserStore()
      const generation = userStore.session_generation
      const previous = this.recentItems.find((item) => item.id === id)
      try {
        const response = await useNotificationApi().markRead(id)
        if (!this.isCurrentSession(generation) || !response.data) return null
        const index = this.recentItems.findIndex((item) => item.id === id)
        if (index >= 0) this.recentItems[index] = response.data
        if (previous && !previous.read) this.unreadCount = Math.max(0, this.unreadCount - 1)
        void this.refreshUnreadCount()
        return response.data
      } catch (error) {
        if (!this.isStale(error, generation)) {
          this.error = error instanceof Error ? error.message : t('ui.notificationTagReadFailed')
        }
        return null
      }
    },
    async markAllRead() {
      const userStore = useUserStore()
      const generation = userStore.session_generation
      try {
        await useNotificationApi().markAllRead()
        if (!this.isCurrentSession(generation)) return false
        const readAt = new Date().toISOString()
        this.recentItems = this.recentItems.map((item) => ({
          ...item,
          read: true,
          read_at: item.read_at || readAt,
        }))
        this.unreadCount = 0
        void this.refreshUnreadCount()
        return true
      } catch (error) {
        if (!this.isStale(error, generation)) {
          this.error = error instanceof Error ? error.message : t('ui.allReadOperationsFailed')
        }
        return false
      }
    },
    startPolling() {
      const userStore = useUserStore()
      if (!userStore.isLogin || this.pollingTimer) return
      void this.refreshUnreadCount()
      this.visibilityHandler = () => {
        if (document.visibilityState === 'visible') void this.refreshUnreadCount()
      }
      document.addEventListener('visibilitychange', this.visibilityHandler)
      this.pollingTimer = setInterval(() => {
        if (document.visibilityState === 'visible') void this.refreshUnreadCount()
      }, pollingInterval)
    },
    stopPolling() {
      if (this.pollingTimer) clearInterval(this.pollingTimer)
      this.pollingTimer = null
      if (this.visibilityHandler) {
        document.removeEventListener('visibilitychange', this.visibilityHandler)
      }
      this.visibilityHandler = null
    },
    reset() {
      this.stopPolling()
      this.unreadCount = 0
      this.recentItems = []
      this.loading = false
      this.error = ''
    },
  },
})
