package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	OrgStructureTypeManagement = "management"
	OrgStructureTypeLegal      = "legal"

	OrgSyncRecordActionCreate   = "create"
	OrgSyncRecordActionUpdate   = "update"
	OrgSyncRecordActionDisable  = "disable"
	OrgSyncRecordActionClose    = "close"
	OrgSyncRecordActionNoop     = "noop"
	OrgSyncRecordActionError    = "error"
	OrgSyncRecordActionDeferred = "deferred"
)

// OrgLegalEntity 是法律或核算主体的只读镜像。
type OrgLegalEntity struct {
	Basic

	// 外部来源管理字段。
	SourceSystemCode        string     `gorm:"size:64;not null;uniqueIndex:uni_org_legal_entity_source,priority:1;uniqueIndex:uni_org_legal_entity_code,priority:1;uniqueIndex:uni_org_legal_entity_source_code,priority:1,where:source_code IS NOT NULL AND source_code <> ''" json:"source_system_code"`
	SourceId                string     `gorm:"size:128;not null;uniqueIndex:uni_org_legal_entity_source,priority:2" json:"source_id"`
	SourceCode              string     `gorm:"size:128;uniqueIndex:uni_org_legal_entity_source_code,priority:2,where:source_code IS NOT NULL AND source_code <> '';index:idx_org_legal_entity_source_code" json:"source_code"`
	Code                    string     `gorm:"size:128;not null;uniqueIndex:uni_org_legal_entity_code,priority:2" json:"code"`
	Name                    string     `gorm:"size:255;not null;index:idx_org_legal_entity_name" json:"name"`
	ShortName               string     `gorm:"size:128;index:idx_org_legal_entity_short_name" json:"short_name"`
	EntityType              string     `gorm:"size:32;not null;default:legal_company;index:idx_org_legal_entity_type" json:"entity_type"`
	ParentId                *int       `gorm:"type:bigint;index:idx_org_legal_entity_parent" json:"parent_id"`
	UnifiedSocialCreditCode string     `gorm:"size:64;uniqueIndex:uni_org_legal_entity_credit,where:unified_social_credit_code IS NOT NULL AND unified_social_credit_code <> '' AND source_deleted = false" json:"unified_social_credit_code"`
	AccountingCode          string     `gorm:"size:64;index:idx_org_legal_entity_accounting_code" json:"accounting_code"`
	Status                  string     `gorm:"size:32;not null;default:enabled;index:idx_org_legal_entity_status" json:"status"`
	ValidFrom               *time.Time `gorm:"type:timestamp;index:idx_org_legal_entity_valid_from" json:"valid_from"`
	ValidTo                 *time.Time `gorm:"type:timestamp;index:idx_org_legal_entity_valid_to" json:"valid_to"`
	SourceVersion           string     `gorm:"size:64;index:idx_org_legal_entity_source_version" json:"source_version"`
	SourceUpdatedAt         *time.Time `gorm:"type:timestamp;index:idx_org_legal_entity_source_updated_at" json:"source_updated_at"`
	LastSyncAt              *time.Time `gorm:"type:timestamp;index:idx_org_legal_entity_last_sync_at" json:"last_sync_at"`
	SourceStatus            string     `gorm:"size:32;index:idx_org_legal_entity_source_status" json:"source_status"`
	SourceDeleted           bool       `gorm:"not null;default:false;index:idx_org_legal_entity_source_deleted" json:"source_deleted"`
	SyncStatus              string     `gorm:"size:32;not null;default:pending;index:idx_org_legal_entity_sync_status" json:"sync_status"`
	LastError               string     `gorm:"type:text" json:"last_error"`

	// 平台管理扩展字段。
	LocalNote           string         `gorm:"type:text" json:"local_note"`
	LocalTags           datatypes.JSON `gorm:"type:jsonb" json:"local_tags"`
	DisplayOrder        *int           `gorm:"index:idx_org_legal_entity_display_order" json:"display_order"`
	LocalHandlingStatus string         `gorm:"size:32;index:idx_org_legal_entity_handling_status" json:"local_handling_status"`

	Parent *OrgLegalEntity `gorm:"foreignKey:ParentId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"parent,omitempty"`
}

