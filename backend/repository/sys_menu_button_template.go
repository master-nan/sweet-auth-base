/**
 * @Author: Nan
 * @Date: 2026/6/23
 */

package repository

import "backend/model"

type SysMenuButtonTemplateRepository interface {
	BasicRepository[model.SysMenuButtonTemplate]
	FindEnabledByScene(string) ([]model.SysMenuButtonTemplate, error)
}
