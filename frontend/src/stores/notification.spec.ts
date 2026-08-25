import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const api = vi.hoisted(() => ({
  unreadCount: vi.fn(),
  recent: vi.fn(),
  markRead: vi.fn(),
  markAllRead: vi.fn(),
}))
const user = vi.hoisted(() => ({ isLogin: true, session_generation: 1 }))

vi.mock('src/api/services/notification', () => ({ useNotificationApi: () => api }))
vi.mock('src/stores/user', () => ({ useUserStore: () => user }))
vi.mock('src/boot/axios', () => ({
  StaleSessionResponseError: class StaleSessionResponseError extends Error {},
}))

import { useNotificationStore } from './notification'

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

describe('notification store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    user.isLogin = true
    user.session_generation = 1
    api.unreadCount.mockReset().mockResolvedValue({ data: { unread_count: 3 } })
    api.recent.mockReset().mockResolvedValue({ data: [] })
    api.markRead.mockReset()
    api.markAllRead.mockReset().mockResolvedValue({ data: { updated_count: 2 } })
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
  })

  it('polls once per minute only while visible and avoids duplicate timers', async () => {
    vi.useFakeTimers()
    const store = useNotificationStore()
    store.startPolling()
    store.startPolling()
    vi.runAllTicks()
    await Promise.resolve()
    expect(vi.getTimerCount()).toBe(1)
    expect(api.unreadCount).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(60_000)
    expect(api.unreadCount).toHaveBeenCalledTimes(2)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    await vi.advanceTimersByTimeAsync(60_000)
    expect(api.unreadCount).toHaveBeenCalledTimes(2)

    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    document.dispatchEvent(new Event('visibilitychange'))
    vi.runAllTicks()
    await Promise.resolve()
    expect(api.unreadCount).toHaveBeenCalledTimes(3)
    store.stopPolling()
    expect(vi.getTimerCount()).toBe(0)
    vi.useRealTimers()
  })

  it('does not let a late session A response overwrite session B', async () => {
    let resolveA!: (value: { data: { unread_count: number } }) => void
    const pendingA = new Promise<{ data: { unread_count: number } }>((resolve) => {
      resolveA = resolve
    })
    api.unreadCount
      .mockReset()
      .mockReturnValueOnce(pendingA)
      .mockResolvedValueOnce({ data: { unread_count: 7 } })
    const store = useNotificationStore()

    const requestA = store.refreshUnreadCount()
    user.session_generation = 2
    const requestB = store.refreshUnreadCount()
    await requestB
    expect(store.unreadCount).toBe(7)
    resolveA({ data: { unread_count: 99 } })
    await requestA
    expect(store.unreadCount).toBe(7)
  })

  it('updates unread state immediately after mark read and mark all', async () => {
    const store = useNotificationStore()
    store.unreadCount = 2
    store.recentItems = [
      {
        id: 1,
        category: 'SYSTEM',
        level: 'INFO',
        title: '通知',
        content_preview: '内容',
        read: false,
        created_at: '2026-08-25 10:00:00',
      },
    ]
    api.markRead.mockResolvedValue({
      data: {
        ...store.recentItems[0],
        read: true,
        read_at: '2026-08-25 10:01:00',
        content: '内容',
        source: { module: 'system', type: 'notice' },
      },
    })
    api.unreadCount.mockResolvedValue({ data: { unread_count: 1 } })

    const detail = await store.markRead(1)
    expect(detail?.read).toBe(true)
    expect(store.unreadCount).toBe(1)
    expect(store.recentItems[0]?.read).toBe(true)

    api.unreadCount.mockResolvedValue({ data: { unread_count: 0 } })
    await store.markAllRead()
    expect(store.unreadCount).toBe(0)
    expect(store.recentItems.every((item) => item.read)).toBe(true)
    await flush()
  })
})
