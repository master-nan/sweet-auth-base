package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	dataOwnershipAuditType     = "data_ownership"
	dataOwnershipCreateAction  = "create"
	dataOwnershipUpdateAction  = "update"
	dataOwnershipDisableAction = "disable"
	dataOwnershipRemoveAction  = "remove"
)

var dataOwnershipCodePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

// DataOwnershipConfigService owns ownership-definition configuration only.
// It does not resolve subject scope, call providers, or build query filters.
type DataOwnershipConfigService struct {
	resourceRepo             repository.DataResourceRepository
	dimensionRepo            repository.DataDimensionDefinitionRepository
	ownershipRepo            repository.DataOwnershipFieldRepository
	tableFieldRepo           repository.SysTableFieldRepository
	registeredFieldValidator datapermission.OwnershipFieldBindingValidator
	sf                       *utils.Snowflake
	auditWriter              TransactionalAuditWriter
}

func NewDataOwnershipConfigService(
	resourceRepo repository.DataResourceRepository,
	dimensionRepo repository.DataDimensionDefinitionRepository,
	ownershipRepo repository.DataOwnershipFieldRepository,
	tableFieldRepo repository.SysTableFieldRepository,
	registeredFieldValidator datapermission.OwnershipFieldBindingValidator,
	sf *utils.Snowflake,
	auditWriter TransactionalAuditWriter,
) *DataOwnershipConfigService {
	return &DataOwnershipConfigService{
		resourceRepo:             resourceRepo,
		dimensionRepo:            dimensionRepo,
		ownershipRepo:            ownershipRepo,
		tableFieldRepo:           tableFieldRepo,
		registeredFieldValidator: registeredFieldValidator,
		sf:                       sf,
		auditWriter:              auditWriter,
	}
}

func (s *DataOwnershipConfigService) PageDimensions(
	ctx *gin.Context,
	req request.DataDimensionDefinitionQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataDimensionDefinitionListRes], error) {
	var result response.ListResult[response.DataDimensionDefinitionListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	rows, err := s.dimensionRepo.Query(ctx, &req, table)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.DataDimensionDefinitionListRes, 0, len(rows.Data))
	for _, dimension := range rows.Data {
		result.Data = append(result.Data, response.NewDataDimensionDefinitionListRes(dimension))
	}
	return result, nil
}

