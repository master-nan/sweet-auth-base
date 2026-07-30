package impl

import (
	"time"

	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DataDimensionDefinitionRepositoryImpl struct {
	*BasicRepositoryImpl[model.DataDimensionDefinition]
}

type DataResourceRepositoryImpl struct {
	*BasicRepositoryImpl[model.DataResource]
}

type DataResourceOperationRepositoryImpl struct {
	*BasicRepositoryImpl[model.DataResourceOperation]
}

type DataOwnershipFieldRepositoryImpl struct {
	*BasicRepositoryImpl[model.DataOwnershipField]
}

type DataPolicyRepositoryImpl struct {
	*BasicRepositoryImpl[model.DataPolicy]
}

type DataPolicyRuleRepositoryImpl struct {
	*BasicRepositoryImpl[model.DataPolicyRule]
}

type DataGrantRepositoryImpl struct {
	*BasicRepositoryImpl[model.DataGrant]
}

var (
	_ repository.DataDimensionDefinitionRepository = (*DataDimensionDefinitionRepositoryImpl)(nil)
	_ repository.DataResourceRepository            = (*DataResourceRepositoryImpl)(nil)
	_ repository.DataResourceOperationRepository   = (*DataResourceOperationRepositoryImpl)(nil)
	_ repository.DataOwnershipFieldRepository      = (*DataOwnershipFieldRepositoryImpl)(nil)
	_ repository.DataPolicyRepository              = (*DataPolicyRepositoryImpl)(nil)
	_ repository.DataPolicyRuleRepository          = (*DataPolicyRuleRepositoryImpl)(nil)
	_ repository.DataGrantRepository               = (*DataGrantRepositoryImpl)(nil)
)

func NewDataDimensionDefinitionRepositoryImpl(primaryDB *database.PrimaryDB) *DataDimensionDefinitionRepositoryImpl {
	return &DataDimensionDefinitionRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.DataDimensionDefinition{}),
	}
}

func NewDataResourceRepositoryImpl(primaryDB *database.PrimaryDB) *DataResourceRepositoryImpl {
	return &DataResourceRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.DataResource{}),
	}
}

func NewDataResourceOperationRepositoryImpl(primaryDB *database.PrimaryDB) *DataResourceOperationRepositoryImpl {
	return &DataResourceOperationRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.DataResourceOperation{}),
	}
}

func NewDataOwnershipFieldRepositoryImpl(primaryDB *database.PrimaryDB) *DataOwnershipFieldRepositoryImpl {
	return &DataOwnershipFieldRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.DataOwnershipField{}),
	}
}

func NewDataPolicyRepositoryImpl(primaryDB *database.PrimaryDB) *DataPolicyRepositoryImpl {
	return &DataPolicyRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.DataPolicy{}),
	}
}

func NewDataPolicyRuleRepositoryImpl(primaryDB *database.PrimaryDB) *DataPolicyRuleRepositoryImpl {
	return &DataPolicyRuleRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.DataPolicyRule{}),
	}
}

func NewDataGrantRepositoryImpl(primaryDB *database.PrimaryDB) *DataGrantRepositoryImpl {
	return &DataGrantRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.DataGrant{}),
	}
}

func (r *DataDimensionDefinitionRepositoryImpl) Query(
	ctx *gin.Context,
	req *request.DataDimensionDefinitionQueryReq,
	table model.SysTable,
) (response.ListResult[model.DataDimensionDefinition], error) {
	if req == nil {
		req = &request.DataDimensionDefinitionQueryReq{}
	}
	return queryDataPermissionConfig(
		r.BasicRepositoryImpl,
		ctx,
		req.DataPermissionConfigQueryReq,
		map[string]any{
			"category":   optionalConfigString(req.Category),
			"value_type": optionalConfigString(req.ValueType),
			"state":      optionalConfigBool(req.State),
		},
		table,
		"sys_data_dimension_definition",
		dataDimensionDefinitionQueryFields,
		dataDimensionDefinitionColumns,
	)
}

