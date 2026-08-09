package retrycontract

import (
	myerrors "backend/internal/errors"
	"backend/model"
	"encoding/json"
	"math"
	"math/big"
	"strings"
	"time"
)

const (
	SnapshotVersion = 1

	RemoteIdempotencyNone             = "none"
	RemoteIdempotencySafeMethod       = "safe_method"
	RemoteIdempotencyIdempotentMethod = "idempotent_method"
	RemoteIdempotencyKeyHeader        = "remote_key_header"
	RemoteIdempotencyHeaderName       = "Idempotency-Key"

	ReasonPolicyInvalid     = "retry_policy_invalid"
	ReasonAttemptsExhausted = "retry_attempts_exhausted"
	ReasonWindowExpired     = "retry_window_expired"
)

// Snapshot 是 Execution 创建时冻结的最小重试策略契约。
type Snapshot struct {
	Version                  int      `json:"version"`
	PolicyCode               string   `json:"policy_code"`
	PolicyVersion            int      `json:"policy_version"`
	MaxAttempts              int      `json:"max_attempts"`
	InitialDelayMs           int64    `json:"initial_delay_ms"`
	MaxDelayMs               int64    `json:"max_delay_ms"`
	BackoffType              string   `json:"backoff_type"`
	BackoffMultiplier        string   `json:"backoff_multiplier"`
	JitterType               string   `json:"jitter_type"`
	JitterRatio              string   `json:"jitter_ratio"`
	RetryWindowMs            int64    `json:"retry_window_ms"`
	RetryableErrorCategories []string `json:"retryable_error_categories"`
	RetryableHTTPStatuses    []int    `json:"retryable_http_statuses"`
	RespectRetryAfter        bool     `json:"respect_retry_after"`
	IdempotencyMode          string   `json:"idempotency_mode"`
	RemoteIdempotencyHeader  string   `json:"remote_idempotency_header"`
}

func Parse(raw []byte) (Snapshot, error) {
	var snapshot Snapshot
	if len(raw) == 0 || !json.Valid(raw) || json.Unmarshal(raw, &snapshot) != nil {
		return Snapshot{}, myerrors.ErrIntegrationRetrySnapshotInvalid
	}
	if err := Validate(snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func Validate(value Snapshot) error {
	if value.Version != SnapshotVersion || strings.TrimSpace(value.PolicyCode) == "" || value.PolicyVersion < 1 ||
		value.MaxAttempts < 1 || value.MaxAttempts > 10 || value.InitialDelayMs < 1000 || value.InitialDelayMs > 3600000 ||
		value.MaxDelayMs < value.InitialDelayMs || value.MaxDelayMs > 86400000 || value.RetryWindowMs < 60000 || value.RetryWindowMs > 604800000 {
		return myerrors.ErrIntegrationRetrySnapshotInvalid
	}
	multiplier, ok := new(big.Rat).SetString(value.BackoffMultiplier)
	if !ok {
		return myerrors.ErrIntegrationRetrySnapshotInvalid
	}
	switch value.BackoffType {
	case model.RetryBackoffTypeFixed:
		if multiplier.Cmp(big.NewRat(1, 1)) != 0 {
			return myerrors.ErrIntegrationRetrySnapshotInvalid
		}
	case model.RetryBackoffTypeExponential:
		if multiplier.Cmp(big.NewRat(11, 10)) < 0 || multiplier.Cmp(big.NewRat(4, 1)) > 0 {
			return myerrors.ErrIntegrationRetrySnapshotInvalid
		}
	default:
		return myerrors.ErrIntegrationRetrySnapshotInvalid
	}
	if value.JitterType == model.RetryJitterTypeNone && value.JitterRatio != "0" ||
		value.JitterType == model.RetryJitterTypeFull && value.JitterRatio != "1" ||
		value.JitterType != model.RetryJitterTypeNone && value.JitterType != model.RetryJitterTypeFull {
		return myerrors.ErrIntegrationRetrySnapshotInvalid
	}
	for _, category := range value.RetryableErrorCategories {
		if category != model.IntegrationErrorCategoryNetwork && category != model.IntegrationErrorCategoryTimeout && category != model.IntegrationErrorCategoryRemote {
			return myerrors.ErrIntegrationRetrySnapshotInvalid
		}
	}
	for _, status := range value.RetryableHTTPStatuses {
		if status != 429 && status != 502 && status != 503 && status != 504 {
			return myerrors.ErrIntegrationRetrySnapshotInvalid
		}
	}
	if !ValidRemoteIdempotency(value.IdempotencyMode, value.RemoteIdempotencyHeader) {
		return myerrors.ErrIntegrationRetrySnapshotInvalid
	}
	var requiredWindow int64
	for attempt := 1; attempt < value.MaxAttempts; attempt++ {
		delay, err := CalculateBackoff(value, attempt)
		if err != nil || delay.Milliseconds() > math.MaxInt64-requiredWindow {
			return myerrors.ErrIntegrationRetrySnapshotInvalid
		}
		requiredWindow += delay.Milliseconds()
	}
	if value.RetryWindowMs < requiredWindow {
		return myerrors.ErrIntegrationRetrySnapshotInvalid
	}
	return nil
}

func CalculateBackoff(snapshot Snapshot, attemptNo int) (time.Duration, error) {
	if attemptNo < 1 {
		return 0, myerrors.ErrIntegrationRetryScheduleInvalid
	}
	value := big.NewRat(snapshot.InitialDelayMs, 1)
	if snapshot.BackoffType == model.RetryBackoffTypeExponential {
		multiplier, ok := new(big.Rat).SetString(snapshot.BackoffMultiplier)
		if !ok {
			return 0, myerrors.ErrIntegrationRetrySnapshotInvalid
		}
		for index := 1; index < attemptNo; index++ {
			value.Mul(value, multiplier)
			if value.Cmp(big.NewRat(snapshot.MaxDelayMs, 1)) >= 0 {
				return time.Duration(snapshot.MaxDelayMs) * time.Millisecond, nil
			}
		}
	}
	numerator, denominator := value.Num(), value.Denom()
	rounded := new(big.Int).Add(numerator, new(big.Int).Sub(denominator, big.NewInt(1)))
	rounded.Quo(rounded, denominator)
	if !rounded.IsInt64() || rounded.Int64() > snapshot.MaxDelayMs || rounded.Int64() > math.MaxInt64/int64(time.Millisecond) {
		return time.Duration(snapshot.MaxDelayMs) * time.Millisecond, nil
	}
	return time.Duration(rounded.Int64()) * time.Millisecond, nil
}

func ValidRemoteIdempotency(mode, header string) bool {
	switch mode {
	case RemoteIdempotencyNone, RemoteIdempotencySafeMethod, RemoteIdempotencyIdempotentMethod:
		return strings.TrimSpace(header) == ""
	case RemoteIdempotencyKeyHeader:
		return strings.EqualFold(strings.TrimSpace(header), RemoteIdempotencyHeaderName)
	default:
		return false
	}
}
