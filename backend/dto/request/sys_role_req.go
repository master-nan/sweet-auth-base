/**
 * @Author: Nan
 * @Date: 2024/8/1 下午10:32
 */

package request

type RoleCreateReq struct {
	Name string `json:"name" binding:"required"`
	Memo string `json:"memo" binding:"required"`
}

type RoleUpdateReq struct {
	Id   int    `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
	Memo string `json:"memo" binding:"required"`
}

// RoleAssignPermissionsReq 分配权限
type RoleAssignPermissionsReq struct {
	RoleId    int   `json:"role_id" binding:"required"`
	MenuIds   []int `json:"menu_ids" binding:"required"`
	ButtonIds []int `json:"button_ids" binding:"required"`
}
