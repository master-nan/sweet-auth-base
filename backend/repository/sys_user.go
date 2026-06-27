/**
 * @Author: Nan
 * @Date: 2024/6/3 下午6:07
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"

	"gorm.io/gorm"
)

type SysUserRepository interface {
	BasicRepository[model.SysUser]
	GetByUserName(string) (model.SysUser, error)
	GetUserList(*request.Basic, model.SysTable) (response.ListResult[model.SysUser], error)
	UpdateBatch(*gorm.DB, []request.SysUserUpdateReq, []int) error
}
