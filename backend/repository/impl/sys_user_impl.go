/**
 * @Author: Nan
 * @Date: 2024/6/3 下午6:08
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/internal/utils"
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

type SysUserRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysUser]
}

func NewSysUserRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysUserRepositoryImpl {
	return &SysUserRepositoryImpl{PrimaryDB.DB, NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysUser{})}
}

func (s *SysUserRepositoryImpl) GetByUserName(username string) (model.SysUser, error) {
	var user model.SysUser
	err := s.db.Where(&model.SysUser{UserName: username}).Or(&model.SysUser{PhoneNumber: username}).Or(&model.SysUser{Email: username}).Preload("Roles").First(&user).Error
	return user, err
}

func (s *SysUserRepositoryImpl) GetUserList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysUser], error) {
	var repo response.ListResult[model.SysUser]
	var sysUserList []model.SysUser
	total, err := s.PaginateAndCountAsync(basic, &sysUserList, table)
	repo.Data = sysUserList
	repo.Total = int(total)
	return repo, err
}

func (s *SysUserRepositoryImpl) UpdateBatch(tx *gorm.DB, updateUsers []request.SysUserUpdateReq, updateUserIds []int) error {
	if len(updateUsers) == 0 || len(updateUserIds) == 0 {
		return nil
	}
	caseStatements := make(map[string]string)
	for _, field := range []string{"user_name", "phone_number", "email", "gmt_delete"} {
		caseStatements[field] = "CASE id"
	}
	for i, id := range updateUserIds {
		caseStatements["user_name"] += fmt.Sprintf(" WHEN %d THEN '%s'", id, utils.EscapeString(updateUsers[i].UserName))
		caseStatements["phone_number"] += fmt.Sprintf(" WHEN %d THEN '%s'", id, utils.EscapeString(updateUsers[i].PhoneNumber))
		caseStatements["email"] += fmt.Sprintf(" WHEN %d THEN '%s'", id, utils.EscapeString(updateUsers[i].Email))
		if updateUsers[i].GmtDelete == nil {
			caseStatements["gmt_delete"] += fmt.Sprintf(" WHEN %d THEN null", id)
		} else {
			caseStatements["gmt_delete"] += fmt.Sprintf(" WHEN %d THEN '%s'", id, updateUsers[i].GmtDelete)
		}
	}
	for field := range caseStatements {
		caseStatements[field] += " END"
	}
	updateQuery := tx.Model(&model.SysUser{}).Unscoped().Where("id IN ?", updateUserIds)
	updateExprs := make(map[string]interface{})

	for field, caseStmt := range caseStatements {
		updateExprs[field] = gorm.Expr(caseStmt)
	}
	updateQuery = updateQuery.Updates(updateExprs)
	return updateQuery.Error
}
