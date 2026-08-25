package repository

import (
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type NotificationRecord struct {
	Id             int
	Category       model.NotificationCategory
	Level          model.NotificationLevel
	Title          string
	Content        string
	SourceModule   string
	SourceType     string
	SourceId       string
	ActionMenuName string
	ActionPath     string
	ReadAt         *model.CustomTime
	CreatedAt      model.CustomTime
}

type NotificationListFilter struct {
	UserId     int
	Page       int
	Num        int
	Keyword    string
	ReadStatus model.NotificationReadStatus
	Category   model.NotificationCategory
}

type NotificationPage struct {
	Data  []NotificationRecord
	Total int64
}

type NotificationRepository interface {
	DBWithContext(context.Context) *gorm.DB
	CreateNotification(*gorm.DB, *model.Notification) error
	CreateRecipients(*gorm.DB, []model.NotificationRecipient) (int64, error)
	FindByDedup(context.Context, string, string) (model.Notification, error)
	RecipientUserIDs(*gorm.DB, int) ([]int, error)
	CountActiveUsers(*gorm.DB, []int) (int64, error)
	ActiveMenuRoutePaths(*gorm.DB, []string) (map[string]string, error)
	UnreadCount(context.Context, int) (int64, error)
	Recent(context.Context, int, int) ([]NotificationRecord, error)
	Query(context.Context, NotificationListFilter) (NotificationPage, error)
	Detail(context.Context, int, int) (NotificationRecord, error)
	MarkRead(context.Context, int, int, model.CustomTime) (bool, error)
	MarkAllRead(context.Context, int, model.CustomTime) (int64, error)
	AccessibleMenuNames(context.Context, int, []string) (map[string]bool, error)
}
