/**
 * @Author: Nan
 * @Date: 2024/7/19 上午11:24
 */

package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type SysMenuRepository interface {
	BasicRepository[model.SysMenu]
	GetMenus() ([]model.SysMenu, error)
	GetMenusByRoleIds([]int) ([]model.SysMenu, error)
	FindPublishedLowCodeMenus(*gorm.DB, string) ([]model.SysMenu, error)
	FindPublishedLowCodeMenusByTableCode(*gorm.DB, string) ([]model.SysMenu, error)
	FindFixedMenusByTableCode(*gorm.DB, string) ([]model.SysMenu, error)
	UpdateMenuFields(*gorm.DB, int, map[string]any) error
	HideMenusByIds(*gorm.DB, []int) error
}
