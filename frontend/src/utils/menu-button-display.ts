import type { MenuButton } from 'src/api/services/sys-menu'
import { SysMenuButtonDisplayMode, SysMenuButtonPosition } from 'src/types/enum'

interface MenuButtonDisplayOptions {
  label?: string
  icon?: string | undefined
  position?: SysMenuButtonPosition
}

type Translate = (key: string) => string

const BUILT_IN_BUTTON_PREFIXES = [
  'develop_',
  'integration_',
  'organization_',
  'report_',
  'system_',
]

const BUILT_IN_BUTTON_ACTION_KEYS: Record<string, string> = {
  assign_permission: 'ui.allocationOfCompetence',
  assign_role: 'ui.assignRoles',
  cancel: 'ui.cancelExecution',
  create: 'ui.create',
  create_button: 'ui.addButton',
  create_child: 'ui.addSubmenu',
  create_version: 'ui.createANewVersion',
  delete: 'ui.delete',
  delete_button: 'ui.removeButton',
  detail: 'ui.details',
  disable: 'ui.disabled',
  duplicate: 'ui.copy',
  enable: 'ui.enabled',
  export: 'ui.export',
  field_manager: 'ui.field',
  init_meta: 'ui.initializeMetadata',
  navigate: 'ui.design',
  publish: 'ui.publishAction',
  publish_menu: 'ui.releaseToMenu',
  query: 'ui.query',
  reset_password: 'ui.resetPassword',
  revoke: 'ui.revokeAction',
  rotate: 'ui.rotation',
  rotate_secret: 'ui.rotationKey',
  run: 'ui.run',
  save: 'ui.save',
  sync_fields: 'ui.syncFields',
  unlock_login: 'ui.unlock',
  unpublish_menu: 'ui.cancelReleaseMenu',
  update: 'ui.edit',
  update_button: 'ui.editButton',
}

const BUILT_IN_BUTTON_CODE_KEYS: Record<string, string> = {
  integration_execution_detail: 'ui.implementationDetails',
  integration_log_detail: 'ui.callLogDetails',
  integration_sync_task_run: 'ui.runOnce',
  report_manage_create: 'ui.newReport',
  report_manage_design: 'ui.design',
  report_manage_preview: 'ui.runPreview',
  system_data_permission_config_grant_create: 'ui.addPermissionGrant',
  system_data_permission_config_grant_preflight: 'ui.checkPermissionGrant',
  system_data_permission_config_ownership_create: 'ui.addOwnershipDefinition',
  system_data_permission_config_policy_create: 'ui.addPermissionPolicy',
  system_data_permission_config_policy_preflight: 'ui.checkPermissionPolicy',
  system_data_permission_config_policy_rule_replace: 'ui.configurePermissionPolicyRules',
  system_data_permission_config_policy_update: 'ui.editPermissionPolicy',
  system_data_permission_config_resource_create: 'ui.addDataResource',
  system_data_permission_config_resource_operation_replace: 'ui.configureDataResourceOperations',
  system_data_permission_config_resource_preflight: 'ui.checkDataResource',
  system_data_permission_config_resource_update: 'ui.editDataResource',
}

const isBuiltInButton = (code: string) =>
  BUILT_IN_BUTTON_PREFIXES.some((prefix) => code.startsWith(prefix))

export const resolveMenuButtonLabel = (button: MenuButton, translate: Translate) => {
  const code = String(button.code || '')
  if (!isBuiltInButton(code)) return button.name

  const key = BUILT_IN_BUTTON_CODE_KEYS[code] || BUILT_IN_BUTTON_ACTION_KEYS[button.event_action]
  return key ? translate(key) : button.name
}

const normalizeDisplayMode = (mode?: string) => {
  const normalized = (mode || SysMenuButtonDisplayMode.AUTO).trim().toLowerCase()
  if (normalized === 'icon') return SysMenuButtonDisplayMode.ICON
  if (normalized === 'text') return SysMenuButtonDisplayMode.TEXT
  if (normalized === 'icon_text') return SysMenuButtonDisplayMode.ICON_TEXT
  return SysMenuButtonDisplayMode.AUTO
}

const defaultDisplayMode = (position?: SysMenuButtonPosition) => {
  if (position === SysMenuButtonPosition.LINE) return SysMenuButtonDisplayMode.ICON
  if (position === SysMenuButtonPosition.FORM_BOTTOM) return SysMenuButtonDisplayMode.TEXT
  return SysMenuButtonDisplayMode.ICON_TEXT
}

export const menuButtonDisplayProps = (btn: MenuButton, options: MenuButtonDisplayOptions = {}) => {
  const label = options.label || btn.name
  const icon = (options.icon ?? btn.icon) || undefined
  const position = options.position || btn.position
  const configuredMode = normalizeDisplayMode(String(btn.display_mode || ''))
  const mode =
    configuredMode === SysMenuButtonDisplayMode.AUTO ? defaultDisplayMode(position) : configuredMode

  const showIcon = !!icon && mode !== SysMenuButtonDisplayMode.TEXT
  const showText = mode !== SysMenuButtonDisplayMode.ICON || !icon

  return {
    icon: showIcon ? icon : undefined,
    label: showText ? label : undefined,
    round: showIcon && !showText,
    'aria-label': label,
  }
}
