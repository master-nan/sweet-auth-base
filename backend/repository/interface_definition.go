package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type InterfaceDefinitionRepository interface {
	DBWithContext(context.Context) *gorm.DB
	Create(*gorm.DB, *model.InterfaceDefinition) error
	FindByID(context.Context, int) (model.InterfaceDefinition, error)
	FindByIDForUpdate(*gorm.DB, int) (model.InterfaceDefinition, error)
	Query(context.Context, request.InterfaceDefinitionQueryReq, model.SysTable) (response.ListResult[model.InterfaceDefinition], error)
	NextVersion(*gorm.DB, int, string) (int, error)
	HasEnabledVersion(*gorm.DB, int, string, int) (bool, error)
	UpdateFields(*gorm.DB, int, int, map[string]any) (bool, error)
	CredentialReferenceValid(*gorm.DB, int, int) (bool, error)
	RetryPolicyReferenceValid(*gorm.DB, int) (bool, error)
}
