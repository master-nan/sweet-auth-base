package request

import (
	"backend/enum"
	"time"
)

type IntegrationExecutionQueryReq struct {
	Page                  int         `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num                   int         `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order                 Order       `form:"order" json:"order"`
	QuickQuery            *QuickQuery `form:"quick_query" json:"quick_query"`
	ExternalSystemID      int         `form:"external_system_id" json:"external_system_id" binding:"omitempty,gt=0"`
	InterfaceDefinitionID int         `form:"interface_definition_id" json:"interface_definition_id" binding:"omitempty,gt=0"`
	TriggerSource         string      `form:"trigger_source" json:"trigger_source" binding:"omitempty,oneof=manual system_event scheduled"`
	Status                string      `form:"status" json:"status" binding:"omitempty,oneof=created running retry_waiting succeeded failed cancelled"`
	CreatedFrom           *time.Time  `form:"created_from" json:"created_from"`
	CreatedTo             *time.Time  `form:"created_to" json:"created_to"`
}

type IntegrationLogQueryReq struct {
	Page                  int         `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num                   int         `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order                 Order       `form:"order" json:"order"`
	QuickQuery            *QuickQuery `form:"quick_query" json:"quick_query"`
	ExecutionID           int         `form:"execution_id" json:"execution_id" binding:"omitempty,gt=0"`
	ExecutionNo           string      `form:"execution_no" json:"execution_no" binding:"omitempty,max=64"`
	ExternalSystemID      int         `form:"external_system_id" json:"external_system_id" binding:"omitempty,gt=0"`
	InterfaceDefinitionID int         `form:"interface_definition_id" json:"interface_definition_id" binding:"omitempty,gt=0"`
	AttemptNo             int         `form:"attempt_no" json:"attempt_no" binding:"omitempty,gt=0"`
	Status                string      `form:"status" json:"status" binding:"omitempty,oneof=running succeeded failed cancelled"`
	ErrorCategory         string      `form:"error_category" json:"error_category" binding:"omitempty,oneof=configuration credential network timeout remote response business concurrency system"`
	StartedFrom           *time.Time  `form:"started_from" json:"started_from"`
	StartedTo             *time.Time  `form:"started_to" json:"started_to"`
}

func (r IntegrationLogQueryReq) ToBasic() Basic {
	basic := Basic{Page: r.Page, Num: r.Num, Order: r.Order, QuickQuery: r.QuickQuery}
	filters := make(map[string]any, 4)
	if r.ExecutionID > 0 {
		filters["execution_id"] = r.ExecutionID
	}
	if r.AttemptNo > 0 {
		filters["attempt_no"] = r.AttemptNo
	}
	if r.Status != "" {
		filters["status"] = r.Status
	}
	if r.ErrorCategory != "" {
		filters["error_category"] = r.ErrorCategory
	}
	if len(filters) > 0 {
		basic.Filters = filters
	}
	timeRules := make([]QueryRule, 0, 2)
	if r.StartedFrom != nil {
		timeRules = append(timeRules, QueryRule{Field: "started_at", ExpressionType: enum.Gte, Value: *r.StartedFrom, Type: enum.DatetimeFieldType})
	}
	if r.StartedTo != nil {
		timeRules = append(timeRules, QueryRule{Field: "started_at", ExpressionType: enum.Lte, Value: *r.StartedTo, Type: enum.DatetimeFieldType})
	}
	if len(timeRules) > 0 {
		basic.Expressions = append(basic.Expressions, ExpressionGroup{Logic: enum.And, Rules: timeRules})
	}
	return basic
}

func (r IntegrationExecutionQueryReq) ToBasic() Basic {
	basic := Basic{Page: r.Page, Num: r.Num, Order: r.Order, QuickQuery: r.QuickQuery}
	filters := make(map[string]any, 4)
	if r.ExternalSystemID > 0 {
		filters["external_system_id"] = r.ExternalSystemID
	}
	if r.InterfaceDefinitionID > 0 {
		filters["interface_definition_id"] = r.InterfaceDefinitionID
	}
	if r.TriggerSource != "" {
		filters["trigger_source"] = r.TriggerSource
	}
	if r.Status != "" {
		filters["status"] = r.Status
	}
	if len(filters) > 0 {
		basic.Filters = filters
	}
	timeRules := make([]QueryRule, 0, 2)
	if r.CreatedFrom != nil {
		timeRules = append(timeRules, QueryRule{
			Field: "gmt_create", ExpressionType: enum.Gte,
			Value: *r.CreatedFrom, Type: enum.DatetimeFieldType,
		})
	}
	if r.CreatedTo != nil {
		timeRules = append(timeRules, QueryRule{
			Field: "gmt_create", ExpressionType: enum.Lte,
			Value: *r.CreatedTo, Type: enum.DatetimeFieldType,
		})
	}
	if len(timeRules) > 0 {
		basic.Expressions = append(basic.Expressions, ExpressionGroup{Logic: enum.And, Rules: timeRules})
	}
	return basic
}

type IntegrationExecutionCreateReq struct {
	ExternalSystemID      int    `form:"external_system_id" json:"external_system_id" binding:"required,gt=0"`
	InterfaceDefinitionID int    `form:"interface_definition_id" json:"interface_definition_id" binding:"required,gt=0"`
	TriggerSource         string `form:"trigger_source" json:"trigger_source" binding:"required,oneof=manual system_event scheduled"`
	IdempotencyScope      string `form:"idempotency_scope" json:"idempotency_scope" binding:"required,max=64"`
	IdempotencyKey        string `form:"idempotency_key" json:"idempotency_key" binding:"required,max=128"`
	InputHash             string `form:"input_hash" json:"input_hash" binding:"required,len=64,hexadecimal"`
}

type IntegrationExecutionStateReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,gt=0"`
}

type IntegrationExecutionCompleteReq struct {
	Revision         int    `form:"revision" json:"revision" binding:"required,gt=0"`
	ResultHTTPStatus *int   `form:"result_http_status" json:"result_http_status" binding:"omitempty,gte=100,lte=599"`
	ResultSizeBytes  int64  `form:"result_size_bytes" json:"result_size_bytes" binding:"gte=0"`
	ResultHash       string `form:"result_hash" json:"result_hash" binding:"omitempty,len=64,hexadecimal"`
	ResultSummary    string `form:"result_summary" json:"result_summary" binding:"omitempty,max=1024"`
}

type IntegrationExecutionFailReq struct {
	Revision      int    `form:"revision" json:"revision" binding:"required,gt=0"`
	TargetStatus  string `form:"target_status" json:"target_status" binding:"required,oneof=failed retry_waiting"`
	ErrorCategory string `form:"error_category" json:"error_category" binding:"required,oneof=configuration credential network timeout remote response business concurrency system"`
	ResultSummary string `form:"result_summary" json:"result_summary" binding:"omitempty,max=1024"`
}
