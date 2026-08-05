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

type IntegrationExecutionService struct {
	executions repository.IntegrationExecutionRepository
	logs       repository.IntegrationLogRepository
	systems    repository.ExternalSystemRepository
	interfaces repository.InterfaceDefinitionRepository
	sf         *utils.Snowflake
	now        func() time.Time
}

func NewIntegrationExecutionService(
	executions repository.IntegrationExecutionRepository,
	logs repository.IntegrationLogRepository,
	systems repository.ExternalSystemRepository,
	interfaces repository.InterfaceDefinitionRepository,
	sf *utils.Snowflake,
) *IntegrationExecutionService {
	return &IntegrationExecutionService{
		executions: executions,
		logs:       logs,
		systems:    systems,
		interfaces: interfaces,
		sf:         sf,
		now:        model.Now,
	}
}

func (s *IntegrationExecutionService) Create(
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
		return nil
	})
	if errors.Is(err, errIntegrationIdempotencyRace) {
		return s.resolveIdempotencyRace(ctx, req, interfaceVersion)
	}
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, err
	}
	return response.NewIntegrationExecutionDetailRes(value, nil), nil
}

func (s *IntegrationExecutionService) Get(
	ctx context.Context,
	id int,
) (response.IntegrationExecutionDetailRes, error) {
	value, err := s.executions.WithContext(ctx).FindById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.IntegrationExecutionDetailRes{}, myerrors.ErrIntegrationExecutionNotFound
	}
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	logs, err := s.logs.ListByExecutionID(ctx, value.Id)
	if err != nil {
		return response.IntegrationExecutionDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewIntegrationExecutionDetailRes(value, logs), nil
}

func (s *IntegrationExecutionService) Page(
	ctx context.Context,
	req request.IntegrationExecutionQueryReq,
	table model.SysTable,
) (response.ListResult[response.IntegrationExecutionListRes], error) {
	basic := req.ToBasic()
	result, err := s.executions.GetIntegrationExecutionList(ctx, &basic, table)
	if err != nil {
		return response.ListResult[response.IntegrationExecutionListRes]{}, myerrors.WrapDatabaseError(err)
	}
	items := make([]response.IntegrationExecutionListRes, 0, len(result.Data))
	for _, value := range result.Data {
		items = append(items, response.NewIntegrationExecutionListRes(value))
	}
	return response.ListResult[response.IntegrationExecutionListRes]{Data: items, Total: result.Total}, nil
}

func (s *IntegrationExecutionService) Cancel(
	ctx context.Context,
	id int,
	revision int,
) (response.IntegrationExecutionDetailRes, error) {
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
			"revision":     current.Revision + 1,
		}
		updated, err := s.executions.UpdateFieldsByRevision(tx, current.Id, revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updated {
			return myerrors.ErrIntegrationExecutionRevisionConflict
		}
		current.Status = model.IntegrationExecutionStatusCancelled
		current.CancelledAt = &now
		current.CompletedAt = &now
		current.Revision++
		value = current
		return nil
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