// OrgUnit 是管理组织的只读镜像。
type OrgUnit struct {
	Basic

	// 外部来源管理字段。
	SourceSystemCode     string     `gorm:"size:64;not null;uniqueIndex:uni_org_unit_source,priority:1;uniqueIndex:uni_org_unit_code,priority:1;uniqueIndex:uni_org_unit_source_code,priority:1,where:source_code IS NOT NULL AND source_code <> ''" json:"source_system_code"`
	SourceId             string     `gorm:"size:128;not null;uniqueIndex:uni_org_unit_source,priority:2" json:"source_id"`
	SourceCode           string     `gorm:"size:128;uniqueIndex:uni_org_unit_source_code,priority:2,where:source_code IS NOT NULL AND source_code <> '';index:idx_org_unit_source_code" json:"source_code"`
	Code                 string     `gorm:"size:128;not null;uniqueIndex:uni_org_unit_code,priority:2" json:"code"`
	Name                 string     `gorm:"size:255;not null;index:idx_org_unit_name" json:"name"`
	UnitType             string     `gorm:"size:32;not null;default:department;index:idx_org_unit_type" json:"unit_type"`
	PrimaryLegalEntityId *int       `gorm:"type:bigint;index:idx_org_unit_primary_legal_entity" json:"primary_legal_entity_id"`
	Status               string     `gorm:"size:32;not null;default:enabled;index:idx_org_unit_status" json:"status"`
	ValidFrom            *time.Time `gorm:"type:timestamp;index:idx_org_unit_valid_from" json:"valid_from"`
	ValidTo              *time.Time `gorm:"type:timestamp;index:idx_org_unit_valid_to" json:"valid_to"`
	SourceVersion        string     `gorm:"size:64;index:idx_org_unit_source_version" json:"source_version"`
	SourceUpdatedAt      *time.Time `gorm:"type:timestamp;index:idx_org_unit_source_updated_at" json:"source_updated_at"`
	LastSyncAt           *time.Time `gorm:"type:timestamp;index:idx_org_unit_last_sync_at" json:"last_sync_at"`
	SourceStatus         string     `gorm:"size:32;index:idx_org_unit_source_status" json:"source_status"`
	SourceDeleted        bool       `gorm:"not null;default:false;index:idx_org_unit_source_deleted" json:"source_deleted"`
	SyncStatus           string     `gorm:"size:32;not null;default:pending;index:idx_org_unit_sync_status" json:"sync_status"`
	LastError            string     `gorm:"type:text" json:"last_error"`

	// 平台管理扩展字段。
	LocalNote           string         `gorm:"type:text" json:"local_note"`
	LocalTags           datatypes.JSON `gorm:"type:jsonb" json:"local_tags"`
	DisplayOrder        *int           `gorm:"index:idx_org_unit_display_order" json:"display_order"`
	LocalHandlingStatus string         `gorm:"size:32;index:idx_org_unit_handling_status" json:"local_handling_status"`

	PrimaryLegalEntity *OrgLegalEntity `gorm:"foreignKey:PrimaryLegalEntityId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"primary_legal_entity,omitempty"`
}

