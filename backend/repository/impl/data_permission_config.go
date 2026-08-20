package impl

import (
	"context"
	"time"

	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"

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

func (r *DataDimensionDefinitionRepositoryImpl) GetDataDimensionDefinitionList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.DataDimensionDefinition], error) {
	return getDataPermissionConfigList(
		r.BasicRepositoryImpl,
		ctx,
		basic,
		table,
		"sys_data_dimension_definition",
		dataDimensionDefinitionQueryFields,
		dataDimensionDefinitionColumns,
	)
}

func (r *DataResourceRepositoryImpl) GetDataResourceList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.DataResource], error) {
	return getDataPermissionConfigList(
		r.BasicRepositoryImpl,
		ctx,
		basic,
		table,
		"sys_data_resource",
		dataResourceQueryFields,
		dataResourceColumns,
	)
}

func (r *DataResourceRepositoryImpl) ListByTableId(
	ctx context.Context,
	tableId int,
) ([]model.DataResource, error) {
	values := make([]model.DataResource, 0)
	err := r.DBWithContext(ctx).
		Select(dataResourceColumns).
		Where("resource_type = ? AND table_id = ?", model.DataResourceTypeLowCodeTable, tableId).
		Order("id ASC").
		Find(&values).Error
	return values, err
}

func (r *DataResourceOperationRepositoryImpl) GetDataResourceOperationList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.DataResourceOperation], error) {
	return getDataPermissionConfigList(
		r.BasicRepositoryImpl,
		ctx,
		basic,
		table,
		"sys_data_resource_operation",
		dataResourceOperationQueryFields,
		dataResourceOperationColumns,
	)
}

func (r *DataResourceOperationRepositoryImpl) FindByStableKey(
	ctx context.Context,
	resourceId int,
	operation string,
) (model.DataResourceOperation, error) {
	var value model.DataResourceOperation
	err := r.DBWithContext(ctx).
		Select(dataResourceOperationColumns).
		Where("resource_id = ? AND operation = ?", resourceId, operation).
		First(&value).Error
	return value, err
}

func (r *DataResourceOperationRepositoryImpl) FindByStableKeyForConfigDB(
	db *gorm.DB,
	resourceId int,
	operation string,
) (model.DataResourceOperation, error) {
	var value model.DataResourceOperation
	err := db.Select(dataResourceOperationMutationColumns).
		Where("resource_id = ? AND operation = ?", resourceId, operation).
		First(&value).Error
	return value, err
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

func (r *DataOwnershipFieldRepositoryImpl) GetDataOwnershipFieldList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.DataOwnershipField], error) {
	return getDataPermissionConfigList(
		r.BasicRepositoryImpl,
		ctx,
		basic,
		table,
		"sys_data_ownership_field",
		dataOwnershipFieldQueryFields,
		dataOwnershipFieldColumns,
	)
}

func (r *DataOwnershipFieldRepositoryImpl) FindByStableKey(
	ctx context.Context,
	resourceId int,
	ownershipCode string,
) (model.DataOwnershipField, error) {
	var value model.DataOwnershipField
	err := r.DBWithContext(ctx).
		Select(dataOwnershipFieldColumns).
		Where("resource_id = ? AND ownership_code = ?", resourceId, ownershipCode).
		First(&value).Error
	return value, err
}

