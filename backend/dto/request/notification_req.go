package request

import "backend/model"

type NotificationRecentReq struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=10"`
}

type NotificationQueryReq struct {
	Page       int                          `json:"page" binding:"omitempty,min=1"`
	Num        int                          `json:"num" binding:"omitempty,min=1,max=50"`
	Keyword    string                       `json:"keyword" binding:"omitempty,max=100"`
	ReadStatus model.NotificationReadStatus `json:"read_status"`
	Category   model.NotificationCategory   `json:"category"`
}
