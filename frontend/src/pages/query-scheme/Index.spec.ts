import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { QuerySchemeType, QuerySchemeValidationStatus } from 'src/modules/query-scheme/types'

const api = vi.hoisted(() => ({
  list: vi.fn(),
  detail: vi.fn(),
  copyToPersonal: vi.fn(),
  setPersonalDefault: vi.fn(),
  setSharedEnabled: vi.fn(),
  deletePersonal: vi.fn(),
  deleteShared: vi.fn(),
}))
const user = vi.hoisted(() => ({
  buttons: [],
  menus: [{
    name: 'system',
    menu_buttons: [{
      event_action: 'query_scheme_shared_manage',
      state: true,
      is_disabled: false,
    }],
  }],
}))

vi.mock('src/api/services/query-scheme', () => ({ useQuerySchemeApi: () => api }))
vi.mock('src/stores/user', () => ({ useUserStore: () => user }))
vi.mock('src/composables/query-scope', () => ({
  collectQueryScopes: () => [{ scope_code: 'system.user.list', scope_label: 'router.system.user', route_name: 'system_user' }],
}))
vi.mock('src/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({
    confirmAction: () => ({ onOk: vi.fn() }),
    confirmDanger: () => ({ onOk: vi.fn() }),
  }),
}))
vi.mock('./QuerySchemeDetailDrawer.vue', () => ({ default: { name: 'QuerySchemeDetailDrawer', template: '<div />' } }))
vi.mock('./QuerySchemeEditDialog.vue', () => ({ default: { name: 'QuerySchemeEditDialog', template: '<div />' } }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (value: string) => value === 'router.system.user' ? '用户管理' : value }),
}))
vi.mock('quasar', async (importOriginal: () => Promise<Record<string, unknown>>) => ({
  ...(await importOriginal()),
  useQuasar: () => ({ notify: vi.fn(), dialog: vi.fn(() => ({ onOk: vi.fn() })) }),
}))

import QuerySchemeManager from './Index.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const ToolbarStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', [slots['quick-search']?.(), slots['right-actions']?.(), slots.default?.()])
  },
})
const QTabStub = defineComponent({
  name: 'QTab',
  props: { name: String, label: String },
  setup(props) {
    return () => h('button', props.label)
  },
})
const QTableStub = defineComponent({
  name: 'QTable',
  props: { rows: Array },
  setup(props, { slots }) {
    return () => h('div', [
      slots.top?.(),
      props.rows?.length
        ? slots['body-cell-actions']?.({ row: props.rows[0] })
        : slots['no-data']?.(),
      slots.bottom?.(),
    ])
  },
})
const QSelectStub = defineComponent({
  props: { options: Array },
  setup(props) {
    return () => h('div', (props.options || []).map((option) => String((option as { label?: string }).label || '')))
  },
})
const QBtnStub = defineComponent({
  props: { label: String, ariaLabel: String },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () => h('button', {
      'aria-label': props.ariaLabel,
      onClick: () => emit('click'),
    }, [props.label, slots.default?.()])
  },
})
const QItemStub = defineComponent({
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, slots.default?.())
  },
})
const managerStubs = {
  BaseContent: SlotStub,
  QTable: QTableStub,
  QTabs: SlotStub,
  QTab: QTabStub,
  QInput: true,
  QSelect: QSelectStub,
  QBtn: QBtnStub,
  QTd: SlotStub,
  QMenu: SlotStub,
  QList: SlotStub,
  QItem: QItemStub,
  QItemSection: SlotStub,
  QTooltip: true,
  QIcon: true,
  QSpace: true,
  QSeparator: true,
  StandardTableToolbar: ToolbarStub,
  TablePagination: true,
  QuerySchemeDetailDrawer: true,
  QuerySchemeEditDialog: true,
}
const mountManager = () => mount(QuerySchemeManager, { global: { stubs: managerStubs } })

describe('QuerySchemeManager', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.list.mockResolvedValue({
      data: [{
        id: 1,
        name: '常用用户',
        scope_code: 'system.user.list',
        scope_label: 'router.system.user',
        type: QuerySchemeType.PERSONAL,
        is_default: true,
        enabled: true,
        revision: 1,
        status: QuerySchemeValidationStatus.VALID,
        creator_display_name: '管理员',
        updated_at: '2026-08-19 10:00:00',
      }],
      total: 1,
    })
  })

  it('loads personal schemes and exposes all four management categories', async () => {
    const wrapper = mountManager()
    await flushPromises()

    expect(api.list).toHaveBeenCalledWith(expect.objectContaining({ scheme_type: QuerySchemeType.PERSONAL }))
    expect(wrapper.text()).toContain('我的方案')
    expect(wrapper.text()).toContain('公共方案')
    expect(wrapper.text()).toContain('角色方案')
    expect(wrapper.text()).toContain('页面默认')
    expect(wrapper.text()).toContain('用户管理')
    expect(wrapper.text()).not.toContain('router.system.user')
    expect(wrapper.html()).not.toContain('revision')
    expect(['使用方案', '编辑方案', '查看方案详情'].every((label) =>
      wrapper.find(`[aria-label="${label}"]`).exists(),
    )).toBe(true)
    expect(wrapper.find('[aria-label="更多方案操作"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('取消默认')
    expect(wrapper.text()).toContain('删除方案')
  })

  it('distinguishes no matches from a loading failure', async () => {
    api.list.mockResolvedValueOnce({ data: [], total: 0 })
    const emptyWrapper = mountManager()
    await flushPromises()
    expect(emptyWrapper.text()).toContain('当前分类暂无查询方案')
    emptyWrapper.unmount()

    api.list.mockRejectedValueOnce(new Error('database details'))
    const failedWrapper = mountManager()
    await flushPromises()
    expect(failedWrapper.text()).toContain('查询方案加载失败，可重试')
    expect(failedWrapper.text()).toContain('重试')
    expect(failedWrapper.text()).not.toContain('database details')
  })
})
