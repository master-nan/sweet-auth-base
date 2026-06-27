/**
 * @Author: Nan
 * @Date: 2024/8/1 下午10:37
 */

package impl

import (
	"backend/internal/database"
	"backend/model"

	"gorm.io/gorm"
)

type SysMenuButtonRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysMenuButton]
}

func NewSysMenuButtonRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysMenuButtonRepositoryImpl {
	return &SysMenuButtonRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysMenuButton{}),
	}
}

func (s *SysMenuButtonRepositoryImpl) FindByMenuIdAndCode(tx *gorm.DB, menuId int, code string) (model.SysMenuButton, error) {
	var button model.SysMenuButton
	if tx == nil {
		tx = s.db
	}
	err := tx.Where("menu_id = ? AND code = ?", menuId, code).First(&button).Error
	return button, err
}

func (s *SysMenuButtonRepositoryImpl) FindLegacyLowCodeButtons(tx *gorm.DB, menuId int) ([]model.SysMenuButton, error) {
	var buttons []model.SysMenuButton
	if tx == nil {
		tx = s.db
	}
	err := tx.Where("menu_id = ? AND code >= ? AND code < ?", menuId, "system_", "system`").Find(&buttons).Error
	return buttons, err
}

func (s *SysMenuButtonRepositoryImpl) UpdateMenuButtonFields(tx *gorm.DB, id int, values map[string]any) error {
	if tx == nil {
		tx = s.db
	}
	return tx.Model(&model.SysMenuButton{}).Where("id = ?", id).Updates(values).Error
}
