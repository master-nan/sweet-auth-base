import { defineComponent, h, reactive } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const notify = vi.hoisted(() => vi.fn())
const router = vi.hoisted(() => ({
  push: vi.fn(),
  resolve: vi.fn(() => ({ matched: [{ path: '/admin/system/application' }] })),
}))
const api = vi.hoisted(() => ({ detail: vi.fn() }))
const store = reactive({
  unreadCount: 0,
  recentItems: [] as Array<Record<string, unknown>>,
  loading: false,
  error: '',
  refreshUnreadCount: vi.fn(),
  loadRecent: vi.fn(),
  markRead: vi.fn(),
  markAllRead: vi.fn(),
})

vi.mock('@/stores/notification', () => ({ useNotificationStore: () => store }))
vi.mock('@/api/services/notification', () => ({
  NOTIFICATION_CATEGORY_LABELS: {
    SYSTEM: '系统',
    BUSINESS: '业务',
    TASK: '任务',
    REMINDER: '提醒',
    SECURITY: '安全',
    INTEGRATION: '集成',
  },
  NOTIFICATION_LEVEL_COLORS: {
    INFO: 'primary',
    SUCCESS: 'positive',
    WARNING: 'warning',
    ERROR: 'negative',
  },
  useNotificationApi: () => api,
}))
vi.mock('vue-router', () => ({ useRouter: () => router }))
vi.mock('quasar', () => ({
  useQuasar: () => ({ notify }),
  date: { formatDate: (value: string) => value },
}))

import NotificationPopover from './NotificationPopover.vue'

const SlotStub = (name: string, tag = 'div') =>
  defineComponent({
    name,
    setup(_, { slots }) {
      return () => h(tag, {}, slots.default?.())
    },
  })

const QMenuStub = defineComponent({
  name: 'QMenu',
  emits: ['before-show'],
  setup(_, { slots }) {
    return () => h('div', { 'data-menu': true }, slots.default?.())
  },
})

const QItemStub = defineComponent({
  name: 'QItem',
  emits: ['click'],
  setup(_, { slots, emit }) {
    return () =>
      h(
        'button',
        { 'data-notification-item': true, onClick: () => emit('click') },
        slots.default?.(),
      )
  },
})

const QBtnStub = defineComponent({
  name: 'QBtn',
  inheritAttrs: false,
  setup(_, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.())
  },
})

const QBadgeStub = defineComponent({
  name: 'QBadge',
  setup(_, { slots }) {
    return () => h('span', { 'data-badge': true }, slots.default?.())
  },
})

const stubs = {
  QMenu: QMenuStub,
  QItem: QItemStub,
  QList: SlotStub('QList'),
  QItemSection: SlotStub('QItemSection'),
  QItemLabel: SlotStub('QItemLabel'),
  QBtn: QBtnStub,
  QBadge: QBadgeStub,
  QTooltip: SlotStub('QTooltip'),
  QIcon: SlotStub('QIcon'),
  QSpace: SlotStub('QSpace'),
  QSeparator: SlotStub('QSeparator'),
  QSkeleton: SlotStub('QSkeleton'),
  QDialog: SlotStub('QDialog'),
  QCard: SlotStub('QCard'),
  QCardSection: SlotStub('QCardSection'),
}

const mountPopover = () => mount(NotificationPopover, { global: { stubs } })

const summary = (overrides: Record<string, unknown> = {}) => ({
  id: 1,
  category: 'SYSTEM',
  level: 'INFO',
  title: '系统通知',
  content_preview: '通知内容',
  read: false,
  created_at: '2026-08-25 10:00:00',
  ...overrides,
})

describe('NotificationPopover', () => {
  beforeEach(() => {
    store.unreadCount = 0
    store.recentItems = []
    store.loading = false
    store.error = ''
    store.refreshUnreadCount.mockReset().mockResolvedValue(true)
    store.loadRecent.mockReset().mockResolvedValue(true)
    store.markRead.mockReset().mockResolvedValue(null)
    store.markAllRead.mockReset().mockResolvedValue(true)
    api.detail.mockReset()
    router.push.mockReset()
    router.resolve.mockReset().mockReturnValue({ matched: [{ path: '/admin/system/application' }] })
    notify.mockReset()
  })

  it('refreshes unread and recent data when the bell menu opens', async () => {
    const wrapper = mountPopover()
    expect(wrapper.find('[aria-label="通知"] [data-menu]').exists()).toBe(true)
    wrapper.findComponent(QMenuStub).vm.$emit('before-show')
    await wrapper.vm.$nextTick()
    expect(store.refreshUnreadCount).toHaveBeenCalledOnce()
    expect(store.loadRecent).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('暂无通知')
  })

  it('renders real unread values and caps counts above 99', async () => {
    const wrapper = mountPopover()
    expect(wrapper.find('[data-badge]').exists()).toBe(false)

    store.unreadCount = 5
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-badge]').text()).toBe('5')
    store.unreadCount = 99
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-badge]').text()).toBe('99')
    store.unreadCount = 100
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-badge]').text()).toBe('99+')
  })

  it('renders untrusted content as text and opens an authorized action', async () => {
    store.unreadCount = 1
    store.recentItems = [
      summary({
        content_preview: '<script>alert(1)</script>',
        action: { available: true, path: '/admin/system/application' },
      }),
    ]
    store.markRead.mockResolvedValue({
      ...store.recentItems[0],
      read: true,
      content: '<script>alert(1)</script>',
      source: { module: 'system', type: 'notice' },
    })
    const wrapper = mountPopover()
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.text()).toContain('<script>alert(1)</script>')
    await wrapper.find('[data-notification-item]').trigger('click')
    expect(store.markRead).toHaveBeenCalledWith(1)
    expect(router.push).toHaveBeenCalledWith('/admin/system/application')
  })

  it('keeps message content visible when the action is unavailable', async () => {
    store.recentItems = [summary({ action: { available: false } })]
    store.markRead.mockResolvedValue({
      ...store.recentItems[0],
      read: true,
      content: '只能查看消息，不能打开目标页面。',
      source: { module: 'system', type: 'notice' },
    })
    const wrapper = mountPopover()
    await wrapper.find('[data-notification-item]').trigger('click')
    expect(notify).toHaveBeenCalledWith({ type: 'warning', message: '当前无权访问目标页面' })
    expect(router.push).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('只能查看消息，不能打开目标页面。')
  })
})
