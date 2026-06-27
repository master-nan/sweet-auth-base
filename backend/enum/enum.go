/**
 * @Author: Nan
 * @Date: 2023/9/7 16:25
 */

package enum

import (
	"encoding/json"
	"errors"
	"strings"
)

// DataPermissionsEnum 数据权限字典
type DataPermissionsEnum uint8

const (
	Whole   DataPermissionsEnum = iota + 1 //全部
	Custom                                 //自定义
	Tacitly                                // 默认
)

// SysMenuPageType 菜单页面类型
type SysMenuPageType string

const (
	MenuPageTypeDirectory SysMenuPageType = "directory"
	MenuPageTypeFixed     SysMenuPageType = "fixed"
	MenuPageTypeLowCode   SysMenuPageType = "low_code"
)

// SysMenuButtonPosition 按钮位置字典
type SysMenuButtonPosition uint8

const (
	Line         SysMenuButtonPosition = iota + 1 // 行按钮
	Top                                           // 表格顶部
	Bottom                                        // 表格底部
	FormTop                                       // 表单顶部
	FormBottom                                    // 表单底部
	DetailTop                                     // 详情顶部
	DetailBottom                                  // 详情底部
)

// SysMenuButtonDisplayMode 按钮展示方式字典
type SysMenuButtonDisplayMode string

const (
	ButtonDisplayAuto     SysMenuButtonDisplayMode = "auto"
	ButtonDisplayIcon     SysMenuButtonDisplayMode = "icon"
	ButtonDisplayText     SysMenuButtonDisplayMode = "text"
	ButtonDisplayIconText SysMenuButtonDisplayMode = "icon_text"
)

func NormalizeSysMenuButtonDisplayMode(value string) (SysMenuButtonDisplayMode, bool) {
	mode := SysMenuButtonDisplayMode(strings.ToLower(strings.TrimSpace(value)))
	if mode == "" {
		return ButtonDisplayAuto, true
	}
	switch mode {
	case ButtonDisplayAuto, ButtonDisplayIcon, ButtonDisplayText, ButtonDisplayIconText:
		return mode, true
	default:
		return "", false
	}
}

// SysMenuButtonEventAction 菜单按钮事件动作字典
type SysMenuButtonEventAction string

const (
	ButtonActionQuery            SysMenuButtonEventAction = "query"
	ButtonActionMetadata         SysMenuButtonEventAction = "metadata"
	ButtonActionDetail           SysMenuButtonEventAction = "detail"
	ButtonActionCreate           SysMenuButtonEventAction = "create"
	ButtonActionCreateChild      SysMenuButtonEventAction = "create_child"
	ButtonActionUpdate           SysMenuButtonEventAction = "update"
	ButtonActionDelete           SysMenuButtonEventAction = "delete"
	ButtonActionRefresh          SysMenuButtonEventAction = "refresh"
	ButtonActionBatchDelete      SysMenuButtonEventAction = "batch_delete"
	ButtonActionCopy             SysMenuButtonEventAction = "copy"
	ButtonActionDuplicate        SysMenuButtonEventAction = "duplicate"
	ButtonActionExport           SysMenuButtonEventAction = "export"
	ButtonActionNavigate         SysMenuButtonEventAction = "navigate"
	ButtonActionCustom           SysMenuButtonEventAction = "custom"
	ButtonActionSave             SysMenuButtonEventAction = "save"
	ButtonActionOrder            SysMenuButtonEventAction = "order"
	ButtonActionRefreshCache     SysMenuButtonEventAction = "refresh_cache"
	ButtonActionTestEmail        SysMenuButtonEventAction = "test_email"
	ButtonActionCreateButton     SysMenuButtonEventAction = "create_button"
	ButtonActionUpdateButton     SysMenuButtonEventAction = "update_button"
	ButtonActionDeleteButton     SysMenuButtonEventAction = "delete_button"
	ButtonActionQueryButton      SysMenuButtonEventAction = "query_button"
	ButtonActionButtonMetadata   SysMenuButtonEventAction = "button_metadata"
	ButtonActionCreateItem       SysMenuButtonEventAction = "create_item"
	ButtonActionUpdateItem       SysMenuButtonEventAction = "update_item"
	ButtonActionDeleteItem       SysMenuButtonEventAction = "delete_item"
	ButtonActionQueryItem        SysMenuButtonEventAction = "query_item"
	ButtonActionDetailItem       SysMenuButtonEventAction = "detail_item"
	ButtonActionItemMetadata     SysMenuButtonEventAction = "item_metadata"
	ButtonActionAssignPermission SysMenuButtonEventAction = "assign_permission"
	ButtonActionAssignData       SysMenuButtonEventAction = "assign_data_permission"
	ButtonActionQueryUserMenu    SysMenuButtonEventAction = "query_user_menu"
	ButtonActionQueryDataPerm    SysMenuButtonEventAction = "query_data_permission"
	ButtonActionQueryPermMenu    SysMenuButtonEventAction = "query_permission_menu"
	ButtonActionResetPassword    SysMenuButtonEventAction = "reset_password"
	ButtonActionUnlockLogin      SysMenuButtonEventAction = "unlock_login"
	ButtonActionRotateSecret     SysMenuButtonEventAction = "rotate_secret"
	ButtonActionPublish          SysMenuButtonEventAction = "publish"
	ButtonActionUnpublish        SysMenuButtonEventAction = "unpublish"
	ButtonActionInitMeta         SysMenuButtonEventAction = "init_meta"
	ButtonActionSyncFields       SysMenuButtonEventAction = "sync_fields"
	ButtonActionSyncIndex        SysMenuButtonEventAction = "sync_index"
	ButtonActionFieldManager     SysMenuButtonEventAction = "field_manager"
	ButtonActionCreateField      SysMenuButtonEventAction = "create_field"
	ButtonActionUpdateField      SysMenuButtonEventAction = "update_field"
	ButtonActionDeleteField      SysMenuButtonEventAction = "delete_field"
	ButtonActionQueryField       SysMenuButtonEventAction = "query_field"
	ButtonActionDetailField      SysMenuButtonEventAction = "detail_field"
	ButtonActionCreateIndex      SysMenuButtonEventAction = "create_index"
	ButtonActionUpdateIndex      SysMenuButtonEventAction = "update_index"
	ButtonActionDeleteIndex      SysMenuButtonEventAction = "delete_index"
	ButtonActionQueryIndex       SysMenuButtonEventAction = "query_index"
	ButtonActionDetailIndex      SysMenuButtonEventAction = "detail_index"
	ButtonActionCreateRelation   SysMenuButtonEventAction = "create_relation"
	ButtonActionUpdateRelation   SysMenuButtonEventAction = "update_relation"
	ButtonActionDeleteRelation   SysMenuButtonEventAction = "delete_relation"
	ButtonActionQueryRelation    SysMenuButtonEventAction = "query_relation"
	ButtonActionDetailRelation   SysMenuButtonEventAction = "detail_relation"
)