func (s *DataOwnershipConfigService) CreateOwnership(
	ctx *gin.Context,
	req request.DataOwnershipFieldCreateReq,
) (response.DataOwnershipFieldDetailRes, error) {
	if ctx == nil {
		return response.DataOwnershipFieldDetailRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	ownership := newDataOwnershipFromRequest(req)
	if err := validateDataOwnershipIdentity(ownership); err != nil {
		return response.DataOwnershipFieldDetailRes{}, err
	}

	err := RunInTransaction(ctx, s.ownershipRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		resource, err := s.findConfigurableResource(tx, ownership.ResourceId)
		if err != nil {
			return err
		}
		dimension, err := s.findActiveDimension(tx, ownership.DimensionId)
		if err != nil {
			return err
		}
		if _, err = s.ownershipRepo.FindByStableKeyForConfigDB(
			tx,
			ownership.ResourceId,
			ownership.OwnershipCode,
		); err == nil {
			return myerrors.ErrDataOwnershipDuplicate
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.WrapDatabaseError(err)
		}
		if err = s.validateOwnershipBinding(tx, resource, dimension, ownership); err != nil {
			return err
		}
		ownership.Id, err = s.generateId()
		if err != nil {
			return err
		}
		if err = s.ownershipRepo.Create(tx, &ownership); err != nil {
			if isDataPermissionConfigDuplicate(err) {
				return myerrors.ErrDataOwnershipDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		if !ownership.State {
			if _, err = s.ownershipRepo.UpdateFieldsForConfig(
				tx,
				ownership.Id,
				map[string]any{"state": false},
			); err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
		return s.recordOwnershipAudit(
			ctx,
			tx,
			dataOwnershipCreateAction,
			ownership,
			map[string]TransactionalAuditChange{
				"resource_id":    {OldValue: nil, NewValue: ownership.ResourceId},
				"ownership_code": {OldValue: nil, NewValue: ownership.OwnershipCode},
				"dimension_id":   {OldValue: nil, NewValue: ownership.DimensionId},
				"binding_type":   {OldValue: nil, NewValue: ownership.BindingType},
				"value_type":     {OldValue: nil, NewValue: ownership.ValueType},
			},
		)
	})
	if err != nil {
		return response.DataOwnershipFieldDetailRes{}, err
	}
	return s.ownershipDetail(ctx, ownership)
}

func (s *DataOwnershipConfigService) GetOwnership(
	ctx *gin.Context,
	ownershipId int,
) (response.DataOwnershipFieldDetailRes, error) {
	if ownershipId <= 0 {
		return response.DataOwnershipFieldDetailRes{}, myerrors.NewParameterError("ownership_id必须大于0")
	}
	ownership, err := s.ownershipRepo.FindByIdForConfig(ctx, ownershipId)
	if err != nil {
		return response.DataOwnershipFieldDetailRes{}, mapDataOwnershipReadError(err)
	}
	return s.ownershipDetail(ctx, ownership)
}

func (s *DataOwnershipConfigService) PageOwnerships(
	ctx *gin.Context,
	req request.DataOwnershipFieldQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataOwnershipFieldListRes], error) {
	var result response.ListResult[response.DataOwnershipFieldListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	rows, err := s.ownershipRepo.Query(ctx, &req, table)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.DataOwnershipFieldListRes, 0, len(rows.Data))
	for _, ownership := range rows.Data {
		result.Data = append(result.Data, response.NewDataOwnershipFieldListRes(ownership))
	}
	return result, nil
}

func (s *DataOwnershipConfigService) ListOwnershipsByResource(
	ctx *gin.Context,
	resourceId int,
) ([]response.DataOwnershipFieldListRes, error) {
	if resourceId <= 0 {
		return nil, myerrors.NewParameterError("resource_id必须大于0")
	}
	if _, err := s.resourceRepo.FindByIdForConfig(ctx, resourceId); err != nil {
		return nil, mapDataResourceReadError(err)
	}
	rows, err := s.ownershipRepo.ListByResourceForConfigDB(
		s.ownershipRepo.DBWithContext(ctx),
		resourceId,
	)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	result := make([]response.DataOwnershipFieldListRes, 0, len(rows))
	for _, ownership := range rows {
		result = append(result, response.NewDataOwnershipFieldListRes(ownership))
	}
	return result, nil
}

func (s *DataOwnershipConfigService) UpdateOwnership(
	ctx *gin.Context,
	req request.DataOwnershipFieldUpdateReq,
) (response.DataOwnershipFieldDetailRes, error) {
	if ctx == nil {
		return response.DataOwnershipFieldDetailRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if req.Id <= 0 {
		return response.DataOwnershipFieldDetailRes{}, myerrors.NewParameterError("ownership_id必须大于0")
	}

	var updated model.DataOwnershipField
	err := RunInTransaction(ctx, s.ownershipRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.findOwnershipForUpdate(tx, req.Id)
		if err != nil {
			return err
		}
		if !ownershipIdentityMatchesRequest(current, req) {
			return myerrors.ErrDataOwnershipFieldImmutable
		}
		updated = current
		if req.State == nil || *req.State == current.State {
			return nil
		}
		if !*req.State {
			referenced, err := s.ownershipHasPolicyReferences(tx, current, true)
			if err != nil {
				return err
			}
			if referenced {
				return myerrors.ErrDataOwnershipReferenced
			}
		} else {
			resource, err := s.findConfigurableResource(tx, current.ResourceId)
			if err != nil {
				return err
			}
			dimension, err := s.findActiveDimension(tx, current.DimensionId)
			if err != nil {
				return err
			}
			if err = s.validateOwnershipBinding(tx, resource, dimension, current); err != nil {
				return err
			}
		}
		changed, err := s.ownershipRepo.UpdateFieldsForConfig(
			tx,
			current.Id,
			map[string]any{"state": *req.State},
		)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !changed {
			return myerrors.ErrDataOwnershipNotFound
		}
		updated.State = *req.State
		return s.recordOwnershipAudit(
			ctx,
			tx,
			dataOwnershipUpdateAction,
			updated,
			map[string]TransactionalAuditChange{
				"state": {OldValue: current.State, NewValue: *req.State},
			},
		)
	})
	if err != nil {
		return response.DataOwnershipFieldDetailRes{}, err
	}
	return s.ownershipDetail(ctx, updated)
}

func (s *DataOwnershipConfigService) DisableOwnership(ctx *gin.Context, ownershipId int) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if ownershipId <= 0 {
		return myerrors.NewParameterError("ownership_id必须大于0")
	}
	return RunInTransaction(ctx, s.ownershipRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		ownership, err := s.findOwnershipForUpdate(tx, ownershipId)
		if err != nil {
			return err
		}
		if !ownership.State {
			return nil
		}
		referenced, err := s.ownershipHasPolicyReferences(tx, ownership, true)
		if err != nil {
			return err
		}
		if referenced {
			return myerrors.ErrDataOwnershipReferenced
		}
		changed, err := s.ownershipRepo.UpdateFieldsForConfig(
			tx,
			ownership.Id,
			map[string]any{"state": false},
		)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !changed {
			return myerrors.ErrDataOwnershipNotFound
		}
		return s.recordOwnershipAudit(
			ctx,
			tx,
			dataOwnershipDisableAction,
			ownership,
			map[string]TransactionalAuditChange{
				"state": {OldValue: true, NewValue: false},
			},
		)
	})
}

