package model

const (
	UserSessionStatusActive          = "active"
	UserSessionStatusLoggedOut       = "logged_out"
	UserSessionStatusForcedOffline   = "forced_offline"
	UserSessionStatusPasswordChanged = "password_changed"
	UserSessionStatusAccountDisabled = "account_disabled"
	UserSessionStatusAccountDeleted  = "account_deleted"
	UserSessionStatusExpired         = "expired"
)

// SysUserSession 记录一次设备登录。SessionKeyHash 是 JWT sid 的摘要，数据库不保存原始 sid 或 Token。
type SysUserSession struct {
	Basic
	SessionKeyHash   string      `gorm:"size:64;not null;uniqueIndex:ux_sys_user_session_key;comment:Session ID SHA-256" json:"-"`
	UserID           int         `gorm:"type:bigint;not null;index:idx_sys_user_session_user_status;comment:用户ID" json:"user_id"`
	UserNameSnapshot string      `gorm:"size:64;not null;default:'';comment:登录时用户名" json:"user_name_snapshot"`
	Status           string      `gorm:"size:32;not null;index:idx_sys_user_session_user_status;comment:会话状态" json:"status"`
	LoginAt          CustomTime  `gorm:"not null;index:idx_sys_user_session_login;comment:登录时间" json:"login_at"`
	LastSeenAt       CustomTime  `gorm:"not null;index:idx_sys_user_session_last_seen;comment:最后心跳时间" json:"last_seen_at"`
	ExpiresAt        CustomTime  `gorm:"not null;index:idx_sys_user_session_expires;comment:刷新令牌到期时间" json:"expires_at"`
	LogoutAt         *CustomTime `gorm:"comment:退出时间" json:"logout_at"`
	LogoutReason     string      `gorm:"size:160;not null;default:'';comment:退出原因" json:"logout_reason"`
	ClosedByUserID   int         `gorm:"type:bigint;not null;default:0;comment:结束会话的操作人ID" json:"closed_by_user_id"`
	ClosedByUserName string      `gorm:"size:64;not null;default:'';comment:结束会话的操作人用户名" json:"closed_by_user_name"`
	LoginChannel     string      `gorm:"size:32;not null;comment:登录渠道" json:"login_channel"`
	IPAddress        string      `gorm:"size:64;not null;default:'';comment:登录IP" json:"ip_address"`
	UserAgent        string      `gorm:"type:text;not null;default:'';comment:浏览器User-Agent" json:"user_agent"`
	DeviceType       string      `gorm:"size:32;not null;default:'';comment:设备类型" json:"device_type"`
	Browser          string      `gorm:"size:64;not null;default:'';comment:浏览器" json:"browser"`
	OperatingSystem  string      `gorm:"size:64;not null;default:'';comment:操作系统" json:"operating_system"`
	User             SysUser     `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}
