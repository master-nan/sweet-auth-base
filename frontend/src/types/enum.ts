export enum SysTableType {
  SYSTEM = 1,
  VIEW,
}

export const SysTableTypeMap = {
  [SysTableType.SYSTEM]: '系统表',
  [SysTableType.VIEW]: '视图',
}

export enum SysMasterDetailMode {
  AUTO = 'auto',
  SUMMARY = 'summary',
  TABLE = 'table',
  STACKED = 'stacked',
}

export const SysMasterDetailModeMap = {
  [SysMasterDetailMode.AUTO]: '自动',
  [SysMasterDetailMode.SUMMARY]: '摘要主表',
  [SysMasterDetailMode.TABLE]: '主表表格',
  [SysMasterDetailMode.STACKED]: '上下主子表',
}

export enum SysFormOpenMode {
  AUTO = 'auto',
  DIALOG = 'dialog',
  PAGE = 'page',
}

export const SysFormOpenModeMap = {
  [SysFormOpenMode.AUTO]: '自动',
  [SysFormOpenMode.DIALOG]: '弹框',
  [SysFormOpenMode.PAGE]: '页签',
}

export enum SysDetailOpenMode {
  AUTO = 'auto',
  DIALOG = 'dialog',
  PAGE = 'page',
}

export const SysDetailOpenModeMap = {
  [SysDetailOpenMode.AUTO]: '自动',
  [SysDetailOpenMode.DIALOG]: '弹框',
  [SysDetailOpenMode.PAGE]: '页签',
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
  YREA_MONTH_PICKER,
  FILE_PICKER,
  BOOLEAN,
  JSON_EDITOR,
  ARRAY_INPUT,
  KEY_VALUE_EDITOR,
  CASCADER,
  RICH_TEXT,
}

export const SysTableFieldInputTypeMap = {
  [SysTableFieldInputType.INPUT]: '输入框',
  [SysTableFieldInputType.INPUT_NUMBER]: '数字输入',
  [SysTableFieldInputType.TEXTAREA]: '多行文本',
  [SysTableFieldInputType.SELECT]: '下拉选择',
  [SysTableFieldInputType.DATE_PICKER]: '日期选择',
  [SysTableFieldInputType.DATETIME_PICKER]: '日期时间',
  [SysTableFieldInputType.TIME_PICKER]: '时间选择',
  [SysTableFieldInputType.YEAR_PICKER]: '年份选择',
  [SysTableFieldInputType.YREA_MONTH_PICKER]: '年月选择',
  [SysTableFieldInputType.FILE_PICKER]: '文件选择',
  [SysTableFieldInputType.BOOLEAN]: '布尔开关',
  [SysTableFieldInputType.JSON_EDITOR]: 'JSON编辑器',
  [SysTableFieldInputType.ARRAY_INPUT]: '数组输入',
  [SysTableFieldInputType.KEY_VALUE_EDITOR]: '键值对编辑',
  [SysTableFieldInputType.CASCADER]: '级联选择',
  [SysTableFieldInputType.RICH_TEXT]: '富文本编辑器',
}

export enum ExpressionLogic {
  AND = 1,
  OR,
}

export const ExpressionLogicMap = {
  [ExpressionLogic.AND]: '与',
  [ExpressionLogic.OR]: '或',
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
  [ExpressionType.GT]: '大于',
  [ExpressionType.LT]: '小于',
  [ExpressionType.GTE]: '大于等于',
  [ExpressionType.LTE]: '小于等于',
  [ExpressionType.EQ]: '等于',
  [ExpressionType.NE]: '不等于',
  [ExpressionType.LIKE]: '包含',
  [ExpressionType.NOT_LIKE]: '不包含',
  [ExpressionType.IN]: '在里面',
  [ExpressionType.NOT_IN]: '不在里面',
  [ExpressionType.IS_NULL]: '空',
  [ExpressionType.IS_NOT_NULL]: '非空',
  [ExpressionType.BETWEEN]: '区间',
  [ExpressionType.NOT_BETWEEN]: '不在区间',
}