// RemoveOwnership follows the platform soft-delete baseline. Any policy-rule
// reference, active or inactive, protects the ownership identity.
func (s *DataOwnershipConfigService) RemoveOwnership(ctx *gin.Context, ownershipId int) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if ownershipId <= 0 {
		return myerrors.NewParameterError("ownership_id必须大于0")
	}
	return RunInTransaction(ctx, s.ownershipRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		ownership, err := s.findOwnershipForUpdate(tx, ownershipId)
		if err != nil {
			return err
		}
		referenced, err := s.ownershipHasPolicyReferences(tx, ownership, false)
		if err != nil {
			return err
		}
		if referenced {
			return myerrors.ErrDataOwnershipReferenced
		}
		if err = s.ownershipRepo.DeleteById(tx, ownership.Id); err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		return s.recordOwnershipAudit(
			ctx,
			tx,
			dataOwnershipRemoveAction,
			ownership,
			map[string]TransactionalAuditChange{
				"deleted": {OldValue: false, NewValue: true},
			},
		)
	})
}

func newDataOwnershipFromRequest(req request.DataOwnershipFieldCreateReq) model.DataOwnershipField {
	ownership := model.DataOwnershipField{
		Basic: model.Basic{
			State: req.State == nil || *req.State,
		},
		ResourceId:    req.ResourceId,
		OwnershipCode: strings.TrimSpace(req.OwnershipCode),
		DimensionId:   req.DimensionId,
		BindingType:   strings.TrimSpace(req.BindingType),
		ValueType:     strings.TrimSpace(req.ValueType),
	}
	switch ownership.BindingType {
	case model.DataOwnershipBindingTypeMetadataField:
		ownership.TableFieldId = cloneDataResourceInt(req.BindingTarget.ReferenceId)
	case model.DataOwnershipBindingTypeRegisteredField:
		if req.BindingTarget.ReferenceCode != nil {
			code := strings.TrimSpace(*req.BindingTarget.ReferenceCode)
			ownership.AdapterFieldCode = &code
		}
	}
	return ownership
}

func validateDataOwnershipIdentity(ownership model.DataOwnershipField) error {
	if ownership.ResourceId <= 0 || ownership.DimensionId <= 0 {
		return myerrors.ErrDataOwnershipBindingInvalid
	}
	if !dataOwnershipCodePattern.MatchString(ownership.OwnershipCode) {
		return myerrors.ErrDataOwnershipCodeInvalid
	}
	if ownership.ValueType != model.DataDimensionValueTypeBigint &&
		ownership.ValueType != model.DataDimensionValueTypeString {
		return myerrors.ErrDataOwnershipValueTypeMismatch
	}
	switch ownership.BindingType {
	case model.DataOwnershipBindingTypeMetadataField:
		if ownership.TableFieldId == nil || *ownership.TableFieldId <= 0 || ownership.AdapterFieldCode != nil {
			return myerrors.ErrDataOwnershipBindingInvalid
		}
	case model.DataOwnershipBindingTypeRegisteredField:
		if ownership.AdapterFieldCode == nil ||
			!dataOwnershipCodePattern.MatchString(*ownership.AdapterFieldCode) ||
			ownership.TableFieldId != nil {
			return myerrors.ErrDataOwnershipRegisteredFieldInvalid
		}
	default:
		return myerrors.ErrDataOwnershipBindingInvalid
	}
	return nil
}

