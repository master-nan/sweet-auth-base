package impl

import (
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationRepositoryImpl struct {
	db *gorm.DB
}

func NewNotificationRepositoryImpl(primaryDB *database.PrimaryDB) *NotificationRepositoryImpl {
	return &NotificationRepositoryImpl{db: primaryDB.DB}
}

func (repositoryImpl *NotificationRepositoryImpl) DBWithContext(ctx context.Context) *gorm.DB {
	return repositoryImpl.db.WithContext(ctx)
}

func (repositoryImpl *NotificationRepositoryImpl) CreateNotification(db *gorm.DB, value *model.Notification) error {
	return db.Create(value).Error
}

func (repositoryImpl *NotificationRepositoryImpl) CreateRecipients(
	db *gorm.DB,
	values []model.NotificationRecipient,
	ignoreConflicts bool,
) (int64, error) {
	if len(values) == 0 {
		return 0, nil
	}
	query := db
	if ignoreConflicts {
		query = query.Clauses(clause.OnConflict{DoNothing: true})
	}
	result := query.CreateInBatches(&values, 500)
	return result.RowsAffected, result.Error
}

func (repositoryImpl *NotificationRepositoryImpl) FindByDedup(
	ctx context.Context,
	sourceModule string,
	dedupKey string,
) (model.Notification, error) {
	var value model.Notification
	err := repositoryImpl.db.WithContext(ctx).
		Where("source_module = ? AND dedup_key = ?", sourceModule, dedupKey).
		First(&value).Error
	return value, err
}

func (repositoryImpl *NotificationRepositoryImpl) RecipientUserIDs(db *gorm.DB, notificationId int) ([]int, error) {
	var userIds []int
	err := db.Model(&model.NotificationRecipient{}).
		Where("notification_id = ?", notificationId).
		Order("user_id ASC").Pluck("user_id", &userIds).Error
	return userIds, err
}

func (repositoryImpl *NotificationRepositoryImpl) CountActiveUsers(db *gorm.DB, userIds []int) (int64, error) {
	var count int64
	err := db.Model(&model.SysUser{}).
		Where("id IN ? AND state = TRUE AND gmt_delete IS NULL", userIds).
		Count(&count).Error
	return count, err
}

func (repositoryImpl *NotificationRepositoryImpl) ActiveMenuExists(db *gorm.DB, name string) (bool, error) {
	var count int64
	err := db.Model(&model.SysMenu{}).
		Where("name = ? AND state = TRUE AND gmt_delete IS NULL", name).
		Count(&count).Error
	return count > 0, err
}

func (repositoryImpl *NotificationRepositoryImpl) UnreadCount(ctx context.Context, userId int) (int64, error) {
	var count int64
	err := repositoryImpl.db.WithContext(ctx).Model(&model.NotificationRecipient{}).
		Where("user_id = ? AND read_at IS NULL", userId).
		Count(&count).Error
	return count, err
}

func notificationRecordQuery(db *gorm.DB) *gorm.DB {
	return db.Table("notification_recipient AS recipient").
		Select(`notification.id, notification.category, notification.level, notification.title,
			notification.content, notification.source_module, notification.source_type,
			notification.source_id, notification.action_menu_name, notification.action_path,
			recipient.read_at, notification.created_at`).
		Joins("JOIN notification ON notification.id = recipient.notification_id")
}

func (repositoryImpl *NotificationRepositoryImpl) Recent(
	ctx context.Context,
	userId int,
	limit int,
) ([]repository.NotificationRecord, error) {
	var values []repository.NotificationRecord
	err := notificationRecordQuery(repositoryImpl.db.WithContext(ctx)).
		Where("recipient.user_id = ?", userId).
		Order("CASE WHEN recipient.read_at IS NULL THEN 0 ELSE 1 END ASC").
		Order("recipient.created_at DESC, recipient.notification_id DESC").
		Limit(limit).Scan(&values).Error
	return values, err
}

func (repositoryImpl *NotificationRepositoryImpl) Query(
	ctx context.Context,
	filter repository.NotificationListFilter,
) (repository.NotificationPage, error) {
	query := notificationRecordQuery(repositoryImpl.db.WithContext(ctx)).Where("recipient.user_id = ?", filter.UserId)
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("lower(notification.title) LIKE ? OR lower(notification.content) LIKE ?", pattern, pattern)
	}
	switch filter.ReadStatus {
	case model.NotificationReadUnread:
		query = query.Where("recipient.read_at IS NULL")
	case model.NotificationReadRead:
		query = query.Where("recipient.read_at IS NOT NULL")
	}
	if filter.Category.Valid() {
		query = query.Where("notification.category = ?", filter.Category)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return repository.NotificationPage{}, err
	}
	page, num := filter.Page, filter.Num
	if page < 1 {
		page = 1
	}
	if num < 1 {
		num = 15
	}
	if num > 50 {
		num = 50
	}
	var values []repository.NotificationRecord
	if err := query.Order("recipient.created_at DESC, recipient.notification_id DESC").
		Offset((page - 1) * num).Limit(num).Scan(&values).Error; err != nil {
		return repository.NotificationPage{}, err
	}
	return repository.NotificationPage{Data: values, Total: total}, nil
}

