package model

import (
	"backend/enum"
	"database/sql"
)

type SysConfigure struct {
	Basic
	// 安全配置
	EnableCaptcha bool `gorm:"comment:登录验证码" json:"enable_captcha"`
	//密码长度
	PasswordLength int `gorm:"default:8;comment:密码长度" json:"password_length"`
	//密码复杂度
	PasswordComplexity int `gorm:"default:2;comment:密码复杂度" json:"password_complexity"`
	//密码过期时间
	PasswordExpireTime int `gorm:"default:90;comment:密码过期时间" json:"password_expire_time"`
	//密码错误次数
	PasswordErrorCount int `gorm:"default:5;comment:密码错误次数" json:"password_error_count"`
	//密码错误锁定时长（分钟）
	PasswordLockMinutes int `gorm:"default:15;comment:密码错误锁定时长分钟" json:"password_lock_minutes"`
	// 默认密码策略
	PasswordPolicy string `gorm:"size:256;comment:默认密码策略" json:"password_policy"`

	// 系统基本信息
	SystemName        string `gorm:"size:64;comment:系统名称" json:"system_name"`
	SystemVersion     string `gorm:"size:32;comment:系统版本" json:"system_version"`
	SystemLogo        string `gorm:"size:256;comment:系统Logo" json:"system_logo"`
	SystemDescription string `gorm:"size:512;comment:系统描述" json:"system_description"`

	// 邮件配置
	EnableEmail    bool   `gorm:"default:false;comment:启用邮件服务" json:"enable_email"`
	SmtpServer     string `gorm:"size:128;comment:SMTP服务器" json:"smtp_server"`
	SmtpPort       int    `gorm:"default:465;comment:SMTP端口" json:"smtp_port"`
	SenderEmail    string `gorm:"size:128;comment:发件人邮箱" json:"sender_email"`
	SenderPassword string `gorm:"size:128;comment:发件人密码" json:"sender_password"`
}

type SysMenu struct {
	Basic
	Pid            int                    `gorm:"type:int" json:"pid"`
	Name           string                 `gorm:"size:32;comment:路由" json:"name"`
	Path           string                 `gorm:"size:128;comment:路径" json:"path"`
	Component      string                 `gorm:"size:64;comment:路由主体" json:"component"`
	Title          string                 `gorm:"size:64;comment:显示标题" json:"title"`
	IsHidden       bool                   `gorm:"default:false;comment:是否隐藏" json:"is_hidden"`
	Sequence       uint8                  `gorm:"comment:排序;type:smallint" json:"sequence"`
	PageType       enum.SysMenuPageType   `gorm:"size:32;comment:页面类型" json:"page_type"`
	TableCode      string                 `gorm:"size:128;comment:绑定表编码" json:"table_code"`
	Option         string                 `gorm:"type:text;comment:扩展配置" json:"option"`
	Icon           *string                `gorm:"size:32;comment:图标" json:"icon"`
	Redirect       *string                `gorm:"size:128;comment:重定向地址" json:"redirect"`
	IsUnfold       bool                   `gorm:"default:false;comment:默认展开" json:"is_unfold"`
	DetailOpenMode enum.SysDetailOpenMode `gorm:"-" json:"detail_open_mode,omitempty"`
	MenuButtons    []SysMenuButton        `gorm:"foreignKey:MenuId;references:Id" json:"menu_buttons"`
	Roles          []SysRole              `gorm:"many2many:sys_role_menu" json:"roles"`
	Children       []SysMenu              `gorm:"-" json:"children"` // 子菜单
}

