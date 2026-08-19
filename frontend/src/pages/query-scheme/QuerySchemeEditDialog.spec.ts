import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { QuerySchemeType, QuerySchemeValidationStatus } from 'src/modules/query-scheme/types'

const api = vi.hoisted(() => ({
  getScopeConfig: vi.fn(),
  updateShared: vi.fn(),
  setSharedEnabled: vi.fn(),
}))
const tableApi = vi.hoisted(() => ({ queryRuntimeTableByCode: vi.fn() }))
vi.mock('src/api/services/query-scheme', () => ({ useQuerySchemeApi: () => api }))
vi.mock('src/api/services/sys-table', () => ({ useTableApi: () => tableApi }))
vi.mock('src/components/Query/AdvancedQuery.vue', () => ({
  default: { name: 'AdvancedQuery', template: '<div />' },
}))
vi.mock('src/components/QueryScheme/QuerySchemePreview.vue', () => ({
  default: { name: 'QuerySchemePreview', template: '<div />' },
}))
vi.mock('src/components/Select/RoleSelect.vue', () => ({
  default: { name: 'RoleSelect', template: '<div />' },
}))
vi.mock('quasar', async (importOriginal: () => Promise<Record<string, unknown>>) => ({
  ...(await importOriginal()),
  useQuasar: () => ({ screen: { lt: { md: false } } }),
}))

import QuerySchemeEditDialog from './QuerySchemeEditDialog.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const ButtonStub = defineComponent({
  props: { label: String, disable: Boolean },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () => h('button', {
      disabled: props.disable,
      onClick: () => emit('click'),
    }, props.label || slots.default?.())
  },
})

describe('QuerySchemeEditDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.getScopeConfig.mockResolvedValue({
      data: {
        scope_code: 'system.user.list',
        scope_label: '用户管理',
        table_code: 'sys_user',
        quick_presets: [],
        virtual_sort_fields: [],
        dynamic_binding_kinds: [],
      },
    })
    api.updateShared.mockResolvedValue({ data: { id: 7 } })
    tableApi.queryRuntimeTableByCode.mockResolvedValue({ data: { table_fields: [] } })
  })

  it('updates shared content once and leaves enabled state to the manager action', async () => {
    const wrapper = mount(QuerySchemeEditDialog, {
      props: {
        modelValue: false,
        schemeType: QuerySchemeType.PUBLIC,
        scopeOptions: [{ label: '用户管理', value: 'system.user.list' }],
        detail: {
          id: 7,
          name: '公共用户查询',
          scope_code: 'system.user.list',
          scope_label: '用户管理',
          type: QuerySchemeType.PUBLIC,
          is_default: false,
          enabled: false,
          revision: 3,
          status: QuerySchemeValidationStatus.VALID,
          updated_at: '2026-08-19 10:00:00',
          query_payload: {
            expressions: [],
            quick_query: { keyword: '' },
            order: { field: '', is_asc: false },
            bindings: [],
          },
          issues: [],
        },
      },
      global: {
        directives: { closePopup: () => undefined },
        stubs: {
          QDialog: SlotStub,
          QCard: SlotStub,
          QCardSection: SlotStub,
          QCardActions: SlotStub,
          QBtn: ButtonStub,
          QSpace: true,
          QSeparator: true,
          QSelect: true,
          QInput: true,
          QCheckbox: true,
          QBanner: true,
          QTooltip: true,
          AdvancedQuery: true,
          QuerySchemePreview: true,
          RoleSelect: true,
        },
      },
    })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    await nextTick()

    const save = wrapper.findAll('button').find((button) => button.text() === '保存')
    expect(save).toBeDefined()
    await save?.trigger('click')
    await flushPromises()

    expect(api.updateShared).toHaveBeenCalledOnce()
    expect(api.updateShared).toHaveBeenCalledWith(
      7,
      expect.objectContaining({ name: '公共用户查询', revision: 3 }),
    )
    expect(api.setSharedEnabled).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('启用')
  })
})
