package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"fmt"
	"sort"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrgLegalEntityRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgLegalEntity]
}

type OrgUnitRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgUnit]
}

type OrgStructureRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgStructure]
}

type OrgStructureNodeRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgStructureNode]
}

type OrgPositionRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgPosition]
}

type OrgEmployeeRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgEmployee]
}

type OrgAssignmentRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgAssignment]
}

type OrgSyncBatchRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgSyncBatch]
}

type OrgSyncRecordRepositoryImpl struct {
	*BasicRepositoryImpl[model.OrgSyncRecord]
}

var (
	_ repository.OrgLegalEntityRepository   = (*OrgLegalEntityRepositoryImpl)(nil)
	_ repository.OrgUnitRepository          = (*OrgUnitRepositoryImpl)(nil)
	_ repository.OrgStructureRepository     = (*OrgStructureRepositoryImpl)(nil)
	_ repository.OrgStructureNodeRepository = (*OrgStructureNodeRepositoryImpl)(nil)
	_ repository.OrgPositionRepository      = (*OrgPositionRepositoryImpl)(nil)
	_ repository.OrgEmployeeRepository      = (*OrgEmployeeRepositoryImpl)(nil)
	_ repository.OrgAssignmentRepository    = (*OrgAssignmentRepositoryImpl)(nil)
	_ repository.OrgSyncBatchRepository     = (*OrgSyncBatchRepositoryImpl)(nil)
	_ repository.OrgSyncRecordRepository    = (*OrgSyncRecordRepositoryImpl)(nil)
)

func NewOrgLegalEntityRepositoryImpl(primaryDB *database.PrimaryDB) *OrgLegalEntityRepositoryImpl {
	return &OrgLegalEntityRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgLegalEntity{}),
	}
}

func NewOrgUnitRepositoryImpl(primaryDB *database.PrimaryDB) *OrgUnitRepositoryImpl {
	return &OrgUnitRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgUnit{}),
	}
}

func NewOrgStructureRepositoryImpl(primaryDB *database.PrimaryDB) *OrgStructureRepositoryImpl {
	return &OrgStructureRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgStructure{}),
	}
}

func NewOrgStructureNodeRepositoryImpl(primaryDB *database.PrimaryDB) *OrgStructureNodeRepositoryImpl {
	return &OrgStructureNodeRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgStructureNode{}),
	}
}

func NewOrgPositionRepositoryImpl(primaryDB *database.PrimaryDB) *OrgPositionRepositoryImpl {
	return &OrgPositionRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgPosition{}),
	}
}

func NewOrgEmployeeRepositoryImpl(primaryDB *database.PrimaryDB) *OrgEmployeeRepositoryImpl {
	return &OrgEmployeeRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgEmployee{}),
	}
}

func NewOrgAssignmentRepositoryImpl(primaryDB *database.PrimaryDB) *OrgAssignmentRepositoryImpl {
	return &OrgAssignmentRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgAssignment{}),
	}
}

func NewOrgSyncBatchRepositoryImpl(primaryDB *database.PrimaryDB) *OrgSyncBatchRepositoryImpl {
	return &OrgSyncBatchRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgSyncBatch{}),
	}
}

func NewOrgSyncRecordRepositoryImpl(primaryDB *database.PrimaryDB) *OrgSyncRecordRepositoryImpl {
	return &OrgSyncRecordRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.OrgSyncRecord{}),
	}
}

func (r *OrgLegalEntityRepositoryImpl) Query(
	ctx *gin.Context,
	req *request.OrgLegalEntityQueryReq,
	table model.SysTable,
	scope repository.OrgLegalEntityReadScope,
) (response.ListResult[model.OrgLegalEntity], error) {
	if req == nil {
		req = &request.OrgLegalEntityQueryReq{}
	}
	query := legalEntityScopedBasic(req.Basic, scope)
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		query,
		map[string]any{
			"source_system_code": optionalString(req.SourceSystemCode),
			"entity_type":        optionalString(req.EntityType),
			"parent_id":          optionalInt(req.ParentId),
			"status":             optionalString(req.Status),
		},
		legalEntityScopedTable(organizationQueryTable(table, "org_legal_entity")),
		orgLegalEntityListColumns(),
	)
}

