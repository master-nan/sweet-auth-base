/**
 * 列表列格式化工具
 * 统一处理日期时间、字典、布尔、关联表字段的列显示
 */
import { SysTableFieldType, SysTableFieldInputType } from 'src/types/enum'
import type { TableField } from 'src/api/services/sys-table'
import { useGeneralizationApi } from 'src/api/services/generalization'
import type { Query, TableColumn } from 'src/types/global'
import { parseLinkageConfig } from 'src/utils/field-metadata'
import { useUserStore } from 'src/stores/user'
import { resolveRelationMenuId } from 'src/utils/menu-context'

export { parseLinkageConfig } from 'src/utils/field-metadata'

// ─── 日期时间格式化 ───────────────────────────────────

/**
 * 格式化 ISO 日期时间字符串为可读格式
 * "2025-03-24T20:57:58+08:00" → "2025-03-24 20:57:58"
 */
export const formatDateTime = (val: any): string => {
  if (!val) return ''
  const str = String(val)
  // 处理 ISO 格式: 2025-03-24T20:57:58+08:00 或 2025-03-24T20:57:58Z
  if (str.includes('T')) {
    try {
      const d = new Date(str)
      if (isNaN(d.getTime())) return str
      const pad = (n: number) => String(n).padStart(2, '0')
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
    } catch {
      return str
    }
  }
  return str
}

/**
 * 格式化日期字符串
 * "2025-03-24T00:00:00+08:00" → "2025-03-24"
 */
export const formatDate = (val: any): string => {
  if (!val) return ''
  const str = String(val)
  if (str.includes('T')) {
    try {
      const d = new Date(str)
      if (isNaN(d.getTime())) return str
      const pad = (n: number) => String(n).padStart(2, '0')
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
    } catch {
      return str
    }
  }
  return str
}

/**
 * 格式化时间字符串
 */
export const formatTime = (val: any): string => {
  if (!val) return ''
  const str = String(val)
  if (str.includes('T')) {
    try {
      const d = new Date(str)
      if (isNaN(d.getTime())) return str
      const pad = (n: number) => String(n).padStart(2, '0')
      return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
    } catch {
      return str
    }
  }
  return str
}

// ─── 关联表查找映射 ───────────────────────────────────

export type LookupMap = Record<string, string>

const rowsFromRelationResponse = (response: any): Array<Record<string, any>> => {
  const rawRows = response?.data
  return Array.isArray(rawRows) ? rawRows : rawRows?.data || []
}

const relationValueKeyForFilter = (cfg: any) => String(cfg?.valueKey || 'id').trim()

const mergeRowsIntoLookup = (lookup: LookupMap, rows: Array<Record<string, any>>, cfg: any) => {
  const labelKey = cfg?.labelKey || 'label'
  const valueKey = cfg?.valueKey || 'value'
  rows.forEach((row) => {
    const rawLabel = row[labelKey]
    const rawValue = row[valueKey]
    const label =
      rawLabel ?? row.label ?? row.name ?? row.title ?? row.menu_name ?? row.dict_name ?? ''
    const value = rawValue ?? row.value ?? row.id
    if (value != null) {
      lookup[String(value)] = String(label)
    }
  })
}

const relationLookupFields = (fields: TableField[]) => {
  return fields
    .filter((field) => field.is_list_show && !field.dict_code)
    .map((field) => ({ field, linkage: parseLinkageConfig(field) }))
    .filter(
      (item): item is { field: TableField; linkage: Record<string, any> } =>
        item.linkage?.mode === 'relation' || item.linkage?.mode === 'cascader',
    )
}

/**
 * 为字段列表中所有关联表字段（linkage_config 中有 relation/cascader 模式）
 * 批量加载关联表数据并构建 value → label 映射。
 *
 * @returns Record<fieldCode, LookupMap>
 */
export const buildRelationLookups = async (
  fields: TableField[],
): Promise<Record<string, LookupMap>> => {
  const userStore = useUserStore()
  const generalizationApi = useGeneralizationApi()
  const lookups: Record<string, LookupMap> = {}
  const tasks = relationLookupFields(fields).map(({ field, linkage }) => ({
    fieldCode: field.field_code,
    cfg: linkage,
  }))

  if (tasks.length === 0) return lookups

  const promises = tasks.map(async ({ fieldCode, cfg }) => {
    try {
      const tableCode = cfg.tableCode
      if (!tableCode) return

      const query: Query = {
        page: 1,
        num: cfg.pageSize || 500,
        table_code: tableCode,
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
      }
      const menuId = resolveRelationMenuId(userStore.menus, cfg)
      if (menuId > 0) {
        query.menu_id = menuId
      }

      const map: LookupMap = {}
      const res = await generalizationApi.queryGeneralizationByCode(tableCode, query)
      mergeRowsIntoLookup(map, rowsFromRelationResponse(res), cfg)
      lookups[fieldCode] = map
    } catch (error) {
      console.warn(`加载关联字段 ${fieldCode} 查找表失败`, error)
    }
  })

  await Promise.all(promises)
  return lookups
}

