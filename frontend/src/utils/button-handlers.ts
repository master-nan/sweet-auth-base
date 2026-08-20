import { SysMenuButtonEventAction } from 'src/types/enum'

export interface ButtonActionContext {
  table_code?: string
  row?: Record<string, any>
  selection?: Array<Record<string, any>>
  params?: Record<string, any>
  formData?: Record<string, any>
  onCreate?: () => void | Promise<void>
  onUpdate?: (row?: Record<string, any>) => void | Promise<void>
  onDelete?: (row?: Record<string, any>) => void | Promise<void>
  onRefresh?: () => void | Promise<void>
  onCustom?: (name: string, ctx: ButtonActionContext) => void | Promise<void>
  onCloseDialog?: () => void
  onClearSelection?: () => void
  onBatchDelete?: (rows: Array<Record<string, any>>) => void | Promise<void>
  onCopy?: (row: Record<string, any>) => void | Promise<void>
  onExport?: (ctx: ButtonActionContext) => void | Promise<void>
  onNavigate?: (path: string, ctx: ButtonActionContext) => void | Promise<void>
  onOpenDetail?: (row: Record<string, any>) => void | Promise<void>
}

export type ButtonActionHandler = (ctx: ButtonActionContext) => void | Promise<void>

/**
 * Hook handler 返回 false 表示中止后续执行（仅 before hooks 生效）
 */
export type ButtonHookHandler = (ctx: ButtonActionContext) => boolean | Promise<boolean>

export interface ButtonDisableContext {
  row?: Record<string, any> | null
  selection?: Array<Record<string, any>>
  selectionCount?: number
  query?: Record<string, any>
  params?: Record<string, any>
}

export interface ButtonDisableConfig {
  is_disabled?: boolean
  disable_when?: string
}

export const buttonHandlerOptionKeys = [
  { label: 'button.action.create', value: SysMenuButtonEventAction.CREATE },
  { label: 'button.action.update', value: SysMenuButtonEventAction.UPDATE },
  { label: 'button.action.delete', value: SysMenuButtonEventAction.DELETE },
  { label: 'button.action.refresh', value: SysMenuButtonEventAction.REFRESH },
  { label: 'button.action.batch_delete', value: SysMenuButtonEventAction.BATCH_DELETE },
  { label: 'button.action.copy', value: SysMenuButtonEventAction.COPY },
  { label: 'button.action.export', value: SysMenuButtonEventAction.EXPORT },
  { label: 'button.action.navigate', value: SysMenuButtonEventAction.NAVIGATE },
  { label: 'button.action.detail', value: SysMenuButtonEventAction.DETAIL },
  { label: 'button.action.custom', value: SysMenuButtonEventAction.CUSTOM },
]

export const getButtonHandlerOptions = (t: (key: string) => string) => {
  return buttonHandlerOptionKeys.map((item) => ({
    label: t(item.label),
    value: item.value,
  }))
}

// ==================== Hook 注册表 ====================

const beforeHookRegistry: Record<string, ButtonHookHandler> = {
  /** 要求必须选中至少一行 */
  requireSelection: (ctx) => {
    if (!ctx.selection || ctx.selection.length === 0) {
      return false
    }
    return true
  },
  /** 要求必须有行上下文 */
  requireRow: (ctx) => {
    if (!ctx.row || Object.keys(ctx.row).length === 0) {
      return false
    }
    return true
  },
}

const afterHookRegistry: Record<string, ButtonHookHandler> = {
  /** 刷新数据 */
  refresh: async (ctx) => {
    await ctx.onRefresh?.()
    return true
  },
  /** 清空选中 */
  clearSelection: (ctx) => {
    ctx.onClearSelection?.()
    return true
  },
  /** 关闭对话框 */
  closeDialog: (ctx) => {
    ctx.onCloseDialog?.()
    return true
  },
}

/** 注册自定义 before hook */
export const registerBeforeHook = (name: string, handler: ButtonHookHandler) => {
  beforeHookRegistry[name] = handler
}

/** 注册自定义 after hook */
export const registerAfterHook = (name: string, handler: ButtonHookHandler) => {
  afterHookRegistry[name] = handler
}

/** 获取所有可用的 before hook 选项 */
export const getBeforeHookOptions = () => [
  { label: '要求选中行', value: 'requireSelection' },
  { label: '要求行上下文', value: 'requireRow' },
]

/** 获取所有可用的 after hook 选项 */
export const getAfterHookOptions = () => [
  { label: '刷新数据', value: 'refresh' },
  { label: '清空选中', value: 'clearSelection' },
  { label: '关闭对话框', value: 'closeDialog' },
]

/**
 * 解析 hooks JSON 字符串为字符串数组
 */
