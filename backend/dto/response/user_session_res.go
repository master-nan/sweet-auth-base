package response

import "backend/model"

type UserSessionRes struct {
	ID               int               `json:"id"`
	UserID           int               `json:"user_id"`
	UserName         string            `json:"user_name"`
	UserDeleted      bool              `json:"user_deleted"`
	Status           string            `json:"status"`
	Online           bool              `json:"online"`
	Current          bool              `json:"current"`
	LoginAt          model.CustomTime  `json:"login_at"`
	LastSeenAt       model.CustomTime  `json:"last_seen_at"`
	ExpiresAt        model.CustomTime  `json:"expires_at"`
	LogoutAt         *model.CustomTime `json:"logout_at"`
	LogoutReason     string            `json:"logout_reason"`
	ClosedByUserID   int               `json:"closed_by_user_id"`
	ClosedByUserName string            `json:"closed_by_user_name"`
	LoginChannel     string            `json:"login_channel"`
	IPAddress        string            `json:"ip_address"`
	UserAgent        string            `json:"user_agent"`
	DeviceType       string            `json:"device_type"`
	Browser          string            `json:"browser"`
	OperatingSystem  string            `json:"operating_system"`
}

type UserSessionListRes struct {
	Items          []UserSessionRes `json:"items"`
	Total          int              `json:"total"`
	OnlineUsers    int              `json:"online_users"`
	OnlineSessions int              `json:"online_sessions"`
	// OnlineDevices 为旧前端保留，含义同 OnlineSessions。
	OnlineDevices int `json:"online_devices"`
}
