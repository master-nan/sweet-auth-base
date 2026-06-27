import { ExpressionType, SysTableFieldType } from 'src/types/enum'
import type { ExpressionGroup, Query, QueryRule } from 'src/types/global'

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
  if (fieldType === SysTableFieldType.FLOAT) {
    return normalizeNumericQueryValue(value, false)
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
