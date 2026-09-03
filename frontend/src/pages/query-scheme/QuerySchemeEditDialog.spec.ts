import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { QuerySchemeType, QuerySchemeValidationStatus } from '@/modules/query-scheme/types'

const api = vi.hoisted(() => ({
  getScopeConfig: vi.fn(),
  updateShared: vi.fn(),
  setSharedEnabled: vi.fn(),
}))
const tableApi = vi.hoisted(() => ({ queryRuntimeTableByCode: vi.fn() }))
vi.mock('@/api/services/query-scheme', () => ({ useQuerySchemeApi: () => api }))
vi.mock('@/api/services/sys-table', () => ({ useTableApi: () => tableApi }))
vi.mock('@/components/Query/AdvancedQuery.vue', () => ({
  default: {
    name: 'AdvancedQuery',
    props: ['modelValue', 'queryModel', 'bindings', 'usage', 'title'],
    emits: [
      'update:modelValue',
      'update:queryModel',
      'update:bindings',
      'confirm',
      'cancel',
    ],
    template: '<div />',
  },
}))
vi.mock('@/components/QueryScheme/QuerySchemePreview.vue', () => ({
  default: { name: 'QuerySchemePreview', props: ['payload', 'fields'], template: '<div />' },
}))
vi.mock('@/components/Select/RoleSelect.vue', () => ({
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

  it('commits condition drafts only after the second dialog is confirmed', async () => {
    const wrapper = mount(QuerySchemeEditDialog, {
      props: {
        modelValue: false,
        schemeType: QuerySchemeType.PUBLIC,
        scopeOptions: [{ label: '用户管理', value: 'system.user.list' }],
        detail: {
          id: 8,
          name: '公共用户查询',
          scope_code: 'system.user.list',
          scope_label: '用户管理',
          type: QuerySchemeType.PUBLIC,
          is_default: false,
          enabled: true,
          revision: 2,
          status: QuerySchemeValidationStatus.VALID,
          updated_at: '2026-08-19 10:00:00',
          query_payload: {
            expressions: [],
            quick_query: { keyword: 'before' },
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
          RoleSelect: true,
        },
      },
    })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text() === '编辑查询条件')!.trigger('click')
    await nextTick()

    const editor = wrapper.findComponent({ name: 'AdvancedQuery' })
    const preview = wrapper.findComponent({ name: 'QuerySchemePreview' })
    expect(editor.props('usage')).toBe('scheme-condition-editor')
    expect(editor.props('title')).toBe('编辑查询条件')
    editor.vm.$emit('update:queryModel', {
      ...editor.props('queryModel'),
      quick_query: { keyword: 'cancelled' },
    })
    editor.vm.$emit('cancel')
    await nextTick()
    expect(preview.props('payload').quick_query.keyword).toBe('before')

    await wrapper.findAll('button').find((button) => button.text() === '编辑查询条件')!.trigger('click')
    editor.vm.$emit('update:queryModel', {
      ...editor.props('queryModel'),
      quick_query: { keyword: 'confirmed' },
    })
    editor.vm.$emit('confirm')
    await nextTick()
    expect(preview.props('payload').quick_query.keyword).toBe('confirmed')
  })

  it('absorbs a revision conflict after the shared interceptor reports it', async () => {
    api.updateShared.mockRejectedValueOnce(
      new Error('方案已被其他操作更新，请刷新后重试'),
    )
    const wrapper = mount(QuerySchemeEditDialog, {
      props: {
        modelValue: false,
        schemeType: QuerySchemeType.PUBLIC,
        scopeOptions: [{ label: '用户管理', value: 'system.user.list' }],
        detail: {
          id: 9,
          name: '并发查询方案',
          scope_code: 'system.user.list',
          scope_label: '用户管理',
          type: QuerySchemeType.PUBLIC,
          is_default: false,
          enabled: true,
          revision: 4,
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
          RoleSelect: true,
        },
      },
    })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text() === '保存')!.trigger('click')
    await flushPromises()

    expect(api.updateShared).toHaveBeenCalledOnce()
    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})
