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
  line: [] as Array<{
    id: number
    name: string
    event_action: string
    icon: string
    color: string
    position: number
  }>,
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
import DataPermissionDetailDialog from './components/DataPermissionDetailDialog.vue'

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
    color: { type: String, default: '' },
    disable: Boolean,
  },
  emits: ['click'],
  setup(props, { attrs, emit }) {
    return () =>
      h(
        'button',
        {
          'data-icon': props.icon,
          'data-color': props.color,
          'aria-label': attrs['aria-label'],
          disabled: props.disable,
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

const SelectStub = defineComponent({
  name: 'QSelect',
  props: {
    modelValue: { type: [String, Number, Boolean, Array, Object], default: null },
    label: { type: String, default: '' },
    options: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('button', {
        'data-testid': 'select',
        'data-label': props.label,
        'data-empty':
          props.modelValue === null || props.modelValue === undefined ? 'true' : 'false',
        'data-value':
          props.modelValue === null || props.modelValue === undefined
            ? ''
            : ['string', 'number', 'boolean'].includes(typeof props.modelValue)
              ? `${props.modelValue as string | number | boolean}`
              : '',
        onClick: () =>
          emit(
            'update:modelValue',
            (props.options[0] as { value?: unknown } | undefined)?.value ?? null,
          ),
      })
  },
})

const TableStub = defineComponent({
  name: 'QTable',
  props: {
    rows: { type: Array, default: () => [] },
    columns: { type: Array, default: () => [] },
  },
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-testid': 'table', 'data-row-count': props.rows.length }, [
        slots.top?.(),
        ...props.rows.flatMap((row: any) =>
          props.columns.map((column: any) => {
            const value = typeof column.field === 'function' ? column.field(row) : row[column.field]
            const cell = slots[`body-cell-${column.name}`]
            return cell ? cell({ row, value }) : h('span', value)
          }),
        ),
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
    permissionButtons.line.splice(0)
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

  it('renders a grant subject name without exposing its numeric role id', async () => {
    apiMocks.queryGrants.mockResolvedValue({
      data: [
        {
          id: 77,
          subject_type: 'role',
          subject_id: 9527,
          subject: { id: 9527, name: '华东只读角色', code: '' },
          resource_id: 10,
          operation: 'query',
          policy_id: 20,
          state: true,
        },
      ],
      total: 1,
    })
    const wrapper = mountPage()
    await flushPromises()
    await wrapper.find('[data-tab="grants"]').trigger('click')
    await flushPromises()

    const table = wrapper.find('[data-panel="grants"] [data-testid="table"]')
    expect(table.text()).toContain('华东只读角色')
    expect(table.text()).not.toContain('#9527')
  })

  it('keeps unselected ownership and grant references empty instead of rendering zero', async () => {
    const FormDialogStub = defineComponent({
      name: 'FormDialogShell',
      setup(_, { slots }) {
        return () => h('section', slots.default?.())
      },
    })
    const FormStub = defineComponent({
      name: 'QForm',
      setup(_, { expose, slots }) {
        expose({ validate: vi.fn().mockResolvedValue(true), resetValidation: vi.fn() })
        return () => h('form', slots.default?.())
      },
    })
    const mountDialog = (kind: 'ownership' | 'grant') =>
      shallowMount(DataPermissionConfigDialog, {
        props: { modelValue: true, kind },
        global: {
          stubs: {
            FormDialogShell: FormDialogStub,
            QForm: FormStub,
            QInput: InputStub,
            QSelect: SelectStub,
            QToggle: true,
            QList: SlotStub,
            QItem: SlotStub,
            QItemSection: SlotStub,
            QSpace: true,
            QBtn: ButtonStub,
          },
        },
      })

    const ownershipDialog = mountDialog('ownership')
    await flushPromises()
    for (const label of ['数据资源', '数据维度']) {
      expect(ownershipDialog.find(`[data-label="${label}"]`).attributes('data-empty')).toBe('true')
    }

    const grantDialog = mountDialog('grant')
    await flushPromises()
    for (const label of ['授权主体', '数据资源', '资源操作', '权限策略']) {
      expect(grantDialog.find(`[data-label="${label}"]`).attributes('data-empty')).toBe('true')
    }
    expect(grantDialog.text()).not.toContain('>0<')
  })

  it('shows grant references and policy rules with business labels in detail', async () => {
    apiMocks.getGrant.mockResolvedValue({
      data: {
        id: 77,
        subject_type: 'role',
        subject_id: 9527,
        subject: { id: 9527, name: '华东只读角色', code: 'east_readonly' },
        resource_id: 10,
        resource: { id: 10, name: '运输订单', code: 'transport_order' },
        operation: 'query',
        policy_id: 20,
        policy: { id: 20, name: '本组织及下级', code: 'org_descendants' },
        state: true,
      },
    })
    const detail = shallowMount(DataPermissionDetailDialog, {
      props: { modelValue: false, kind: 'grant', id: 77 },
      global: {
        stubs: {
          FormDialogShell: SlotStub,
          QInnerLoading: true,
          QSpinner: true,
          QList: SlotStub,
          QItem: SlotStub,
          QItemSection: SlotStub,
          QItemLabel: SlotStub,
          QBadge: SlotStub,
          QChip: SlotStub,
          QTable: TableStub,
        },
      },
    })
    await detail.setProps({ modelValue: true })
    await flushPromises()

    expect(detail.text()).toContain('华东只读角色 · east_readonly')
    expect(detail.text()).toContain('运输订单')
    expect(detail.text()).toContain('本组织及下级')
    expect(detail.text()).not.toContain('#9527')

    apiMocks.getPolicy.mockResolvedValue({
      data: {
        id: 20,
        policy_code: 'org_descendants',
        name: '本组织及下级',
        state: true,
        rules: [
          {
            id: 21,
            sequence: 1,
            ownership_code: 'management_org_id',
            dimension_id: 3,
            dimension: { id: 3, name: '管理组织', code: 'management_org' },
            scope_source: 'effective_org_units',
            relation: 'self_and_descendants',
            operator: 'in',
          },
        ],
      },
    })
    await detail.setProps({ kind: 'policy', id: 20 })
    await flushPromises()

    expect(detail.text()).toContain('当前有效组织')
    expect(detail.text()).toContain('本级及下级')
    expect(detail.text()).toContain('包含于')
    expect(detail.text()).not.toContain('effective_org_units')
  })

  it('shows enable and disable actions from the current row state with one permission button', async () => {
    permissionButtons.line.push({
      id: 3,
      name: '启停数据权限',
      event_action: 'toggle_permission',
      icon: 'verified_user',
      color: 'warning',
      position: 1,
    })
    apiMocks.queryResources.mockResolvedValue({
      data: [
        {
          id: 10,
          resource_code: 'enabled_resource',
          name: '已启用资源',
          resource_type: 'business_service',
          permission_enabled: true,
          state: true,
        },
        {
          id: 11,
          resource_code: 'disabled_resource',
          name: '未启用资源',
          resource_type: 'business_service',
          permission_enabled: false,
          state: true,
        },
      ],
      total: 2,
    })

    const wrapper = mountPage()
    await flushPromises()

    const disableButton = wrapper.find('button[aria-label="停用数据权限"]')
    const enableButton = wrapper.find('button[aria-label="启用数据权限"]')
    expect(disableButton.attributes('data-icon')).toBe('pause_circle')
    expect(disableButton.attributes('data-color')).toBe('warning')
    expect(enableButton.attributes('data-icon')).toBe('play_circle')
    expect(enableButton.attributes('data-color')).toBe('positive')
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