func (r *OrgLegalEntityRepositoryImpl) FindByIdForRead(ctx *gin.Context, id int) (model.OrgLegalEntity, error) {
	var entity model.OrgLegalEntity
	err := organizationDB(r.db, ctx).
		Select(orgLegalEntityDetailColumns()).
		First(&entity, id).Error
	return entity, err
}

func (r *OrgLegalEntityRepositoryImpl) ListForTree(
	ctx *gin.Context,
	scope repository.OrgLegalEntityReadScope,
) ([]model.OrgLegalEntity, error) {
	var entities []model.OrgLegalEntity
	query := applyLegalEntityReadScope(
		organizationDB(r.db, ctx).Model(&model.OrgLegalEntity{}),
		scope,
	)
	err := query.Select(orgLegalEntityTreeColumns()).Find(&entities).Error
	return entities, err
}

func (r *OrgLegalEntityRepositoryImpl) FindByIdsForDisplay(ctx *gin.Context, ids []int) ([]model.OrgLegalEntity, error) {
	if len(ids) == 0 {
		return []model.OrgLegalEntity{}, nil
	}
	var entities []model.OrgLegalEntity
	err := organizationDB(r.db, ctx).
		Select(orgLegalEntityTreeColumns()).
		Where("id IN ?", ids).
		Find(&entities).Error
	return entities, err
}

func (r *OrgLegalEntityRepositoryImpl) FindBySourceIdentity(ctx *gin.Context, sourceSystemCode, sourceId string) (model.OrgLegalEntity, error) {
	var entity model.OrgLegalEntity
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND source_id = ?", sourceSystemCode, sourceId).
		First(&entity).Error
	return entity, err
}

func (r *OrgLegalEntityRepositoryImpl) FindByCode(ctx *gin.Context, sourceSystemCode, code string) (model.OrgLegalEntity, error) {
	var entity model.OrgLegalEntity
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND code = ?", sourceSystemCode, code).
		First(&entity).Error
	return entity, err
}

func (r *OrgLegalEntityRepositoryImpl) UpdateSourceFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgLegalEntity{}, id, values, orgLegalEntitySourceFields)
}

func (r *OrgLegalEntityRepositoryImpl) UpdatePlatformFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgLegalEntity{}, id, values, orgLegalEntityPlatformFields)
}

func (r *OrgUnitRepositoryImpl) Query(ctx *gin.Context, req *request.OrgUnitQueryReq, table model.SysTable) (response.ListResult[model.OrgUnit], error) {
	if req == nil {
		req = &request.OrgUnitQueryReq{}
	}
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		req.Basic,
		map[string]any{
			"source_system_code":      optionalString(req.SourceSystemCode),
			"unit_type":               optionalString(req.UnitType),
			"primary_legal_entity_id": optionalInt(req.PrimaryLegalEntityId),
			"status":                  optionalString(req.Status),
		},
		organizationQueryTable(table, "org_unit"),
		orgUnitListColumns(),
	)
}

func (r *OrgUnitRepositoryImpl) FindBySourceIdentity(ctx *gin.Context, sourceSystemCode, sourceId string) (model.OrgUnit, error) {
	var unit model.OrgUnit
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND source_id = ?", sourceSystemCode, sourceId).
		First(&unit).Error
	return unit, err
}

func (r *OrgUnitRepositoryImpl) FindByCode(ctx *gin.Context, sourceSystemCode, code string) (model.OrgUnit, error) {
	var unit model.OrgUnit
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND code = ?", sourceSystemCode, code).
		First(&unit).Error
	return unit, err
}

func (r *OrgUnitRepositoryImpl) UpdateSourceFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgUnit{}, id, values, orgUnitSourceFields)
}

func (r *OrgUnitRepositoryImpl) UpdatePlatformFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgUnit{}, id, values, orgUnitPlatformFields)
}

