import { translate as t } from 'src/boot/i18n'
export enum SysTableType {
  SYSTEM = 1,
  VIEW,
}

export const SysTableTypeMap = {
  get [SysTableType.SYSTEM]() {
    return t('ui.systemChart')
  },
  get [SysTableType.VIEW]() {
    return t('ui.view')
  },
}

export enum SysMasterDetailMode {
  AUTO = 'auto',
  SUMMARY = 'summary',
  TABLE = 'table',
  STACKED = 'stacked',
}

export const SysMasterDetailModeMap = {
  get [SysMasterDetailMode.AUTO]() {
    return t('ui.auto')
  },
  get [SysMasterDetailMode.SUMMARY]() {
    return t('ui.summaryMasterTable')
  },
  get [SysMasterDetailMode.TABLE]() {
    return t('ui.mainTableTable')
  },
  get [SysMasterDetailMode.STACKED]() {
    return t('ui.masterSWatch')
  },
}

export enum SysFormOpenMode {
  AUTO = 'auto',
  DIALOG = 'dialog',
  PAGE = 'page',
}

export const SysFormOpenModeMap = {
  get [SysFormOpenMode.AUTO]() {
    return t('ui.auto')
  },
  get [SysFormOpenMode.DIALOG]() {
    return t('ui.box')
  },
  get [SysFormOpenMode.PAGE]() {
    return t('ui.pages')
  },
}

export enum SysDetailOpenMode {
  AUTO = 'auto',
  DIALOG = 'dialog',
  PAGE = 'page',
}

export const SysDetailOpenModeMap = {
  get [SysDetailOpenMode.AUTO]() {
    return t('ui.auto')
  },
  get [SysDetailOpenMode.DIALOG]() {
    return t('ui.box')
  },
  get [SysDetailOpenMode.PAGE]() {
    return t('ui.pages')
  },
}

export enum SysTableFieldInputType {
  INPUT = 1,
  INPUT_NUMBER,
  TEXTAREA,
  SELECT,
  DATE_PICKER,
  DATETIME_PICKER,
  TIME_PICKER,
  YEAR_PICKER,
  YEAR_MONTH_PICKER,
  FILE_PICKER,
  BOOLEAN,
  JSON_EDITOR,
  ARRAY_INPUT,
  KEY_VALUE_EDITOR,
  CASCADER,
  RICH_TEXT,
}

export const SysTableFieldInputTypeMap = {
  get [SysTableFieldInputType.INPUT]() {
    return t('ui.inputBox')
  },
  get [SysTableFieldInputType.INPUT_NUMBER]() {
    return t('ui.numberInput')
  },
  get [SysTableFieldInputType.TEXTAREA]() {
    return t('ui.multilineText')
  },
  get [SysTableFieldInputType.SELECT]() {
    return t('ui.dropdownSelection')
  },
  get [SysTableFieldInputType.DATE_PICKER]() {
    return t('ui.dateSelection')
  },
  get [SysTableFieldInputType.DATETIME_PICKER]() {
    return t('ui.dateAndTime')
  },
  get [SysTableFieldInputType.TIME_PICKER]() {
    return t('ui.timeSelection')
  },
  get [SysTableFieldInputType.YEAR_PICKER]() {
    return t('ui.yearPickerType')
  },
  get [SysTableFieldInputType.YEAR_MONTH_PICKER]() {
    return t('ui.yearSelection')
  },
  get [SysTableFieldInputType.FILE_PICKER]() {
    return t('ui.fileSelection')
  },
  get [SysTableFieldInputType.BOOLEAN]() {
    return t('ui.booleanSwitch')
  },
  get [SysTableFieldInputType.JSON_EDITOR]() {
    return t('ui.jsonEditor')
  },
  get [SysTableFieldInputType.ARRAY_INPUT]() {
    return t('ui.clusterInput')
  },
  get [SysTableFieldInputType.KEY_VALUE_EDITOR]() {
    return t('ui.keyToEdit')
  },
  get [SysTableFieldInputType.CASCADER]() {
    return t('ui.cascadeSelection')
  },
  get [SysTableFieldInputType.RICH_TEXT]() {
    return t('ui.richTextEditor')
  },
}

