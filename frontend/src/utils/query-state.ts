import { ExpressionType, SysTableFieldType } from 'src/types/enum'
import type { ExpressionGroup, Query, QueryRule } from 'src/types/global'
import { ExpressionLogic } from 'src/types/enum'
import type {
  QuerySchemeBinding,
  QuerySchemePayloadV1,
} from 'src/modules/query-scheme/types'

const nullExpressionTypes = new Set<ExpressionType>([
  ExpressionType.IS_NULL,
  ExpressionType.IS_NOT_NULL,
])
const multiValueExpressionTypes = new Set<ExpressionType>([
  ExpressionType.IN,
  ExpressionType.NOT_IN,
])
const textMultiKeywordExpressionTypes = new Set<ExpressionType>([
  ExpressionType.LIKE,
  ExpressionType.NOT_LIKE,
])
const rangeExpressionTypes = new Set<ExpressionType>([
  ExpressionType.BETWEEN,
  ExpressionType.NOT_BETWEEN,
])

export const isNullExpressionType = (expressionType?: ExpressionType) => {
  return expressionType !== undefined && nullExpressionTypes.has(expressionType)
}

export const isMultiValueExpressionType = (expressionType?: ExpressionType) => {
  return expressionType !== undefined && multiValueExpressionTypes.has(expressionType)
}

export const isTextMultiKeywordExpressionType = (expressionType?: ExpressionType) => {
  return expressionType !== undefined && textMultiKeywordExpressionTypes.has(expressionType)
}

export const isRangeExpressionType = (expressionType?: ExpressionType) => {
  return expressionType !== undefined && rangeExpressionTypes.has(expressionType)
}

export const hasQueryRuleValue = (value: unknown) => {
  if (Array.isArray(value)) return value.length > 0
  return value !== null && value !== undefined && value !== ''
}

export const hasQueryRuleRangeValue = (value: unknown) => {
  if (!Array.isArray(value) || value.length !== 2) return false
  return hasQueryRuleValue(value[0]) && hasQueryRuleValue(value[1])
}

export const isBlankQueryRule = (rule: QueryRule) => {
  return !rule.field && rule.expression_type === undefined && !hasQueryRuleValue(rule.value)
}

export const isEffectiveQueryRule = (rule: QueryRule) => {
  if (!rule.field) return false
  if (isNullExpressionType(rule.expression_type)) return true
  if (isRangeExpressionType(rule.expression_type)) return hasQueryRuleRangeValue(rule.value)
  return hasQueryRuleValue(rule.value)
}

export const isIncompleteQueryRule = (rule: QueryRule) => {
  if (isBlankQueryRule(rule)) return false
  if (!rule.field || rule.expression_type === undefined) return true
  if (isNullExpressionType(rule.expression_type)) return false
  if (isRangeExpressionType(rule.expression_type)) return !hasQueryRuleRangeValue(rule.value)
  return !hasQueryRuleValue(rule.value)
}

const countExpressionRules = (expression: ExpressionGroup): number => {
  const ruleCount = expression.rules.filter(isEffectiveQueryRule).length
  const nestedCount =
    expression.nested?.reduce((total, nested) => total + countExpressionRules(nested), 0) ?? 0
  return ruleCount + nestedCount
}

export const countEffectiveQueryRules = (query: Pick<Query, 'expressions'>) => {
  return query.expressions.reduce(
    (total, expression) => total + countExpressionRules(expression),
    0,
  )
}

export const hasEffectiveQueryRules = (query: Pick<Query, 'expressions'>) => {
  return countEffectiveQueryRules(query) > 0
}

