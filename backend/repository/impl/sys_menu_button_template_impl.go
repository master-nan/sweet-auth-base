/**
 * @Author: Nan
 * @Date: 2026/6/23
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
)

type SysMenuButtonTemplateRepositoryImpl struct {
	*BasicRepositoryImpl[model.SysMenuButtonTemplate]
}

func NewSysMenuButtonTemplateRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysMenuButtonTemplateRepositoryImpl {
	return &SysMenuButtonTemplateRepositoryImpl{
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysMenuButtonTemplate{}),
	}
}

func (s *SysMenuButtonTemplateRepositoryImpl) FindEnabledByScene(scene string) ([]model.SysMenuButtonTemplate, error) {
	var templates []model.SysMenuButtonTemplate
	err := s.db.Where("scene = ? AND state = ?", scene, true).
		Order("sequence ASC, id ASC").
		Find(&templates).Error
	return templates, err
}
