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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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
	integrationExecutionAuditStart        = "integration.execution.start"
	integrationExecutionAuditComplete     = "integration.execution.complete"
	integrationExecutionAuditFail         = "integration.execution.fail"
	integrationExecutionAuditCancel       = "integration.execution.cancel"
)

type IntegrationExecutionService struct {
	executions repository.IntegrationExecutionRepository
	logs       repository.IntegrationLogRepository
	systems    repository.ExternalSystemRepository
	interfaces repository.InterfaceDefinitionRepository
	sf         *utils.Snowflake
	audit      StandardContextAuditWriter
	now        func() time.Time
}

func NewIntegrationExecutionService(
	executions repository.IntegrationExecutionRepository,
	logs repository.IntegrationLogRepository,
	systems repository.ExternalSystemRepository,
	interfaces repository.InterfaceDefinitionRepository,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
) *IntegrationExecutionService {
	return &IntegrationExecutionService{
		executions: executions,
		logs:       logs,
		systems:    systems,
		interfaces: interfaces,
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
	err = RunInTransaction(ctx, s.executions.DBWithContext(ctx), func(tx *gorm.DB) error {
		system, definition, err := s.loadExecutionConfiguration(tx, req)
		if err != nil {
			return err
		}
		interfaceVersion = definition.Version

		existing, err := s.executions.FindByIdempotency(
			tx,
			definition.Id,
			definition.Version,
			req.IdempotencyScope,
			req.IdempotencyKey,
		)
		if err == nil {
			if existing.InputHash != req.InputHash {
				return myerrors.ErrIntegrationExecutionIdempotencyConflict
			}
			value = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.WrapDatabaseError(err)
		}

		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return myerrors.WrapSystemError(err)
		}
		value = model.IntegrationExecution{
			Basic:                 model.Basic{Id: int(id), State: true},
			ExecutionNo:           fmt.Sprintf("INT-%d", id),
			ExternalSystemID:      system.Id,
			ExternalSystemCode:    system.SystemCode,
			ExternalSystemName:    system.Name,
			InterfaceDefinitionID: definition.Id,
			InterfaceCode:         definition.InterfaceCode,
			InterfaceName:         definition.Name,
			InterfaceVersion:      definition.Version,
			TriggerSource:         req.TriggerSource,
			Status:                model.IntegrationExecutionStatusCreated,
			IdempotencyScope:      req.IdempotencyScope,
			IdempotencyKey:        req.IdempotencyKey,
			InputHash:             req.InputHash,
			Revision:              1,
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
		return s.resolveIdempotencyRace(ctx, req, interfaceVersion)
	}
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, err
	}
	return response.NewIntegrationExecutionDetailRes(value, nil), nil
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
	logs, err := s.logs.ListByExecutionID(ctx, value.Id)
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewIntegrationExecutionDetailRes(value, logs), nil
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

func (s *IntegrationExecutionService) StartExecution(
	ctx context.Context,
	id int,
	revision int,
) (response.IntegrationExecutionDetailRes, error) {
	now := s.now()
	return s.transitionExecution(ctx, id, revision, model.IntegrationExecutionStatusRunning,
		[]string{model.IntegrationExecutionStatusCreated, model.IntegrationExecutionStatusRetryWaiting},
		map[string]any{"started_at": now, "next_run_at": nil}, integrationExecutionAuditStart)
}

func (s *IntegrationExecutionService) CompleteExecution(
	ctx context.Context,
	id int,
	req request.IntegrationExecutionCompleteReq,
) (response.IntegrationExecutionDetailRes, error) {
	req.ResultHash = strings.ToLower(strings.TrimSpace(req.ResultHash))
	req.ResultSummary = strings.TrimSpace(req.ResultSummary)
	if req.ResultSizeBytes < 0 || !validIntegrationHTTPStatus(req.ResultHTTPStatus) ||
		(req.ResultHash != "" && !integrationInputHashPattern.MatchString(req.ResultHash)) || len(req.ResultSummary) > 1024 {
		return response.IntegrationExecutionDetailRes{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
	}
	now := s.now()
	return s.transitionExecution(ctx, id, req.Revision, model.IntegrationExecutionStatusSucceeded,
		[]string{model.IntegrationExecutionStatusRunning}, map[string]any{
			"result_http_status": req.ResultHTTPStatus,
			"result_size_bytes":  req.ResultSizeBytes,
			"result_hash":        req.ResultHash,
			"result_summary":     req.ResultSummary,
			"error_category":     "",
			"completed_at":       now,
			"next_run_at":        nil,
		}, integrationExecutionAuditComplete)
}

func (s *IntegrationExecutionService) FailExecution(
	ctx context.Context,
	id int,
	req request.IntegrationExecutionFailReq,
) (response.IntegrationExecutionDetailRes, error) {
	req.TargetStatus = strings.TrimSpace(req.TargetStatus)
	req.ErrorCategory = strings.TrimSpace(req.ErrorCategory)
	req.ResultSummary = strings.TrimSpace(req.ResultSummary)
	if !validIntegrationFailureTarget(req.TargetStatus) || !validIntegrationErrorCategory(req.ErrorCategory) || len(req.ResultSummary) > 1024 {
		return response.IntegrationExecutionDetailRes{}, myerrors.ErrIntegrationExecutionConfigurationInvalid
	}
	sources := []string{model.IntegrationExecutionStatusRunning}
	updates := map[string]any{
		"error_category": req.ErrorCategory,
		"result_summary": req.ResultSummary,
	}
	if req.TargetStatus == model.IntegrationExecutionStatusFailed {
		sources = append(sources, model.IntegrationExecutionStatusRetryWaiting)
		updates["completed_at"] = s.now()
		updates["next_run_at"] = nil
	} else {
		updates["completed_at"] = nil
	}
	return s.transitionExecution(ctx, id, req.Revision, req.TargetStatus, sources, updates, integrationExecutionAuditFail)
}

func (s *IntegrationExecutionService) CancelExecution(
	ctx context.Context,
	id int,
	revision int,
) (response.IntegrationExecutionDetailRes, error) {
	now := s.now()
	return s.transitionExecution(ctx, id, revision, model.IntegrationExecutionStatusCancelled,
		[]string{model.IntegrationExecutionStatusCreated, model.IntegrationExecutionStatusRetryWaiting},
		map[string]any{"cancelled_at": now, "completed_at": now, "next_run_at": nil}, integrationExecutionAuditCancel)
}

func (s *IntegrationExecutionService) transitionExecution(
	ctx context.Context,
	id int,
	revision int,
	target string,
	commandSources []string,
	updates map[string]any,
	action string,
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
		if !containsIntegrationExecutionStatus(commandSources, current.Status) ||
			!allowedIntegrationExecutionTransition(current.Status, target) {
			return myerrors.ErrIntegrationExecutionStatusInvalid
		}
		if target == model.IntegrationExecutionStatusRunning && current.StartedAt != nil {
			delete(updates, "started_at")
		}

		if updates == nil {
			updates = make(map[string]any)
		}
		updates["status"] = target
		updates["revision"] = current.Revision + 1
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
		return s.writeAudit(ctx, tx, action, value, current.Status, value.Status)
	})
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, err
	}
	logs, err := s.logs.ListByExecutionID(ctx, value.Id)
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewIntegrationExecutionDetailRes(value, logs), nil
}