export const splitMultiValueText = (value: string) => {
  return value
    .split(/[\n,，;；]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export type QueryRuleFieldTypeResolver = (rule: QueryRule) => SysTableFieldType | undefined

const integerFieldTypes = new Set<SysTableFieldType>([
  SysTableFieldType.BIGINT,
  SysTableFieldType.INT,
  SysTableFieldType.TINYINT,
  SysTableFieldType.SMALLINT,
])

const normalizeNumericQueryValue = (value: unknown, requireInteger: boolean) => {
  if (typeof value === 'number') {
    if (!Number.isFinite(value)) return value
    if (requireInteger && !Number.isInteger(value)) return value
    return value
  }
  if (typeof value !== 'string') return value
  const raw = value.trim()
  if (raw === '') return value
  const nextValue = Number(raw)
  if (!Number.isFinite(nextValue)) return value
  if (requireInteger && !Number.isInteger(nextValue)) return value
  return nextValue
}

const normalizeBooleanQueryValue = (value: unknown) => {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') {
    if (value === 1) return true
    if (value === 0) return false
    return value
  }
  if (typeof value !== 'string') return value
  switch (value.trim().toLowerCase()) {
    case 'true':
    case '1':
    case '是':
      return true
    case 'false':
    case '0':
    case '否':
      return false
    default:
      return value
  }
}

export const normalizeQueryValueByFieldType = (
  value: unknown,
  fieldType?: SysTableFieldType,
) => {
  if (value === null || value === undefined || value === '') return value
  if (fieldType === SysTableFieldType.FLOAT || fieldType === SysTableFieldType.DECIMAL) {
    if (typeof value === 'string') return value.trim()
    if (typeof value === 'number' && Number.isSafeInteger(value)) return `${value}`
    return value
  }
  if (fieldType !== undefined && integerFieldTypes.has(fieldType)) {
    return normalizeNumericQueryValue(value, true)
  }
  if (fieldType === SysTableFieldType.BOOLEAN) {
    return normalizeBooleanQueryValue(value)
  }
  return typeof value === 'string' ? value.trim() : value
}

export const normalizeQueryRuleValue = (
  rule: QueryRule,
  resolveFieldType?: QueryRuleFieldTypeResolver,
) => {
  const fieldType = resolveFieldType?.(rule) ?? rule.type
  const normalizeItem = (value: unknown) => normalizeQueryValueByFieldType(value, fieldType)
  if (isNullExpressionType(rule.expression_type)) return null
  if (isRangeExpressionType(rule.expression_type)) {
    if (!Array.isArray(rule.value)) return []
    return rule.value
      .slice(0, 2)
      .map(normalizeItem)
      .filter((item) => hasQueryRuleValue(item))
  }
  if (isTextMultiKeywordExpressionType(rule.expression_type) && typeof rule.value === 'string') {
    const values = splitMultiValueText(rule.value).map(normalizeItem).filter(hasQueryRuleValue)
    return values.length > 1 ? values : (values[0] ?? '')
  }
  if (!isMultiValueExpressionType(rule.expression_type)) return normalizeItem(rule.value)
  if (Array.isArray(rule.value)) {
    return rule.value
      .map(normalizeItem)
      .filter((item) => hasQueryRuleValue(item))
  }
  if (typeof rule.value === 'string') return splitMultiValueText(rule.value).map(normalizeItem)
  if (!hasQueryRuleValue(rule.value)) return []
  return [normalizeItem(rule.value)]
}

export const normalizeQueryRuleForSubmit = (
  rule: QueryRule,
  resolveFieldType?: QueryRuleFieldTypeResolver,
): QueryRule => {
  const fieldType = resolveFieldType?.(rule) ?? rule.type
  const nextRule: QueryRule = {
    ...rule,
    value: normalizeQueryRuleValue(rule, resolveFieldType),
  }
  if (fieldType !== undefined) {
    nextRule.type = fieldType
  } else {
    delete nextRule.type
  }
  return nextRule
}

const sanitizeExpressionGroup = (
  expression: ExpressionGroup,
  resolveFieldType?: QueryRuleFieldTypeResolver,
): ExpressionGroup | null => {
  const rules = expression.rules
    .filter(isEffectiveQueryRule)
    .map((rule) => normalizeQueryRuleForSubmit(rule, resolveFieldType))
    .filter(isEffectiveQueryRule)
  const nested =
    expression.nested
      ?.map((item) => sanitizeExpressionGroup(item, resolveFieldType))
      .filter((item): item is ExpressionGroup => !!item) ?? []

  if (rules.length === 0 && nested.length === 0) return null
  return {
    ...expression,
    rules,
    nested,
  }
}

export const sanitizeQueryExpressions = (
  expressions: ExpressionGroup[],
  resolveFieldType?: QueryRuleFieldTypeResolver,
) => {
  return expressions
    .map((expression) => sanitizeExpressionGroup(expression, resolveFieldType))
    .filter((expression): expression is ExpressionGroup => !!expression)
}

const normalizeExpressionLogic = (groups: ExpressionGroup[]): ExpressionGroup[] =>
  groups.map((group) => ({
    ...group,
    logic: group.logic ?? ExpressionLogic.AND,
    rules: group.rules.map((rule) => ({ ...rule })),
    nested: normalizeExpressionLogic(group.nested || []),
  }))

const stableValue = (value: unknown): unknown => {
  if (Array.isArray(value)) return value.map(stableValue)
  if (!value || typeof value !== 'object') return value
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, item]) => [key, stableValue(item)]),
  )
}

