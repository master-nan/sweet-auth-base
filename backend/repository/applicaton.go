/**
 * @Author: Nan
 * @Date: 2024/10/23 21:52
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
)

type ApplicationRepository interface {
	BasicRepository[model.Application]
	IsAppKeyExists(string) bool
	IsAppSecretExists(string) bool
	FindByAppKey(string) (model.Application, error)
	GetApplicationList(*request.Basic, model.SysTable) (response.ListResult[model.Application], error)
}
