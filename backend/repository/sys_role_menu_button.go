/**
 * @Author: Nan
 * @Date: 2024/7/19 下午5:57
 */

package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type SysRoleMenuButtonRepository interface {
	BasicRepository[model.SysRoleMenuButton]
	GetRoleMenuButtons(roleId, menuId int) ([]model.SysMenuButton, error)
	CountActiveButtonPolicy(*gorm.DB, int, string, string) (int64, error)
	DeleteByButtonIds(*gorm.DB, []int) error
	DeleteByMenuIds(*gorm.DB, []int) error
	CreateIfNotExists(*gorm.DB, model.SysRoleMenuButton) error
}
