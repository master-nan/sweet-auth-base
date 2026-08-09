package response

import (
	"backend/model"
	"encoding/json"
)

type RetryPolicyListRes struct {
	Id             int              `json:"id"`
	PolicyCode     string           `json:"policy_code"`
	PolicyName     string           `json:"policy_name"`
	Version        int              `json:"version"`
	Status         string           `json:"status"`
	MaxAttempts    int              `json:"max_attempts"`
	BackoffType    string           `json:"backoff_type"`
	InitialDelayMs int64            `json:"initial_delay_ms"`
	MaxDelayMs     int64            `json:"max_delay_ms"`
	RetryWindowMs  int64            `json:"retry_window_ms"`
	Revision       int              `json:"revision"`
	GmtModify      model.CustomTime `json:"gmt_modify"`
}

type RetryPolicyDetailRes struct {
	RetryPolicyListRes
	Description              string           `json:"description"`
	BackoffMultiplier        float64          `json:"backoff_multiplier"`
	JitterType               string           `json:"jitter_type"`
	JitterRatio              float64          `json:"jitter_ratio"`
	RetryableErrorCategories []string         `json:"retryable_error_categories"`
	RetryableHTTPStatuses    []int            `json:"retryable_http_statuses"`
	RespectRetryAfter        bool             `json:"respect_retry_after"`
	GmtCreate                model.CustomTime `json:"gmt_create"`
}

func NewRetryPolicyListRes(value model.RetryPolicy) RetryPolicyListRes {
	return RetryPolicyListRes{
		Id: value.Id, PolicyCode: value.PolicyCode, PolicyName: value.PolicyName, Version: value.Version,
		Status: value.Status, MaxAttempts: value.MaxAttempts, BackoffType: value.BackoffType,
		InitialDelayMs: value.InitialDelayMs, MaxDelayMs: value.MaxDelayMs, RetryWindowMs: value.RetryWindowMs,
		Revision: value.Revision, GmtModify: value.GmtModify,
	}
}

func NewRetryPolicyDetailRes(value model.RetryPolicy) RetryPolicyDetailRes {
	result := RetryPolicyDetailRes{
		RetryPolicyListRes: NewRetryPolicyListRes(value), Description: value.Description,
		BackoffMultiplier: value.BackoffMultiplier, JitterType: value.JitterType, JitterRatio: value.JitterRatio,
		RespectRetryAfter: value.RespectRetryAfter, GmtCreate: value.GmtCreate,
	}
	_ = json.Unmarshal(value.RetryableErrorCategories, &result.RetryableErrorCategories)
	_ = json.Unmarshal(value.RetryableHTTPStatuses, &result.RetryableHTTPStatuses)
	if result.RetryableErrorCategories == nil {
		result.RetryableErrorCategories = []string{}
	}
	if result.RetryableHTTPStatuses == nil {
		result.RetryableHTTPStatuses = []int{}
	}
	return result
}
