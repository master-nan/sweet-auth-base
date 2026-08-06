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
	defaultExecutionLeaseDuration = 2 * time.Minute
	defaultExecutionBatchSize     = 8
	maxExecutionBatchSize         = 32
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
	RetryEligible                bool
	CredentialCode               string
	CredentialVersion            string
	CredentialFingerprintSummary string
}

// IntegrationExecutionEngine 负责编排领取、凭证解析、HTTP 调用和原子状态收敛。
// 它不启动常驻循环；应用生命周期接入由后续任务完成。
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
}

// NewIntegrationExecutionEngine 构造可直接调用的受控 Engine。
func NewIntegrationExecutionEngine(
	executions repository.IntegrationExecutionRepository,
	systems repository.ExternalSystemRepository,
	interfaces repository.InterfaceDefinitionRepository,
	credentials repository.CredentialRepository,
	provider *CredentialProvider,
	transport TransportClient,
	guard ConcurrencyGuard,
	sf *utils.Snowflake,
	options ExecutionEngineOptions,
) (*IntegrationExecutionEngine, error) {
	if executions == nil || systems == nil || interfaces == nil || credentials == nil || provider == nil || transport == nil || guard == nil || sf == nil {
		return nil, myerrors.ErrIntegrationConfigurationUnavailable
	}
	workerID := strings.TrimSpace(options.WorkerID)
	if workerID == "" || len(workerID) > 128 {
		return nil, myerrors.ErrIntegrationConfigurationUnavailable
	}
	leaseDuration := options.LeaseDuration
	if leaseDuration == 0 {
		leaseDuration = defaultExecutionLeaseDuration
	}
	if leaseDuration < 5*time.Second || leaseDuration > 10*time.Minute {
		return nil, myerrors.ErrIntegrationConfigurationUnavailable
	}
	batchSize := options.BatchSize
	if batchSize == 0 {
		batchSize = defaultExecutionBatchSize
	}
	if batchSize < 1 || batchSize > maxExecutionBatchSize {
		return nil, myerrors.ErrIntegrationConfigurationUnavailable
	}
	return &IntegrationExecutionEngine{
		executions: executions, systems: systems, interfaces: interfaces, credentials: credentials,
		provider: provider, transport: transport, guard: guard, sf: sf, workerID: workerID,
		leaseDuration: leaseDuration, batchSize: batchSize, now: model.Now,
	}, nil
}

// ClaimCreatedExecutions 在领取事务提交后返回可运行的 Execution；领取失败不会触发 HTTP。
func (e *IntegrationExecutionEngine) ClaimCreatedExecutions(ctx context.Context) ([]repository.ClaimedIntegrationExecution, error) {
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
	claimed, err := e.executions.ClaimCreatedExecutions(ctx, repository.IntegrationExecutionClaimRequest{
		WorkerID: e.workerID, LeaseExpiresAt: now.Add(e.leaseDuration), StartedAt: now,
		RequestID: correlation.RequestID, TraceID: correlation.TraceID, AttemptIDs: attemptIDs,
	})
	if err != nil {
		if errors.Is(err, repository.ErrIntegrationExecutionClaimUnavailable) {
			return nil, myerrors.ErrIntegrationExecutionClaimConflict
		}
		return nil, myerrors.WrapDatabaseError(err)
	}
	return claimed, nil
}

// RunOnce 领取最多一个受控批次并逐项执行。批量有上限，未创建后台 goroutine。
func (e *IntegrationExecutionEngine) RunOnce(ctx context.Context) (int, error) {
	claimed, err := e.ClaimCreatedExecutions(ctx)
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
	if !claimed.Execution.LeaseExpiresAt.After(e.now()) {
		return AttemptResult{}, myerrors.ErrIntegrationExecutionLeaseLost
	}
	defer func() {
		if recover() == nil {
			return
		}
		result = e.failureResult(claimed.Attempt.StartedAt, model.IntegrationErrorCategorySystem, "worker_panic_recovered", "集成执行发生内部异常", model.IntegrationResultCertaintyConfirmed, false)
		if completeErr := e.completeClaim(ctx, claimed, result); completeErr != nil {
			err = myerrors.ErrIntegrationExecutionResultUnknown
			return
		}
		e.logAttempt(ctx, claimed, result, "panic_recovered")
		err = myerrors.ErrIntegrationWorkerPanicRecovered
	}()
	release, err := e.guard.Acquire(claimed.Execution.ExternalSystemID, claimed.Execution.InterfaceDefinitionID)
	if err != nil {
		result := e.failureResult(claimed.Attempt.StartedAt, model.IntegrationErrorCategoryConcurrency, "concurrency_limit_reached", "集成执行并发已达上限", model.IntegrationResultCertaintyConfirmed, false)
		return result, e.completeClaim(ctx, claimed, result)
	}
	defer release()

	result = e.executeAttempt(ctx, claimed)
	if err := e.completeClaim(ctx, claimed, result); err != nil {
		// 远端调用完成后若持久化失败，不能宣称成功；后续仅能通过租约恢复收敛为 unknown。
		e.logAttempt(ctx, claimed, result, "complete_failed")
		return result, myerrors.ErrIntegrationExecutionResultUnknown
	}
	e.logAttempt(ctx, claimed, result, "completed")
	return result, nil
}