func (r *DataDimensionDefinitionRepositoryImpl) FindByIdForConfig(ctx *gin.Context, id int) (model.DataDimensionDefinition, error) {
	return findDataPermissionConfigById[model.DataDimensionDefinition](r.db, ctx, id, dataDimensionDefinitionColumns)
}

func (r *DataDimensionDefinitionRepositoryImpl) FindByCode(ctx *gin.Context, code string) (model.DataDimensionDefinition, error) {
	return findDataPermissionConfigOne[model.DataDimensionDefinition](r.db, ctx, dataDimensionDefinitionColumns, "code = ?", code)
}

func (r *DataDimensionDefinitionRepositoryImpl) FindByIdsForConfig(ctx *gin.Context, ids []int) ([]model.DataDimensionDefinition, error) {
	return findDataPermissionConfigByIds[model.DataDimensionDefinition](r.db, ctx, ids, dataDimensionDefinitionColumns)
}

func (r *DataDimensionDefinitionRepositoryImpl) FindByIdForConfigDB(
	db *gorm.DB,
	id int,
) (model.DataDimensionDefinition, error) {
	return findDataPermissionConfigOneDB[model.DataDimensionDefinition](
		db,
		dataDimensionDefinitionColumns,
		"id = ?",
		id,
	)
}

func (r *DataResourceRepositoryImpl) Query(
	ctx *gin.Context,
	req *request.DataResourceQueryReq,
	table model.SysTable,
) (response.ListResult[model.DataResource], error) {
	if req == nil {
		req = &request.DataResourceQueryReq{}
	}
	return queryDataPermissionConfig(
		r.BasicRepositoryImpl,
		ctx,
		req.DataPermissionConfigQueryReq,
		map[string]any{
			"resource_type":      optionalConfigString(req.ResourceType),
			"permission_enabled": optionalConfigBool(req.PermissionEnabled),
			"state":              optionalConfigBool(req.State),
		},
		table,
		"sys_data_resource",
		dataResourceQueryFields,
		dataResourceColumns,
	)
}

func (r *DataResourceRepositoryImpl) FindByIdForConfig(ctx *gin.Context, id int) (model.DataResource, error) {
	return findDataPermissionConfigById[model.DataResource](r.db, ctx, id, dataResourceColumns)
}

func (r *DataResourceRepositoryImpl) FindByCode(ctx *gin.Context, code string) (model.DataResource, error) {
	return findDataPermissionConfigOne[model.DataResource](r.db, ctx, dataResourceColumns, "resource_code = ?", code)
}

func (r *DataResourceRepositoryImpl) FindByIdsForConfig(ctx *gin.Context, ids []int) ([]model.DataResource, error) {
	return findDataPermissionConfigByIds[model.DataResource](r.db, ctx, ids, dataResourceColumns)
}

func (r *DataResourceRepositoryImpl) FindByIdForConfigDB(db *gorm.DB, id int) (model.DataResource, error) {
	return findDataPermissionConfigOneDB[model.DataResource](db, dataResourceMutationColumns, "id = ?", id)
}

func (r *DataResourceRepositoryImpl) FindByCodeForConfigDB(db *gorm.DB, code string) (model.DataResource, error) {
	return findDataPermissionConfigOneDB[model.DataResource](db, dataResourceMutationColumns, "resource_code = ?", code)
}

func (r *DataResourceRepositoryImpl) UpdateFieldsForConfig(
	db *gorm.DB,
	id int,
	fields map[string]any,
) (bool, error) {
	result := db.Model(&model.DataResource{}).Where("id = ?", id).Updates(fields)
	return result.RowsAffected > 0, result.Error
}

