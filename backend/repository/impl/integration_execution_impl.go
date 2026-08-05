package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"context"

	"gorm.io/gorm"
)

type IntegrationExecutionRepositoryImpl struct {
	*BasicRepositoryImpl[model.IntegrationExecution]
}

func NewIntegrationExecutionRepositoryImpl(primaryDB *database.PrimaryDB) *IntegrationExecutionRepositoryImpl {
	return &IntegrationExecutionRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.IntegrationExecution{}),
	}
}

func (r *IntegrationExecutionRepositoryImpl) GetIntegrationExecutionList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.IntegrationExecution], error) {
	var values []model.IntegrationExecution
	total, err := r.WithContext(ctx).PaginateAndCountAsync(basic, &values, table)
	return response.ListResult[model.IntegrationExecution]{Data: values, Total: int(total)}, err
}

func (r *IntegrationExecutionRepositoryImpl) FindByIdempotency(
	db *gorm.DB,
	interfaceDefinitionID int,
	interfaceVersion int,
	scope string,
	key string,
) (model.IntegrationExecution, error) {
	var value model.IntegrationExecution
	err := db.Model(&model.IntegrationExecution{}).
		Where(
			"interface_definition_id = ? AND interface_version = ? AND idempotency_scope = ? AND idempotency_key = ?",
			interfaceDefinitionID,
			interfaceVersion,
			scope,
			key,
		).
		First(&value).Error
	return value, err
}

func (r *IntegrationExecutionRepositoryImpl) ListCandidatesByStatus(
	ctx context.Context,
	statuses []string,
	limit int,
) ([]model.IntegrationExecution, error) {
	if len(statuses) == 0 || limit <= 0 {
		return []model.IntegrationExecution{}, nil
	}
	var values []model.IntegrationExecution
	err := r.DBWithContext(ctx).
		Model(&model.IntegrationExecution{}).
		Where("status IN ?", statuses).
		Order("gmt_create ASC, id ASC").
		Limit(limit).
		Find(&values).Error
	return values, err
}

type IntegrationLogRepositoryImpl struct {
	*BasicRepositoryImpl[model.IntegrationLog]
}

func NewIntegrationLogRepositoryImpl(primaryDB *database.PrimaryDB) *IntegrationLogRepositoryImpl {
	return &IntegrationLogRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.IntegrationLog{}),
	}
}

func (r *IntegrationLogRepositoryImpl) ListByExecutionID(
	ctx context.Context,
	executionID int,
) ([]model.IntegrationLog, error) {
	var values []model.IntegrationLog
	err := r.DBWithContext(ctx).
		Model(&model.IntegrationLog{}).
		Where("execution_id = ?", executionID).
		Order("attempt_no ASC").
		Find(&values).Error
	return values, err
}

var _ repository.IntegrationExecutionRepository = (*IntegrationExecutionRepositoryImpl)(nil)
var _ repository.IntegrationLogRepository = (*IntegrationLogRepositoryImpl)(nil)