// OrgStructure 标识一套管理组织架构。
type OrgStructure struct {
	Basic

	// 外部来源管理字段。
	Code             string     `gorm:"size:64;not null;uniqueIndex:uni_org_structure_code" json:"code"`
	Name             string     `gorm:"size:128;not null;index:idx_org_structure_name" json:"name"`
	StructureType    string     `gorm:"size:32;not null;default:management;index:idx_org_structure_type" json:"structure_type"`
	SourceSystemCode string     `gorm:"size:64;not null;uniqueIndex:uni_org_structure_source,priority:1,where:source_id IS NOT NULL AND source_id <> ''" json:"source_system_code"`
	SourceId         string     `gorm:"size:128;uniqueIndex:uni_org_structure_source,priority:2,where:source_id IS NOT NULL AND source_id <> ''" json:"source_id"`
	Status           string     `gorm:"size:32;not null;default:enabled;index:idx_org_structure_status" json:"status"`
	IsDefault        bool       `gorm:"not null;default:false;index:idx_org_structure_default" json:"is_default"`
	ValidFrom        *time.Time `gorm:"type:timestamp;index:idx_org_structure_valid_from" json:"valid_from"`
	ValidTo          *time.Time `gorm:"type:timestamp;index:idx_org_structure_valid_to" json:"valid_to"`
	SourceVersion    string     `gorm:"size:64;index:idx_org_structure_source_version" json:"source_version"`
	LastSyncAt       *time.Time `gorm:"type:timestamp;index:idx_org_structure_last_sync_at" json:"last_sync_at"`
	SyncStatus       string     `gorm:"size:32;not null;default:pending;index:idx_org_structure_sync_status" json:"sync_status"`
}

// OrgStructureNode 将管理组织映射到组织架构树。
// 业务记录必须保存 OrgUnitId，不得保存运行时节点 ID。
type OrgStructureNode struct {
	Basic

	// 外部来源管理字段。
	StructureId      int        `gorm:"type:bigint;not null;uniqueIndex:uni_org_structure_node_current,priority:1,where:status = 'enabled' AND source_deleted = false AND valid_to IS NULL;index:idx_org_structure_node_structure_parent,priority:1;index:idx_org_structure_node_structure_path,priority:1;index:idx_org_structure_node_structure_unit,priority:1" json:"structure_id"`
	OrgUnitId        int        `gorm:"type:bigint;not null;uniqueIndex:uni_org_structure_node_current,priority:2,where:status = 'enabled' AND source_deleted = false AND valid_to IS NULL;index:idx_org_structure_node_structure_unit,priority:2" json:"org_unit_id"`
	ParentNodeId     *int       `gorm:"type:bigint;index:idx_org_structure_node_structure_parent,priority:2" json:"parent_node_id"`
	SourceSystemCode string     `gorm:"size:64;not null;uniqueIndex:uni_org_structure_node_source,priority:1" json:"source_system_code"`
	SourceId         string     `gorm:"size:128;not null;uniqueIndex:uni_org_structure_node_source,priority:2" json:"source_id"`
	SourceParentId   string     `gorm:"size:128;index:idx_org_structure_node_source_parent" json:"source_parent_id"`
	Path             string     `gorm:"size:1024;not null;index:idx_org_structure_node_structure_path,priority:2" json:"path"`
	Level            int        `gorm:"not null;default:1;index:idx_org_structure_node_level" json:"level"`
	Sort             int        `gorm:"not null;default:0;index:idx_org_structure_node_sort" json:"sort"`
	ValidFrom        *time.Time `gorm:"type:timestamp;index:idx_org_structure_node_valid_from" json:"valid_from"`
	ValidTo          *time.Time `gorm:"type:timestamp;index:idx_org_structure_node_valid_to" json:"valid_to"`
	Status           string     `gorm:"size:32;not null;default:enabled;index:idx_org_structure_node_structure_parent,priority:3;index:idx_org_structure_node_status" json:"status"`
	SourceDeleted    bool       `gorm:"not null;default:false;index:idx_org_structure_node_source_deleted" json:"source_deleted"`
	SyncStatus       string     `gorm:"size:32;not null;default:pending;index:idx_org_structure_node_sync_status" json:"sync_status"`

	Structure *OrgStructure     `gorm:"foreignKey:StructureId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"structure,omitempty"`
	OrgUnit   *OrgUnit          `gorm:"foreignKey:OrgUnitId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"org_unit,omitempty"`
	Parent    *OrgStructureNode `gorm:"foreignKey:ParentNodeId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"parent,omitempty"`
}

