/**
 * @Author: Nan
 * @Date: 2024/8/1 下午10:36
 */

package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type SysMenuButtonRepository interface {
	BasicRepository[model.SysMenuButton]
	FindByMenuIdAndCode(*gorm.DB, int, string) (model.SysMenuButton, error)
	FindLegacyLowCodeButtons(*gorm.DB, int) ([]model.SysMenuButton, error)
	UpdateMenuButtonFields(*gorm.DB, int, map[string]any) error
}