func (e *IntegrationExecutionEngine) executeAttempt(ctx context.Context, claimed repository.ClaimedIntegrationExecution) AttemptResult {
	startedAt := claimed.Attempt.StartedAt
	system, definition, credentialIdentity, err := e.loadRuntimeConfiguration(ctx, claimed.Execution)
	if err != nil {
		return e.failureResult(startedAt, model.IntegrationErrorCategoryConfiguration, "configuration_unavailable", "集成运行配置不可用", model.IntegrationResultCertaintyConfirmed, false)
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
		return e.failureResult(startedAt, model.IntegrationErrorCategoryConfiguration, executionInputReasonCode(err), "集成执行输入快照不可用", model.IntegrationResultCertaintyConfirmed, false)
	}
	headers := snapshotTransportHeaders(snapshot.Headers)
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
		return e.failureResult(startedAt, model.IntegrationErrorCategoryConfiguration, "transport_request_invalid", "集成运行配置不可用", model.IntegrationResultCertaintyConfirmed, false)
	}
	resolveRequest, err := NewCredentialResolveRequest(system.Id, definition.Id, credentialIdentity.ID, credentialIdentity.CredentialCode, credentialIdentity.CredentialType, claimed.Execution.ExecutionNo)
	if err != nil {
		return e.failureResult(startedAt, model.IntegrationErrorCategoryCredential, "credential_resolution_failed", "集成运行凭证解析失败", model.IntegrationResultCertaintyConfirmed, false)
	}
	resolution, err := e.provider.Resolve(ctx, resolveRequest)
	if err != nil {
		return e.failureResult(startedAt, model.IntegrationErrorCategoryCredential, "credential_resolution_failed", "集成运行凭证解析失败", model.IntegrationResultCertaintyConfirmed, false)
	}
	request, err = request.WithAuthentication(resolution.Authentication())
	if err != nil {
		return withCredentialSummary(e.failureResult(startedAt, model.IntegrationErrorCategoryCredential, "credential_injection_invalid", "集成运行凭证解析失败", model.IntegrationResultCertaintyConfirmed, false), resolution)
	}
	transportResult, transportErr := e.transport.Execute(ctx, request)
	return withCredentialSummary(attemptResultFromTransport(startedAt, definition.HTTPMethod, transportResult, transportErr), resolution)
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
	case errors.Is(err, myerrors.ErrIntegrationExecutionInputTooLarge):
		return "execution_input_too_large"
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
	identity, err := e.credentials.GetRuntimeCredentialIdentity(ctx, *definition.CredentialID)
	if err != nil || identity.ID != *definition.CredentialID || identity.ExternalSystemID != system.Id {
		return model.ExternalSystem{}, model.InterfaceDefinition{}, repository.CredentialRuntimeIdentity{}, myerrors.ErrIntegrationConfigurationUnavailable
	}
	return system, definition, identity, nil
}