func (r *OrgStructureRepositoryImpl) Query(ctx *gin.Context, req *request.OrgStructureQueryReq, table model.SysTable) (response.ListResult[model.OrgStructure], error) {
	if req == nil {
		req = &request.OrgStructureQueryReq{}
	}
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		req.Basic,
		map[string]any{
			"source_system_code": optionalString(req.SourceSystemCode),
			"structure_type":     optionalString(req.StructureType),
			"status":             optionalString(req.Status),
			"is_default":         optionalBool(req.IsDefault),
		},
		organizationQueryTable(table, "org_structure"),
		orgStructureListColumns(),
	)
}

func (r *OrgStructureRepositoryImpl) FindBySourceIdentity(ctx *gin.Context, sourceSystemCode, sourceId string) (model.OrgStructure, error) {
	var structure model.OrgStructure
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND source_id = ?", sourceSystemCode, sourceId).
		First(&structure).Error
	return structure, err
}

func (r *OrgStructureRepositoryImpl) FindByCode(ctx *gin.Context, code string) (model.OrgStructure, error) {
	var structure model.OrgStructure
	err := organizationDB(r.db, ctx).Where("code = ?", code).First(&structure).Error
	return structure, err
}

func (r *OrgStructureRepositoryImpl) UpdateSourceFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgStructure{}, id, values, orgStructureSourceFields)
}

func (r *OrgStructureNodeRepositoryImpl) Query(ctx *gin.Context, req *request.OrgStructureNodeQueryReq, table model.SysTable) (response.ListResult[model.OrgStructureNode], error) {
	if req == nil {
		req = &request.OrgStructureNodeQueryReq{}
	}
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		req.Basic,
		map[string]any{
			"structure_id":   optionalInt(req.StructureId),
			"org_unit_id":    optionalInt(req.OrgUnitId),
			"parent_node_id": optionalInt(req.ParentNodeId),
			"status":         optionalString(req.Status),
		},
		organizationQueryTable(table, "org_structure_node"),
		orgStructureNodeListColumns(),
	)
}

func (r *OrgStructureNodeRepositoryImpl) FindBySourceIdentity(ctx *gin.Context, sourceSystemCode, sourceId string) (model.OrgStructureNode, error) {
	var node model.OrgStructureNode
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND source_id = ?", sourceSystemCode, sourceId).
		First(&node).Error
	return node, err
}

func (r *OrgStructureNodeRepositoryImpl) UpdateSourceFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgStructureNode{}, id, values, orgStructureNodeSourceFields)
}

func (r *OrgPositionRepositoryImpl) Query(ctx *gin.Context, req *request.OrgPositionQueryReq, table model.SysTable) (response.ListResult[model.OrgPosition], error) {
	if req == nil {
		req = &request.OrgPositionQueryReq{}
	}
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		req.Basic,
		map[string]any{
			"source_system_code":  optionalString(req.SourceSystemCode),
			"org_unit_id":         optionalInt(req.OrgUnitId),
			"position_type":       optionalString(req.PositionType),
			"is_manager_position": optionalBool(req.IsManagerPosition),
			"status":              optionalString(req.Status),
		},
		organizationQueryTable(table, "org_position"),
		orgPositionListColumns(),
	)
}

func (r *OrgPositionRepositoryImpl) FindBySourceIdentity(ctx *gin.Context, sourceSystemCode, sourceId string) (model.OrgPosition, error) {
	var position model.OrgPosition
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND source_id = ?", sourceSystemCode, sourceId).
		First(&position).Error
	return position, err
}

func (r *OrgPositionRepositoryImpl) FindByCode(ctx *gin.Context, sourceSystemCode, code string) (model.OrgPosition, error) {
	var position model.OrgPosition
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND code = ?", sourceSystemCode, code).
		First(&position).Error
	return position, err
}

func (r *OrgPositionRepositoryImpl) UpdateSourceFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgPosition{}, id, values, orgPositionSourceFields)
}

func (r *OrgPositionRepositoryImpl) UpdatePlatformFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgPosition{}, id, values, orgPositionPlatformFields)
}

func (r *OrgEmployeeRepositoryImpl) Query(ctx *gin.Context, req *request.OrgEmployeeQueryReq, table model.SysTable) (response.ListResult[model.OrgEmployee], error) {
	if req == nil {
		req = &request.OrgEmployeeQueryReq{}
	}
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		req.Basic,
		map[string]any{
			"source_system_code":      optionalString(req.SourceSystemCode),
			"employment_status":       optionalString(req.EmploymentStatus),
			"primary_legal_entity_id": optionalInt(req.PrimaryLegalEntityId),
			"user_id":                 optionalInt(req.BoundUserId),
		},
		organizationQueryTable(table, "org_employee"),
		orgEmployeeListColumns(),
	)
}

