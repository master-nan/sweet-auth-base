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

type InterfaceDefinitionRepositoryImpl struct {
	*BasicRepositoryImpl[model.InterfaceDefinition]
}

func NewInterfaceDefinitionRepositoryImpl(primaryDB *database.PrimaryDB) *InterfaceDefinitionRepositoryImpl {
	return &InterfaceDefinitionRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.InterfaceDefinition{}),
	}
}

func (r *InterfaceDefinitionRepositoryImpl) GetInterfaceDefinitionList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.InterfaceDefinition], error) {
	var values []model.InterfaceDefinition
	total, err := r.WithContext(ctx).PaginateAndCountAsync(basic, &values, table)
	return response.ListResult[model.InterfaceDefinition]{Data: values, Total: int(total)}, err
}

// GetRuntimeInterfaceDefinition 仅读取运行时凭证归属校验需要的字段。
func (r *InterfaceDefinitionRepositoryImpl) GetRuntimeInterfaceDefinition(
	ctx context.Context,
	id int,
) (repository.InterfaceDefinitionRuntimeRecord, error) {
	var value repository.InterfaceDefinitionRuntimeRecord
	err := r.DBWithContext(ctx).
		Model(&model.InterfaceDefinition{}).
		Select([]string{"id", "external_system_id", "credential_id"}).
		Where("id = ?", id).
		First(&value).Error
	return value, err
}

func (r *InterfaceDefinitionRepositoryImpl) NextVersion(tx *gorm.DB, systemID int, code string) (int, error) {
	var maxVersion int
	err := tx.Model(&model.InterfaceDefinition{}).
		Where("external_system_id = ? AND interface_code = ?", systemID, code).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVersion).Error
	return maxVersion + 1, err
}

func (r *InterfaceDefinitionRepositoryImpl) HasEnabledVersion(tx *gorm.DB, systemID int, code string, excludeID int) (bool, error) {
	var count int64
	err := tx.Model(&model.InterfaceDefinition{}).
		Where("external_system_id = ? AND interface_code = ? AND status = ? AND id <> ?", systemID, code, model.InterfaceDefinitionStatusEnabled, excludeID).
		Count(&count).Error
	return count > 0, err
}

func (r *InterfaceDefinitionRepositoryImpl) RetryPolicyReferenceValid(tx *gorm.DB, id int) (bool, error) {
	if !tx.Migrator().HasTable("integration_retry_policy") {
		return false, nil
	}
	var count int64
	err := tx.Table("integration_retry_policy").Where("id = ? AND state = ? AND gmt_delete IS NULL", id, true).Count(&count).Error
	return count == 1, err
}

var _ repository.InterfaceDefinitionRepository = (*InterfaceDefinitionRepositoryImpl)(nil)