export const hydrateRelationLookups = async (
  fields: TableField[],
  rows: Array<Record<string, any>>,
  existingLookups: Record<string, LookupMap> = {},
  fallbackMenuId = 0,
): Promise<Record<string, LookupMap>> => {
  if (rows.length === 0) return existingLookups

  const userStore = useUserStore()
  const generalizationApi = useGeneralizationApi()
  const tasks = relationLookupFields(fields)

  await Promise.all(
    tasks.map(async ({ field, linkage }) => {
      const fieldCode = field.field_code
      const lookup = existingLookups[fieldCode] || {}
      existingLookups[fieldCode] = lookup

      const missingValues = new Map<string, unknown>()
      rows.forEach((row) => {
        const value = row[fieldCode]
        if (value === null || value === undefined || value === '') return
        const key = String(value)
        if (!lookup[key]) {
          missingValues.set(key, value)
        }
      })
      if (missingValues.size === 0) return

      const tableCode = linkage.tableCode
      if (!tableCode) return

      const valueKey = relationValueKeyForFilter(linkage)
      if (!valueKey) return

      const query: Query = {
        page: 1,
        num: Math.max(missingValues.size, 20),
        table_code: tableCode,
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
        filters: {
          [valueKey]: Array.from(missingValues.values()),
        },
      }
      const menuId = resolveRelationMenuId(userStore.menus, linkage, fallbackMenuId)
      if (menuId > 0) {
        query.menu_id = menuId
      }

      try {
        const res = await generalizationApi.queryGeneralizationByCode(tableCode, query)
        mergeRowsIntoLookup(lookup, rowsFromRelationResponse(res), linkage)
      } catch (error) {
        console.warn(`补齐关联字段 ${fieldCode} 当前页查找表失败`, error)
      }
    }),
  )

  return existingLookups
}

// ─── 列格式化器构建 ───────────────────────────────────

interface FormatContext {
  /** 字典标签获取函数 */
  getDictLabel: (dictCode: string, value: any) => string
  /** 关联表查找映射 */
  relationLookups?: Record<string, LookupMap>
}

/**
 * 根据字段元数据为列构建 format 函数。
 *
 * 优先级:
 * 1. 字典字段 → dictLabel
 * 2. 关联表字段 → relationLookup
 * 3. 布尔字段 → 是/否
 * 4. 日期时间字段 → 格式化
 * 5. 无特殊处理 → undefined (使用原始值)
 */
export const buildColumnFormat = (
  field: TableField,
  ctx: FormatContext,
): ((val: any, row?: any) => string) | undefined => {
  // 字典字段
  if (field.dict_code) {
    const isBool = field.field_type === SysTableFieldType.BOOLEAN
    return (val: any) => {
      // 布尔值额外尝试用 '1'/'0' 匹配
      let label = ctx.getDictLabel(field.dict_code, val)
      if (!label && isBool) {
        label = ctx.getDictLabel(field.dict_code, val === true || val === 1 ? '1' : '0')
      }
      if (label) return label
      // fallback: 布尔字段显示"是/否"，其他显示原始值
      if (isBool) return val === true || val === 1 ? '是' : '否'
      return val == null ? '' : String(val)
    }
  }

  const linkage = parseLinkageConfig(field)
  if (linkage?.mode === 'relation' || linkage?.mode === 'cascader') {
    return (val: any) => {
      if (val == null || val === '') return ''
      const lookup = ctx.relationLookups?.[field.field_code]
      return lookup?.[String(val)] || String(val)
    }
  }

  // 布尔字段
  if (field.field_type === SysTableFieldType.BOOLEAN) {
    return (val: any) => (val === true || val === 1 ? '是' : '否')
  }

  // 日期时间字段
  if (field.field_type === SysTableFieldType.DATETIME) {
    return formatDateTime
  }
  if (field.field_type === SysTableFieldType.DATE) {
    return formatDate
  }
  if (field.field_type === SysTableFieldType.TIME) {
    return formatTime
  }

  // 文件字段由表格/详情里的 FileDisplay 组件展示文件名和访问入口，这里只保留兜底文本。
  if (field.input_type === SysTableFieldInputType.FILE_PICKER) {
    return (val: any) => {
      if (val == null || val === '' || val === '0') return ''
      return '已上传'
    }
  }

  return undefined
}

/**
 * 批量为所有列表显示字段构建 format 函数映射
 *
 * @returns Record<fieldCode, formatFn>
 */
export const buildAllColumnFormats = (
  fields: TableField[],
  ctx: FormatContext,
): Record<string, (val: any, row?: any) => string> => {
  const result: Record<string, (val: any, row?: any) => string> = {}
  fields.forEach((field) => {
    if (!field.is_list_show) return
    const fmt = buildColumnFormat(field, ctx)
    if (fmt) {
      result[field.field_code] = fmt
    }
  })
  return result
}

// ─── 高级封装：自动构建列数组 ─────────────────────────

