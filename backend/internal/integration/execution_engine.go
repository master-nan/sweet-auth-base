// Package integration 提供 Integration Runtime 的内部执行引擎。
package integration

import (
	"backend/internal/audit"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	defaultExecutionBatchSize = 8
	maxExecutionBatchSize     = 32
)

// ExecutionEngineOptions 仅接收 Worker 服务端配置，不接收 Controller DTO 或 Gin 上下文。
type ExecutionEngineOptions struct {
	WorkerID      string
	LeaseDuration time.Duration
	BatchSize     int
}

// AttemptResult 是一次真实调用的受控事实摘要。它不包含 Payload、认证信息或密文。
type AttemptResult struct {
	Succeeded                    bool
	HTTPMethod                   string
	HTTPStatus                   *int
	ResponseSize                 int64
	ResponseHash                 string
	ContentType                  string
	StartedAt                    time.Time
	CompletedAt                  time.Time
	Duration                     time.Duration
	ErrorCategory                string
	ReasonCode                   string
	SafeMessage                  string
	Certainty                    string
	RequestProgress              string
	RetryAfterRaw                string
	ExecutionStatus              string
	RetryReasonCode              string
	RetryDelay                   time.Duration
	RetryScheduledAt             *time.Time
	RetryAfterSource             string
	CredentialCode               string
	CredentialVersion            string
	CredentialFingerprintSummary string
	SyncBusinessStatus           string
	SyncBusinessReasonCode       string
	SyncBusinessSuccessCount     int
	SyncBusinessFailedCount      int
	SyncBusinessReference        string
}

// IntegrationExecutionEngine 负责编排领取、凭证解析、HTTP调用和原子状态收敛。
// 它不启动常驻循环；应用生命周期由IntegrationWorkerRunner负责。
type IntegrationExecutionEngine struct {
	executions    repository.IntegrationExecutionRepository
	systems       repository.ExternalSystemRepository
	interfaces    repository.InterfaceDefinitionRepository
	credentials   repository.CredentialRepository
	provider      *CredentialProvider
	transport     TransportClient
	guard         ConcurrencyGuard
	sf            *utils.Snowflake
	workerID      string
	leaseDuration time.Duration
	batchSize     int
	now           func() time.Time
	retryDecision *RetryDecisionService
	syncBatches   repository.IntegrationSyncBatchRepository
	syncConsumers SyncResultConsumerRegistry
}

