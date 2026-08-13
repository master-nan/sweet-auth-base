package repository

import (
	"backend/model"
	"context"
	"time"

	"gorm.io/gorm"
)

// OrganizationHRSyncRepository 只提供 HR 领域同步需要的持久化原语。
// 身份、状态、结构和冲突语义由 Service 决定。
type OrganizationHRSyncRepository interface {
	DBWithContext(context.Context) *gorm.DB
	CurrentDatabaseTime(*gorm.DB) (time.Time, error)
	LockSourceIdentity(*gorm.DB, string) error

	FindExecutionByNo(*gorm.DB, string) (model.IntegrationExecution, error)
	FindIntegrationSyncBatchByID(*gorm.DB, int) (model.IntegrationSyncBatch, error)
	FindSyncBatchByExecutionForUpdate(*gorm.DB, int) (model.OrgSyncBatch, error)
	CreateSyncBatch(*gorm.DB, *model.OrgSyncBatch) error
	UpdateSyncBatch(*gorm.DB, int, map[string]any) error

	FindLegalEntityBySource(*gorm.DB, string, string) (model.OrgLegalEntity, error)
	FindLegalEntityByCode(*gorm.DB, string, string) (model.OrgLegalEntity, error)
	CreateLegalEntity(*gorm.DB, *model.OrgLegalEntity) error
	UpdateLegalEntity(*gorm.DB, int, map[string]any) error
	ListLegalEntities(*gorm.DB, string) ([]model.OrgLegalEntity, error)

	FindOrgUnitBySource(*gorm.DB, string, string) (model.OrgUnit, error)
	FindOrgUnitByCode(*gorm.DB, string, string) (model.OrgUnit, error)
	CreateOrgUnit(*gorm.DB, *model.OrgUnit) error
	UpdateOrgUnit(*gorm.DB, int, map[string]any) error
	ListOrgUnits(*gorm.DB, string) ([]model.OrgUnit, error)
	FindPositionBySource(*gorm.DB, string, string) (model.OrgPosition, error)
	FindPositionByCode(*gorm.DB, string, string) (model.OrgPosition, error)
	CreatePosition(*gorm.DB, *model.OrgPosition) error
	UpdatePosition(*gorm.DB, int, map[string]any) error
	FindEmployeeBySource(*gorm.DB, string, string) (model.OrgEmployee, error)
	FindEmployeeByNumber(*gorm.DB, string, string) (model.OrgEmployee, error)
	CreateEmployee(*gorm.DB, *model.OrgEmployee) error
	UpdateEmployee(*gorm.DB, int, map[string]any) error
	ListAssignmentsByEmployeeForUpdate(*gorm.DB, int) ([]model.OrgAssignment, error)
	UpdateAssignment(*gorm.DB, int, map[string]any) error

	FindStructureByCode(*gorm.DB, string) (model.OrgStructure, error)
	CreateStructure(*gorm.DB, *model.OrgStructure) error
	FindStructureNodeBySource(*gorm.DB, string, string) (model.OrgStructureNode, error)
	CreateStructureNode(*gorm.DB, *model.OrgStructureNode) error
	UpdateStructureNode(*gorm.DB, int, map[string]any) error
	ListStructureNodes(*gorm.DB, int) ([]model.OrgStructureNode, error)
	ListStructureNodesBySource(*gorm.DB, string) ([]model.OrgStructureNode, error)

	FindSyncRecordForUpdate(*gorm.DB, int, string, string) (model.OrgSyncRecord, error)
	CreateSyncRecord(*gorm.DB, *model.OrgSyncRecord) error
	UpdateSyncRecord(*gorm.DB, int, map[string]any) error
}
