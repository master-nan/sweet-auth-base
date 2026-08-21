import type { TableField } from 'src/api/services/sys-table'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { primitiveText } from 'src/utils/primitive-text'

type ParamOption = {
  label: string
  value: unknown
}

type ParamDataSourceSchema =
  | string
  | {
      type?: string
      mode?: string
      dict_code?: string
      dictCode?: string
      table_code?: string
      tableCode?: string
      label_field?: string
      labelField?: string
      label_key?: string
      labelKey?: string
      value_field?: string
      valueField?: string
      value_key?: string
      valueKey?: string
      parent_field?: string
      parentField?: string
      parent_key?: string
      parentKey?: string
      page_size?: number
      pageSize?: number
      target_menu_id?: number
      targetMenuId?: number
      filter_mapping?: Record<string, string>
      filterMapping?: Record<string, string>
      selectable?: 'leaf' | 'any' | 'level'
      show_path?: boolean
      showPath?: boolean
      root_value?: unknown
      rootValue?: unknown
    }

type ParamFieldSchema = {
  field_code?: string
  code?: string
  name?: string
  field_name?: string
  label?: string
  required?: boolean
  field_type?: SysTableFieldType | string | number
  input_type?: SysTableFieldInputType | string | number
  default_value?: unknown
  defaultValue?: unknown
  numeric_precision?: number
  numericPrecision?: number
  numeric_scale?: number
  numericScale?: number
  placeholder?: string
  options?: Array<ParamOption | string | number | boolean>
  dict_code?: string
  dictCode?: string
  linkage_config?: unknown
  linkageConfig?: unknown
  relation?: ParamDataSourceSchema
  cascader?: ParamDataSourceSchema
  data_source?: ParamDataSourceSchema
  dataSource?: ParamDataSourceSchema
  binding?: string
  form_span?: number
  formSpan?: number
  detail_span?: number
  detailSpan?: number
}

type ParamDataSourceResult = {
  dictCode: string
  linkageConfig: string
  linkageMode: string | undefined
}

const inputTypeMap: Record<string, SysTableFieldInputType> = {
  input: SysTableFieldInputType.INPUT,
  text: SysTableFieldInputType.INPUT,
  textarea: SysTableFieldInputType.TEXTAREA,
  number: SysTableFieldInputType.INPUT_NUMBER,
  input_number: SysTableFieldInputType.INPUT_NUMBER,
  select: SysTableFieldInputType.SELECT,
  date: SysTableFieldInputType.DATE_PICKER,
  datetime: SysTableFieldInputType.DATETIME_PICKER,
  time: SysTableFieldInputType.TIME_PICKER,
  year: SysTableFieldInputType.YEAR_PICKER,
  year_month: SysTableFieldInputType.YEAR_MONTH_PICKER,
  file: SysTableFieldInputType.FILE_PICKER,
  file_picker: SysTableFieldInputType.FILE_PICKER,
  boolean: SysTableFieldInputType.BOOLEAN,
  bool: SysTableFieldInputType.BOOLEAN,
  json: SysTableFieldInputType.JSON_EDITOR,
  json_editor: SysTableFieldInputType.JSON_EDITOR,
  array: SysTableFieldInputType.ARRAY_INPUT,
  array_input: SysTableFieldInputType.ARRAY_INPUT,
  key_value: SysTableFieldInputType.KEY_VALUE_EDITOR,
  key_value_editor: SysTableFieldInputType.KEY_VALUE_EDITOR,
  cascader: SysTableFieldInputType.CASCADER,
  rich_text: SysTableFieldInputType.RICH_TEXT,
  richtext: SysTableFieldInputType.RICH_TEXT,
}

const fieldTypeMap: Record<string, SysTableFieldType> = {
  bigint: SysTableFieldType.BIGINT,
  integer: SysTableFieldType.INT,
  int: SysTableFieldType.INT,
  number: SysTableFieldType.DECIMAL,
  smallint: SysTableFieldType.SMALLINT,
  decimal: SysTableFieldType.DECIMAL,
  numeric: SysTableFieldType.DECIMAL,
  varchar: SysTableFieldType.VARCHAR,
  string: SysTableFieldType.VARCHAR,
  text: SysTableFieldType.TEXT,
  textarea: SysTableFieldType.TEXT,
  boolean: SysTableFieldType.BOOLEAN,
  bool: SysTableFieldType.BOOLEAN,
  date: SysTableFieldType.DATE,
  datetime: SysTableFieldType.DATETIME,
  time: SysTableFieldType.TIME,
  json: SysTableFieldType.JSON,
  object: SysTableFieldType.JSON,
  array: SysTableFieldType.JSON,
}

