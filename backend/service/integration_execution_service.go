package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/integration"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	integrationIdempotencyScopePattern = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{0,63}$`)
	integrationInputHashPattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	errIntegrationIdempotencyRace      = errors.New("integration execution idempotency race")
)

const (
	integrationExecutionAuditResourceType = "integration_execution"
	integrationExecutionAuditCreate       = "integration.execution.create"
	integrationExecutionAuditCancel       = "integration.execution.cancel"
)

type IntegrationExecutionService struct {
	executions repository.IntegrationExecutionRepository
	logs       repository.IntegrationLogRepository
	systems    repository.ExternalSystemRepository
	interfaces repository.InterfaceDefinitionRepository
	policies   repository.RetryPolicyRepository
	sf         *utils.Snowflake
	audit      StandardContextAuditWriter
	now        func() time.Time
}

func NewIntegrationExecutionService(
	executions repository.IntegrationExecutionRepository,
	logs repository.IntegrationLogRepository,
	systems repository.ExternalSystemRepository,
	interfaces repository.InterfaceDefinitionRepository,
	policies repository.RetryPolicyRepository,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
) *IntegrationExecutionService {
	return &IntegrationExecutionService{
		executions: executions,
		logs:       logs,
		systems:    systems,
		interfaces: interfaces,
		policies:   policies,
		sf:         sf,
		audit:      audit,
		now:        model.Now,
	}
}

func (s *IntegrationExecutionService) CreateExecution(
	ctx context.Context,
	req request.IntegrationExecutionCreateReq,
) (response.IntegrationExecutionDetailRes, error) {
	req, err := normalizeIntegrationExecutionCreateReq(req)
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, err
	}

	var value model.IntegrationExecution
	var interfaceVersion int
	var serverInputHash string
	err = RunInTransaction(ctx, s.executions.DBWithContext(ctx), func(tx *gorm.DB) error {
		system, definition, err := s.loadExecutionConfiguration(tx, req)
		if err != nil {
			return err
		}
		interfaceVersion = definition.Version
		_, snapshot, inputHash, err := integration.BuildExecutionInputSnapshot(
			definition.InputContract,
			definition.HTTPMethod,
			definition.RelativePath,
			definition.Version,
			integration.ExecutionInputValues{
				PathParams: req.Input.PathParams, QueryParams: req.Input.QueryParams,
				Headers: req.Input.Headers, JSONBody: req.Input.JSONBody,
			},
		)
		if err != nil {
			return err
		}
		serverInputHash = inputHash
		if req.InputHash != "" && req.InputHash != serverInputHash {
			return myerrors.ErrIntegrationExecutionInputHashMismatch
		}

		existing, err := s.executions.FindByIdempotency(
			tx,
			definition.Id,
			definition.Version,
			req.IdempotencyScope,
			req.IdempotencyKey,
		)
		if err == nil {
			if existing.InputHash != serverInputHash {
				return myerrors.ErrIntegrationExecutionIdempotencyConflict
			}
			value = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.WrapDatabaseError(err)
		}

		retryPolicyID, retryPolicySnapshot, retryPolicySnapshotVersion, err := s.freezeRetryPolicy(tx, definition)
		if err != nil {
			return err
		}

		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return myerrors.WrapSystemError(err)
		}
		value = model.IntegrationExecution{
			Basic:                      model.Basic{Id: int(id), State: true},
			ExecutionNo:                fmt.Sprintf("INT-%d", id),
			ExternalSystemID:           system.Id,
			ExternalSystemCode:         system.SystemCode,
			ExternalSystemName:         system.Name,
			InterfaceDefinitionID:      definition.Id,
			InterfaceCode:              definition.InterfaceCode,
			InterfaceName:              definition.Name,
			InterfaceVersion:           definition.Version,
			TriggerSource:              req.TriggerSource,
			Status:                     model.IntegrationExecutionStatusCreated,
			IdempotencyScope:           req.IdempotencyScope,
			IdempotencyKey:             req.IdempotencyKey,
			InputHash:                  serverInputHash,
			InputSnapshot:              datatypes.JSON(snapshot),
			InputSnapshotVersion:       integration.ExecutionInputSnapshotVersion,
			InputSnapshotSize:          len(snapshot),
			RetryPolicyID:              retryPolicyID,
			RetryPolicySnapshot:        datatypes.JSON(retryPolicySnapshot),
			RetryPolicySnapshotVersion: retryPolicySnapshotVersion,
			Revision:                   1,
		}
		if err := s.executions.Create(tx, &value); err != nil {
			if isIntegrationExecutionIdempotencyDuplicate(err) {
				return errIntegrationIdempotencyRace
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, integrationExecutionAuditCreate, value, "", value.Status)
	})
	if errors.Is(err, errIntegrationIdempotencyRace) {
		return s.resolveIdempotencyRace(ctx, req, interfaceVersion, serverInputHash)
	}
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, err
	}
	return response.NewIntegrationExecutionDetailRes(value), nil
}