func NormalizeSysMenuButtonEventAction(value string) (SysMenuButtonEventAction, bool) {
	action := SysMenuButtonEventAction(strings.TrimSpace(value))
	switch action {
	case ButtonActionQuery,
		ButtonActionMetadata,
		ButtonActionDetail,
		ButtonActionCreate,
		ButtonActionCreateChild,
		ButtonActionUpdate,
		ButtonActionDelete,
		ButtonActionRefresh,
		ButtonActionBatchDelete,
		ButtonActionCopy,
		ButtonActionDuplicate,
		ButtonActionExport,
		ButtonActionNavigate,
		ButtonActionCustom,
		ButtonActionSave,
		ButtonActionOrder,
		ButtonActionRefreshCache,
		ButtonActionTestEmail,
		ButtonActionCreateButton,
		ButtonActionUpdateButton,
		ButtonActionDeleteButton,
		ButtonActionQueryButton,
		ButtonActionButtonMetadata,
		ButtonActionCreateItem,
		ButtonActionUpdateItem,
		ButtonActionDeleteItem,
		ButtonActionQueryItem,
		ButtonActionDetailItem,
		ButtonActionItemMetadata,
		ButtonActionAssignPermission,
		ButtonActionAssignData,
		ButtonActionQueryUserMenu,
		ButtonActionQueryDataPerm,
		ButtonActionQueryPermMenu,
		ButtonActionResetPassword,
		ButtonActionUnlockLogin,
		ButtonActionRotateSecret,
		ButtonActionPublish,
		ButtonActionUnpublish,
		ButtonActionInitMeta,
		ButtonActionSyncFields,
		ButtonActionSyncIndex,
		ButtonActionFieldManager,
		ButtonActionCreateField,
		ButtonActionUpdateField,
		ButtonActionDeleteField,
		ButtonActionQueryField,
		ButtonActionDetailField,
		ButtonActionCreateIndex,
		ButtonActionUpdateIndex,
		ButtonActionDeleteIndex,
		ButtonActionQueryIndex,
		ButtonActionDetailIndex,
		ButtonActionCreateRelation,
		ButtonActionUpdateRelation,
		ButtonActionDeleteRelation,
		ButtonActionQueryRelation,
		ButtonActionDetailRelation:
		return action, true
	default:
		return "", false
	}
}

func IsReadMenuButtonAction(action SysMenuButtonEventAction) bool {
	return action == ButtonActionQuery || action == ButtonActionDetail
}

// SysMasterDetailMode 主子表展示模式字典
type SysMasterDetailMode string

const (
	MasterDetailAuto    SysMasterDetailMode = "auto"
	MasterDetailSummary SysMasterDetailMode = "summary"
	MasterDetailTable   SysMasterDetailMode = "table"
	MasterDetailStacked SysMasterDetailMode = "stacked"
)

func NormalizeSysMasterDetailMode(value string) (SysMasterDetailMode, bool) {
	mode := SysMasterDetailMode(strings.ToLower(strings.TrimSpace(value)))
	if mode == "" {
		return MasterDetailAuto, true
	}
	switch mode {
	case MasterDetailAuto, MasterDetailSummary, MasterDetailTable, MasterDetailStacked:
		return mode, true
	default:
		return "", false
	}
}

