import { computed, defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  queryDimensions: vi.fn(),
  queryResources: vi.fn(),
  getResource: vi.fn(),
  createResource: vi.fn(),
  updateResource: vi.fn(),
  listResourceOperations: vi.fn(),
  replaceResourceOperations: vi.fn(),
  setResourcePermission: vi.fn(),
  queryOwnerships: vi.fn(),
  listResourceOwnerships: vi.fn(),
  getOwnership: vi.fn(),
  createOwnership: vi.fn(),
  updateOwnership: vi.fn(),
  disableOwnership: vi.fn(),
  queryPolicies: vi.fn(),
  getPolicy: vi.fn(),
  queryPolicyRules: vi.fn(),
  createPolicy: vi.fn(),
  updatePolicy: vi.fn(),
  replacePolicyRules: vi.fn(),
  setPolicyState: vi.fn(),
  queryGrants: vi.fn(),
  getGrant: vi.fn(),
  createGrant: vi.fn(),
  setGrantState: vi.fn(),
  preflight: vi.fn(),
}))

const permissionButtons = vi.hoisted(() => ({
  top: [
    {
      id: 1,
      name: '新增资源',
      event_action: 'create_resource',
      icon: 'add',
      color: 'primary',
    },
    {
      id: 2,
      name: '新增策略',
      event_action: 'create_policy',
      icon: 'add',
      color: 'primary',
    },
  ],
  line: [],
}))

vi.mock('boot/axios', () => ({
  instance: {},
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ meta: { icon: 'shield' } }),
}))

vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))

vi.mock('src/components/Table/TablePagination.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('src/components/Query/AdvancedQuery.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('./components/DataPermissionDetailDialog.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('src/api/services/data-permission-config', () => {
  return {
    useDataPermissionConfigApi: () => apiMocks,
  }
})

vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => ({
    top_buttons: computed(() => permissionButtons.top),
    line_buttons: computed(() => permissionButtons.line),
  }),
}))

vi.mock('src/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({
    confirmAction: vi.fn(() => ({ onOk: vi.fn() })),
  }),
}))

vi.mock('src/api/services/sys-table', () => {
  return {
    useTableApi: () => ({
      queryTable: vi.fn().mockResolvedValue({ data: [], total: 0 }),
      queryTableById: vi.fn().mockResolvedValue({ data: { table_fields: [] } }),
    }),
  }
})

vi.mock('src/api/services/sys-role', () => ({
  useRoleApi: () => ({
    queryRole: vi.fn().mockResolvedValue({ data: [], total: 0 }),
  }),
}))

vi.mock('src/api/services/sys-user', () => ({
  useSysUserApi: () => ({
    queryUser: vi.fn().mockResolvedValue({ data: [], total: 0 }),
  }),
}))

import DataPermissionPage from './Index.vue'
import DataPermissionConfigDialog from './components/DataPermissionConfigDialog.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const ButtonStub = defineComponent({
  name: 'QBtn',
  inheritAttrs: false,
  props: {
    label: { type: String, default: '' },
    icon: { type: String, default: '' },
  },
  emits: ['click'],
  setup(props, { emit }) {
    return () =>
      h(
        'button',
        {
          'data-icon': props.icon,
          onClick: () => emit('click'),
        },
        props.label,
      )
  },
})

const AvatarStub = defineComponent({
  name: 'QAvatar',
  props: {
    icon: { type: String, default: '' },
  },
  setup(props) {
    return () => h('div', { 'data-testid': 'page-avatar', 'data-icon': props.icon })
  },
})

const InputStub = defineComponent({
  name: 'QInput',
  props: {
    modelValue: { type: [String, Number], default: '' },
    placeholder: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'keyup'],
  setup(props, { emit, slots }) {
    return () =>
      h('label', [
        h('input', {
          value: props.modelValue,
          placeholder: props.placeholder,
          onInput: (event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
        }),
        slots.append?.(),
      ])
  },
})

const TableStub = defineComponent({
  name: 'QTable',
  props: {
    rows: { type: Array, default: () => [] },
  },
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-testid': 'table', 'data-row-count': props.rows.length }, [
        slots.top?.(),
        slots.bottom?.(),
      ])
  },
})

const TabStub = defineComponent({
  name: 'QTab',
  props: {
    label: { type: String, default: '' },
  },
  setup(props) {
    return () => h('button', props.label)
  },
})

const ConfigDialogStub = defineComponent({
  name: 'DataPermissionConfigDialog',
  props: {
    modelValue: Boolean,
    kind: { type: String, default: '' },
  },
  setup(props) {
    return () =>
      props.modelValue
        ? h('div', { 'data-testid': 'config-dialog', 'data-kind': props.kind })
        : null
  },
})

const DetailDialogStub = defineComponent({
  name: 'DataPermissionDetailDialog',
  setup() {
    return () => null
  },
})

const mountPage = () =>
  shallowMount(DataPermissionPage, {
    global: {
      stubs: {
        BaseContent: SlotStub,
        QCard: SlotStub,
        QCardSection: SlotStub,
        QAvatar: AvatarStub,
        QSpace: true,
        QSeparator: true,
        QTabs: SlotStub,
        QTab: TabStub,
        QTabPanels: SlotStub,
        QTabPanel: SlotStub,
        QTable: TableStub,
        QTd: SlotStub,
        QBadge: SlotStub,
        QInput: InputStub,
        QIcon: true,
        QBtn: ButtonStub,
        QTooltip: SlotStub,
        QSelect: true,
        QBanner: SlotStub,
        TablePagination: true,
        AdvancedQuery: true,
        DataPermissionConfigDialog: ConfigDialogStub,
        DataPermissionDetailDialog: DetailDialogStub,
      },
    },
  })

