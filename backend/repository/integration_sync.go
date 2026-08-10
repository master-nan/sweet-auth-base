package repository

import (
	"backend/model"
	"time"

	"gorm.io/gorm"
)

type IntegrationSyncTaskRepository interface {
	BasicRepository[model.IntegrationSyncTask]
	NextVersion(*gorm.DB, string) (int, error)
	HasEnabledVersion(*gorm.DB, string, int) (bool, error)
	FindVersionsByCodeForUpdate(*gorm.DB, string) ([]model.IntegrationSyncTask, error)
}

type IntegrationSyncBatchRepository interface {
	BasicRepository[model.IntegrationSyncBatch]
	CountActiveByTaskCode(*gorm.DB, string) (int64, error)
	FindByBatchNo(string) (model.IntegrationSyncBatch, error)
	FindByTriggerKey(string) (model.IntegrationSyncBatch, error)
	FindScheduledCandidates(*gorm.DB, time.Time, int) ([]model.IntegrationSyncTask, error)
}
