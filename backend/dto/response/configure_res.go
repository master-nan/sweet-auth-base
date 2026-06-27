package response

import "backend/model"

type PublicConfigureRes struct {
	EnableCaptcha       bool   `json:"enable_captcha"`
	PasswordLength      int    `json:"password_length"`
	PasswordComplexity  int    `json:"password_complexity"`
	PasswordExpireTime  int    `json:"password_expire_time"`
	PasswordErrorCount  int    `json:"password_error_count"`
	PasswordLockMinutes int    `json:"password_lock_minutes"`
	PasswordPolicy      string `json:"password_policy"`
	SystemName          string `json:"system_name"`
	SystemVersion       string `json:"system_version"`
	SystemLogo          string `json:"system_logo"`
	SystemDescription   string `json:"system_description"`
}

type ConfigureRes struct {
	PublicConfigureRes
	Id          int              `json:"id"`
	GmtCreate   model.CustomTime `json:"gmt_create"`
	CreateName  *string          `json:"create_name"`
	GmtModify   model.CustomTime `json:"gmt_modify"`
	ModifyName  *string          `json:"modify_name"`
	State       bool             `json:"state"`
	EnableEmail bool             `json:"enable_email"`
	SmtpServer  string           `json:"smtp_server"`
	SmtpPort    int              `json:"smtp_port"`
	SenderEmail string           `json:"sender_email"`
}
