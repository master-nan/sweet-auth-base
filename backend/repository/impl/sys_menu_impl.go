/**
 * @Author: Nan
 * @Date: 2024/7/19 上午11:27
 */

package impl

import (
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	"strings"

	"gorm.io/gorm"
)

type SysMenuRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysMenu]
}

func NewSysMenuRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysMenuRepositoryImpl {
	return &SysMenuRepositoryImpl{PrimaryDB.DB, NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysMenu{})}
}

func (s *SysMenuRepositoryImpl) GetMenus() ([]model.SysMenu, error) {
	var menus []model.SysMenu
	err := s.db.Preload("MenuButtons").Find(&menus).Error
	return menus, err
}

// GetMenusByRoleIds 获取角色的所有菜单
func (s *SysMenuRepositoryImpl) GetMenusByRoleIds(roleIds []int) ([]model.SysMenu, error) {
	var menus []model.SysMenu
	err := s.db.Preload("Roles", "Roles.id IN ?", roleIds).Find(&menus).Error
	return menus, err
}

func (s *SysMenuRepositoryImpl) FindPublishedLowCodeMenus(tx *gorm.DB, tableCode string) ([]model.SysMenu, error) {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return nil, nil
	}
	if tx == nil {
		tx = s.db
	}
	var menus []model.SysMenu
	err := tx.
		Where("table_code = ? AND page_type = ?", tableCode, enum.MenuPageTypeLowCode).
		Order("state DESC, is_hidden ASC, id ASC").
		Find(&menus).Error
	return menus, err
}

func (s *SysMenuRepositoryImpl) FindPublishedLowCodeMenusByTableCode(tx *gorm.DB, tableCode string) ([]model.SysMenu, error) {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return nil, nil
	}
	if tx == nil {
		tx = s.db
	}
	var menus []model.SysMenu
	err := tx.
		Preload("MenuButtons").
		Where("table_code = ? AND page_type = ?", tableCode, enum.MenuPageTypeLowCode).
		Order("state DESC, is_hidden ASC, id ASC").
		Find(&menus).Error
	return menus, err
}

func (s *SysMenuRepositoryImpl) FindFixedMenusByTableCode(tx *gorm.DB, tableCode string) ([]model.SysMenu, error) {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return nil, nil
	}
	if tx == nil {
		tx = s.db
	}
	var menus []model.SysMenu
	err := tx.
		Where("table_code = ? AND page_type = ?", tableCode, enum.MenuPageTypeFixed).
		Order("state DESC, is_hidden ASC, id ASC").
		Find(&menus).Error
	return menus, err
}

func (s *SysMenuRepositoryImpl) UpdateMenuFields(tx *gorm.DB, id int, values map[string]any) error {
	if tx == nil {
		tx = s.db
	}
	return tx.Model(&model.SysMenu{}).Where("id = ?", id).Updates(values).Error
}

func (s *SysMenuRepositoryImpl) HideMenusByIds(tx *gorm.DB, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	if tx == nil {
		tx = s.db
	}
	return tx.Model(&model.SysMenu{}).Where("id IN ?", ids).Updates(map[string]any{
		"is_hidden": true,
		"state":     false,
	}).Error
}
