package integration

import (
	myerrors "backend/internal/errors"
	"backend/internal/integration/retrycontract"
	"backend/model"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	RemoteIdempotencyNone             = retrycontract.RemoteIdempotencyNone
	RemoteIdempotencySafeMethod       = retrycontract.RemoteIdempotencySafeMethod
	RemoteIdempotencyIdempotentMethod = retrycontract.RemoteIdempotencyIdempotentMethod
	RemoteIdempotencyKeyHeader        = retrycontract.RemoteIdempotencyKeyHeader
	RemoteIdempotencyHeaderName       = retrycontract.RemoteIdempotencyHeaderName

	RequestProgressNotSent          = "not_sent"
	RequestProgressSentUnknown      = "sent_unknown"
	RequestProgressResponseReceived = "response_received"

	RetryAfterSourceNone            = "none"
	RetryAfterSourceLocal           = "local"
	RetryAfterSourceHTTPDelta       = "http_delta"
	RetryAfterSourceHTTPDate        = "http_date"
	RetryAfterSourceIgnored         = "ignored"
	RetryAfterSourceInvalidFallback = "invalid_fallback"

	RetryReasonAllowed                  = "retry_allowed"
	RetryReasonAttemptsExhausted        = "retry_attempts_exhausted"
	RetryReasonWindowExpired            = "retry_window_expired"
	RetryReasonErrorNotAllowed          = "retry_error_not_allowed"
	RetryReasonHTTPStatusNotAllowed     = "retry_http_status_not_allowed"
	RetryReasonUnknownNotIdempotent     = "retry_unknown_not_idempotent"
	RetryReasonExecutionCancelled       = "retry_execution_cancelled"
	RetryReasonPolicyInvalid            = "retry_policy_invalid"
	RetryReasonAfterWindowExceeded      = "retry_after_window_exceeded"
	RetryReasonMethodNotSupported       = "retry_method_not_supported"
	RetryReasonRemoteIdempotencyMissing = "retry_remote_idempotency_missing"
	RetryReasonAfterInvalid             = "retry_after_invalid"
	RetryReasonScheduleInvalid          = "retry_schedule_invalid"

	minimumRetryDelay = time.Second
)

// RetryDecisionInput 只能由 Engine 根据已冻结配置和 Attempt 技术事实构造。
type RetryDecisionInput struct {
	PolicySnapshot          RetryPolicySnapshot
	HTTPMethod              string
	AttemptNo               int
	ErrorCategory           string
	ReasonCode              string
	HTTPStatus              int
	ResultDeterminacy       string
	RequestProgress         string
	RetryAfterRaw           string
	FirstAttemptAt          time.Time
	CurrentTime             time.Time
	ExecutionStatus         string
	ExecutionCancelled      bool
	RemoteIdempotencyMode   string
	HasRemoteIdempotencyKey bool
}

// RetryDecision 是安全摘要，不携带 Payload、Header 原值或底层错误。
type RetryDecision struct {
	retryable         bool
	finalState        string
	reasonCode        string
	nextRetryAt       time.Time
	retryDelay        time.Duration
	attemptsRemaining int
	retryAfterSource  string
	determinacy       string
}

func (d RetryDecision) Retryable() bool           { return d.retryable }
func (d RetryDecision) FinalState() string        { return d.finalState }
func (d RetryDecision) ReasonCode() string        { return d.reasonCode }
func (d RetryDecision) NextRetryAt() time.Time    { return d.nextRetryAt }
func (d RetryDecision) RetryDelay() time.Duration { return d.retryDelay }
func (d RetryDecision) AttemptsRemaining() int    { return d.attemptsRemaining }
func (d RetryDecision) RetryAfterSource() string  { return d.retryAfterSource }
func (d RetryDecision) Determinacy() string       { return d.determinacy }

// RandomSource 使 full jitter 可确定测试；实现必须支持并发调用。
type RandomSource interface {
	Int63n(maxExclusive int64) int64
}

type lockedRandomSource struct {
	mu  sync.Mutex
	rng *rand.Rand
}

func NewRetryRandomSource(seed int64) RandomSource {
	return &lockedRandomSource{rng: rand.New(rand.NewSource(seed))}
}

