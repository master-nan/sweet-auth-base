package repository

import (
	"time"

	"backend/dto/request"
	"backend/dto/response"
	"backend/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DataDimensionDefinitionRepository interface {
	BasicRepository[model.DataDimensionDefinition]
	Query(*gin.Context, *request.DataDimensionDefinitionQueryReq, model.SysTable) (response.ListResult[model.DataDimensionDefinition], error)
	FindByIdForConfig(*gin.Context, int) (model.DataDimensionDefinition, error)
	FindByCode(*gin.Context, string) (model.DataDimensionDefinition, error)
	FindByIdsForConfig(*gin.Context, []int) ([]model.DataDimensionDefinition, error)
	FindByIdForConfigDB(*gorm.DB, int) (model.DataDimensionDefinition, error)
}

type DataResourceRepository interface {
	BasicRepository[model.DataResource]
	Query(*gin.Context, *request.DataResourceQueryReq, model.SysTable) (response.ListResult[model.DataResource], error)
	FindByIdForConfig(*gin.Context, int) (model.DataResource, error)
	FindByCode(*gin.Context, string) (model.DataResource, error)
	FindByIdsForConfig(*gin.Context, []int) ([]model.DataResource, error)
	FindByIdForConfigDB(*gorm.DB, int) (model.DataResource, error)
	FindByCodeForConfigDB(*gorm.DB, string) (model.DataResource, error)
	ListByTableId(*gin.Context, int) ([]model.DataResource, error)
	UpdateFieldsForConfig(*gorm.DB, int, map[string]any) (bool, error)
}

type DataResourceOperationRepository interface {
	BasicRepository[model.DataResourceOperation]
	Query(*gin.Context, *request.DataResourceOperationQueryReq, model.SysTable) (response.ListResult[model.DataResourceOperation], error)
	FindByIdForConfig(*gin.Context, int) (model.DataResourceOperation, error)
	FindByStableKey(*gin.Context, int, string) (model.DataResourceOperation, error)
	FindByIdsForConfig(*gin.Context, []int) ([]model.DataResourceOperation, error)
	FindByIdForConfigDB(*gorm.DB, int) (model.DataResourceOperation, error)
	FindByStableKeyForConfigDB(*gorm.DB, int, string) (model.DataResourceOperation, error)
	ListByResourceForConfigDB(*gorm.DB, int) ([]model.DataResourceOperation, error)
	UpdateFieldsForConfig(*gorm.DB, int, map[string]any) (bool, error)
	UpdateFieldsByResourceForConfig(*gorm.DB, int, map[string]any) error
	CountByResourceForConfig(*gorm.DB, int) (int64, error)
}

type DataOwnershipFieldRepository interface {
	BasicRepository[model.DataOwnershipField]
	Query(*gin.Context, *request.DataOwnershipFieldQueryReq, model.SysTable) (response.ListResult[model.DataOwnershipField], error)
	FindByIdForConfig(*gin.Context, int) (model.DataOwnershipField, error)
	FindByStableKey(*gin.Context, int, string) (model.DataOwnershipField, error)
	FindByIdsForConfig(*gin.Context, []int) ([]model.DataOwnershipField, error)
	FindByIdForConfigDB(*gorm.DB, int) (model.DataOwnershipField, error)
	FindByStableKeyForConfigDB(*gorm.DB, int, string) (model.DataOwnershipField, error)
	ListByResourceForConfigDB(*gorm.DB, int) ([]model.DataOwnershipField, error)
	ListByResource(*gin.Context, int) ([]model.DataOwnershipField, error)
	UpdateFieldsForConfig(*gorm.DB, int, map[string]any) (bool, error)
	CountByResourceForConfig(*gorm.DB, int) (int64, error)
	CountByIdentityForConfig(*gorm.DB, string, *int, bool) (int64, error)
	CountPolicyRuleReferencesForConfig(*gorm.DB, int, string, int, bool, time.Time) (int64, error)
}

type DataPolicyRepository interface {
	BasicRepository[model.DataPolicy]
	Query(*gin.Context, *request.DataPolicyQueryReq, model.SysTable) (response.ListResult[model.DataPolicy], error)
	FindByIdForConfig(*gin.Context, int) (model.DataPolicy, error)
	FindByCode(*gin.Context, string) (model.DataPolicy, error)
	FindByIdsForConfig(*gin.Context, []int) ([]model.DataPolicy, error)
	FindByIdForConfigDB(*gorm.DB, int) (model.DataPolicy, error)
	FindByCodeForConfigDB(*gorm.DB, string) (model.DataPolicy, error)
	UpdateFieldsForConfig(*gorm.DB, int, map[string]any) (bool, error)
}

type DataPolicyRuleRepository interface {
	BasicRepository[model.DataPolicyRule]
	Query(*gin.Context, *request.DataPolicyRuleQueryReq, model.SysTable) (response.ListResult[model.DataPolicyRule], error)
	FindByIdForConfig(*gin.Context, int) (model.DataPolicyRule, error)
	FindByStableKey(*gin.Context, int, int) (model.DataPolicyRule, error)
	FindByIdsForConfig(*gin.Context, []int) ([]model.DataPolicyRule, error)
	FindByIdForConfigDB(*gorm.DB, int) (model.DataPolicyRule, error)
	FindByStableKeyForConfigDB(*gorm.DB, int, int) (model.DataPolicyRule, error)
	ListByPolicy(*gin.Context, int) ([]model.DataPolicyRule, error)
	ListByPolicyForConfigDB(*gorm.DB, int) ([]model.DataPolicyRule, error)
	UpdateFieldsForConfig(*gorm.DB, int, map[string]any) (bool, error)
}

type DataGrantRepository interface {
	BasicRepository[model.DataGrant]
	Query(*gin.Context, *request.DataGrantQueryReq, model.SysTable) (response.ListResult[model.DataGrant], error)
	FindByIdForConfig(*gin.Context, int) (model.DataGrant, error)
	FindByStableKey(*gin.Context, string, int, int, string, int) (model.DataGrant, error)
	FindByIdsForConfig(*gin.Context, []int) ([]model.DataGrant, error)
	FindByIdForConfigDB(*gorm.DB, int) (model.DataGrant, error)
	FindByStableKeyForConfigDB(*gorm.DB, string, int, int, string, int) (model.DataGrant, error)
	ListEffectiveBySubjects(*gin.Context, int, []int, int, string, time.Time) ([]model.DataGrant, error)
	UpdateFieldsForConfig(*gorm.DB, int, map[string]any) (bool, error)
	ListByResourceForConfigDB(*gorm.DB, int) ([]model.DataGrant, error)
	ListByPolicyForConfigDB(*gorm.DB, int) ([]model.DataGrant, error)
	RoleExistsForConfig(*gorm.DB, int) (bool, error)
	UserExistsForConfig(*gorm.DB, int) (bool, error)
	CountByResourceForConfig(*gorm.DB, int) (int64, error)
	CountByResourceOperationForConfig(*gorm.DB, int, string) (int64, error)
}
