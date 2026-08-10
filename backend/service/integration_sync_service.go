package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/integration"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/robfig/cron/v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	syncTaskAuditResourceType = "integration_sync_task"
	syncTaskAuditCreate       = "integration.sync_task.create"
	syncTaskAuditUpdate       = "integration.sync_task.update"
	syncTaskAuditVersion      = "integration.sync_task.create_version"
	syncTaskAuditEnable       = "integration.sync_task.enable"
	syncTaskAuditDisable      = "integration.sync_task.disable"

	syncTaskMaxDurationSeconds = 604800
)

var syncTaskCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

type SyncTaskService struct {
	tasks      repository.IntegrationSyncTaskRepository
	batches    repository.IntegrationSyncBatchRepository
	systems    repository.ExternalSystemRepository
	interfaces repository.InterfaceDefinitionRepository
	policies   repository.RetryPolicyRepository
	registry   integration.SyncConsumerRegistry
	sf         *utils.Snowflake
	audit      StandardContextAuditWriter
	lease      time.Duration
}

func NewSyncTaskService(
	tasks repository.IntegrationSyncTaskRepository,
	batches repository.IntegrationSyncBatchRepository,
	systems repository.ExternalSystemRepository,
	interfaces repository.InterfaceDefinitionRepository,
	policies repository.RetryPolicyRepository,
	registry integration.SyncConsumerRegistry,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
	cfg *config.Server,
) *SyncTaskService {
	lease := integration.IntegrationDefaultLeaseDuration
	if cfg != nil && cfg.Integration.Worker.LeaseDuration > 0 {
		lease = time.Duration(cfg.Integration.Worker.LeaseDuration) * time.Second
	}
	return &SyncTaskService{tasks: tasks, batches: batches, systems: systems, interfaces: interfaces, policies: policies, registry: registry, sf: sf, audit: audit, lease: lease}
}

func (s *SyncTaskService) CreateSyncTask(ctx context.Context, req request.SyncTaskCreateReq) (response.SyncTaskDetailRes, error) {
	value, err := s.newSyncTask(ctx, req)
	if err != nil {
		return response.SyncTaskDetailRes{}, err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return response.SyncTaskDetailRes{}, myerrors.WrapSystemError(err)
	}
	value.Basic = model.Basic{Id: int(id), State: false}
	err = RunInTransaction(ctx, s.tasks.DBWithContext(ctx), func(tx *gorm.DB) error {
		next, err := s.tasks.NextVersion(tx, value.TaskCode)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if next != 1 {
			return myerrors.ErrSyncTaskCodeDuplicate
		}
		if err := s.tasks.Create(tx, &value); err != nil {
			if isSyncDuplicate(err) {
				return myerrors.ErrSyncTaskCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, syncTaskAuditCreate, value, nil)
	})
	if err != nil {
		return response.SyncTaskDetailRes{}, err
	}
	return s.taskDetail(ctx, value)
}

func (s *SyncTaskService) PageSyncTask(ctx context.Context, req request.SyncTaskQueryReq, table model.SysTable) (response.ListResult[response.SyncTaskListRes], error) {
	basic := req.ToBasic()
	var values []model.IntegrationSyncTask
	total, err := s.tasks.WithContext(ctx).PaginateAndCountAsync(&basic, &values, table)
	if err != nil {
		return response.ListResult[response.SyncTaskListRes]{}, myerrors.WrapDatabaseError(err)
	}
	items := make([]response.SyncTaskListRes, 0, len(values))
	for _, value := range values {
		system, definition, err := s.taskReferences(ctx, value)
		if err != nil {
			return response.ListResult[response.SyncTaskListRes]{}, err
		}
		items = append(items, response.NewSyncTaskListRes(value, system, definition, syncInputPlanSummary(value.InputPlan)))
	}
	return response.ListResult[response.SyncTaskListRes]{Data: items, Total: int(total)}, nil
}

func (s *SyncTaskService) GetSyncTask(ctx context.Context, id int) (response.SyncTaskDetailRes, error) {
	value, err := s.tasks.WithContext(ctx).FindById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.SyncTaskDetailRes{}, myerrors.ErrSyncTaskNotFound
	}
	if err != nil {
		return response.SyncTaskDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return s.taskDetail(ctx, value)
}

func (s *SyncTaskService) GetSyncTaskForEdit(ctx context.Context, id int) (response.SyncTaskEditRes, error) {
	value, err := s.tasks.WithContext(ctx).FindById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.SyncTaskEditRes{}, myerrors.ErrSyncTaskNotFound
	}
	if err != nil {
		return response.SyncTaskEditRes{}, myerrors.WrapDatabaseError(err)
	}
	if value.Status != model.IntegrationSyncTaskStatusDraft {
		return response.SyncTaskEditRes{}, myerrors.ErrSyncTaskFieldImmutable
	}
	detail, err := s.taskDetail(ctx, value)
	if err != nil {
		return response.SyncTaskEditRes{}, err
	}
	plan, err := integration.DecodeSyncExecutionInputPlan(value.InputPlan)
	if err != nil {
		return response.SyncTaskEditRes{}, err
	}
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return response.SyncTaskEditRes{}, myerrors.ErrSyncInputPlanInvalid
	}
	return response.SyncTaskEditRes{SyncTaskDetailRes: detail, InputPlan: planJSON}, nil
}

