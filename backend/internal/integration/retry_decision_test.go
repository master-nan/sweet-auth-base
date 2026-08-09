package integration

import (
	"backend/model"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	myerrors "backend/internal/errors"
)

type fixedRandomSource struct{ value int64 }

func (r fixedRandomSource) Int63n(maxExclusive int64) int64 {
	if maxExclusive <= 1 || r.value < 0 {
		return 0
	}
	if r.value >= maxExclusive {
		return maxExclusive - 1
	}
	return r.value
}

func TestRetryDecisionAttemptAndHTTPRules(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	service := mustRetryDecisionService(t, fixedRandomSource{})
	for _, test := range []struct {
		name      string
		attempt   int
		status    int
		category  string
		retryable bool
		reason    string
		remaining int
	}{
		{name: "single attempt", attempt: 1, status: 503, category: model.IntegrationErrorCategoryRemote, reason: RetryReasonAttemptsExhausted},
		{name: "attempt one", attempt: 1, status: 503, category: model.IntegrationErrorCategoryRemote, retryable: true, reason: RetryReasonAllowed, remaining: 2},
		{name: "attempt two", attempt: 2, status: 503, category: model.IntegrationErrorCategoryRemote, retryable: true, reason: RetryReasonAllowed, remaining: 1},
		{name: "attempt three", attempt: 3, status: 503, category: model.IntegrationErrorCategoryRemote, reason: RetryReasonAttemptsExhausted},
		{name: "429", attempt: 1, status: 429, category: model.IntegrationErrorCategoryRemote, retryable: true, reason: RetryReasonAllowed, remaining: 2},
		{name: "502", attempt: 1, status: 502, category: model.IntegrationErrorCategoryRemote, retryable: true, reason: RetryReasonAllowed, remaining: 2},
		{name: "504", attempt: 1, status: 504, category: model.IntegrationErrorCategoryRemote, retryable: true, reason: RetryReasonAllowed, remaining: 2},
		{name: "500", attempt: 1, status: 500, category: model.IntegrationErrorCategoryRemote, reason: RetryReasonHTTPStatusNotAllowed, remaining: 2},
		{name: "404", attempt: 1, status: 404, category: model.IntegrationErrorCategoryRemote, reason: RetryReasonHTTPStatusNotAllowed, remaining: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := retryDecisionSnapshot()
			if test.name == "single attempt" {
				snapshot.MaxAttempts = 1
			}
			decision, err := service.Decide(retryDecisionInput(snapshot, now, test.attempt, test.category, test.status))
			if err != nil || decision.Retryable() != test.retryable || decision.ReasonCode() != test.reason || decision.AttemptsRemaining() != test.remaining {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
		})
	}
}

func TestRetryDecisionHardDenialsOverridePolicy(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	service := mustRetryDecisionService(t, fixedRandomSource{})
	for _, test := range []struct {
		reason string
		status int
	}{
		{reason: "invalid_config"}, {reason: "ssrf_rejected"}, {reason: "tls_error"}, {reason: "credential_expired"},
		{reason: "execution_input_hash_mismatch"}, {reason: "response_too_large"},
		{reason: "unsupported_content_type"}, {reason: "cancelled"}, {reason: "internal_error"},
		{reason: "remote_http_error", status: http.StatusUnauthorized},
		{reason: "remote_http_error", status: http.StatusForbidden},
	} {
		input := retryDecisionInput(retryDecisionSnapshot(), now, 1, model.IntegrationErrorCategoryRemote, test.status)
		input.ReasonCode = test.reason
		decision, err := service.Decide(input)
		if err != nil || decision.Retryable() || decision.ReasonCode() != RetryReasonErrorNotAllowed {
			t.Fatalf("reason=%s status=%d decision=%+v err=%v", test.reason, test.status, decision, err)
		}
	}
}

