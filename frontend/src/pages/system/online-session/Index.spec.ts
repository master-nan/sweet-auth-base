import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type * as Quasar from 'quasar'

const api = vi.hoisted(() => ({
  query: vi.fn(),
  export: vi.fn(),
  revoke: vi.fn(),
  revokeUser: vi.fn(),
}))

vi.mock('quasar', async (importOriginal) => ({
  ...(await importOriginal<typeof Quasar>()),
  useQuasar: () => ({ notify: vi.fn() }),
}))
vi.mock('@/api/services/user-session', () => ({ useUserSessionApi: () => api }))
vi.mock('@/composables/page-buttons', () => ({
  usePageButtons: () => ({ line_buttons: [], top_buttons: [] }),
}))
vi.mock('@/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({ confirmWithReason: vi.fn() }),
}))
vi.mock('@/utils/menu-button-display', () => ({ menuButtonDisplayProps: () => ({}) }))
vi.mock('@/utils/download', () => ({
  downloadBlob: vi.fn(),
  parseContentDispositionFilename: vi.fn(),
}))

import OnlineSessionPage from './Index.vue'

const SlotStub = (name: string, tag = 'div') =>
  defineComponent({
    name,
    setup(_, { slots }) {
      return () => h(tag, {}, slots.default?.())
    },
  })

const QTableStub = defineComponent({
  name: 'QTable',
  props: {
    rows: { type: Array, default: () => [] },
    pagination: { type: Object, default: () => ({}) },
    hidePagination: Boolean,
  },
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-rendered-rows': props.rows.length }, [
        slots.top?.(),
        ...props.rows.map((row) => slots['body-cell-status']?.({ row })),
        slots.bottom?.(),
      ])
  },
})

const session = (id: number) => ({
  id,
  user_id: 1,
  user_name: 'admin',
  user_deleted: false,
  status: 'active',
  online: false,
  current: false,
  login_at: '2026-08-31 03:00:00',
  last_seen_at: '2026-08-31 03:00:00',
  expires_at: '2026-09-30 03:00:00',
  logout_reason: '',
  closed_by_user_id: 0,
  closed_by_user_name: '',
  login_channel: 'admin_password',
  ip_address: '127.0.0.1',
  user_agent: 'curl/8.7.1',
  device_type: '桌面设备',
  browser: '未知浏览器',
  operating_system: '未知系统',
})

const mountPage = () =>
  mount(OnlineSessionPage, {
    global: {
      stubs: {
        BaseContent: SlotStub('BaseContent', 'main'),
        StandardTableToolbar: SlotStub('StandardTableToolbar', 'header'),
        TableColumnSelector: true,
        TablePagination: true,
        SweetDateTimePicker: true,
        FormDialogShell: SlotStub('FormDialogShell'),
        DetailFieldGrid: true,
        StatusChip: defineComponent({
          name: 'StatusChip',
          props: ['label'],
          template: '<span class="status-chip">{{ label }}</span>',
        }),
        QTable: QTableStub,
        QTd: SlotStub('QTd'),
        QSpace: SlotStub('QSpace'),
        QInput: true,
        QSelect: true,
        QBtn: true,
        QIcon: true,
        QMenu: true,
        QSeparator: true,
        QTooltip: true,
      },
    },
  })

describe('OnlineSessionPage', () => {
  beforeEach(() => {
    api.query.mockReset().mockResolvedValue({
      data: {
        items: Array.from({ length: 20 }, (_, index) => session(index + 1)),
        total: 21,
        online_users: 1,
        online_sessions: 1,
      },
    })
  })

  it('renders the complete server page without Quasar paginating it again', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const table = wrapper.findComponent(QTableStub)
    expect(api.query).toHaveBeenCalledWith({
      keyword: '',
      status: 'online',
      page: 1,
      num: 20,
    })
    expect(table.props('rows')).toHaveLength(20)
    expect(table.props('pagination')).toEqual({ rowsPerPage: 0 })
    expect(table.props('hidePagination')).toBe(true)
    expect(wrapper.findAll('.status-chip')).toHaveLength(20)
    expect(wrapper.text()).toContain('有效，当前未在线')
  })
})