func allowedIntegrationExecutionTransition(from string, to string) bool {
	switch from {
	case model.IntegrationExecutionStatusCreated:
		return to == model.IntegrationExecutionStatusRunning || to == model.IntegrationExecutionStatusCancelled
	case model.IntegrationExecutionStatusRunning:
		return to == model.IntegrationExecutionStatusSucceeded ||
			to == model.IntegrationExecutionStatusFailed ||
			to == model.IntegrationExecutionStatusRetryWaiting ||
			to == model.IntegrationExecutionStatusCancelled
	case model.IntegrationExecutionStatusRetryWaiting:
		return to == model.IntegrationExecutionStatusRunning ||
			to == model.IntegrationExecutionStatusFailed ||
			to == model.IntegrationExecutionStatusCancelled
	default:
		return false
	}
}

func containsIntegrationExecutionStatus(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validIntegrationFailureTarget(value string) bool {
	return value == model.IntegrationExecutionStatusFailed || value == model.IntegrationExecutionStatusRetryWaiting
}

func validIntegrationHTTPStatus(value *int) bool {
	return value == nil || (*value >= 100 && *value <= 599)
}

func integrationExecutionReadError(err error) error {
	if myerrors.CategoryOf(err) != response.ErrorCategorySystem {
		return err
	}
	return myerrors.WrapDatabaseError(err)
}

func validIntegrationErrorCategory(value string) bool {
	switch value {
	case model.IntegrationErrorCategoryConfiguration,
		model.IntegrationErrorCategoryCredential,
		model.IntegrationErrorCategoryNetwork,
		model.IntegrationErrorCategoryTimeout,
		model.IntegrationErrorCategoryRemote,
		model.IntegrationErrorCategoryResponse,
		model.IntegrationErrorCategoryBusiness,
		model.IntegrationErrorCategoryConcurrency,
		model.IntegrationErrorCategorySystem:
		return true
	default:
		return false
	}
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
	return system, definition, nil
}

func (s *IntegrationExecutionService) resolveIdempotencyRace(
	ctx context.Context,
	req request.IntegrationExecutionCreateReq,
	interfaceVersion int,
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
	if value.InputHash != req.InputHash {
		return response.IntegrationExecutionDetailRes{}, myerrors.ErrIntegrationExecutionIdempotencyConflict
	}
	return response.NewIntegrationExecutionDetailRes(value, nil), nil
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
		!integrationInputHashPattern.MatchString(req.InputHash) {
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
