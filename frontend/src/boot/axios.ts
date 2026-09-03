import { translate as t } from '@/boot/i18n'
import { defineBoot } from '#q-app'
import axios, { type AxiosInstance } from 'axios'
import type { InternalAxiosRequestConfig, AxiosRequestHeaders, AxiosResponse } from 'axios'
import { isStaleSessionSnapshot, useUserStore } from '@/stores/user'
import { LocalStorage, Notify } from 'quasar'
import type { ResponseData } from '@/types/global'
import { useLoadingStore } from '@/stores/loading'

export interface HttpResponse<T = unknown> {
  total?: number
  success: boolean
  code?: number
  message?: string
  error_code?: number
  error_message?: string
  data?: Array<T> | T
}

declare module '@vue/runtime-core' {
  interface ComponentCustomProperties {
    $axios: AxiosInstance
    $instance: AxiosInstance
  }
}

const instance = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/sweet_admin',
  timeout: 300000,
  get timeoutErrorMessage() {
    return t('ui.requestTimeoutCheckNetworkConnection')
  },
  withCredentials: true,
})

type SessionBoundRequestConfig = InternalAxiosRequestConfig & {
  sweetSessionToken?: string
  sweetSessionGeneration?: number
  sweetAuthRetried?: boolean
}

export class StaleSessionResponseError extends Error {
  constructor() {
    super(t('ui.theRequestedLoginSessionHasBeenSwitched'))
    this.name = 'StaleSessionResponseError'
  }
}

function persistedAccessToken() {
  return String(LocalStorage.getItem('access_token') || '')
}

export const refreshClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/sweet_admin',
  timeout: 30_000,
  withCredentials: true,
})

let refreshPromise: Promise<string> | null = null

const proactiveRefreshWindowMs = 30_000

export function accessTokenNeedsRefresh(
  value: string,
  now = Date.now(),
  refreshWindowMs = proactiveRefreshWindowMs,
) {
  try {
    const payload = value.split('.')[1]
    if (!payload) return false
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/')
    const normalized = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=')
    const claims = JSON.parse(atob(normalized)) as { exp?: number }
    return typeof claims.exp === 'number' && claims.exp * 1000 <= now + refreshWindowMs
  } catch {
    return false
  }
}

export function refreshAccessToken() {
  if (refreshPromise) return refreshPromise
  refreshPromise = refreshClient
    .post<ResponseData<{ access_token: string }>>('/admin/refresh')
    .then((response) => {
      const accessToken = String(response.data.data?.access_token || '')
      if (!response.data.success || !accessToken) throw new Error(t('ui.loginExpiredPleaseReEntry'))
      useUserStore().replaceAccessToken(accessToken)
      return accessToken
    })
    .finally(() => {
      refreshPromise = null
    })
  return refreshPromise
}

function canRefreshRequest(config: SessionBoundRequestConfig | undefined) {
  if (!config || config.sweetAuthRetried) return false
  const url = config.url || ''
  return !['/admin/login', '/admin/refresh', '/admin/logout'].some((path) => url.includes(path))
}

export function isStaleSessionResponse(
  config: SessionBoundRequestConfig | undefined,
  currentToken: string,
  currentGeneration: number,
  storedToken: string,
) {
  if (!config) return false
  return isStaleSessionSnapshot(
    config.sweetSessionToken,
    config.sweetSessionGeneration,
    currentToken,
    currentGeneration,
    storedToken,
  )
}

Notify.setDefaults({
  position: 'top-right',
})

let lastErrorNotifyKey = ''
let lastErrorNotifyAt = 0

function notifyRequestError(message: string, key: string) {
  const now = Date.now()
  if (lastErrorNotifyKey === key && now - lastErrorNotifyAt < 1500) return

  lastErrorNotifyKey = key
  lastErrorNotifyAt = now
  Notify.create({
    message,
    type: 'negative',
    timeout: 5 * 1000,
    position: 'top-right',
    progress: true,
    group: key,
    actions: [
      {
        icon: 'close',
        color: 'white',
        round: true,
        size: 'sm',
      },
    ],
  })
}

instance.interceptors.request.use(
  async (config: InternalAxiosRequestConfig): Promise<InternalAxiosRequestConfig> => {
    // 针对特定API设置更长的超时时间
    if (config.url?.includes('/large-data') || config.url?.includes('/export')) {
      config.timeout = 60000 // 60秒
    }
    // 在拦截器内部获取 store 实例，避免初始化顺序问题
    const loadingStore = useLoadingStore()
    const userStore = useUserStore()

    const storedToken = persistedAccessToken()
    if (storedToken !== userStore.getLoginToken) {
      userStore.syncPersistedSession(storedToken)
    }

    const skipGlobalLoading = config.headers?.['X-Skip-Global-Loading'] === 'true'
    if (skipGlobalLoading) {
      delete config.headers['X-Skip-Global-Loading']
    } else {
      loadingStore.setLoading(true)
    }
    let token = userStore.getLoginToken
    if (token && accessTokenNeedsRefresh(token) && canRefreshRequest(config)) {
      try {
        token = await refreshAccessToken()
      } catch (error) {
        loadingStore.setLoading(false)
        userStore.setLogout()
        throw error
      }
    }
    const sessionConfig = config as SessionBoundRequestConfig
    sessionConfig.sweetSessionToken = token
    sessionConfig.sweetSessionGeneration = userStore.session_generation
    if (token) {
      if (!config.headers) {
        config.headers = {} as AxiosRequestHeaders
      }
      config.headers.Authorization = 'Bearer ' + token
    }
    return config
  },
  (error) => {
    // loadingStore.setLoading(false)
    console.error(error)
    return Promise.reject(
      error instanceof Error ? error : new Error(error?.message || 'Unknown error'),
    )
  },
)