func (r *lockedRandomSource) Int63n(maxExclusive int64) int64 {
	if maxExclusive <= 1 {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Int63n(maxExclusive)
}

type RetryDecisionService struct {
	random RandomSource
}

func NewRetryDecisionService(random RandomSource) (*RetryDecisionService, error) {
	if random == nil {
		return nil, myerrors.ErrIntegrationRetryPolicyInvalid
	}
	return &RetryDecisionService{random: random}, nil
}

// Decide 返回正常的“不可重试”决定；只有快照或调度输入损坏才返回 Go error。
func (s *RetryDecisionService) Decide(input RetryDecisionInput) (RetryDecision, error) {
	if s == nil || s.random == nil {
		return RetryDecision{}, myerrors.ErrIntegrationRetryPolicyInvalid
	}
	if err := validateRetryDecisionInput(input); err != nil {
		return RetryDecision{}, err
	}
	remaining := max(0, input.PolicySnapshot.MaxAttempts-input.AttemptNo)
	denied := func(reason string) RetryDecision {
		return RetryDecision{finalState: model.IntegrationExecutionStatusFailed, reasonCode: reason,
			attemptsRemaining: remaining, retryAfterSource: RetryAfterSourceNone, determinacy: input.ResultDeterminacy}
	}
	if input.ExecutionCancelled || input.ExecutionStatus != model.IntegrationExecutionStatusRunning {
		return denied(RetryReasonExecutionCancelled), nil
	}
	if isHardRetryDenied(input.ReasonCode, input.HTTPStatus) {
		return denied(RetryReasonErrorNotAllowed), nil
	}
	if !supportedRetryMethod(input.HTTPMethod) {
		return denied(RetryReasonMethodNotSupported), nil
	}
	if input.ResultDeterminacy == model.IntegrationResultCertaintyUnknown {
		if reason := unknownIdempotencyDenial(input); reason != "" {
			return denied(reason), nil
		}
	}
	if remaining == 0 {
		return denied(RetryReasonAttemptsExhausted), nil
	}
	windowEnd := input.FirstAttemptAt.Add(time.Duration(input.PolicySnapshot.RetryWindowMs) * time.Millisecond)
	if !input.CurrentTime.Before(windowEnd) {
		return denied(RetryReasonWindowExpired), nil
	}
	if !containsString(input.PolicySnapshot.RetryableErrorCategories, input.ErrorCategory) {
		return denied(RetryReasonErrorNotAllowed), nil
	}
	if input.ErrorCategory == model.IntegrationErrorCategoryRemote {
		if input.RequestProgress != RequestProgressResponseReceived || !isPlatformRetryHTTPStatus(input.HTTPStatus) || !containsInt(input.PolicySnapshot.RetryableHTTPStatuses, input.HTTPStatus) {
			return denied(RetryReasonHTTPStatusNotAllowed), nil
		}
	} else if input.ErrorCategory != model.IntegrationErrorCategoryNetwork && input.ErrorCategory != model.IntegrationErrorCategoryTimeout {
		return denied(RetryReasonErrorNotAllowed), nil
	} else if input.ErrorCategory == model.IntegrationErrorCategoryNetwork && input.RequestProgress == RequestProgressResponseReceived {
		return denied(RetryReasonErrorNotAllowed), nil
	}

	baseDelay, err := calculateRetryBackoff(input.PolicySnapshot, input.AttemptNo)
	if err != nil {
		return RetryDecision{}, err
	}
	localDelay := baseDelay
	if input.PolicySnapshot.JitterType == model.RetryJitterTypeFull {
		localDelay = time.Duration(s.random.Int63n(baseDelay.Milliseconds()+1)) * time.Millisecond
		if localDelay < minimumRetryDelay {
			localDelay = minimumRetryDelay
		}
	}
	delay, source, reason := applyRetryAfter(input, localDelay)
	if reason == RetryReasonScheduleInvalid {
		decision := denied(reason)
		decision.retryAfterSource = source
		return decision, nil
	}
	next := input.CurrentTime.Add(delay)
	if !next.Before(windowEnd) {
		decision := denied(RetryReasonWindowExpired)
		if source == RetryAfterSourceHTTPDate || source == RetryAfterSourceHTTPDelta {
			decision.reasonCode = RetryReasonAfterWindowExceeded
		}
		decision.retryAfterSource = source
		return decision, nil
	}
	return RetryDecision{
		retryable: true, finalState: model.IntegrationExecutionStatusRetryWaiting, reasonCode: reason,
		nextRetryAt: next.UTC(), retryDelay: delay, attemptsRemaining: remaining,
		retryAfterSource: source, determinacy: input.ResultDeterminacy,
	}, nil
}

func validateRetryDecisionInput(input RetryDecisionInput) error {
	if err := ValidateRetryPolicySnapshot(input.PolicySnapshot); err != nil {
		return err
	}
	if input.AttemptNo < 1 || input.FirstAttemptAt.IsZero() || input.CurrentTime.IsZero() || input.CurrentTime.Before(input.FirstAttemptAt) ||
		(input.ResultDeterminacy != model.IntegrationResultCertaintyConfirmed && input.ResultDeterminacy != model.IntegrationResultCertaintyUnknown) ||
		(input.RequestProgress != RequestProgressNotSent && input.RequestProgress != RequestProgressSentUnknown && input.RequestProgress != RequestProgressResponseReceived) ||
		input.RemoteIdempotencyMode != input.PolicySnapshot.IdempotencyMode {
		return myerrors.ErrIntegrationRetryScheduleInvalid
	}
	return nil
}

func ValidateRetryPolicySnapshot(value RetryPolicySnapshot) error {
	return retrycontract.Validate(value)
}

func calculateRetryBackoff(snapshot RetryPolicySnapshot, attemptNo int) (time.Duration, error) {
	return retrycontract.CalculateBackoff(snapshot, attemptNo)
}

func applyRetryAfter(input RetryDecisionInput, local time.Duration) (time.Duration, string, string) {
	raw := strings.TrimSpace(input.RetryAfterRaw)
	if raw == "" {
		return local, RetryAfterSourceLocal, RetryReasonAllowed
	}
	if input.ErrorCategory != model.IntegrationErrorCategoryRemote || !isPlatformRetryHTTPStatus(input.HTTPStatus) {
		return local, RetryAfterSourceIgnored, RetryReasonAllowed
	}
	if !input.PolicySnapshot.RespectRetryAfter {
		return local, RetryAfterSourceIgnored, RetryReasonAllowed
	}
	delay, source, valid := parseRetryAfter(raw, input.CurrentTime)
	if !valid {
		return local, RetryAfterSourceInvalidFallback, RetryReasonAfterInvalid
	}
	maxDelay := time.Duration(input.PolicySnapshot.MaxDelayMs) * time.Millisecond
	if delay > maxDelay {
		return 0, source, RetryReasonScheduleInvalid
	}
	if delay > local {
		return delay, source, RetryReasonAllowed
	}
	return local, source, RetryReasonAllowed
}

func parseRetryAfter(raw string, current time.Time) (time.Duration, string, bool) {
	if strings.ContainsAny(raw, "\r\n") {
		return 0, RetryAfterSourceInvalidFallback, false
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if seconds < 0 || seconds > math.MaxInt64/int64(time.Second) {
			return 0, RetryAfterSourceInvalidFallback, false
		}
		return time.Duration(seconds) * time.Second, RetryAfterSourceHTTPDelta, true
	}
	value, err := http.ParseTime(raw)
	if err != nil {
		return 0, RetryAfterSourceInvalidFallback, false
	}
	delay := value.Sub(current)
	if delay < 0 {
		delay = 0
	}
	return delay, RetryAfterSourceHTTPDate, true
}

func unknownIdempotencyDenial(input RetryDecisionInput) string {
	switch strings.ToUpper(strings.TrimSpace(input.HTTPMethod)) {
	case http.MethodGet:
		return ""
	case http.MethodPut, http.MethodDelete:
		if input.RemoteIdempotencyMode == RemoteIdempotencyIdempotentMethod {
			return ""
		}
		return RetryReasonUnknownNotIdempotent
	case http.MethodPost, http.MethodPatch:
		if input.RemoteIdempotencyMode != RemoteIdempotencyKeyHeader {
			return RetryReasonUnknownNotIdempotent
		}
		if !input.HasRemoteIdempotencyKey {
			return RetryReasonRemoteIdempotencyMissing
		}
		return ""
	default:
		return RetryReasonMethodNotSupported
	}
}

func isHardRetryDenied(reason string, status int) bool {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return true
	}
	reason = strings.ToLower(strings.TrimSpace(reason))
	for _, prefix := range []string{
		"invalid_config", "configuration_unavailable", "integration_execution_runtime_incompatible",
		"transport_configuration_invalid", "ssrf_rejected", "tls_error", "credential_", "execution_input_",
		"response_too_large", "response_invalid", "unsupported_content_type", "cancelled", "internal_error",
		"worker_panic_recovered", "transport_failed",
	} {
		if strings.HasPrefix(reason, prefix) {
			return true
		}
	}
	return false
}

func supportedRetryMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isPlatformRetryHTTPStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validRemoteIdempotency(mode, header string) bool {
	return retrycontract.ValidRemoteIdempotency(mode, header)
}

// ValidRemoteIdempotencyContract 校验 InterfaceDefinition 的 Method 与远端幂等声明组合。
func ValidRemoteIdempotencyContract(method, mode, header string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	mode = strings.ToLower(strings.TrimSpace(mode))
	header = strings.TrimSpace(header)
	switch method {
	case model.InterfaceMethodGET:
		return mode == RemoteIdempotencySafeMethod && header == ""
	case model.InterfaceMethodPUT, model.InterfaceMethodDELETE:
		return (mode == RemoteIdempotencyNone || mode == RemoteIdempotencyIdempotentMethod) && header == ""
	case model.InterfaceMethodPOST, model.InterfaceMethodPATCH:
		return mode == RemoteIdempotencyNone && header == "" ||
			mode == RemoteIdempotencyKeyHeader && header == RemoteIdempotencyHeaderName
	default:
		return false
	}
}
