/**
 * @Author: Nan
 * @Date: 2026/6/23
 */

package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type SysMenuButtonTemplateRepository interface {
	BasicRepository[model.SysMenuButtonTemplate]
	FindEnabledByScene(string) ([]model.SysMenuButtonTemplate, error)
	FindEnabledBySceneWithDB(*gorm.DB, string) ([]model.SysMenuButtonTemplate, error)
}