export enum SysTableFieldType {
  BIGINT = 1,
  FLOAT,
  VARCHAR,
  TEXT,
  BOOLEAN,
  DATE,
  DATETIME,
  TIME,
  TINYINT,
  JSON,
  INT,
}

export const SysTableFieldTypeMap = {
  [SysTableFieldType.BIGINT]: '大数字',
  [SysTableFieldType.FLOAT]: '浮点',
  [SysTableFieldType.VARCHAR]: '字符串',
  [SysTableFieldType.TEXT]: '文本',
  [SysTableFieldType.BOOLEAN]: '布尔',
  [SysTableFieldType.DATE]: '日期',
  [SysTableFieldType.DATETIME]: '日期时间',
  [SysTableFieldType.TIME]: '时间',
  [SysTableFieldType.TINYINT]: '微型整数',
  [SysTableFieldType.JSON]: 'JSON',
  [SysTableFieldType.INT]: '数字',
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
  [SysMenuButtonPosition.LINE]: '行按钮',
  [SysMenuButtonPosition.TOP]: '表格顶部',
  [SysMenuButtonPosition.BOTTOM]: '表格底部',
  [SysMenuButtonPosition.FORM_TOP]: '表单顶部',
  [SysMenuButtonPosition.FORM_BOTTOM]: '表单底部',
  [SysMenuButtonPosition.DETAIL_TOP]: '详情顶部',
  [SysMenuButtonPosition.DETAIL_BOTTOM]: '详情底部',
}

export enum SysMenuButtonDisplayMode {
  AUTO = 'auto',
  ICON = 'icon',
  TEXT = 'text',
  ICON_TEXT = 'icon_text',
}