type SysMenuButton struct {
	Basic
	MenuId       int                           `gorm:"comment:menu_id" json:"menu_id" binding:"required"`
	Name         string                        `gorm:"size:128;comment:按钮名称" json:"name" binding:"required"`
	Code         string                        `gorm:"size:128;comment:按钮编码" json:"code" binding:"required"`
	Memo         string                        `gorm:"size:128;comment:备注" json:"memo"`
	Position     enum.SysMenuButtonPosition    `gorm:"type:smallint;default:1;comment:位置" json:"position" binding:"required"`
	EventType    string                        `gorm:"size:64;comment:事件类型" json:"event_type"`
	EventAction  string                        `gorm:"size:256;comment:事件动作" json:"event_action"`
	Icon         string                        `gorm:"size:32;comment:图标" json:"icon"`
	Color        string                        `gorm:"size:32;comment:颜色" json:"color"`
	DisplayMode  enum.SysMenuButtonDisplayMode `gorm:"size:16;default:auto;comment:展示方式" json:"display_mode"`
	Sequence     uint8                         `gorm:"comment:排序;type:smallint" json:"sequence"`
	Path         string                        `gorm:"size:256;comment:接口路径" json:"api_path"`
	Method       string                        `gorm:"size:16;comment:请求方法" json:"http_method"`
	ParamsSchema string                        `gorm:"type:text;comment:参数Schema" json:"params_schema"`
	ConfirmText  string                        `gorm:"size:256;comment:确认提示" json:"confirm_text"`
	DisableWhen  string                        `gorm:"type:text;comment:禁用条件" json:"disable_when"`
	IsButton     bool                          `gorm:"default:true;comment:是否页面按钮" json:"is_button"`
	IsHidden     bool                          `gorm:"default:false;comment:是否隐藏" json:"is_hidden"`
	IsDisabled   bool                          `gorm:"default:false;comment:是否禁用" json:"is_disabled"`
	BeforeHooks  string                        `gorm:"type:text;comment:前置钩子JSON" json:"before_hooks"` // 执行事件前的钩子函数
	AfterHooks   string                        `gorm:"type:text;comment:后置钩子JSON" json:"after_hooks"`  // 执行事件后的钩子函数
	Roles        []SysRole                     `gorm:"many2many:sys_role_menu_button;foreignKey:Id;joinForeignKey:ButtonId;References:Id;joinReferences:RoleId" json:"roles"`
	Menus        []SysMenu                     `gorm:"many2many:sys_role_menu_button;foreignKey:Id;joinForeignKey:ButtonId;References:Id;joinReferences:MenuId" json:"menus"`
}

func (b SysMenuButton) IsPageButton() bool {
	return b.IsButton
}

type SysMenuButtonTemplate struct {
	Basic
	Scene        string                        `gorm:"size:64;uniqueIndex:uni_button_template_scene_code_suffix;comment:模板场景" json:"scene"`
	Name         string                        `gorm:"size:128;comment:按钮名称" json:"name"`
	CodeSuffix   string                        `gorm:"size:64;uniqueIndex:uni_button_template_scene_code_suffix;comment:按钮编码后缀" json:"code_suffix"`
	Memo         string                        `gorm:"size:128;comment:备注" json:"memo"`
	Position     enum.SysMenuButtonPosition    `gorm:"type:smallint;default:1;comment:位置" json:"position"`
	EventType    string                        `gorm:"size:64;comment:事件类型" json:"event_type"`
	EventAction  string                        `gorm:"size:256;comment:事件动作" json:"event_action"`
	Icon         string                        `gorm:"size:32;comment:图标" json:"icon"`
	Color        string                        `gorm:"size:32;comment:颜色" json:"color"`
	DisplayMode  enum.SysMenuButtonDisplayMode `gorm:"size:16;default:auto;comment:展示方式" json:"display_mode"`
	Sequence     uint8                         `gorm:"comment:排序;type:smallint" json:"sequence"`
	Path         string                        `gorm:"size:256;comment:接口路径" json:"api_path"`
	Method       string                        `gorm:"size:16;comment:请求方法" json:"http_method"`
	ParamsSchema string                        `gorm:"type:text;comment:参数Schema" json:"params_schema"`
	ConfirmText  string                        `gorm:"size:256;comment:确认提示" json:"confirm_text"`
	DisableWhen  string                        `gorm:"type:text;comment:禁用条件" json:"disable_when"`
	IsButton     bool                          `gorm:"default:true;comment:是否页面按钮" json:"is_button"`
	IsDisabled   bool                          `gorm:"default:false;comment:是否禁用" json:"is_disabled"`
	BeforeHooks  string                        `gorm:"type:text;comment:前置钩子JSON" json:"before_hooks"`
	AfterHooks   string                        `gorm:"type:text;comment:后置钩子JSON" json:"after_hooks"`
}

