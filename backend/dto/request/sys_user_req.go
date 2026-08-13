/**
 * @Author: Nan
 * @Date: 2024/6/28 下午3:37
 */

package request

type SysUserCreateReq struct {
	UserName    string `json:"user_name" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Email       string `json:"email" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type SysUserUpdateReq struct {
	Id          int    `json:"id" binding:"required"`
	UserName    string `json:"user_name" binding:"required"`
	Email       string `json:"email" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type SysUserUpdatePasswordReq struct {
	Id       int    `json:"id" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SysUserAssignRolesReq struct {
	RoleIds []int `json:"role_ids"`
}