func (r *DataResourceOperationRepositoryImpl) Query(
	ctx *gin.Context,
	req *request.DataResourceOperationQueryReq,
	table model.SysTable,
) (response.ListResult[model.DataResourceOperation], error) {
	if req == nil {
		req = &request.DataResourceOperationQueryReq{}
	}
	return queryDataPermissionConfig(
		r.BasicRepositoryImpl,
		ctx,
		req.DataPermissionConfigQueryReq,
		map[string]any{
			"resource_id":        optionalConfigInt(req.ResourceId),
			"operation":          optionalConfigString(req.Operation),
			"permission_enabled": optionalConfigBool(req.PermissionEnabled),
			"state":              optionalConfigBool(req.State),
		},
		table,
		"sys_data_resource_operation",
		dataResourceOperationQueryFields,
		dataResourceOperationColumns,
	)
}

func (r *DataResourceOperationRepositoryImpl) FindByIdForConfig(ctx *gin.Context, id int) (model.DataResourceOperation, error) {
	return findDataPermissionConfigById[model.DataResourceOperation](r.db, ctx, id, dataResourceOperationColumns)
}

func (r *DataResourceOperationRepositoryImpl) FindByStableKey(
	ctx *gin.Context,
	resourceId int,
	operation string,
) (model.DataResourceOperation, error) {
	return findDataPermissionConfigOne[model.DataResourceOperation](
		r.db, ctx, dataResourceOperationColumns,
		"resource_id = ? AND operation = ?", resourceId, operation,
	)
}

func (r *DataResourceOperationRepositoryImpl) FindByIdsForConfig(ctx *gin.Context, ids []int) ([]model.DataResourceOperation, error) {
	return findDataPermissionConfigByIds[model.DataResourceOperation](r.db, ctx, ids, dataResourceOperationColumns)
}

func (r *DataResourceOperationRepositoryImpl) FindByIdForConfigDB(
	db *gorm.DB,
	id int,
) (model.DataResourceOperation, error) {
	return findDataPermissionConfigOneDB[model.DataResourceOperation](db, dataResourceOperationMutationColumns, "id = ?", id)
}

func (r *DataResourceOperationRepositoryImpl) FindByStableKeyForConfigDB(
	db *gorm.DB,
	resourceId int,
	operation string,
) (model.DataResourceOperation, error) {
	return findDataPermissionConfigOneDB[model.DataResourceOperation](
		db,
		dataResourceOperationMutationColumns,
		"resource_id = ? AND operation = ?",
		resourceId,
		operation,
	)
}

func (r *DataResourceOperationRepositoryImpl) ListByResourceForConfigDB(
	db *gorm.DB,
	resourceId int,
) ([]model.DataResourceOperation, error) {
	values := make([]model.DataResourceOperation, 0)
	err := db.Select(dataResourceOperationMutationColumns).
		Where("resource_id = ?", resourceId).
		Order("operation ASC, id ASC").
		Find(&values).Error
	return values, err
}

func (r *DataResourceOperationRepositoryImpl) UpdateFieldsForConfig(
	db *gorm.DB,
	id int,
	fields map[string]any,
) (bool, error) {
	result := db.Model(&model.DataResourceOperation{}).Where("id = ?", id).Updates(fields)
	return result.RowsAffected > 0, result.Error
}

func (r *DataResourceOperationRepositoryImpl) UpdateFieldsByResourceForConfig(
	db *gorm.DB,
	resourceId int,
	fields map[string]any,
) error {
	return db.Model(&model.DataResourceOperation{}).
		Where("resource_id = ?", resourceId).
		Updates(fields).Error
}

func (r *DataResourceOperationRepositoryImpl) CountByResourceForConfig(
	db *gorm.DB,
	resourceId int,
) (int64, error) {
	var count int64
	err := db.Model(&model.DataResourceOperation{}).
		Where("resource_id = ?", resourceId).
		Count(&count).Error
	return count, err
}