export const SysMenuButtonDisplayModeMap = {
  [SysMenuButtonDisplayMode.AUTO]: '自动',
  [SysMenuButtonDisplayMode.ICON]: '仅图标',
  [SysMenuButtonDisplayMode.TEXT]: '仅文字',
  [SysMenuButtonDisplayMode.ICON_TEXT]: '图标文字',
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
  [SysMenuButtonEventAction.QUERY]: '查询',
  [SysMenuButtonEventAction.METADATA]: '页面元数据',
  [SysMenuButtonEventAction.DETAIL]: '详情',
  [SysMenuButtonEventAction.CREATE]: '新增',
  [SysMenuButtonEventAction.CREATE_CHILD]: '新增子级',
  [SysMenuButtonEventAction.UPDATE]: '编辑',
  [SysMenuButtonEventAction.DELETE]: '删除',
  [SysMenuButtonEventAction.REFRESH]: '刷新',
  [SysMenuButtonEventAction.BATCH_DELETE]: '批量删除',
  [SysMenuButtonEventAction.COPY]: '复制',
  [SysMenuButtonEventAction.DUPLICATE]: '复制记录',
  [SysMenuButtonEventAction.EXPORT]: '导出',
  [SysMenuButtonEventAction.NAVIGATE]: '页面跳转',
  [SysMenuButtonEventAction.CUSTOM]: '自定义',
  [SysMenuButtonEventAction.SAVE]: '保存',
  [SysMenuButtonEventAction.ORDER]: '排序',
  [SysMenuButtonEventAction.REFRESH_CACHE]: '刷新缓存',
  [SysMenuButtonEventAction.TEST_EMAIL]: '测试邮件',
  [SysMenuButtonEventAction.CREATE_BUTTON]: '新增按钮',
  [SysMenuButtonEventAction.UPDATE_BUTTON]: '编辑按钮',
  [SysMenuButtonEventAction.DELETE_BUTTON]: '删除按钮',
  [SysMenuButtonEventAction.QUERY_BUTTON]: '按钮查询',
  [SysMenuButtonEventAction.BUTTON_METADATA]: '按钮元数据',
  [SysMenuButtonEventAction.CREATE_ITEM]: '新增字典项',
  [SysMenuButtonEventAction.UPDATE_ITEM]: '编辑字典项',
  [SysMenuButtonEventAction.DELETE_ITEM]: '删除字典项',
  [SysMenuButtonEventAction.QUERY_ITEM]: '字典项查询',
  [SysMenuButtonEventAction.DETAIL_ITEM]: '字典项详情',
  [SysMenuButtonEventAction.ITEM_METADATA]: '字典项元数据',
  [SysMenuButtonEventAction.ASSIGN_PERMISSION]: '分配权限',
  [SysMenuButtonEventAction.QUERY_USER_MENU]: '用户菜单查询',
  [SysMenuButtonEventAction.QUERY_PERMISSION_MENU]: '授权菜单查询',
  [SysMenuButtonEventAction.RESET_PASSWORD]: '重置密码',
  [SysMenuButtonEventAction.UNLOCK_LOGIN]: '解除锁定',
  [SysMenuButtonEventAction.ROTATE_SECRET]: '轮换密钥',
  [SysMenuButtonEventAction.PUBLISH]: '发布',
  [SysMenuButtonEventAction.UNPUBLISH]: '取消发布',
  [SysMenuButtonEventAction.RUN]: '运行',
  [SysMenuButtonEventAction.VERSION]: '版本',
  [SysMenuButtonEventAction.DISABLE]: '停用',
  [SysMenuButtonEventAction.PUBLISH_MENU]: '发布到菜单',
  [SysMenuButtonEventAction.UNPUBLISH_MENU]: '取消发布菜单',
  [SysMenuButtonEventAction.INIT_META]: '初始化元数据',
  [SysMenuButtonEventAction.SYNC_FIELDS]: '同步字段',
  [SysMenuButtonEventAction.SYNC_INDEX]: '同步索引',
  [SysMenuButtonEventAction.FIELD_MANAGER]: '字段管理',
  [SysMenuButtonEventAction.CREATE_FIELD]: '新增字段',
  [SysMenuButtonEventAction.UPDATE_FIELD]: '编辑字段',
  [SysMenuButtonEventAction.DELETE_FIELD]: '删除字段',
  [SysMenuButtonEventAction.QUERY_FIELD]: '字段列表',
  [SysMenuButtonEventAction.DETAIL_FIELD]: '字段详情',
  [SysMenuButtonEventAction.CREATE_INDEX]: '新增索引',
  [SysMenuButtonEventAction.UPDATE_INDEX]: '编辑索引',
  [SysMenuButtonEventAction.DELETE_INDEX]: '删除索引',
  [SysMenuButtonEventAction.QUERY_INDEX]: '索引列表',
  [SysMenuButtonEventAction.DETAIL_INDEX]: '索引详情',
  [SysMenuButtonEventAction.CREATE_RELATION]: '新增关系',
  [SysMenuButtonEventAction.UPDATE_RELATION]: '编辑关系',
  [SysMenuButtonEventAction.DELETE_RELATION]: '删除关系',
  [SysMenuButtonEventAction.QUERY_RELATION]: '关系列表',
  [SysMenuButtonEventAction.DETAIL_RELATION]: '关系详情',
}

export enum SysTableRelationType {
  ONE_TO_ONE = 1,
  ONE_TO_MANY,
  MANY_TO_ONE,
  MANY_TO_MANY,
}

export const SysTableRelationTypeMap = {
  [SysTableRelationType.ONE_TO_ONE]: '一对一',
  [SysTableRelationType.ONE_TO_MANY]: '一对多',
  [SysTableRelationType.MANY_TO_ONE]: '多对一',
  [SysTableRelationType.MANY_TO_MANY]: '多对多',
}

export enum SysTableFieldCategory {
  NORMAL = 'normal_field',
  VIRTUAL = 'virtual_field',
  CALCULATED = 'calculated_field',
}

export const SysTableFieldCategoryMap = {
  [SysTableFieldCategory.NORMAL]: '普通字段',
  [SysTableFieldCategory.VIRTUAL]: '虚拟列',
  [SysTableFieldCategory.CALCULATED]: '计算字段',
}
