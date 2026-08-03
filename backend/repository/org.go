package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	// ErrOrganizationFieldBoundary 在调用方试图跨越外部来源管理与平台管理字段边界时，
	// 于写入前返回。
	ErrOrganizationFieldBoundary = errors.New("repository: organization field is outside the allowed update boundary")
	ErrOrganizationTxRequired    = errors.New("repository: organization update requires a transaction")
)

// OrgReadScope 在进入 Repository 前由 OrgService 规范化。
// Repository 仅将已确定的可见性边界转换为数据库条件。
type OrgReadScope struct {
	AsOf            time.Time
	IncludeDisabled bool
	IncludeHistory  bool
}

// OrgAssignmentReadScope 由 OrgService 决定。
// Repository 将选定的时间视图转换为 SQL，但不选择主任职。
type OrgAssignmentReadScope struct {
	AsOf      time.Time
	TimeScope string
}

// OrgBoundUserSummary 是 Organization Service 使用的 Repository 投影。
// 它不包含认证或授权数据。
type OrgBoundUserSummary struct {
	UserId   int
	UserName string
}

type OrgBindingUserOption struct {
	UserId   int
	UserName string
	Disabled bool
}

type OrgLegalEntityReadScope = OrgReadScope

type OrgLegalEntityRepository interface {
	BasicRepository[model.OrgLegalEntity]
	Query(*gin.Context, *request.OrgLegalEntityQueryReq, model.SysTable, OrgLegalEntityReadScope) (response.ListResult[model.OrgLegalEntity], error)
	FindByIdForRead(*gin.Context, int) (model.OrgLegalEntity, error)
	ListForTree(*gin.Context, OrgLegalEntityReadScope) ([]model.OrgLegalEntity, error)
	FindByIdsForDisplay(*gin.Context, []int) ([]model.OrgLegalEntity, error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgLegalEntity, error)
	FindByCode(*gin.Context, string, string) (model.OrgLegalEntity, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgUnitRepository interface {
	BasicRepository[model.OrgUnit]
	Query(*gin.Context, *request.OrgUnitQueryReq, model.SysTable) (response.ListResult[model.OrgUnit], error)
	QueryForRead(*gin.Context, *request.OrgUnitQueryReq, model.SysTable, OrgReadScope, *int) (response.ListResult[model.OrgUnit], error)
	FindByIdForRead(*gin.Context, int) (model.OrgUnit, error)
	FindByIdsForDisplay(*gin.Context, []int) ([]model.OrgUnit, error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgUnit, error)
	FindByCode(*gin.Context, string, string) (model.OrgUnit, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgStructureRepository interface {
	BasicRepository[model.OrgStructure]
	Query(*gin.Context, *request.OrgStructureQueryReq, model.SysTable) (response.ListResult[model.OrgStructure], error)
	QueryForRead(*gin.Context, *request.OrgStructureQueryReq, model.SysTable, OrgReadScope) (response.ListResult[model.OrgStructure], error)
	FindByIdForRead(*gin.Context, int) (model.OrgStructure, error)
	FindByIdsForDisplay(*gin.Context, []int) ([]model.OrgStructure, error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgStructure, error)
	FindByCode(*gin.Context, string) (model.OrgStructure, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgStructureNodeRepository interface {
	BasicRepository[model.OrgStructureNode]
	Query(*gin.Context, *request.OrgStructureNodeQueryReq, model.SysTable) (response.ListResult[model.OrgStructureNode], error)
	ListByStructureForRead(*gin.Context, int, OrgReadScope, int) ([]model.OrgStructureNode, error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgStructureNode, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgPositionRepository interface {
	BasicRepository[model.OrgPosition]
	Query(*gin.Context, *request.OrgPositionQueryReq, model.SysTable) (response.ListResult[model.OrgPosition], error)
	QueryForRead(*gin.Context, *request.OrgPositionQueryReq, model.SysTable, OrgReadScope) (response.ListResult[model.OrgPosition], error)
	FindByIdForRead(*gin.Context, int) (model.OrgPosition, error)
	FindByIdsForDisplay(*gin.Context, []int) ([]model.OrgPosition, error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgPosition, error)
	FindByCode(*gin.Context, string, string) (model.OrgPosition, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgEmployeeRepository interface {
	BasicRepository[model.OrgEmployee]
	Query(*gin.Context, *request.OrgEmployeeQueryReq, model.SysTable) (response.ListResult[model.OrgEmployee], error)
	QueryForRead(*gin.Context, *request.OrgEmployeeQueryReq, model.SysTable, OrgReadScope) (response.ListResult[model.OrgEmployee], error)
	FindByIdForRead(*gin.Context, int) (model.OrgEmployee, error)
	FindByIdsForDisplay(*gin.Context, []int) ([]model.OrgEmployee, error)
	FindBoundUserSummaries(*gin.Context, []int) ([]OrgBoundUserSummary, error)
	QueryUsersForBinding(*gin.Context, string, int, int) (response.ListResult[OrgBindingUserOption], error)
	FindByIdForBinding(*gorm.DB, int) (model.OrgEmployee, error)
	FindUserForBinding(*gorm.DB, int) (OrgBoundUserSummary, error)
	FindByBoundUserIdForBinding(*gorm.DB, int) (model.OrgEmployee, error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgEmployee, error)
	FindByEmployeeNo(*gin.Context, string, string) (model.OrgEmployee, error)
	FindByBoundUserId(*gin.Context, int) (model.OrgEmployee, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgAssignmentRepository interface {
	BasicRepository[model.OrgAssignment]
	Query(*gin.Context, *request.OrgAssignmentQueryReq, model.SysTable) (response.ListResult[model.OrgAssignment], error)
	QueryForRead(*gin.Context, *request.OrgAssignmentQueryReq, model.SysTable, OrgAssignmentReadScope) (response.ListResult[model.OrgAssignment], error)
	ListEffectiveByEmployee(*gin.Context, int, time.Time, int) ([]model.OrgAssignment, error)
	FindByIdForRead(*gin.Context, int) (model.OrgAssignment, error)
	FindBySourceIdentity(*gin.Context, string, string) (model.OrgAssignment, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgSyncBatchRepository interface {
	BasicRepository[model.OrgSyncBatch]
	Query(*gin.Context, *request.OrgSyncBatchQueryReq, model.SysTable) (response.ListResult[model.OrgSyncBatch], error)
	FindByIdForRead(*gin.Context, int) (model.OrgSyncBatch, error)
	FindByBatchNo(*gin.Context, string) (model.OrgSyncBatch, error)
}

type OrgSyncRecordRepository interface {
	BasicRepository[model.OrgSyncRecord]
	Query(*gin.Context, *request.OrgSyncRecordQueryReq, model.SysTable) (response.ListResult[model.OrgSyncRecord], error)
	FindByIdForRead(*gin.Context, int) (model.OrgSyncRecord, error)
}
