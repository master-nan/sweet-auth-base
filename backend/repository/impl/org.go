package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	queryutil "backend/repository/util"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *OrgUnitRepositoryImpl) QueryForRead(
	ctx *gin.Context,
	req *request.OrgUnitQueryReq,
	table model.SysTable,
	scope repository.OrgReadScope,
	structureId *int,
) (response.ListResult[model.OrgUnit], error) {
	if req == nil {
		req = &request.OrgUnitQueryReq{}
	}
	basic := cloneOrganizationBasic(req.Basic)
	basic.IncludeDeleted = false
	query := applyOrganizationReadScope(
		organizationDB(r.db, ctx).Model(&model.OrgUnit{}),
		"org_unit",
		scope,
		true,
	)
	query = applyOrganizationTypedFilters(query, map[string]any{
		"org_unit.source_system_code":      optionalString(req.SourceSystemCode),
		"org_unit.unit_type":               optionalString(req.UnitType),
		"org_unit.primary_legal_entity_id": optionalInt(req.PrimaryLegalEntityId),
		"org_unit.status":                  optionalString(req.Status),
	})
	if structureId != nil {
		nodeScope := applyOrganizationReadScope(
			organizationDB(r.db, ctx).Model(&model.OrgStructureNode{}).
				Select("1").
				Where("org_structure_node.structure_id = ?", *structureId).
				Where("org_structure_node.org_unit_id = org_unit.id"),
			"org_structure_node",
			scope,
			true,
		)
		query = query.Where("EXISTS (?)", nodeScope)
	}
	return paginateOrganizationReadQuery[model.OrgUnit](
		query,
		&basic,
		organizationQueryTable(table, "org_unit"),
		orgUnitListColumns(),
	)
}

func (r *OrgUnitRepositoryImpl) FindByIdForRead(ctx *gin.Context, id int) (model.OrgUnit, error) {
	var unit model.OrgUnit
	err := organizationDB(r.db, ctx).
		Select(orgUnitReadColumns()).
		First(&unit, id).Error
	return unit, err
}