func (r *OrgEmployeeRepositoryImpl) FindBySourceIdentity(ctx *gin.Context, sourceSystemCode, sourceId string) (model.OrgEmployee, error) {
	var employee model.OrgEmployee
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND source_id = ?", sourceSystemCode, sourceId).
		First(&employee).Error
	return employee, err
}

func (r *OrgEmployeeRepositoryImpl) FindByEmployeeNo(ctx *gin.Context, sourceSystemCode, employeeNo string) (model.OrgEmployee, error) {
	var employee model.OrgEmployee
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND employee_no = ?", sourceSystemCode, employeeNo).
		First(&employee).Error
	return employee, err
}

func (r *OrgEmployeeRepositoryImpl) FindByBoundUserId(ctx *gin.Context, userId int) (model.OrgEmployee, error) {
	var employee model.OrgEmployee
	err := organizationDB(r.db, ctx).Where("user_id = ?", userId).First(&employee).Error
	return employee, err
}

func (r *OrgEmployeeRepositoryImpl) UpdateSourceFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgEmployee{}, id, values, orgEmployeeSourceFields)
}

func (r *OrgEmployeeRepositoryImpl) UpdatePlatformFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgEmployee{}, id, values, orgEmployeePlatformFields)
}

func (r *OrgAssignmentRepositoryImpl) Query(ctx *gin.Context, req *request.OrgAssignmentQueryReq, table model.SysTable) (response.ListResult[model.OrgAssignment], error) {
	if req == nil {
		req = &request.OrgAssignmentQueryReq{}
	}
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		req.Basic,
		map[string]any{
			"source_system_code": optionalString(req.SourceSystemCode),
			"employee_id":        optionalInt(req.EmployeeId),
			"legal_entity_id":    optionalInt(req.LegalEntityId),
			"org_unit_id":        optionalInt(req.OrgUnitId),
			"position_id":        optionalInt(req.PositionId),
			"assignment_type":    optionalString(req.AssignmentType),
			"is_primary":         optionalBool(req.IsPrimary),
			"is_manager":         optionalBool(req.IsManager),
			"status":             optionalString(req.Status),
		},
		organizationQueryTable(table, "org_assignment"),
		orgAssignmentListColumns(),
	)
}

func (r *OrgAssignmentRepositoryImpl) FindBySourceIdentity(ctx *gin.Context, sourceSystemCode, sourceId string) (model.OrgAssignment, error) {
	var assignment model.OrgAssignment
	err := organizationDB(r.db, ctx).
		Where("source_system_code = ? AND source_id = ?", sourceSystemCode, sourceId).
		First(&assignment).Error
	return assignment, err
}

func (r *OrgAssignmentRepositoryImpl) UpdateSourceFields(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgAssignment{}, id, values, orgAssignmentSourceFields)
}

func (r *OrgSyncBatchRepositoryImpl) Query(ctx *gin.Context, req *request.OrgSyncBatchQueryReq, table model.SysTable) (response.ListResult[model.OrgSyncBatch], error) {
	if req == nil {
		req = &request.OrgSyncBatchQueryReq{}
	}
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		req.Basic,
		map[string]any{
			"execution_id": optionalInt(req.ExecutionId),
			"sync_type":    optionalString(req.SyncType),
			"object_scope": optionalString(req.ObjectScope),
			"status":       optionalString(req.Status),
		},
		organizationQueryTable(table, "org_sync_batch"),
		orgSyncBatchListColumns(),
	)
}

func (r *OrgSyncBatchRepositoryImpl) FindByBatchNo(ctx *gin.Context, batchNo string) (model.OrgSyncBatch, error) {
	var batch model.OrgSyncBatch
	err := organizationDB(r.db, ctx).Where("batch_no = ?", batchNo).First(&batch).Error
	return batch, err
}

