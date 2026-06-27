/**
 * @Author: Nan
 * @Date: 2024/7/19 下午5:59
 */

package impl

import (
	"backend/internal/database"
	"backend/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SysRoleMenuRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysRoleMenu]
}

func NewSysRoleMenuRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysRoleMenuRepositoryImpl {
	return &SysRoleMenuRepositoryImpl{PrimaryDB.DB, NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysRoleMenu{})}
}

func (s *SysRoleMenuRepositoryImpl) GetRoleMenus(roleId int) ([]model.SysMenu, error) {
	var menus []model.SysMenu
	err := s.db.Preload("Roles", "Roles.id = ?", roleId).Find(&menus).Error
	return menus, err
}

// GetRoleMenusByRoleIds 获取角色的所有菜单
func (s *SysRoleMenuRepositoryImpl) GetRoleMenusByRoleIds(roleIds []int) ([]model.SysRoleMenu, error) {
	var menus []model.SysRoleMenu
	err := s.db.Where("role_id IN ?", roleIds).Find(&menus).Error
	return menus, err
}

func (s *SysRoleMenuRepositoryImpl) DeleteRoleMenuByRoleIdAndMenuId(tx *gorm.DB, roleId, menuId int) error {
	if tx == nil {
		tx = s.db
	}
	return tx.Where("role_id = ? and menu_id = ?", roleId, menuId).Delete(&model.SysRoleMenu{}).Error
}

func (s *SysRoleMenuRepositoryImpl) DeleteByMenuIds(tx *gorm.DB, menuIds []int) error {
	if len(menuIds) == 0 {
		return nil
	}
	if tx == nil {
		tx = s.db
	}
	return tx.Where("menu_id IN ?", menuIds).Delete(&model.SysRoleMenu{}).Error
}

// CreateIfNotExists 插入角色菜单关系；如果唯一键已经存在，直接忽略。
// 发布低代码菜单时会重复补 super_admin 权限，用这个方法避免重复发布时报冲突。
func (s *SysRoleMenuRepositoryImpl) CreateIfNotExists(tx *gorm.DB, roleMenu model.SysRoleMenu) error {
	if tx == nil {
		tx = s.db
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&roleMenu).Error
}
