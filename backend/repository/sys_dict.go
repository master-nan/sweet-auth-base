/**
 * @Author: Nan
 * @Date: 2024/5/25 下午12:01
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
)

type SysDictRepository interface {
	BasicRepository[model.SysDict]
	GetSysDictList(*request.Basic, model.SysTable) (response.ListResult[model.SysDict], error)
	GetSysDictByCode(string) (model.SysDict, error)
}
