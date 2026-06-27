/**
 * @Author: Nan
 * @Date: 2024/7/19 下午5:07
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"

	"gorm.io/gorm"
)

type SysRoleRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysRole]
}

func NewSysRoleRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysRoleRepositoryImpl {
	return &SysRoleRepositoryImpl{PrimaryDB.DB, NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysRole{})}
}

func (s *SysRoleRepositoryImpl) GetRoles() ([]model.SysRole, error) {
	var roles []model.SysRole
	err := s.db.Preload("Menus").Preload("Buttons").Find(&roles).Error
	return roles, err
}

func (s *SysRoleRepositoryImpl) GetRoleButtons(roleId int) ([]model.SysMenuButton, error) {
	var role model.SysRole
	err := s.db.Preload("Buttons").First(&role, roleId).Error
	if err != nil {
		return nil, err
	}
	return role.Buttons, nil
}

func (s *SysRoleRepositoryImpl) GetRoleList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysRole], error) {
	var repo response.ListResult[model.SysRole]
	var sysRoleList []model.SysRole
	total, err := s.PaginateAndCountAsync(basic, &sysRoleList, table)
	repo.Data = sysRoleList
	repo.Total = int(total)
	return repo, err
}