export enum ExpressionLogic {
  AND = 1,
  OR,
}

export const ExpressionLogicMap = {
  get [ExpressionLogic.AND]() {
    return t('ui.and')
  },
  get [ExpressionLogic.OR]() {
    return t('ui.or')
  },
}

export enum ExpressionType {
  GT = 1,
  LT,
  GTE,
  LTE,
  EQ,
  NE,
  LIKE,
  NOT_LIKE,
  IN,
  NOT_IN,
  IS_NULL,
  IS_NOT_NULL,
  BETWEEN,
  NOT_BETWEEN,
}

export const ExpressionTypeMap = {
  get [ExpressionType.GT]() {
    return t('ui.greaterThan')
  },
  get [ExpressionType.LT]() {
    return t('ui.lessThan')
  },
  get [ExpressionType.GTE]() {
    return t('ui.greaterThanOrEqualTo')
  },
  get [ExpressionType.LTE]() {
    return t('ui.lessThanOrEqualTo')
  },
  get [ExpressionType.EQ]() {
    return t('ui.equals')
  },
  get [ExpressionType.NE]() {
    return t('ui.notEqualTo')
  },
  get [ExpressionType.LIKE]() {
    return t('ui.containsOperator')
  },
  get [ExpressionType.NOT_LIKE]() {
    return t('ui.notContainsOperator')
  },
  get [ExpressionType.IN]() {
    return t('ui.inside')
  },
  get [ExpressionType.NOT_IN]() {
    return t('ui.itSNotInside')
  },
  get [ExpressionType.IS_NULL]() {
    return t('ui.empty')
  },
  get [ExpressionType.IS_NOT_NULL]() {
    return t('ui.nonEmpty')
  },
  get [ExpressionType.BETWEEN]() {
    return t('ui.area')
  },
  get [ExpressionType.NOT_BETWEEN]() {
    return t('ui.notInTheCompartment')
  },
}

export enum SysTableFieldType {
  BIGINT = 1,
  DECIMAL = 2,
  VARCHAR = 3,
  TEXT = 4,
  BOOLEAN = 5,
  DATE = 6,
  DATETIME = 7,
  TIME = 8,
  SMALLINT = 9,
  JSON = 10,
  INT = 11,
}

export const SysTableFieldTypeMap = {
  get [SysTableFieldType.BIGINT]() {
    return t('ui.largeNumber')
  },
  get [SysTableFieldType.DECIMAL]() {
    return t('ui.exactDecimal')
  },
  get [SysTableFieldType.VARCHAR]() {
    return t('ui.string')
  },
  get [SysTableFieldType.TEXT]() {
    return t('ui.text')
  },
  get [SysTableFieldType.BOOLEAN]() {
    return t('ui.boolean')
  },
  get [SysTableFieldType.DATE]() {
    return t('ui.date')
  },
  get [SysTableFieldType.DATETIME]() {
    return t('ui.dateAndTime')
  },
  get [SysTableFieldType.TIME]() {
    return t('ui.time')
  },
  get [SysTableFieldType.SMALLINT]() {
    return t('ui.smallIntegerType')
  },
  [SysTableFieldType.JSON]: 'JSON',
  get [SysTableFieldType.INT]() {
    return t('ui.number')
  },
}

export enum SysMenuButtonPosition {
  LINE = 1,
  TOP,
  BOTTOM,
  FORM_TOP,
  FORM_BOTTOM,
  DETAIL_TOP,
  DETAIL_BOTTOM,
}