func (s *IntegrationExecutionService) freezeRetryPolicy(tx *gorm.DB, definition model.InterfaceDefinition) (*int, []byte, int, error) {
	if definition.RetryPolicyID == nil {
		return nil, []byte(`{}`), 0, nil
	}
	policy, err := s.policies.FindByIdForUpdate(tx, *definition.RetryPolicyID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, 0, myerrors.ErrInterfaceRetryPolicyInvalid
	}
	if err != nil {
		return nil, nil, 0, myerrors.WrapDatabaseError(err)
	}
	if policy.Status != model.RetryPolicyStatusEnabled || !policy.State {
		return nil, nil, 0, myerrors.ErrInterfaceRetryPolicyInvalid
	}
	if err := validateRetryPolicyConfiguration(policy); err != nil {
		return nil, nil, 0, myerrors.ErrInterfaceRetryPolicyInvalid
	}
	snapshot, err := integration.BuildRetryPolicySnapshot(policy)
	if err != nil {
		return nil, nil, 0, myerrors.ErrRetryPolicyConfigurationInvalid
	}
	return definition.RetryPolicyID, snapshot, integration.RetryPolicySnapshotVersion, nil
}

func (s *IntegrationExecutionService) GetExecution(
	ctx context.Context,
	id int,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (response.IntegrationExecutionDetailRes, error) {
	value, err := s.executions.FindByIDWithPermission(ctx, id, table, permission)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.IntegrationExecutionDetailRes{}, myerrors.ErrIntegrationExecutionNotFound
	}
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, integrationExecutionReadError(err)
	}
	return response.NewIntegrationExecutionDetailRes(value), nil
}

