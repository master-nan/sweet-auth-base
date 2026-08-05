package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type IntegrationExecutionRepository interface {
	BasicRepository[model.IntegrationExecution]
	GetIntegrationExecutionList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.IntegrationExecution], error)
	FindByIdempotency(*gorm.DB, int, int, string, string) (model.IntegrationExecution, error)
	ListCandidatesByStatus(context.Context, []string, int) ([]model.IntegrationExecution, error)
}

type IntegrationLogRepository interface {
	BasicRepository[model.IntegrationLog]
	ListByExecutionID(context.Context, int) ([]model.IntegrationLog, error)
}