func (r *OrgSyncRecordRepositoryImpl) Query(ctx *gin.Context, req *request.OrgSyncRecordQueryReq, table model.SysTable) (response.ListResult[model.OrgSyncRecord], error) {
	if req == nil {
		req = &request.OrgSyncRecordQueryReq{}
	}
	return queryOrganization(
		r.BasicRepositoryImpl,
		ctx,
		req.Basic,
		map[string]any{
			"batch_id":              optionalInt(req.BatchId),
			"execution_id":          optionalInt(req.ExecutionId),
			"object_type":           optionalString(req.ObjectType),
			"action":                optionalString(req.Action),
			"status":                optionalString(req.Status),
			"dependency_type":       optionalString(req.DependencyType),
			"local_handling_status": optionalString(req.LocalHandlingStatus),
		},
		organizationQueryTable(table, "org_sync_record"),
		orgSyncRecordListColumns(),
	)
}

func queryOrganization[T any](
	repo repository.BasicRepository[T],
	ctx *gin.Context,
	basic request.Basic,
	typedFilters map[string]any,
	table model.SysTable,
	columns []string,
) (response.ListResult[T], error) {
	query := cloneOrganizationBasic(basic)
	if query.Filters == nil {
		query.Filters = make(map[string]any)
	}
	for field, value := range typedFilters {
		if value != nil {
			query.Filters[field] = value
		}
	}

	readRepo := repo
	if ctx != nil {
		readRepo = readRepo.WithContext(ctx)
	}
	var rows []T
	total, err := readRepo.
		WithSelect(columns...).
		PaginateAndCountAsync(&query, &rows, table)
	return response.ListResult[T]{Data: rows, Total: int(total)}, err
}

func legalEntityScopedBasic(
	source request.Basic,
	scope repository.OrgLegalEntityReadScope,
) request.Basic {
	query := cloneOrganizationBasic(source)
	// Legal-entity history is controlled only by the explicit organization
	// visibility flags. The generic soft-delete switch is not part of this API.
	query.IncludeDeleted = false
	if legalEntityInternalFieldRequested(query, "source_deleted") {
		query.Expressions = append(query.Expressions, request.ExpressionGroup{
			Logic: enum.And,
			Rules: []request.QueryRule{{
				Field:          "id",
				ExpressionType: enum.Eq,
				Value:          -1,
			}},
		})
	}

	if !scope.IncludeDisabled {
		query.Expressions = append(query.Expressions, request.ExpressionGroup{
			Logic: enum.And,
			Rules: []request.QueryRule{{
				Field:          "status",
				ExpressionType: enum.Eq,
				Value:          "enabled",
			}},
		})
	}
	if scope.IncludeHistory {
		return query
	}

	asOf := scope.AsOf
	if asOf.IsZero() {
		asOf = model.Now()
	}
	query.Expressions = append(
		query.Expressions,
		request.ExpressionGroup{
			Logic: enum.And,
			Rules: []request.QueryRule{{
				Field:          "source_deleted",
				ExpressionType: enum.Eq,
				Value:          false,
			}},
		},
		request.ExpressionGroup{
			Logic: enum.Or,
			Rules: []request.QueryRule{
				{Field: "valid_from", ExpressionType: enum.IsNull},
				{Field: "valid_from", ExpressionType: enum.Lte, Value: asOf},
			},
		},
		request.ExpressionGroup{
			Logic: enum.Or,
			Rules: []request.QueryRule{
				{Field: "valid_to", ExpressionType: enum.IsNull},
				{Field: "valid_to", ExpressionType: enum.Gte, Value: asOf},
			},
		},
	)
	return query
}

func legalEntityScopedTable(table model.SysTable) model.SysTable {
	table = ensureOrganizationQueryField(table, "id", enum.BigIntFieldType)
	table = ensureOrganizationQueryField(table, "status", enum.VarcharFieldType)
	table = ensureOrganizationQueryField(table, "source_deleted", enum.BooleanFieldType)
	table = ensureOrganizationQueryField(table, "valid_from", enum.DatetimeFieldType)
	table = ensureOrganizationQueryField(table, "valid_to", enum.DatetimeFieldType)
	return table
}