const parseHooks = (hooksJson?: string): string[] => {
  if (!hooksJson) return []
  try {
    const parsed = JSON.parse(hooksJson)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

/**
 * 执行 before hooks，任一返回 false 则中止
 */
export const runBeforeHooks = async (
  hooksJson: string | undefined,
  ctx: ButtonActionContext,
): Promise<boolean> => {
  const hooks = parseHooks(hooksJson)
  for (const hookName of hooks) {
    const handler = beforeHookRegistry[hookName]
    if (handler) {
      const result = await handler(ctx)
      if (!result) return false
    }
  }
  return true
}

/**
 * 执行 after hooks
 */
export const runAfterHooks = async (
  hooksJson: string | undefined,
  ctx: ButtonActionContext,
): Promise<void> => {
  const hooks = parseHooks(hooksJson)
  for (const hookName of hooks) {
    const handler = afterHookRegistry[hookName]
    if (handler) {
      await handler(ctx)
    }
  }
}

const getValueByPath = (target: any, path: string) => {
  if (!path) return undefined
  return path.split('.').reduce((acc, key) => (acc == null ? undefined : acc[key]), target)
}

const resolveDisableValue = (field: string, ctx: ButtonDisableContext) => {
  if (!field) return undefined
  if (field.startsWith('row.')) return getValueByPath(ctx.row, field.slice(4))
  if (field.startsWith('selection.')) return getValueByPath(ctx.selection, field.slice(10))
  if (field.startsWith('query.')) return getValueByPath(ctx.query, field.slice(6))
  if (field.startsWith('params.')) return getValueByPath(ctx.params, field.slice(7))
  if (ctx.row && field in ctx.row) return ctx.row[field]
  return (ctx as Record<string, any>)[field]
}

const isEmptyValue = (value: any) => {
  return (
    value === null ||
    value === undefined ||
    value === '' ||
    (Array.isArray(value) && value.length === 0)
  )
}

const evaluateDisableRule = (rule: any, ctx: ButtonDisableContext) => {
  const op = String(rule?.op || 'eq').toLowerCase()
  const left = resolveDisableValue(String(rule?.field || ''), ctx)
  const right = rule?.value

  switch (op) {
    case 'eq':
      return left === right
    case 'ne':
      return left !== right
    case 'gt':
      return Number(left) > Number(right)
    case 'gte':
      return Number(left) >= Number(right)
    case 'lt':
      return Number(left) < Number(right)
    case 'lte':
      return Number(left) <= Number(right)
    case 'in':
      return Array.isArray(right) ? right.includes(left) : false
    case 'not_in':
      return Array.isArray(right) ? !right.includes(left) : false
    case 'includes':
      return Array.isArray(left) ? left.includes(right) : String(left || '').includes(String(right))
    case 'not_includes':
      return Array.isArray(left)
        ? !left.includes(right)
        : !String(left || '').includes(String(right))
    case 'empty':
      return isEmptyValue(left)
    case 'not_empty':
      return !isEmptyValue(left)
    case 'truthy':
      return !!left
    case 'falsy':
      return !left
    default:
      return left === right
  }
}

const evaluateDisableGroup = (group: any, ctx: ButtonDisableContext): boolean => {
  if (!group) return false
  if (Array.isArray(group)) return group.every((rule) => evaluateDisableGroup(rule, ctx))
  if (group.all) return group.all.every((rule: any) => evaluateDisableGroup(rule, ctx))
  if (group.any) return group.any.some((rule: any) => evaluateDisableGroup(rule, ctx))
  if (group.not) return !evaluateDisableGroup(group.not, ctx)
  if (group.field) return evaluateDisableRule(group, ctx)
  return false
}

export const evaluateButtonDisabled = (
  button: ButtonDisableConfig,
  ctx: ButtonDisableContext = {},
) => {
  if (button.is_disabled) return true
  if (!button.disable_when) return false
  try {
    return evaluateDisableGroup(JSON.parse(button.disable_when), {
      selection: [],
      selectionCount: 0,
      query: {},
      params: {},
      ...ctx,
    })
  } catch (error) {
    console.warn('按钮禁用条件解析失败', error)
    return false
  }
}

const builtinHandlers: Record<string, ButtonActionHandler> = {
  [SysMenuButtonEventAction.CREATE]: async (ctx) => {
    await ctx.onCreate?.()
  },
  [SysMenuButtonEventAction.UPDATE]: async (ctx) => {
    await ctx.onUpdate?.(ctx.row)
  },
  [SysMenuButtonEventAction.DELETE]: async (ctx) => {
    await ctx.onDelete?.(ctx.row)
  },
  [SysMenuButtonEventAction.REFRESH]: async (ctx) => {
    await ctx.onRefresh?.()
  },
  [SysMenuButtonEventAction.BATCH_DELETE]: async (ctx) => {
    if (ctx.selection && ctx.selection.length > 0) {
      await ctx.onBatchDelete?.(ctx.selection)
    }
  },
  [SysMenuButtonEventAction.COPY]: async (ctx) => {
    if (ctx.row) {
      await ctx.onCopy?.(ctx.row)
    }
  },
  [SysMenuButtonEventAction.EXPORT]: async (ctx) => {
    await ctx.onExport?.(ctx)
  },
  [SysMenuButtonEventAction.NAVIGATE]: async (ctx) => {
    // api_path 此时作为目标路由路径
    await ctx.onNavigate?.('', ctx)
  },
  [SysMenuButtonEventAction.DETAIL]: async (ctx) => {
    if (ctx.row) {
      await ctx.onOpenDetail?.(ctx.row)
    }
  },
  [SysMenuButtonEventAction.CUSTOM]: async (ctx) => {
    await ctx.onCustom?.('custom', ctx)
  },
}

const resolveButtonHandler = (name: string): ButtonActionHandler | undefined => {
  return builtinHandlers[name]
}

export const executeButtonAction = async (name: string, ctx: ButtonActionContext) => {
  const action = name.trim()
  const handler = resolveButtonHandler(action)
  if (!handler) {
    if (ctx.onCustom) {
      await ctx.onCustom(action, ctx)
      return
    }
    throw new Error(`未注册的按钮动作: ${action}`)
  }
  await handler(ctx)
}