func (r *DataOwnershipFieldRepositoryImpl) Query(
	ctx *gin.Context,
	req *request.DataOwnershipFieldQueryReq,
	table model.SysTable,
) (response.ListResult[model.DataOwnershipField], error) {
	if req == nil {
		req = &request.DataOwnershipFieldQueryReq{}
	}
	return queryDataPermissionConfig(
		r.BasicRepositoryImpl,
		ctx,
		req.DataPermissionConfigQueryReq,
		map[string]any{
			"resource_id":  optionalConfigInt(req.ResourceId),
			"dimension_id": optionalConfigInt(req.DimensionId),
			"binding_type": optionalConfigString(req.BindingType),
			"value_type":   optionalConfigString(req.ValueType),
			"state":        optionalConfigBool(req.State),
		},
		table,
		"sys_data_ownership_field",
		dataOwnershipFieldQueryFields,
		dataOwnershipFieldColumns,
	)
}

func (r *DataOwnershipFieldRepositoryImpl) FindByIdForConfig(ctx *gin.Context, id int) (model.DataOwnershipField, error) {
	return findDataPermissionConfigById[model.DataOwnershipField](r.db, ctx, id, dataOwnershipFieldColumns)
}

func (r *DataOwnershipFieldRepositoryImpl) FindByStableKey(
	ctx *gin.Context,
	resourceId int,
	ownershipCode string,
) (model.DataOwnershipField, error) {
	return findDataPermissionConfigOne[model.DataOwnershipField](
		r.db, ctx, dataOwnershipFieldColumns,
		"resource_id = ? AND ownership_code = ?", resourceId, ownershipCode,
	)
}

func (r *DataOwnershipFieldRepositoryImpl) FindByIdsForConfig(ctx *gin.Context, ids []int) ([]model.DataOwnershipField, error) {
	return findDataPermissionConfigByIds[model.DataOwnershipField](r.db, ctx, ids, dataOwnershipFieldColumns)
}

func (r *DataOwnershipFieldRepositoryImpl) FindByIdForConfigDB(
	db *gorm.DB,
	id int,
) (model.DataOwnershipField, error) {
	return findDataPermissionConfigOneDB[model.DataOwnershipField](
		db,
		dataOwnershipFieldMutationColumns,
		"id = ?",
		id,
	)
}

func (r *DataOwnershipFieldRepositoryImpl) FindByStableKeyForConfigDB(
	db *gorm.DB,
	resourceId int,
	ownershipCode string,
) (model.DataOwnershipField, error) {
	return findDataPermissionConfigOneDB[model.DataOwnershipField](
		db,
		dataOwnershipFieldMutationColumns,
		"resource_id = ? AND ownership_code = ?",
		resourceId,
		ownershipCode,
	)
}

func (r *DataOwnershipFieldRepositoryImpl) ListByResourceForConfigDB(
	db *gorm.DB,
	resourceId int,
) ([]model.DataOwnershipField, error) {
	values := make([]model.DataOwnershipField, 0)
	err := db.Select(dataOwnershipFieldMutationColumns).
		Where("resource_id = ?", resourceId).
		Order("ownership_code ASC, id ASC").
		Find(&values).Error
	return values, err
}

func (r *DataOwnershipFieldRepositoryImpl) UpdateFieldsForConfig(
	db *gorm.DB,
	id int,
	fields map[string]any,
) (bool, error) {
	result := db.Model(&model.DataOwnershipField{}).Where("id = ?", id).Updates(fields)
	return result.RowsAffected > 0, result.Error
}

func (r *DataOwnershipFieldRepositoryImpl) CountByResourceForConfig(db *gorm.DB, resourceId int) (int64, error) {
	var count int64
	err := db.Model(&model.DataOwnershipField{}).
		Where("resource_id = ?", resourceId).
		Count(&count).Error
	return count, err
}