func ensureOrganizationQueryField(
	table model.SysTable,
	fieldCode string,
	fieldType enum.SysTableFieldType,
) model.SysTable {
	for _, field := range table.TableFields {
		if field.FieldCode == fieldCode {
			return table
		}
	}
	table.TableFields = append(table.TableFields, model.SysTableField{
		FieldCode: fieldCode,
		FieldType: fieldType,
	})
	return table
}

func legalEntityInternalFieldRequested(query request.Basic, field string) bool {
	if query.Order.Field == field {
		return true
	}
	if _, exists := query.Filters[field]; exists {
		return true
	}
	for _, group := range query.Expressions {
		if expressionGroupReferencesField(group, field) {
			return true
		}
	}
	return false
}

func expressionGroupReferencesField(group request.ExpressionGroup, field string) bool {
	for _, rule := range group.Rules {
		if rule.Field == field {
			return true
		}
	}
	for _, nested := range group.Nested {
		if expressionGroupReferencesField(nested, field) {
			return true
		}
	}
	return false
}

func applyLegalEntityReadScope(
	query *gorm.DB,
	scope repository.OrgLegalEntityReadScope,
) *gorm.DB {
	if !scope.IncludeDisabled {
		query = query.Where("status = ?", "enabled")
	}
	if scope.IncludeHistory {
		return query
	}
	asOf := scope.AsOf
	if asOf.IsZero() {
		asOf = model.Now()
	}
	return query.
		Where("source_deleted = ?", false).
		Where("(valid_from IS NULL OR valid_from <= ?)", asOf).
		Where("(valid_to IS NULL OR valid_to >= ?)", asOf)
}

func cloneOrganizationBasic(source request.Basic) request.Basic {
	cloned := source
	if source.Filters != nil {
		cloned.Filters = make(map[string]any, len(source.Filters))
		for field, value := range source.Filters {
			cloned.Filters[field] = value
		}
	}
	if source.QuickQuery != nil {
		quick := *source.QuickQuery
		cloned.QuickQuery = &quick
	}
	if source.Expressions != nil {
		cloned.Expressions = make([]request.ExpressionGroup, len(source.Expressions))
		for index, group := range source.Expressions {
			cloned.Expressions[index] = cloneExpressionGroup(group)
		}
	}
	return cloned
}

func cloneExpressionGroup(source request.ExpressionGroup) request.ExpressionGroup {
	cloned := source
	cloned.Rules = append([]request.QueryRule(nil), source.Rules...)
	if source.Nested != nil {
		cloned.Nested = make([]request.ExpressionGroup, len(source.Nested))
		for index, nested := range source.Nested {
			cloned.Nested[index] = cloneExpressionGroup(nested)
		}
	}
	return cloned
}

func optionalString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func optionalInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func organizationDB(db *gorm.DB, ctx *gin.Context) *gorm.DB {
	if ctx == nil {
		return db
	}
	return db.WithContext(ctx)
}

func updateOrganizationFields(
	tx *gorm.DB,
	entity any,
	id int,
	values map[string]any,
	allowed map[string]struct{},
) error {
	if tx == nil {
		return repository.ErrOrganizationTxRequired
	}
	if len(values) == 0 {
		return nil
	}

	columns := make([]string, 0, len(values))
	safeValues := make(map[string]any, len(values))
	for field, value := range values {
		if _, ok := allowed[field]; !ok {
			return fmt.Errorf("%w: %s", repository.ErrOrganizationFieldBoundary, field)
		}
		columns = append(columns, field)
		safeValues[field] = value
	}
	sort.Strings(columns)
	return tx.Model(entity).
		Where("id = ?", id).
		Select(columns).
		Updates(safeValues).Error
}

func organizationFieldSet(fields ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		result[field] = struct{}{}
	}
	return result
}

