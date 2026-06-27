/**
 * @Author: Nan
 * @Date: 2024/7/19 下午5:56
 */

package repository

import (
	"backend/model"
)

type SysUserRoleRepository interface {
	BasicRepository[model.SysUserRole]
	GetUserRoles(userId int) ([]model.SysRole, error)
}
