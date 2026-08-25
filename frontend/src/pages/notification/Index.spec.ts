import { defineComponent, h, reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const notify = vi.hoisted(() => vi.fn())
const router = vi.hoisted(() => ({
  push: vi.fn(),
  resolve: vi.fn(() => ({ matched: [{ path: '/admin/system/application' }] })),
}))
const api = vi.hoisted(() => ({ query: vi.fn() }))
const store = reactive({
  unreadCount: 2,
  refreshUnreadCount: vi.fn(),
  markRead: vi.fn(),
  markAllRead: vi.fn(),
})

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify }),
  date: { formatDate: (value: string) => value },
}))
vi.mock('vue-router', () => ({ useRouter: () => router }))
vi.mock('src/api/services/notification', () => ({
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
vi.mock('src/stores/notification', () => ({ useNotificationStore: () => store }))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
  default: defineComponent({
    name: 'BaseContent',
    setup(_, { slots }) {
      return () => h('main', {}, slots.default?.())
    },
  }),
}))
vi.mock('src/components/Table/StandardTableToolbar.vue', () => ({
  default: defineComponent({
    name: 'StandardTableToolbar',
    emits: ['refresh'],
    setup(_, { slots }) {
      return () => h('header', {}, [slots['quick-search']?.(), slots['right-actions']?.()])
    },
  }),
}))
vi.mock('src/components/Table/TablePagination.vue', () => ({
  default: defineComponent({
    name: 'TablePagination',
    props: ['page', 'pageSize', 'total'],
    emits: ['update:page', 'update:pageSize'],
    template: '<div data-pagination />',
  }),
}))
vi.mock('src/components/Display/StatusChip.vue', () => ({
  default: defineComponent({
    name: 'StatusChip',
    props: ['label'],
    template: '<span>{{ label }}</span>',
  }),
}))

import NotificationCenterPage from './Index.vue'

const SlotStub = (name: string, tag = 'div') =>
  defineComponent({
    name,
    setup(_, { slots }) {
      return () => h(tag, {}, slots.default?.())
    },
  })

const QItemStub = defineComponent({
  name: 'QItem',
  emits: ['click'],
  setup(_, { slots, emit }) {
    return () =>
      h(
        'button',
        { 'data-notification-row': true, onClick: () => emit('click') },
        slots.default?.(),
      )
  },
})

const stubs = {
  QBtn: defineComponent({
    name: 'QBtn',
    props: ['label', 'icon'],
    emits: ['click'],
    setup(props, { emit }) {
      return () =>
        h(
          'button',
          { 'data-label': props.label, 'data-icon': props.icon, onClick: () => emit('click') },
          props.label || props.icon,
        )
    },
  }),
  QBtnToggle: defineComponent({
    name: 'QBtnToggle',
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: '<div data-read-status />',
  }),
  QInput: defineComponent({
    name: 'QInput',
    props: ['modelValue'],
    emits: ['update:modelValue', 'keyup'],
    setup(_, { slots }) {
      return () => h('div', {}, slots.append?.())
    },
  }),
  QSelect: defineComponent({
    name: 'QSelect',
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: '<div data-category />',
  }),
  QItem: QItemStub,
  QList: SlotStub('QList'),
  QItemSection: SlotStub('QItemSection'),
  QItemLabel: SlotStub('QItemLabel'),
  QAvatar: SlotStub('QAvatar'),
  QIcon: SlotStub('QIcon'),
  QTooltip: SlotStub('QTooltip'),
  QSeparator: SlotStub('QSeparator'),
  QSkeleton: SlotStub('QSkeleton'),
  QDialog: SlotStub('QDialog'),
  QCard: SlotStub('QCard'),
  QCardSection: SlotStub('QCardSection'),
  QCardActions: SlotStub('QCardActions'),
}

const item = (overrides: Record<string, unknown> = {}) => ({
  id: 1,
  category: 'REMINDER',
  level: 'WARNING',
  title: '一条很长但不会破坏列表布局的学习任务截止提醒',
  content_preview: '请在今天下班前完成学习任务。',
  read: false,
  created_at: '2026-08-25 10:00:00',
  ...overrides,
})

const mountPage = () => mount(NotificationCenterPage, { global: { stubs } })

describe('NotificationCenterPage', () => {
  beforeEach(() => {
    api.query.mockReset().mockResolvedValue({ data: [item()], total: 31 })
    store.unreadCount = 2
    store.refreshUnreadCount.mockReset().mockResolvedValue(true)
    store.markRead.mockReset().mockResolvedValue({
      ...item({ read: true }),
      content: '<script>alert(1)</script>\n详细通知内容',
      source: { module: 'learning', type: 'learning_plan', id: '17' },
    })
    store.markAllRead.mockReset().mockResolvedValue(true)
    router.push.mockReset()
    router.resolve.mockReset().mockReturnValue({ matched: [{ path: '/admin/system/application' }] })
    notify.mockReset()
  })

  it('queries by keyword, read status and category with fixed pagination semantics', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(api.query).toHaveBeenLastCalledWith({
      page: 1,
      num: 15,
      keyword: '',
      read_status: 'ALL',
    })
    expect(wrapper.text()).toContain('一条很长但不会破坏列表布局')

    wrapper.findComponent({ name: 'QInput' }).vm.$emit('update:modelValue', '  学习任务  ')
    wrapper.findComponent({ name: 'QBtnToggle' }).vm.$emit('update:modelValue', 'UNREAD')
    wrapper.findComponent({ name: 'QSelect' }).vm.$emit('update:modelValue', 'REMINDER')
    await wrapper.find('[data-label="查询"]').trigger('click')
    await flushPromises()
    expect(api.query).toHaveBeenLastCalledWith({
      page: 1,
      num: 15,
      keyword: '学习任务',
      read_status: 'UNREAD',
      category: 'REMINDER',
    })

    wrapper.findComponent({ name: 'TablePagination' }).vm.$emit('update:page', 2)
    await flushPromises()
    expect(api.query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, num: 15 }))
  })

  it('marks one notification and all notifications through the owner store', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await wrapper.find('[data-notification-row]').trigger('click')
    await flushPromises()
    expect(store.markRead).toHaveBeenCalledWith(1)
    expect(wrapper.text()).toContain('详细通知内容')
    expect(wrapper.text()).toContain('<script>alert(1)</script>')
    expect(wrapper.find('script').exists()).toBe(false)

    await wrapper.find('[data-label="全部已读"]').trigger('click')
    await flushPromises()
    expect(store.markAllRead).toHaveBeenCalledOnce()
    expect(api.query.mock.calls.length).toBeGreaterThan(1)
  })

  it('renders empty and recoverable error states', async () => {
    api.query.mockResolvedValueOnce({ data: [], total: 0 })
    const empty = mountPage()
    await flushPromises()
    expect(empty.text()).toContain('暂无符合条件的通知')

    api.query.mockRejectedValueOnce(new Error('通知服务暂时不可用'))
    const failed = mountPage()
    await flushPromises()
    expect(failed.text()).toContain('通知服务暂时不可用')
    api.query.mockResolvedValueOnce({ data: [], total: 0 })
    await failed.find('[data-label="重试"]').trigger('click')
    await flushPromises()
    expect(failed.text()).toContain('暂无符合条件的通知')
  })
})