func TestRetryDecisionDeterminacyAndRemoteIdempotency(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	service := mustRetryDecisionService(t, fixedRandomSource{})
	for _, test := range []struct {
		name      string
		method    string
		mode      string
		hasKey    bool
		retryable bool
		reason    string
	}{
		{name: "unknown get", method: http.MethodGet, mode: RemoteIdempotencySafeMethod, retryable: true, reason: RetryReasonAllowed},
		{name: "unknown put explicit", method: http.MethodPut, mode: RemoteIdempotencyIdempotentMethod, retryable: true, reason: RetryReasonAllowed},
		{name: "unknown delete none", method: http.MethodDelete, mode: RemoteIdempotencyNone, reason: RetryReasonUnknownNotIdempotent},
		{name: "unknown post remote key", method: http.MethodPost, mode: RemoteIdempotencyKeyHeader, hasKey: true, retryable: true, reason: RetryReasonAllowed},
		{name: "unknown patch missing key", method: http.MethodPatch, mode: RemoteIdempotencyKeyHeader, reason: RetryReasonRemoteIdempotencyMissing},
		{name: "unknown post local key only", method: http.MethodPost, mode: RemoteIdempotencyNone, hasKey: true, reason: RetryReasonUnknownNotIdempotent},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := retryDecisionSnapshot()
			snapshot.IdempotencyMode = test.mode
			snapshot.RemoteIdempotencyHeader = ""
			if test.mode == RemoteIdempotencyKeyHeader {
				snapshot.RemoteIdempotencyHeader = RemoteIdempotencyHeaderName
			}
			input := retryDecisionInput(snapshot, now, 1, model.IntegrationErrorCategoryTimeout, 0)
			input.HTTPMethod = test.method
			input.ResultDeterminacy = model.IntegrationResultCertaintyUnknown
			input.RequestProgress = RequestProgressSentUnknown
			input.RemoteIdempotencyMode = test.mode
			input.HasRemoteIdempotencyKey = test.hasKey
			decision, err := service.Decide(input)
			if err != nil || decision.Retryable() != test.retryable || decision.ReasonCode() != test.reason {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
		})
	}
}

func TestRetryDecisionBackoffJitterAndRetryAfter(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	t.Run("fixed and exponential", func(t *testing.T) {
		service := mustRetryDecisionService(t, fixedRandomSource{})
		fixed := retryDecisionSnapshot()
		decision, err := service.Decide(retryDecisionInput(fixed, now, 2, model.IntegrationErrorCategoryRemote, 503))
		if err != nil || decision.RetryDelay() != 2*time.Second {
			t.Fatalf("fixed=%v err=%v", decision.RetryDelay(), err)
		}
		exponential := fixed
		exponential.BackoffType = model.RetryBackoffTypeExponential
		exponential.BackoffMultiplier = "1.5"
		exponential.MaxDelayMs = 2500
		exponential.RetryWindowMs = 60000
		decision, err = service.Decide(retryDecisionInput(exponential, now, 2, model.IntegrationErrorCategoryRemote, 503))
		if err != nil || decision.RetryDelay() != 2500*time.Millisecond {
			t.Fatalf("exponential=%v err=%v", decision.RetryDelay(), err)
		}
	})

	t.Run("full jitter deterministic bounds", func(t *testing.T) {
		snapshot := retryDecisionSnapshot()
		snapshot.JitterType = model.RetryJitterTypeFull
		snapshot.JitterRatio = "1"
		minimum, err := mustRetryDecisionService(t, fixedRandomSource{value: 0}).Decide(retryDecisionInput(snapshot, now, 1, model.IntegrationErrorCategoryRemote, 503))
		if err != nil || minimum.RetryDelay() != time.Second {
			t.Fatalf("minimum jitter=%v err=%v", minimum.RetryDelay(), err)
		}
		maximum, err := mustRetryDecisionService(t, fixedRandomSource{value: 2000}).Decide(retryDecisionInput(snapshot, now, 1, model.IntegrationErrorCategoryRemote, 503))
		if err != nil || maximum.RetryDelay() != 2*time.Second {
			t.Fatalf("maximum jitter=%v err=%v", maximum.RetryDelay(), err)
		}
	})

	t.Run("retry after formats and fallback", func(t *testing.T) {
		service := mustRetryDecisionService(t, fixedRandomSource{})
		for _, test := range []struct {
			name   string
			raw    string
			delay  time.Duration
			source string
			reason string
		}{
			{name: "delta", raw: "5", delay: 5 * time.Second, source: RetryAfterSourceHTTPDelta, reason: RetryReasonAllowed},
			{name: "date", raw: now.Add(4 * time.Second).Format(http.TimeFormat), delay: 4 * time.Second, source: RetryAfterSourceHTTPDate, reason: RetryReasonAllowed},
			{name: "lower than local", raw: "1", delay: 2 * time.Second, source: RetryAfterSourceHTTPDelta, reason: RetryReasonAllowed},
			{name: "invalid", raw: "later", delay: 2 * time.Second, source: RetryAfterSourceInvalidFallback, reason: RetryReasonAfterInvalid},
			{name: "negative", raw: "-1", delay: 2 * time.Second, source: RetryAfterSourceInvalidFallback, reason: RetryReasonAfterInvalid},
			{name: "overflow", raw: "999999999999999999999999", delay: 2 * time.Second, source: RetryAfterSourceInvalidFallback, reason: RetryReasonAfterInvalid},
		} {
			t.Run(test.name, func(t *testing.T) {
				input := retryDecisionInput(retryDecisionSnapshot(), now, 1, model.IntegrationErrorCategoryRemote, 503)
				input.RetryAfterRaw = test.raw
				decision, err := service.Decide(input)
				if err != nil || decision.RetryDelay() != test.delay || decision.RetryAfterSource() != test.source || decision.ReasonCode() != test.reason {
					t.Fatalf("decision=%+v err=%v", decision, err)
				}
			})
		}
		input := retryDecisionInput(retryDecisionSnapshot(), now, 1, model.IntegrationErrorCategoryRemote, 503)
		input.PolicySnapshot.RespectRetryAfter = false
		input.RetryAfterRaw = "5"
		decision, err := service.Decide(input)
		if err != nil || decision.RetryDelay() != 2*time.Second || decision.RetryAfterSource() != RetryAfterSourceIgnored {
			t.Fatalf("respect=false decision=%+v err=%v", decision, err)
		}
	})
}

func TestRetryDecisionWindowAndInvalidSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	service := mustRetryDecisionService(t, fixedRandomSource{})
	input := retryDecisionInput(retryDecisionSnapshot(), now, 1, model.IntegrationErrorCategoryRemote, 503)
	input.FirstAttemptAt = now.Add(-time.Duration(input.PolicySnapshot.RetryWindowMs) * time.Millisecond)
	decision, err := service.Decide(input)
	if err != nil || decision.ReasonCode() != RetryReasonWindowExpired {
		t.Fatalf("window decision=%+v err=%v", decision, err)
	}
	input = retryDecisionInput(retryDecisionSnapshot(), now, 1, model.IntegrationErrorCategoryRemote, 503)
	input.RetryAfterRaw = "120"
	decision, err = service.Decide(input)
	if err != nil || decision.Retryable() || decision.ReasonCode() != RetryReasonScheduleInvalid {
		t.Fatalf("retry after max decision=%+v err=%v", decision, err)
	}
	invalid := retryDecisionSnapshot()
	invalid.MaxAttempts = 0
	_, err = service.Decide(retryDecisionInput(invalid, now, 1, model.IntegrationErrorCategoryRemote, 503))
	if !errors.Is(err, myerrors.ErrIntegrationRetrySnapshotInvalid) {
		t.Fatalf("invalid snapshot err=%v", err)
	}
}

func TestRetryDecisionConcurrent(t *testing.T) {
	service := mustRetryDecisionService(t, NewRetryRandomSource(42))
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	snapshot := retryDecisionSnapshot()
	snapshot.JitterType = model.RetryJitterTypeFull
	snapshot.JitterRatio = "1"
	var wait sync.WaitGroup
	for index := 0; index < 64; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			decision, err := service.Decide(retryDecisionInput(snapshot, now, 1, model.IntegrationErrorCategoryRemote, 503))
			if err != nil || !decision.Retryable() || decision.RetryDelay() < time.Second || decision.RetryDelay() > 2*time.Second {
				t.Errorf("decision=%+v err=%v", decision, err)
			}
		}()
	}
	wait.Wait()
}

func retryDecisionSnapshot() RetryPolicySnapshot {
	return RetryPolicySnapshot{
		Version: RetryPolicySnapshotVersion, PolicyCode: "transient", PolicyVersion: 1,
		MaxAttempts: 3, InitialDelayMs: 2000, MaxDelayMs: 10000,
		BackoffType: model.RetryBackoffTypeFixed, BackoffMultiplier: "1",
		JitterType: model.RetryJitterTypeNone, JitterRatio: "0", RetryWindowMs: 60000,
		RetryableErrorCategories: []string{model.IntegrationErrorCategoryNetwork, model.IntegrationErrorCategoryRemote, model.IntegrationErrorCategoryTimeout},
		RetryableHTTPStatuses:    []int{429, 502, 503, 504}, RespectRetryAfter: true,
		IdempotencyMode: RemoteIdempotencySafeMethod,
	}
}

func retryDecisionInput(snapshot RetryPolicySnapshot, now time.Time, attempt int, category string, status int) RetryDecisionInput {
	return RetryDecisionInput{
		PolicySnapshot: snapshot, HTTPMethod: http.MethodGet, AttemptNo: attempt,
		ErrorCategory: category, ReasonCode: "remote_http_error", HTTPStatus: status,
		ResultDeterminacy: model.IntegrationResultCertaintyConfirmed, RequestProgress: RequestProgressResponseReceived,
		FirstAttemptAt: now.Add(-time.Second), CurrentTime: now, ExecutionStatus: model.IntegrationExecutionStatusRunning,
		RemoteIdempotencyMode: snapshot.IdempotencyMode,
	}
}

func mustRetryDecisionService(t *testing.T, random RandomSource) *RetryDecisionService {
	t.Helper()
	service, err := NewRetryDecisionService(random)
	if err != nil {
		t.Fatalf("new retry decision service: %v", err)
	}
	return service
}