type SysUser struct {
	Basic
	UserName          string      `gorm:"size:128;uniqueIndex:uni_user_name;comment:用户名" json:"user_name"`
	Password          string      `gorm:"size:128;comment:密码" json:"password"`
	Email             string      `gorm:"size:128;index:index_email;comment:邮箱" json:"email"`
	PhoneNumber       string      `gorm:"size:128;index:index_phone_number;comment:电话" json:"phone_number"`
	GmtLastLogin      *CustomTime `gorm:"type:timestamp;comment:最后登录时间" json:"gmt_last_login"`
	PasswordChangedAt *CustomTime `gorm:"type:timestamp;comment:密码最后修改时间" json:"password_changed_at"`
	Language          string      `gorm:"size:32;comment:语言包" json:"language"`
	AccessTokens      string      `gorm:"type:text;comment:用户最近5次Token" json:"access_tokens"`
	Roles             []SysRole   `gorm:"many2many:sys_user_role;foreignKey:Id;joinForeignKey:UserId;References:Id;joinReferences:RoleId" json:"roles"`
	IsReset           bool        `gorm:"default:true" json:"is_reset"`
}

type SysDataDimension struct {
	Basic
	Code        string `gorm:"size:64;uniqueIndex:uni_data_dimension_code;comment:维度编码" json:"code"`
	Name        string `gorm:"size:128;comment:维度名称" json:"name"`
	ValueType   string `gorm:"size:32;default:string;comment:值类型" json:"value_type"`
	SourceType  string `gorm:"size:32;default:none;comment:来源类型" json:"source_type"`
	SourceCode  string `gorm:"size:128;comment:来源编码" json:"source_code"`
	LabelField  string `gorm:"size:128;comment:展示字段" json:"label_field"`
	ValueField  string `gorm:"size:128;comment:值字段" json:"value_field"`
	ParentField string `gorm:"size:128;comment:父级字段" json:"parent_field"`
	Memo        string `gorm:"size:256;comment:备注" json:"memo"`
}

type SysDataScopeBinding struct {
	Basic
	MenuId        int              `gorm:"index:idx_data_scope_binding_menu;comment:菜单ID" json:"menu_id"`
	TableCode     string           `gorm:"size:128;index:idx_data_scope_binding_table;comment:表编码" json:"table_code"`
	DimensionCode string           `gorm:"size:64;index:idx_data_scope_binding_dimension;comment:维度编码" json:"dimension_code"`
	FieldCode     string           `gorm:"size:128;comment:表字段编码" json:"field_code"`
	MatchType     string           `gorm:"size:32;default:in;comment:匹配方式" json:"match_type"`
	Required      bool             `gorm:"default:true;comment:是否必配授权" json:"required"`
	Actions       string           `gorm:"type:text;comment:生效动作JSON" json:"-"`
	ActionList    []string         `gorm:"-" json:"actions"`
	Menu          SysMenu          `gorm:"foreignKey:MenuId;references:Id" json:"menu,omitempty"`
	Dimension     SysDataDimension `gorm:"foreignKey:DimensionCode;references:Code" json:"dimension,omitempty"`
}

type SysRoleDataScope struct {
	Basic
	RoleId         int              `gorm:"index:idx_role_data_scope_role;comment:角色ID" json:"role_id"`
	MenuId         int              `gorm:"index:idx_role_data_scope_menu;comment:菜单ID" json:"menu_id"`
	TableCode      string           `gorm:"size:128;index:idx_role_data_scope_table;comment:表编码" json:"table_code"`
	DimensionCode  string           `gorm:"size:64;index:idx_role_data_scope_dimension;comment:维度编码" json:"dimension_code"`
	Strategy       string           `gorm:"size:32;default:none;comment:范围策略" json:"strategy"`
	ScopeValues    string           `gorm:"type:text;comment:范围值JSON" json:"-"`
	ScopeValueList []string         `gorm:"-" json:"scope_values"`
	Role           SysRole          `gorm:"foreignKey:RoleId;references:Id" json:"role,omitempty"`
	Menu           SysMenu          `gorm:"foreignKey:MenuId;references:Id" json:"menu,omitempty"`
	Dimension      SysDataDimension `gorm:"foreignKey:DimensionCode;references:Code" json:"dimension,omitempty"`
}

