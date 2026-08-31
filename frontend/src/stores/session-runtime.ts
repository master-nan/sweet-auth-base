import { translate as t } from 'src/i18n/runtime/instance'
import { defineStore } from 'pinia'
import { Notify } from 'quasar'
import { instance, refreshAccessToken } from 'src/boot/axios'
import { useUserStore } from 'src/stores/user'

const heartbeatInterval = 60_000
const reconnectDelay = 3_000

type SessionEvent = {
  event: string
  data: Record<string, unknown>
}

type SessionRuntimeState = {
  heartbeatTimer: ReturnType<typeof setInterval> | null
  streamController: AbortController | null
  visibilityHandler: (() => void) | null
  running: boolean
}

export const useSessionRuntimeStore = defineStore('session-runtime', {
  state: (): SessionRuntimeState => ({
    heartbeatTimer: null,
    streamController: null,
    visibilityHandler: null,
    running: false,
  }),

  actions: {
    start() {
      const userStore = useUserStore()
      if (!userStore.isLogin || this.running) return
      this.running = true
      void this.sendHeartbeat()
      this.heartbeatTimer = setInterval(() => {
        if (document.visibilityState === 'visible') void this.sendHeartbeat()
      }, heartbeatInterval)
      this.visibilityHandler = () => {
        if (document.visibilityState === 'visible') void this.sendHeartbeat()
      }
      document.addEventListener('visibilitychange', this.visibilityHandler)
      void this.runEventStream(userStore.session_generation)
    },

    async sendHeartbeat() {
      if (!useUserStore().isLogin) return
      try {
        await instance.post(
          '/admin/runtime/session/heartbeat',
          {},
          { headers: { 'X-Skip-Global-Loading': 'true' } },
        )
      } catch {
        // 401 会由 Axios 刷新或退出；短暂网络中断交给下一次心跳重试。
      }
    },

    async runEventStream(generation: number) {
      while (this.running && useUserStore().isLogin) {
        const userStore = useUserStore()
        if (generation !== userStore.session_generation) return
        const controller = new AbortController()
        this.streamController = controller
        try {
          const baseURL = String(import.meta.env.VITE_API_URL || '/sweet_admin').replace(/\/$/, '')
          const response = await fetch(`${baseURL}/admin/runtime/session/events`, {
            method: 'GET',
            credentials: 'include',
            headers: { Authorization: `Bearer ${userStore.getLoginToken}` },
            signal: controller.signal,
          })
          if (response.status === 401) {
            try {
              await refreshAccessToken()
            } catch {
              useUserStore().setLogout()
            }
            return
          }
          if (!response.ok || !response.body) throw new Error(t('ui.sessionEventConnectionFailed'))
          const keepRunning = await this.readEvents(response.body, generation)
          if (!keepRunning) return
        } catch {
          if (controller.signal.aborted || !this.running) return
          if (generation !== useUserStore().session_generation) return
        } finally {
          if (this.streamController === controller) this.streamController = null
        }
        await new Promise((resolve) => setTimeout(resolve, reconnectDelay))
      }
    },

    async readEvents(stream: ReadableStream<Uint8Array>, generation: number) {
      const reader = stream.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      while (this.running && generation === useUserStore().session_generation) {
        const { value, done } = await reader.read()
        if (done) return true
        buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, '\n')
        const blocks = buffer.split('\n\n')
        buffer = blocks.pop() || ''
        for (const block of blocks) {
          const event = parseSessionEvent(block)
          if (event && !(await this.handleEvent(event))) return false
        }
      }
      return false
    },

    async handleEvent(message: SessionEvent) {
      if (message.event === 'session_revoked') {
        const text =
          typeof message.data.message === 'string'
            ? message.data.message
            : t('ui.currentLoginIsOfflineForAdministrator')
        Notify.create({ type: 'warning', position: 'top-right', message: text, timeout: 5000 })
        useUserStore().setLogout()
        return false
      }
      if (message.event === 'access_expired') {
        try {
          await refreshAccessToken()
        } catch {
          useUserStore().setLogout()
        }
        return false
      }
      return true
    },

    reset() {
      this.running = false
      if (this.heartbeatTimer) clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
      this.streamController?.abort()
      this.streamController = null
      if (this.visibilityHandler)
        document.removeEventListener('visibilitychange', this.visibilityHandler)
      this.visibilityHandler = null
    },
  },
})

function parseSessionEvent(block: string): SessionEvent | null {
  if (!block || block.startsWith(':')) return null
  let event = 'message'
  const dataLines: string[] = []
  block.split('\n').forEach((line) => {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
  })
  if (dataLines.length === 0) return null
  try {
    return { event, data: JSON.parse(dataLines.join('\n')) as Record<string, unknown> }
  } catch {
    return { event, data: { message: dataLines.join('\n') } }
  }
}
