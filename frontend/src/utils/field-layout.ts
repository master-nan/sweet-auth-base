import type { TableField } from 'src/api/services/sys-table'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'

const normalizeSpan = (value: unknown, max: number) => {
  const num = Number(value)
  if (!Number.isFinite(num) || num <= 0) return 0
  return Math.min(Math.max(Math.trunc(num), 1), max)
}

export const normalizeFieldLabel = (value: unknown) => {
  let text = ''
  if (typeof value === 'string') {
    text = value.trim()
  } else if (typeof value === 'number' || typeof value === 'boolean') {
    text = String(value).trim()
  }
  if (text.length < 2) return text
  const first = text[0]
  const last = text[text.length - 1]
  if ((first === "'" || first === '"' || first === '`') && first === last) {
    const inner = text.slice(1, -1).trim()
    return inner || text
  }
  return text
}

export const getFieldFormSpan = (field: TableField) => {
  const configured = normalizeSpan(field.form_span, 2)
  if (configured > 0) return configured

  switch (field.input_type) {
    case SysTableFieldInputType.TEXTAREA:
    case SysTableFieldInputType.FILE_PICKER:
    case SysTableFieldInputType.JSON_EDITOR:
    case SysTableFieldInputType.ARRAY_INPUT:
    case SysTableFieldInputType.KEY_VALUE_EDITOR:
    case SysTableFieldInputType.CASCADER:
    case SysTableFieldInputType.RICH_TEXT:
      return 2
    default:
      return 1
  }
}

export const getFieldDetailSpan = (field: TableField) => {
  const configured = normalizeSpan(field.detail_span, 4)
  if (configured > 0) return configured

  switch (field.input_type) {
    case SysTableFieldInputType.RICH_TEXT:
    case SysTableFieldInputType.JSON_EDITOR:
    case SysTableFieldInputType.ARRAY_INPUT:
    case SysTableFieldInputType.KEY_VALUE_EDITOR:
    case SysTableFieldInputType.TEXTAREA:
      return 4
    case SysTableFieldInputType.FILE_PICKER:
      return 2
    default:
      if (field.field_type === SysTableFieldType.TEXT || field.field_type === SysTableFieldType.JSON) {
        return 4
      }
      return 1
  }
}

export const getFieldFormGridClass = (field: TableField) =>
  getFieldFormSpan(field) >= 2 ? 'col-12' : 'col-12 col-sm-6'
