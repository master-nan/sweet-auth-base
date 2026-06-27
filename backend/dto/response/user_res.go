/**
 * @Author: Nan
 * @Date: 2024/7/3 下午12:03
 */

package response

import "backend/model"

type SysUserRes struct {
	BasicRes
	UserName          string            `json:"user_name"`
	Email             string            `json:"email"`
	PhoneNumber       string            `json:"phone_number"`
	GmtLastLogin      model.CustomTime  `json:"gmt_last_login"`
	PasswordChangedAt *model.CustomTime `json:"password_changed_at"`
	Language          string            `json:"language"`
	IsReset           bool              `json:"is_reset"`
	Roles             []RoleSimpleRes   `json:"roles"`
}

type RoleSimpleRes struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Memo string `json:"memo"`
}
