package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	dataResourceAuditType              = "data_resource"
	dataResourceCreateAction           = "create"
	dataResourceUpdateAction           = "update"
	dataResourceDisableAction          = "disable"
	dataResourceRemoveAction           = "remove"
	dataResourceOperationAddAction     = "add_operations"
	dataResourceOperationReplaceAction = "replace_operations"
	dataResourceOperationDisableAction = "disable_operation"
	dataResourceOperationRemoveAction  = "remove_operation"
)

var dataResourceCodePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._:-][a-z0-9]+)*$`)

var dataResourceOperationSet = map[string]struct{}{
	model.DataPermissionOperationQuery:  {},
	model.DataPermissionOperationDetail: {},
	model.DataPermissionOperationCreate: {},
	model.DataPermissionOperationUpdate: {},
	model.DataPermissionOperationDelete: {},
	model.DataPermissionOperationExport: {},
	model.DataPermissionOperationRun:    {},
}

// DataResourceConfigService 仅负责数据资源配置规则。
// 它不解析数据范围、不调用 Provider，也不依赖菜单权限。
type DataResourceConfigService struct {
	resourceRepo  repository.DataResourceRepository
	operationRepo repository.DataResourceOperationRepository
	ownershipRepo repository.DataOwnershipFieldRepository
	grantRepo     repository.DataGrantRepository
	sf            *utils.Snowflake
	auditWriter   TransactionalAuditWriter
}

func NewDataResourceConfigService(
	resourceRepo repository.DataResourceRepository,
	operationRepo repository.DataResourceOperationRepository,
	ownershipRepo repository.DataOwnershipFieldRepository,
	grantRepo repository.DataGrantRepository,
	sf *utils.Snowflake,
	auditWriter TransactionalAuditWriter,
) *DataResourceConfigService {
	return &DataResourceConfigService{
		resourceRepo:  resourceRepo,
		operationRepo: operationRepo,
		ownershipRepo: ownershipRepo,
		grantRepo:     grantRepo,
		sf:            sf,
		auditWriter:   auditWriter,
	}
}

func (s *DataResourceConfigService) CreateResource(
	ctx context.Context,
	req request.DataResourceCreateReq,
) (response.DataResourceDetailRes, error) {
	if ctx == nil {
		return response.DataResourceDetailRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	resource, err := newDataResourceFromRequest(req)
	if err != nil {
		return response.DataResourceDetailRes{}, err
	}
	operationItems := make([]request.DataResourceOperationCreateItemReq, 0)
	if len(req.Operations) > 0 {
		operationItems, err = normalizeDataResourceOperationItems(req.Operations)
		if err != nil {
			return response.DataResourceDetailRes{}, err
		}
	}
	if !resource.State && len(operationItems) > 0 {
		return response.DataResourceDetailRes{}, myerrors.ErrDataResourceStateInvalid
	}

	err = RunInTransaction(ctx, s.resourceRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if _, err := s.resourceRepo.FindByFieldWithDB(tx, "resource_code", resource.ResourceCode); err == nil {
			return myerrors.ErrDataResourceCodeDuplicate
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.WrapDatabaseError(err)
		}

		id, err := s.generateId()
		if err != nil {
			return err
		}
		resource.Id = id
		if err = s.resourceRepo.Create(tx, &resource); err != nil {
			if isDataPermissionConfigDuplicate(err) {
				return myerrors.ErrDataResourceCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		if !resource.State {
			if _, err = s.resourceRepo.UpdateFields(
				tx,
				resource.Id,
				map[string]any{"state": false},
			); err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
		if err = s.createResourceOperations(tx, resource.Id, operationItems); err != nil {
			return err
		}
		return s.recordResourceAudit(
			ctx,
			tx,
			dataResourceCreateAction,
			resource,
			map[string]TransactionalAuditChange{
				"resource_code":      {OldValue: nil, NewValue: resource.ResourceCode},
				"resource_type":      {OldValue: nil, NewValue: resource.ResourceType},
				"permission_enabled": {OldValue: nil, NewValue: false},
				"operations":         {OldValue: nil, NewValue: operationCodes(operationItems)},
			},
		)
	})
	if err != nil {
		return response.DataResourceDetailRes{}, err
	}
	return response.NewDataResourceDetailRes(resource), nil
}

func (s *DataResourceConfigService) UpdateResource(
	ctx context.Context,
	req request.DataResourceUpdateReq,
) (response.DataResourceDetailRes, error) {
	if ctx == nil {
		return response.DataResourceDetailRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if req.Id <= 0 {
		return response.DataResourceDetailRes{}, myerrors.NewParameterError("id必须大于0")
	}

	var updated model.DataResource
	err := RunInTransaction(ctx, s.resourceRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.findResourceForUpdate(tx, req.Id)
		if err != nil {
			return err
		}
		fields, proposed, changes, semanticsChanged, err := s.resourceUpdateFields(current, req)
		if err != nil {
			return err
		}
		if semanticsChanged {
			referenced, err := s.resourceHasReferences(tx, current.Id)
			if err != nil {
				return err
			}
			if referenced {
				return myerrors.ErrDataResourceFieldImmutable
			}
		}
		if len(fields) > 0 {
			changed, err := s.resourceRepo.UpdateFields(tx, current.Id, fields)
			if err != nil {
				return myerrors.WrapDatabaseError(err)
			}
			if !changed {
				return myerrors.ErrDataResourceNotFound
			}
		}
		if req.State != nil && !*req.State {
			if err := s.operationRepo.UpdateFieldsByResourceForConfig(
				tx,
				current.Id,
				map[string]any{"state": false, "permission_enabled": false},
			); err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
		updated, err = s.resourceRepo.FindByIdWithDB(tx, current.Id)
		if err != nil {
			return mapDataResourceReadError(err)
		}
		if len(changes) == 0 {
			return nil
		}
		proposed.Id = current.Id
		return s.recordResourceAudit(ctx, tx, dataResourceUpdateAction, proposed, changes)
	})
	if err != nil {
		return response.DataResourceDetailRes{}, err
	}
	return response.NewDataResourceDetailRes(updated), nil
}

func (s *DataResourceConfigService) GetResource(
	ctx context.Context,
	resourceId int,
) (response.DataResourceDetailRes, error) {
	if resourceId <= 0 {
		return response.DataResourceDetailRes{}, myerrors.NewParameterError("resource_id必须大于0")
	}
	resource, err := s.resourceRepo.WithContext(ctx).FindById(resourceId)
	if err != nil {
		return response.DataResourceDetailRes{}, mapDataResourceReadError(err)
	}
	return response.NewDataResourceDetailRes(resource), nil
}

func (s *DataResourceConfigService) PageResources(
	ctx context.Context,
	req request.DataResourceQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataResourceListRes], error) {
	var result response.ListResult[response.DataResourceListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	basic := req.ToBasic()
	rows, err := s.resourceRepo.GetDataResourceList(ctx, &basic, table)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.DataResourceListRes, 0, len(rows.Data))
	for _, resource := range rows.Data {
		result.Data = append(result.Data, response.NewDataResourceListRes(resource))
	}
	return result, nil
}

func (s *DataResourceConfigService) DisableResource(ctx context.Context, resourceId int) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if resourceId <= 0 {
		return myerrors.NewParameterError("resource_id必须大于0")
	}
	return RunInTransaction(ctx, s.resourceRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		resource, err := s.findResourceForUpdate(tx, resourceId)
		if err != nil {
			return err
		}
		if _, err = s.resourceRepo.UpdateFields(
			tx,
			resourceId,
			map[string]any{"state": false, "permission_enabled": false},
		); err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err = s.operationRepo.UpdateFieldsByResourceForConfig(
			tx,
			resourceId,
			map[string]any{"state": false, "permission_enabled": false},
		); err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		return s.recordResourceAudit(
			ctx,
			tx,
			dataResourceDisableAction,
			resource,
			map[string]TransactionalAuditChange{
				"state":              {OldValue: resource.State, NewValue: false},
				"permission_enabled": {OldValue: resource.PermissionEnabled, NewValue: false},
			},
		)
	})
}

// RemoveResource 执行平台软删除。
// 已被引用的资源会在持久化前被拒绝，本 Service 不执行物理删除。
func (s *DataResourceConfigService) RemoveResource(ctx context.Context, resourceId int) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if resourceId <= 0 {
		return myerrors.NewParameterError("resource_id必须大于0")
	}
	return RunInTransaction(ctx, s.resourceRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		resource, err := s.findResourceForUpdate(tx, resourceId)
		if err != nil {
			return err
		}
		referenced, err := s.resourceHasReferences(tx, resourceId)
		if err != nil {
			return err
		}
		if referenced {
			return myerrors.ErrDataResourceReferenced
		}
		if err = s.resourceRepo.DeleteById(tx, resourceId); err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		return s.recordResourceAudit(
			ctx,
			tx,
			dataResourceRemoveAction,
			resource,
			map[string]TransactionalAuditChange{
				"deleted": {OldValue: false, NewValue: true},
			},
		)
	})
}

func (s *DataResourceConfigService) AddResourceOperations(
	ctx context.Context,
	req request.DataResourceOperationBatchReq,
) ([]response.DataResourceOperationListRes, error) {
	if ctx == nil {
		return nil, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	items, err := normalizeDataResourceOperationItems(req.Items)
	if err != nil {
		return nil, err
	}
	err = RunInTransaction(ctx, s.resourceRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		resource, err := s.findConfigurableResource(tx, req.ResourceId)
		if err != nil {
			return err
		}
		for _, item := range items {
			if _, err := s.operationRepo.FindByStableKeyForConfigDB(
				tx,
				resource.Id,
				item.Operation,
			); err == nil {
				return myerrors.ErrDataResourceOperationDuplicate
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.WrapDatabaseError(err)
			}
		}
		if err = s.createResourceOperations(tx, resource.Id, items); err != nil {
			return err
		}
		return s.recordResourceAudit(
			ctx,
			tx,
			dataResourceOperationAddAction,
			resource,
			map[string]TransactionalAuditChange{
				"operations": {OldValue: nil, NewValue: operationCodes(items)},
			},
		)
	})
	if err != nil {
		return nil, err
	}
	return s.ListResourceOperations(ctx, req.ResourceId)
}

func (s *DataResourceConfigService) ReplaceResourceOperations(
	ctx context.Context,
	req request.DataResourceOperationBatchReq,
) ([]response.DataResourceOperationListRes, error) {
	if ctx == nil {
		return nil, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	items, err := normalizeDataResourceOperationItems(req.Items)
	if err != nil {
		return nil, err
	}
	err = RunInTransaction(ctx, s.resourceRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		resource, err := s.findConfigurableResource(tx, req.ResourceId)
		if err != nil {
			return err
		}
		existing, err := s.operationRepo.ListByResourceForConfigDB(tx, resource.Id)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		existingByOperation := make(map[string]model.DataResourceOperation, len(existing))
		oldCodes := make([]string, 0, len(existing))
		for _, operation := range existing {
			existingByOperation[operation.Operation] = operation
			if operation.State {
				oldCodes = append(oldCodes, operation.Operation)
			}
		}
		requested := make(map[string]request.DataResourceOperationCreateItemReq, len(items))
		for _, item := range items {
			requested[item.Operation] = item
			if current, ok := existingByOperation[item.Operation]; ok {
				fields := map[string]any{
					"state":              operationState(item.State),
					"permission_enabled": false,
					"description":        strings.TrimSpace(item.Description),
				}
				if _, err = s.operationRepo.UpdateFields(tx, current.Id, fields); err != nil {
					return myerrors.WrapDatabaseError(err)
				}
				continue
			}
			if err = s.createResourceOperations(
				tx,
				resource.Id,
				[]request.DataResourceOperationCreateItemReq{item},
			); err != nil {
				return err
			}
		}
		for _, current := range existing {
			if _, keep := requested[current.Operation]; keep {
				continue
			}
			if _, err = s.operationRepo.UpdateFields(
				tx,
				current.Id,
				map[string]any{"state": false, "permission_enabled": false},
			); err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
		return s.recordResourceAudit(
			ctx,
			tx,
			dataResourceOperationReplaceAction,
			resource,
			map[string]TransactionalAuditChange{
				"operations": {OldValue: oldCodes, NewValue: operationCodes(items)},
			},
		)
	})
	if err != nil {
		return nil, err
	}
	return s.ListResourceOperations(ctx, req.ResourceId)
}

func (s *DataResourceConfigService) ListResourceOperations(
	ctx context.Context,
	resourceId int,
) ([]response.DataResourceOperationListRes, error) {
	if resourceId <= 0 {
		return nil, myerrors.NewParameterError("resource_id必须大于0")
	}
	if _, err := s.resourceRepo.WithContext(ctx).FindById(resourceId); err != nil {
		return nil, mapDataResourceReadError(err)
	}
	rows, err := s.operationRepo.ListByResourceForConfigDB(
		s.operationRepo.DBWithContext(ctx),
		resourceId,
	)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	result := make([]response.DataResourceOperationListRes, 0, len(rows))
	for _, operation := range rows {
		result = append(result, response.NewDataResourceOperationListRes(operation))
	}
	return result, nil
}

func (s *DataResourceConfigService) DisableResourceOperation(
	ctx context.Context,
	operationId int,
) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if operationId <= 0 {
		return myerrors.NewParameterError("operation_id必须大于0")
	}
	return RunInTransaction(ctx, s.operationRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		operation, err := s.findOperationForUpdate(tx, operationId)
		if err != nil {
			return err
		}
		resource, err := s.findResourceForUpdate(tx, operation.ResourceId)
		if err != nil {
			return err
		}
		if _, err = s.operationRepo.UpdateFields(
			tx,
			operation.Id,
			map[string]any{"state": false, "permission_enabled": false},
		); err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		return s.recordResourceAudit(
			ctx,
			tx,
			dataResourceOperationDisableAction,
			resource,
			map[string]TransactionalAuditChange{
				operation.Operation: {OldValue: operation.State, NewValue: false},
			},
		)
	})
}

func (s *DataResourceConfigService) RemoveResourceOperation(
	ctx context.Context,
	operationId int,
) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if operationId <= 0 {
		return myerrors.NewParameterError("operation_id必须大于0")
	}
	return RunInTransaction(ctx, s.operationRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		operation, err := s.findOperationForUpdate(tx, operationId)
		if err != nil {
			return err
		}
		resource, err := s.findResourceForUpdate(tx, operation.ResourceId)
		if err != nil {
			return err
		}
		references, err := s.grantRepo.CountByResourceOperationForConfig(
			tx,
			operation.ResourceId,
			operation.Operation,
		)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if references > 0 {
			return myerrors.ErrDataResourceOperationReferenced
		}
		if err = s.operationRepo.DeleteById(tx, operation.Id); err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		return s.recordResourceAudit(
			ctx,
			tx,
			dataResourceOperationRemoveAction,
			resource,
			map[string]TransactionalAuditChange{
				operation.Operation: {OldValue: true, NewValue: nil},
			},
		)
	})
}

func newDataResourceFromRequest(req request.DataResourceCreateReq) (model.DataResource, error) {
	resource := model.DataResource{
		ResourceCode:      strings.TrimSpace(req.ResourceCode),
		Name:              strings.TrimSpace(req.Name),
		ResourceType:      strings.TrimSpace(req.ResourceType),
		AdapterCode:       strings.TrimSpace(req.AdapterCode),
		PermissionEnabled: false,
		Description:       strings.TrimSpace(req.Description),
	}
	resource.State = true
	if req.State != nil {
		resource.State = *req.State
	}
	if req.PermissionEnabled != nil && *req.PermissionEnabled {
		return model.DataResource{}, myerrors.ErrDataResourcePermissionEnableDenied
	}
	if err := validateDataResourceIdentity(resource.ResourceCode, resource.Name, resource.ResourceType); err != nil {
		return model.DataResource{}, err
	}
	if err := applyDataResourceTarget(&resource, req.Target); err != nil {
		return model.DataResource{}, err
	}
	if resource.AdapterCode == "" {
		return model.DataResource{}, myerrors.ErrDataResourceTargetInvalid
	}
	return resource, nil
}

func (s *DataResourceConfigService) resourceUpdateFields(
	current model.DataResource,
	req request.DataResourceUpdateReq,
) (map[string]any, model.DataResource, map[string]TransactionalAuditChange, bool, error) {
	fields := make(map[string]any)
	changes := make(map[string]TransactionalAuditChange)
	proposed := current
	semanticsChanged := false

	if req.ResourceCode != nil {
		code := strings.TrimSpace(*req.ResourceCode)
		if code != current.ResourceCode {
			return nil, proposed, nil, false, myerrors.ErrDataResourceFieldImmutable
		}
	}
	if req.PermissionEnabled != nil {
		if *req.PermissionEnabled {
			return nil, proposed, nil, false, myerrors.ErrDataResourcePermissionEnableDenied
		}
		if current.PermissionEnabled {
			fields["permission_enabled"] = false
			changes["permission_enabled"] = TransactionalAuditChange{
				OldValue: current.PermissionEnabled,
				NewValue: false,
			}
			proposed.PermissionEnabled = false
		}
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, proposed, nil, false, myerrors.ErrDataResourceNameRequired
		}
		if name != current.Name {
			fields["name"] = name
			changes["name"] = TransactionalAuditChange{OldValue: current.Name, NewValue: name}
			proposed.Name = name
		}
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if description != current.Description {
			fields["description"] = description
			changes["description"] = TransactionalAuditChange{
				OldValue: current.Description,
				NewValue: description,
			}
			proposed.Description = description
		}
	}
	if req.State != nil && *req.State != current.State {
		fields["state"] = *req.State
		changes["state"] = TransactionalAuditChange{OldValue: current.State, NewValue: *req.State}
		proposed.State = *req.State
	}
	if req.ResourceType != nil {
		resourceType := strings.TrimSpace(*req.ResourceType)
		if !isDataResourceType(resourceType) {
			return nil, proposed, nil, false, myerrors.ErrDataResourceTypeInvalid
		}
		if resourceType != current.ResourceType {
			fields["resource_type"] = resourceType
			changes["resource_type"] = TransactionalAuditChange{
				OldValue: current.ResourceType,
				NewValue: resourceType,
			}
			proposed.ResourceType = resourceType
			semanticsChanged = true
		}
	}
	if req.Target != nil {
		oldTarget := dataResourceTargetSummary(current)
		if err := applyDataResourceTarget(&proposed, *req.Target); err != nil {
			return nil, proposed, nil, false, err
		}
		newTarget := dataResourceTargetSummary(proposed)
		if oldTarget != newTarget {
			fields["table_id"] = proposed.TableId
			fields["service_code"] = proposed.ServiceCode
			fields["report_definition_id"] = proposed.ReportDefinitionId
			changes["target"] = TransactionalAuditChange{OldValue: oldTarget, NewValue: newTarget}
			semanticsChanged = true
		}
	} else if req.ResourceType != nil && proposed.ResourceType != current.ResourceType {
		return nil, proposed, nil, false, myerrors.ErrDataResourceTargetInvalid
	}
	if req.AdapterCode != nil {
		adapterCode := strings.TrimSpace(*req.AdapterCode)
		if adapterCode == "" {
			return nil, proposed, nil, false, myerrors.ErrDataResourceTargetInvalid
		}
		if adapterCode != current.AdapterCode {
			fields["adapter_code"] = adapterCode
			changes["adapter_code"] = TransactionalAuditChange{
				OldValue: current.AdapterCode,
				NewValue: adapterCode,
			}
			proposed.AdapterCode = adapterCode
			semanticsChanged = true
		}
	}
	if err := validateDataResourceIdentity(proposed.ResourceCode, proposed.Name, proposed.ResourceType); err != nil {
		return nil, proposed, nil, false, err
	}
	if err := validateDataResourceTarget(proposed); err != nil {
		return nil, proposed, nil, false, err
	}
	return fields, proposed, changes, semanticsChanged, nil
}

func validateDataResourceIdentity(code, name, resourceType string) error {
	if !dataResourceCodePattern.MatchString(code) || strings.Contains(code, "/") {
		return myerrors.ErrDataResourceCodeInvalid
	}
	if strings.TrimSpace(name) == "" {
		return myerrors.ErrDataResourceNameRequired
	}
	if !isDataResourceType(resourceType) {
		return myerrors.ErrDataResourceTypeInvalid
	}
	return nil
}

func isDataResourceType(resourceType string) bool {
	switch resourceType {
	case model.DataResourceTypeLowCodeTable,
		model.DataResourceTypeBusinessService,
		model.DataResourceTypeReport:
		return true
	default:
		return false
	}
}

func applyDataResourceTarget(resource *model.DataResource, target request.DataResourceTargetReq) error {
	resource.TableId = nil
	resource.ServiceCode = nil
	resource.ReportDefinitionId = nil
	switch resource.ResourceType {
	case model.DataResourceTypeLowCodeTable:
		if target.ReferenceId == nil || *target.ReferenceId <= 0 || target.ReferenceCode != nil {
			return myerrors.ErrDataResourceTargetInvalid
		}
		resource.TableId = cloneDataResourceInt(target.ReferenceId)
	case model.DataResourceTypeBusinessService:
		if target.ReferenceCode == nil || strings.TrimSpace(*target.ReferenceCode) == "" || target.ReferenceId != nil {
			return myerrors.ErrDataResourceTargetInvalid
		}
		code := strings.TrimSpace(*target.ReferenceCode)
		resource.ServiceCode = &code
	case model.DataResourceTypeReport:
		if target.ReferenceId == nil || *target.ReferenceId <= 0 || target.ReferenceCode != nil {
			return myerrors.ErrDataResourceTargetInvalid
		}
		resource.ReportDefinitionId = cloneDataResourceInt(target.ReferenceId)
	default:
		return myerrors.ErrDataResourceTypeInvalid
	}
	return nil
}

func validateDataResourceTarget(resource model.DataResource) error {
	target := request.DataResourceTargetReq{}
	switch resource.ResourceType {
	case model.DataResourceTypeLowCodeTable:
		target.ReferenceId = resource.TableId
	case model.DataResourceTypeBusinessService:
		target.ReferenceCode = resource.ServiceCode
	case model.DataResourceTypeReport:
		target.ReferenceId = resource.ReportDefinitionId
	default:
		return myerrors.ErrDataResourceTypeInvalid
	}
	copy := resource
	return applyDataResourceTarget(&copy, target)
}

func normalizeDataResourceOperationItems(
	items []request.DataResourceOperationCreateItemReq,
) ([]request.DataResourceOperationCreateItemReq, error) {
	if len(items) == 0 {
		return nil, myerrors.NewParameterError("操作列表不能为空")
	}
	if len(items) > len(dataResourceOperationSet) {
		return nil, myerrors.NewParameterError("操作数量超过允许范围")
	}
	result := make([]request.DataResourceOperationCreateItemReq, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item.Operation = strings.TrimSpace(item.Operation)
		item.Description = strings.TrimSpace(item.Description)
		if _, ok := dataResourceOperationSet[item.Operation]; !ok {
			return nil, myerrors.ErrDataResourceOperationInvalid
		}
		if item.PermissionEnabled != nil && *item.PermissionEnabled {
			return nil, myerrors.ErrDataResourcePermissionEnableDenied
		}
		if _, duplicate := seen[item.Operation]; duplicate {
			return nil, myerrors.ErrDataResourceOperationDuplicate
		}
		seen[item.Operation] = struct{}{}
		result = append(result, item)
	}
	return result, nil
}

func (s *DataResourceConfigService) createResourceOperations(
	tx *gorm.DB,
	resourceId int,
	items []request.DataResourceOperationCreateItemReq,
) error {
	for _, item := range items {
		id, err := s.generateId()
		if err != nil {
			return err
		}
		operation := model.DataResourceOperation{
			Basic: model.Basic{
				Id:    id,
				State: operationState(item.State),
			},
			ResourceId:        resourceId,
			Operation:         item.Operation,
			PermissionEnabled: false,
			Description:       item.Description,
		}
		if err = s.operationRepo.Create(tx, &operation); err != nil {
			if isDataPermissionConfigDuplicate(err) {
				return myerrors.ErrDataResourceOperationDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		if !operation.State {
			if _, err = s.operationRepo.UpdateFields(
				tx,
				operation.Id,
				map[string]any{"state": false},
			); err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
	}
	return nil
}

func (s *DataResourceConfigService) findResourceForUpdate(
	tx *gorm.DB,
	resourceId int,
) (model.DataResource, error) {
	resource, err := s.resourceRepo.FindByIdWithDB(tx, resourceId)
	if err != nil {
		return model.DataResource{}, mapDataResourceReadError(err)
	}
	return resource, nil
}

func (s *DataResourceConfigService) findConfigurableResource(
	tx *gorm.DB,
	resourceId int,
) (model.DataResource, error) {
	if resourceId <= 0 {
		return model.DataResource{}, myerrors.NewParameterError("resource_id必须大于0")
	}
	resource, err := s.findResourceForUpdate(tx, resourceId)
	if err != nil {
		return model.DataResource{}, err
	}
	if !resource.State {
		return model.DataResource{}, myerrors.ErrDataResourceStateInvalid
	}
	return resource, nil
}

func (s *DataResourceConfigService) findOperationForUpdate(
	tx *gorm.DB,
	operationId int,
) (model.DataResourceOperation, error) {
	operation, err := s.operationRepo.FindByIdWithDB(tx, operationId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.DataResourceOperation{}, myerrors.ErrDataResourceOperationNotFound
	}
	if err != nil {
		return model.DataResourceOperation{}, myerrors.WrapDatabaseError(err)
	}
	return operation, nil
}

func (s *DataResourceConfigService) resourceHasReferences(tx *gorm.DB, resourceId int) (bool, error) {
	operationCount, err := s.operationRepo.CountByResourceForConfig(tx, resourceId)
	if err != nil {
		return false, myerrors.WrapDatabaseError(err)
	}
	ownershipCount, err := s.ownershipRepo.CountByResourceForConfig(tx, resourceId)
	if err != nil {
		return false, myerrors.WrapDatabaseError(err)
	}
	grantCount, err := s.grantRepo.CountByResourceForConfig(tx, resourceId)
	if err != nil {
		return false, myerrors.WrapDatabaseError(err)
	}
	return operationCount > 0 || ownershipCount > 0 || grantCount > 0, nil
}

func (s *DataResourceConfigService) generateId() (int, error) {
	if s.sf == nil {
		return 0, myerrors.WrapSystemError(errors.New("data resource id generator is required"))
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, myerrors.WrapSystemError(err)
	}
	return int(id), nil
}

func (s *DataResourceConfigService) recordResourceAudit(
	ctx context.Context,
	tx *gorm.DB,
	action string,
	resource model.DataResource,
	changes map[string]TransactionalAuditChange,
) error {
	if s.auditWriter == nil {
		return myerrors.WrapSystemError(ErrTransactionalAuditRepositoryRequired)
	}
	if err := s.auditWriter.RecordTransactionalAudit(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: dataResourceAuditType,
		ResourceCode: resource.ResourceCode,
		ResourceId:   strconv.Itoa(resource.Id),
		Changes:      changes,
	}); err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func mapDataResourceReadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataResourceNotFound
	}
	return myerrors.WrapDatabaseError(err)
}

func isDataPermissionConfigDuplicate(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func operationState(state *bool) bool {
	return state == nil || *state
}

func operationCodes(items []request.DataResourceOperationCreateItemReq) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Operation)
	}
	return result
}

func dataResourceTargetSummary(resource model.DataResource) string {
	switch resource.ResourceType {
	case model.DataResourceTypeLowCodeTable:
		if resource.TableId != nil {
			return "table:" + strconv.Itoa(*resource.TableId)
		}
	case model.DataResourceTypeBusinessService:
		if resource.ServiceCode != nil {
			return "service:" + *resource.ServiceCode
		}
	case model.DataResourceTypeReport:
		if resource.ReportDefinitionId != nil {
			return "report:" + strconv.Itoa(*resource.ReportDefinitionId)
		}
	}
	return ""
}

func cloneDataResourceInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