// NewIntegrationExecutionEngine 构造可直接调用的受控 Engine。
func NewIntegrationExecutionEngine(
	executions repository.IntegrationExecutionRepository,
	systems repository.ExternalSystemRepository,
	interfaces repository.InterfaceDefinitionRepository,
	credentials repository.CredentialRepository,
	syncBatches repository.IntegrationSyncBatchRepository,
	provider *CredentialProvider,
	transport TransportClient,
	guard ConcurrencyGuard,
	syncConsumers SyncResultConsumerRegistry,
	sf *utils.Snowflake,
	options ExecutionEngineOptions,
) (*IntegrationExecutionEngine, error) {
	if executions == nil || systems == nil || interfaces == nil || credentials == nil || syncBatches == nil || provider == nil || transport == nil || guard == nil || syncConsumers == nil || sf == nil {
		return nil, myerrors.ErrIntegrationConfigurationUnavailable
	}
	workerID := strings.TrimSpace(options.WorkerID)
	if workerID == "" || len(workerID) > 128 {
		return nil, myerrors.ErrIntegrationConfigurationUnavailable
	}
	leaseDuration := options.LeaseDuration
	if leaseDuration == 0 {
		leaseDuration = IntegrationDefaultLeaseDuration
	}
	if err := ValidateLeaseDuration(leaseDuration); err != nil {
		return nil, err
	}
	batchSize := options.BatchSize
	if batchSize == 0 {
		batchSize = defaultExecutionBatchSize
	}
	if batchSize < 1 || batchSize > maxExecutionBatchSize {
		return nil, myerrors.ErrIntegrationConfigurationUnavailable
	}
	retryDecision, err := NewRetryDecisionService(NewRetryRandomSource(model.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	return &IntegrationExecutionEngine{
		executions: executions, systems: systems, interfaces: interfaces, credentials: credentials,
		provider: provider, transport: transport, guard: guard, sf: sf, workerID: workerID,
		leaseDuration: leaseDuration, batchSize: batchSize, now: model.Now, retryDecision: retryDecision,
		syncBatches: syncBatches, syncConsumers: syncConsumers,
	}, nil
}

// ClaimReadyExecutions 在领取事务提交后返回首次调用或到期重试；领取失败不会触发 HTTP。
func (e *IntegrationExecutionEngine) ClaimReadyExecutions(ctx context.Context) ([]repository.ClaimedIntegrationExecution, error) {
	if e == nil {
		return nil, myerrors.ErrIntegrationConfigurationUnavailable
	}
	attemptIDs := make([]int, 0, e.batchSize)
	for index := 0; index < e.batchSize; index++ {
		id, err := e.sf.GenerateUniqueID()
		if err != nil {
			return nil, myerrors.ErrIntegrationAttemptCreateFailed
		}
		attemptIDs = append(attemptIDs, int(id))
	}
	now := e.now()
	correlation := audit.GetCorrelationIDs(ctx)
	claimed, err := e.executions.ClaimReadyExecutions(ctx, repository.IntegrationExecutionClaimRequest{
		WorkerID: e.workerID, LeaseExpiresAt: now.Add(e.leaseDuration), StartedAt: now,
		RequestID: correlation.RequestID, TraceID: correlation.TraceID, AttemptIDs: attemptIDs,
	})
	if err != nil {
		if errors.Is(err, repository.ErrIntegrationRetryAttemptCreateFailed) {
			return nil, myerrors.ErrIntegrationRetryAttemptCreateFailed
		}
		if errors.Is(err, repository.ErrIntegrationExecutionClaimUnavailable) {
			return nil, myerrors.ErrIntegrationExecutionClaimConflict
		}
		return nil, myerrors.WrapDatabaseError(err)
	}
	return claimed, nil
}

// RunOnce 领取最多一个受控批次并逐项执行。批量有上限，未创建后台 goroutine。
func (e *IntegrationExecutionEngine) RunOnce(ctx context.Context) (int, error) {
	claimed, err := e.ClaimReadyExecutions(ctx)
	if err != nil {
		return 0, err
	}
	for _, item := range claimed {
		if _, err := e.RunExecution(ctx, item); err != nil {
			return 0, err
		}
	}
	return len(claimed), nil
}

// RunExecution 在领取事务之外调用 Provider 和 Transport，然后以新短事务收敛状态。
func (e *IntegrationExecutionEngine) RunExecution(
	ctx context.Context,
	claimed repository.ClaimedIntegrationExecution,
) (result AttemptResult, err error) {
	if e == nil || claimed.Execution.Status != model.IntegrationExecutionStatusRunning || claimed.Attempt.Status != model.IntegrationLogStatusRunning ||
		claimed.Execution.LeaseOwner != e.workerID || claimed.Execution.LeaseExpiresAt == nil {
		return AttemptResult{}, myerrors.ErrIntegrationExecutionClaimConflict
	}
	databaseNow, timeErr := e.executions.CurrentDatabaseTime(ctx)
	if timeErr != nil {
		return AttemptResult{}, myerrors.ErrIntegrationExecutionResultUnknown
	}
	if !claimed.Execution.LeaseExpiresAt.After(databaseNow) {
		return AttemptResult{}, myerrors.ErrIntegrationExecutionLeaseLost
	}
	defer func() {
		if recover() == nil {
			return
		}
		result = e.failureResult(claimed.Attempt.StartedAt, model.IntegrationErrorCategorySystem, "worker_panic_recovered", "集成执行发生内部异常", model.IntegrationResultCertaintyConfirmed)
		if syncErr := e.syncCompletionTime(ctx, claimed.Attempt.StartedAt, &result); syncErr != nil {
			err = myerrors.ErrIntegrationExecutionResultUnknown
			return
		}
		e.applyRetryDecision(claimed, &result)
		if completeErr := e.completeClaim(ctx, claimed, result); completeErr != nil {
			err = myerrors.ErrIntegrationExecutionResultUnknown
			return
		}
		e.logAttempt(ctx, claimed, result, "panic_recovered")
		err = myerrors.ErrIntegrationWorkerPanicRecovered
	}()
	release, err := e.guard.Acquire(claimed.Execution.ExternalSystemID, claimed.Execution.InterfaceDefinitionID)
	if err != nil {
		result := e.failureResult(claimed.Attempt.StartedAt, model.IntegrationErrorCategoryConcurrency, "concurrency_limit_reached", "集成执行并发已达上限", model.IntegrationResultCertaintyConfirmed)
		if syncErr := e.syncCompletionTime(ctx, claimed.Attempt.StartedAt, &result); syncErr != nil {
			return result, myerrors.ErrIntegrationExecutionResultUnknown
		}
		e.applyRetryDecision(claimed, &result)
		return result, e.completeClaim(ctx, claimed, result)
	}
	defer release()

	result = e.executeAttempt(ctx, claimed)
	if syncErr := e.syncCompletionTime(ctx, claimed.Attempt.StartedAt, &result); syncErr != nil {
		return result, myerrors.ErrIntegrationExecutionResultUnknown
	}
	e.applyRetryDecision(claimed, &result)
	if err := e.completeClaim(ctx, claimed, result); err != nil {
		// 远端调用完成后若持久化失败，不能宣称成功；后续仅能通过租约恢复收敛为 unknown。
		e.logAttempt(ctx, claimed, result, "complete_failed")
		return result, myerrors.ErrIntegrationExecutionResultUnknown
	}
	e.logAttempt(ctx, claimed, result, "completed")
	return result, nil
}

func (e *IntegrationExecutionEngine) syncCompletionTime(ctx context.Context, startedAt time.Time, result *AttemptResult) error {
	databaseNow, err := e.executions.CurrentDatabaseTime(ctx)
	if err != nil {
		return err
	}
	result.CompletedAt = databaseNow
	result.Duration = databaseNow.Sub(startedAt)
	if result.Duration < 0 {
		result.Duration = 0
	}
	return nil
}

func (e *IntegrationExecutionEngine) executeAttempt(ctx context.Context, claimed repository.ClaimedIntegrationExecution) AttemptResult {
	startedAt := claimed.Attempt.StartedAt
	system, definition, credentialIdentity, err := e.loadRuntimeConfiguration(ctx, claimed.Execution)
	if err != nil {
		if errors.Is(err, myerrors.ErrIntegrationExecutionRuntimeIncompatible) {
			return e.failureResult(startedAt, model.IntegrationErrorCategoryConfiguration, "integration_execution_runtime_incompatible", "集成执行引用的接口不符合当前运行契约", model.IntegrationResultCertaintyConfirmed)
		}
		return e.failureResult(startedAt, model.IntegrationErrorCategoryConfiguration, "configuration_unavailable", "集成运行配置不可用", model.IntegrationResultCertaintyConfirmed)
	}
	syncRuntime, err := e.loadSyncRuntime(ctx, claimed.Execution, definition)
	if err != nil {
		return e.failureResult(startedAt, model.IntegrationErrorCategoryConfiguration, "sync_consumer_not_registered", "同步 Consumer 配置不可用", model.IntegrationResultCertaintyConfirmed)
	}
	snapshot, err := LoadExecutionInputSnapshot(
		definition.InputContract,
		definition.HTTPMethod,
		definition.RelativePath,
		definition.Version,
		claimed.Execution.InputSnapshot,
		claimed.Execution.InputSnapshotSize,
		claimed.Execution.InputHash,
	)
	if err != nil {
		return e.failureResult(startedAt, model.IntegrationErrorCategoryConfiguration, executionInputReasonCode(err), "集成执行输入快照不可用", model.IntegrationResultCertaintyConfirmed)
	}
	headers := snapshotTransportHeaders(snapshot.Headers)
	if claimed.Execution.RemoteIdempotencyMode == RemoteIdempotencyKeyHeader &&
		strings.EqualFold(claimed.Execution.RemoteIdempotencyHeader, RemoteIdempotencyHeaderName) {
		headers[RemoteIdempotencyHeaderName] = claimed.Execution.RemoteIdempotencyKey
	}
	correlation := audit.GetCorrelationIDs(ctx)
	if correlation.RequestID != "" {
		headers["X-Request-ID"] = correlation.RequestID
	}
	if correlation.TraceID != "" {
		headers["X-Trace-ID"] = correlation.TraceID
	}
	request, err := NewTransportRequest(TransportRequestInput{
		Method: definition.HTTPMethod, BaseURL: system.BaseURL, RelativePath: definition.RelativePath,
		PathParameters: snapshot.PathParams, QueryParameters: snapshot.QueryParams,
		Headers: headers, JSONBody: snapshot.JSONBody,
		Timeouts:         TransportTimeouts{Request: time.Duration(definition.TimeoutSeconds) * time.Second},
		MaxResponseBytes: definition.ResponseLimit,
	})
	if err != nil {
		return e.failureResult(startedAt, model.IntegrationErrorCategoryConfiguration, "transport_request_invalid", "集成运行配置不可用", model.IntegrationResultCertaintyConfirmed)
	}
	resolveRequest, err := NewCredentialResolveRequest(system.Id, definition.Id, credentialIdentity.ID, credentialIdentity.CredentialCode, credentialIdentity.CredentialType, claimed.Execution.ExecutionNo)
	if err != nil {
		return e.failureResult(startedAt, model.IntegrationErrorCategoryCredential, "credential_resolution_failed", "集成运行凭证解析失败", model.IntegrationResultCertaintyConfirmed)
	}
	resolution, err := e.provider.Resolve(ctx, resolveRequest)
	if err != nil {
		return e.failureResult(startedAt, model.IntegrationErrorCategoryCredential, "credential_resolution_failed", "集成运行凭证解析失败", model.IntegrationResultCertaintyConfirmed)
	}
	request, err = request.WithAuthentication(resolution.Authentication())
	if err != nil {
		return withCredentialSummary(e.failureResult(startedAt, model.IntegrationErrorCategoryCredential, "credential_injection_invalid", "集成运行凭证解析失败", model.IntegrationResultCertaintyConfirmed), resolution)
	}
	transportResult, transportErr := e.transport.Execute(ctx, request)
	result := attemptResultFromTransport(startedAt, transportResult, transportErr, e.now())
	result.HTTPMethod = definition.HTTPMethod
	if result.Succeeded && syncRuntime != nil {
		result = e.consumeSyncResult(ctx, claimed.Execution, *syncRuntime, transportResult, result)
	}
	return withCredentialSummary(result, resolution)
}

type syncExecutionRuntime struct {
	batch    model.IntegrationSyncBatch
	consumer ResolvedSyncResultConsumer
}

func (e *IntegrationExecutionEngine) loadSyncRuntime(ctx context.Context, execution model.IntegrationExecution, definition model.InterfaceDefinition) (*syncExecutionRuntime, error) {
	if execution.SyncBatchID == nil {
		return nil, nil
	}
	if execution.SyncSliceNo == nil || *execution.SyncSliceNo <= 0 || execution.SyncConsumerVersion == nil ||
		strings.TrimSpace(execution.SyncConsumerCode) == "" {
		return nil, myerrors.ErrSyncConsumerIncompatible
	}
	batch, err := e.syncBatches.WithContext(ctx).FindById(*execution.SyncBatchID)
	if err != nil || batch.Id != *execution.SyncBatchID || batch.ConsumerCode != execution.SyncConsumerCode ||
		batch.ConsumerVersion != *execution.SyncConsumerVersion || batch.InterfaceVersion != execution.InterfaceVersion ||
		batch.InterfaceCode != execution.InterfaceCode || batch.SystemCode != execution.ExternalSystemCode ||
		batch.Status != model.IntegrationSyncBatchStatusRunning || batch.TaskCode == "" || batch.TaskVersion <= 0 {
		return nil, myerrors.ErrSyncConsumerIncompatible
	}
	if _, err := e.syncConsumers.ValidateReference(SyncConsumerReference{
		Code: execution.SyncConsumerCode, Version: *execution.SyncConsumerVersion, ResponseLimit: definition.ResponseLimit,
		CheckpointMode: batch.CheckpointMode, RequestTimeout: time.Duration(definition.TimeoutSeconds) * time.Second,
		LeaseDuration: e.leaseDuration,
	}); err != nil {
		return nil, err
	}
	consumer, err := e.syncConsumers.Resolve(execution.SyncConsumerCode, *execution.SyncConsumerVersion)
	if err != nil {
		return nil, err
	}
	return &syncExecutionRuntime{batch: batch, consumer: consumer}, nil
}

func (e *IntegrationExecutionEngine) consumeSyncResult(
	ctx context.Context,
	execution model.IntegrationExecution,
	runtime syncExecutionRuntime,
	transport TransportResult,
	result AttemptResult,
) AttemptResult {
	metadata := runtime.consumer.Metadata()
	if transport.ResponseSize > metadata.MaxResponseBytes || !containsFold(metadata.ContentTypes, transport.ContentType) {
		return syncBusinessFailure(result, "sync_consumer_incompatible", 1, "")
	}
	request, err := NewSyncConsumptionRequest(SyncConsumptionRequestInput{
		ExecutionNo: execution.ExecutionNo, SyncBatchNo: runtime.batch.BatchNo, TaskCode: runtime.batch.TaskCode,
		TaskVersion: runtime.batch.TaskVersion, SliceNo: *execution.SyncSliceNo, WindowStart: execution.SyncWindowStart,
		WindowEnd: execution.SyncWindowEnd, ContentType: transport.ContentType, ResponseSize: transport.ResponseSize,
		ResponseHash: transport.ResponseHash, Body: transport.Body(),
	})
	if err != nil {
		return syncBusinessFailure(result, "sync_consumption_request_invalid", 1, "")
	}
	consumed, err := runtime.consumer.Consume(ctx, request)
	if err != nil {
		reason := SyncBusinessReasonProcessingFailed
		switch {
		case errors.Is(err, myerrors.ErrSyncConsumerTimeout):
			reason = "sync_consumer_timeout"
		case errors.Is(err, myerrors.ErrSyncConsumerPanic):
			reason = "sync_consumer_panic"
		case errors.Is(err, myerrors.ErrSyncConsumptionResultInvalid):
			reason = "sync_consumer_result_invalid"
		}
		return syncBusinessFailure(result, reason, 1, "")
	}
	if !consumed.Success() {
		failedCount := consumed.BusinessFailedCount()
		if failedCount < 1 {
			failedCount = 1
		}
		return syncBusinessFailure(result, consumed.ReasonCode(), failedCount, consumed.BusinessReference())
	}
	result.SyncBusinessStatus = model.IntegrationSyncBusinessStatusSucceeded
	result.SyncBusinessSuccessCount = consumed.BusinessSuccessCount()
	result.SyncBusinessFailedCount = 0
	result.SyncBusinessReference = consumed.BusinessReference()
	result.SafeMessage = "远端调用及同步业务处理成功"
	return result
}

func syncBusinessFailure(result AttemptResult, reason string, failedCount int, reference string) AttemptResult {
	result.Succeeded = false
	result.ErrorCategory = model.IntegrationErrorCategoryBusiness
	result.ReasonCode = SyncBusinessReasonProcessingFailed
	result.SafeMessage = "同步业务处理失败"
	result.Certainty = model.IntegrationResultCertaintyConfirmed
	result.ExecutionStatus = model.IntegrationExecutionStatusFailed
	result.RetryReasonCode = RetryReasonErrorNotAllowed
	result.RetryScheduledAt = nil
	result.RetryDelay = 0
	result.RetryAfterSource = RetryAfterSourceNone
	result.SyncBusinessStatus = model.IntegrationSyncBusinessStatusFailed
	result.SyncBusinessReasonCode = strings.TrimSpace(reason)
	result.SyncBusinessFailedCount = failedCount
	result.SyncBusinessReference = strings.TrimSpace(reference)
	return result
}

func snapshotTransportHeaders(values map[string][]string) map[string]string {
	result := make(map[string]string, len(values)+2)
	for name, items := range values {
		if len(items) == 1 {
			result[name] = items[0]
		}
	}
	return result
}

func executionInputReasonCode(err error) string {
	switch {
	case errors.Is(err, myerrors.ErrIntegrationExecutionInputMissing):
		return "execution_input_missing"
	case errors.Is(err, myerrors.ErrIntegrationExecutionInputStorageTooLarge):
		return "execution_input_storage_too_large"
	case errors.Is(err, myerrors.ErrIntegrationExecutionInputSemanticTooLarge):
		return "execution_input_semantic_too_large"
	case errors.Is(err, myerrors.ErrIntegrationExecutionInputSizeMismatch):
		return "execution_input_size_mismatch"
	case errors.Is(err, myerrors.ErrIntegrationExecutionInputContractMismatch):
		return "execution_input_contract_mismatch"
	case errors.Is(err, myerrors.ErrIntegrationExecutionInputHashMismatch):
		return "execution_input_hash_mismatch"
	case errors.Is(err, myerrors.ErrIntegrationExecutionInputVersionUnsupported):
		return "execution_input_version_unsupported"
	default:
		return "execution_input_invalid"
	}
}

func (e *IntegrationExecutionEngine) loadRuntimeConfiguration(
	ctx context.Context,
	execution model.IntegrationExecution,
) (model.ExternalSystem, model.InterfaceDefinition, repository.CredentialRuntimeIdentity, error) {
	system, err := e.systems.FindByIdWithDB(e.systems.DBWithContext(ctx), execution.ExternalSystemID)
	if err != nil || system.Id != execution.ExternalSystemID || system.SystemCode != execution.ExternalSystemCode ||
		system.Status != model.ExternalSystemStatusEnabled || !system.State {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, repository.CredentialRuntimeIdentity{}, myerrors.ErrIntegrationConfigurationUnavailable
	}
	definition, err := e.interfaces.FindByIdWithDB(e.interfaces.DBWithContext(ctx), execution.InterfaceDefinitionID)
	if err != nil || definition.Id != execution.InterfaceDefinitionID || definition.ExternalSystemID != system.Id ||
		definition.InterfaceCode != execution.InterfaceCode || definition.Version != execution.InterfaceVersion ||
		definition.Status != model.InterfaceDefinitionStatusEnabled || !definition.State || definition.CredentialID == nil {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, repository.CredentialRuntimeIdentity{}, myerrors.ErrIntegrationConfigurationUnavailable
	}
	if err := ValidateInterfaceRuntimeContract(definition.TimeoutSeconds, definition.ResponseLimit); err != nil {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, repository.CredentialRuntimeIdentity{}, myerrors.ErrIntegrationExecutionRuntimeIncompatible
	}
	if !ValidRemoteIdempotencyContract(definition.HTTPMethod, definition.IdempotencyMode, definition.RemoteIdempotencyHeader) ||
		definition.IdempotencyMode != execution.RemoteIdempotencyMode || definition.RemoteIdempotencyHeader != execution.RemoteIdempotencyHeader {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, repository.CredentialRuntimeIdentity{}, myerrors.ErrIntegrationExecutionRuntimeIncompatible
	}
	identity, err := e.credentials.GetRuntimeCredentialIdentity(ctx, *definition.CredentialID)
	if err != nil || identity.ID != *definition.CredentialID || identity.ExternalSystemID != system.Id {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, repository.CredentialRuntimeIdentity{}, myerrors.ErrIntegrationConfigurationUnavailable
	}
	return system, definition, identity, nil
}

func (e *IntegrationExecutionEngine) completeClaim(ctx context.Context, claimed repository.ClaimedIntegrationExecution, result AttemptResult) error {
	status := result.ExecutionStatus
	if status == "" {
		status = model.IntegrationExecutionStatusFailed
	}
	_, err := e.executions.CompleteAttemptAndExecution(ctx, repository.IntegrationAttemptCompletion{
		ExecutionID: claimed.Execution.Id, AttemptID: claimed.Attempt.Id, AttemptNo: claimed.Attempt.AttemptNo,
		WorkerID: e.workerID, ExpectedRevision: claimed.Execution.Revision, ExecutionStatus: status,
		CompletedAt: result.CompletedAt, HTTPStatus: result.HTTPStatus, ResultSizeBytes: result.ResponseSize,
		ResultHash: result.ResponseHash, ResultSummary: result.SafeMessage, ErrorCategory: result.ErrorCategory,
		ErrorCode: result.ReasonCode, ResultCertainty: result.Certainty, ResponseContentType: result.ContentType,
		CredentialCode: result.CredentialCode, CredentialVersion: result.CredentialVersion,
		CredentialFingerprintSummary: result.CredentialFingerprintSummary,
		Retryable:                    result.ExecutionStatus == model.IntegrationExecutionStatusRetryWaiting,
		RetryReasonCode:              result.RetryReasonCode, RetryDelayMs: result.RetryDelay.Milliseconds(),
		RetryScheduledAt: result.RetryScheduledAt, RetryAfterSource: result.RetryAfterSource,
		SyncBusinessStatus: result.SyncBusinessStatus, SyncBusinessReasonCode: result.SyncBusinessReasonCode,
		SyncBusinessSuccessCount: result.SyncBusinessSuccessCount, SyncBusinessFailedCount: result.SyncBusinessFailedCount,
		SyncBusinessReference: result.SyncBusinessReference,
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrIntegrationExecutionLeaseLost) {
		return myerrors.ErrIntegrationExecutionLeaseLost
	}
	if errors.Is(err, repository.ErrIntegrationAttemptAlreadyCompleted) {
		return myerrors.ErrIntegrationAttemptAlreadyCompleted
	}
	if claimed.Attempt.AttemptNo > 1 {
		return myerrors.ErrIntegrationRetryExecutionCompleteFailed
	}
	return myerrors.ErrIntegrationExecutionCompleteFailed
}

func (e *IntegrationExecutionEngine) applyRetryDecision(claimed repository.ClaimedIntegrationExecution, result *AttemptResult) {
	if result == nil {
		return
	}
	if result.Succeeded {
		result.ExecutionStatus = model.IntegrationExecutionStatusSucceeded
		result.RetryAfterSource = RetryAfterSourceNone
		return
	}
	if result.ErrorCategory == model.IntegrationErrorCategoryBusiness {
		result.ExecutionStatus = model.IntegrationExecutionStatusFailed
		result.RetryReasonCode = RetryReasonErrorNotAllowed
		result.RetryAfterSource = RetryAfterSourceNone
		result.RetryScheduledAt = nil
		result.RetryDelay = 0
		return
	}
	result.ExecutionStatus = model.IntegrationExecutionStatusFailed
	result.RetryAfterSource = RetryAfterSourceNone
	if claimed.Execution.RetryPolicySnapshotVersion != RetryPolicySnapshotVersion {
		result.RetryReasonCode = RetryReasonErrorNotAllowed
		return
	}
	policy, err := ParseRetryPolicySnapshot(claimed.Execution.RetryPolicySnapshot)
	if err != nil {
		result.RetryReasonCode = RetryReasonPolicyInvalid
		return
	}
	firstAttemptAt := claimed.Attempt.StartedAt
	if claimed.Execution.StartedAt != nil {
		firstAttemptAt = *claimed.Execution.StartedAt
	}
	progress := result.RequestProgress
	if progress == "" {
		progress = RequestProgressSentUnknown
	}
	status := 0
	if result.HTTPStatus != nil {
		status = *result.HTTPStatus
	}
	decision, err := e.retryDecision.Decide(RetryDecisionInput{
		PolicySnapshot: policy, HTTPMethod: result.HTTPMethod, AttemptNo: claimed.Attempt.AttemptNo,
		ErrorCategory: result.ErrorCategory, ReasonCode: result.ReasonCode, HTTPStatus: status,
		ResultDeterminacy: result.Certainty, RequestProgress: progress, RetryAfterRaw: result.RetryAfterRaw,
		FirstAttemptAt: firstAttemptAt, CurrentTime: result.CompletedAt, ExecutionStatus: claimed.Execution.Status,
		RemoteIdempotencyMode:   claimed.Execution.RemoteIdempotencyMode,
		HasRemoteIdempotencyKey: claimed.Execution.RemoteIdempotencyMode == RemoteIdempotencyKeyHeader && strings.TrimSpace(claimed.Execution.RemoteIdempotencyKey) != "",
	})
	if err != nil {
		result.RetryReasonCode = RetryReasonPolicyInvalid
		return
	}
	result.ExecutionStatus = decision.FinalState()
	result.RetryReasonCode = decision.ReasonCode()
	result.RetryDelay = decision.RetryDelay()
	result.RetryAfterSource = decision.RetryAfterSource()
	if decision.Retryable() {
		next := decision.NextRetryAt()
		result.RetryScheduledAt = &next
	}
}

func withCredentialSummary(result AttemptResult, resolution CredentialResolution) AttemptResult {
	result.CredentialCode = resolution.CredentialCode()
	result.CredentialVersion = resolution.SecurityVersionSummary()
	result.CredentialFingerprintSummary = resolution.FingerprintSummary()
	return result
}

// RecoverExpiredLease 将超时 running Execution 收敛为失败且结果 unknown，不会自动重发远端请求。
func (e *IntegrationExecutionEngine) RecoverExpiredLease(ctx context.Context, limit int) (int, error) {
	if e == nil || limit <= 0 || limit > maxExecutionBatchSize {
		return 0, myerrors.ErrIntegrationLeaseRecoveryFailed
	}
	values, err := e.executions.FindExpiredRunningExecutions(ctx, limit)
	if err != nil {
		return 0, myerrors.ErrIntegrationLeaseRecoveryFailed
	}
	recovered := 0
	for _, value := range values {
		changed, recoverErr := e.executions.RecoverExpiredExecution(ctx, repository.ExpiredExecutionRecovery{ExecutionID: value.Id})
		if recoverErr != nil && !errors.Is(recoverErr, repository.ErrIntegrationExecutionLeaseLost) && !errors.Is(recoverErr, repository.ErrIntegrationAttemptAlreadyCompleted) {
			return recovered, myerrors.ErrIntegrationLeaseRecoveryFailed
		}
		if changed {
			recovered++
		}
	}
	return recovered, nil
}

func (e *IntegrationExecutionEngine) failureResult(startedAt time.Time, category, reason, message, certainty string) AttemptResult {
	completedAt := e.now()
	return AttemptResult{StartedAt: startedAt, CompletedAt: completedAt, Duration: completedAt.Sub(startedAt),
		ErrorCategory: category, ReasonCode: reason, SafeMessage: message, Certainty: certainty,
		RequestProgress: RequestProgressNotSent, ExecutionStatus: model.IntegrationExecutionStatusFailed, RetryAfterSource: RetryAfterSourceNone}
}

func attemptResultFromTransport(startedAt time.Time, transportResult TransportResult, transportErr error, completedAt time.Time) AttemptResult {
	result := AttemptResult{HTTPStatus: optionalHTTPStatus(transportResult.StatusCode), ResponseSize: transportResult.ResponseSize,
		ResponseHash: transportResult.ResponseHash, ContentType: transportResult.ContentType, StartedAt: startedAt,
		CompletedAt: completedAt, Duration: completedAt.Sub(startedAt), Certainty: string(transportResult.Determinacy),
		RequestProgress: string(transportResult.RequestProgress), ExecutionStatus: model.IntegrationExecutionStatusFailed,
		RetryAfterSource: RetryAfterSourceNone, RetryAfterRaw: transportResult.ResponseHeaders()["retry-after"]}
	if result.Certainty == "" {
		result.Certainty = model.IntegrationResultCertaintyUnknown
	}
	if transportErr == nil && transportResult.ErrorCategory == "" && transportResult.StatusCode >= http.StatusOK && transportResult.StatusCode < http.StatusMultipleChoices {
		result.Succeeded = true
		result.Certainty = model.IntegrationResultCertaintyConfirmed
		result.SafeMessage = "远端调用成功"
		result.ExecutionStatus = model.IntegrationExecutionStatusSucceeded
		return result
	}
	result.ErrorCategory, result.ReasonCode, result.SafeMessage = mapTransportError(transportResult.ErrorCategory, transportErr)
	return result
}

func optionalHTTPStatus(value int) *int {
	if value < 100 || value > 599 {
		return nil
	}
	result := value
	return &result
}

func mapTransportError(category TransportErrorCategory, err error) (string, string, string) {
	if err != nil && errors.Is(err, context.Canceled) {
		return model.IntegrationErrorCategorySystem, "cancelled", "集成调用已取消"
	}
	switch category {
	case TransportErrorTimeout:
		return model.IntegrationErrorCategoryTimeout, "timeout", "集成远程调用超时"
	case TransportErrorNetwork:
		return model.IntegrationErrorCategoryNetwork, "network_error", "集成远程调用失败"
	case TransportErrorTLS:
		return model.IntegrationErrorCategoryNetwork, "tls_error", "集成远程调用安全校验失败"
	case TransportErrorRemoteHTTP:
		return model.IntegrationErrorCategoryRemote, "remote_http_error", "集成远端返回失败状态"
	case TransportErrorResponseTooLarge, TransportErrorUnsupportedContentType:
		return model.IntegrationErrorCategoryResponse, "response_invalid", "集成远端响应不符合限制"
	case TransportErrorInvalidConfig, TransportErrorInvalidURL, TransportErrorSSRFRejected, TransportErrorRedirectRejected:
		return model.IntegrationErrorCategoryConfiguration, "transport_configuration_invalid", "集成运行配置不可用"
	case TransportErrorCancelled:
		return model.IntegrationErrorCategorySystem, "cancelled", "集成调用已取消"
	default:
		return model.IntegrationErrorCategorySystem, "transport_failed", "集成远程调用失败"
	}
}

func (e *IntegrationExecutionEngine) logAttempt(ctx context.Context, claimed repository.ClaimedIntegrationExecution, result AttemptResult, phase string) {
	correlation := audit.GetCorrelationIDs(ctx)
	policyCode := ""
	policyVersion := 0
	if policy, err := ParseRetryPolicySnapshot(claimed.Execution.RetryPolicySnapshot); err == nil {
		policyCode = policy.PolicyCode
		policyVersion = policy.PolicyVersion
	}
	zap.L().Info("integration execution attempt",
		zap.String("request_id", correlation.RequestID), zap.String("trace_id", correlation.TraceID),
		zap.String("execution_no", claimed.Execution.ExecutionNo), zap.Int("attempt_no", claimed.Attempt.AttemptNo),
		zap.String("system_code", claimed.Execution.ExternalSystemCode), zap.String("interface_code", claimed.Execution.InterfaceCode),
		zap.String("worker_id", e.workerID), zap.String("phase", phase), zap.Int("http_status", transportStatus(result.HTTPStatus)),
		zap.Duration("duration", result.Duration), zap.String("error_category", result.ErrorCategory),
		zap.String("result_certainty", result.Certainty),
		zap.Bool("is_retry", claimed.Attempt.AttemptNo > 1),
		zap.Bool("retryable", result.ExecutionStatus == model.IntegrationExecutionStatusRetryWaiting),
		zap.String("retry_reason_code", result.RetryReasonCode),
		zap.Time("retry_scheduled_at", retryScheduledAt(result.RetryScheduledAt)),
		zap.String("policy_code", policyCode), zap.Int("policy_version", policyVersion),
	)
}

func retryScheduledAt(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func transportStatus(status *int) int {
	if status == nil {
		return 0
	}
	return *status
}
