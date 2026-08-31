import { translate as t } from 'src/boot/i18n'
type SelectLikeOption = {
  label?: unknown
  value?: unknown
  name?: unknown
  field?: unknown
  [key: string]: unknown
}

const normalizeArray = (value: unknown): unknown[] => {
  if (Array.isArray(value)) return value
  return value === null || value === undefined || value === '' ? [] : [value]
}

const optionValue = (option: unknown) => {
  const item = option as SelectLikeOption | null
  return item?.value ?? item?.name ?? option
}

const optionLabel = (option: unknown) => {
  const item = option as SelectLikeOption | null
  return toDisplayText(item?.label ?? item?.field ?? item?.name ?? item?.value ?? option)
}

const toDisplayText = (value: unknown) => {
  if (value === null || value === undefined) return ''
  switch (typeof value) {
    case 'string':
      return value
    case 'number':
    case 'boolean':
    case 'bigint':
      return String(value)
    default:
      return ''
  }
}

const isSameValue = (left: unknown, right: unknown) => {
  return toDisplayText(left) === toDisplayText(right)
}

export const compactSelectionDisplay = (
  value: unknown,
  options: unknown[] = [],
  maxVisible = 2,
  emptyText = '',
) => {
  const values = normalizeArray(value)
  if (values.length === 0) return emptyText

  const labels = values.map((selectedValue) => {
    const option = options.find((item) => isSameValue(optionValue(item), selectedValue))
    return option ? optionLabel(option) : toDisplayText(selectedValue)
  })

  const visibleLabels = labels.slice(0, maxVisible).join('、')
  return labels.length > maxVisible
    ? t('ui.visibleLabelsAndTotal', { visibleLabels: visibleLabels, value2: labels.length })
    : visibleLabels
}

export const compactSelectionTooltip = (value: unknown, options: unknown[] = []) => {
  return normalizeArray(value)
    .map((selectedValue) => {
      const option = options.find((item) => isSameValue(optionValue(item), selectedValue))
      return option ? optionLabel(option) : toDisplayText(selectedValue)
    })
    .join('、')
}
