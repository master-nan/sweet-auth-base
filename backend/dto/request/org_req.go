package request

// OrgLegalEntityQueryReq defines the repository-safe query fields for legal
// entities. Source identity and synchronization internals are intentionally
// excluded from user-facing query DTOs.
type OrgLegalEntityQueryReq struct {
	Basic
	SourceSystemCode string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	EntityType       string `form:"entity_type" json:"entity_type" binding:"omitempty,oneof=group legal_company branch accounting_unit"`
	ParentId         *int   `form:"parent_id" json:"parent_id" binding:"omitempty,gt=0"`
	Status           string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgUnitQueryReq defines the repository-safe query fields for management
// organization units.
type OrgUnitQueryReq struct {
	Basic
	SourceSystemCode     string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	UnitType             string `form:"unit_type" json:"unit_type" binding:"omitempty,oneof=business_unit region center department team project_group"`
	PrimaryLegalEntityId *int   `form:"primary_legal_entity_id" json:"primary_legal_entity_id" binding:"omitempty,gt=0"`
	Status               string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgStructureQueryReq defines the repository-safe query fields for management
// structure definitions.
type OrgStructureQueryReq struct {
	Basic
	SourceSystemCode string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	StructureType    string `form:"structure_type" json:"structure_type" binding:"omitempty,oneof=management"`
	Status           string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
	IsDefault        *bool  `form:"is_default" json:"is_default"`
}

// OrgStructureNodeQueryReq defines the repository-safe query fields for
// runtime organization tree nodes. Path is an internal query accelerator and
// is not accepted from ordinary clients.
type OrgStructureNodeQueryReq struct {
	Basic
	StructureId  *int   `form:"structure_id" json:"structure_id" binding:"omitempty,gt=0"`
	OrgUnitId    *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	ParentNodeId *int   `form:"parent_node_id" json:"parent_node_id" binding:"omitempty,gt=0"`
	Status       string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgEmployeeQueryReq never exposes mobile, email, source identity, or source
// version as ordinary query fields.
type OrgEmployeeQueryReq struct {
	Basic
	SourceSystemCode     string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	EmploymentStatus     string `form:"employment_status" json:"employment_status" binding:"omitempty,oneof=active probation suspended resigned retired"`
	PrimaryLegalEntityId *int   `form:"primary_legal_entity_id" json:"primary_legal_entity_id" binding:"omitempty,gt=0"`
	BoundUserId          *int   `form:"user_id" json:"user_id" binding:"omitempty,gt=0"`
}

// OrgPositionQueryReq defines the repository-safe query fields for positions.
type OrgPositionQueryReq struct {
	Basic
	SourceSystemCode  string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	OrgUnitId         *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionType      string `form:"position_type" json:"position_type" binding:"omitempty,oneof=management professional technical operation service"`
	IsManagerPosition *bool  `form:"is_manager_position" json:"is_manager_position"`
	Status            string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgAssignmentQueryReq defines the repository-safe query fields for employee
// assignments.
type OrgAssignmentQueryReq struct {
	Basic
	SourceSystemCode string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	EmployeeId       *int   `form:"employee_id" json:"employee_id" binding:"omitempty,gt=0"`
	LegalEntityId    *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId        *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionId       *int   `form:"position_id" json:"position_id" binding:"omitempty,gt=0"`
	AssignmentType   string `form:"assignment_type" json:"assignment_type" binding:"omitempty,oneof=primary part_time temporary project"`
	IsPrimary        *bool  `form:"is_primary" json:"is_primary"`
	IsManager        *bool  `form:"is_manager" json:"is_manager"`
	Status           string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgSyncBatchQueryReq is a read-only query contract for organization-domain
// synchronization batches. Integration payload fields are outside this DTO.
type OrgSyncBatchQueryReq struct {
	Basic
	ExecutionId *int   `form:"execution_id" json:"execution_id" binding:"omitempty,gt=0"`
	SyncType    string `form:"sync_type" json:"sync_type" binding:"omitempty,oneof=full incremental manual_retry"`
	ObjectScope string `form:"object_scope" json:"object_scope" binding:"omitempty,max=64"`
	Status      string `form:"status" json:"status" binding:"omitempty,oneof=pending processing success failed dependency_waiting ignored"`
}

// OrgSyncRecordQueryReq exposes diagnostic classification fields but not raw
// source identity, dependency keys, or error messages.
type OrgSyncRecordQueryReq struct {
	Basic
	BatchId             *int   `form:"batch_id" json:"batch_id" binding:"omitempty,gt=0"`
	ExecutionId         *int   `form:"execution_id" json:"execution_id" binding:"omitempty,gt=0"`
	ObjectType          string `form:"object_type" json:"object_type" binding:"omitempty,max=64"`
	Action              string `form:"action" json:"action" binding:"omitempty,oneof=insert update disable delete_to_disable skip no_change"`
	Status              string `form:"status" json:"status" binding:"omitempty,oneof=pending processing success failed dependency_waiting ignored"`
	DependencyType      string `form:"dependency_type" json:"dependency_type" binding:"omitempty,oneof=legal_entity org_unit structure_node employee position assignment"`
	LocalHandlingStatus string `form:"local_handling_status" json:"local_handling_status" binding:"omitempty,max=32"`
}
