package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type ExternalSystemRepository interface {
	BasicRepository[model.ExternalSystem]
	GetExternalSystemList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.ExternalSystem], error)
	HasConfigurationReferences(*gorm.DB, int) (bool, error)
}