const normalizeKey = (value: string) => value.trim().toLowerCase().replace(/-/g, '_')

const stringifyOptionLabel = (value: unknown) => {
  if (value === undefined || value === null) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return JSON.stringify(value)
}

const normalizeOption = (option: ParamOption | string | number | boolean): ParamOption => {
  if (typeof option !== 'object' || option === null) {
    return {
      label: String(option),
      value: option,
    }
  }
  return {
    label: option.label ?? stringifyOptionLabel(option.value),
    value: option.value,
  }
}

const normalizeOptions = (options?: ParamFieldSchema['options']) => {
  if (!Array.isArray(options)) return undefined
  return options.map(normalizeOption)
}

const toEnumNumber = <T extends number>(value: unknown, allowedValues: T[]) => {
  if (typeof value !== 'number' && typeof value !== 'string') return undefined
  const num = Number(value)
  if (!Number.isFinite(num)) return undefined
  return allowedValues.includes(num as T) ? (num as T) : undefined
}

const resolveInputTypeValue = (value: ParamFieldSchema['input_type']) => {
  const numeric = toEnumNumber(
    value,
    Object.values(SysTableFieldInputType).filter(Number.isInteger) as SysTableFieldInputType[],
  )
  if (numeric) return numeric
  if (typeof value === 'string') {
    return inputTypeMap[normalizeKey(value)]
  }
  return undefined
}

const resolveExplicitFieldType = (value: ParamFieldSchema['field_type']) => {
  const numeric = toEnumNumber(
    value,
    Object.values(SysTableFieldType).filter(Number.isInteger) as SysTableFieldType[],
  )
  if (numeric) return numeric
  if (typeof value === 'string') {
    return fieldTypeMap[normalizeKey(value)]
  }
  return undefined
}

const inferFieldTypeByInputType = (
  inputType: ParamFieldSchema['input_type'],
  hasOptions: boolean,
  hasDict: boolean,
  linkageMode?: string,
) => {
  const resolvedInputType = resolveInputTypeValue(inputType)
  if (hasOptions || hasDict || linkageMode) return SysTableFieldType.VARCHAR

  switch (resolvedInputType) {
    case SysTableFieldInputType.INPUT_NUMBER:
      return SysTableFieldType.BIGINT
    case SysTableFieldInputType.TEXTAREA:
    case SysTableFieldInputType.RICH_TEXT:
    case SysTableFieldInputType.FILE_PICKER:
      return SysTableFieldType.TEXT
    case SysTableFieldInputType.BOOLEAN:
      return SysTableFieldType.BOOLEAN
    case SysTableFieldInputType.DATE_PICKER:
    case SysTableFieldInputType.YEAR_PICKER:
    case SysTableFieldInputType.YEAR_MONTH_PICKER:
      return SysTableFieldType.DATE
    case SysTableFieldInputType.DATETIME_PICKER:
      return SysTableFieldType.DATETIME
    case SysTableFieldInputType.TIME_PICKER:
      return SysTableFieldType.TIME
    case SysTableFieldInputType.JSON_EDITOR:
    case SysTableFieldInputType.ARRAY_INPUT:
    case SysTableFieldInputType.KEY_VALUE_EDITOR:
      return SysTableFieldType.JSON
    default:
      return SysTableFieldType.VARCHAR
  }
}

const resolveFieldType = (
  value: ParamFieldSchema['field_type'],
  inputType: ParamFieldSchema['input_type'],
  hasOptions: boolean,
  hasDict: boolean,
  linkageMode?: string,
) => {
  const explicit = resolveExplicitFieldType(value)
  if (explicit) return explicit
  if (typeof value === 'string' && value.trim()) return SysTableFieldType.VARCHAR
  if (hasOptions || hasDict || linkageMode) return SysTableFieldType.VARCHAR
  if (inputType !== undefined && inputType !== null && inputType !== '') {
    return inferFieldTypeByInputType(inputType, hasOptions, hasDict, linkageMode)
  }
  return SysTableFieldType.VARCHAR
}