func (r *DataOwnershipFieldRepositoryImpl) CountByIdentityForConfig(
	db *gorm.DB,
	ownershipCode string,
	dimensionId *int,
	activeOnly bool,
) (int64, error) {
	query := db.Model(&model.DataOwnershipField{}).
		Where("ownership_code = ?", ownershipCode)
	if dimensionId != nil {
		query = query.Where("dimension_id = ?", *dimensionId)
	}
	if activeOnly {
		query = query.Where("state = ?", true)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r *DataOwnershipFieldRepositoryImpl) CountPolicyRuleReferencesForConfig(
	db *gorm.DB,
	resourceId int,
	ownershipCode string,
	dimensionId int,
	activeOnly bool,
	asOf time.Time,
) (int64, error) {
	query := db.Table("sys_data_policy_rule AS rule").
		Joins("JOIN sys_data_policy AS policy ON policy.id = rule.policy_id AND policy.gmt_delete IS NULL").
		Joins("JOIN sys_data_grant AS grant_record ON grant_record.policy_id = policy.id AND grant_record.gmt_delete IS NULL").
		Where("rule.gmt_delete IS NULL").
		Where("grant_record.resource_id = ?", resourceId).
		Where("rule.ownership_code = ? AND rule.dimension_id = ?", ownershipCode, dimensionId)
	if activeOnly {
		query = query.
			Where("rule.state = ? AND policy.state = ? AND grant_record.state = ?", true, true, true).
			Where("(grant_record.valid_from IS NULL OR grant_record.valid_from <= ?)", asOf).
			Where("(grant_record.valid_to IS NULL OR grant_record.valid_to >= ?)", asOf)
	}
	var count int64
	err := query.Distinct("rule.id").Count(&count).Error
	return count, err
}

func (r *DataPolicyRepositoryImpl) Query(
	ctx *gin.Context,
	req *request.DataPolicyQueryReq,
	table model.SysTable,
) (response.ListResult[model.DataPolicy], error) {
	if req == nil {
		req = &request.DataPolicyQueryReq{}
	}
	return queryDataPermissionConfig(
		r.BasicRepositoryImpl,
		ctx,
		req.DataPermissionConfigQueryReq,
		map[string]any{
			"policy_type": optionalConfigString(req.PolicyType),
			"state":       optionalConfigBool(req.State),
		},
		table,
		"sys_data_policy",
		dataPolicyQueryFields,
		dataPolicyColumns,
	)
}

func (r *DataPolicyRepositoryImpl) FindByIdForConfig(ctx *gin.Context, id int) (model.DataPolicy, error) {
	return findDataPermissionConfigById[model.DataPolicy](r.db, ctx, id, dataPolicyColumns)
}

func (r *DataPolicyRepositoryImpl) FindByCode(ctx *gin.Context, code string) (model.DataPolicy, error) {
	return findDataPermissionConfigOne[model.DataPolicy](r.db, ctx, dataPolicyColumns, "code = ?", code)
}

func (r *DataPolicyRepositoryImpl) FindByIdsForConfig(ctx *gin.Context, ids []int) ([]model.DataPolicy, error) {
	return findDataPermissionConfigByIds[model.DataPolicy](r.db, ctx, ids, dataPolicyColumns)
}

func (r *DataPolicyRepositoryImpl) FindByIdForConfigDB(db *gorm.DB, id int) (model.DataPolicy, error) {
	return findDataPermissionConfigOneDB[model.DataPolicy](db, dataPolicyColumns, "id = ?", id)
}

func (r *DataPolicyRepositoryImpl) FindByCodeForConfigDB(db *gorm.DB, code string) (model.DataPolicy, error) {
	return findDataPermissionConfigOneDB[model.DataPolicy](db, dataPolicyColumns, "code = ?", code)
}

func (r *DataPolicyRepositoryImpl) UpdateFieldsForConfig(
	db *gorm.DB,
	id int,
	fields map[string]any,
) (bool, error) {
	result := db.Model(&model.DataPolicy{}).Where("id = ?", id).Updates(fields)
	return result.RowsAffected > 0, result.Error
}

func (r *DataPolicyRuleRepositoryImpl) Query(
	ctx *gin.Context,
	req *request.DataPolicyRuleQueryReq,
	table model.SysTable,
) (response.ListResult[model.DataPolicyRule], error) {
	if req == nil {
		req = &request.DataPolicyRuleQueryReq{}
	}
	return queryDataPermissionConfig(
		r.BasicRepositoryImpl,
		ctx,
		req.DataPermissionConfigQueryReq,
		map[string]any{
			"policy_id":      optionalConfigInt(req.PolicyId),
			"dimension_id":   optionalConfigInt(req.DimensionId),
			"ownership_code": optionalConfigString(req.OwnershipCode),
			"scope_source":   optionalConfigString(req.ScopeSource),
			"relation":       optionalConfigString(req.Relation),
			"operator":       optionalConfigString(req.Operator),
			"state":          optionalConfigBool(req.State),
		},
		table,
		"sys_data_policy_rule",
		dataPolicyRuleQueryFields,
		dataPolicyRuleColumns,
	)
}

func (r *DataPolicyRuleRepositoryImpl) FindByIdForConfig(ctx *gin.Context, id int) (model.DataPolicyRule, error) {
	return findDataPermissionConfigById[model.DataPolicyRule](r.db, ctx, id, dataPolicyRuleColumns)
}

func (r *DataPolicyRuleRepositoryImpl) FindByStableKey(
	ctx *gin.Context,
	policyId int,
	sequence int,
) (model.DataPolicyRule, error) {
	return findDataPermissionConfigOne[model.DataPolicyRule](
		r.db, ctx, dataPolicyRuleColumns,
		"policy_id = ? AND sequence = ?", policyId, sequence,
	)
}

func (r *DataPolicyRuleRepositoryImpl) FindByIdsForConfig(ctx *gin.Context, ids []int) ([]model.DataPolicyRule, error) {
	return findDataPermissionConfigByIds[model.DataPolicyRule](r.db, ctx, ids, dataPolicyRuleColumns)
}

func (r *DataPolicyRuleRepositoryImpl) FindByIdForConfigDB(db *gorm.DB, id int) (model.DataPolicyRule, error) {
	return findDataPermissionConfigOneDB[model.DataPolicyRule](db, dataPolicyRuleColumns, "id = ?", id)
}

func (r *DataPolicyRuleRepositoryImpl) FindByStableKeyForConfigDB(
	db *gorm.DB,
	policyId int,
	sequence int,
) (model.DataPolicyRule, error) {
	return findDataPermissionConfigOneDB[model.DataPolicyRule](
		db,
		dataPolicyRuleColumns,
		"policy_id = ? AND sequence = ?",
		policyId,
		sequence,
	)
}

func (r *DataPolicyRuleRepositoryImpl) ListByPolicyForConfigDB(
	db *gorm.DB,
	policyId int,
) ([]model.DataPolicyRule, error) {
	values := make([]model.DataPolicyRule, 0)
	err := db.Select(dataPolicyRuleColumns).
		Where("policy_id = ?", policyId).
		Order("sequence ASC, id ASC").
		Find(&values).Error
	return values, err
}

func (r *DataPolicyRuleRepositoryImpl) UpdateFieldsForConfig(
	db *gorm.DB,
	id int,
	fields map[string]any,
) (bool, error) {
	result := db.Model(&model.DataPolicyRule{}).Where("id = ?", id).Updates(fields)
	return result.RowsAffected > 0, result.Error
}

func (r *DataGrantRepositoryImpl) Query(
	ctx *gin.Context,
	req *request.DataGrantQueryReq,
	table model.SysTable,
) (response.ListResult[model.DataGrant], error) {
	if req == nil {
		req = &request.DataGrantQueryReq{}
	}
	return queryDataPermissionConfig(
		r.BasicRepositoryImpl,
		ctx,
		req.DataPermissionConfigQueryReq,
		map[string]any{
			"subject_type": optionalConfigString(req.SubjectType),
			"subject_id":   optionalConfigInt(req.SubjectId),
			"resource_id":  optionalConfigInt(req.ResourceId),
			"operation":    optionalConfigString(req.Operation),
			"policy_id":    optionalConfigInt(req.PolicyId),
			"state":        optionalConfigBool(req.State),
		},
		table,
		"sys_data_grant",
		dataGrantQueryFields,
		dataGrantColumns,
	)
}

func (r *DataGrantRepositoryImpl) FindByIdForConfig(ctx *gin.Context, id int) (model.DataGrant, error) {
	return findDataPermissionConfigById[model.DataGrant](r.db, ctx, id, dataGrantColumns)
}

func (r *DataGrantRepositoryImpl) FindByStableKey(
	ctx *gin.Context,
	subjectType string,
	subjectId int,
	resourceId int,
	operation string,
	policyId int,
) (model.DataGrant, error) {
	return findDataPermissionConfigOne[model.DataGrant](
		r.db, ctx, dataGrantColumns,
		"subject_type = ? AND subject_id = ? AND resource_id = ? AND operation = ? AND policy_id = ?",
		subjectType, subjectId, resourceId, operation, policyId,
	)
}

func (r *DataGrantRepositoryImpl) FindByIdsForConfig(ctx *gin.Context, ids []int) ([]model.DataGrant, error) {
	return findDataPermissionConfigByIds[model.DataGrant](r.db, ctx, ids, dataGrantColumns)
}

func (r *DataGrantRepositoryImpl) CountByResourceForConfig(db *gorm.DB, resourceId int) (int64, error) {
	var count int64
	err := db.Model(&model.DataGrant{}).
		Where("resource_id = ?", resourceId).
		Count(&count).Error
	return count, err
}

func (r *DataGrantRepositoryImpl) CountByResourceOperationForConfig(
	db *gorm.DB,
	resourceId int,
	operation string,
) (int64, error) {
	var count int64
	err := db.Model(&model.DataGrant{}).
		Where("resource_id = ? AND operation = ?", resourceId, operation).
		Count(&count).Error
	return count, err
}

func queryDataPermissionConfig[T any](
	repo repository.BasicRepository[T],
	ctx *gin.Context,
	req request.DataPermissionConfigQueryReq,
	typedFilters map[string]any,
	table model.SysTable,
	tableCode string,
	allowedFields map[string]struct{},
	columns []string,
) (response.ListResult[T], error) {
	basic := req.ToBasic()
	basic.Filters = make(map[string]any, len(typedFilters))
	for field, value := range typedFilters {
		if value != nil {
			basic.Filters[field] = value
		}
	}

	readRepo := repo
	if ctx != nil {
		readRepo = readRepo.WithContext(ctx)
	}
	rows := make([]T, 0)
	total, err := readRepo.
		WithSelect(columns...).
		PaginateAndCountAsync(
			&basic,
			&rows,
			dataPermissionConfigQueryTable(table, tableCode, allowedFields),
		)
	return response.ListResult[T]{Data: rows, Total: int(total)}, err
}

func findDataPermissionConfigById[T any](
	db *gorm.DB,
	ctx *gin.Context,
	id int,
	columns []string,
) (T, error) {
	return findDataPermissionConfigOne[T](db, ctx, columns, "id = ?", id)
}

func findDataPermissionConfigOne[T any](
	db *gorm.DB,
	ctx *gin.Context,
	columns []string,
	query string,
	args ...any,
) (T, error) {
	return findDataPermissionConfigOneDB[T](
		dataPermissionConfigDB(db, ctx),
		columns,
		query,
		args...,
	)
}

func findDataPermissionConfigOneDB[T any](
	db *gorm.DB,
	columns []string,
	query string,
	args ...any,
) (T, error) {
	var value T
	err := db.
		Select(columns).
		Where(query, args...).
		First(&value).Error
	return value, err
}

func findDataPermissionConfigByIds[T any](
	db *gorm.DB,
	ctx *gin.Context,
	ids []int,
	columns []string,
) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}
	values := make([]T, 0, len(ids))
	err := dataPermissionConfigDB(db, ctx).
		Select(columns).
		Where("id IN ?", ids).
		Order("id ASC").
		Find(&values).Error
	return values, err
}

