/**
 * @Author: Nan
 * @Date: 2024/5/21 下午2:46
 */

package request

type ConfigureUpdateReq struct {
	Id int `json:"id"`
	// 安全配置
	EnableCaptcha       *bool  `json:"enable_captcha" binding:"required"`
	PasswordLength      int    `json:"password_length" binding:"required"`
	PasswordComplexity  int    `json:"password_complexity" binding:"required"`
	PasswordExpireTime  int    `json:"password_expire_time" binding:"required"`
	PasswordErrorCount  int    `json:"password_error_count" binding:"required"`
	PasswordLockMinutes int    `json:"password_lock_minutes" binding:"required"`
	PasswordPolicy      string `json:"password_policy" binding:"required"`

	// 系统基本信息
	SystemName        string `json:"system_name"`
	SystemVersion     string `json:"system_version"`
	SystemLogo        string `json:"system_logo"`
	SystemDescription string `json:"system_description"`

	// 邮件配置
	EnableEmail    *bool  `json:"enable_email"`
	SmtpServer     string `json:"smtp_server"`
	SmtpPort       int    `json:"smtp_port"`
	SenderEmail    string `json:"sender_email"`
	SenderPassword string `json:"sender_password"`
}

type ConfigureTestEmailReq struct {
	To string `json:"to" binding:"required,email"`
}
