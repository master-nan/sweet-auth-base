package repository

import (
	"context"
	"time"

	"backend/dto/request"
	"backend/dto/response"
	"backend/model"

	"gorm.io/gorm"
)

type DataDimensionDefinitionRepository interface {
	BasicRepository[model.DataDimensionDefinition]
	GetDataDimensionDefinitionList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.DataDimensionDefinition], error)
}

type DataResourceRepository interface {
	BasicRepository[model.DataResource]
	GetDataResourceList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.DataResource], error)
	ListByTableId(context.Context, int) ([]model.DataResource, error)
}

type DataResourceOperationRepository interface {
	BasicRepository[model.DataResourceOperation]
	GetDataResourceOperationList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.DataResourceOperation], error)
	FindByStableKey(context.Context, int, string) (model.DataResourceOperation, error)
	FindByStableKeyForConfigDB(*gorm.DB, int, string) (model.DataResourceOperation, error)
	ListByResourceForConfigDB(*gorm.DB, int) ([]model.DataResourceOperation, error)
	UpdateFieldsByResourceForConfig(*gorm.DB, int, map[string]any) error
	CountByResourceForConfig(*gorm.DB, int) (int64, error)
}

type DataOwnershipFieldRepository interface {
	BasicRepository[model.DataOwnershipField]
	GetDataOwnershipFieldList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.DataOwnershipField], error)
	FindByStableKey(context.Context, int, string) (model.DataOwnershipField, error)
	FindByStableKeyForConfigDB(*gorm.DB, int, string) (model.DataOwnershipField, error)
	ListByResourceForConfigDB(*gorm.DB, int) ([]model.DataOwnershipField, error)
	ListByResource(context.Context, int) ([]model.DataOwnershipField, error)
	CountByResourceForConfig(*gorm.DB, int) (int64, error)
	CountByIdentityForConfig(*gorm.DB, string, *int, bool) (int64, error)
	CountPolicyRuleReferencesForConfig(*gorm.DB, int, string, int, bool, time.Time) (int64, error)
	ListActiveByOwnershipCodesForConfigDB(*gorm.DB, []string) ([]model.DataOwnershipField, error)
}

type DataPolicyRepository interface {
	BasicRepository[model.DataPolicy]
	GetDataPolicyList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.DataPolicy], error)
}

type DataPolicyRuleRepository interface {
	BasicRepository[model.DataPolicyRule]
	GetDataPolicyRuleList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.DataPolicyRule], error)
	FindByStableKey(context.Context, int, int) (model.DataPolicyRule, error)
	FindByStableKeyForConfigDB(*gorm.DB, int, int) (model.DataPolicyRule, error)
	ListByPolicy(context.Context, int) ([]model.DataPolicyRule, error)
	ListByPolicyForConfigDB(*gorm.DB, int) ([]model.DataPolicyRule, error)
}

type DataGrantRepository interface {
	BasicRepository[model.DataGrant]
	GetDataGrantList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.DataGrant], error)
	FindByStableKey(context.Context, string, int, int, string, int) (model.DataGrant, error)
	FindByStableKeyForConfigDB(*gorm.DB, string, int, int, string, int) (model.DataGrant, error)
	ListEffectiveBySubjects(context.Context, int, []int, int, string, time.Time) ([]model.DataGrant, error)
	ListByResourceForConfigDB(*gorm.DB, int) ([]model.DataGrant, error)
	ListByPolicyForConfigDB(*gorm.DB, int) ([]model.DataGrant, error)
	RoleExistsForConfig(*gorm.DB, int) (bool, error)
	UserExistsForConfig(*gorm.DB, int) (bool, error)
	FindActiveSubjectIDsForConfigDB(*gorm.DB, string, []int) ([]int, error)
	CountByResourceForConfig(*gorm.DB, int) (int64, error)
	CountByResourceOperationForConfig(*gorm.DB, int, string) (int64, error)
}
