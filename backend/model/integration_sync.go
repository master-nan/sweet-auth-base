package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	IntegrationSyncTaskStatusDraft    = "draft"
	IntegrationSyncTaskStatusEnabled  = "enabled"
	IntegrationSyncTaskStatusDisabled = "disabled"

	IntegrationSyncScheduleNone = "none"
	IntegrationSyncScheduleCron = "cron"

	IntegrationSyncCheckpointNone      = "none"
	IntegrationSyncCheckpointTimestamp = "timestamp"

	IntegrationSyncBatchStatusCreated   = "created"
	IntegrationSyncBatchStatusRunning   = "running"
	IntegrationSyncBatchStatusSucceeded = "succeeded"
	IntegrationSyncBatchStatusFailed    = "failed"

	IntegrationSyncTriggerManual    = "manual"
	IntegrationSyncTriggerScheduled = "scheduled"
)

// IntegrationSyncTask 描述版本化同步调度配置，不执行 HTTP 或业务处理。
type IntegrationSyncTask struct {
	Basic

	TaskCode              string         `gorm:"size:64;not null;uniqueIndex:uni_integration_sync_task_identity,priority:1;index:idx_integration_sync_task_code" json:"task_code"`
	TaskName              string         `gorm:"size:128;not null;index:idx_integration_sync_task_name" json:"task_name"`
	Version               int            `gorm:"not null;uniqueIndex:uni_integration_sync_task_identity,priority:2" json:"version"`
	Description           string         `gorm:"size:512" json:"description"`
	Status                string         `gorm:"size:16;not null;default:draft;index:idx_integration_sync_task_status;index:idx_integration_sync_task_schedule,priority:1" json:"status"`
	ExternalSystemID      int            `gorm:"type:bigint;not null;index:idx_integration_sync_task_system" json:"external_system_id"`
	InterfaceDefinitionID int            `gorm:"type:bigint;not null;index:idx_integration_sync_task_interface" json:"interface_definition_id"`
	ConsumerCode          string         `gorm:"size:64;not null;index:idx_integration_sync_task_consumer" json:"consumer_code"`
	ConsumerVersion       int            `gorm:"not null" json:"consumer_version"`
	ScheduleType          string         `gorm:"size:16;not null;default:none" json:"schedule_type"`
	CronExpression        string         `gorm:"size:128" json:"cron_expression"`
	Timezone              string         `gorm:"size:64;not null;default:UTC" json:"timezone"`
	NextScheduledAt       *time.Time     `gorm:"index:idx_integration_sync_task_schedule,priority:2" json:"-"`
	LastScheduledAt       *time.Time     `json:"-"`
	CheckpointMode        string         `gorm:"size:16;not null;default:none" json:"checkpoint_mode"`
	InitialCheckpointAt   *time.Time     `json:"initial_checkpoint_at"`
	CheckpointAt          *time.Time     `json:"-"`
	LookbackSeconds       int            `gorm:"not null;default:0" json:"lookback_seconds"`
	WindowSliceSeconds    int            `gorm:"not null;default:0" json:"window_slice_seconds"`
	InputPlan             datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
	Revision              int            `gorm:"not null;default:1" json:"revision"`

	ExternalSystem      ExternalSystem      `gorm:"foreignKey:ExternalSystemID;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	InterfaceDefinition InterfaceDefinition `gorm:"foreignKey:InterfaceDefinitionID;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

func (IntegrationSyncTask) TableName() string { return "integration_sync_task" }

// IntegrationSyncBatch 表示 SyncTask 的一次技术运行实例；V1 仅提供查询基础。
type IntegrationSyncBatch struct {
	Basic

	BatchNo               string     `gorm:"size:64;not null;uniqueIndex:uni_integration_sync_batch_no" json:"batch_no"`
	SyncTaskID            int        `gorm:"type:bigint;not null;index:idx_integration_sync_batch_task" json:"sync_task_id"`
	TaskCode              string     `gorm:"size:64;not null;index:idx_integration_sync_batch_task_code" json:"task_code"`
	TaskName              string     `gorm:"size:128;not null" json:"task_name"`
	TaskVersion           int        `gorm:"not null" json:"task_version"`
	TaskRevision          int        `gorm:"not null;default:1" json:"task_revision"`
	SystemCode            string     `gorm:"size:64;not null" json:"system_code"`
	InterfaceCode         string     `gorm:"size:64;not null" json:"interface_code"`
	InterfaceVersion      int        `gorm:"not null" json:"interface_version"`
	ConsumerCode          string     `gorm:"size:64;not null" json:"consumer_code"`
	ConsumerVersion       int        `gorm:"not null" json:"consumer_version"`
	TriggerType           string     `gorm:"size:16;not null;index:idx_integration_sync_batch_trigger" json:"trigger_type"`
	TriggerKey            string     `gorm:"size:128;not null;uniqueIndex:uni_integration_sync_batch_trigger_key" json:"trigger_key"`
	ScheduledFor          *time.Time `gorm:"index:idx_integration_sync_batch_scheduled" json:"scheduled_for"`
	TriggeredByUserID     *int       `gorm:"type:bigint" json:"triggered_by_user_id"`
	TriggeredByUserName   string     `gorm:"size:128" json:"triggered_by_user_name"`
	Status                string     `gorm:"size:16;not null;default:created;index:idx_integration_sync_batch_status" json:"status"`
	StartedAt             *time.Time `gorm:"index:idx_integration_sync_batch_started" json:"started_at"`
	CompletedAt           *time.Time `json:"completed_at"`
	WindowStart           *time.Time `json:"window_start"`
	WindowEnd             *time.Time `json:"window_end"`
	CheckpointBefore      *time.Time `json:"checkpoint_before"`
	CheckpointAfter       *time.Time `json:"checkpoint_after"`
	CheckpointMode        string     `gorm:"size:16;not null" json:"checkpoint_mode"`
	LookbackSeconds       int        `gorm:"not null;default:0" json:"lookback_seconds"`
	WindowSliceSeconds    int        `gorm:"not null;default:0" json:"window_slice_seconds"`
	PlannedSliceCount     int        `gorm:"not null;default:0" json:"planned_slice_count"`
	CurrentSliceNo        int        `gorm:"not null;default:0" json:"current_slice_no"`
	ExecutionCount        int        `gorm:"not null;default:0" json:"execution_count"`
	TechnicalSuccessCount int        `gorm:"not null;default:0" json:"technical_success_count"`
	TechnicalFailedCount  int        `gorm:"not null;default:0" json:"technical_failed_count"`
	BusinessSuccessCount  int        `gorm:"not null;default:0" json:"business_success_count"`
	BusinessFailedCount   int        `gorm:"not null;default:0" json:"business_failed_count"`
	ReasonCode            string     `gorm:"size:64" json:"reason_code"`
	ResultSummary         string     `gorm:"size:1024" json:"result_summary"`
	Revision              int        `gorm:"not null;default:1" json:"revision"`

	SyncTask IntegrationSyncTask `gorm:"foreignKey:SyncTaskID;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

func (IntegrationSyncBatch) TableName() string { return "integration_sync_batch" }