// SysFormOpenMode 低代码表单打开方式字典
type SysFormOpenMode string

const (
	FormOpenAuto   SysFormOpenMode = "auto"
	FormOpenDialog SysFormOpenMode = "dialog"
	FormOpenPage   SysFormOpenMode = "page"
)

func NormalizeSysFormOpenMode(value string) (SysFormOpenMode, bool) {
	mode := SysFormOpenMode(strings.ToLower(strings.TrimSpace(value)))
	if mode == "" {
		return FormOpenAuto, true
	}
	switch mode {
	case FormOpenAuto, FormOpenDialog, FormOpenPage:
		return mode, true
	default:
		return "", false
	}
}

// SysDetailOpenMode 低代码详情打开方式字典
type SysDetailOpenMode string

const (
	DetailOpenAuto   SysDetailOpenMode = "auto"
	DetailOpenDialog SysDetailOpenMode = "dialog"
	DetailOpenPage   SysDetailOpenMode = "page"
)

func NormalizeSysDetailOpenMode(value string) (SysDetailOpenMode, bool) {
	mode := SysDetailOpenMode(strings.ToLower(strings.TrimSpace(value)))
	if mode == "" {
		return DetailOpenAuto, true
	}
	switch mode {
	case DetailOpenAuto, DetailOpenDialog, DetailOpenPage:
		return mode, true
	default:
		return "", false
	}
}

// SysTableType 表类型字典
type SysTableType uint8

const (
	System SysTableType = iota + 1 // 系统表
	View                           // 视图
)

// SysTableFieldType 字段数据库存储类型
type SysTableFieldType uint8

const (
	BigIntFieldType SysTableFieldType = iota + 1 //
	FloatFieldType
	VarcharFieldType
	TextFieldType
	BooleanFieldType
	DateFieldType
	DatetimeFieldType
	TimeFieldType
	TinyintFieldType
	JsonFieldType
	IntFieldType
)

// SysTableFieldInputType 字段
// 页面输入类型
type SysTableFieldInputType uint8

const (
	InputType SysTableFieldInputType = iota + 1
	InputNumberInputType
	TextareaInputType
	SelectInputType
	DatePickerInputType
	DatetimePickerInputType
	TimePickerInputType
	YearPickerInputType
	YearMonthPickerInputType
	FilePickerInputType
	BooleanInputType
	JsonInputType
	ArrayInputType
	KeyValueInputType
	CascaderInputType
	RichTextInputType
)

// ExpressionType 表达式
type ExpressionType uint8

const (
	Gt  ExpressionType = iota + 1 // Gt
	Lt                            // Lt
	Gte                           // Gte
	Lte
	Eq
	Ne
	Like
	NotLike
	In
	NotIn
	IsNull
	IsNotNull
	Between
	NotBetween
)

// ExpressionLogic 关联方式
type ExpressionLogic uint8

const (
	And ExpressionLogic = iota + 1
	Or
)

// SysTableRelationType 表关系
type SysTableRelationType uint8

const (
	OneToOne SysTableRelationType = iota + 1
	OneToMany
	ManyToOne
	ManyToMany
)

// SysTableFieldCategory 表字段类型
type SysTableFieldCategory string

const (
	NormalField     SysTableFieldCategory = "normal_field"     // 默认字段
	VirtualField    SysTableFieldCategory = "virtual_field"    // 虚拟列
	CalculatedField SysTableFieldCategory = "calculated_field" // 计算字段
)

// TokenTypeEnum token类型
type TokenTypeEnum string

const (
	AccessToken  TokenTypeEnum = "access"
	RefreshToken TokenTypeEnum = "refresh"
	AppToken     TokenTypeEnum = "app"
)

type SmsStatus int

const (
	SmsStatusSending SmsStatus = iota + 1 // 发送中
	SmsStatusSuccess                      // 发送成功
	SmsStatusFailed                       // 发送失败
)

// String 返回状态对应的字符串
func (s SmsStatus) String() string {
	switch s {
	case SmsStatusSending:
		return "sending"
	case SmsStatusSuccess:
		return "success"
	case SmsStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// MarshalJSON 实现将 SmsStatus 序列化为 JSON 时输出字符串
func (s SmsStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON 实现将 JSON 数据反序列化到 SmsStatus 中
func (s *SmsStatus) UnmarshalJSON(data []byte) error {
	// 先尝试解码字符串
	var statusStr string
	if err := json.Unmarshal(data, &statusStr); err == nil {
		switch statusStr {
		case "sending":
			*s = SmsStatusSending
		case "success":
			*s = SmsStatusSuccess
		case "failed":
			*s = SmsStatusFailed
		default:
			return errors.New("invalid sms status")
		}
		return nil
	}

	// 如果不是字符串，则尝试解码为整数
	var statusInt int
	if err := json.Unmarshal(data, &statusInt); err != nil {
		return err
	}
	*s = SmsStatus(statusInt)
	return nil
}