var (
	orgLegalEntitySourceFields = organizationFieldSet(
		"source_code", "code", "name", "short_name", "entity_type", "parent_id",
		"unified_social_credit_code", "accounting_code", "status", "valid_from",
		"valid_to", "source_version", "source_updated_at", "last_sync_at",
		"source_status", "source_deleted", "sync_status", "last_error",
	)
	orgLegalEntityPlatformFields = organizationFieldSet(
		"local_note", "local_tags", "display_order", "local_handling_status",
	)
	orgUnitSourceFields = organizationFieldSet(
		"source_code", "code", "name", "unit_type", "primary_legal_entity_id",
		"status", "valid_from", "valid_to", "source_version", "source_updated_at",
		"last_sync_at", "source_status", "source_deleted", "sync_status", "last_error",
	)
	orgUnitPlatformFields = organizationFieldSet(
		"local_note", "local_tags", "display_order", "local_handling_status",
	)
	orgStructureSourceFields = organizationFieldSet(
		"code", "name", "structure_type", "source_id", "status", "is_default",
		"valid_from", "valid_to", "source_version", "last_sync_at", "sync_status",
	)
	orgStructureNodeSourceFields = organizationFieldSet(
		"structure_id", "org_unit_id", "parent_node_id", "source_parent_id", "path",
		"level", "sort", "valid_from", "valid_to", "status", "source_deleted", "sync_status",
	)
	orgPositionSourceFields = organizationFieldSet(
		"source_code", "code", "name", "org_unit_id", "position_type", "job_level",
		"is_manager_position", "status", "valid_from", "valid_to", "source_version",
		"last_sync_at", "source_deleted", "sync_status",
	)
	orgPositionPlatformFields = organizationFieldSet("local_note")
	orgEmployeeSourceFields   = organizationFieldSet(
		"source_code", "employee_no", "name", "mobile", "email", "employment_status",
		"primary_legal_entity_id", "valid_from", "valid_to", "source_version",
		"source_updated_at", "last_sync_at", "source_deleted", "sync_status",
	)
	orgEmployeePlatformFields = organizationFieldSet("user_id", "local_note", "local_tags")
	orgAssignmentSourceFields = organizationFieldSet(
		"employee_id", "legal_entity_id", "org_unit_id", "position_id",
		"assignment_type", "is_primary", "is_manager", "valid_from", "valid_to",
		"status", "source_version", "source_deleted", "sync_status",
	)
)

func orgLegalEntityListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "short_name",
		"entity_type", "parent_id", "unified_social_credit_code", "accounting_code",
		"status", "valid_from", "valid_to",
	}
}

func orgLegalEntityDetailColumns() []string {
	return append(
		orgLegalEntityListColumns(),
		"local_note",
		"local_tags",
		"display_order",
		"local_handling_status",
	)
}

func orgLegalEntityTreeColumns() []string {
	return append(orgLegalEntityListColumns(), "display_order")
}

func orgUnitListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "unit_type",
		"primary_legal_entity_id", "status", "valid_from", "valid_to", "display_order",
	}
}

func orgStructureListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "structure_type",
		"status", "is_default", "valid_from", "valid_to",
	}
}

func orgStructureNodeListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "structure_id", "org_unit_id",
		"parent_node_id", "path", "level", "sort", "valid_from", "valid_to", "status",
	}
}

func orgPositionListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "org_unit_id",
		"position_type", "job_level", "is_manager_position", "status", "valid_from", "valid_to",
	}
}

func orgEmployeeListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "employee_no", "name",
		"employment_status", "primary_legal_entity_id", "user_id", "valid_from", "valid_to",
	}
}

func orgAssignmentListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "employee_id", "legal_entity_id",
		"org_unit_id", "position_id", "assignment_type", "is_primary", "is_manager",
		"valid_from", "valid_to", "status",
	}
}

func orgSyncBatchListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "batch_no", "execution_id",
		"sync_type", "object_scope", "started_at", "completed_at", "total_count",
		"success_count", "failed_count", "skipped_count", "status",
	}
}

func orgSyncRecordListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "batch_id", "execution_id",
		"object_type", "source_code", "local_id", "action", "status", "error_code",
		"dependency_type", "retry_count", "last_retry_at", "local_handling_status",
	}
}

// organizationQueryTable keeps sys_table/sys_table_field as the single query
// metadata source while removing fields that metadata has not opened for
// lists, quick search, advanced search, sorting, or stable identity.
func organizationQueryTable(table model.SysTable, tableCode string) model.SysTable {
	table.TableCode = tableCode
	fields := make([]model.SysTableField, 0, len(table.TableFields))
	for _, field := range table.TableFields {
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