func (s *SyncTaskService) UpdateDraftSyncTask(ctx context.Context, id int, req request.SyncTaskUpdateReq) (response.SyncTaskDetailRes, error) {
	var updated model.IntegrationSyncTask
	err := RunInTransaction(ctx, s.tasks.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.tasks.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrSyncTaskNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status != model.IntegrationSyncTaskStatusDraft {
			return myerrors.ErrSyncTaskFieldImmutable
		}
		if current.Revision != req.Revision {
			return myerrors.ErrSyncTaskRevisionConflict
		}
		next := applySyncTaskUpdates(current, req)
		if err := s.normalizeAndValidateTask(ctx, tx, &next, false); err != nil {
			return err
		}
		updates := syncTaskTechnicalUpdates(next)
		updates["revision"] = current.Revision + 1
		ok, err := s.tasks.UpdateFieldsByRevision(tx, id, req.Revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrSyncTaskRevisionConflict
		}
		next.Revision = current.Revision + 1
		updated = next
		return s.writeAudit(ctx, tx, syncTaskAuditUpdate, updated, &current)
	})
	if err != nil {
		return response.SyncTaskDetailRes{}, err
	}
	return s.taskDetail(ctx, updated)
}

func (s *SyncTaskService) CreateSyncTaskVersion(ctx context.Context, id, revision int) (response.SyncTaskDetailRes, error) {
	var created model.IntegrationSyncTask
	err := RunInTransaction(ctx, s.tasks.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.tasks.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrSyncTaskNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Revision != revision {
			return myerrors.ErrSyncTaskRevisionConflict
		}
		if current.Status == model.IntegrationSyncTaskStatusDraft {
			return myerrors.ErrSyncTaskStatusInvalid
		}
		nextVersion, err := s.tasks.NextVersion(tx, current.TaskCode)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		newID, err := s.sf.GenerateUniqueID()
		if err != nil {
			return myerrors.WrapSystemError(err)
		}
		created = current
		created.Basic = model.Basic{Id: int(newID), State: false}
		created.Version = nextVersion
		created.Status = model.IntegrationSyncTaskStatusDraft
		created.CheckpointAt = nil
		created.NextScheduledAt = nil
		created.LastScheduledAt = nil
		created.Revision = 1
		if err := s.tasks.Create(tx, &created); err != nil {
			if isSyncDuplicate(err) {
				return myerrors.ErrSyncTaskCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, syncTaskAuditVersion, created, nil)
	})
	if err != nil {
		return response.SyncTaskDetailRes{}, err
	}
	return s.taskDetail(ctx, created)
}

