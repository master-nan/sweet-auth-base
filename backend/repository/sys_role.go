/**
 * @Author: Nan
 * @Date: 2024/7/19 下午5:06
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
)

type SysRoleRepository interface {
	BasicRepository[model.SysRole]
	GetRoles() ([]model.SysRole, error)
	GetRoleButtons(roleId int) ([]model.SysMenuButton, error)
	GetRoleList(*request.Basic, model.SysTable) (response.ListResult[model.SysRole], error)
}