export const normalizeQuerySchemePayload = (
  query: Pick<Query, 'expressions' | 'quick_query' | 'order'>,
  bindings: QuerySchemeBinding[] = [],
): QuerySchemePayloadV1 => ({
  expressions: normalizeExpressionLogic(
    query.expressions
      .map((group, index) => sanitizeSchemeGroup(group, `/expressions/${index}`, bindings))
      .filter((group): group is ExpressionGroup => !!group),
  ),
  quick_query: { keyword: query.quick_query?.keyword?.trim() || '' },
  order: {
    field: query.order?.field?.trim() || '',
    is_asc: !!query.order?.field && !!query.order?.is_asc,
  },
  bindings: bindings
    .map((binding) => ({
      pointer: binding.pointer.trim(),
      kind: binding.kind,
      ...(binding.params && Object.keys(binding.params).length
        ? { params: { ...binding.params } }
        : {}),
    }))
    .sort((left, right) => left.pointer.localeCompare(right.pointer)),
})

const sanitizeSchemeGroup = (
  group: ExpressionGroup,
  path: string,
  bindings: QuerySchemeBinding[],
): ExpressionGroup | null => {
  const rules = group.rules
    .map((rule, index) => {
      const valuePointer = `${path}/rules/${index}/value`
      const bound = bindings.some(
        (binding) =>
          binding.pointer === valuePointer || binding.pointer.startsWith(`${valuePointer}/`),
      )
      if (!bound && !isEffectiveQueryRule(rule)) return null
      return normalizeQueryRuleForSubmit(rule)
    })
    .filter((rule): rule is QueryRule => !!rule)
  const nested = (group.nested || [])
    .map((child, index) => sanitizeSchemeGroup(child, `${path}/nested/${index}`, bindings))
    .filter((child): child is ExpressionGroup => !!child)
  if (!rules.length && !nested.length) return null
  return { ...group, rules, nested }
}

export const serializeQuerySchemePayload = (payload: QuerySchemePayloadV1) =>
  JSON.stringify(stableValue(payload))

export const queryExpressionDepth = (groups: ExpressionGroup[]): number => {
  const groupDepth = (group: ExpressionGroup): number =>
    1 + Math.max(0, ...(group.nested || []).map(groupDepth))
  return Math.max(0, ...groups.map(groupDepth))
}

export const isSimpleQueryExpression = (groups: ExpressionGroup[]) =>
  groups.length <= 1 &&
  (groups[0]?.logic ?? ExpressionLogic.AND) === ExpressionLogic.AND &&
  (groups[0]?.nested?.length || 0) === 0