func (s *SyncTaskService) EnableSyncTask(ctx context.Context, id, revision int) (response.SyncTaskDetailRes, error) {
	var updated model.IntegrationSyncTask
	err := RunInTransaction(ctx, s.tasks.DBWithContext(ctx), func(tx *gorm.DB) error {
		seed, err := s.tasks.FindByIdWithDB(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrSyncTaskNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		versions, err := s.tasks.FindVersionsByCodeForUpdate(tx, seed.TaskCode)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		current, ok := findSyncTaskVersion(versions, id)
		if !ok {
			return myerrors.ErrSyncTaskNotFound
		}
		if current.Revision != revision {
			return myerrors.ErrSyncTaskRevisionConflict
		}
		if current.Status == model.IntegrationSyncTaskStatusEnabled {
			updated = current
			return nil
		}
		if current.Status != model.IntegrationSyncTaskStatusDraft && current.Status != model.IntegrationSyncTaskStatusDisabled {
			return myerrors.ErrSyncTaskStatusInvalid
		}
		active, err := s.batches.CountActiveByTaskCode(tx, current.TaskCode)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if active > 0 {
			return myerrors.ErrSyncTaskActiveBatch
		}
		if current.CheckpointMode == model.IntegrationSyncCheckpointTimestamp {
			if inherited := latestSyncCheckpoint(versions, id); inherited != nil {
				current.CheckpointAt = inherited
			}
			if current.CheckpointAt == nil {
				current.CheckpointAt = cloneTime(current.InitialCheckpointAt)
			}
			if current.CheckpointAt == nil {
				return myerrors.ErrSyncCheckpointInvalid
			}
		} else {
			current.InitialCheckpointAt, current.CheckpointAt = nil, nil
		}
		if err := s.normalizeAndValidateTask(ctx, tx, &current, true); err != nil {
			return err
		}
		now, err := syncDatabaseNow(tx)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.ScheduleType == model.IntegrationSyncScheduleCron {
			current.NextScheduledAt, err = nextSyncSchedule(current.CronExpression, current.Timezone, now)
			if err != nil {
				return err
			}
		} else {
			current.NextScheduledAt = nil
		}
		for _, version := range versions {
			if version.Id == current.Id || version.Status != model.IntegrationSyncTaskStatusEnabled {
				continue
			}
			ok, err := s.tasks.UpdateFieldsByRevision(tx, version.Id, version.Revision, map[string]any{"status": model.IntegrationSyncTaskStatusDisabled, "state": false, "next_scheduled_at": nil, "revision": version.Revision + 1})
			if err != nil {
				return myerrors.WrapDatabaseError(err)
			}
			if !ok {
				return myerrors.ErrSyncTaskRevisionConflict
			}
		}
		updates := map[string]any{"status": model.IntegrationSyncTaskStatusEnabled, "state": true, "checkpoint_at": current.CheckpointAt, "initial_checkpoint_at": current.InitialCheckpointAt, "next_scheduled_at": current.NextScheduledAt, "revision": current.Revision + 1}
		ok, err = s.tasks.UpdateFieldsByRevision(tx, current.Id, current.Revision, updates)
		if err != nil {
			if isSyncDuplicate(err) {
				return myerrors.ErrSyncTaskEnabledConflict
			}
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrSyncTaskRevisionConflict
		}
		previous := current
		current.Status, current.State, current.Revision = model.IntegrationSyncTaskStatusEnabled, true, current.Revision+1
		updated = current
		return s.writeAudit(ctx, tx, syncTaskAuditEnable, updated, &previous)
	})
	if err != nil {
		return response.SyncTaskDetailRes{}, err
	}
	return s.taskDetail(ctx, updated)
}

func (s *SyncTaskService) DisableSyncTask(ctx context.Context, id, revision int) (response.SyncTaskDetailRes, error) {
	var updated model.IntegrationSyncTask
	err := RunInTransaction(ctx, s.tasks.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.tasks.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrSyncTaskNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Revision != revision {
			return myerrors.ErrSyncTaskRevisionConflict
		}
		if current.Status == model.IntegrationSyncTaskStatusDisabled {
			updated = current
			return nil
		}
		if current.Status != model.IntegrationSyncTaskStatusEnabled {
			return myerrors.ErrSyncTaskStatusInvalid
		}
		active, err := s.batches.CountActiveByTaskCode(tx, current.TaskCode)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if active > 0 {
			return myerrors.ErrSyncTaskActiveBatch
		}
		updates := map[string]any{"status": model.IntegrationSyncTaskStatusDisabled, "state": false, "next_scheduled_at": nil, "revision": current.Revision + 1}
		ok, err := s.tasks.UpdateFieldsByRevision(tx, id, revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrSyncTaskRevisionConflict
		}
		previous := current
		current.Status, current.State, current.NextScheduledAt, current.Revision = model.IntegrationSyncTaskStatusDisabled, false, nil, current.Revision+1
		updated = current
		return s.writeAudit(ctx, tx, syncTaskAuditDisable, updated, &previous)
	})
	if err != nil {
		return response.SyncTaskDetailRes{}, err
	}
	return s.taskDetail(ctx, updated)
}

