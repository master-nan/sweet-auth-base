/**
 * @Author: Nan
 * @Date: 2024/7/19 下午5:57
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"gorm.io/gorm"
)

type SysUserRoleRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysUserRole]
}

func NewSysUserRoleRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysUserRoleRepositoryImpl {
	return &SysUserRoleRepositoryImpl{PrimaryDB.DB, NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysUserRole{})}
}

func (s *SysUserRoleRepositoryImpl) GetUserRoles(userId int) ([]model.SysRole, error) {
	var roles []model.SysRole
	err := s.db.
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role.id").
		Where("sys_user_role.user_id = ?", userId).
		Find(&roles).Error
	return roles, err
}
