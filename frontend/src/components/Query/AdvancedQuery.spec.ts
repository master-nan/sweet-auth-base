import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ExpressionLogic, ExpressionType, SysTableFieldType } from 'src/types/enum'
import type { Query, QueryRule } from 'src/types/global'
import type { OrganizationSelectorType } from 'src/types/organization-selector'

const loadDictsMock = vi.hoisted(() => vi.fn())
const notifyMock = vi.hoisted(() => vi.fn())
const postMock = vi.hoisted(() => vi.fn())

vi.mock('quasar', () => ({
  QForm: {},
  useQuasar: () => ({
    screen: {
      lt: {
        md: false,
      },
    },
    notify: notifyMock,
  }),
}))

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

vi.mock('src/router', () => ({
  Router: { currentRoute: { value: { name: '' } } },
}))

vi.mock('boot/axios', () => ({
  instance: {
    post: postMock,
  },
}))

import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
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

const QBtnStub = defineComponent({
  name: 'QBtn',
  props: { label: String },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          type: 'button',
          onClick: () => emit('click'),
        },
        props.label || slots.default?.(),
      )
  },
})

const AdvancedQueryRuleRowStub = defineComponent({
  name: 'AdvancedQueryRuleRow',
  props: {
    rule: {
      type: Object,
      required: true,
    },
    expressionTypeOptionsForRule: {
      type: Function,
      required: true,
    },
    organizationSelectorConfigForRule: {
      type: Function,
      required: true,
    },
    updateOrganizationSelectorValue: {
      type: Function,
      required: true,
    },
  },
  emits: ['update:logic'],
  setup() {
    return () => h('div', { 'data-testid': 'advanced-query-rule-row' })
  },
})

const makeField = (selectorType?: OrganizationSelectorType) => ({
  field_code: selectorType ? `${selectorType}_id` : 'remark',
  field_name: selectorType || 'remark',
  field_type: selectorType ? SysTableFieldType.BIGINT : SysTableFieldType.VARCHAR,
  input_type: selectorType ? 'selector' : 'input',
  selector_type: selectorType,
  dict_code: '',
  linkage_config: '',
  allowed_operators: selectorType
    ? undefined
    : [ExpressionType.LIKE, ExpressionType.EQ, ExpressionType.IS_NULL],
})

const makeQuery = (fieldCode: string, expressionType = ExpressionType.EQ): Query => ({
  page: 1,
  num: 20,
  expressions: [
    {
      logic: ExpressionLogic.AND,
      rules: [
        {
          field: fieldCode,
          expression_type: expressionType,
          value: expressionType === ExpressionType.IN ? [] : null,
        },
      ],
      nested: [],
    },
  ],
})

const mountQuery = async (
  selectorType?: OrganizationSelectorType,
  expressionType = ExpressionType.EQ,
  usage: 'business-query' | 'scheme-condition-editor' = 'business-query',
) => {
  const field = makeField(selectorType)
  const query = makeQuery(field.field_code, expressionType)
  const wrapper = mount(AdvancedQuery, {
    props: {
      modelValue: true,
      queryModel: query,
      fields: [field],
      usage,
    },
    global: {
      stubs: {
        QDialog: SlotStub,
        QCard: SlotStub,
        QCardSection: SlotStub,
        QCardActions: SlotStub,
        QForm: QFormStub,
        QBtn: QBtnStub,
        QIcon: true,
        QSpace: true,
        QSeparator: true,
        QTooltip: true,
        AdvancedQueryRuleRow: AdvancedQueryRuleRowStub,
      },
    },
  })
  await nextTick()
  return { wrapper, query }
}

const ruleRowFunctions = (wrapper: Awaited<ReturnType<typeof mountQuery>>['wrapper']) => {
  const row = wrapper.findComponent({ name: 'AdvancedQueryRuleRow' })
  return {
    rule: row.props('rule') as QueryRule,
    resolveConfig: row.props('organizationSelectorConfigForRule') as (
      rule: QueryRule,
    ) => { selectorType: OrganizationSelectorType } | null,
    expressionOptions: row.props('expressionTypeOptionsForRule') as (
      rule: QueryRule,
    ) => Array<{ label: string; value: ExpressionType }>,
    updateValue: row.props('updateOrganizationSelectorValue') as (
      rule: QueryRule,
      value: unknown,
    ) => void,
  }
}

describe('AdvancedQuery organization selector integration', () => {
  beforeEach(() => {
    loadDictsMock.mockResolvedValue(undefined)
    postMock.mockResolvedValue({ data: { data: [] } })
  })

  it.each<OrganizationSelectorType>(['employee', 'position', 'org_unit', 'legal_entity'])(
    'resolves %s metadata through the shared selector resolver',
    async (selectorType) => {
      const { wrapper } = await mountQuery(selectorType)
      const { rule, resolveConfig } = ruleRowFunctions(wrapper)

      expect(resolveConfig(rule)).toMatchObject({ selectorType })
    },
  )

  it('limits selector operators to equals and in', async () => {
    const { wrapper } = await mountQuery('employee')
    const { rule, expressionOptions } = ruleRowFunctions(wrapper)

    expect(expressionOptions(rule).map((option) => option.value)).toEqual([
      ExpressionType.EQ,
      ExpressionType.IN,
    ])
  })

  it('normalizes selector submissions to internal IDs only', async () => {
    const { wrapper } = await mountQuery('employee')
    const functions = ruleRowFunctions(wrapper)

    functions.updateValue(functions.rule, '21')
    expect(functions.rule.value).toBe(21)

    functions.rule.expression_type = ExpressionType.IN
    functions.updateValue(functions.rule, [21, '22', 'EMP-23', 21])
    expect(functions.rule.value).toEqual([21, 22])

    const searchButton = wrapper
      .findAllComponents(QBtnStub)
      .find((button) => button.text().includes('搜索'))
    expect(searchButton).toBeDefined()
    await searchButton!.trigger('click')
    await nextTick()

    const submitted = wrapper.emitted('update:queryModel')?.at(-1)?.[0] as Query
    expect(submitted.expressions[0]?.rules[0]).toMatchObject({
      field: 'employee_id',
      expression_type: ExpressionType.IN,
      value: [21, 22],
    })
  })

  it('keeps ordinary field metadata on its existing query behavior', async () => {
    const { wrapper } = await mountQuery()
    const { rule, resolveConfig, expressionOptions } = ruleRowFunctions(wrapper)

    expect(resolveConfig(rule)).toBeNull()
    expect(expressionOptions(rule).map((option) => option.value)).toContain(
      ExpressionType.LIKE,
    )
  })

  it('uses reset and search actions for immediate business queries', async () => {
    const { wrapper } = await mountQuery()

    expect(wrapper.text()).toContain('重置')
    expect(wrapper.text()).toContain('搜索')
    expect(wrapper.text()).not.toContain('确定')
    expect(wrapper.text()).not.toContain('取消')
  })

  it('uses cancel and confirm actions when editing scheme conditions', async () => {
    const { wrapper, query } = await mountQuery(undefined, ExpressionType.EQ, 'scheme-condition-editor')
    query.expressions[0]!.rules[0]!.value = 'ready'

    expect(wrapper.text()).toContain('取消')
    expect(wrapper.text()).toContain('确定')
    expect(wrapper.text()).not.toContain('重置')
    const confirm = wrapper.findAllComponents(QBtnStub).find((button) => button.text() === '确定')
    await confirm!.trigger('click')
    await nextTick()

    expect(wrapper.emitted('confirm')).toHaveLength(1)
    expect(wrapper.emitted('search')).toBeUndefined()
  })
})
