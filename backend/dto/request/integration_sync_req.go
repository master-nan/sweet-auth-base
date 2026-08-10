package request

import (
	"encoding/json"
	"time"
)

type SyncWindowBindingReq struct {
	Location string `json:"location" binding:"required,oneof=path query header body"`
	Code     string `json:"code" binding:"required,max=128"`
	Format   string `json:"format" binding:"required,oneof=rfc3339 unix_seconds unix_milliseconds"`
}

type SyncStaticInputReq struct {
	PathParams  map[string]string   `json:"path_params"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	JSONBody    json.RawMessage     `json:"json_body,omitempty"`
}

type SyncExecutionInputPlanReq struct {
	Version            int                   `json:"version" binding:"required,eq=1"`
	StaticInput        SyncStaticInputReq    `json:"static_input" binding:"required"`
	WindowStartBinding *SyncWindowBindingReq `json:"window_start_binding,omitempty"`
	WindowEndBinding   *SyncWindowBindingReq `json:"window_end_binding,omitempty"`
}

type SyncTaskQueryReq struct {
	Page             int               `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num              int               `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order            Order             `form:"order" json:"order"`
	Expressions      []ExpressionGroup `form:"expressions" json:"expressions" binding:"omitempty,max=8"`
	QuickQuery       *QuickQuery       `form:"quick_query" json:"quick_query"`
	Status           string            `form:"status" json:"status" binding:"omitempty,oneof=draft enabled disabled"`
	ScheduleType     string            `form:"schedule_type" json:"schedule_type" binding:"omitempty,oneof=none cron"`
	CheckpointMode   string            `form:"checkpoint_mode" json:"checkpoint_mode" binding:"omitempty,oneof=none timestamp"`
	ExternalSystemID *int              `form:"external_system_id" json:"external_system_id" binding:"omitempty,gt=0"`
}

func (r SyncTaskQueryReq) ToBasic() Basic {
	basic := Basic{Page: r.Page, Num: r.Num, Order: r.Order, Expressions: r.Expressions, QuickQuery: r.QuickQuery}
	filters := map[string]any{}
	if r.Status != "" {
		filters["status"] = r.Status
	}
	if r.ScheduleType != "" {
		filters["schedule_type"] = r.ScheduleType
	}
	if r.CheckpointMode != "" {
		filters["checkpoint_mode"] = r.CheckpointMode
	}
	if r.ExternalSystemID != nil {
		filters["external_system_id"] = *r.ExternalSystemID
	}
	if len(filters) > 0 {
		basic.Filters = filters
	}
	return basic
}

type SyncTaskCreateReq struct {
	TaskCode              string                    `json:"task_code" binding:"required,max=64"`
	TaskName              string                    `json:"task_name" binding:"required,max=128"`
	Description           string                    `json:"description" binding:"omitempty,max=512"`
	ExternalSystemID      int                       `json:"external_system_id" binding:"required,gt=0"`
	InterfaceDefinitionID int                       `json:"interface_definition_id" binding:"required,gt=0"`
	ConsumerCode          string                    `json:"consumer_code" binding:"required,max=64"`
	ConsumerVersion       int                       `json:"consumer_version" binding:"required,gt=0"`
	ScheduleType          string                    `json:"schedule_type" binding:"required,oneof=none cron"`
	CronExpression        string                    `json:"cron_expression" binding:"omitempty,max=128"`
	Timezone              string                    `json:"timezone" binding:"omitempty,max=64"`
	CheckpointMode        string                    `json:"checkpoint_mode" binding:"required,oneof=none timestamp"`
	InitialCheckpointAt   *time.Time                `json:"initial_checkpoint_at"`
	LookbackSeconds       int                       `json:"lookback_seconds" binding:"gte=0,lte=604800"`
	WindowSliceSeconds    int                       `json:"window_slice_seconds" binding:"gte=0,lte=604800"`
	InputPlan             SyncExecutionInputPlanReq `json:"input_plan" binding:"required"`
}

type SyncTaskUpdateReq struct {
	TaskName               *string                    `json:"task_name" binding:"omitempty,max=128"`
	Description            *string                    `json:"description" binding:"omitempty,max=512"`
	ExternalSystemID       *int                       `json:"external_system_id" binding:"omitempty,gt=0"`
	InterfaceDefinitionID  *int                       `json:"interface_definition_id" binding:"omitempty,gt=0"`
	ConsumerCode           *string                    `json:"consumer_code" binding:"omitempty,max=64"`
	ConsumerVersion        *int                       `json:"consumer_version" binding:"omitempty,gt=0"`
	ScheduleType           *string                    `json:"schedule_type" binding:"omitempty,oneof=none cron"`
	CronExpression         *string                    `json:"cron_expression" binding:"omitempty,max=128"`
	Timezone               *string                    `json:"timezone" binding:"omitempty,max=64"`
	CheckpointMode         *string                    `json:"checkpoint_mode" binding:"omitempty,oneof=none timestamp"`
	InitialCheckpointAt    *time.Time                 `json:"initial_checkpoint_at"`
	ClearInitialCheckpoint bool                       `json:"clear_initial_checkpoint"`
	LookbackSeconds        *int                       `json:"lookback_seconds" binding:"omitempty,gte=0,lte=604800"`
	WindowSliceSeconds     *int                       `json:"window_slice_seconds" binding:"omitempty,gte=0,lte=604800"`
	InputPlan              *SyncExecutionInputPlanReq `json:"input_plan"`
	Revision               int                        `json:"revision" binding:"required,gt=0"`
}

type SyncTaskVersionReq struct {
	Revision int `json:"revision" binding:"required,gt=0"`
}
type SyncTaskStateReq struct {
	Revision int `json:"revision" binding:"required,gt=0"`
}

type SyncBatchQueryReq struct {
	Page        int               `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num         int               `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order       Order             `form:"order" json:"order"`
	Expressions []ExpressionGroup `form:"expressions" json:"expressions" binding:"omitempty,max=8"`
	QuickQuery  *QuickQuery       `form:"quick_query" json:"quick_query"`
	Status      string            `form:"status" json:"status" binding:"omitempty,oneof=created running succeeded failed"`
	TriggerType string            `form:"trigger_type" json:"trigger_type" binding:"omitempty,oneof=manual scheduled"`
	SyncTaskID  *int              `form:"sync_task_id" json:"sync_task_id" binding:"omitempty,gt=0"`
}

func (r SyncBatchQueryReq) ToBasic() Basic {
	basic := Basic{Page: r.Page, Num: r.Num, Order: r.Order, Expressions: r.Expressions, QuickQuery: r.QuickQuery}
	filters := map[string]any{}
	if r.Status != "" {
		filters["status"] = r.Status
	}
	if r.TriggerType != "" {
		filters["trigger_type"] = r.TriggerType
	}
	if r.SyncTaskID != nil {
		filters["sync_task_id"] = *r.SyncTaskID
	}
	if len(filters) > 0 {
		basic.Filters = filters
	}
	return basic
}
