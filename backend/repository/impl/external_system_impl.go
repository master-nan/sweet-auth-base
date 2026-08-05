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

type ExternalSystemRepositoryImpl struct {
	*BasicRepositoryImpl[model.ExternalSystem]
}

func NewExternalSystemRepositoryImpl(primaryDB *database.PrimaryDB) *ExternalSystemRepositoryImpl {
	return &ExternalSystemRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.ExternalSystem{}),
	}
}

func (r *ExternalSystemRepositoryImpl) GetExternalSystemList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.ExternalSystem], error) {
	var values []model.ExternalSystem
	total, err := r.WithContext(ctx).PaginateAndCountAsync(basic, &values, table)
	return response.ListResult[model.ExternalSystem]{Data: values, Total: int(total)}, err
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
