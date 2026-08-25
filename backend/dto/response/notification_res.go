package response

import "backend/model"

type NotificationUnreadCountRes struct {
	UnreadCount int64 `json:"unread_count"`
}

type NotificationActionRes struct {
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
}

type NotificationSummaryRes struct {
	Id             int                        `json:"id"`
	Category       model.NotificationCategory `json:"category"`
	Level          model.NotificationLevel    `json:"level"`
	Title          string                     `json:"title"`
	ContentPreview string                     `json:"content_preview"`
	Read           bool                       `json:"read"`
	ReadAt         *model.CustomTime          `json:"read_at,omitempty"`
	CreatedAt      model.CustomTime           `json:"created_at"`
	Action         *NotificationActionRes     `json:"action,omitempty"`
}

type NotificationSourceRes struct {
	Module string `json:"module"`
	Type   string `json:"type"`
	Id     string `json:"id,omitempty"`
}

type NotificationDetailRes struct {
	NotificationSummaryRes
	Content string                `json:"content"`
	Source  NotificationSourceRes `json:"source"`
}

type NotificationMarkAllReadRes struct {
	UpdatedCount int64 `json:"updated_count"`
}
