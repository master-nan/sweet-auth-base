/**
 * @Author: Nan
 * @Date: 2024/6/28 下午3:37
 */

package request

import "backend/model"

type SysUserCreateReq struct {
	UserName    string `json:"user_name" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Email       string `json:"email" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type SysUserUpdateReq struct {
	Id           int               `json:"id" binding:"required"`
	UserName     string            `json:"user_name" binding:"required"`
	Email        string            `json:"email" binding:"required"`
	PhoneNumber  string            `json:"phone_number" binding:"required"`
	AccessTokens string            `json:"access_tokens"`
	GmtLastLogin *model.CustomTime `json:"gmt_last_login"`
	IsReset      *bool             `json:"is_reset" `
	GmtDelete    *model.CustomTime `json:"gmt_delete"`
}

type SysUserUpdatePasswordReq struct {
	Id                int               `json:"id" binding:"required"`
	Password          string            `json:"password" binding:"required"`
	IsReset           *bool             `json:"is_reset"`
	PasswordChangedAt *model.CustomTime `json:"password_changed_at"`
}