const resolveInputType = (
  value: ParamFieldSchema['input_type'],
  fieldType: SysTableFieldType,
  hasOptions: boolean,
  hasDict: boolean,
  linkageMode?: string,
) => {
  if (linkageMode === 'cascader') return SysTableFieldInputType.CASCADER
  const explicitInputType = resolveInputTypeValue(value)
  if (explicitInputType) return explicitInputType
  if (hasOptions || hasDict || linkageMode === 'relation') return SysTableFieldInputType.SELECT
  switch (fieldType) {
    case SysTableFieldType.INT:
    case SysTableFieldType.BIGINT:
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

const readString = (obj: any, keys: string[]) => {
  for (const key of keys) {
    const value = obj?.[key]
    if (value !== undefined && value !== null && String(value).trim()) {
      return String(value).trim()
    }
  }
  return ''
}

const readNumber = (obj: any, keys: string[]) => {
  for (const key of keys) {
    const value = Number(obj?.[key])
    if (Number.isFinite(value) && value > 0) return value
  }
  return undefined
}

const readObject = (obj: any, keys: string[]) => {
  for (const key of keys) {
    const value = obj?.[key]
    if (value && typeof value === 'object' && !Array.isArray(value)) return value
  }
  return undefined
}

const parseJsonObject = (value: unknown) => {
  if (!value) return null
  if (typeof value === 'object' && !Array.isArray(value)) return value as Record<string, any>
  if (typeof value !== 'string') return null
  try {
    const parsed = JSON.parse(value)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? (parsed as Record<string, any>)
      : null
  } catch {
    return null
  }
}

const normalizeDataSource = (item: ParamFieldSchema): ParamDataSourceResult => {
  const result = {
    dictCode: item.dict_code || item.dictCode || '',
    linkageConfig: '',
    linkageMode: undefined as string | undefined,
  }

  const existingLinkage = item.linkage_config ?? item.linkageConfig
  const existingLinkageObj = parseJsonObject(existingLinkage)
  if (existingLinkageObj?.linkage?.enabled) {
    result.linkageConfig =
      typeof existingLinkage === 'string' ? existingLinkage : JSON.stringify(existingLinkageObj)
    result.linkageMode = existingLinkageObj.linkage.mode
    return result
  }

  const source = item.cascader || item.relation || item.data_source || item.dataSource
  if (!source) return result

  if (typeof source === 'string') {
    if (source.startsWith('dict:')) {
      result.dictCode = source.slice(5)
    }
    return result
  }

  const sourceType = normalizeKey(
    String(source.type || source.mode || (item.cascader ? 'cascader' : 'relation')),
  )
  if (sourceType === 'dict' || sourceType === 'dictionary') {
    result.dictCode = readString(source, ['dict_code', 'dictCode', 'code'])
    return result
  }

  const tableCode = readString(source, ['table_code', 'tableCode', 'table'])
  if (!tableCode) return result

  const linkageMode = sourceType === 'cascader' || item.cascader ? 'cascader' : 'relation'
  const linkage: Record<string, unknown> = {
    enabled: true,
    mode: linkageMode,
    tableCode,
    labelKey: readString(source, ['label_field', 'labelField', 'label_key', 'labelKey']) || 'name',
    valueKey: readString(source, ['value_field', 'valueField', 'value_key', 'valueKey']) || 'id',
  }

  const pageSize = readNumber(source, ['page_size', 'pageSize'])
  if (pageSize) linkage.pageSize = pageSize
  const targetMenuId = readNumber(source, ['target_menu_id', 'targetMenuId'])
  if (targetMenuId) linkage.targetMenuId = targetMenuId
  const filterMapping = readObject(source, ['filter_mapping', 'filterMapping'])
  if (filterMapping) linkage.filterMapping = filterMapping

  if (linkageMode === 'cascader') {
    linkage.parentKey =
      readString(source, ['parent_field', 'parentField', 'parent_key', 'parentKey']) || 'parent_id'
    linkage.selectable = source.selectable || 'any'
    linkage.showPath = source.show_path ?? source.showPath ?? true
    if (source.root_value !== undefined || source.rootValue !== undefined) {
      linkage.rootValue = source.root_value ?? source.rootValue
    }
  }

  result.linkageConfig = JSON.stringify({ linkage })
  result.linkageMode = linkageMode
  return result
}

const buildParamField = (item: ParamFieldSchema): TableField | null => {
  const code = item.field_code || item.code || item.name
  if (!code) return null

  const options = normalizeOptions(item.options)
  const dataSource = normalizeDataSource(item)
  const fieldType = resolveFieldType(
    item.field_type,
    item.input_type,
    !!options?.length,
    !!dataSource.dictCode,
    dataSource.linkageMode,
  )
  const inputType = resolveInputType(
    item.input_type,
    fieldType,
    !!options?.length,
    !!dataSource.dictCode,
    dataSource.linkageMode,
  )
  const isDecimal = fieldType === SysTableFieldType.DECIMAL
  const defaultValue = item.default_value ?? item.defaultValue ?? ''
  const field: TableField = {
    id: 0,
    table_id: 0,
    field_name: item.field_name || item.label || code,
    field_code: code,
    field_type: fieldType,
    field_length: 255,
    field_decimal_length: 0,
    input_type: inputType,
    numeric_precision: isDecimal ? (item.numeric_precision ?? item.numericPrecision ?? 38) : 0,
    numeric_scale: isDecimal ? (item.numeric_scale ?? item.numericScale ?? 18) : 0,
    default_value: isDecimal ? primitiveText(defaultValue) : (defaultValue as string),
    dict_code: dataSource.dictCode,
    is_primary_key: false,
    is_index: false,
    is_quick_search: false,
    is_advanced_search: false,
    is_sort: false,
    is_null: !item.required,
    is_list_show: false,
    is_insert_show: true,
    is_update_show: true,
    sequence: 0,
    original_field_id: 0,
    binding: '',
    linkage_config: dataSource.linkageConfig,
  }

  if (item.binding) {
    field.binding = item.binding
  }
  const formSpan = item.form_span || item.formSpan
  if (formSpan !== undefined) {
    field.form_span = formSpan
  }
  const detailSpan = item.detail_span || item.detailSpan
  if (detailSpan !== undefined) {
    field.detail_span = detailSpan
  }

  if (options) {
    ;(field as any).options = options
  }
  if (item.placeholder) {
    ;(field as any).placeholder = item.placeholder
  }

  return field
}

const buildJsonSchemaField = (
  code: string,
  prop: any,
  requiredList: string[],
): TableField | null => {
  const enumValues = prop?.enum
  const enumNames = prop?.enumNames
  const options = Array.isArray(enumValues)
    ? enumValues.map((value: unknown, index: number) => ({
        value,
        label: enumNames?.[index] ?? String(value),
      }))
    : undefined

  const field: ParamFieldSchema = {
    field_code: code,
    field_name: prop?.title || code,
    required: requiredList.includes(code),
    field_type: prop?.type,
    input_type: Array.isArray(enumValues) ? 'select' : prop?.input_type || prop?.['x-input-type'],
    default_value: prop?.default,
    numeric_precision:
      prop?.numeric_precision ?? prop?.numericPrecision ?? prop?.['x-numeric-precision'],
    numeric_scale: prop?.numeric_scale ?? prop?.numericScale ?? prop?.['x-numeric-scale'],
    placeholder: prop?.placeholder,
    dict_code: prop?.dict_code || prop?.dictCode || prop?.['x-dict-code'],
    linkage_config: prop?.linkage_config || prop?.linkageConfig || prop?.['x-linkage-config'],
    data_source: prop?.data_source || prop?.dataSource || prop?.['x-data-source'],
    relation: prop?.relation || prop?.['x-relation'],
    cascader: prop?.cascader || prop?.['x-cascader'],
    binding: prop?.binding || prop?.['x-binding'],
    form_span: prop?.form_span || prop?.formSpan || prop?.['x-form-span'],
    detail_span: prop?.detail_span || prop?.detailSpan || prop?.['x-detail-span'],
  }

  if (options) {
    field.options = options
  }

  return buildParamField(field)
}

export const parseParamsSchema = (schemaText?: string) => {
  if (!schemaText) return []
  try {
    const parsed = JSON.parse(schemaText)
    if (Array.isArray(parsed)) {
      return parsed.map(buildParamField).filter(Boolean) as TableField[]
    }

    if (Array.isArray(parsed.fields)) {
      return parsed.fields.map(buildParamField).filter(Boolean) as TableField[]
    }

    if (parsed && typeof parsed === 'object' && parsed.properties) {
      const requiredList = Array.isArray(parsed.required) ? parsed.required : []
      return Object.entries(parsed.properties)
        .map(([code, prop]) => buildJsonSchemaField(code, prop, requiredList))
        .filter(Boolean) as TableField[]
    }

    return []
  } catch (error) {
    console.error('解析按钮参数Schema失败', error)
    return []
  }
}
