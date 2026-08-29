import { defineBoot } from '#q-app/wrappers'
import axios, { type AxiosInstance } from 'axios'
import type { InternalAxiosRequestConfig, AxiosRequestHeaders, AxiosResponse } from 'axios'
import { isStaleSessionSnapshot, useUserStore } from 'src/stores/user'
import { LocalStorage, Notify } from 'quasar'
import type { ResponseData } from 'src/types/global'
import { useLoadingStore } from 'src/stores/loading'

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
  timeoutErrorMessage: '请求超时，请检查网络连接',
})

type SessionBoundRequestConfig = InternalAxiosRequestConfig & {
  sweetSessionToken?: string
  sweetSessionGeneration?: number
}

export class StaleSessionResponseError extends Error {
  constructor() {
    super('请求所属登录会话已切换')
    this.name = 'StaleSessionResponseError'
  }
}

function persistedAccessToken() {
  return String(LocalStorage.getItem('access_token') || '')
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
  (config: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {
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
    const token = userStore.getLoginToken
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
        res.error_message || '未知错误',
        `business:${response.config.method || 'get'}:${response.config.url || ''}:${res.error_code || res.code || ''}:${res.error_message || ''}`,
      )
      const error = new Error(res.error_message || '未知错误') as Error & {
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
    if (
      isStaleSessionResponse(
        error.config as SessionBoundRequestConfig | undefined,
        userStore.getLoginToken,
        userStore.session_generation,
        persistedAccessToken(),
      )
    ) {
      return Promise.reject(new StaleSessionResponseError())
    }
    if (typeof error.response === 'undefined') {
      error.message = '网络异常'
      notifyRequestError(error.message || 'Request Error', 'network:error')
      return Promise.reject(error instanceof Error ? error : new Error(error.message || '网络异常'))
    }
    if (error.config?.responseType === 'blob') {
      return Promise.reject(error instanceof Error ? error : new Error(error.message || '导出失败'))
    }

    switch (res.status) {
      case 404:
        error.message = '资源不存在(404)'
        break
      case 408:
        error.message = '请求超时(408)'
        break
      case 500:
        error.message = '服务器错误(500)'
        break
      case 501:
        error.message = '服务未实现(501)'
        break
      case 502:
        error.message = '网络错误(502)'
        break
      case 503:
        error.message = '服务不可用(503)'
        break
      case 504:
        error.message = '网络超时(504)'
        break
      case 505:
        error.message = 'HTTP版本不受支持(505)'
        break
      default:
        error.message = res.data.error_message
        break
    }
    notifyRequestError(
      res.data.error_message || error.message || 'Request Error',
      `http:${res.status}:${res.config?.method || 'get'}:${res.config?.url || ''}:${res.data.error_code || ''}:${res.data.error_message || ''}`,
    )
    if (res.status === 401) {
      userStore.setLogout()
    }

    return Promise.reject(new Error(res.data.error_message || '未知错误'))
  },
)
export default defineBoot(({ app }) => {
  app.config.globalProperties.$axios = axios
  app.config.globalProperties.$instance = instance
})

export { instance }
