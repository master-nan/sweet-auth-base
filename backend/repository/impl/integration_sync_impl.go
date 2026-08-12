package impl

import (
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IntegrationSyncTaskRepositoryImpl struct {
	*BasicRepositoryImpl[model.IntegrationSyncTask]
}

func NewIntegrationSyncTaskRepositoryImpl(primaryDB *database.PrimaryDB) *IntegrationSyncTaskRepositoryImpl {
	return &IntegrationSyncTaskRepositoryImpl{BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.IntegrationSyncTask{})}
}

func (r *IntegrationSyncTaskRepositoryImpl) NextVersion(tx *gorm.DB, code string) (int, error) {
	var value int
	err := tx.Model(&model.IntegrationSyncTask{}).Where("task_code = ?", code).Select("COALESCE(MAX(version), 0)").Scan(&value).Error
	return value + 1, err
}

func (r *IntegrationSyncTaskRepositoryImpl) HasEnabledVersion(tx *gorm.DB, code string, excludeID int) (bool, error) {
	var count int64
	err := tx.Model(&model.IntegrationSyncTask{}).Where("task_code = ? AND status = ? AND id <> ?", code, model.IntegrationSyncTaskStatusEnabled, excludeID).Count(&count).Error
	return count > 0, err
}

func (r *IntegrationSyncTaskRepositoryImpl) FindVersionsByCodeForUpdate(tx *gorm.DB, code string) ([]model.IntegrationSyncTask, error) {
	var values []model.IntegrationSyncTask
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("task_code = ?", code).Order("version ASC").Find(&values).Error
	return values, err
}

type IntegrationSyncBatchRepositoryImpl struct {
	*BasicRepositoryImpl[model.IntegrationSyncBatch]
}

func NewIntegrationSyncBatchRepositoryImpl(primaryDB *database.PrimaryDB) *IntegrationSyncBatchRepositoryImpl {
	return &IntegrationSyncBatchRepositoryImpl{BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.IntegrationSyncBatch{})}
}

func (r *IntegrationSyncBatchRepositoryImpl) CountActiveByTaskCode(tx *gorm.DB, code string) (int64, error) {
	var count int64
	err := tx.Model(&model.IntegrationSyncBatch{}).Where("task_code = ? AND status IN ?", code, []string{model.IntegrationSyncBatchStatusCreated, model.IntegrationSyncBatchStatusRunning}).Count(&count).Error
	return count, err
}

func (r *IntegrationSyncBatchRepositoryImpl) FindByBatchNo(value string) (model.IntegrationSyncBatch, error) {
	return r.FindByField("batch_no", value)
}

func (r *IntegrationSyncBatchRepositoryImpl) FindByTriggerKey(value string) (model.IntegrationSyncBatch, error) {
	return r.FindByField("trigger_key", value)
}

func (r *IntegrationSyncBatchRepositoryImpl) CurrentDatabaseTime(tx *gorm.DB) (time.Time, error) {
	var now time.Time
	if tx.Dialector.Name() == "postgres" {
		err := tx.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&now).Error
		return now.UTC(), err
	}
	var epoch int64
	err := tx.Raw("SELECT unixepoch()").Scan(&epoch).Error
	return time.Unix(epoch, 0).UTC(), err
}

func (r *IntegrationSyncBatchRepositoryImpl) FindScheduledCandidates(tx *gorm.DB, limit int) ([]model.IntegrationSyncTask, error) {
	var values []model.IntegrationSyncTask
	if limit <= 0 {
		return values, nil
	}
	dueExpression := "next_scheduled_at <= (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')"
	if tx.Dialector.Name() == "sqlite" {
		dueExpression = "datetime(next_scheduled_at) <= CURRENT_TIMESTAMP"
	}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND schedule_type = ? AND next_scheduled_at IS NOT NULL AND "+dueExpression, model.IntegrationSyncTaskStatusEnabled, model.IntegrationSyncScheduleCron).
		Order("next_scheduled_at ASC, id ASC").Limit(limit).Find(&values).Error
	return values, err
}

func (r *IntegrationSyncBatchRepositoryImpl) FindActiveBatches(ctx context.Context, limit int) ([]model.IntegrationSyncBatch, error) {
	values := make([]model.IntegrationSyncBatch, 0)
	if limit <= 0 {
		return values, nil
	}
	err := r.DBWithContext(ctx).Where("status IN ?", []string{model.IntegrationSyncBatchStatusCreated, model.IntegrationSyncBatchStatusRunning}).
		Order("gmt_create ASC, id ASC").Limit(limit).Find(&values).Error
	return values, err
}

var _ repository.IntegrationSyncTaskRepository = (*IntegrationSyncTaskRepositoryImpl)(nil)
var _ repository.IntegrationSyncBatchRepository = (*IntegrationSyncBatchRepositoryImpl)(nil)