func dataPermissionConfigDB(db *gorm.DB, ctx *gin.Context) *gorm.DB {
	if ctx == nil {
		return db
	}
	return db.WithContext(ctx)
}

func dataPermissionConfigQueryTable(
	table model.SysTable,
	tableCode string,
	allowedFields map[string]struct{},
) model.SysTable {
	table.TableCode = tableCode
	fields := make([]model.SysTableField, 0, len(table.TableFields))
	for _, field := range table.TableFields {
		if _, ok := allowedFields[field.FieldCode]; !ok {
			continue
		}
		if field.IsPrimaryKey ||
			field.IsListShow ||
			field.IsQuickSearch ||
			field.IsAdvancedSearch ||
			field.IsSort {
			fields = append(fields, field)
		}
	}
	table.TableFields = fields
	return table
}

func optionalConfigString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func optionalConfigInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalConfigBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func dataPermissionConfigFieldSet(fields ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		result[field] = struct{}{}
	}
	return result
}

var (
	dataDimensionDefinitionQueryFields = dataPermissionConfigFieldSet(
		"id", "code", "name", "category", "value_type", "provider_code", "selector_type", "state",
	)
	dataResourceQueryFields = dataPermissionConfigFieldSet(
		"id", "resource_code", "name", "resource_type", "adapter_code", "permission_enabled", "state",
	)
	dataResourceOperationQueryFields = dataPermissionConfigFieldSet(
		"id", "resource_id", "operation", "permission_enabled", "state",
	)
	dataOwnershipFieldQueryFields = dataPermissionConfigFieldSet(
		"id", "resource_id", "ownership_code", "dimension_id", "binding_type", "value_type", "state",
	)
	dataPolicyQueryFields = dataPermissionConfigFieldSet(
		"id", "code", "name", "policy_type", "state",
	)
	dataPolicyRuleQueryFields = dataPermissionConfigFieldSet(
		"id", "policy_id", "sequence", "dimension_id", "ownership_code", "scope_source", "relation", "operator", "structure_code", "state",
	)
	dataGrantQueryFields = dataPermissionConfigFieldSet(
		"id", "subject_type", "subject_id", "resource_id", "operation", "policy_id", "valid_from", "valid_to", "state",
	)
)