func (r *DataOwnershipFieldRepositoryImpl) FindByStableKeyForConfigDB(
	db *gorm.DB,
	resourceId int,
	ownershipCode string,
) (model.DataOwnershipField, error) {
	var value model.DataOwnershipField
	err := db.Select(dataOwnershipFieldMutationColumns).
		Where("resource_id = ? AND ownership_code = ?", resourceId, ownershipCode).
		First(&value).Error
	return value, err
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

func (r *DataOwnershipFieldRepositoryImpl) ListByResource(
	ctx context.Context,
	resourceId int,
) ([]model.DataOwnershipField, error) {
	values := make([]model.DataOwnershipField, 0)
	err := r.DBWithContext(ctx).
		Select(dataOwnershipFieldColumns).
		Where("resource_id = ?", resourceId).
		Order("ownership_code ASC, id ASC").
		Find(&values).Error
	return values, err
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

func (r *DataOwnershipFieldRepositoryImpl) ListActiveByOwnershipCodesForConfigDB(db *gorm.DB, codes []string) ([]model.DataOwnershipField, error) {
	if len(codes) == 0 {
		return []model.DataOwnershipField{}, nil
	}
	var ownerships []model.DataOwnershipField
	err := db.Where("ownership_code IN ? AND state = ?", codes, true).Find(&ownerships).Error
	return ownerships, err
}

func (r *DataPolicyRepositoryImpl) GetDataPolicyList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.DataPolicy], error) {
	return getDataPermissionConfigList(
		r.BasicRepositoryImpl,
		ctx,
		basic,
		table,
		"sys_data_policy",
		dataPolicyQueryFields,
		dataPolicyColumns,
	)
}

func (r *DataPolicyRuleRepositoryImpl) GetDataPolicyRuleList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.DataPolicyRule], error) {
	return getDataPermissionConfigList(
		r.BasicRepositoryImpl,
		ctx,
		basic,
		table,
		"sys_data_policy_rule",
		dataPolicyRuleQueryFields,
		dataPolicyRuleColumns,
	)
}

func (r *DataPolicyRuleRepositoryImpl) FindByStableKey(
	ctx context.Context,
	policyId int,
	sequence int,
) (model.DataPolicyRule, error) {
	var value model.DataPolicyRule
	err := r.DBWithContext(ctx).
		Select(dataPolicyRuleColumns).
		Where("policy_id = ? AND sequence = ?", policyId, sequence).
		First(&value).Error
	return value, err
}

func (r *DataPolicyRuleRepositoryImpl) FindByStableKeyForConfigDB(
	db *gorm.DB,
	policyId int,
	sequence int,
) (model.DataPolicyRule, error) {
	var value model.DataPolicyRule
	err := db.Select(dataPolicyRuleColumns).
		Where("policy_id = ? AND sequence = ?", policyId, sequence).
		First(&value).Error
	return value, err
}

func (r *DataPolicyRuleRepositoryImpl) ListByPolicy(
	ctx context.Context,
	policyId int,
) ([]model.DataPolicyRule, error) {
	return listDataPolicyRulesByPolicy(
		r.DBWithContext(ctx),
		policyId,
	)
}

func (r *DataPolicyRuleRepositoryImpl) ListByPolicyForConfigDB(
	db *gorm.DB,
	policyId int,
) ([]model.DataPolicyRule, error) {
	return listDataPolicyRulesByPolicy(db, policyId)
}

func listDataPolicyRulesByPolicy(
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

func (r *DataGrantRepositoryImpl) GetDataGrantList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.DataGrant], error) {
	return getDataPermissionConfigList(
		r.BasicRepositoryImpl,
		ctx,
		basic,
		table,
		"sys_data_grant",
		dataGrantQueryFields,
		dataGrantColumns,
	)
}

func (r *DataGrantRepositoryImpl) FindByStableKey(
	ctx context.Context,
	subjectType string,
	subjectId int,
	resourceId int,
	operation string,
	policyId int,
) (model.DataGrant, error) {
	var value model.DataGrant
	err := r.DBWithContext(ctx).
		Select(dataGrantColumns).
		Where(
			"subject_type = ? AND subject_id = ? AND resource_id = ? AND operation = ? AND policy_id = ?",
			subjectType, subjectId, resourceId, operation, policyId,
		).
		First(&value).Error
	return value, err
}

func (r *DataGrantRepositoryImpl) FindByStableKeyForConfigDB(
	db *gorm.DB,
	subjectType string,
	subjectId int,
	resourceId int,
	operation string,
	policyId int,
) (model.DataGrant, error) {
	var value model.DataGrant
	err := db.Select(dataGrantColumns).Where(
		"subject_type = ? AND subject_id = ? AND resource_id = ? AND operation = ? AND policy_id = ?",
		subjectType,
		subjectId,
		resourceId,
		operation,
		policyId,
	).First(&value).Error
	return value, err
}

