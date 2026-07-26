import { defineComponent, h, nextTick, ref } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { TableField } from 'src/api/services/sys-table'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import type { OrganizationSelectorType } from 'src/types/organization-selector'

const loadDictsMock = vi.hoisted(() => vi.fn())
const queryDictMock = vi.hoisted(() => vi.fn())
const postMock = vi.hoisted(() => vi.fn())

vi.mock('src/stores/dict', () => ({
  useDictStore: () => ({
    getDictOptions: () => [],
    loadDicts: loadDictsMock,
  }),
}))

vi.mock('src/stores/user', () => ({
  useUserStore: () => ({
    menus: [],
  }),
}))

vi.mock('src/stores/loading', () => ({
  useLoadingStore: () => ({
    loading: false,
  }),
}))

vi.mock('src/api/services/sys-dict', () => ({
  useDictApi: () => ({
    queryDict: queryDictMock,
  }),
}))

vi.mock('boot/axios', () => ({
  instance: {
    post: postMock,
  },
}))

vi.mock('pinia', () => ({
  storeToRefs: () => ({
    loading: ref(false),
  }),
}))

import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'

const FormDialogShellStub = defineComponent({
  name: 'FormDialogShell',
  emits: ['submit', 'update:modelValue'],
  setup(_, { slots }) {
    return () => h('section', { 'data-testid': 'form-dialog-shell' }, slots.default?.())
  },
})

const QFormStub = defineComponent({
  name: 'QForm',
  setup(_, { expose, slots }) {
    expose({
      validate: vi.fn().mockResolvedValue(true),
      resetValidation: vi.fn(),
    })
    return () => h('form', slots.default?.())
  },
})

const OrganizationSelectStub = defineComponent({
  name: 'OrganizationSelect',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [Number, Array],
      default: null,
    },
    selectorType: {
      type: String,
      required: true,
    },
    multiple: Boolean,
    includeHistory: Boolean,
    disabled: Boolean,
    clearable: Boolean,
  },
  emits: ['update:modelValue'],
  setup() {
    return () => h('div', { 'data-testid': 'organization-select' })
  },
})

const QInputStub = defineComponent({
  name: 'QInput',
  props: {
    modelValue: {
      type: [String, Number],
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup() {
    return () => h('input', { 'data-testid': 'q-input' })
  },
})

type SelectorMetadata = {
  selector_type?: OrganizationSelectorType
  multiple?: boolean
  include_history?: boolean
  disabled?: boolean
}

const makeField = (
  fieldCode: string,
  metadata: SelectorMetadata = {},
  overrides: Partial<TableField> = {},
) =>
  ({
    id: 1,
    table_id: 1,
    field_name: fieldCode,
    field_code: fieldCode,
    field_type: SysTableFieldType.BIGINT,
    field_length: 0,
    field_decimal_length: 0,
    input_type: SysTableFieldInputType.SELECT,
    default_value: '',
    dict_code: '',
    is_primary_key: false,
    is_index: false,
    is_quick_search: false,
    is_advanced_search: false,
    is_sort: false,
    is_null: true,
    is_list_show: true,
    is_insert_show: true,
    is_update_show: true,
    sequence: 1,
    original_field_id: 0,
    binding: '',
    linkage_config: '',
    ...metadata,
    ...overrides,
  }) as TableField

const mountDialog = async (field: TableField, editData: Record<string, unknown> = {}) => {
  const wrapper = shallowMount(DynamicFormDialog, {
    props: {
      modelValue: true,
      fields: [field],
      editData,
    },
    global: {
      stubs: {
        FormDialogShell: FormDialogShellStub,
        QForm: QFormStub,
        OrganizationSelect: OrganizationSelectStub,
        QInput: QInputStub,
      },
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

describe('DynamicFormDialog organization selector integration', () => {
  beforeEach(() => {
    loadDictsMock.mockResolvedValue(undefined)
    queryDictMock.mockResolvedValue({ data: [] })
    postMock.mockResolvedValue({ data: { data: [] } })
  })

  it.each<[OrganizationSelectorType, string]>([
    ['employee', 'employee_id'],
    ['position', 'position_id'],
    ['org_unit', 'org_unit_id'],
    ['legal_entity', 'legal_entity_id'],
  ])('renders %s metadata with OrganizationSelect', async (selectorType, fieldCode) => {
    const wrapper = await mountDialog(makeField(fieldCode, { selector_type: selectorType }))
    const selector = wrapper.findComponent(OrganizationSelectStub)

    expect(selector.exists()).toBe(true)
    expect(selector.props('selectorType')).toBe(selectorType)
  })

  it('keeps an ordinary field on its existing input path', async () => {
    const wrapper = await mountDialog(
      makeField(
        'remark',
        {},
        {
          field_type: SysTableFieldType.VARCHAR,
          input_type: SysTableFieldInputType.INPUT,
        },
      ),
    )

    expect(wrapper.findComponent(OrganizationSelectStub).exists()).toBe(false)
    expect(wrapper.find('[data-testid="q-input"]').exists()).toBe(true)
  })

  it('gives selector metadata priority over dictionary and linkage configuration', async () => {
    const wrapper = await mountDialog(
      makeField(
        'employee_id',
        { selector_type: 'employee' },
        {
          dict_code: 'org_employment_status',
          linkage_config: JSON.stringify({
            linkage: {
              enabled: true,
              mode: 'relation',
              tableCode: 'org_employee',
            },
          }),
        },
      ),
    )

    expect(wrapper.findComponent(OrganizationSelectStub).exists()).toBe(true)
    expect(loadDictsMock).not.toHaveBeenCalled()
    expect(postMock).not.toHaveBeenCalled()
  })

  it('normalizes selector values to internal IDs before submit', async () => {
    const wrapper = await mountDialog(
      makeField('employee_id', { selector_type: 'employee' }),
      { employee_id: 12 },
    )
    const selector = wrapper.findComponent(OrganizationSelectStub)

    selector.vm.$emit('update:modelValue', 31)
    await nextTick()
    wrapper.findComponent(FormDialogShellStub).vm.$emit('submit')
    await flushPromises()

    expect(wrapper.emitted('submit')?.at(-1)?.[0]).toMatchObject({
      data: {
        employee_id: 31,
      },
    })
  })

  it('passes multiple, history, disabled and existing values to the selector', async () => {
    const wrapper = await mountDialog(
      makeField('position_id', {
        selector_type: 'position',
        multiple: true,
        include_history: true,
        disabled: true,
      }),
      { position_id: [8, 9] },
    )
    const selector = wrapper.findComponent(OrganizationSelectStub)

    expect(selector.props()).toMatchObject({
      modelValue: [8, 9],
      selectorType: 'position',
      multiple: true,
      includeHistory: true,
      disabled: true,
    })
  })
})