func (repositoryImpl *NotificationRepositoryImpl) Detail(
	ctx context.Context,
	userId int,
	notificationId int,
) (repository.NotificationRecord, error) {
	var value repository.NotificationRecord
	err := notificationRecordQuery(repositoryImpl.db.WithContext(ctx)).
		Where("recipient.user_id = ? AND recipient.notification_id = ?", userId, notificationId).
		Take(&value).Error
	return value, err
}

func (repositoryImpl *NotificationRepositoryImpl) MarkRead(
	ctx context.Context,
	userId int,
	notificationId int,
	readAt model.CustomTime,
) (bool, error) {
	var count int64
	if err := repositoryImpl.db.WithContext(ctx).Model(&model.NotificationRecipient{}).
		Where("user_id = ? AND notification_id = ?", userId, notificationId).
		Count(&count).Error; err != nil {
		return false, err
	}
	if count == 0 {
		return false, gorm.ErrRecordNotFound
	}
	result := repositoryImpl.db.WithContext(ctx).Model(&model.NotificationRecipient{}).
		Where("user_id = ? AND notification_id = ? AND read_at IS NULL", userId, notificationId).
		Update("read_at", readAt)
	return result.RowsAffected == 1, result.Error
}

func (repositoryImpl *NotificationRepositoryImpl) MarkAllRead(
	ctx context.Context,
	userId int,
	readAt model.CustomTime,
) (int64, error) {
	result := repositoryImpl.db.WithContext(ctx).Model(&model.NotificationRecipient{}).
		Where("user_id = ? AND read_at IS NULL", userId).
		Update("read_at", readAt)
	return result.RowsAffected, result.Error
}

func (repositoryImpl *NotificationRepositoryImpl) AccessibleMenuNames(
	ctx context.Context,
	userId int,
	names []string,
) (map[string]bool, error) {
	result := make(map[string]bool, len(names))
	if len(names) == 0 {
		return result, nil
	}
	var accessible []string
	err := repositoryImpl.db.WithContext(ctx).Table("sys_menu AS menu").
		Joins("JOIN sys_role_menu role_menu ON role_menu.menu_id = menu.id").
		Joins("JOIN sys_user_role user_role ON user_role.role_id = role_menu.role_id AND user_role.user_id = ?", userId).
		Joins("JOIN sys_role role ON role.id = user_role.role_id AND role.state = TRUE AND role.gmt_delete IS NULL").
		Where("menu.name IN ? AND menu.state = TRUE AND menu.gmt_delete IS NULL", names).
		Distinct("menu.name").Pluck("menu.name", &accessible).Error
	if err != nil {
		return nil, err
	}
	for _, name := range accessible {
		result[name] = true
	}
	return result, nil
}
