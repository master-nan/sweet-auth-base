package integration

import (
	myerrors "backend/internal/errors"
	"backend/model"
	"encoding/json"
	"sort"
	"strconv"
)

const RetryPolicySnapshotVersion = 1

// RetryPolicySnapshot 是 Execution 创建时冻结的最小策略值对象。
type RetryPolicySnapshot struct {
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

type RetryPolicySnapshotOptions struct {
	IdempotencyMode         string
	RemoteIdempotencyHeader string
}

func BuildRetryPolicySnapshot(value model.RetryPolicy, options ...RetryPolicySnapshotOptions) ([]byte, error) {
	var categories []string
	if err := json.Unmarshal(value.RetryableErrorCategories, &categories); err != nil {
		return nil, err
	}
	var statuses []int
	if err := json.Unmarshal(value.RetryableHTTPStatuses, &statuses); err != nil {
		return nil, err
	}
	sort.Strings(categories)
	sort.Ints(statuses)
	configuration := RetryPolicySnapshotOptions{IdempotencyMode: RemoteIdempotencyNone}
	if len(options) > 0 {
		configuration = options[0]
	}
	snapshot := RetryPolicySnapshot{
		Version: RetryPolicySnapshotVersion, PolicyCode: value.PolicyCode, PolicyVersion: value.Version,
		MaxAttempts: value.MaxAttempts, InitialDelayMs: value.InitialDelayMs, MaxDelayMs: value.MaxDelayMs,
		BackoffType: value.BackoffType, BackoffMultiplier: strconv.FormatFloat(value.BackoffMultiplier, 'f', -1, 64),
		JitterType: value.JitterType, JitterRatio: strconv.FormatFloat(value.JitterRatio, 'f', -1, 64),
		RetryWindowMs: value.RetryWindowMs, RetryableErrorCategories: categories,
		RetryableHTTPStatuses: statuses, RespectRetryAfter: value.RespectRetryAfter,
		IdempotencyMode: configuration.IdempotencyMode, RemoteIdempotencyHeader: configuration.RemoteIdempotencyHeader,
	}
	if err := ValidateRetryPolicySnapshot(snapshot); err != nil {
		return nil, err
	}
	return json.Marshal(snapshot)
}

func ParseRetryPolicySnapshot(raw []byte) (RetryPolicySnapshot, error) {
	var snapshot RetryPolicySnapshot
	if len(raw) == 0 || !json.Valid(raw) || json.Unmarshal(raw, &snapshot) != nil {
		return RetryPolicySnapshot{}, myerrors.ErrIntegrationRetrySnapshotInvalid
	}
	if err := ValidateRetryPolicySnapshot(snapshot); err != nil {
		return RetryPolicySnapshot{}, err
	}
	return snapshot, nil
}