func (r *OrgUnitRepositoryImpl) FindByIdsForDisplay(ctx *gin.Context, ids []int) ([]model.OrgUnit, error) {
	if len(ids) == 0 {
		return []model.OrgUnit{}, nil
	}
	var units []model.OrgUnit
	err := organizationDB(r.db, ctx).
		Select(orgUnitReadColumns()).
		Where("id IN ?", ids).
		Find(&units).Error
	return units, err
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

func (r *OrgStructureRepositoryImpl) QueryForRead(
	ctx *gin.Context,
	req *request.OrgStructureQueryReq,
	table model.SysTable,
	scope repository.OrgReadScope,
) (response.ListResult[model.OrgStructure], error) {
	if req == nil {
		req = &request.OrgStructureQueryReq{}
	}
	basic := cloneOrganizationBasic(req.Basic)
	basic.IncludeDeleted = false
	query := applyOrganizationReadScope(
		organizationDB(r.db, ctx).Model(&model.OrgStructure{}),
		"org_structure",
		scope,
		false,
	)
	query = applyOrganizationTypedFilters(query, map[string]any{
		"org_structure.source_system_code": optionalString(req.SourceSystemCode),
		"org_structure.structure_type":     optionalString(req.StructureType),
		"org_structure.status":             optionalString(req.Status),
		"org_structure.is_default":         optionalBool(req.IsDefault),
	})
	if req.LegalEntityId != nil {
		nodeAndUnitScope := applyOrganizationReadScope(
			organizationDB(r.db, ctx).
				Table("org_structure_node").
				Select("1").
				Joins("JOIN org_unit ON org_unit.id = org_structure_node.org_unit_id").
				Where("org_structure_node.structure_id = org_structure.id").
				Where("org_unit.primary_legal_entity_id = ?", *req.LegalEntityId).
				Where("org_structure_node.gmt_delete IS NULL"),
			"org_structure_node",
			scope,
			true,
		)
		nodeAndUnitScope = applyOrganizationReadScope(
			nodeAndUnitScope,
			"org_unit",
			scope,
			true,
		).Where("org_unit.gmt_delete IS NULL")
		query = query.Where("EXISTS (?)", nodeAndUnitScope)
	}
	return paginateOrganizationReadQuery[model.OrgStructure](
		query,
		&basic,
		organizationQueryTable(table, "org_structure"),
		orgStructureListColumns(),
	)
}

func (r *OrgStructureRepositoryImpl) FindByIdForRead(ctx *gin.Context, id int) (model.OrgStructure, error) {
	var structure model.OrgStructure
	err := organizationDB(r.db, ctx).
		Select(orgStructureReadColumns()).
		First(&structure, id).Error
	return structure, err
}

func (r *OrgStructureRepositoryImpl) FindByIdsForDisplay(ctx *gin.Context, ids []int) ([]model.OrgStructure, error) {
	if len(ids) == 0 {
		return []model.OrgStructure{}, nil
	}
	var structures []model.OrgStructure
	err := organizationDB(r.db, ctx).
		Select(orgStructureReadColumns()).
		Where("id IN ?", ids).
		Find(&structures).Error
	return structures, err
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

func (r *OrgStructureNodeRepositoryImpl) ListByStructureForRead(
	ctx *gin.Context,
	structureId int,
	scope repository.OrgReadScope,
	limit int,
) ([]model.OrgStructureNode, error) {
	if limit <= 0 {
		return []model.OrgStructureNode{}, nil
	}
	var nodes []model.OrgStructureNode
	query := applyOrganizationReadScope(
		organizationDB(r.db, ctx).Model(&model.OrgStructureNode{}),
		"org_structure_node",
		scope,
		true,
	)
	err := query.
		Select(orgStructureNodeReadColumns()).
		Where("org_structure_node.structure_id = ?", structureId).
		Order("org_structure_node.sort ASC").
		Order("org_structure_node.id ASC").
		Limit(limit).
		Find(&nodes).Error
	return nodes, err
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

func (r *OrgPositionRepositoryImpl) QueryForRead(
	ctx *gin.Context,
	req *request.OrgPositionQueryReq,
	table model.SysTable,
	scope repository.OrgReadScope,
) (response.ListResult[model.OrgPosition], error) {
	if req == nil {
		req = &request.OrgPositionQueryReq{}
	}
	basic := cloneOrganizationBasic(req.Basic)
	basic.IncludeDeleted = false
	query := applyOrganizationReadScope(
		organizationDB(r.db, ctx).Model(&model.OrgPosition{}),
		"org_position",
		scope,
		true,
	)
	query = applyOrganizationTypedFilters(query, map[string]any{
		"org_position.source_system_code":  optionalString(req.SourceSystemCode),
		"org_position.org_unit_id":         optionalInt(req.OrgUnitId),
		"org_position.position_type":       optionalString(req.PositionType),
		"org_position.is_manager_position": optionalBool(req.IsManagerPosition),
		"org_position.status":              optionalString(req.Status),
	})
	if req.LegalEntityId != nil {
		unitScope := applyOrganizationReadScope(
			organizationDB(r.db, ctx).
				Model(&model.OrgUnit{}).
				Select("1").
				Where("org_unit.id = org_position.org_unit_id").
				Where("org_unit.primary_legal_entity_id = ?", *req.LegalEntityId),
			"org_unit",
			scope,
			true,
		)
		query = query.Where("EXISTS (?)", unitScope)
	}
	return paginateOrganizationReadQuery[model.OrgPosition](
		query,
		&basic,
		organizationQueryTable(table, "org_position"),
		orgPositionReadColumns(),
	)
}

func (r *OrgPositionRepositoryImpl) FindByIdForRead(ctx *gin.Context, id int) (model.OrgPosition, error) {
	var position model.OrgPosition
	err := organizationDB(r.db, ctx).
		Select(orgPositionDetailColumns()).
		First(&position, id).Error
	return position, err
}

func (r *OrgPositionRepositoryImpl) FindByIdsForDisplay(ctx *gin.Context, ids []int) ([]model.OrgPosition, error) {
	if len(ids) == 0 {
		return []model.OrgPosition{}, nil
	}
	var positions []model.OrgPosition
	err := organizationDB(r.db, ctx).
		Select(orgPositionReadColumns()).
		Where("id IN ?", ids).
		Find(&positions).Error
	return positions, err
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

func (r *OrgEmployeeRepositoryImpl) QueryForRead(
	ctx *gin.Context,
	req *request.OrgEmployeeQueryReq,
	table model.SysTable,
	scope repository.OrgReadScope,
) (response.ListResult[model.OrgEmployee], error) {
	if req == nil {
		req = &request.OrgEmployeeQueryReq{}
	}
	basic := cloneOrganizationBasic(req.Basic)
	basic.IncludeDeleted = false
	query := applyEmployeeReadScope(
		organizationDB(r.db, ctx).Model(&model.OrgEmployee{}),
		"org_employee",
		scope,
	)
	query = applyOrganizationTypedFilters(query, map[string]any{
		"org_employee.source_system_code":      optionalString(req.SourceSystemCode),
		"org_employee.employment_status":       optionalString(req.EmploymentStatus),
		"org_employee.primary_legal_entity_id": optionalInt(req.PrimaryLegalEntityId),
		"org_employee.user_id":                 optionalInt(req.BoundUserId),
	})
	switch req.BoundStatus {
	case "bound":
		query = query.Where("org_employee.user_id IS NOT NULL")
	case "unbound":
		query = query.Where("org_employee.user_id IS NULL")
	}

	if req.LegalEntityId != nil || req.OrgUnitId != nil || req.PositionId != nil {
		assignmentScope := applyOrganizationReadScope(
			organizationDB(r.db, ctx).
				Model(&model.OrgAssignment{}).
				Select("1").
				Where("org_assignment.employee_id = org_employee.id"),
			"org_assignment",
			scope,
			true,
		)
		assignmentScope = applyOrganizationTypedFilters(assignmentScope, map[string]any{
			"org_assignment.legal_entity_id": optionalInt(req.LegalEntityId),
			"org_assignment.org_unit_id":     optionalInt(req.OrgUnitId),
			"org_assignment.position_id":     optionalInt(req.PositionId),
		})
		query = query.Where("EXISTS (?)", assignmentScope)
	}

	return paginateOrganizationReadQuery[model.OrgEmployee](
		query,
		&basic,
		organizationEmployeeQueryTable(table),
		orgEmployeeReadColumns(),
	)
}

func (r *OrgEmployeeRepositoryImpl) FindByIdForRead(ctx *gin.Context, id int) (model.OrgEmployee, error) {
	var employee model.OrgEmployee
	err := organizationDB(r.db, ctx).
		Select(orgEmployeeDetailColumns()).
		First(&employee, id).Error
	return employee, err
}

func (r *OrgEmployeeRepositoryImpl) FindByIdsForDisplay(ctx *gin.Context, ids []int) ([]model.OrgEmployee, error) {
	if len(ids) == 0 {
		return []model.OrgEmployee{}, nil
	}
	var employees []model.OrgEmployee
	err := organizationDB(r.db, ctx).
		Select(orgEmployeeReadColumns()).
		Where("id IN ?", ids).
		Find(&employees).Error
	return employees, err
}

func (r *OrgEmployeeRepositoryImpl) FindBoundUserSummaries(
	ctx *gin.Context,
	ids []int,
) ([]repository.OrgBoundUserSummary, error) {
	if len(ids) == 0 {
		return []repository.OrgBoundUserSummary{}, nil
	}
	var users []repository.OrgBoundUserSummary
	err := organizationDB(r.db, ctx).
		Model(&model.SysUser{}).
		Select("id AS user_id", "user_name").
		Where("id IN ?", ids).
		Find(&users).Error
	return users, err
}

func (r *OrgEmployeeRepositoryImpl) QueryUsersForBinding(
	ctx *gin.Context,
	keyword string,
	page int,
	num int,
) (response.ListResult[repository.OrgBindingUserOption], error) {
	var result response.ListResult[repository.OrgBindingUserOption]
	query := organizationDB(r.db, ctx).
		Model(&model.SysUser{}).
		Where("state = ?", true).
		Where(
			"NOT EXISTS (SELECT 1 FROM org_employee WHERE org_employee.user_id = sys_user.id AND org_employee.gmt_delete IS NULL)",
		)
	if normalized := strings.ToLower(strings.TrimSpace(keyword)); normalized != "" {
		query = query.Where("LOWER(user_name) LIKE ?", "%"+normalized+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return result, err
	}
	result.Total = int(total)
	result.Data = make([]repository.OrgBindingUserOption, 0, num)
	err := query.
		Select(
			"id AS user_id",
			"user_name",
			"false AS disabled",
		).
		Order("user_name ASC").
		Order("id ASC").
		Offset((page - 1) * num).
		Limit(num).
		Scan(&result.Data).Error
	return result, err
}

func (r *OrgEmployeeRepositoryImpl) FindByIdForBinding(tx *gorm.DB, id int) (model.OrgEmployee, error) {
	if tx == nil {
		return model.OrgEmployee{}, repository.ErrOrganizationTxRequired
	}
	var employee model.OrgEmployee
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id", "user_id").
		First(&employee, id).Error
	return employee, err
}

func (r *OrgEmployeeRepositoryImpl) FindUserForBinding(
	tx *gorm.DB,
	userId int,
) (repository.OrgBoundUserSummary, error) {
	if tx == nil {
		return repository.OrgBoundUserSummary{}, repository.ErrOrganizationTxRequired
	}
	var user repository.OrgBoundUserSummary
	err := tx.
		Model(&model.SysUser{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id AS user_id", "user_name").
		Where("id = ?", userId).
		First(&user).Error
	return user, err
}

func (r *OrgEmployeeRepositoryImpl) FindByBoundUserIdForBinding(
	tx *gorm.DB,
	userId int,
) (model.OrgEmployee, error) {
	if tx == nil {
		return model.OrgEmployee{}, repository.ErrOrganizationTxRequired
	}
	var employee model.OrgEmployee
	err := tx.
		Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id", "user_id").
		Where("user_id = ?", userId).
		First(&employee).Error
	return employee, err
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
			"employee_id":     optionalInt(req.EmployeeId),
			"legal_entity_id": optionalInt(req.LegalEntityId),
			"org_unit_id":     optionalInt(req.OrgUnitId),
			"position_id":     optionalInt(req.PositionId),
			"assignment_type": optionalString(req.AssignmentType),
			"is_primary":      optionalBool(req.IsPrimary),
			"is_manager":      optionalBool(req.IsManager),
			"status":          optionalString(req.Status),
		},
		organizationQueryTable(table, "org_assignment"),
		orgAssignmentListColumns(),
	)
}

func (r *OrgAssignmentRepositoryImpl) QueryForRead(
	ctx *gin.Context,
	req *request.OrgAssignmentQueryReq,
	table model.SysTable,
	scope repository.OrgAssignmentReadScope,
) (response.ListResult[model.OrgAssignment], error) {
	if req == nil {
		req = &request.OrgAssignmentQueryReq{}
	}
	basic := cloneOrganizationBasic(req.Basic)
	basic.IncludeDeleted = false
	query := organizationDB(r.db, ctx).Model(&model.OrgAssignment{})
	query = applyOrganizationTypedFilters(query, map[string]any{
		"org_assignment.employee_id":     optionalInt(req.EmployeeId),
		"org_assignment.legal_entity_id": optionalInt(req.LegalEntityId),
		"org_assignment.org_unit_id":     optionalInt(req.OrgUnitId),
		"org_assignment.position_id":     optionalInt(req.PositionId),
		"org_assignment.assignment_type": optionalString(req.AssignmentType),
		"org_assignment.is_primary":      optionalBool(req.IsPrimary),
		"org_assignment.is_manager":      optionalBool(req.IsManager),
		"org_assignment.status":          optionalString(req.Status),
	})
	query = applyAssignmentReadScope(query, scope)
	query, basic = applyAssignmentReadOrder(query, basic, scope.TimeScope)

	return paginateOrganizationReadQuery[model.OrgAssignment](
		query,
		&basic,
		organizationQueryTable(table, "org_assignment"),
		orgAssignmentReadColumns(),
	)
}

func (r *OrgAssignmentRepositoryImpl) ListEffectiveByEmployee(
	ctx *gin.Context,
	employeeId int,
	asOf time.Time,
	limit int,
) ([]model.OrgAssignment, error) {
	if asOf.IsZero() {
		asOf = model.Now()
	}
	if limit <= 0 {
		limit = 1
	}
	var assignments []model.OrgAssignment
	err := organizationDB(r.db, ctx).
		Model(&model.OrgAssignment{}).
		Select(orgAssignmentReadColumns()).
		Where("employee_id = ?", employeeId).
		Where("status = ?", "enabled").
		Where("source_deleted = ?", false).
		Where("(valid_from IS NULL OR valid_from <= ?)", asOf).
		Where("(valid_to IS NULL OR valid_to >= ?)", asOf).
		Order("legal_entity_id ASC").
		Order("org_unit_id ASC").
		Order("CASE WHEN position_id IS NULL THEN 1 ELSE 0 END ASC").
		Order("position_id ASC").
		Order("id ASC").
		Limit(limit).
		Find(&assignments).Error
	return assignments, err
}

func (r *OrgAssignmentRepositoryImpl) FindByIdForRead(ctx *gin.Context, id int) (model.OrgAssignment, error) {
	var assignment model.OrgAssignment
	err := organizationDB(r.db, ctx).
		Select(orgAssignmentReadColumns()).
		First(&assignment, id).Error
	return assignment, err
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

func (r *OrgSyncBatchRepositoryImpl) FindByIdForRead(ctx *gin.Context, id int) (model.OrgSyncBatch, error) {
	var batch model.OrgSyncBatch
	err := organizationDB(r.db, ctx).
		Select(orgSyncBatchListColumns()).
		Where("id = ?", id).
		First(&batch).Error
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
			"local_id":              optionalInt(req.LocalId),
			"action":                optionalString(req.Action),
			"status":                optionalString(req.Status),
			"dependency_type":       optionalString(req.DependencyType),
			"local_handling_status": optionalString(req.LocalHandlingStatus),
		},
		organizationQueryTable(table, "org_sync_record"),
		orgSyncRecordListColumns(),
	)
}

func (r *OrgSyncRecordRepositoryImpl) FindByIdForRead(ctx *gin.Context, id int) (model.OrgSyncRecord, error) {
	var record model.OrgSyncRecord
	err := organizationDB(r.db, ctx).
		Select(append(orgSyncRecordListColumns(), "dependency_key")).
		Where("id = ?", id).
		First(&record).Error
	return record, err
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

func paginateOrganizationReadQuery[T any](
	query *gorm.DB,
	basic *request.Basic,
	table model.SysTable,
	columns []string,
) (response.ListResult[T], error) {
	var result response.ListResult[T]
	query = queryutil.ExecuteQuery(query, basic, table)

	var total int64
	if err := query.Session(&gorm.Session{}).
		Limit(-1).
		Offset(-1).
		Count(&total).Error; err != nil {
		return result, err
	}
	rows := make([]T, 0)
	if err := query.Session(&gorm.Session{}).
		Select(columns).
		Find(&rows).Error; err != nil {
		return result, err
	}
	result.Data = rows
	result.Total = int(total)
	return result, nil
}

func applyOrganizationTypedFilters(query *gorm.DB, filters map[string]any) *gorm.DB {
	for field, value := range filters {
		if value != nil {
			query = query.Where(field+" = ?", value)
		}
	}
	return query
}

func applyOrganizationReadScope(
	query *gorm.DB,
	tableName string,
	scope repository.OrgReadScope,
	hasSourceDeleted bool,
) *gorm.DB {
	if !scope.IncludeDisabled {
		query = query.Where(tableName+".status = ?", "enabled")
	}
	if scope.IncludeHistory {
		return query
	}
	asOf := scope.AsOf
	if asOf.IsZero() {
		asOf = model.Now()
	}
	if hasSourceDeleted {
		query = query.Where(tableName+".source_deleted = ?", false)
	}
	return query.
		Where("("+tableName+".valid_from IS NULL OR "+tableName+".valid_from <= ?)", asOf).
		Where("("+tableName+".valid_to IS NULL OR "+tableName+".valid_to >= ?)", asOf)
}

func applyEmployeeReadScope(
	query *gorm.DB,
	tableName string,
	scope repository.OrgReadScope,
) *gorm.DB {
	if !scope.IncludeDisabled {
		query = query.Where(
			tableName+".employment_status IN ?",
			[]string{"active", "probation"},
		)
	}
	if scope.IncludeHistory {
		return query
	}
	asOf := scope.AsOf
	if asOf.IsZero() {
		asOf = model.Now()
	}
	return query.
		Where(tableName+".source_deleted = ?", false).
		Where("("+tableName+".valid_from IS NULL OR "+tableName+".valid_from <= ?)", asOf).
		Where("("+tableName+".valid_to IS NULL OR "+tableName+".valid_to >= ?)", asOf)
}

func applyAssignmentReadScope(
	query *gorm.DB,
	scope repository.OrgAssignmentReadScope,
) *gorm.DB {
	asOf := scope.AsOf
	if asOf.IsZero() {
		asOf = model.Now()
	}
	switch scope.TimeScope {
	case request.OrgAssignmentScopeHistory:
		return query.Where(
			"(org_assignment.status <> ? OR org_assignment.source_deleted = ? OR "+
				"(org_assignment.valid_to IS NOT NULL AND org_assignment.valid_to < ?))",
			"enabled",
			true,
			asOf,
		)
	case request.OrgAssignmentScopeFuture:
		return query.
			Where("org_assignment.status = ?", "enabled").
			Where("org_assignment.source_deleted = ?", false).
			Where("org_assignment.valid_from IS NOT NULL AND org_assignment.valid_from > ?", asOf)
	case request.OrgAssignmentScopeTimeline:
		return query
	default:
		return query.
			Where("org_assignment.status = ?", "enabled").
			Where("org_assignment.source_deleted = ?", false).
			Where("(org_assignment.valid_from IS NULL OR org_assignment.valid_from <= ?)", asOf).
			Where("(org_assignment.valid_to IS NULL OR org_assignment.valid_to >= ?)", asOf)
	}
}

func applyAssignmentReadOrder(
	query *gorm.DB,
	basic request.Basic,
	timeScope string,
) (*gorm.DB, request.Basic) {
	if basic.Order.Field != "" && timeScope != request.OrgAssignmentScopeTimeline {
		return query, basic
	}
	basic.Order = request.Order{}
	switch timeScope {
	case request.OrgAssignmentScopeFuture:
		query = query.Order(
			"CASE WHEN org_assignment.valid_from IS NULL THEN 1 ELSE 0 END ASC",
		).Order("org_assignment.valid_from ASC").Order("org_assignment.id ASC")
	case request.OrgAssignmentScopeHistory:
		query = query.Order(
			"CASE WHEN org_assignment.valid_to IS NULL THEN 1 ELSE 0 END ASC",
		).Order("org_assignment.valid_to DESC").Order("org_assignment.id DESC")
	default:
		query = query.Order(
			"CASE WHEN org_assignment.valid_from IS NULL THEN 1 ELSE 0 END ASC",
		).Order("org_assignment.valid_from DESC").Order("org_assignment.id DESC")
	}
	return query, basic
}

func legalEntityScopedBasic(
	source request.Basic,
	scope repository.OrgLegalEntityReadScope,
) request.Basic {
	query := cloneOrganizationBasic(source)
	// 法人主体历史记录仅由显式组织可见性标记控制，通用软删除开关不属于此 API。
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
	return append(orgLegalEntityListColumns(), "display_order", "source_deleted")
}

func orgUnitListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "unit_type",
		"primary_legal_entity_id", "status", "valid_from", "valid_to", "display_order",
	}
}

func orgUnitReadColumns() []string {
	return append(
		orgUnitListColumns(),
		"source_deleted",
		"local_note",
		"local_tags",
		"local_handling_status",
	)
}

func orgStructureListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "structure_type",
		"status", "is_default", "valid_from", "valid_to",
	}
}

func orgStructureReadColumns() []string {
	return orgStructureListColumns()
}

func orgStructureNodeListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "structure_id", "org_unit_id",
		"parent_node_id", "path", "level", "sort", "valid_from", "valid_to", "status",
	}
}