func (s *SyncTaskService) ListSyncConsumers(context.Context) []response.SyncConsumerMetadataRes {
	values := s.registry.ListMetadata()
	result := make([]response.SyncConsumerMetadataRes, 0, len(values))
	for _, value := range values {
		result = append(result, response.SyncConsumerMetadataRes{
			Code: value.Code, Version: value.Version, Name: value.Name,
			ContentTypes: append([]string(nil), value.ContentTypes...), MaxResponseBytes: value.MaxResponseBytes,
			MaxDurationMs: value.MaxDuration.Milliseconds(), CheckpointModes: append([]string(nil), value.CheckpointModes...),
		})
	}
	return result
}

type SyncBatchService struct {
	repository repository.IntegrationSyncBatchRepository
}

func NewSyncBatchService(repository repository.IntegrationSyncBatchRepository) *SyncBatchService {
	return &SyncBatchService{repository: repository}
}

func (s *SyncBatchService) PageSyncBatch(ctx context.Context, req request.SyncBatchQueryReq, table model.SysTable) (response.ListResult[response.SyncBatchListRes], error) {
	basic := req.ToBasic()
	var values []model.IntegrationSyncBatch
	total, err := s.repository.WithContext(ctx).PaginateAndCountAsync(&basic, &values, table)
	if err != nil {
		return response.ListResult[response.SyncBatchListRes]{}, myerrors.WrapDatabaseError(err)
	}
	items := make([]response.SyncBatchListRes, 0, len(values))
	for _, value := range values {
		items = append(items, response.NewSyncBatchListRes(value))
	}
	return response.ListResult[response.SyncBatchListRes]{Data: items, Total: int(total)}, nil
}

func (s *SyncBatchService) GetSyncBatch(ctx context.Context, id int) (response.SyncBatchDetailRes, error) {
	value, err := s.repository.WithContext(ctx).FindById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.SyncBatchDetailRes{}, myerrors.ErrSyncBatchNotFound
	}
	if err != nil {
		return response.SyncBatchDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewSyncBatchDetailRes(value), nil
}

func (s *SyncTaskService) newSyncTask(ctx context.Context, req request.SyncTaskCreateReq) (model.IntegrationSyncTask, error) {
	if !syncTaskCodePattern.MatchString(strings.TrimSpace(req.TaskCode)) {
		return model.IntegrationSyncTask{}, myerrors.ErrSyncTaskCodeInvalid
	}
	plan, err := json.Marshal(req.InputPlan)
	if err != nil {
		return model.IntegrationSyncTask{}, myerrors.ErrSyncInputPlanInvalid
	}
	value := model.IntegrationSyncTask{TaskCode: strings.TrimSpace(req.TaskCode), TaskName: strings.TrimSpace(req.TaskName), Version: 1, Description: strings.TrimSpace(req.Description), Status: model.IntegrationSyncTaskStatusDraft,
		ExternalSystemID: req.ExternalSystemID, InterfaceDefinitionID: req.InterfaceDefinitionID, ConsumerCode: strings.TrimSpace(req.ConsumerCode), ConsumerVersion: req.ConsumerVersion,
		ScheduleType: req.ScheduleType, CronExpression: strings.TrimSpace(req.CronExpression), Timezone: strings.TrimSpace(req.Timezone), CheckpointMode: req.CheckpointMode,
		InitialCheckpointAt: cloneTime(req.InitialCheckpointAt), LookbackSeconds: req.LookbackSeconds, WindowSliceSeconds: req.WindowSliceSeconds, InputPlan: datatypes.JSON(plan), Revision: 1}
	if err := s.normalizeAndValidateTask(ctx, nil, &value, false); err != nil {
		return model.IntegrationSyncTask{}, err
	}
	return value, nil
}

