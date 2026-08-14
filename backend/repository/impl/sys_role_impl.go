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
)

type SysRoleRepositoryImpl struct {
	*BasicRepositoryImpl[model.SysRole]
}

func NewSysRoleRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysRoleRepositoryImpl {
	return &SysRoleRepositoryImpl{NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysRole{})}
}

func (s *SysRoleRepositoryImpl) GetRoleList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysRole], error) {
	var repo response.ListResult[model.SysRole]
	var sysRoleList []model.SysRole
	total, err := s.PaginateAndCountAsync(basic, &sysRoleList, table)
	repo.Data = sysRoleList
	repo.Total = int(total)
	return repo, err
}