type SysUserDataScopeOverride struct {
	Basic
	UserId         int              `gorm:"index:idx_user_data_scope_user;comment:用户ID" json:"user_id"`
	MenuId         int              `gorm:"index:idx_user_data_scope_menu;comment:菜单ID" json:"menu_id"`
	TableCode      string           `gorm:"size:128;index:idx_user_data_scope_table;comment:表编码" json:"table_code"`
	DimensionCode  string           `gorm:"size:64;index:idx_user_data_scope_dimension;comment:维度编码" json:"dimension_code"`
	Strategy       string           `gorm:"size:32;default:none;comment:范围策略" json:"strategy"`
	ScopeValues    string           `gorm:"type:text;comment:范围值JSON" json:"-"`
	ScopeValueList []string         `gorm:"-" json:"scope_values"`
	OverrideMode   string           `gorm:"size:32;default:replace;comment:覆盖模式" json:"override_mode"`
	ExpireAt       *CustomTime      `gorm:"type:timestamp;comment:过期时间" json:"expire_at"`
	User           SysUser          `gorm:"foreignKey:UserId;references:Id" json:"user,omitempty"`
	Menu           SysMenu          `gorm:"foreignKey:MenuId;references:Id" json:"menu,omitempty"`
	Dimension      SysDataDimension `gorm:"foreignKey:DimensionCode;references:Code" json:"dimension,omitempty"`
}

type SysUserDimensionValue struct {
	Basic
	UserId         int              `gorm:"index:idx_user_dimension_value_user;comment:用户ID" json:"user_id"`
	DimensionCode  string           `gorm:"size:64;index:idx_user_dimension_value_dimension;comment:维度编码" json:"dimension_code"`
	ScopeValues    string           `gorm:"type:text;comment:维度值JSON" json:"-"`
	ScopeValueList []string         `gorm:"-" json:"scope_values"`
	User           SysUser          `gorm:"foreignKey:UserId;references:Id" json:"user,omitempty"`
	Dimension      SysDataDimension `gorm:"foreignKey:DimensionCode;references:Code" json:"dimension,omitempty"`
}