func orgStructureNodeReadColumns() []string {
	return append(orgStructureNodeListColumns(), "source_deleted")
}

func orgPositionListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "code", "name", "org_unit_id",
		"position_type", "job_level", "is_manager_position", "status", "valid_from", "valid_to",
	}
}

func orgPositionReadColumns() []string {
	return append(orgPositionListColumns(), "source_deleted")
}

func orgPositionDetailColumns() []string {
	return append(orgPositionReadColumns(), "local_note")
}

func orgEmployeeListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "employee_no", "name",
		"employment_status", "primary_legal_entity_id", "user_id", "valid_from", "valid_to",
	}
}

func orgEmployeeReadColumns() []string {
	return append(orgEmployeeListColumns(), "source_deleted")
}

func orgEmployeeDetailColumns() []string {
	return append(orgEmployeeReadColumns(), "mobile", "email", "local_note", "local_tags")
}

func orgAssignmentListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "employee_id", "legal_entity_id",
		"org_unit_id", "position_id", "assignment_type", "is_primary", "is_manager",
		"valid_from", "valid_to", "status",
	}
}

func orgAssignmentReadColumns() []string {
	return append(orgAssignmentListColumns(), "source_deleted")
}

func orgSyncBatchListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "batch_no", "execution_id",
		"sync_type", "object_scope", "started_at", "completed_at", "total_count",
		"success_count", "failed_count", "skipped_count", "status", "error_summary",
	}
}

func orgSyncRecordListColumns() []string {
	return []string{
		"id", "gmt_create", "gmt_modify", "state", "batch_id", "execution_id",
		"object_type", "source_id", "local_id", "action", "status", "error_code",
		"error_message", "dependency_type", "retry_count", "last_retry_at",
		"local_handling_status",
	}
}

// organizationQueryTable 保持 sys_table/sys_table_field 为唯一查询元数据来源，
// 同时移除元数据未开放给列表、快速查询、高级查询、排序或稳定身份使用的字段。
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

// organizationEmployeeQueryTable 遵循已确定的元数据边界：
// employee_no 和 name 是普通快速查询字段，联系方式和来源身份字段不向此读 Service 开放。
func organizationEmployeeQueryTable(table model.SysTable) model.SysTable {
	table = organizationQueryTable(table, "org_employee")
	for index := range table.TableFields {
		table.TableFields[index].IsQuickSearch =
			table.TableFields[index].FieldCode == "employee_no" ||
				table.TableFields[index].FieldCode == "name"
	}
	return table
}
