package response

import (
	"backend/model"
	"encoding/json"
	"time"
)

type SyncReferenceSummaryRes struct {
	ID      int    `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Version int    `json:"version,omitempty"`
}

type SyncConsumerMetadataRes struct {
	Code             string   `json:"code"`
	Version          int      `json:"version"`
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	ContentTypes     []string `json:"content_types"`
	MaxResponseBytes int64    `json:"max_response_bytes"`
	MaxDurationMs    int64    `json:"max_duration_ms"`
	CheckpointModes  []string `json:"checkpoint_modes"`
}

type SyncInputPlanSummaryRes struct {
	Version              int  `json:"version"`
	StaticParameterCount int  `json:"static_parameter_count"`
	HasWindowBindings    bool `json:"has_window_bindings"`
}

type SyncTaskListRes struct {
	ID                  int                     `json:"id"`
	TaskCode            string                  `json:"task_code"`
	TaskName            string                  `json:"task_name"`
	Version             int                     `json:"version"`
	Status              string                  `json:"status"`
	ExternalSystem      SyncReferenceSummaryRes `json:"external_system"`
	InterfaceDefinition SyncReferenceSummaryRes `json:"interface_definition"`
	Consumer            SyncReferenceSummaryRes `json:"consumer"`
	ScheduleType        string                  `json:"schedule_type"`
	CronSummary         string                  `json:"cron_summary,omitempty"`
	Timezone            string                  `json:"timezone"`
	CheckpointMode      string                  `json:"checkpoint_mode"`
	CheckpointAt        *time.Time              `json:"checkpoint_at,omitempty"`
	LookbackSeconds     int                     `json:"lookback_seconds"`
	WindowSliceSeconds  int                     `json:"window_slice_seconds"`
	InputPlanSummary    SyncInputPlanSummaryRes `json:"input_plan_summary"`
	Revision            int                     `json:"revision"`
	GmtModify           model.CustomTime        `json:"gmt_modify"`
}

type SyncTaskDetailRes struct {
	SyncTaskListRes
	Description         string           `json:"description"`
	InitialCheckpointAt *time.Time       `json:"initial_checkpoint_at,omitempty"`
	NextScheduledAt     *time.Time       `json:"next_scheduled_at,omitempty"`
	LastScheduledAt     *time.Time       `json:"last_scheduled_at,omitempty"`
	GmtCreate           model.CustomTime `json:"gmt_create"`
}

type SyncTaskEditRes struct {
	SyncTaskDetailRes
	InputPlan json.RawMessage `json:"input_plan"`
}

func NewSyncTaskListRes(value model.IntegrationSyncTask, system model.ExternalSystem, definition model.InterfaceDefinition, summary SyncInputPlanSummaryRes) SyncTaskListRes {
	return SyncTaskListRes{ID: value.Id, TaskCode: value.TaskCode, TaskName: value.TaskName, Version: value.Version, Status: value.Status,
		ExternalSystem:      SyncReferenceSummaryRes{ID: system.Id, Code: system.SystemCode, Name: system.Name},
		InterfaceDefinition: SyncReferenceSummaryRes{ID: definition.Id, Code: definition.InterfaceCode, Name: definition.Name, Version: definition.Version},
		Consumer:            SyncReferenceSummaryRes{Code: value.ConsumerCode, Version: value.ConsumerVersion}, ScheduleType: value.ScheduleType,
		CronSummary: value.CronExpression, Timezone: value.Timezone, CheckpointMode: value.CheckpointMode, CheckpointAt: value.CheckpointAt,
		LookbackSeconds: value.LookbackSeconds, WindowSliceSeconds: value.WindowSliceSeconds, InputPlanSummary: summary, Revision: value.Revision, GmtModify: value.GmtModify}
}

func NewSyncTaskDetailRes(value model.IntegrationSyncTask, system model.ExternalSystem, definition model.InterfaceDefinition, summary SyncInputPlanSummaryRes) SyncTaskDetailRes {
	return SyncTaskDetailRes{SyncTaskListRes: NewSyncTaskListRes(value, system, definition, summary), Description: value.Description, InitialCheckpointAt: value.InitialCheckpointAt, NextScheduledAt: value.NextScheduledAt, LastScheduledAt: value.LastScheduledAt, GmtCreate: value.GmtCreate}
}

type SyncBatchListRes struct {
	ID                    int              `json:"id"`
	BatchNo               string           `json:"batch_no"`
	SyncTaskID            int              `json:"sync_task_id"`
	TaskCode              string           `json:"task_code"`
	TaskName              string           `json:"task_name"`
	TaskVersion           int              `json:"task_version"`
	TriggerType           string           `json:"trigger_type"`
	Status                string           `json:"status"`
	WindowStart           *time.Time       `json:"window_start,omitempty"`
	WindowEnd             *time.Time       `json:"window_end,omitempty"`
	CheckpointBefore      *time.Time       `json:"checkpoint_before,omitempty"`
	CheckpointAfter       *time.Time       `json:"checkpoint_after,omitempty"`
	PlannedSliceCount     int              `json:"planned_slice_count"`
	CurrentSliceNo        int              `json:"current_slice_no"`
	ExecutionCount        int              `json:"execution_count"`
	TechnicalSuccessCount int              `json:"technical_success_count"`
	TechnicalFailedCount  int              `json:"technical_failed_count"`
	BusinessSuccessCount  int              `json:"business_success_count"`
	BusinessFailedCount   int              `json:"business_failed_count"`
	ReasonCode            string           `json:"reason_code,omitempty"`
	StartedAt             *time.Time       `json:"started_at,omitempty"`
	CompletedAt           *time.Time       `json:"completed_at,omitempty"`
	GmtCreate             model.CustomTime `json:"gmt_create"`
}

type SyncBatchDetailRes struct {
	SyncBatchListRes
	SystemCode         string `json:"system_code"`
	InterfaceCode      string `json:"interface_code"`
	InterfaceVersion   int    `json:"interface_version"`
	ConsumerCode       string `json:"consumer_code"`
	ConsumerVersion    int    `json:"consumer_version"`
	CheckpointMode     string `json:"checkpoint_mode"`
	LookbackSeconds    int    `json:"lookback_seconds"`
	WindowSliceSeconds int    `json:"window_slice_seconds"`
	ResultSummary      string `json:"result_summary,omitempty"`
	Revision           int    `json:"revision"`
}

func NewSyncBatchListRes(value model.IntegrationSyncBatch) SyncBatchListRes {
	return SyncBatchListRes{ID: value.Id, BatchNo: value.BatchNo, SyncTaskID: value.SyncTaskID, TaskCode: value.TaskCode, TaskName: value.TaskName, TaskVersion: value.TaskVersion,
		TriggerType: value.TriggerType, Status: value.Status, WindowStart: value.WindowStart, WindowEnd: value.WindowEnd, CheckpointBefore: value.CheckpointBefore, CheckpointAfter: value.CheckpointAfter,
		PlannedSliceCount: value.PlannedSliceCount, CurrentSliceNo: value.CurrentSliceNo, ExecutionCount: value.ExecutionCount, TechnicalSuccessCount: value.TechnicalSuccessCount,
		TechnicalFailedCount: value.TechnicalFailedCount, BusinessSuccessCount: value.BusinessSuccessCount, BusinessFailedCount: value.BusinessFailedCount,
		ReasonCode: value.ReasonCode, StartedAt: value.StartedAt, CompletedAt: value.CompletedAt, GmtCreate: value.GmtCreate}
}

func NewSyncBatchDetailRes(value model.IntegrationSyncBatch) SyncBatchDetailRes {
	return SyncBatchDetailRes{SyncBatchListRes: NewSyncBatchListRes(value), SystemCode: value.SystemCode, InterfaceCode: value.InterfaceCode, InterfaceVersion: value.InterfaceVersion,
		ConsumerCode: value.ConsumerCode, ConsumerVersion: value.ConsumerVersion, CheckpointMode: value.CheckpointMode, LookbackSeconds: value.LookbackSeconds,
		WindowSliceSeconds: value.WindowSliceSeconds, ResultSummary: value.ResultSummary, Revision: value.Revision}
}