type SysUserRole struct {
	UserId int `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	RoleId int `gorm:"primaryKey;autoIncrement:false" json:"role_id"`
}

type SysRole struct {
	Basic
	Name    string          `gorm:"size:128;comment:角色名称" json:"name"`
	Memo    string          `gorm:"size:128;comment:备注" json:"memo"`
	Menus   []SysMenu       `gorm:"many2many:sys_role_menu;foreignKey:Id;joinForeignKey:RoleId;References:Id;joinReferences:MenuId" json:"menus"`
	Buttons []SysMenuButton `gorm:"many2many:sys_role_menu_button;foreignKey:Id;joinForeignKey:RoleId;References:Id;joinReferences:ButtonId" json:"buttons"`
	Users   []SysUser       `gorm:"many2many:sys_user_role;foreignKey:Id;joinForeignKey:RoleId;References:Id;joinReferences:UserId" json:"users"`
}

type SysRoleMenu struct {
	RoleId int `gorm:"primaryKey;autoIncrement:false" json:"role_id"`
	MenuId int `gorm:"primaryKey;autoIncrement:false" json:"menu_id"`
}

type SysRoleMenuButton struct {
	RoleId   int `gorm:"primaryKey;autoIncrement:false" json:"role_id"`
	MenuId   int `gorm:"primaryKey;autoIncrement:false" json:"menu_id"`
	ButtonId int `gorm:"primaryKey;autoIncrement:false" json:"button_id"`
}

type SysTable struct {
	Basic
	TableName        string                   `gorm:"size:128;comment:表名" json:"table_name"`
	TableCode        string                   `gorm:"size:128;uniqueIndex:uni_table_code_index;comment:数据库中表名" json:"table_code"`
	TableType        enum.SysTableType        `gorm:"type:smallint;default:1;comment:表类型" json:"table_type"`
	MasterDetailMode enum.SysMasterDetailMode `gorm:"size:16;default:auto;comment:主子表展示模式" json:"master_detail_mode"`
	FormOpenMode     enum.SysFormOpenMode     `gorm:"size:16;default:auto;comment:表单打开方式" json:"form_open_mode"`
	DetailOpenMode   enum.SysDetailOpenMode   `gorm:"size:16;default:auto;comment:详情打开方式" json:"detail_open_mode"`
	ParentId         int                      `gorm:"comment:父节点Id" json:"parent_id"`
	SQL              string                   `gorm:"type:text;comment:视图定义SQL" json:"sql"`
	TableFields      []SysTableField          `gorm:"foreignKey:TableId;references:Id" json:"table_fields"`
	TableRelations   []SysTableRelation       `gorm:"foreignKey:TableId" json:"table_relations" `
	TableIndexes     []SysTableIndex          `gorm:"foreignKey:TableId" json:"table_indexes"`
}

type SysTableField struct {
	Basic
	TableId            int                         `gorm:"comment:table_id;uniqueIndex:union_uni_table_id_field_code_index" json:"table_id" binding:"required"`
	FieldName          string                      `gorm:"size:128;comment:列名" json:"field_name"`
	FieldCode          string                      `gorm:"size:128;uniqueIndex:union_uni_table_id_field_code_index;comment:表字段名" json:"field_code"`
	FieldType          enum.SysTableFieldType      `gorm:"type:smallint;default:1;comment:字段类型" json:"field_type"`
	FieldLength        int                         `gorm:"default:0;comment:字段长度" json:"field_length"`
	FieldDecimalLength int                         `gorm:"default:0;comment:小数位数" json:"field_decimal_length"`
	InputType          enum.SysTableFieldInputType `gorm:"type:smallint;default:1;comment:输入类型" json:"input_type"`
	FormSpan           uint8                       `gorm:"type:smallint;default:0;comment:表单占位列数，0为自动" json:"form_span"`
	DetailSpan         uint8                       `gorm:"type:smallint;default:0;comment:详情占位列数，0为自动" json:"detail_span"`
	DefaultValue       *string                     `gorm:"size:128;comment:默认值" json:"default_value,omitempty"`
	DictCode           *string                     `gorm:"size:128;comment:所用字典" json:"dict_code"`
	Dict               SysDict                     `gorm:"foreignKey:DictCode;references:DictCode" json:"dict,omitempty"`
	IsPrimaryKey       bool                        `gorm:"default:false;comment:是否主键" json:"is_primary_key"`
	IsIndex            bool                        `gorm:"default:false;comment:是否索引" json:"is_index"`
	IsQuickSearch      bool                        `gorm:"default:false;comment:是否快捷搜索" json:"is_quick_search"`
	IsAdvancedSearch   bool                        `gorm:"default:false;comment:是否高级搜索" json:"is_advanced_search"`
	IsSort             bool                        `gorm:"default:false;comment:是否可排序" json:"is_sort"`
	IsNull             bool                        `gorm:"default:true;comment:是否可空" json:"is_null"`
	IsListShow         bool                        `gorm:"default:true;comment:是否列表显示" json:"is_list_show"`
	IsInsertShow       bool                        `gorm:"default:true;comment:是否新增显示" json:"is_insert_show"`
	IsUpdateShow       bool                        `gorm:"default:true;comment:是否更新显示" json:"is_update_show"`
	Sequence           uint8                       `gorm:"comment:排序;type:smallint" binding:"required" json:"sequence"`
	OriginalFieldId    int                         `gorm:"comment:原字段Id" json:"original_field_id"`
	Binding            string                      `gorm:"size:256;comment:验证器" json:"binding"`        // 用于存储绑定规则
	FieldCategory      enum.SysTableFieldCategory  `gorm:"size:64;comment:字段类别" json:"field_category"` // 字段类别（普通字段、虚拟列、计算字段）
	Expression         *string                     `gorm:"size:256;comment:计算字段表达式" json:"expression"` // 计算字段表达式或虚拟列表达式
	Tag                *string                     `gorm:"size:256;comment:标签" json:"tag"`
	LinkageConfig      *string                     `gorm:"type:text;comment:联动配置" json:"linkage_config"`
}

type SysTableIndex struct {
	Basic
	TableId     int             `gorm:"index;comment:表Id" json:"table_id"`
	IndexName   string          `gorm:"size:128;comment:索引名称" json:"index_name"`
	IsUnique    bool            `gorm:"comment:是否唯一索引" json:"is_unique"`
	IndexFields []SysTableField `gorm:"many2many:sys_table_index_field;foreignKey:Id;joinForeignKey:IndexId;References:Id;joinReferences:FieldId" json:"index_fields"`
}

type SysTableIndexField struct {
	IndexId int `gorm:"primaryKey;autoIncrement:false" json:"index_id"`
	FieldId int `gorm:"primaryKey;autoIncrement:false" json:"field_id"`
}

type SysTableRelation struct {
	Basic
	TableId        int                       `gorm:"index;comment:主表Id" json:"table_id"`
	RelatedTableId int                       `gorm:"index;comment:关联表Id" json:"related_table_id"` // 关联的表的Id
	ReferenceKey   string                    `gorm:"size:128;comment:主表字段" json:"reference_key"`  // 主表对应字段
	ForeignKey     string                    `gorm:"size:128;comment:关联表字段" json:"foreign_key"`   // 关联表 字段
	OnDelete       string                    `gorm:"size:128;comment:删除时策略" json:"on_delete"`
	OnUpdate       string                    `gorm:"size:128;comment:更新时策略" json:"on_update"`
	RelationType   enum.SysTableRelationType `gorm:"size:128;comment:关系类型" json:"relation_type"`
	ManyTableCode  string                    `gorm:"size:128;comment:多对多关系中间表" json:"many_table_code"` // 多对多关系使用到的中间表
}

type SysDict struct {
	Basic
	DictName  string        `gorm:"size:128;comment:字典名称;uniqueIndex:uni_dict_name_index" json:"dict_name"`
	DictCode  string        `gorm:"size:128;comment:字典编码;uniqueIndex:uni_dict_code_index" json:"dict_code"`
	DictItems []SysDictItem `gorm:"foreignKey:DictId;references:Id" json:"dict_items"`
}

type SysDictItem struct {
	Basic
	DictId    int    `gorm:"comment:dict_id" json:"dict_id"`
	ItemName  string `gorm:"size:128;comment:字典名称" json:"item_name"`
	ItemCode  string `gorm:"size:128;comment:字典编码;uniqueIndex:uni_item_code_index" json:"item_code"`
	ItemValue string `gorm:"size:128;comment:字典值" json:"item_value"`
}

type TableColumnMate struct {
	ColumnName             string         `gorm:"column:COLUMN_NAME" json:"column_name"`                           // 列名
	OrdinalPosition        int            `gorm:"column:ORDINAL_POSITION" json:"ordinal_position"`                 // 列名所在位置，即排序-
	ColumnDefault          sql.NullString `gorm:"column:COLUMN_DEFAULT" json:"column_default"`                     // 默认值
	IsNullable             string         `gorm:"column:IS_NULLABLE" json:"is_nullable"`                           // 是否可以为null
	DataType               string         `gorm:"column:DATA_TYPE" json:"data_type"`                               // 数据类型
	CharacterMaximumLength sql.NullInt64  `gorm:"column:CHARACTER_MAXIMUM_LENGTH" json:"character_maximum_length"` // 字符类型最大长度
	CharacterOctetLength   sql.NullInt64  `gorm:"column:CHARACTER_OCTET_LENGTH" json:"character_octet_length"`     // 字符类型最大字节长度
	NumericPrecision       sql.NullInt64  `gorm:"column:NUMERIC_PRECISION" json:"numeric_precision"`               // 数值型类的精度
	NumericScale           sql.NullInt64  `gorm:"column:NUMERIC_SCALE" json:"numeric_scale"`                       // 数值型列的小数位数
	DatetimePrecision      sql.NullInt64  `gorm:"column:DATETIME_PRECISION" json:"datetime_precision"`             // 日期时间型列的精度
	ColumnType             string         `gorm:"column:COLUMN_TYPE" json:"column_type"`                           // 列的完整类型描述
	ColumnKey              string         `gorm:"column:COLUMN_KEY" json:"column_key"`                             // 列是否被索引。
	Extra                  string         `gorm:"column:EXTRA" json:"extra"`                                       // 其他信息，是否自增
	ColumnComment          string         `gorm:"column:COLUMN_COMMENT" json:"column_comment"`                     // 列备注
}

type TableIndexMate struct {
	ColumnName string `gorm:"column:COLUMN_NAME" json:"column_name"` // 列名
	IndexName  string `gorm:"column:INDEX_NAME" json:"index_name"`   // 索引名称
	NonUnique  bool   `gorm:"column:NON_UNIQUE" json:"non_unique"`   // 是否唯一索引
}