const emptyList = () => Promise.resolve({ data: [], total: 0 })

describe('Data permission configuration center', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
    apiMocks.queryDimensions.mockImplementation(emptyList)
    apiMocks.queryResources.mockImplementation(emptyList)
    apiMocks.queryOwnerships.mockImplementation(emptyList)
    apiMocks.queryPolicies.mockImplementation(emptyList)
    apiMocks.queryPolicyRules.mockImplementation(emptyList)
    apiMocks.queryGrants.mockImplementation(emptyList)
  })

  it('renders the configuration center and loads resource configuration data', async () => {
    apiMocks.queryResources.mockResolvedValue({
      data: [
        {
          id: 10,
          resource_code: 'transport_order',
          name: '运输订单',
          resource_type: 'business_service',
          permission_enabled: false,
          state: true,
        },
      ],
      total: 1,
    })

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('数据权限')
    expect(wrapper.find('[data-testid="page-avatar"]').attributes('data-icon')).toBe('shield')
    expect(wrapper.text()).toContain('数据资源')
    expect(wrapper.text()).toContain('归属定义')
    expect(wrapper.text()).toContain('权限策略')
    expect(wrapper.text()).toContain('权限授权')
    expect(wrapper.text()).toContain('配置检查')
    expect(apiMocks.queryResources).toHaveBeenCalled()
    expect(apiMocks.queryDimensions).toHaveBeenCalled()
    const table = wrapper.find('[data-panel="resources"] [data-testid="table"]')
    expect(table.attributes('data-row-count')).toBe('1')
    expect(table.text()).toContain('新增资源')
  })

  it('keeps top actions inside their matching list panels', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const resourcePanel = wrapper.find('[data-panel="resources"]')
    const policyPanel = wrapper.find('[data-panel="policies"]')
    expect(resourcePanel.text()).toContain('新增资源')
    expect(resourcePanel.text()).not.toContain('新增策略')
    expect(policyPanel.text()).toContain('新增策略')
    expect(policyPanel.text()).not.toContain('新增资源')
  })

  it('loads each list once and preserves its state when switching back', async () => {
    const wrapper = mountPage()
    await flushPromises()
    apiMocks.queryPolicies.mockClear()

    await wrapper.find('[data-tab="policies"]').trigger('click')
    await flushPromises()

    expect(apiMocks.queryPolicies).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('组合归属、范围来源和关系规则')
    const table = wrapper.find('[data-panel="policies"] [data-testid="table"]')
    expect(table.text()).toContain('新增策略')
    expect(table.text()).not.toContain('新增资源')

    await wrapper.find('[data-tab="resources"]').trigger('click')
    await wrapper.find('[data-tab="policies"]').trigger('click')
    await flushPromises()

    expect(apiMocks.queryPolicies).toHaveBeenCalledTimes(1)
  })

  it('sends the keyword through the resource query and opens the configured create dialog', async () => {
    const wrapper = mountPage()
    await flushPromises()
    apiMocks.queryResources.mockClear()

    const resourcePanel = wrapper.find('[data-panel="resources"]')
    const keywordInput = resourcePanel.find('input[placeholder="搜索关键词"]')
    await keywordInput.setValue('运输')
    const queryButton = resourcePanel.findAll('button').find((button) => button.text() === '搜索')
    await queryButton?.trigger('click')
    await flushPromises()

    expect(apiMocks.queryResources).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 1,
        quick_query: { keyword: '运输' },
      }),
    )

    const createButton = resourcePanel
      .findAll('button')
      .find((button) => button.text() === '新增资源')
    await createButton?.trigger('click')
    expect(wrapper.find('[data-testid="config-dialog"]').attributes('data-kind')).toBe('resource')
  })

  it('preserves quick queries independently across list panels', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const resourceInput = wrapper.find('[data-panel="resources"] input[placeholder="搜索关键词"]')
    await resourceInput.setValue('运输')

    await wrapper.find('[data-tab="policies"]').trigger('click')
    await flushPromises()
    const policyInput = wrapper.find('[data-panel="policies"] input[placeholder="搜索关键词"]')
    await policyInput.setValue('组织策略')

    apiMocks.queryResources.mockClear()
    await wrapper.find('[data-tab="resources"]').trigger('click')
    await flushPromises()

    expect(resourceInput.element).toHaveProperty('value', '运输')
    expect(policyInput.element).toHaveProperty('value', '组织策略')
    expect(apiMocks.queryResources).not.toHaveBeenCalled()
  })

  it('stops submission when the common form validation fails', async () => {
    const validate = vi.fn().mockResolvedValue(false)
    const FormDialogStub = defineComponent({
      name: 'FormDialogShell',
      emits: ['submit', 'update:modelValue'],
      setup(_, { emit, slots }) {
        return () =>
          h('section', [
            slots.default?.(),
            h('button', { 'data-testid': 'submit', onClick: () => emit('submit') }, '提交'),
          ])
      },
    })
    const FormStub = defineComponent({
      name: 'QForm',
      setup(_, { expose, slots }) {
        expose({ validate })
        return () => h('form', slots.default?.())
      },
    })
    const dialog = shallowMount(DataPermissionConfigDialog, {
      props: {
        modelValue: true,
        kind: 'resource',
      },
      global: {
        stubs: {
          FormDialogShell: FormDialogStub,
          QForm: FormStub,
          QInput: InputStub,
          QSelect: true,
          QToggle: true,
        },
      },
    })
    await flushPromises()
    await dialog.find('[data-testid="submit"]').trigger('click')
    await flushPromises()

    expect(validate).toHaveBeenCalled()
    expect(apiMocks.createResource).not.toHaveBeenCalled()
  })
})
