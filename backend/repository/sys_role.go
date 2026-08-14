/**
 * @Author: Nan
 * @Date: 2024/7/19 下午5:06
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
)

type SysRoleRepository interface {
	BasicRepository[model.SysRole]
	GetRoleList(*request.Basic, model.SysTable) (response.ListResult[model.SysRole], error)
}
