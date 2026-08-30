import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AxiosError, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'

const storage = vi.hoisted(() => new Map<string, unknown>())
const session = vi.hoisted(() => ({
  token: 'old-access-token',
  generation: 1,
  replaceAccessToken: vi.fn<(value: string) => void>(),
  setLogout: vi.fn(),
  syncPersistedSession: vi.fn(),
}))
const loading = vi.hoisted(() => ({ setLoading: vi.fn() }))
const notifications = vi.hoisted(() => ({ setDefaults: vi.fn(), create: vi.fn() }))

vi.mock('#q-app/wrappers', () => ({ defineBoot: (factory: unknown) => factory }))
vi.mock('quasar', () => ({
  LocalStorage: {
    getItem: (key: string) => storage.get(key) ?? null,
    remove: (key: string) => storage.delete(key),
    set: (key: string, value: unknown) => storage.set(key, value),
  },
  Notify: notifications,
}))
vi.mock('src/stores/loading', () => ({ useLoadingStore: () => loading }))
vi.mock('src/stores/user', () => ({
  isStaleSessionSnapshot: () => false,
  useUserStore: () => ({
    get getLoginToken() {
      return session.token
    },
    get session_generation() {
      return session.generation
    },
    replaceAccessToken(value: string) {
      session.token = value
      session.generation += 1
      storage.set('access_token', value)
      session.replaceAccessToken(value)
    },
    setLogout: session.setLogout,
    syncPersistedSession: session.syncPersistedSession,
  }),
}))

import { instance, refreshClient } from './axios'

const response = <T>(
  config: InternalAxiosRequestConfig,
  status: number,
  data: T,
): AxiosResponse<T> => ({
  config,
  data,
  headers: {},
  status,
  statusText: status === 200 ? 'OK' : 'Unauthorized',
})

describe('Axios access token refresh', () => {
  beforeEach(() => {
    storage.clear()
    storage.set('access_token', 'old-access-token')
    session.token = 'old-access-token'
    session.generation = 1
    session.replaceAccessToken.mockReset()
    session.setLogout.mockReset()
    session.syncPersistedSession.mockReset()
    loading.setLoading.mockReset()
    notifications.create.mockReset()
  })

  it('refreshes once and retries all 10 requests after concurrent 401 responses', async () => {
    let refreshCalls = 0
    const attempts = new Map<string, number>()

    refreshClient.defaults.adapter = async (config) => {
      refreshCalls += 1
      await new Promise((resolve) => setTimeout(resolve, 20))
      return response(config, 200, {
        success: true,
        data: { access_token: 'new-access-token' },
      })
    }
    instance.defaults.adapter = (config) => {
      const url = String(config.url || '')
      attempts.set(url, (attempts.get(url) || 0) + 1)
      if (config.headers.Authorization === 'Bearer old-access-token') {
        throw new AxiosError(
          'Unauthorized',
          'ERR_BAD_REQUEST',
          config,
          undefined,
          response(config, 401, { success: false, error_message: 'Token 已过期' }),
        )
      }
      return Promise.resolve(response(config, 200, { success: true, data: url }))
    }

    const results = await Promise.all(
      Array.from({ length: 10 }, (_, index) => instance.get(`/protected/${index}`)),
    )

    expect(refreshCalls).toBe(1)
    expect(session.replaceAccessToken).toHaveBeenCalledOnce()
    expect(session.replaceAccessToken).toHaveBeenCalledWith('new-access-token')
    expect(session.setLogout).not.toHaveBeenCalled()
    expect(results.map((item) => item.data.data)).toEqual(
      Array.from({ length: 10 }, (_, index) => `/protected/${index}`),
    )
    expect([...attempts.values()]).toEqual(Array.from({ length: 10 }, () => 2))
  })
})
