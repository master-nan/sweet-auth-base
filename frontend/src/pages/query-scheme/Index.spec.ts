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
  menus: [
    {
      name: 'system',
      menu_buttons: [
        {
          event_action: 'query_scheme_shared_manage',
          state: true,
          is_disabled: false,
        },
      ],
    },
  ],
}))
const quasar = vi.hoisted(() => ({
  notify: vi.fn(),
  dialog: vi.fn(),
  dialogOnOk: vi.fn(),
}))
const router = vi.hoisted(() => ({ push: vi.fn() }))

vi.mock('src/api/services/query-scheme', () => ({ useQuerySchemeApi: () => api }))
vi.mock('src/stores/user', () => ({ useUserStore: () => user }))
vi.mock('src/composables/query-scope', () => ({
  collectQueryScopes: () => [
    {
      scope_code: 'system.user.list',
      scope_label: 'router.system.user',
      route_name: 'system_user',
    },
  ],
}))
vi.mock('src/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({
    confirmAction: () => ({ onOk: vi.fn() }),
    confirmDanger: () => ({ onOk: vi.fn() }),
  }),
}))
vi.mock('./QuerySchemeDetailDrawer.vue', () => ({
  default: { name: 'QuerySchemeDetailDrawer', template: '<div />' },
}))
vi.mock('./QuerySchemeEditDialog.vue', () => ({
  default: { name: 'QuerySchemeEditDialog', template: '<div />' },
}))
vi.mock('vue-router', () => ({ useRouter: () => router }))
vi.mock('quasar', async (importOriginal: () => Promise<Record<string, unknown>>) => ({
  ...(await importOriginal()),
  useQuasar: () => ({ notify: quasar.notify, dialog: quasar.dialog }),
}))