var (
	dataDimensionDefinitionColumns = []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "category",
		"value_type", "provider_code", "selector_type",
	}
	dataResourceColumns = []string{
		"id", "gmt_create", "gmt_modify", "state", "resource_code", "name",
		"resource_type", "table_id", "service_code", "report_definition_id",
		"adapter_code", "permission_enabled",
	}
	dataResourceOperationColumns = []string{
		"id", "gmt_create", "gmt_modify", "state", "resource_id", "operation",
		"permission_enabled",
	}
	dataResourceMutationColumns = append(
		append([]string(nil), dataResourceColumns...),
		"description",
	)
	dataResourceOperationMutationColumns = append(
		append([]string(nil), dataResourceOperationColumns...),
		"description",
	)
	dataOwnershipFieldColumns = []string{
		"id", "gmt_create", "gmt_modify", "state", "resource_id", "ownership_code",
		"dimension_id", "binding_type", "table_field_id", "adapter_field_code", "value_type",
	}
	dataOwnershipFieldMutationColumns = append(
		append([]string(nil), dataOwnershipFieldColumns...),
		"description",
	)
	dataPolicyColumns = []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "policy_type",
		"description",
	}
	dataPolicyRuleColumns = []string{
		"id", "gmt_create", "gmt_modify", "state", "policy_id", "sequence",
		"dimension_id", "ownership_code", "scope_source", "relation", "operator",
		"specified_values", "structure_code", "description",
	}
	dataGrantColumns = []string{
		"id", "gmt_create", "gmt_modify", "state", "subject_type", "subject_id",
		"resource_id", "operation", "policy_id", "valid_from", "valid_to",
	}
)
