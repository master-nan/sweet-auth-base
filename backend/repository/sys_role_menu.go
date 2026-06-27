/**
 * @Author: Nan
 * @Date: 2024/7/19 下午5:56
 */

package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type SysRoleMenuRepository interface {
	BasicRepository[model.SysRoleMenu]
	GetRoleMenus(int) ([]model.SysMenu, error)
	GetRoleMenusByRoleIds([]int) ([]model.SysRoleMenu, error)
	DeleteRoleMenuByRoleIdAndMenuId(*gorm.DB, int, int) error
	DeleteByMenuIds(*gorm.DB, []int) error
	CreateIfNotExists(*gorm.DB, model.SysRoleMenu) error
}