func (s *SyncTaskService) normalizeAndValidateTask(ctx context.Context, tx *gorm.DB, value *model.IntegrationSyncTask, enabling bool) error {
	value.TaskName, value.Description, value.ConsumerCode = strings.TrimSpace(value.TaskName), strings.TrimSpace(value.Description), strings.TrimSpace(value.ConsumerCode)
	value.ScheduleType, value.CronExpression, value.Timezone = strings.ToLower(strings.TrimSpace(value.ScheduleType)), strings.TrimSpace(value.CronExpression), strings.TrimSpace(value.Timezone)
	value.CheckpointMode = strings.ToLower(strings.TrimSpace(value.CheckpointMode))
	if value.Timezone == "" {
		value.Timezone = "UTC"
	}
	if value.TaskName == "" || !syncTaskCodePattern.MatchString(value.TaskCode) || value.Version < 1 || value.ConsumerCode == "" || value.ConsumerVersion < 1 || value.LookbackSeconds < 0 || value.LookbackSeconds > syncTaskMaxDurationSeconds {
		return myerrors.ErrSyncTaskConfigurationInvalid
	}
	if _, err := time.LoadLocation(value.Timezone); err != nil {
		return myerrors.ErrSyncTimezoneInvalid
	}
	if value.ScheduleType == model.IntegrationSyncScheduleNone {
		value.CronExpression, value.NextScheduledAt = "", nil
	} else if value.ScheduleType == model.IntegrationSyncScheduleCron {
		if _, err := parseSyncCron(value.CronExpression); err != nil {
			return err
		}
	} else {
		return myerrors.ErrSyncScheduleInvalid
	}
	if value.CheckpointMode == model.IntegrationSyncCheckpointNone {
		value.InitialCheckpointAt, value.CheckpointAt, value.LookbackSeconds, value.WindowSliceSeconds = nil, nil, 0, 0
	} else if value.CheckpointMode == model.IntegrationSyncCheckpointTimestamp {
		if value.WindowSliceSeconds < 60 || value.WindowSliceSeconds > syncTaskMaxDurationSeconds || (!enabling && value.InitialCheckpointAt == nil) || (enabling && value.InitialCheckpointAt == nil && value.CheckpointAt == nil) {
			return myerrors.ErrSyncCheckpointInvalid
		}
	} else {
		return myerrors.ErrSyncCheckpointInvalid
	}
	system, err := s.findSyncSystem(ctx, tx, value.ExternalSystemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrSyncInterfaceInvalid
		}
		return myerrors.WrapDatabaseError(err)
	}
	if system.Status != model.ExternalSystemStatusEnabled || !system.State {
		return myerrors.ErrSyncInterfaceInvalid
	}
	definition, err := s.findSyncInterface(ctx, tx, value.InterfaceDefinitionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrSyncInterfaceInvalid
		}
		return myerrors.WrapDatabaseError(err)
	}
	if definition.Status != model.InterfaceDefinitionStatusEnabled || !definition.State || definition.ExternalSystemID != value.ExternalSystemID {
		return myerrors.ErrSyncInterfaceInvalid
	}
	if integration.ValidateInterfaceRuntimeContract(definition.TimeoutSeconds, definition.ResponseLimit) != nil {
		return myerrors.ErrSyncInterfaceInvalid
	}
	if definition.RetryPolicyID != nil {
		var policy model.RetryPolicy
		if tx != nil {
			policy, err = s.policies.FindByIdWithDB(tx, *definition.RetryPolicyID)
		} else {
			policy, err = s.policies.WithContext(ctx).FindById(*definition.RetryPolicyID)
		}
		if err != nil || policy.Status != model.RetryPolicyStatusEnabled || !policy.State || validateRetryPolicyConfiguration(policy) != nil {
			return myerrors.ErrSyncInterfaceInvalid
		}
	}
	normalized, _, err := integration.NormalizeSyncExecutionInputPlan(value.InputPlan, definition.InputContract, definition.HTTPMethod, definition.RelativePath, definition.Version, value.CheckpointMode)
	if err != nil {
		return err
	}
	value.InputPlan = datatypes.JSON(normalized)
	_, err = s.registry.ValidateReference(integration.SyncConsumerReference{Code: value.ConsumerCode, Version: value.ConsumerVersion, ContentType: "application/json", ResponseLimit: definition.ResponseLimit, CheckpointMode: value.CheckpointMode,
		RequestTimeout: time.Duration(definition.TimeoutSeconds) * time.Second, LeaseDuration: s.lease})
	return err
}

