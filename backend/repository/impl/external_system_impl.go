package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"backend/repository/util"
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExternalSystemRepositoryImpl struct {
	db *gorm.DB
}

func NewExternalSystemRepositoryImpl(primaryDB *database.PrimaryDB) *ExternalSystemRepositoryImpl {
	return &ExternalSystemRepositoryImpl{db: primaryDB.DB}
}

func (r *ExternalSystemRepositoryImpl) DBWithContext(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *ExternalSystemRepositoryImpl) Create(tx *gorm.DB, value *model.ExternalSystem) error {
	return tx.Create(value).Error
}

func (r *ExternalSystemRepositoryImpl) FindByID(ctx context.Context, id int) (model.ExternalSystem, error) {
	var value model.ExternalSystem
	err := r.db.WithContext(ctx).First(&value, id).Error
	return value, err
}

func (r *ExternalSystemRepositoryImpl) FindByIDForUpdate(tx *gorm.DB, id int) (model.ExternalSystem, error) {
	var value model.ExternalSystem
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&value, id).Error
	return value, err
}

func (r *ExternalSystemRepositoryImpl) FindByCode(tx *gorm.DB, code string) (model.ExternalSystem, error) {
	var value model.ExternalSystem
	err := tx.Where("system_code = ?", code).First(&value).Error
	return value, err
}

func (r *ExternalSystemRepositoryImpl) Query(
	ctx context.Context,
	req request.ExternalSystemQueryReq,
	table model.SysTable,
) (response.ListResult[model.ExternalSystem], error) {
	basic := req.ToBasic()
	query := util.ExecuteQuery(r.db.WithContext(ctx).Model(&model.ExternalSystem{}), &basic, table)
	if req.SystemType != "" {
		query = query.Where("system_type = ?", req.SystemType)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if owner := strings.TrimSpace(req.Owner); owner != "" {
		pattern := "%" + strings.ToLower(owner) + "%"
		query = query.Where("LOWER(owner_identifier) LIKE ? OR LOWER(owner_name) LIKE ?", pattern, pattern)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Limit(-1).Offset(-1).Count(&total).Error; err != nil {
		return response.ListResult[model.ExternalSystem]{}, err
	}
	var values []model.ExternalSystem
	if err := query.Find(&values).Error; err != nil {
		return response.ListResult[model.ExternalSystem]{}, err
	}
	return response.ListResult[model.ExternalSystem]{Data: values, Total: int(total)}, nil
}

func (r *ExternalSystemRepositoryImpl) UpdateFields(
	tx *gorm.DB,
	id int,
	revision int,
	updates map[string]any,
) (bool, error) {
	result := tx.Model(&model.ExternalSystem{}).
		Where("id = ? AND revision = ?", id, revision).
		Updates(updates)
	return result.RowsAffected == 1, result.Error
}

func (r *ExternalSystemRepositoryImpl) HasConfigurationReferences(tx *gorm.DB, id int) (bool, error) {
	for _, table := range []string{"integration_interface_definition", "integration_credential", "integration_sync_task"} {
		if !tx.Migrator().HasTable(table) {
			continue
		}
		var count int64
		if err := tx.Table(table).Where("external_system_id = ? AND gmt_delete IS NULL", id).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

var _ repository.ExternalSystemRepository = (*ExternalSystemRepositoryImpl)(nil)
