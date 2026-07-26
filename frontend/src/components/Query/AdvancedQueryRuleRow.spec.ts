import { defineComponent, h, nextTick } from 'vue'
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('boot/axios', () => ({
  instance: {
    get: vi.fn(),
  },
}))

import AdvancedQueryRuleRow from 'src/components/Query/AdvancedQueryRuleRow.vue'
import { ExpressionLogic, ExpressionType } from 'src/types/enum'
import type { QueryRule } from 'src/types/global'
import type {
  OrganizationSelectorRuntimeConfig,
  OrganizationSelectorType,
} from 'src/types/organization-selector'

const OrganizationSelectStub = defineComponent({
  name: 'OrganizationSelect',
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
  },
  emits: ['update:modelValue'],
  setup() {
    return () => h('div', { 'data-testid': 'organization-select' })
  },
})

const QInputStub = defineComponent({
  name: 'QInput',
  setup() {
    return () => h('input', { 'data-testid': 'q-input' })
  },
})

const QSelectStub = defineComponent({
  name: 'QSelect',
  setup(_, { slots }) {
    return () => h('div', { 'data-testid': 'q-select' }, slots.default?.())
  },
})

const makeRule = (expressionType = ExpressionType.EQ): QueryRule => ({
  field: 'subject_id',
  expression_type: expressionType,
  value: expressionType === ExpressionType.IN ? [] : null,
})

const selectorConfig = (
  selectorType: OrganizationSelectorType,
): OrganizationSelectorRuntimeConfig => ({
  selectorType,
  multiple: false,
  includeHistory: true,
  disabled: false,
})

const mountRuleRow = (
  rule: QueryRule,
  config: OrganizationSelectorRuntimeConfig | null,
  updateOrganizationSelectorValue = vi.fn(),
) =>
  shallowMount(AdvancedQueryRuleRow, {
    props: {
      rule,
      logic: ExpressionLogic.AND,
      isFirst: true,
      canRemove: false,
      fields: [{ field_code: rule.field, field_name: rule.field }],
      fieldLabelKey: 'field_name',
      fieldValueKey: 'field_code',
      expressionLogicOptions: [],
      expressionTypeOptionsForRule: () => [],
      booleanOptions: [],
      organizationSelectorConfigForRule: () => config,
      updateOrganizationSelectorValue,
      isNullOperator: () => false,
      hasDictRule: () => false,
      hasRelationRule: () => false,
      isBooleanRule: () => false,
      isMultiValueRule: (currentRule) =>
        currentRule.expression_type === ExpressionType.IN,
      isFreeInputMultiValueRule: () => false,
      isRangeRule: () => false,
      dictOptionsForRule: () => [],
      relationOptionsForRule: () => [],
      isRelationLoading: () => false,
      hasMoreRelationOptions: () => false,
      valueRules: () => [],
      inputTypeForRule: () => 'text',
      valuePlaceholderForRule: () => '',
      rangePlaceholderForRule: () => '',
      filterRelationOptions: () => undefined,
      preloadRelationOptions: () => undefined,
      loadMoreRelationOptions: () => undefined,
    },
    global: {
      stubs: {
        OrganizationSelect: OrganizationSelectStub,
        QInput: QInputStub,
        QSelect: QSelectStub,
        QBtn: true,
        QItem: true,
        QItemSection: true,
        QItemLabel: true,
        QTooltip: true,
      },
    },
  })

describe('AdvancedQueryRuleRow organization selector integration', () => {
  it.each<OrganizationSelectorType>(['employee', 'position', 'org_unit', 'legal_entity'])(
    'renders the shared %s selector',
    (selectorType) => {
      const wrapper = mountRuleRow(makeRule(), selectorConfig(selectorType))
      const selector = wrapper.findComponent(OrganizationSelectStub)

      expect(selector.exists()).toBe(true)
      expect(selector.props()).toMatchObject({
        selectorType,
        multiple: false,
        includeHistory: true,
      })
    },
  )

  it('uses a single ID for equals and an ID array for in', async () => {
    const equalsRule = makeRule(ExpressionType.EQ)
    const equalsUpdate = vi.fn()
    const equalsWrapper = mountRuleRow(
      equalsRule,
      selectorConfig('employee'),
      equalsUpdate,
    )
    const equalsSelector = equalsWrapper.findComponent(OrganizationSelectStub)

    expect(equalsSelector.props('multiple')).toBe(false)
    equalsSelector.vm.$emit('update:modelValue', 17)
    await nextTick()
    expect(equalsUpdate).toHaveBeenCalledWith(equalsRule, 17)

    const inRule = makeRule(ExpressionType.IN)
    const inUpdate = vi.fn()
    const inWrapper = mountRuleRow(inRule, selectorConfig('employee'), inUpdate)
    const inSelector = inWrapper.findComponent(OrganizationSelectStub)

    expect(inSelector.props('multiple')).toBe(true)
    inSelector.vm.$emit('update:modelValue', [17, 18])
    await nextTick()
    expect(inUpdate).toHaveBeenCalledWith(inRule, [17, 18])
  })

  it('keeps an ordinary field on the existing input path', () => {
    const wrapper = mountRuleRow(makeRule(), null)

    expect(wrapper.findComponent(OrganizationSelectStub).exists()).toBe(false)
    expect(wrapper.find('[data-testid="q-input"]').exists()).toBe(true)
  })
})