// OrgPosition 是管理组织中岗位的只读镜像。
type OrgPosition struct {
	Basic

	// 外部来源管理字段。
	SourceSystemCode  string     `gorm:"size:64;not null;uniqueIndex:uni_org_position_source,priority:1;uniqueIndex:uni_org_position_code,priority:1;uniqueIndex:uni_org_position_source_code,priority:1,where:source_code IS NOT NULL AND source_code <> ''" json:"source_system_code"`
	SourceId          string     `gorm:"size:128;not null;uniqueIndex:uni_org_position_source,priority:2" json:"source_id"`
	SourceCode        string     `gorm:"size:128;uniqueIndex:uni_org_position_source_code,priority:2,where:source_code IS NOT NULL AND source_code <> '';index:idx_org_position_source_code" json:"source_code"`
	Code              string     `gorm:"size:128;not null;uniqueIndex:uni_org_position_code,priority:2;uniqueIndex:uni_org_position_unit_code,priority:2,where:status = 'enabled' AND source_deleted = false AND valid_to IS NULL" json:"code"`
	Name              string     `gorm:"size:255;not null;index:idx_org_position_name" json:"name"`
	OrgUnitId         int        `gorm:"type:bigint;not null;index:idx_org_position_org_unit;uniqueIndex:uni_org_position_unit_code,priority:1,where:status = 'enabled' AND source_deleted = false AND valid_to IS NULL" json:"org_unit_id"`
	PositionType      string     `gorm:"size:32;not null;default:professional;index:idx_org_position_type" json:"position_type"`
	JobLevel          string     `gorm:"size:64;index:idx_org_position_job_level" json:"job_level"`
	IsManagerPosition bool       `gorm:"not null;default:false;index:idx_org_position_manager" json:"is_manager_position"`
	Status            string     `gorm:"size:32;not null;default:enabled;index:idx_org_position_status" json:"status"`
	ValidFrom         *time.Time `gorm:"type:timestamp;index:idx_org_position_valid_from" json:"valid_from"`
	ValidTo           *time.Time `gorm:"type:timestamp;index:idx_org_position_valid_to" json:"valid_to"`
	SourceVersion     string     `gorm:"size:64;index:idx_org_position_source_version" json:"source_version"`
	LastSyncAt        *time.Time `gorm:"type:timestamp;index:idx_org_position_last_sync_at" json:"last_sync_at"`
	SourceDeleted     bool       `gorm:"not null;default:false;index:idx_org_position_source_deleted" json:"source_deleted"`
	SyncStatus        string     `gorm:"size:32;not null;default:pending;index:idx_org_position_sync_status" json:"sync_status"`

	// 平台管理扩展字段。
	LocalNote string `gorm:"type:text" json:"local_note"`

	OrgUnit *OrgUnit `gorm:"foreignKey:OrgUnitId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"org_unit,omitempty"`
}