func (r *DataGrantRepositoryImpl) ListEffectiveBySubjects(
	ctx context.Context,
	userId int,
	roleIds []int,
	resourceId int,
	operation string,
	asOf time.Time,
) ([]model.DataGrant, error) {
	values := make([]model.DataGrant, 0)
	if userId <= 0 || resourceId <= 0 || operation == "" || asOf.IsZero() {
		return values, nil
	}
	query := r.DBWithContext(ctx).
		Select(dataGrantColumns).
		Where("resource_id = ? AND operation = ? AND state = ?", resourceId, operation, true)
	if len(roleIds) == 0 {
		query = query.Where(
			"subject_type = ? AND subject_id = ?",
			model.DataGrantSubjectTypeUser,
			userId,
		)
	} else {
		query = query.Where(
			"((subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id IN ?))",
			model.DataGrantSubjectTypeUser,
			userId,
			model.DataGrantSubjectTypeRole,
			roleIds,
		)
	}
	err := query.
		Where("valid_from IS NULL OR valid_from <= ?", asOf).
		Where("valid_to IS NULL OR valid_to >= ?", asOf).
		Order("subject_type ASC, subject_id ASC, policy_id ASC, id ASC").
		Find(&values).Error
	return values, err
}

func (r *DataGrantRepositoryImpl) ListByResourceForConfigDB(
	db *gorm.DB,
	resourceId int,
) ([]model.DataGrant, error) {
	values := make([]model.DataGrant, 0)
	err := db.Select(dataGrantColumns).
		Where("resource_id = ?", resourceId).
		Order("id ASC").
		Find(&values).Error
	return values, err
}

func (r *DataGrantRepositoryImpl) ListByPolicyForConfigDB(
	db *gorm.DB,
	policyId int,
) ([]model.DataGrant, error) {
	values := make([]model.DataGrant, 0)
	err := db.Select(dataGrantColumns).
		Where("policy_id = ?", policyId).
		Order("id ASC").
		Find(&values).Error
	return values, err
}

func (r *DataGrantRepositoryImpl) RoleExistsForConfig(db *gorm.DB, roleId int) (bool, error) {
	var count int64
	err := db.Model(&model.SysRole{}).
		Where("id = ? AND state = ?", roleId, true).
		Count(&count).Error
	return count > 0, err
}

func (r *DataGrantRepositoryImpl) UserExistsForConfig(db *gorm.DB, userId int) (bool, error) {
	var count int64
	err := db.Model(&model.SysUser{}).
		Where("id = ? AND state = ?", userId, true).
		Count(&count).Error
	return count > 0, err
}

func (r *DataGrantRepositoryImpl) FindActiveSubjectIDsForConfigDB(db *gorm.DB, subjectType string, ids []int) ([]int, error) {
	if len(ids) == 0 {
		return []int{}, nil
	}
	var active []int
	switch subjectType {
	case model.DataGrantSubjectTypeRole:
		err := db.Model(&model.SysRole{}).Where("id IN ? AND state = ?", ids, true).Pluck("id", &active).Error
		return active, err
	case model.DataGrantSubjectTypeUser:
		err := db.Model(&model.SysUser{}).Where("id IN ? AND state = ?", ids, true).Pluck("id", &active).Error
		return active, err
	default:
		return []int{}, nil
	}
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

func getDataPermissionConfigList[T any](
	repo repository.BasicRepository[T],
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
	tableCode string,
	allowedFields map[string]struct{},
	columns []string,
) (response.ListResult[T], error) {
	readRepo := repo
	if ctx != nil {
		readRepo = readRepo.WithContext(ctx)
	}
	rows := make([]T, 0)
	total, err := readRepo.
		WithSelect(columns...).
		PaginateAndCountAsync(
			basic,
			&rows,
			dataPermissionConfigQueryTable(table, tableCode, allowedFields),
		)
	return response.ListResult[T]{Data: rows, Total: int(total)}, err
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