instance.interceptors.response.use(
  (response: AxiosResponse<ResponseData<any>>): AxiosResponse<ResponseData<any>> => {
    // 在拦截器内部获取 store 实例
    const loadingStore = useLoadingStore()

    loadingStore.setLoading(false)
    if (response.config.responseType === 'blob') {
      return response
    }

    const res = response.data
    if (!res.success) {
      notifyRequestError(
        res.error_message || t('ui.unknownError'),
        `business:${response.config.method || 'get'}:${response.config.url || ''}:${res.error_code || res.code || ''}:${res.error_message || ''}`,
      )
      const error = new Error(res.error_message || t('ui.unknownError')) as Error & {
        response: typeof response
      }
      error.response = response
      throw error
    } else {
      const method = (response.config.method || '').toLowerCase()
      const url = response.config.url || ''
      const isQueryApi = url.includes('/query') || url.includes('/options')
      // 页面会给出具体操作结果；通用“操作成功”既没有额外信息，也容易和页面提示重复。
      if (method !== 'get' && !isQueryApi && res.message && res.message !== '操作成功') {
        Notify.create({
          position: 'top-right',
          progress: true,
          message: res.message as string,
          type: 'positive',
          timeout: 8 * 1000,
          actions: [
            {
              icon: 'close',
              color: 'white',
              round: true,
              size: 'sm',
            },
          ],
        })
      }
    }
    return response
  },
  (error) => {
    // 在拦截器内部获取 store 实例
    const loadingStore = useLoadingStore()
    const userStore = useUserStore()

    loadingStore.setLoading(false)
    const res = error.response
    const requestConfig = error.config as SessionBoundRequestConfig | undefined
    if (res?.status === 401 && canRefreshRequest(requestConfig)) {
      if (requestConfig) requestConfig.sweetAuthRetried = true
      const requestToken = requestConfig?.sweetSessionToken || ''
      const currentToken = userStore.getLoginToken
      const retry =
        requestToken && currentToken && requestToken !== currentToken
          ? Promise.resolve(currentToken)
          : refreshAccessToken()
      return retry
        .then(() => instance.request(requestConfig!))
        .catch((refreshError) => {
          userStore.setLogout()
          notifyRequestError(t('ui.loginExpiredPleaseReEntry'), 'auth:expired')
          return Promise.reject(
            refreshError instanceof Error
              ? refreshError
              : new Error(t('ui.loginExpiredPleaseReEntry')),
          )
        })
    }
    if (
      isStaleSessionResponse(
        requestConfig,
        userStore.getLoginToken,
        userStore.session_generation,
        persistedAccessToken(),
      )
    ) {
      return Promise.reject(new StaleSessionResponseError())
    }
    if (typeof error.response === 'undefined') {
      error.message = t('ui.networkAnomaly')
      notifyRequestError(error.message || 'Request Error', 'network:error')
      return Promise.reject(
        error instanceof Error ? error : new Error(error.message || t('ui.networkAnomaly')),
      )
    }
    if (error.config?.responseType === 'blob') {
      return Promise.reject(
        error instanceof Error ? error : new Error(error.message || t('ui.exportFailed')),
      )
    }

    switch (res.status) {
      case 404:
        error.message = t('ui.resourceNotFound404')
        break
      case 408:
        error.message = t('ui.requestTimeout408')
        break
      case 500:
        error.message = t('ui.serverError500')
        break
      case 501:
        error.message = t('ui.notImplemented501')
        break
      case 502:
        error.message = t('ui.networkError502')
        break
      case 503:
        error.message = t('ui.serviceUnavailable503')
        break
      case 504:
        error.message = t('ui.gatewayTimeout504')
        break
      case 505:
        error.message = t('ui.httpVersionNotSupported505')
        break
      default:
        error.message = res.data.error_message
        break
    }
    notifyRequestError(
      res.data.error_message || error.message || 'Request Error',
      `http:${res.status}:${res.config?.method || 'get'}:${res.config?.url || ''}:${res.data.error_code || ''}:${res.data.error_message || ''}`,
    )
    return Promise.reject(new Error(res.data.error_message || t('ui.unknownError')))
  },
)
export default defineBoot(({ app }) => {
  app.config.globalProperties.$axios = axios
  app.config.globalProperties.$instance = instance
})

export { instance }