import QuerySchemeManager from './Index.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const ToolbarStub = defineComponent({
  setup(_, { slots }) {
    return () =>
      h('div', [slots['quick-search']?.(), slots['right-actions']?.(), slots.default?.()])
  },
})
const TooltipStub = defineComponent({
  name: 'QTooltip',
  setup(_, { slots }) {
    return () => h('span', { class: 'q-tooltip-stub' }, slots.default?.())
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
    return () =>
      h('div', [
        slots.top?.(),
        props.rows?.length
          ? [
              slots['body-cell-name']?.({ row: props.rows[0] }),
              slots['body-cell-actions']?.({ row: props.rows[0] }),
            ]
          : slots['no-data']?.(),
        slots.bottom?.(),
      ])
  },
})
const QSelectStub = defineComponent({
  props: { options: Array },
  setup(props) {
    return () =>
      h(
        'div',
        (props.options || []).map((option) => String((option as { label?: string }).label || '')),
      )
  },
})
const QBtnStub = defineComponent({
  props: { label: String, ariaLabel: String, disable: Boolean },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          'aria-label': props.ariaLabel,
          disabled: props.disable,
          onClick: () => emit('click'),
        },
        [props.label, slots.default?.()],
      )
  },
})
const QItemStub = defineComponent({
  name: 'QItem',
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () =>
      h('button', { class: 'q-item-stub', onClick: () => emit('click') }, slots.default?.())
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
  QTooltip: TooltipStub,
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
    user.menus[0]!.menu_buttons[0]!.state = true
    quasar.dialog.mockReturnValue({ onOk: quasar.dialogOnOk })
    api.copyToPersonal.mockResolvedValue({ data: { id: 9 } })
    api.list.mockResolvedValue({
      data: [
        {
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
        },
      ],
      total: 1,
    })
  })

  it('loads personal schemes and exposes all four management categories', async () => {
    const wrapper = mountManager()
    await flushPromises()

    expect(api.list).toHaveBeenCalledWith(
      expect.objectContaining({ scheme_type: QuerySchemeType.PERSONAL }),
    )
    expect(wrapper.text()).toContain('我的方案')
    expect(wrapper.text()).toContain('公共方案')
    expect(wrapper.text()).toContain('角色方案')
    expect(wrapper.text()).toContain('页面默认')
    expect(wrapper.text()).toContain('用户管理')
    expect(wrapper.text()).not.toContain('router.system.user')
    expect(wrapper.html()).not.toContain('revision')
    expect(
      ['使用方案', '编辑方案', '查看方案详情'].every((label) =>
        wrapper.find(`[aria-label="${label}"]`).exists(),
      ),
    ).toBe(true)
    expect(wrapper.find('[aria-label="更多方案操作"]').exists()).toBe(true)
    expect(wrapper.findAll('.query-scheme-row-action')).toHaveLength(3)
    expect(
      wrapper
        .findAll('.query-scheme-row-action button')
        .every((button) => button.text().trim() === ''),
    ).toBe(true)
    expect(wrapper.text()).toContain('取消默认')
    expect(wrapper.text()).toContain('删除方案')
    expect(wrapper.find('[aria-label="使用方案"]').attributes('disabled')).toBeUndefined()
  })

  it('navigates with transient state instead of creating a query-string tag', async () => {
    const wrapper = mountManager()
    await flushPromises()

    await wrapper.find('[aria-label="使用方案"]').trigger('click')

    expect(router.push).toHaveBeenCalledWith({
      name: 'system_user',
      state: { query_scheme_id: '1' },
    })
  })

  it('only shows more when a fourth available action exists', async () => {
    user.menus[0]!.menu_buttons[0]!.state = false
    api.list.mockResolvedValueOnce({
      data: [
        {
          id: 4,
          name: '公共只读方案',
          scope_code: 'system.user.list',
          scope_label: 'router.system.user',
          type: QuerySchemeType.PUBLIC,
          is_default: false,
          enabled: true,
          revision: 1,
          status: QuerySchemeValidationStatus.VALID,
          creator_display_name: '管理员',
          updated_at: '2026-08-19 10:00:00',
        },
      ],
      total: 1,
    })
    const wrapper = mountManager()
    await flushPromises()

    expect(wrapper.findAll('.query-scheme-row-action')).toHaveLength(3)
    expect(wrapper.find('[aria-label="更多方案操作"]').exists()).toBe(false)
  })

  it('disables use for stopped or invalid schemes and explains why', async () => {
    api.list.mockResolvedValueOnce({
      data: [
        {
          id: 2,
          name: '需要修复的方案',
          scope_code: 'system.user.list',
          scope_label: 'router.system.user',
          type: QuerySchemeType.PERSONAL,
          is_default: false,
          enabled: true,
          revision: 4,
          status: QuerySchemeValidationStatus.DEGRADED,
          creator_display_name: '管理员',
          updated_at: '2026-08-19 10:00:00',
        },
      ],
      total: 1,
    })
    const wrapper = mountManager()
    await flushPromises()

    const useButton = wrapper.find('[aria-label="使用方案"]')
    expect(useButton.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('方案需要修复后才能使用')
    await useButton.trigger('click')
    expect(router.push).not.toHaveBeenCalled()
  })

  it('shows long names with a tooltip and creates a Unicode-safe copy name', async () => {
    const longName = '🚀'.repeat(64)
    api.list.mockResolvedValueOnce({
      data: [
        {
          id: 3,
          name: longName,
          scope_code: 'system.user.list',
          scope_label: 'router.system.user',
          type: QuerySchemeType.PUBLIC,
          is_default: false,
          enabled: true,
          revision: 2,
          status: QuerySchemeValidationStatus.VALID,
          creator_display_name: '管理员',
          updated_at: '2026-08-19 10:00:00',
        },
      ],
      total: 1,
    })
    const wrapper = mountManager()
    await flushPromises()

    const nameCell = wrapper.find('.query-scheme-name')
    expect(nameCell.classes()).toContain('query-scheme-name')
    expect(nameCell.find('.q-tooltip-stub').text()).toBe(longName)
    const copyItem = wrapper
      .findAll('.q-item-stub')
      .find((item) => item.text().includes('复制为我的方案'))
    expect(copyItem).toBeDefined()
    await copyItem!.trigger('click')

    const dialogOptions = quasar.dialog.mock.calls[0]?.[0]
    const copyName = dialogOptions.prompt.model as string
    expect(Array.from(copyName)).toHaveLength(64)
    expect(copyName.endsWith(' 副本')).toBe(true)
    expect(dialogOptions.prompt.isValid('😀'.repeat(64))).toBe(true)
    expect(dialogOptions.prompt.isValid('😀'.repeat(65))).toBe(false)

    const copy = quasar.dialogOnOk.mock.calls[0]?.[0]
    copy('😀'.repeat(65))
    expect(api.copyToPersonal).not.toHaveBeenCalled()
    copy('  我的副本  ')
    await flushPromises()
    expect(api.copyToPersonal).toHaveBeenCalledWith(3, 'system.user.list', '我的副本')
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