// OrgEmployee 是企业人员的只读镜像。
// UserId 是可空的平台账号绑定，不属于人员主数据。
type OrgEmployee struct {
	Basic

	// 外部来源管理字段。
	SourceSystemCode     string     `gorm:"size:64;not null;uniqueIndex:uni_org_employee_source,priority:1;uniqueIndex:uni_org_employee_no,priority:1;uniqueIndex:uni_org_employee_source_code,priority:1,where:source_code IS NOT NULL AND source_code <> ''" json:"source_system_code"`
	SourceId             string     `gorm:"size:128;not null;uniqueIndex:uni_org_employee_source,priority:2" json:"source_id"`
	SourceCode           string     `gorm:"size:128;uniqueIndex:uni_org_employee_source_code,priority:2,where:source_code IS NOT NULL AND source_code <> '';index:idx_org_employee_source_code" json:"source_code"`
	EmployeeNo           string     `gorm:"size:128;not null;uniqueIndex:uni_org_employee_no,priority:2" json:"employee_no"`
	Name                 string     `gorm:"size:128;not null;index:idx_org_employee_name" json:"name"`
	Mobile               string     `gorm:"size:64;index:idx_org_employee_mobile" json:"mobile"`
	Email                string     `gorm:"size:128;index:idx_org_employee_email" json:"email"`
	EmploymentStatus     string     `gorm:"size:32;not null;default:active;index:idx_org_employee_employment_status" json:"employment_status"`
	PrimaryLegalEntityId *int       `gorm:"type:bigint;index:idx_org_employee_primary_legal_entity" json:"primary_legal_entity_id"`
	ValidFrom            *time.Time `gorm:"type:timestamp;index:idx_org_employee_valid_from" json:"valid_from"`
	ValidTo              *time.Time `gorm:"type:timestamp;index:idx_org_employee_valid_to" json:"valid_to"`
	SourceVersion        string     `gorm:"size:64;index:idx_org_employee_source_version" json:"source_version"`
	SourceUpdatedAt      *time.Time `gorm:"type:timestamp;index:idx_org_employee_source_updated_at" json:"source_updated_at"`
	LastSyncAt           *time.Time `gorm:"type:timestamp;index:idx_org_employee_last_sync_at" json:"last_sync_at"`
	SourceDeleted        bool       `gorm:"not null;default:false;index:idx_org_employee_source_deleted" json:"source_deleted"`
	SyncStatus           string     `gorm:"size:32;not null;default:pending;index:idx_org_employee_sync_status" json:"sync_status"`

	// 平台管理扩展字段。
	UserId    *int           `gorm:"type:bigint;uniqueIndex:uni_org_employee_user,where:user_id IS NOT NULL" json:"user_id"`
	LocalNote string         `gorm:"type:text" json:"local_note"`
	LocalTags datatypes.JSON `gorm:"type:jsonb" json:"local_tags"`

	PrimaryLegalEntity *OrgLegalEntity `gorm:"foreignKey:PrimaryLegalEntityId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"primary_legal_entity,omitempty"`
	User               *SysUser        `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:SET NULL" json:"user,omitempty"`
}

// OrgAssignment 是人员任职关系的只读镜像。
type OrgAssignment struct {
	Basic

	// 外部来源管理字段。
	SourceSystemCode string     `gorm:"size:64;not null;uniqueIndex:uni_org_assignment_source,priority:1" json:"source_system_code"`
	SourceId         string     `gorm:"size:128;not null;uniqueIndex:uni_org_assignment_source,priority:2" json:"source_id"`
	EmployeeId       int        `gorm:"type:bigint;not null;index:idx_org_assignment_employee;uniqueIndex:uni_org_assignment_current_primary,where:is_primary = true AND status = 'enabled' AND source_deleted = false AND valid_to IS NULL" json:"employee_id"`
	LegalEntityId    int        `gorm:"type:bigint;not null;index:idx_org_assignment_legal_entity" json:"legal_entity_id"`
	OrgUnitId        int        `gorm:"type:bigint;not null;index:idx_org_assignment_org_unit" json:"org_unit_id"`
	PositionId       *int       `gorm:"type:bigint;index:idx_org_assignment_position" json:"position_id"`
	AssignmentType   string     `gorm:"size:32;not null;default:primary;index:idx_org_assignment_type" json:"assignment_type"`
	IsPrimary        bool       `gorm:"not null;default:false;index:idx_org_assignment_primary" json:"is_primary"`
	IsManager        bool       `gorm:"not null;default:false;index:idx_org_assignment_manager" json:"is_manager"`
	ValidFrom        *time.Time `gorm:"type:timestamp;index:idx_org_assignment_valid_from" json:"valid_from"`
	ValidTo          *time.Time `gorm:"type:timestamp;index:idx_org_assignment_valid_to" json:"valid_to"`
	Status           string     `gorm:"size:32;not null;default:enabled;index:idx_org_assignment_status" json:"status"`
	SourceVersion    string     `gorm:"size:64;index:idx_org_assignment_source_version" json:"source_version"`
	SourceDeleted    bool       `gorm:"not null;default:false;index:idx_org_assignment_source_deleted" json:"source_deleted"`
	SyncStatus       string     `gorm:"size:32;not null;default:pending;index:idx_org_assignment_sync_status" json:"sync_status"`

	Employee    *OrgEmployee    `gorm:"foreignKey:EmployeeId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"employee,omitempty"`
	LegalEntity *OrgLegalEntity `gorm:"foreignKey:LegalEntityId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"legal_entity,omitempty"`
	OrgUnit     *OrgUnit        `gorm:"foreignKey:OrgUnitId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"org_unit,omitempty"`
	Position    *OrgPosition    `gorm:"foreignKey:PositionId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"position,omitempty"`
}