func (s *DataOwnershipConfigService) validateOwnershipBinding(
	tx *gorm.DB,
	resource model.DataResource,
	dimension model.DataDimensionDefinition,
	ownership model.DataOwnershipField,
) error {
	if ownership.ValueType != dimension.ValueType {
		return myerrors.ErrDataOwnershipValueTypeMismatch
	}
	switch ownership.BindingType {
	case model.DataOwnershipBindingTypeMetadataField:
		if resource.ResourceType != model.DataResourceTypeLowCodeTable ||
			resource.TableId == nil ||
			ownership.TableFieldId == nil {
			return myerrors.ErrDataOwnershipBindingInvalid
		}
		field, err := s.tableFieldRepo.FindByIdForConfigDB(tx, *ownership.TableFieldId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrDataOwnershipMetadataFieldNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !field.State {
			return myerrors.ErrDataOwnershipMetadataFieldNotFound
		}
		if field.TableId != *resource.TableId {
			return myerrors.ErrDataOwnershipMetadataFieldMismatch
		}
		if !isMetadataOwnershipField(field) {
			return myerrors.ErrDataOwnershipMetadataFieldForbidden
		}
		if !metadataFieldMatchesOwnershipValueType(field.FieldType, ownership.ValueType) {
			return myerrors.ErrDataOwnershipValueTypeMismatch
		}
		if !metadataFieldMatchesDimension(field.FieldCode, dimension.Code) {
			return myerrors.ErrDataOwnershipMetadataDimension
		}
	case model.DataOwnershipBindingTypeRegisteredField:
		if ownership.AdapterFieldCode == nil ||
			!dataOwnershipCodePattern.MatchString(*ownership.AdapterFieldCode) {
			return myerrors.ErrDataOwnershipRegisteredFieldInvalid
		}
		if resource.ResourceType != model.DataResourceTypeBusinessService ||
			strings.TrimSpace(resource.AdapterCode) == "" {
			return myerrors.ErrDataOwnershipBindingInvalid
		}
		if s.registeredFieldValidator == nil {
			return myerrors.ErrDataOwnershipRegisteredFieldMissing
		}
		if err := s.registeredFieldValidator.ValidateBinding(
			datapermission.OwnershipFieldBindingValidation{
				ResourceCode:     resource.ResourceCode,
				OwnershipCode:    ownership.OwnershipCode,
				AdapterFieldCode: *ownership.AdapterFieldCode,
				ValueType:        ownership.ValueType,
				DimensionCode:    dimension.Code,
			},
		); err != nil {
			return mapRegisteredOwnershipFieldValidationError(err)
		}
	default:
		return myerrors.ErrDataOwnershipBindingInvalid
	}
	return nil
}

func isMetadataOwnershipField(field model.SysTableField) bool {
	if field.Expression != nil && strings.TrimSpace(*field.Expression) != "" {
		return false
	}
	if field.IsPrimaryKey {
		return false
	}
	if field.FieldCategory != "" && field.FieldCategory != enum.NormalField {
		return false
	}
	return !isForbiddenMetadataOwnershipFieldCode(field.FieldCode)
}

func metadataFieldMatchesOwnershipValueType(
	fieldType enum.SysTableFieldType,
	valueType string,
) bool {
	switch valueType {
	case model.DataDimensionValueTypeBigint:
		return fieldType == enum.BigIntFieldType ||
			fieldType == enum.IntFieldType ||
			fieldType == enum.TinyintFieldType
	case model.DataDimensionValueTypeString:
		return fieldType == enum.VarcharFieldType || fieldType == enum.TextFieldType
	default:
		return false
	}
}

func isForbiddenMetadataOwnershipFieldCode(fieldCode string) bool {
	fieldCode = strings.ToLower(strings.TrimSpace(fieldCode))
	if fieldCode == "" {
		return true
	}
	for _, prefix := range []string{
		"gmt_",
		"source_",
		"create_",
		"modify_",
		"delete_",
	} {
		if strings.HasPrefix(fieldCode, prefix) {
			return true
		}
	}
	switch fieldCode {
	case "id",
		"path",
		"level",
		"parent_id",
		"parent_node_id",
		"structure_node_id",
		"tree_path",
		"node_path",
		"name",
		"display_name",
		"display_value",
		"label":
		return true
	}
	return strings.HasSuffix(fieldCode, "_name") ||
		strings.HasSuffix(fieldCode, "_label") ||
		strings.HasSuffix(fieldCode, "_display")
}

func metadataFieldMatchesDimension(fieldCode, dimensionCode string) bool {
	fieldCode = strings.ToLower(strings.TrimSpace(fieldCode))
	dimensionCode = strings.ToLower(strings.TrimSpace(dimensionCode))
	var expectedDimension string
	switch fieldCode {
	case "legal_entity_id", "primary_legal_entity_id":
		expectedDimension = "legal_entity"
	case "org_unit_id", "owner_org_id", "management_org_id", "primary_org_unit_id":
		expectedDimension = "management_org"
	case "employee_id", "owner_employee_id":
		expectedDimension = "employee"
	}
	return expectedDimension == "" || expectedDimension == dimensionCode
}

func mapRegisteredOwnershipFieldValidationError(err error) error {
	switch {
	case errors.Is(err, datapermission.ErrOwnershipFieldResourceMismatch):
		return myerrors.ErrDataOwnershipRegisteredResource
	case errors.Is(err, datapermission.ErrOwnershipFieldValueTypeMismatch):
		return myerrors.ErrDataOwnershipValueTypeMismatch
	case errors.Is(err, datapermission.ErrOwnershipFieldDimensionUnsupported):
		return myerrors.ErrDataOwnershipRegisteredDimension
	case errors.Is(err, datapermission.ErrOwnershipFieldOperationUnsupported):
		return myerrors.ErrDataOwnershipRegisteredOperation
	case errors.Is(err, datapermission.ErrOwnershipFieldRegistrationNotFound):
		return myerrors.ErrDataOwnershipRegisteredFieldMissing
	default:
		return myerrors.ErrDataOwnershipRegisteredFieldInvalid
	}
}

func ownershipIdentityMatchesRequest(
	current model.DataOwnershipField,
	req request.DataOwnershipFieldUpdateReq,
) bool {
	if req.ResourceId != nil && *req.ResourceId != current.ResourceId {
		return false
	}
	if req.OwnershipCode != nil && strings.TrimSpace(*req.OwnershipCode) != current.OwnershipCode {
		return false
	}
	if req.DimensionId != nil && *req.DimensionId != current.DimensionId {
		return false
	}
	if req.BindingType != nil && strings.TrimSpace(*req.BindingType) != current.BindingType {
		return false
	}
	if req.ValueType != nil && strings.TrimSpace(*req.ValueType) != current.ValueType {
		return false
	}
	if req.BindingTarget != nil && !ownershipBindingTargetMatches(current, *req.BindingTarget) {
		return false
	}
	return true
}

func ownershipBindingTargetMatches(
	current model.DataOwnershipField,
	target request.DataOwnershipBindingTargetReq,
) bool {
	switch current.BindingType {
	case model.DataOwnershipBindingTypeMetadataField:
		return target.ReferenceId != nil &&
			current.TableFieldId != nil &&
			*target.ReferenceId == *current.TableFieldId &&
			target.ReferenceCode == nil
	case model.DataOwnershipBindingTypeRegisteredField:
		return target.ReferenceCode != nil &&
			current.AdapterFieldCode != nil &&
			strings.TrimSpace(*target.ReferenceCode) == *current.AdapterFieldCode &&
			target.ReferenceId == nil
	default:
		return false
	}
}

func (s *DataOwnershipConfigService) findConfigurableResource(
	tx *gorm.DB,
	resourceId int,
) (model.DataResource, error) {
	if resourceId <= 0 {
		return model.DataResource{}, myerrors.NewParameterError("resource_id必须大于0")
	}
	resource, err := s.resourceRepo.FindByIdForConfigDB(tx, resourceId)
	if err != nil {
		return model.DataResource{}, mapDataResourceReadError(err)
	}
	if !resource.State {
		return model.DataResource{}, myerrors.ErrDataResourceStateInvalid
	}
	return resource, nil
}

func (s *DataOwnershipConfigService) findActiveDimension(
	tx *gorm.DB,
	dimensionId int,
) (model.DataDimensionDefinition, error) {
	dimension, err := s.dimensionRepo.FindByIdForConfigDB(tx, dimensionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.DataDimensionDefinition{}, myerrors.ErrDataDimensionNotFound
	}
	if err != nil {
		return model.DataDimensionDefinition{}, myerrors.WrapDatabaseError(err)
	}
	if !dimension.State {
		return model.DataDimensionDefinition{}, myerrors.ErrDataDimensionNotFound
	}
	return dimension, nil
}

func (s *DataOwnershipConfigService) findOwnershipForUpdate(
	tx *gorm.DB,
	ownershipId int,
) (model.DataOwnershipField, error) {
	ownership, err := s.ownershipRepo.FindByIdForConfigDB(tx, ownershipId)
	if err != nil {
		return model.DataOwnershipField{}, mapDataOwnershipReadError(err)
	}
	return ownership, nil
}

func (s *DataOwnershipConfigService) ownershipHasPolicyReferences(
	tx *gorm.DB,
	ownership model.DataOwnershipField,
	activeOnly bool,
) (bool, error) {
	count, err := s.ownershipRepo.CountPolicyRuleReferencesForConfig(
		tx,
		ownership.ResourceId,
		ownership.OwnershipCode,
		ownership.DimensionId,
		activeOnly,
		model.Now(),
	)
	if err != nil {
		return false, myerrors.WrapDatabaseError(err)
	}
	return count > 0, nil
}

func (s *DataOwnershipConfigService) ownershipDetail(
	ctx *gin.Context,
	ownership model.DataOwnershipField,
) (response.DataOwnershipFieldDetailRes, error) {
	resource, err := s.resourceRepo.FindByIdForConfig(ctx, ownership.ResourceId)
	if err != nil {
		return response.DataOwnershipFieldDetailRes{}, mapDataResourceReadError(err)
	}
	dimension, err := s.dimensionRepo.FindByIdForConfig(ctx, ownership.DimensionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.DataOwnershipFieldDetailRes{}, myerrors.ErrDataDimensionNotFound
	}
	if err != nil {
		return response.DataOwnershipFieldDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	result := response.NewDataOwnershipFieldDetailRes(ownership)
	result.Resource = &response.DataPermissionReferenceSummaryRes{
		Id:   resource.Id,
		Code: resource.ResourceCode,
		Name: resource.Name,
	}
	result.Dimension = &response.DataPermissionReferenceSummaryRes{
		Id:   dimension.Id,
		Code: dimension.Code,
		Name: dimension.Name,
	}
	return result, nil
}

func (s *DataOwnershipConfigService) generateId() (int, error) {
	if s.sf == nil {
		return 0, myerrors.WrapSystemError(errors.New("data ownership id generator is required"))
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, myerrors.WrapSystemError(err)
	}
	return int(id), nil
}

func (s *DataOwnershipConfigService) recordOwnershipAudit(
	ctx *gin.Context,
	tx *gorm.DB,
	action string,
	ownership model.DataOwnershipField,
	changes map[string]TransactionalAuditChange,
) error {
	if s.auditWriter == nil {
		return myerrors.WrapSystemError(ErrTransactionalAuditRepositoryRequired)
	}
	if err := s.auditWriter.RecordTransactionalAudit(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: dataOwnershipAuditType,
		ResourceCode: strconv.Itoa(ownership.ResourceId) + ":" + ownership.OwnershipCode,
		ResourceId:   strconv.Itoa(ownership.Id),
		Changes:      changes,
	}); err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func mapDataOwnershipReadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataOwnershipNotFound
	}
	return myerrors.WrapDatabaseError(err)
}