func (s *IntegrationExecutionService) GetLog(
	ctx context.Context,
	id int,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (response.IntegrationLogDetailRes, error) {
	value, err := s.logs.FindByIDWithPermission(ctx, id, table, permission)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.IntegrationLogDetailRes{}, myerrors.ErrIntegrationExecutionNotFound
	}
	if err != nil {
		return response.IntegrationLogDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewIntegrationLogDetailRes(value), nil
}

func (s *IntegrationExecutionService) PageLogs(
	ctx context.Context,
	req request.IntegrationLogQueryReq,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (response.ListResult[response.IntegrationLogListRes], error) {
	if req.StartedFrom != nil && req.StartedTo != nil && req.StartedFrom.After(*req.StartedTo) {
		return response.ListResult[response.IntegrationLogListRes]{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
	}
	result, err := s.logs.GetIntegrationLogList(ctx, req, table, permission)
	if err != nil {
		return response.ListResult[response.IntegrationLogListRes]{}, integrationExecutionReadError(err)
	}
	items := make([]response.IntegrationLogListRes, 0, len(result.Data))
	for _, value := range result.Data {
		items = append(items, response.NewIntegrationLogListRes(value))
	}
	return response.ListResult[response.IntegrationLogListRes]{Data: items, Total: result.Total}, nil
}

func (s *IntegrationExecutionService) PageExecution(
	ctx context.Context,
	req request.IntegrationExecutionQueryReq,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (response.ListResult[response.IntegrationExecutionListRes], error) {
	if req.CreatedFrom != nil && req.CreatedTo != nil && req.CreatedFrom.After(*req.CreatedTo) {
		return response.ListResult[response.IntegrationExecutionListRes]{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
	}
	basic := req.ToBasic()
	result, err := s.executions.GetIntegrationExecutionList(ctx, &basic, table, permission)
	if err != nil {
		return response.ListResult[response.IntegrationExecutionListRes]{}, integrationExecutionReadError(err)
	}
	items := make([]response.IntegrationExecutionListRes, 0, len(result.Data))
	for _, value := range result.Data {
		items = append(items, response.NewIntegrationExecutionListRes(value))
	}
	return response.ListResult[response.IntegrationExecutionListRes]{Data: items, Total: result.Total}, nil
}

func (s *IntegrationExecutionService) CancelExecution(
	ctx context.Context,
	id int,
	revision int,
) (response.IntegrationExecutionDetailRes, error) {
	if id <= 0 || revision <= 0 {
		return response.IntegrationExecutionDetailRes{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
	}
	var value model.IntegrationExecution
	err := RunInTransaction(ctx, s.executions.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.executions.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrIntegrationExecutionNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Revision != revision {
			return myerrors.ErrIntegrationExecutionRevisionConflict
		}
		if current.Status != model.IntegrationExecutionStatusCreated &&
			current.Status != model.IntegrationExecutionStatusRetryWaiting {
			return myerrors.ErrIntegrationExecutionStatusInvalid
		}
		now := s.now()
		updates := map[string]any{
			"status":       model.IntegrationExecutionStatusCancelled,
			"cancelled_at": now,
			"completed_at": now,
			"next_run_at":  nil,
			"revision":     current.Revision + 1,
		}
		updated, err := s.executions.UpdateFieldsByRevision(tx, current.Id, revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updated {
			return myerrors.ErrIntegrationExecutionRevisionConflict
		}
		value, err = s.executions.FindByIdWithDB(tx, current.Id)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, integrationExecutionAuditCancel, value, current.Status, value.Status)
	})
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, err
	}
	return response.NewIntegrationExecutionDetailRes(value), nil
}

func integrationExecutionReadError(err error) error {
	if myerrors.CategoryOf(err) != response.ErrorCategorySystem {
		return err
	}
	return myerrors.WrapDatabaseError(err)
}

func (s *IntegrationExecutionService) writeAudit(
	ctx context.Context,
	tx *gorm.DB,
	action string,
	value model.IntegrationExecution,
	oldStatus string,
	newStatus string,
) error {
	changes := map[string]TransactionalAuditChange{
		"status":   {OldValue: oldStatus, NewValue: newStatus},
		"revision": {OldValue: value.Revision - 1, NewValue: value.Revision},
	}
	if action == integrationExecutionAuditCreate {
		changes["revision"] = TransactionalAuditChange{OldValue: 0, NewValue: value.Revision}
	}
	return s.audit.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{
		Action: action, ResourceType: integrationExecutionAuditResourceType,
		ResourceCode: value.ExecutionNo, ResourceId: strconv.Itoa(value.Id), Changes: changes,
	})
}

func (s *IntegrationExecutionService) loadExecutionConfiguration(
	tx *gorm.DB,
	req request.IntegrationExecutionCreateReq,
) (model.ExternalSystem, model.InterfaceDefinition, error) {
	system, err := s.systems.FindByIdForUpdate(tx, req.ExternalSystemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ExternalSystem{}, model.InterfaceDefinition{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
		}
		return model.ExternalSystem{}, model.InterfaceDefinition{}, myerrors.WrapDatabaseError(err)
	}
	definition, err := s.interfaces.FindByIdForUpdate(tx, req.InterfaceDefinitionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ExternalSystem{}, model.InterfaceDefinition{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
		}
		return model.ExternalSystem{}, model.InterfaceDefinition{}, myerrors.WrapDatabaseError(err)
	}
	if system.Status != model.ExternalSystemStatusEnabled ||
		definition.Status != model.InterfaceDefinitionStatusEnabled ||
		definition.ExternalSystemID != system.Id {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
	}
	if err := integration.ValidateInterfaceRuntimeContract(definition.TimeoutSeconds, definition.ResponseLimit); err != nil {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, myerrors.ErrIntegrationExecutionRuntimeIncompatible
	}
	return system, definition, nil
}

func (s *IntegrationExecutionService) resolveIdempotencyRace(
	ctx context.Context,
	req request.IntegrationExecutionCreateReq,
	interfaceVersion int,
	serverInputHash string,
) (response.IntegrationExecutionDetailRes, error) {
	value, err := s.executions.FindByIdempotency(
		s.executions.DBWithContext(ctx),
		req.InterfaceDefinitionID,
		interfaceVersion,
		req.IdempotencyScope,
		req.IdempotencyKey,
	)
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	if value.InputHash != serverInputHash {
		return response.IntegrationExecutionDetailRes{}, myerrors.ErrIntegrationExecutionIdempotencyConflict
	}
	return response.NewIntegrationExecutionDetailRes(value), nil
}

func normalizeIntegrationExecutionCreateReq(
	req request.IntegrationExecutionCreateReq,
) (request.IntegrationExecutionCreateReq, error) {
	req.TriggerSource = strings.TrimSpace(req.TriggerSource)
	req.IdempotencyScope = strings.ToLower(strings.TrimSpace(req.IdempotencyScope))
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	req.InputHash = strings.ToLower(strings.TrimSpace(req.InputHash))
	if req.ExternalSystemID <= 0 || req.InterfaceDefinitionID <= 0 ||
		!validIntegrationTriggerSource(req.TriggerSource) ||
		!integrationIdempotencyScopePattern.MatchString(req.IdempotencyScope) ||
		req.IdempotencyKey == "" || len(req.IdempotencyKey) > 128 ||
		req.InputHash != "" && !integrationInputHashPattern.MatchString(req.InputHash) {
		return request.IntegrationExecutionCreateReq{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
	}
	return req, nil
}

func validIntegrationTriggerSource(value string) bool {
	switch value {
	case model.IntegrationTriggerSourceManual,
		model.IntegrationTriggerSourceSystemEvent,
		model.IntegrationTriggerSourceScheduled:
		return true
	default:
		return false
	}
}

func isIntegrationExecutionIdempotencyDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == "uni_integration_execution_idempotency"
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") &&
		strings.Contains(message, "integration_execution.interface_definition_id") &&
		strings.Contains(message, "integration_execution.idempotency_key")
}
