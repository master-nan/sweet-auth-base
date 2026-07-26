package request

// OrgReadScopeReq defines the shared visibility controls for organization
// mirror reads. A nil OnlyEffective is normalized to true by OrgService.
type OrgReadScopeReq struct {
	OnlyEffective   *bool  `form:"only_effective" json:"only_effective"`
	IncludeDisabled bool   `form:"include_disabled" json:"include_disabled"`
	IncludeHistory  bool   `form:"include_history" json:"include_history"`
	AsOfDate        string `form:"as_of_date" json:"as_of_date" binding:"omitempty,datetime=2006-01-02"`
}

// OrgLegalEntityReadScopeReq is retained as the legal-entity API name for
// compatibility with E02-S02-T001.
type OrgLegalEntityReadScopeReq = OrgReadScopeReq

// OrgLegalEntityQueryReq defines the repository-safe query fields for legal
// entities. Source identity and synchronization internals are intentionally
// excluded from user-facing query DTOs.
type OrgLegalEntityQueryReq struct {
	Basic
	OrgLegalEntityReadScopeReq
	SourceSystemCode string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	EntityType       string `form:"entity_type" json:"entity_type" binding:"omitempty,oneof=group legal_company branch accounting_unit"`
	ParentId         *int   `form:"parent_id" json:"parent_id" binding:"omitempty,gt=0"`
	Status           string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgLegalEntityDetailReq controls visibility for one internal legal_entity_id.
type OrgLegalEntityDetailReq struct {
	OrgLegalEntityReadScopeReq
}

// OrgLegalEntityTreeReq requests a legal-entity tree built exclusively from
// org_legal_entity.parent_id.
type OrgLegalEntityTreeReq struct {
	OrgLegalEntityReadScopeReq
	RootId *int `form:"root_id" json:"root_id" binding:"omitempty,gt=0"`
}

// OrgLegalEntityOptionsReq supports remote option search and replay of
// already-persisted IDs. SelectedIds never changes the option value contract:
// values are always internal legal_entity_id values.
type OrgLegalEntityOptionsReq struct {
	OrgLegalEntityReadScopeReq
	Page        int    `form:"page" json:"page"`
	Num         int    `form:"num" json:"num"`
	Keyword     string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
	SelectedIds []int  `form:"selected_ids" json:"selected_ids" binding:"omitempty,max=100,dive,gt=0"`
}

// OrgUnitQueryReq defines the repository-safe query fields for management
// organization units.
type OrgUnitQueryReq struct {
	Basic
	OrgReadScopeReq
	SourceSystemCode     string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	UnitType             string `form:"unit_type" json:"unit_type" binding:"omitempty,oneof=business_unit region center department team project_group"`
	LegalEntityId        *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	PrimaryLegalEntityId *int   `form:"primary_legal_entity_id" json:"primary_legal_entity_id" binding:"omitempty,gt=0"`
	Status               string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgUnitDetailReq controls visibility for one internal org_unit_id.
type OrgUnitDetailReq struct {
	OrgReadScopeReq
}

// OrgUnitOptionsReq supports remote search and replay of persisted org_unit_id
// values. StructureId restricts candidates through org_structure_node.
type OrgUnitOptionsReq struct {
	OrgReadScopeReq
	Page          int    `form:"page" json:"page"`
	Num           int    `form:"num" json:"num"`
	Keyword       string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
	LegalEntityId *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	StructureId   *int   `form:"structure_id" json:"structure_id" binding:"omitempty,gt=0"`
	SelectedIds   []int  `form:"selected_ids" json:"selected_ids" binding:"omitempty,max=100,dive,gt=0"`
}

// OrgStructureQueryReq defines the repository-safe query fields for management
// structure definitions.
type OrgStructureQueryReq struct {
	Basic
	OrgReadScopeReq
	SourceSystemCode string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	LegalEntityId    *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	StructureType    string `form:"structure_type" json:"structure_type" binding:"omitempty,oneof=management"`
	Status           string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
	IsDefault        *bool  `form:"is_default" json:"is_default"`
}

// OrgStructureDetailReq controls visibility for one internal structure_id.
type OrgStructureDetailReq struct {
	OrgReadScopeReq
}

// OrgStructureOptionsReq supports remote option search and replay of persisted
// structure_id values.
type OrgStructureOptionsReq struct {
	OrgReadScopeReq
	Page          int    `form:"page" json:"page"`
	Num           int    `form:"num" json:"num"`
	Keyword       string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
	LegalEntityId *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	SelectedIds   []int  `form:"selected_ids" json:"selected_ids" binding:"omitempty,max=100,dive,gt=0"`
}

// OrgStructureOrgTreeReq requests a management tree. RootNodeId identifies one
// occurrence in the structure; RootOrgUnitId is a business-unit convenience
// lookup and must resolve to exactly one visible node.
type OrgStructureOrgTreeReq struct {
	OrgReadScopeReq
	StructureId   int    `form:"structure_id" json:"structure_id" binding:"required,gt=0"`
	RootNodeId    *int   `form:"root_node_id" json:"root_node_id" binding:"omitempty,gt=0"`
	RootOrgUnitId *int   `form:"root_org_unit_id" json:"root_org_unit_id" binding:"omitempty,gt=0"`
	Keyword       string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
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
// version as ordinary advanced-query fields. Relationship filters are resolved
// through one matching org_assignment row by the repository.
type OrgEmployeeQueryReq struct {
	Basic
	OrgReadScopeReq
	SourceSystemCode     string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	EmploymentStatus     string `form:"employment_status" json:"employment_status" binding:"omitempty,oneof=active probation suspended resigned retired"`
	PrimaryLegalEntityId *int   `form:"primary_legal_entity_id" json:"primary_legal_entity_id" binding:"omitempty,gt=0"`
	BoundUserId          *int   `form:"user_id" json:"user_id" binding:"omitempty,gt=0"`
	LegalEntityId        *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId            *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionId           *int   `form:"position_id" json:"position_id" binding:"omitempty,gt=0"`
	BoundStatus          string `form:"bound_status" json:"bound_status" binding:"omitempty,oneof=all bound unbound"`
}

// OrgEmployeeDetailReq controls visibility for one internal employee_id.
type OrgEmployeeDetailReq struct {
	OrgReadScopeReq
}

// OrgEmployeeOptionsReq supports remote search and replay of persisted
// employee_id values. Values never use user_id, names, or contact details.
type OrgEmployeeOptionsReq struct {
	OrgReadScopeReq
	Page          int    `form:"page" json:"page"`
	Num           int    `form:"num" json:"num"`
	Keyword       string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
	LegalEntityId *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId     *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionId    *int   `form:"position_id" json:"position_id" binding:"omitempty,gt=0"`
	SelectedIds   []int  `form:"selected_ids" json:"selected_ids" binding:"omitempty,max=100,dive,gt=0"`
}

// OrgPositionQueryReq defines the repository-safe query fields for positions.
type OrgPositionQueryReq struct {
	Basic
	OrgReadScopeReq
	SourceSystemCode  string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	LegalEntityId     *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId         *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionType      string `form:"position_type" json:"position_type" binding:"omitempty,oneof=management professional technical operation service"`
	IsManagerPosition *bool  `form:"is_manager_position" json:"is_manager_position"`
	Status            string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgPositionDetailReq controls visibility for one internal position_id.
type OrgPositionDetailReq struct {
	OrgReadScopeReq
}

// OrgPositionOptionsReq supports remote search and replay of persisted
// position_id values.
type OrgPositionOptionsReq struct {
	OrgReadScopeReq
	Page          int    `form:"page" json:"page"`
	Num           int    `form:"num" json:"num"`
	Keyword       string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
	LegalEntityId *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId     *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	SelectedIds   []int  `form:"selected_ids" json:"selected_ids" binding:"omitempty,max=100,dive,gt=0"`
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
