package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

// InterfaceDefinitionRuntimeRecord 是 Credential Provider 校验接口归属所需的最小投影。
type InterfaceDefinitionRuntimeRecord struct {
	ID               int
	ExternalSystemID int
	CredentialID     *int
}

type InterfaceDefinitionRepository interface {
	BasicRepository[model.InterfaceDefinition]
	GetInterfaceDefinitionList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.InterfaceDefinition], error)
	GetRuntimeInterfaceDefinition(context.Context, int) (InterfaceDefinitionRuntimeRecord, error)
	NextVersion(*gorm.DB, int, string) (int, error)
	HasEnabledVersion(*gorm.DB, int, string, int) (bool, error)
}