func (e *IntegrationExecutionEngine) completeClaim(ctx context.Context, claimed repository.ClaimedIntegrationExecution, result AttemptResult) error {
	status := model.IntegrationExecutionStatusFailed
	if result.Succeeded {
		status = model.IntegrationExecutionStatusSucceeded
	} else if result.RetryEligible && result.Certainty == model.IntegrationResultCertaintyConfirmed {
		status = model.IntegrationExecutionStatusRetryWaiting
	}
	_, err := e.executions.CompleteAttemptAndExecution(ctx, repository.IntegrationAttemptCompletion{
		ExecutionID: claimed.Execution.Id, AttemptID: claimed.Attempt.Id, AttemptNo: claimed.Attempt.AttemptNo,
		WorkerID: e.workerID, ExpectedRevision: claimed.Execution.Revision, ExecutionStatus: status,
		CompletedAt: result.CompletedAt, HTTPStatus: result.HTTPStatus, ResultSizeBytes: result.ResponseSize,
		ResultHash: result.ResponseHash, ResultSummary: result.SafeMessage, ErrorCategory: result.ErrorCategory,
		ErrorCode: result.ReasonCode, ResultCertainty: result.Certainty, ResponseContentType: result.ContentType,
		CredentialCode: result.CredentialCode, CredentialVersion: result.CredentialVersion,
		CredentialFingerprintSummary: result.CredentialFingerprintSummary,
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
	return myerrors.ErrIntegrationExecutionCompleteFailed
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
	now := e.now()
	values, err := e.executions.FindExpiredRunningExecutions(ctx, now, limit)
	if err != nil {
		return 0, myerrors.ErrIntegrationLeaseRecoveryFailed
	}
	recovered := 0
	for _, value := range values {
		changed, recoverErr := e.executions.RecoverExpiredExecution(ctx, repository.ExpiredExecutionRecovery{ExecutionID: value.Id, RecoveredAt: now})
		if recoverErr != nil && !errors.Is(recoverErr, repository.ErrIntegrationExecutionLeaseLost) && !errors.Is(recoverErr, repository.ErrIntegrationAttemptAlreadyCompleted) {
			return recovered, myerrors.ErrIntegrationLeaseRecoveryFailed
		}
		if changed {
			recovered++
		}
	}
	return recovered, nil
}

func (e *IntegrationExecutionEngine) failureResult(startedAt time.Time, category, reason, message, certainty string, retryEligible bool) AttemptResult {
	completedAt := e.now()
	return AttemptResult{StartedAt: startedAt, CompletedAt: completedAt, Duration: completedAt.Sub(startedAt),
		ErrorCategory: category, ReasonCode: reason, SafeMessage: message, Certainty: certainty, RetryEligible: retryEligible}
}

func attemptResultFromTransport(startedAt time.Time, method string, transportResult TransportResult, transportErr error) AttemptResult {
	completedAt := time.Now()
	result := AttemptResult{HTTPStatus: optionalHTTPStatus(transportResult.StatusCode), ResponseSize: transportResult.ResponseSize,
		ResponseHash: transportResult.ResponseHash, ContentType: transportResult.ContentType, StartedAt: startedAt,
		CompletedAt: completedAt, Duration: completedAt.Sub(startedAt), Certainty: string(transportResult.Determinacy)}
	if result.Certainty == "" {
		result.Certainty = model.IntegrationResultCertaintyUnknown
	}
	if transportErr == nil && transportResult.ErrorCategory == "" && transportResult.StatusCode >= http.StatusOK && transportResult.StatusCode < http.StatusMultipleChoices {
		result.Succeeded = true
		result.Certainty = model.IntegrationResultCertaintyConfirmed
		result.SafeMessage = "远端调用成功"
		return result
	}
	result.ErrorCategory, result.ReasonCode, result.SafeMessage = mapTransportError(transportResult.ErrorCategory, transportErr)
	if result.ErrorCategory == model.IntegrationErrorCategoryRemote && transportResult.StatusCode == http.StatusTooManyRequests ||
		(result.ErrorCategory == model.IntegrationErrorCategoryRemote && transportResult.StatusCode >= http.StatusInternalServerError) {
		result.RetryEligible = result.Certainty == model.IntegrationResultCertaintyConfirmed
	}
	if (result.ErrorCategory == model.IntegrationErrorCategoryNetwork || result.ErrorCategory == model.IntegrationErrorCategoryTimeout) &&
		isReadOnlyMethod(method) && result.Certainty == model.IntegrationResultCertaintyUnknown {
		result.RetryEligible = true
	}
	// 写操作在结果 unknown 时不能证明远端幂等，绝不标记为可自动重试。
	if result.Certainty == model.IntegrationResultCertaintyUnknown && !isReadOnlyMethod(method) {
		result.RetryEligible = false
	}
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
	case TransportErrorNetwork, TransportErrorTLS:
		return model.IntegrationErrorCategoryNetwork, "network_error", "集成远程调用失败"
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

func isReadOnlyMethod(method string) bool {
	return strings.EqualFold(strings.TrimSpace(method), http.MethodGet)
}

func (e *IntegrationExecutionEngine) logAttempt(ctx context.Context, claimed repository.ClaimedIntegrationExecution, result AttemptResult, phase string) {
	correlation := audit.GetCorrelationIDs(ctx)
	zap.L().Info("integration execution attempt",
		zap.String("request_id", correlation.RequestID), zap.String("trace_id", correlation.TraceID),
		zap.String("execution_no", claimed.Execution.ExecutionNo), zap.Int("attempt_no", claimed.Attempt.AttemptNo),
		zap.String("system_code", claimed.Execution.ExternalSystemCode), zap.String("interface_code", claimed.Execution.InterfaceCode),
		zap.String("worker_id", e.workerID), zap.String("phase", phase), zap.Int("http_status", transportStatus(result.HTTPStatus)),
		zap.Duration("duration", result.Duration), zap.String("error_category", result.ErrorCategory),
		zap.String("result_certainty", result.Certainty), zap.Bool("retry_eligible", result.RetryEligible),
	)
}

func transportStatus(status *int) int {
	if status == nil {
		return 0
	}
	return *status
}
