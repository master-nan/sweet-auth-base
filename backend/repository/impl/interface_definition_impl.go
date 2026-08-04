package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"backend/repository/util"
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InterfaceDefinitionRepositoryImpl struct{ db *gorm.DB }

func NewInterfaceDefinitionRepositoryImpl(primaryDB *database.PrimaryDB) *InterfaceDefinitionRepositoryImpl {
	return &InterfaceDefinitionRepositoryImpl{db: primaryDB.DB}
}

func (r *InterfaceDefinitionRepositoryImpl) DBWithContext(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *InterfaceDefinitionRepositoryImpl) Create(tx *gorm.DB, value *model.InterfaceDefinition) error {
	return tx.Create(value).Error
}

func (r *InterfaceDefinitionRepositoryImpl) FindByID(ctx context.Context, id int) (model.InterfaceDefinition, error) {
	var value model.InterfaceDefinition
	err := r.db.WithContext(ctx).First(&value, id).Error
	return value, err
}

func (r *InterfaceDefinitionRepositoryImpl) FindByIDForUpdate(tx *gorm.DB, id int) (model.InterfaceDefinition, error) {
	var value model.InterfaceDefinition
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&value, id).Error
	return value, err
}

func (r *InterfaceDefinitionRepositoryImpl) Query(ctx context.Context, req request.InterfaceDefinitionQueryReq, table model.SysTable) (response.ListResult[model.InterfaceDefinition], error) {
	basic := req.ToBasic()
	query := util.ExecuteQuery(r.db.WithContext(ctx).Model(&model.InterfaceDefinition{}), &basic, table)
	if req.ExternalSystemID > 0 {
		query = query.Where("external_system_id = ?", req.ExternalSystemID)
	}
	if req.HTTPMethod != "" {
		query = query.Where("http_method = ?", req.HTTPMethod)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Limit(-1).Offset(-1).Count(&total).Error; err != nil {
		return response.ListResult[model.InterfaceDefinition]{}, err
	}
	var values []model.InterfaceDefinition
	if err := query.Find(&values).Error; err != nil {
		return response.ListResult[model.InterfaceDefinition]{}, err
	}
	return response.ListResult[model.InterfaceDefinition]{Data: values, Total: int(total)}, nil
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

func (r *InterfaceDefinitionRepositoryImpl) UpdateFields(tx *gorm.DB, id, revision int, updates map[string]any) (bool, error) {
	result := tx.Model(&model.InterfaceDefinition{}).Where("id = ? AND revision = ?", id, revision).Updates(updates)
	return result.RowsAffected == 1, result.Error
}

func (r *InterfaceDefinitionRepositoryImpl) CredentialReferenceValid(tx *gorm.DB, id, systemID int) (bool, error) {
	if !tx.Migrator().HasTable("integration_credential") {
		return false, nil
	}
	var count int64
	err := tx.Table("integration_credential").
		Where("id = ? AND external_system_id = ? AND status = ? AND gmt_delete IS NULL", id, systemID, "active").
		Count(&count).Error
	return count == 1, err
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