export const SysMenuButtonPositionMap = {
  get [SysMenuButtonPosition.LINE]() {
    return t('ui.rowButton')
  },
  get [SysMenuButtonPosition.TOP]() {
    return t('ui.topOfTable')
  },
  get [SysMenuButtonPosition.BOTTOM]() {
    return t('ui.bottomOfTable')
  },
  get [SysMenuButtonPosition.FORM_TOP]() {
    return t('ui.topOfForm')
  },
  get [SysMenuButtonPosition.FORM_BOTTOM]() {
    return t('ui.bottomOfForm')
  },
  get [SysMenuButtonPosition.DETAIL_TOP]() {
    return t('ui.topOfDetails')
  },
  get [SysMenuButtonPosition.DETAIL_BOTTOM]() {
    return t('ui.bottomOfDetails')
  },
}

export enum SysMenuButtonDisplayMode {
  AUTO = 'auto',
  ICON = 'icon',
  TEXT = 'text',
  ICON_TEXT = 'icon_text',
}

export const SysMenuButtonDisplayModeMap = {
  get [SysMenuButtonDisplayMode.AUTO]() {
    return t('ui.auto')
  },
  get [SysMenuButtonDisplayMode.ICON]() {
    return t('ui.iconsOnly')
  },
  get [SysMenuButtonDisplayMode.TEXT]() {
    return t('ui.textOnly')
  },
  get [SysMenuButtonDisplayMode.ICON_TEXT]() {
    return t('ui.iconText')
  },
}

export enum SysMenuButtonEventAction {
  QUERY = 'query',
  METADATA = 'metadata',
  DETAIL = 'detail',
  CREATE = 'create',
  CREATE_CHILD = 'create_child',
  UPDATE = 'update',
  DELETE = 'delete',
  REFRESH = 'refresh',
  BATCH_DELETE = 'batch_delete',
  COPY = 'copy',
  DUPLICATE = 'duplicate',
  EXPORT = 'export',
  NAVIGATE = 'navigate',
  CUSTOM = 'custom',
  SAVE = 'save',
  ORDER = 'order',
  REFRESH_CACHE = 'refresh_cache',
  TEST_EMAIL = 'test_email',
  CREATE_BUTTON = 'create_button',
  UPDATE_BUTTON = 'update_button',
  DELETE_BUTTON = 'delete_button',
  QUERY_BUTTON = 'query_button',
  BUTTON_METADATA = 'button_metadata',
  CREATE_ITEM = 'create_item',
  UPDATE_ITEM = 'update_item',
  DELETE_ITEM = 'delete_item',
  QUERY_ITEM = 'query_item',
  DETAIL_ITEM = 'detail_item',
  ITEM_METADATA = 'item_metadata',
  ASSIGN_PERMISSION = 'assign_permission',
  QUERY_USER_MENU = 'query_user_menu',
  QUERY_PERMISSION_MENU = 'query_permission_menu',
  RESET_PASSWORD = 'reset_password',
  UNLOCK_LOGIN = 'unlock_login',
  ROTATE_SECRET = 'rotate_secret',
  PUBLISH = 'publish',
  UNPUBLISH = 'unpublish',
  RUN = 'run',
  VERSION = 'version',
  DISABLE = 'disable',
  PUBLISH_MENU = 'publish_menu',
  UNPUBLISH_MENU = 'unpublish_menu',
  INIT_META = 'init_meta',
  SYNC_FIELDS = 'sync_fields',
  SYNC_INDEX = 'sync_index',
  FIELD_MANAGER = 'field_manager',
  CREATE_FIELD = 'create_field',
  UPDATE_FIELD = 'update_field',
  DELETE_FIELD = 'delete_field',
  QUERY_FIELD = 'query_field',
  DETAIL_FIELD = 'detail_field',
  CREATE_INDEX = 'create_index',
  UPDATE_INDEX = 'update_index',
  DELETE_INDEX = 'delete_index',
  QUERY_INDEX = 'query_index',
  DETAIL_INDEX = 'detail_index',
  CREATE_RELATION = 'create_relation',
  UPDATE_RELATION = 'update_relation',
  DELETE_RELATION = 'delete_relation',
  QUERY_RELATION = 'query_relation',
  DETAIL_RELATION = 'detail_relation',
}

