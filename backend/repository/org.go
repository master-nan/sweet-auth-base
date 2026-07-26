package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	// ErrOrganizationFieldBoundary is returned before a write when a caller
	// attempts to cross the source-managed/platform-managed field boundary.
	ErrOrganizationFieldBoundary = errors.New("repository: organization field is outside the allowed update boundary")
	ErrOrganizationTxRequired    = errors.New("repository: organization update requires a transaction")
)

type OrgLegalEntityRepository interface {
	BasicRepository[model.OrgLegalEntity]
	Query(*gin.Context, *request.OrgLegalEntityQueryReq, model.SysTable) (response.ListResult[model.OrgLegalEntity], error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgLegalEntity, error)
	FindByCode(*gin.Context, string, string) (model.OrgLegalEntity, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgUnitRepository interface {
	BasicRepository[model.OrgUnit]
	Query(*gin.Context, *request.OrgUnitQueryReq, model.SysTable) (response.ListResult[model.OrgUnit], error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgUnit, error)
	FindByCode(*gin.Context, string, string) (model.OrgUnit, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgStructureRepository interface {
	BasicRepository[model.OrgStructure]
	Query(*gin.Context, *request.OrgStructureQueryReq, model.SysTable) (response.ListResult[model.OrgStructure], error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgStructure, error)
	FindByCode(*gin.Context, string) (model.OrgStructure, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgStructureNodeRepository interface {
	BasicRepository[model.OrgStructureNode]
	Query(*gin.Context, *request.OrgStructureNodeQueryReq, model.SysTable) (response.ListResult[model.OrgStructureNode], error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgStructureNode, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgPositionRepository interface {
	BasicRepository[model.OrgPosition]
	Query(*gin.Context, *request.OrgPositionQueryReq, model.SysTable) (response.ListResult[model.OrgPosition], error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgPosition, error)
	FindByCode(*gin.Context, string, string) (model.OrgPosition, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgEmployeeRepository interface {
	BasicRepository[model.OrgEmployee]
	Query(*gin.Context, *request.OrgEmployeeQueryReq, model.SysTable) (response.ListResult[model.OrgEmployee], error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgEmployee, error)
	FindByEmployeeNo(*gin.Context, string, string) (model.OrgEmployee, error)
	FindByBoundUserId(*gin.Context, int) (model.OrgEmployee, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgAssignmentRepository interface {
	BasicRepository[model.OrgAssignment]
	Query(*gin.Context, *request.OrgAssignmentQueryReq, model.SysTable) (response.ListResult[model.OrgAssignment], error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgAssignment, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgSyncBatchRepository interface {
	BasicRepository[model.OrgSyncBatch]
	Query(*gin.Context, *request.OrgSyncBatchQueryReq, model.SysTable) (response.ListResult[model.OrgSyncBatch], error)
	FindByBatchNo(*gin.Context, string) (model.OrgSyncBatch, error)
}

type OrgSyncRecordRepository interface {
	BasicRepository[model.OrgSyncRecord]
	Query(*gin.Context, *request.OrgSyncRecordQueryReq, model.SysTable) (response.ListResult[model.OrgSyncRecord], error)
}