func (s *SyncTaskService) findSyncSystem(ctx context.Context, tx *gorm.DB, id int) (model.ExternalSystem, error) {
	if tx != nil {
		return s.systems.FindByIdWithDB(tx, id)
	}
	return s.systems.WithContext(ctx).FindById(id)
}

func (s *SyncTaskService) findSyncInterface(ctx context.Context, tx *gorm.DB, id int) (model.InterfaceDefinition, error) {
	if tx != nil {
		return s.interfaces.FindByIdWithDB(tx, id)
	}
	return s.interfaces.WithContext(ctx).FindById(id)
}

func applySyncTaskUpdates(current model.IntegrationSyncTask, req request.SyncTaskUpdateReq) model.IntegrationSyncTask {
	value := current
	if req.TaskName != nil {
		value.TaskName = *req.TaskName
	}
	if req.Description != nil {
		value.Description = *req.Description
	}
	if req.ExternalSystemID != nil {
		value.ExternalSystemID = *req.ExternalSystemID
	}
	if req.InterfaceDefinitionID != nil {
		value.InterfaceDefinitionID = *req.InterfaceDefinitionID
	}
	if req.ConsumerCode != nil {
		value.ConsumerCode = *req.ConsumerCode
	}
	if req.ConsumerVersion != nil {
		value.ConsumerVersion = *req.ConsumerVersion
	}
	if req.ScheduleType != nil {
		value.ScheduleType = *req.ScheduleType
	}
	if req.CronExpression != nil {
		value.CronExpression = *req.CronExpression
	}
	if req.Timezone != nil {
		value.Timezone = *req.Timezone
	}
	if req.CheckpointMode != nil {
		value.CheckpointMode = *req.CheckpointMode
	}
	if req.InitialCheckpointAt != nil {
		value.InitialCheckpointAt = cloneTime(req.InitialCheckpointAt)
	}
	if req.ClearInitialCheckpoint {
		value.InitialCheckpointAt = nil
	}
	if req.LookbackSeconds != nil {
		value.LookbackSeconds = *req.LookbackSeconds
	}
	if req.WindowSliceSeconds != nil {
		value.WindowSliceSeconds = *req.WindowSliceSeconds
	}
	if req.InputPlan != nil {
		encoded, _ := json.Marshal(req.InputPlan)
		value.InputPlan = datatypes.JSON(encoded)
	}
	return value
}

func syncTaskTechnicalUpdates(value model.IntegrationSyncTask) map[string]any {
	return map[string]any{"task_name": value.TaskName, "description": value.Description, "external_system_id": value.ExternalSystemID, "interface_definition_id": value.InterfaceDefinitionID,
		"consumer_code": value.ConsumerCode, "consumer_version": value.ConsumerVersion, "schedule_type": value.ScheduleType, "cron_expression": value.CronExpression, "timezone": value.Timezone,
		"checkpoint_mode": value.CheckpointMode, "initial_checkpoint_at": value.InitialCheckpointAt, "checkpoint_at": value.CheckpointAt, "lookback_seconds": value.LookbackSeconds,
		"window_slice_seconds": value.WindowSliceSeconds, "input_plan": value.InputPlan}
}

