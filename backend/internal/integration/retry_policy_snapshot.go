package integration

import (
	"backend/internal/integration/retrycontract"
	"backend/model"
	"encoding/json"
	"sort"
	"strconv"
)

const RetryPolicySnapshotVersion = retrycontract.SnapshotVersion

// RetryPolicySnapshot 是 Execution 创建时冻结的最小策略值对象。
type RetryPolicySnapshot = retrycontract.Snapshot

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
	return retrycontract.Parse(raw)
}