// OrgSyncBatch 记录组织领域同步结果。
// HTTP 技术执行明细仍由集成中心负责。
type OrgSyncBatch struct {
	Basic

	BatchNo      string     `gorm:"size:64;not null;uniqueIndex:uni_org_sync_batch_no" json:"batch_no"`
	ExecutionId  *int       `gorm:"type:bigint;index:idx_org_sync_batch_execution" json:"execution_id"`
	SyncType     string     `gorm:"size:32;not null;default:incremental;index:idx_org_sync_batch_type" json:"sync_type"`
	ObjectScope  string     `gorm:"size:64;not null;default:all;index:idx_org_sync_batch_scope" json:"object_scope"`
	StartedAt    *time.Time `gorm:"type:timestamp;index:idx_org_sync_batch_started_at" json:"started_at"`
	CompletedAt  *time.Time `gorm:"type:timestamp;index:idx_org_sync_batch_completed_at" json:"completed_at"`
	TotalCount   int        `gorm:"not null;default:0" json:"total_count"`
	SuccessCount int        `gorm:"not null;default:0" json:"success_count"`
	FailedCount  int        `gorm:"not null;default:0" json:"failed_count"`
	SkippedCount int        `gorm:"not null;default:0" json:"skipped_count"`
	Status       string     `gorm:"size:32;not null;default:pending;index:idx_org_sync_batch_status" json:"status"`
	ErrorSummary string     `gorm:"type:text" json:"error_summary"`
}

// OrgSyncRecord 记录单个组织领域对象的处理结果。
type OrgSyncRecord struct {
	Basic

	BatchId        int        `gorm:"type:bigint;not null;index:idx_org_sync_record_batch" json:"batch_id"`
	ExecutionId    *int       `gorm:"type:bigint;index:idx_org_sync_record_execution" json:"execution_id"`
	ObjectType     string     `gorm:"size:64;not null;index:idx_org_sync_record_object_type" json:"object_type"`
	SourceId       string     `gorm:"size:128;index:idx_org_sync_record_source_id" json:"source_id"`
	SourceCode     string     `gorm:"size:128;index:idx_org_sync_record_source_code" json:"source_code"`
	LocalId        *int       `gorm:"type:bigint;index:idx_org_sync_record_local_id" json:"local_id"`
	Action         string     `gorm:"size:32;not null;index:idx_org_sync_record_action" json:"action"`
	Status         string     `gorm:"size:32;not null;default:pending;index:idx_org_sync_record_status" json:"status"`
	ErrorCode      string     `gorm:"size:64;index:idx_org_sync_record_error_code" json:"error_code"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message"`
	DependencyType string     `gorm:"size:64;index:idx_org_sync_record_dependency_type" json:"dependency_type"`
	DependencyKey  string     `gorm:"size:128;index:idx_org_sync_record_dependency_key" json:"dependency_key"`
	RetryCount     int        `gorm:"not null;default:0;index:idx_org_sync_record_retry_count" json:"retry_count"`
	LastRetryAt    *time.Time `gorm:"type:timestamp;index:idx_org_sync_record_last_retry_at" json:"last_retry_at"`

	// 平台管理扩展字段。
	LocalHandlingStatus string `gorm:"size:32;index:idx_org_sync_record_handling_status" json:"local_handling_status"`

	Batch *OrgSyncBatch `gorm:"foreignKey:BatchId;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"batch,omitempty"`
}
