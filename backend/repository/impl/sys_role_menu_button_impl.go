/**
 * @Author: Nan
 * @Date: 2024/7/19 下午6:00
 */

package impl

import (
	"backend/internal/database"
	"backend/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SysRoleMenuButtonRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysRoleMenuButton]
}

func NewSysRoleMenuButtonRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysRoleMenuButtonRepositoryImpl {
	return &SysRoleMenuButtonRepositoryImpl{PrimaryDB.DB, NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysRoleMenuButton{})}
}

func (s *SysRoleMenuButtonRepositoryImpl) GetRoleMenuButtons(roleId, menuId int) ([]model.SysMenuButton, error) {
	var buttons []model.SysMenuButton
	err := s.db.Preload("Roles", "id = ?", roleId).
		Preload("Menus", "id = ?", menuId).
		Find(&buttons).Error
	return buttons, err
}

func (s *SysRoleMenuButtonRepositoryImpl) CountActiveButtonPolicy(tx *gorm.DB, roleId int, path string, method string) (int64, error) {
	var count int64
	roleMenuButtonTable := tx.NamingStrategy.TableName("SysRoleMenuButton")
	menuButtonTable := tx.NamingStrategy.TableName("SysMenuButton")
	err := tx.Table(roleMenuButtonTable+" AS rmb").
		Joins("JOIN "+menuButtonTable+" AS b ON b.id = rmb.button_id").
		Where("rmb.role_id = ?", roleId).
		Where("b.gmt_delete IS NULL").
		Where("b.state = ? AND b.is_disabled = ?", true, false).
		Where("b.path = ? AND UPPER(b.method) = ?", path, method).
		Count(&count).Error
	return count, err
}

func (s *SysRoleMenuButtonRepositoryImpl) DeleteByButtonIds(tx *gorm.DB, buttonIds []int) error {
	if len(buttonIds) == 0 {
		return nil
	}
	if tx == nil {
		tx = s.db
	}
	return tx.Where("button_id IN ?", buttonIds).Delete(&model.SysRoleMenuButton{}).Error
}

func (s *SysRoleMenuButtonRepositoryImpl) DeleteByMenuIds(tx *gorm.DB, menuIds []int) error {
	if len(menuIds) == 0 {
		return nil
	}
	if tx == nil {
		tx = s.db
	}
	return tx.Where("menu_id IN ?", menuIds).Delete(&model.SysRoleMenuButton{}).Error
}

// CreateIfNotExists 插入角色菜单按钮关系；如果唯一键已经存在，直接忽略。
// 发布低代码菜单时会重复补 super_admin 权限，用这个方法避免重复发布时报冲突。
func (s *SysRoleMenuButtonRepositoryImpl) CreateIfNotExists(tx *gorm.DB, roleMenuButton model.SysRoleMenuButton) error {
	if tx == nil {
		tx = s.db
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&roleMenuButton).Error
}
