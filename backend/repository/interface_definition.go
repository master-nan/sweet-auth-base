package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type InterfaceDefinitionRepository interface {
	BasicRepository[model.InterfaceDefinition]
	GetInterfaceDefinitionList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.InterfaceDefinition], error)
	NextVersion(*gorm.DB, int, string) (int, error)
	HasEnabledVersion(*gorm.DB, int, string, int) (bool, error)
	RetryPolicyReferenceValid(*gorm.DB, int) (bool, error)
}
