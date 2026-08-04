package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type ExternalSystemRepository interface {
	DBWithContext(context.Context) *gorm.DB
	Create(*gorm.DB, *model.ExternalSystem) error
	FindByID(context.Context, int) (model.ExternalSystem, error)
	FindByIDForUpdate(*gorm.DB, int) (model.ExternalSystem, error)
	FindByCode(*gorm.DB, string) (model.ExternalSystem, error)
	Query(context.Context, request.ExternalSystemQueryReq, model.SysTable) (response.ListResult[model.ExternalSystem], error)
	UpdateFields(*gorm.DB, int, int, map[string]any) (bool, error)
	HasConfigurationReferences(*gorm.DB, int) (bool, error)
}