export interface RuntimeColumnOverride<Row extends object> {
  fieldCode: string
  label?: string
  align?: TableColumn<Row>['align']
  visible?: boolean
  order?: number
  format?: TableColumn<Row>['format']
  style?: TableColumn<Row>['style']
  classes?: TableColumn<Row>['classes']
  headerStyle?: string
  headerClasses?: string
}

export interface RuntimeVirtualColumn<Row extends object>
  extends TableColumn<Row> {
  defaultVisible?: boolean
  order?: number
  serverSortField?: string
}

export interface RuntimeColumnResolution<Row extends object> {
  columns: TableColumn<Row>[]
  visibleColumns: string[]
  advancedFields: TableField[]
  quickSearchFields: TableField[]
  formFields: TableField[]
  detailFields: TableField[]
  sortableFields: ReadonlySet<string>
}

export interface RuntimeColumnResolverOptions<Row extends object> {
  context: FormatContext
  overrides?: RuntimeColumnOverride<Row>[]
  virtualColumns?: RuntimeVirtualColumn<Row>[]
}

/**
 * 将 Runtime Metadata 的基础字段事实与页面覆盖、虚拟列合成为 QTable 视图模型。
 * 页面覆盖只能调整显示，元数据未声明可排序的基础字段不会被提升为可排序字段。
 */
export const resolveRuntimeColumns = <Row extends object>(
  fields: TableField[],
  options: RuntimeColumnResolverOptions<Row>,
): RuntimeColumnResolution<Row> => {
  const orderedFields = fields.slice().sort((left, right) => left.sequence - right.sequence)
  const overrides = new Map(
    (options.overrides || []).map((override) => [override.fieldCode, override]),
  )
  const sortableFields = new Set<string>()
  const resolved = orderedFields
    .filter((field) => field.is_list_show)
    .map((field, index) => {
      const override = overrides.get(field.field_code)
      if (field.is_sort) sortableFields.add(field.field_code)
      const column: TableColumn<Row> & { order: number; defaultVisible: boolean } = {
        name: field.field_code,
        label: override?.label || field.field_name,
        field: field.field_code,
        align: override?.align || 'left',
        sortable: field.is_sort,
        order: override?.order ?? field.sequence ?? index,
        defaultVisible: override?.visible !== false,
      }
      const format = override?.format || buildColumnFormat(field, options.context)
      if (format) column.format = format
      if (override?.style) column.style = override.style
      if (override?.classes) column.classes = override.classes
      if (override?.headerStyle) column.headerStyle = override.headerStyle
      if (override?.headerClasses) column.headerClasses = override.headerClasses
      return column
    })

  const virtual = (options.virtualColumns || []).map((column, index) => {
    if (column.serverSortField) sortableFields.add(column.serverSortField)
    return {
      ...column,
      sortable: !!column.serverSortField,
      order: column.order ?? 10000 + index,
      defaultVisible: column.defaultVisible !== false,
    }
  })

  const combined = [...resolved, ...virtual].sort((left, right) => left.order - right.order)
  const columns = combined.map((resolvedColumn) => {
    const { order, defaultVisible, ...column } = resolvedColumn
    void order
    void defaultVisible
    return column
  })

  return {
    columns,
    visibleColumns: combined.filter((column) => column.defaultVisible).map((column) => column.name),
    advancedFields: orderedFields.filter((field) => field.is_advanced_search),
    quickSearchFields: orderedFields.filter((field) => field.is_quick_search),
    formFields: orderedFields.filter((field) => field.is_insert_show || field.is_update_show),
    detailFields: orderedFields.filter((field) => (field.detail_span || 0) > 0),
    sortableFields,
  }
}

/**
 * 从字段元数据自动构建 QTable columns 数组。
 *
 * 典型用法（无需关联表查找的系统管理页面）：
 * ```ts
 * const { columns: cols, advancedFields } = buildTableColumns(fields, { getDictLabel })
 * ```
 *
 * 带自定义 format 覆盖（如 database 页面的枚举映射）：
 * ```ts
 * const { columns: cols } = buildTableColumns(fields, ctx, {
 *   field_type: (val) => SysTableFieldTypeMap[val] || val,
 * })
 * ```
 */
export const buildTableColumns = (
  fields: TableField[],
  ctx: FormatContext,
  /** 自定义 format 覆盖，优先级高于自动推断 */
  customFormats?: Record<string, (val: any, row?: any) => string>,
): { columns: TableColumn<Record<string, any>>[]; advancedFields: TableField[] } => {
  const overrides = Object.entries(customFormats || {}).map(([fieldCode, format]) => ({
    fieldCode,
    format,
  }))
  const resolution = resolveRuntimeColumns<Record<string, any>>(fields, {
    context: ctx,
    overrides,
    virtualColumns: [
      {
        name: 'actions',
        align: 'center',
        label: '操作',
        field: 'actions',
        defaultVisible: true,
      },
    ],
  })
  return { columns: resolution.columns, advancedFields: resolution.advancedFields }
}
