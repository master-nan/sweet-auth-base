package integration

import (
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
}

func BuildRetryPolicySnapshot(value model.RetryPolicy) ([]byte, error) {
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
	return json.Marshal(RetryPolicySnapshot{
		Version: RetryPolicySnapshotVersion, PolicyCode: value.PolicyCode, PolicyVersion: value.Version,
		MaxAttempts: value.MaxAttempts, InitialDelayMs: value.InitialDelayMs, MaxDelayMs: value.MaxDelayMs,
		BackoffType: value.BackoffType, BackoffMultiplier: strconv.FormatFloat(value.BackoffMultiplier, 'f', -1, 64),
		JitterType: value.JitterType, JitterRatio: strconv.FormatFloat(value.JitterRatio, 'f', -1, 64),
		RetryWindowMs: value.RetryWindowMs, RetryableErrorCategories: categories,
		RetryableHTTPStatuses: statuses, RespectRetryAfter: value.RespectRetryAfter,
	})
}
