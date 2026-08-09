package model

import "gorm.io/datatypes"

const (
	RetryPolicyStatusDraft    = "draft"
	RetryPolicyStatusEnabled  = "enabled"
	RetryPolicyStatusDisabled = "disabled"

	RetryBackoffTypeFixed       = "fixed"
	RetryBackoffTypeExponential = "exponential"

	RetryJitterTypeNone = "none"
	RetryJitterTypeFull = "full"
)

// RetryPolicy 描述一个版本化的集成重试策略，不参与运行时调度和 HTTP 执行。
type RetryPolicy struct {
	Basic

	PolicyCode               string         `gorm:"size:64;not null;uniqueIndex:uni_integration_retry_policy_identity,priority:1;index:idx_integration_retry_policy_code" json:"policy_code"`
	PolicyName               string         `gorm:"size:128;not null;index:idx_integration_retry_policy_name" json:"policy_name"`
	Version                  int            `gorm:"not null;uniqueIndex:uni_integration_retry_policy_identity,priority:2" json:"version"`
	Description              string         `gorm:"size:512" json:"description"`
	Status                   string         `gorm:"size:16;not null;default:draft;index:idx_integration_retry_policy_status" json:"status"`
	MaxAttempts              int            `gorm:"not null;default:3" json:"max_attempts"`
	InitialDelayMs           int64          `gorm:"not null;default:5000" json:"initial_delay_ms"`
	MaxDelayMs               int64          `gorm:"not null;default:300000" json:"max_delay_ms"`
	BackoffType              string         `gorm:"size:16;not null;default:exponential" json:"backoff_type"`
	BackoffMultiplier        float64        `gorm:"type:numeric(6,3);not null;default:2" json:"backoff_multiplier"`
	JitterType               string         `gorm:"size:16;not null;default:full" json:"jitter_type"`
	JitterRatio              float64        `gorm:"type:numeric(4,3);not null;default:1" json:"jitter_ratio"`
	RetryWindowMs            int64          `gorm:"not null;default:86400000" json:"retry_window_ms"`
	RetryableErrorCategories datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"retryable_error_categories"`
	RetryableHTTPStatuses    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"retryable_http_statuses"`
	RespectRetryAfter        bool           `gorm:"not null;default:true" json:"respect_retry_after"`
	Revision                 int            `gorm:"not null;default:1" json:"revision"`
}

func (RetryPolicy) TableName() string {
	return "integration_retry_policy"
}
