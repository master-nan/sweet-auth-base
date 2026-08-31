import { translate as t } from 'src/i18n/runtime/instance'
export const QUERY_SCHEME_NAME_MAX_CODE_POINTS = 64

export const querySchemeNameLength = (value: string) => Array.from(value).length

export const truncateQuerySchemeName = (
  value: string,
  maxLength = QUERY_SCHEME_NAME_MAX_CODE_POINTS,
) => Array.from(value).slice(0, Math.max(0, maxLength)).join('')

export const normalizeQuerySchemeName = (value: string) => value.trim()

export const isValidQuerySchemeName = (value: string) => {
  const normalized = normalizeQuerySchemeName(value)
  const length = querySchemeNameLength(normalized)
  return length > 0 && length <= QUERY_SCHEME_NAME_MAX_CODE_POINTS
}

export const buildQuerySchemeCopyName = (value: string) => {
  const suffix = t('ui.querySchemeCopySuffix')
  const baseLength = QUERY_SCHEME_NAME_MAX_CODE_POINTS - querySchemeNameLength(suffix)
  return `${truncateQuerySchemeName(normalizeQuerySchemeName(value), baseLength)}${suffix}`
}
