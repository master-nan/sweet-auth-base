import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import type { TableField } from 'src/api/services/sys-table'
import type {
  OrganizationSelectorFieldMetadata,
  OrganizationSelectorRuntimeConfig,
  OrganizationSelectorType,
} from 'src/types/organization-selector'

export type FieldControlType =
  | 'input'
  | 'number'
  | 'textarea'
  | 'select'
  | 'date'
  | 'datetime'
  | 'time'
  | 'year'
  | 'year-month'
  | 'file'
  | 'boolean'
  | 'json-editor'
  | 'array-input'
  | 'key-value-editor'
  | 'cascader'
  | 'rich-text'

export type HtmlInputType =
  | 'number'
  | 'text'
  | 'file'
  | 'textarea'
  | 'time'
  | 'search'
  | 'password'
  | 'date'
  | 'email'
  | 'tel'
  | 'url'
  | 'datetime-local'

export const decodeHtmlEntities = (value: string) => {
  if (!value) return value
  return value
    .replace(/&#34;/g, '"')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&apos;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&amp;/g, '&')
}

export const parseJsonSafe = (text: string) => {
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

export const parseLinkageConfig = (field: Partial<TableField>) => {
  const linkageConfig = field.linkage_config
  if (!linkageConfig) return null
  const raw = typeof linkageConfig === 'string' ? decodeHtmlEntities(linkageConfig) : linkageConfig
  const cfg = typeof raw === 'string' ? parseJsonSafe(raw) : raw
  if (!cfg || !cfg.linkage?.enabled) return null
  return cfg.linkage
}

const organizationSelectorTypeMap: Record<string, OrganizationSelectorType> = {
  legal_entity: 'legal_entity',
  org_unit: 'org_unit',
  employee: 'employee',
  position: 'position',
}

const metadataRecord = (value: unknown): Record<string, unknown> | null => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  return value as Record<string, unknown>
}

const parseMetadataRecord = (value: unknown): Record<string, unknown> | null => {
  if (typeof value !== 'string') return metadataRecord(value)
  const parsed = parseJsonSafe(decodeHtmlEntities(value))
  return metadataRecord(parsed)
}

const normalizeOrganizationSelectorType = (value: unknown): OrganizationSelectorType | null => {
  if (typeof value !== 'string') return null
  return organizationSelectorTypeMap[value.trim().toLowerCase()] || null
}

const hasMetadataValue = (value: unknown) => {
  if (typeof value === 'string') return value.trim() !== ''
  return value !== undefined && value !== null
}

const metadataBoolean = (...values: unknown[]) => {
  for (const value of values) {
    if (typeof value === 'boolean') return value
    if (typeof value === 'number' && (value === 0 || value === 1)) return value === 1
    if (typeof value !== 'string') continue

    const normalized = value.trim().toLowerCase()
    if (normalized === 'true' || normalized === '1') return true
    if (normalized === 'false' || normalized === '0') return false
  }
  return false
}

export const resolveOrganizationSelectorConfig = (
  field: OrganizationSelectorFieldMetadata,
): OrganizationSelectorRuntimeConfig | null => {
  const linkageConfig = parseMetadataRecord(field.linkage_config)
  const nestedSelector = metadataRecord(linkageConfig?.selector)
  const directSelectorConfigured = hasMetadataValue(field.selector_type)
  const directSelectorType = normalizeOrganizationSelectorType(field.selector_type)

  if (directSelectorConfigured && !directSelectorType) return null

  const linkageSelectorType = normalizeOrganizationSelectorType(
    nestedSelector?.selector_type ?? linkageConfig?.selector_type,
  )
  const selectorType = directSelectorType || linkageSelectorType
  if (!selectorType) return null

  return {
    selectorType,
    multiple: metadataBoolean(field.multiple, nestedSelector?.multiple, linkageConfig?.multiple),
    includeHistory: metadataBoolean(
      field.include_history,
      nestedSelector?.include_history,
      linkageConfig?.include_history,
    ),
    disabled: metadataBoolean(field.disabled, nestedSelector?.disabled, linkageConfig?.disabled),
  }
}

export const numericFieldTypes = new Set<SysTableFieldType>([
  SysTableFieldType.BIGINT,
  SysTableFieldType.INT,
  SysTableFieldType.SMALLINT,
  SysTableFieldType.DECIMAL,
])

export const isExactDecimalFieldType = (fieldType?: SysTableFieldType) => {
  return fieldType === SysTableFieldType.DECIMAL
}

export const isNumericFieldType = (fieldType?: SysTableFieldType) => {
  return fieldType !== undefined && numericFieldTypes.has(fieldType)
}

export const isBooleanFieldType = (fieldType?: SysTableFieldType) => {
  return fieldType === SysTableFieldType.BOOLEAN
}

export const isBooleanFieldMetadata = (
  field?: Partial<TableField>,
  fallbackType?: SysTableFieldType,
) => {
  return (
    fallbackType === SysTableFieldType.BOOLEAN ||
    field?.field_type === SysTableFieldType.BOOLEAN ||
    field?.input_type === SysTableFieldInputType.BOOLEAN
  )
}

export const booleanLikeFieldCode = (fieldCode: string) => {
  return (
    fieldCode === 'state' ||
    fieldCode === 'success' ||
    fieldCode === 'enabled' ||
    fieldCode === 'disabled' ||
    fieldCode.startsWith('is_') ||
    fieldCode.startsWith('has_') ||
    fieldCode.startsWith('can_') ||
    fieldCode.startsWith('allow_') ||
    fieldCode.startsWith('enable_') ||
    fieldCode.startsWith('disable_')
  )
}

export const metadataDictDefault = (
  fieldCode: string,
  fieldType?: SysTableFieldType,
  tableCode = '',
) => {
  const code = fieldCode.trim()
  const table = tableCode.trim()
  if (code === 'table_type') return 'sys_table_type'
  if (code === 'field_type') return 'sys_table_field_type'
  if (code === 'input_type') return 'sys_table_field_input_type'
  if (code === 'field_category') return 'sys_table_field_category'
  if (code === 'relation_type') return 'sys_table_relation_type'
  if (code === 'position' && (table === 'sys_menu_button' || table === 'sys_menu_button_template')) {
    return 'sys_menu_button_position'
  }
  if (
    code === 'display_mode' &&
    (table === 'sys_menu_button' || table === 'sys_menu_button_template')
  ) {
    return 'sys_menu_button_display_mode'
  }
  if (
    code === 'event_action' &&
    (table === 'sys_menu_button' || table === 'sys_menu_button_template')
  ) {
    return 'sys_menu_button_event_action'
  }
  if (code === 'method' || code === 'http_method') return 'http_method'
  if (isBooleanFieldType(fieldType) || booleanLikeFieldCode(code)) return 'whether'
  return ''
}

export const defaultInputTypeForFieldType = (fieldType: SysTableFieldType) => {
  switch (fieldType) {
    case SysTableFieldType.BIGINT:
    case SysTableFieldType.INT:
    case SysTableFieldType.SMALLINT:
    case SysTableFieldType.DECIMAL:
      return SysTableFieldInputType.INPUT_NUMBER
    case SysTableFieldType.TEXT:
      return SysTableFieldInputType.TEXTAREA
    case SysTableFieldType.BOOLEAN:
      return SysTableFieldInputType.BOOLEAN
    case SysTableFieldType.DATE:
      return SysTableFieldInputType.DATE_PICKER
    case SysTableFieldType.DATETIME:
      return SysTableFieldInputType.DATETIME_PICKER
    case SysTableFieldType.TIME:
      return SysTableFieldInputType.TIME_PICKER
    case SysTableFieldType.JSON:
      return SysTableFieldInputType.JSON_EDITOR
    default:
      return SysTableFieldInputType.INPUT
  }
}

export const selectLikeInputTypes = new Set<SysTableFieldInputType>([
  SysTableFieldInputType.SELECT,
  SysTableFieldInputType.CASCADER,
])

export const inputTypesAllowingDictionary = new Set<SysTableFieldInputType>([
  SysTableFieldInputType.SELECT,
  SysTableFieldInputType.CASCADER,
  SysTableFieldInputType.BOOLEAN,
])

export const coerceFieldValue = (value: unknown, fieldType?: SysTableFieldType) => {
  if (isExactDecimalFieldType(fieldType)) {
    if (typeof value === 'string') return value.trim()
    if (typeof value === 'number' && Number.isSafeInteger(value)) return `${value}`
    return value
  }
  if (isNumericFieldType(fieldType)) {
    return Number(value)
  }
  if (fieldType === SysTableFieldType.BOOLEAN) {
    const normalized = String(value)
    if (normalized === 'true') return true
    if (normalized === 'false') return false
  }
  return value
}

export const coerceDictOptions = <T extends { value: unknown }>(
  options: T[],
  fieldType?: SysTableFieldType,
) => {
  return options.map((option) => ({
    ...option,
    value: coerceFieldValue(option.value, fieldType),
  }))
}

export const fieldHtmlInputType = (inputType?: SysTableFieldInputType): HtmlInputType => {
  switch (inputType) {
    case SysTableFieldInputType.INPUT_NUMBER:
      return 'number'
    case SysTableFieldInputType.DATE_PICKER:
      return 'date'
    case SysTableFieldInputType.DATETIME_PICKER:
      return 'datetime-local'
    case SysTableFieldInputType.TIME_PICKER:
      return 'time'
    default:
      return 'text'
  }
}

export const queryValueHtmlInputType = (field?: Partial<TableField>): HtmlInputType => {
  switch (field?.field_type) {
    case SysTableFieldType.BIGINT:
    case SysTableFieldType.INT:
    case SysTableFieldType.SMALLINT:
    case SysTableFieldType.DECIMAL:
      return 'number'
    case SysTableFieldType.DATE:
      return 'date'
    case SysTableFieldType.DATETIME:
      return 'datetime-local'
    case SysTableFieldType.TIME:
      return 'time'
    default:
      return fieldHtmlInputType(field?.input_type)
  }
}

export const getFieldControlType = (field: TableField): FieldControlType => {
  // File metadata is stored as JSON, but its explicit widget must win over the storage type.
  if (field.input_type === SysTableFieldInputType.FILE_PICKER) {
    return 'file'
  }

  if (field.field_type === SysTableFieldType.JSON) {
    switch (field.input_type) {
      case SysTableFieldInputType.ARRAY_INPUT:
        return 'array-input'
      case SysTableFieldInputType.KEY_VALUE_EDITOR:
        return 'key-value-editor'
      default:
        return 'json-editor'
    }
  }

  if (field.field_type === SysTableFieldType.BOOLEAN) {
    return 'boolean'
  }

  const linkage = parseLinkageConfig(field)
  if (linkage) {
    return linkage.mode === 'cascader' ? 'cascader' : 'select'
  }

  switch (field.input_type) {
    case SysTableFieldInputType.INPUT:
      return 'input'
    case SysTableFieldInputType.INPUT_NUMBER:
      return 'number'
    case SysTableFieldInputType.TEXTAREA:
      return 'textarea'
    case SysTableFieldInputType.SELECT:
      return 'select'
    case SysTableFieldInputType.DATE_PICKER:
      return 'date'
    case SysTableFieldInputType.DATETIME_PICKER:
      return 'datetime'
    case SysTableFieldInputType.TIME_PICKER:
      return 'time'
    case SysTableFieldInputType.YEAR_PICKER:
      return 'year'
    case SysTableFieldInputType.YEAR_MONTH_PICKER:
      return 'year-month'
    case SysTableFieldInputType.BOOLEAN:
      return 'boolean'
    case SysTableFieldInputType.JSON_EDITOR:
      return 'json-editor'
    case SysTableFieldInputType.ARRAY_INPUT:
      return 'array-input'
    case SysTableFieldInputType.KEY_VALUE_EDITOR:
      return 'key-value-editor'
    case SysTableFieldInputType.CASCADER:
      return 'cascader'
    case SysTableFieldInputType.RICH_TEXT:
      return 'rich-text'
    default:
      return getFieldControlTypeByFieldType(field.field_type)
  }
}

export const getFieldControlTypeByFieldType = (fieldType: SysTableFieldType): FieldControlType => {
  switch (fieldType) {
    case SysTableFieldType.BIGINT:
    case SysTableFieldType.INT:
    case SysTableFieldType.SMALLINT:
    case SysTableFieldType.DECIMAL:
      return 'number'
    case SysTableFieldType.TEXT:
      return 'textarea'
    case SysTableFieldType.BOOLEAN:
      return 'boolean'
    case SysTableFieldType.DATE:
      return 'date'
    case SysTableFieldType.DATETIME:
      return 'datetime'
    case SysTableFieldType.TIME:
      return 'time'
    case SysTableFieldType.JSON:
      return 'json-editor'
    default:
      return 'input'
  }
}

const configuredDefaultValueForField = (field: TableField) => {
  const value = (field as any).default_value
  if (value === undefined || value === null || value === '') return undefined

  if (
    field.field_type === SysTableFieldType.BOOLEAN ||
    field.input_type === SysTableFieldInputType.BOOLEAN
  ) {
    if (typeof value === 'boolean') return value
    const normalized = String(value).trim().toLowerCase()
    return normalized === 'true' || normalized === '1' || normalized === '是' || normalized === 'yes'
  }

  if (isExactDecimalFieldType(field.field_type)) {
    return typeof value === 'string' ? value.trim() : value
  }

  if (isNumericFieldType(field.field_type)) {
    const numeric = Number(value)
    return Number.isFinite(numeric) ? numeric : value
  }

  if (
    field.input_type === SysTableFieldInputType.JSON_EDITOR ||
    field.input_type === SysTableFieldInputType.ARRAY_INPUT ||
    field.input_type === SysTableFieldInputType.KEY_VALUE_EDITOR
  ) {
    if (typeof value !== 'string') return value
    try {
      return JSON.parse(value)
    } catch {
      return value
    }
  }

  return value
}

export const compareExactDecimal = (left: unknown, right: unknown): number | null => {
  const normalize = (value: unknown) => {
    if (typeof value !== 'string' && typeof value !== 'number') return null
    const match = String(value)
      .trim()
      .match(/^([+-]?)(\d+)(?:\.(\d+))?$/)
    if (!match) return null
    const integer = match[2]!.replace(/^0+/, '') || '0'
    const fraction = (match[3] || '').replace(/0+$/, '')
    const negative = match[1] === '-' && (integer !== '0' || fraction !== '')
    return { negative, integer, fraction }
  }
  const leftValue = normalize(left)
  const rightValue = normalize(right)
  if (!leftValue || !rightValue) return null
  if (leftValue.negative !== rightValue.negative) return leftValue.negative ? -1 : 1

  let result = leftValue.integer.length - rightValue.integer.length
  if (result === 0 && leftValue.integer !== rightValue.integer) {
    result = leftValue.integer < rightValue.integer ? -1 : 1
  }
  if (result === 0) {
    const width = Math.max(leftValue.fraction.length, rightValue.fraction.length)
    const leftFraction = leftValue.fraction.padEnd(width, '0')
    const rightFraction = rightValue.fraction.padEnd(width, '0')
    if (leftFraction !== rightFraction) result = leftFraction < rightFraction ? -1 : 1
  }
  const normalized = result === 0 ? 0 : result < 0 ? -1 : 1
  return leftValue.negative ? -normalized : normalized
}

export const defaultValueForField = (field: TableField) => {
  const configuredDefault = configuredDefaultValueForField(field)
  if (configuredDefault !== undefined) return configuredDefault

  const controlType = getFieldControlType(field)
  if (controlType === 'select' || controlType === 'cascader') return null
  switch (field.field_type) {
    case SysTableFieldType.BIGINT:
    case SysTableFieldType.INT:
    case SysTableFieldType.SMALLINT:
      return 0
    case SysTableFieldType.DECIMAL:
      return '0'
    case SysTableFieldType.BOOLEAN:
      return false
    default:
      return ''
  }
}
