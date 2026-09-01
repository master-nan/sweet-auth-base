package model

type NotificationCategory string

const (
	NotificationCategorySystem      NotificationCategory = "SYSTEM"
	NotificationCategoryBusiness    NotificationCategory = "BUSINESS"
	NotificationCategoryTask        NotificationCategory = "TASK"
	NotificationCategoryReminder    NotificationCategory = "REMINDER"
	NotificationCategorySecurity    NotificationCategory = "SECURITY"
	NotificationCategoryIntegration NotificationCategory = "INTEGRATION"
)

func (value NotificationCategory) Valid() bool {
	switch value {
	case NotificationCategorySystem,
		NotificationCategoryBusiness,
		NotificationCategoryTask,
		NotificationCategoryReminder,
		NotificationCategorySecurity,
		NotificationCategoryIntegration:
		return true
	default:
		return false
	}
}

type NotificationLevel string

const (
	NotificationLevelInfo    NotificationLevel = "INFO"
	NotificationLevelSuccess NotificationLevel = "SUCCESS"
	NotificationLevelWarning NotificationLevel = "WARNING"
	NotificationLevelError   NotificationLevel = "ERROR"
)

func (value NotificationLevel) Valid() bool {
	switch value {
	case NotificationLevelInfo,
		NotificationLevelSuccess,
		NotificationLevelWarning,
		NotificationLevelError:
		return true
	default:
		return false
	}
}

type NotificationReadStatus string

const (
	NotificationReadAll    NotificationReadStatus = "ALL"
	NotificationReadUnread NotificationReadStatus = "UNREAD"
	NotificationReadRead   NotificationReadStatus = "READ"
)

func (value NotificationReadStatus) Valid() bool {
	switch value {
	case NotificationReadAll, NotificationReadUnread, NotificationReadRead:
		return true
	default:
		return false
	}
}

// Notification 保存一次发送共享且不可变的站内消息事实。
// 用户自己的投递和已读状态由 NotificationRecipient 独立维护。
type Notification struct {
	Basic
	Category       NotificationCategory `gorm:"size:24;not null;comment:通知分类" json:"category"`
	Level          NotificationLevel    `gorm:"size:16;not null;comment:通知级别" json:"level"`
	Title          string               `gorm:"size:160;not null;comment:标题" json:"title"`
	Content        string               `gorm:"type:text;not null;comment:内容" json:"content"`
	SourceModule   string               `gorm:"size:64;not null;comment:来源模块" json:"source_module"`
	SourceType     string               `gorm:"size:64;not null;comment:来源类型" json:"source_type"`
	SourceId       string               `gorm:"size:128;not null;default:'';comment:来源ID" json:"source_id"`
	ActionMenuName string               `gorm:"size:32;not null;default:'';comment:操作菜单" json:"action_menu_name"`
	ActionPath     string               `gorm:"size:512;not null;default:'';comment:操作路径" json:"action_path"`
	DedupKey       *string              `gorm:"size:128;comment:去重键" json:"-"`
}

func (Notification) TableName() string {
	return "notification"
}

// NotificationRecipient 以消息和认证用户组成复合身份，read_at 只记录首次阅读时间。
type NotificationRecipient struct {
	NotificationId int         `gorm:"primaryKey;type:bigint;autoIncrement:false;comment:通知ID" json:"notification_id"`
	UserId         int         `gorm:"primaryKey;type:bigint;autoIncrement:false;comment:用户ID" json:"user_id"`
	ReadAt         *CustomTime `gorm:"comment:阅读时间" json:"read_at"`
	CreatedAt      CustomTime  `gorm:"autoCreateTime;comment:投递时间" json:"created_at"`
}

func (NotificationRecipient) TableName() string {
	return "notification_recipient"
}