export const SysMenuButtonEventActionMap = {
  get [SysMenuButtonEventAction.QUERY]() {
    return t('ui.query')
  },
  get [SysMenuButtonEventAction.METADATA]() {
    return t('ui.pageMetadata')
  },
  get [SysMenuButtonEventAction.DETAIL]() {
    return t('ui.details')
  },
  get [SysMenuButtonEventAction.CREATE]() {
    return t('ui.create')
  },
  get [SysMenuButtonEventAction.CREATE_CHILD]() {
    return t('ui.addSubclass')
  },
  get [SysMenuButtonEventAction.UPDATE]() {
    return t('ui.edit')
  },
  get [SysMenuButtonEventAction.DELETE]() {
    return t('ui.delete')
  },
  get [SysMenuButtonEventAction.REFRESH]() {
    return t('ui.refresh')
  },
  get [SysMenuButtonEventAction.BATCH_DELETE]() {
    return t('ui.batchDelete')
  },
  get [SysMenuButtonEventAction.COPY]() {
    return t('ui.copy')
  },
  get [SysMenuButtonEventAction.DUPLICATE]() {
    return t('ui.copyRecord')
  },
  get [SysMenuButtonEventAction.EXPORT]() {
    return t('ui.export')
  },
  get [SysMenuButtonEventAction.NAVIGATE]() {
    return t('ui.pageJump')
  },
  get [SysMenuButtonEventAction.CUSTOM]() {
    return t('ui.custom')
  },
  get [SysMenuButtonEventAction.SAVE]() {
    return t('ui.save')
  },
  get [SysMenuButtonEventAction.ORDER]() {
    return t('ui.sort')
  },
  get [SysMenuButtonEventAction.REFRESH_CACHE]() {
    return t('ui.refreshCache')
  },
  get [SysMenuButtonEventAction.TEST_EMAIL]() {
    return t('ui.testMail')
  },
  get [SysMenuButtonEventAction.CREATE_BUTTON]() {
    return t('ui.addButton')
  },
  get [SysMenuButtonEventAction.UPDATE_BUTTON]() {
    return t('ui.editButton')
  },
  get [SysMenuButtonEventAction.DELETE_BUTTON]() {
    return t('ui.removeButton')
  },
  get [SysMenuButtonEventAction.QUERY_BUTTON]() {
    return t('ui.buttonQuery')
  },
  get [SysMenuButtonEventAction.BUTTON_METADATA]() {
    return t('ui.buttonMetadata')
  },
  get [SysMenuButtonEventAction.CREATE_ITEM]() {
    return t('ui.addDictionaryEntry')
  },
  get [SysMenuButtonEventAction.UPDATE_ITEM]() {
    return t('ui.editDictionaryItems')
  },
  get [SysMenuButtonEventAction.DELETE_ITEM]() {
    return t('ui.removeDictionaryEntry')
  },
  get [SysMenuButtonEventAction.QUERY_ITEM]() {
    return t('ui.dictionaryEntryQueries')
  },
  get [SysMenuButtonEventAction.DETAIL_ITEM]() {
    return t('ui.dictionaryItemDetails')
  },
  get [SysMenuButtonEventAction.ITEM_METADATA]() {
    return t('ui.dictionaryItemMetadata')
  },
  get [SysMenuButtonEventAction.ASSIGN_PERMISSION]() {
    return t('ui.allocationOfCompetence')
  },
  get [SysMenuButtonEventAction.QUERY_USER_MENU]() {
    return t('ui.userMenuQuery')
  },
  get [SysMenuButtonEventAction.QUERY_PERMISSION_MENU]() {
    return t('ui.authorizedMenuQuery')
  },
  get [SysMenuButtonEventAction.RESET_PASSWORD]() {
    return t('ui.resetPassword')
  },
  get [SysMenuButtonEventAction.UNLOCK_LOGIN]() {
    return t('ui.unlock')
  },
  get [SysMenuButtonEventAction.ROTATE_SECRET]() {
    return t('ui.rotationKey')
  },
  get [SysMenuButtonEventAction.PUBLISH]() {
    return t('ui.publishAction')
  },
  get [SysMenuButtonEventAction.UNPUBLISH]() {
    return t('ui.cancelRelease')
  },
  get [SysMenuButtonEventAction.RUN]() {
    return t('ui.run')
  },
  get [SysMenuButtonEventAction.VERSION]() {
    return t('ui.version')
  },
  get [SysMenuButtonEventAction.DISABLE]() {
    return t('ui.disabled')
  },
  get [SysMenuButtonEventAction.PUBLISH_MENU]() {
    return t('ui.releaseToMenu')
  },
  get [SysMenuButtonEventAction.UNPUBLISH_MENU]() {
    return t('ui.cancelReleaseMenu')
  },
  get [SysMenuButtonEventAction.INIT_META]() {
    return t('ui.initializeMetadata')
  },
  get [SysMenuButtonEventAction.SYNC_FIELDS]() {
    return t('ui.syncFields')
  },
  get [SysMenuButtonEventAction.SYNC_INDEX]() {
    return t('ui.syncIndex')
  },
  get [SysMenuButtonEventAction.FIELD_MANAGER]() {
    return t('ui.fieldManagement')
  },
  get [SysMenuButtonEventAction.CREATE_FIELD]() {
    return t('ui.addField')
  },
  get [SysMenuButtonEventAction.UPDATE_FIELD]() {
    return t('ui.editFields')
  },
  get [SysMenuButtonEventAction.DELETE_FIELD]() {
    return t('ui.deleteField')
  },
  get [SysMenuButtonEventAction.QUERY_FIELD]() {
    return t('ui.fieldList')
  },
  get [SysMenuButtonEventAction.DETAIL_FIELD]() {
    return t('ui.fieldDetails')
  },
  get [SysMenuButtonEventAction.CREATE_INDEX]() {
    return t('ui.addIndex')
  },
  get [SysMenuButtonEventAction.UPDATE_INDEX]() {
    return t('ui.editIndex')
  },
  get [SysMenuButtonEventAction.DELETE_INDEX]() {
    return t('ui.deleteIndex')
  },
  get [SysMenuButtonEventAction.QUERY_INDEX]() {
    return t('ui.indexList')
  },
  get [SysMenuButtonEventAction.DETAIL_INDEX]() {
    return t('ui.indexDetails')
  },
  get [SysMenuButtonEventAction.CREATE_RELATION]() {
    return t('ui.addRelationship')
  },
  get [SysMenuButtonEventAction.UPDATE_RELATION]() {
    return t('ui.editRelations')
  },
  get [SysMenuButtonEventAction.DELETE_RELATION]() {
    return t('ui.removeRelationship')
  },
  get [SysMenuButtonEventAction.QUERY_RELATION]() {
    return t('ui.relationshipList')
  },
  get [SysMenuButtonEventAction.DETAIL_RELATION]() {
    return t('ui.relationshipDetails')
  },
}

export enum SysTableRelationType {
  ONE_TO_ONE = 1,
  ONE_TO_MANY,
  MANY_TO_ONE,
  MANY_TO_MANY,
}

export const SysTableRelationTypeMap = {
  get [SysTableRelationType.ONE_TO_ONE]() {
    return t('ui.oneOnOne')
  },
  get [SysTableRelationType.ONE_TO_MANY]() {
    return t('ui.morePairs')
  },
  get [SysTableRelationType.MANY_TO_ONE]() {
    return t('ui.oneMore')
  },
  get [SysTableRelationType.MANY_TO_MANY]() {
    return t('ui.multiplePairs')
  },
}

export enum SysTableFieldCategory {
  NORMAL = 'normal_field',
  VIRTUAL = 'virtual_field',
  CALCULATED = 'calculated_field',
}

export const SysTableFieldCategoryMap = {
  get [SysTableFieldCategory.NORMAL]() {
    return t('ui.normalFields')
  },
  get [SysTableFieldCategory.VIRTUAL]() {
    return t('ui.virtualColumn')
  },
  get [SysTableFieldCategory.CALCULATED]() {
    return t('ui.calculateFields')
  },
}
