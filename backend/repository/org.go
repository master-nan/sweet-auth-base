package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"
	"errors"
	"time"

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
	Query(context.Context, *request.OrgLegalEntityQueryReq, model.SysTable, OrgLegalEntityReadScope) (response.ListResult[model.OrgLegalEntity], error)
	FindByIdForRead(context.Context, int) (model.OrgLegalEntity, error)
	ListForTree(context.Context, OrgLegalEntityReadScope) ([]model.OrgLegalEntity, error)
	FindByIdsForDisplay(context.Context, []int) ([]model.OrgLegalEntity, error)
	FindBySourceIdentity(context.Context, string, string) (model.OrgLegalEntity, error)
	FindByCode(context.Context, string, string) (model.OrgLegalEntity, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgUnitRepository interface {
	BasicRepository[model.OrgUnit]
	Query(context.Context, *request.OrgUnitQueryReq, model.SysTable) (response.ListResult[model.OrgUnit], error)
	QueryForRead(context.Context, *request.OrgUnitQueryReq, model.SysTable, OrgReadScope, *int) (response.ListResult[model.OrgUnit], error)
	FindByIdForRead(context.Context, int) (model.OrgUnit, error)
	FindByIdsForDisplay(context.Context, []int) ([]model.OrgUnit, error)
	FindBySourceIdentity(context.Context, string, string) (model.OrgUnit, error)
	FindByCode(context.Context, string, string) (model.OrgUnit, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgStructureRepository interface {
	BasicRepository[model.OrgStructure]
	Query(context.Context, *request.OrgStructureQueryReq, model.SysTable) (response.ListResult[model.OrgStructure], error)
	QueryForRead(context.Context, *request.OrgStructureQueryReq, model.SysTable, OrgReadScope) (response.ListResult[model.OrgStructure], error)
	FindByIdForRead(context.Context, int) (model.OrgStructure, error)
	FindByIdsForDisplay(context.Context, []int) ([]model.OrgStructure, error)
	FindBySourceIdentity(context.Context, string, string) (model.OrgStructure, error)
	FindByCode(context.Context, string) (model.OrgStructure, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgStructureNodeRepository interface {
	BasicRepository[model.OrgStructureNode]
	Query(context.Context, *request.OrgStructureNodeQueryReq, model.SysTable) (response.ListResult[model.OrgStructureNode], error)
	ListByStructureForRead(context.Context, int, OrgReadScope, int) ([]model.OrgStructureNode, error)
	FindBySourceIdentity(context.Context, string, string) (model.OrgStructureNode, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgPositionRepository interface {
	BasicRepository[model.OrgPosition]
	Query(context.Context, *request.OrgPositionQueryReq, model.SysTable) (response.ListResult[model.OrgPosition], error)
	QueryForRead(context.Context, *request.OrgPositionQueryReq, model.SysTable, OrgReadScope) (response.ListResult[model.OrgPosition], error)
	FindByIdForRead(context.Context, int) (model.OrgPosition, error)
	FindByIdsForDisplay(context.Context, []int) ([]model.OrgPosition, error)
	FindBySourceIdentity(context.Context, string, string) (model.OrgPosition, error)
	FindByCode(context.Context, string, string) (model.OrgPosition, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgEmployeeRepository interface {
	BasicRepository[model.OrgEmployee]
	Query(context.Context, *request.OrgEmployeeQueryReq, model.SysTable) (response.ListResult[model.OrgEmployee], error)
	QueryForRead(context.Context, *request.OrgEmployeeQueryReq, model.SysTable, OrgReadScope) (response.ListResult[model.OrgEmployee], error)
	FindByIdForRead(context.Context, int) (model.OrgEmployee, error)
	FindByIdsForDisplay(context.Context, []int) ([]model.OrgEmployee, error)
	FindBoundUserSummaries(context.Context, []int) ([]OrgBoundUserSummary, error)
	QueryUsersForBinding(context.Context, string, int, int) (response.ListResult[OrgBindingUserOption], error)
	FindByIdForBinding(*gorm.DB, int) (model.OrgEmployee, error)
	FindUserForBinding(*gorm.DB, int) (OrgBoundUserSummary, error)
	FindByBoundUserIdForBinding(*gorm.DB, int) (model.OrgEmployee, error)
	FindBySourceIdentity(context.Context, string, string) (model.OrgEmployee, error)
	FindByEmployeeNo(context.Context, string, string) (model.OrgEmployee, error)
	FindByBoundUserId(context.Context, int) (model.OrgEmployee, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
	UpdatePlatformFields(*gorm.DB, int, map[string]any) error
}

type OrgAssignmentRepository interface {
	BasicRepository[model.OrgAssignment]
	Query(context.Context, *request.OrgAssignmentQueryReq, model.SysTable) (response.ListResult[model.OrgAssignment], error)
	QueryForRead(context.Context, *request.OrgAssignmentQueryReq, model.SysTable, OrgAssignmentReadScope) (response.ListResult[model.OrgAssignment], error)
	ListEffectiveByEmployee(context.Context, int, time.Time, int) ([]model.OrgAssignment, error)
	FindByIdForRead(context.Context, int) (model.OrgAssignment, error)
	FindBySourceIdentity(context.Context, string, string) (model.OrgAssignment, error)
	UpdateSourceFields(*gorm.DB, int, map[string]any) error
}

type OrgSyncBatchRepository interface {
	BasicRepository[model.OrgSyncBatch]
	Query(context.Context, *request.OrgSyncBatchQueryReq, model.SysTable) (response.ListResult[model.OrgSyncBatch], error)
	FindByIdForRead(context.Context, int) (model.OrgSyncBatch, error)
	FindByBatchNo(context.Context, string) (model.OrgSyncBatch, error)
}

type OrgSyncRecordRepository interface {
	BasicRepository[model.OrgSyncRecord]
	Query(context.Context, *request.OrgSyncRecordQueryReq, model.SysTable) (response.ListResult[model.OrgSyncRecord], error)
	FindByIdForRead(context.Context, int) (model.OrgSyncRecord, error)
}