func (s *SyncTaskService) taskReferences(ctx context.Context, value model.IntegrationSyncTask) (model.ExternalSystem, model.InterfaceDefinition, error) {
	system, err := s.systems.WithContext(ctx).FindById(value.ExternalSystemID)
	if err != nil {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, myerrors.WrapDatabaseError(err)
	}
	definition, err := s.interfaces.WithContext(ctx).FindById(value.InterfaceDefinitionID)
	if err != nil {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, myerrors.WrapDatabaseError(err)
	}
	return system, definition, nil
}

func (s *SyncTaskService) taskDetail(ctx context.Context, value model.IntegrationSyncTask) (response.SyncTaskDetailRes, error) {
	system, definition, err := s.taskReferences(ctx, value)
	if err != nil {
		return response.SyncTaskDetailRes{}, err
	}
	return response.NewSyncTaskDetailRes(value, system, definition, syncInputPlanSummary(value.InputPlan)), nil
}

func syncInputPlanSummary(raw []byte) response.SyncInputPlanSummaryRes {
	value := integration.SummarizeSyncExecutionInputPlan(raw)
	return response.SyncInputPlanSummaryRes{
		Version: value.Version, StaticParameterCount: value.StaticParameterCount, HasWindowBindings: value.HasWindowBindings,
	}
}

func parseSyncCron(expression string) (cron.Schedule, error) {
	if len(strings.Fields(expression)) != 5 {
		return nil, myerrors.ErrSyncScheduleInvalid
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expression)
	if err != nil {
		return nil, myerrors.ErrSyncScheduleInvalid
	}
	return schedule, nil
}

func nextSyncSchedule(expression, timezone string, from time.Time) (*time.Time, error) {
	schedule, err := parseSyncCron(expression)
	if err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, myerrors.ErrSyncTimezoneInvalid
	}
	next := schedule.Next(from.In(location)).UTC()
	return &next, nil
}

func syncDatabaseNow(tx *gorm.DB) (time.Time, error) {
	if tx.Dialector.Name() != "postgres" {
		var epoch int64
		if err := tx.Raw("SELECT unixepoch()").Scan(&epoch).Error; err != nil {
			return time.Time{}, err
		}
		return time.Unix(epoch, 0).UTC(), nil
	}
	var now time.Time
	err := tx.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&now).Error
	return now.UTC(), err
}

func findSyncTaskVersion(values []model.IntegrationSyncTask, id int) (model.IntegrationSyncTask, bool) {
	for _, value := range values {
		if value.Id == id {
			return value, true
		}
	}
	return model.IntegrationSyncTask{}, false
}

func latestSyncCheckpoint(values []model.IntegrationSyncTask, excludeID int) *time.Time {
	var latest *time.Time
	for _, value := range values {
		if value.Id == excludeID || value.Status == model.IntegrationSyncTaskStatusDraft || value.CheckpointAt == nil {
			continue
		}
		if latest == nil || value.CheckpointAt.After(*latest) {
			latest = cloneTime(value.CheckpointAt)
		}
	}
	return latest
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func (s *SyncTaskService) writeAudit(ctx context.Context, tx *gorm.DB, action string, value model.IntegrationSyncTask, previous *model.IntegrationSyncTask) error {
	changes := map[string]TransactionalAuditChange{"task_code": {NewValue: value.TaskCode}, "version": {NewValue: value.Version}, "status": {NewValue: value.Status}, "interface_definition_id": {NewValue: value.InterfaceDefinitionID},
		"consumer": {NewValue: value.ConsumerCode + "@" + strconv.Itoa(value.ConsumerVersion)}, "schedule_type": {NewValue: value.ScheduleType}, "checkpoint_mode": {NewValue: value.CheckpointMode}, "revision": {NewValue: value.Revision}}
	if previous != nil {
		changes["status"] = TransactionalAuditChange{OldValue: previous.Status, NewValue: value.Status}
		changes["revision"] = TransactionalAuditChange{OldValue: previous.Revision, NewValue: value.Revision}
	}
	return s.audit.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{Action: action, ResourceType: syncTaskAuditResourceType, ResourceCode: value.TaskCode + "@" + strconv.Itoa(value.Version), ResourceId: strconv.Itoa(value.Id), Changes: changes})
}

func isSyncDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "unique constraint") || strings.Contains(lower, "duplicate key")
}
