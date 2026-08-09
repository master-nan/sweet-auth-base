package request

type RetryPolicyQueryReq struct {
	Page        int               `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num         int               `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order       Order             `form:"order" json:"order"`
	Expressions []ExpressionGroup `form:"expressions" json:"expressions" binding:"omitempty,max=8"`
	QuickQuery  *QuickQuery       `form:"quick_query" json:"quick_query"`
	Status      string            `form:"status" json:"status" binding:"omitempty,oneof=draft enabled disabled"`
	BackoffType string            `form:"backoff_type" json:"backoff_type" binding:"omitempty,oneof=fixed exponential"`
}

func (r RetryPolicyQueryReq) ToBasic() Basic {
	basic := Basic{Page: r.Page, Num: r.Num, Order: r.Order, Expressions: r.Expressions, QuickQuery: r.QuickQuery}
	filters := make(map[string]any, 2)
	if r.Status != "" {
		filters["status"] = r.Status
	}
	if r.BackoffType != "" {
		filters["backoff_type"] = r.BackoffType
	}
	if len(filters) > 0 {
		basic.Filters = filters
	}
	return basic
}

type RetryPolicyCreateReq struct {
	PolicyCode               string   `form:"policy_code" json:"policy_code" binding:"required,max=64"`
	PolicyName               string   `form:"policy_name" json:"policy_name" binding:"required,max=128"`
	Description              string   `form:"description" json:"description" binding:"omitempty,max=512"`
	MaxAttempts              *int     `form:"max_attempts" json:"max_attempts" binding:"omitempty,gte=1,lte=10"`
	InitialDelayMs           *int64   `form:"initial_delay_ms" json:"initial_delay_ms" binding:"omitempty,gte=1000,lte=3600000"`
	MaxDelayMs               *int64   `form:"max_delay_ms" json:"max_delay_ms" binding:"omitempty,gte=1000,lte=86400000"`
	BackoffType              string   `form:"backoff_type" json:"backoff_type" binding:"omitempty,oneof=fixed exponential"`
	BackoffMultiplier        *float64 `form:"backoff_multiplier" json:"backoff_multiplier" binding:"omitempty,gte=1,lte=4"`
	JitterType               string   `form:"jitter_type" json:"jitter_type" binding:"omitempty,oneof=none full"`
	JitterRatio              *float64 `form:"jitter_ratio" json:"jitter_ratio" binding:"omitempty,gte=0,lte=1"`
	RetryWindowMs            *int64   `form:"retry_window_ms" json:"retry_window_ms" binding:"omitempty,gte=60000,lte=604800000"`
	RetryableErrorCategories []string `form:"retryable_error_categories" json:"retryable_error_categories" binding:"omitempty,max=3,dive,oneof=network timeout remote"`
	RetryableHTTPStatuses    []int    `form:"retryable_http_statuses" json:"retryable_http_statuses" binding:"omitempty,max=4,dive,oneof=429 502 503 504"`
	RespectRetryAfter        *bool    `form:"respect_retry_after" json:"respect_retry_after"`
}

type RetryPolicyUpdateReq struct {
	PolicyName               *string   `form:"policy_name" json:"policy_name" binding:"omitempty,max=128"`
	Description              *string   `form:"description" json:"description" binding:"omitempty,max=512"`
	MaxAttempts              *int      `form:"max_attempts" json:"max_attempts" binding:"omitempty,gte=1,lte=10"`
	InitialDelayMs           *int64    `form:"initial_delay_ms" json:"initial_delay_ms" binding:"omitempty,gte=1000,lte=3600000"`
	MaxDelayMs               *int64    `form:"max_delay_ms" json:"max_delay_ms" binding:"omitempty,gte=1000,lte=86400000"`
	BackoffType              *string   `form:"backoff_type" json:"backoff_type" binding:"omitempty,oneof=fixed exponential"`
	BackoffMultiplier        *float64  `form:"backoff_multiplier" json:"backoff_multiplier" binding:"omitempty,gte=1,lte=4"`
	JitterType               *string   `form:"jitter_type" json:"jitter_type" binding:"omitempty,oneof=none full"`
	JitterRatio              *float64  `form:"jitter_ratio" json:"jitter_ratio" binding:"omitempty,gte=0,lte=1"`
	RetryWindowMs            *int64    `form:"retry_window_ms" json:"retry_window_ms" binding:"omitempty,gte=60000,lte=604800000"`
	RetryableErrorCategories *[]string `form:"retryable_error_categories" json:"retryable_error_categories" binding:"omitempty,max=3,dive,oneof=network timeout remote"`
	RetryableHTTPStatuses    *[]int    `form:"retryable_http_statuses" json:"retryable_http_statuses" binding:"omitempty,max=4,dive,oneof=429 502 503 504"`
	RespectRetryAfter        *bool     `form:"respect_retry_after" json:"respect_retry_after"`
	Revision                 int       `form:"revision" json:"revision" binding:"required,gt=0"`
}

type RetryPolicyVersionReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,gt=0"`
}

type RetryPolicyStateReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,gt=0"`
}
