package response

import (
	"backend/model"
	"encoding/json"
	"strings"
	"time"
)

// OrgBaseRes contains platform record identity and audit timestamps without
// exposing source identity or soft-delete internals.
type OrgBaseRes struct {
	Id        int              `json:"id"`
	GmtCreate model.CustomTime `json:"gmt_create"`
	GmtModify model.CustomTime `json:"gmt_modify"`
	State     bool             `json:"state"`
}

type OrgLegalEntityListRes struct {
	OrgBaseRes
	Code                    string     `json:"code"`
	Name                    string     `json:"name"`
	ShortName               string     `json:"short_name"`
	EntityType              string     `json:"entity_type"`
	ParentId                *int       `json:"parent_id"`
	UnifiedSocialCreditCode string     `json:"unified_social_credit_code"`
	AccountingCode          string     `json:"accounting_code"`
	Status                  string     `json:"status"`
	ValidFrom               *time.Time `json:"valid_from"`
	ValidTo                 *time.Time `json:"valid_to"`
}

type OrgLegalEntityDetailRes struct {
	OrgLegalEntityListRes
	LocalNote           string          `json:"local_note"`
	LocalTags           json.RawMessage `json:"local_tags,omitempty"`
	DisplayOrder        *int            `json:"display_order"`
	LocalHandlingStatus string          `json:"local_handling_status"`
}

// OrgLegalEntityTreeNodeRes is a presentation node for the legal-entity
// hierarchy. Value is always the internal legal_entity_id. Orphan is set when
// the referenced parent is absent from the visible result set.
type OrgLegalEntityTreeNodeRes struct {
	LegalEntityId int                         `json:"legal_entity_id"`
	Value         int                         `json:"value"`
	Label         string                      `json:"label"`
	Code          string                      `json:"code"`
	Name          string                      `json:"name"`
	ShortName     string                      `json:"short_name"`
	EntityType    string                      `json:"entity_type"`
	ParentId      *int                        `json:"parent_id"`
	Status        string                      `json:"status"`
	Disabled      bool                        `json:"disabled"`
	Orphan        bool                        `json:"orphan,omitempty"`
	Children      []OrgLegalEntityTreeNodeRes `json:"children,omitempty"`
}

// OrgSelectorOptionRes is the shared Organization selector wire format.
// Persisted values are platform internal IDs; labels are presentation only.
type OrgSelectorOptionRes struct {
	Value    int    `json:"value"`
	Label    string `json:"label"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}

type OrgUnitListRes struct {
	OrgBaseRes
	Code                 string     `json:"code"`
	Name                 string     `json:"name"`
	UnitType             string     `json:"unit_type"`
	PrimaryLegalEntityId *int       `json:"primary_legal_entity_id"`
	Status               string     `json:"status"`
	ValidFrom            *time.Time `json:"valid_from"`
	ValidTo              *time.Time `json:"valid_to"`
	DisplayOrder         *int       `json:"display_order"`
}

type OrgUnitDetailRes struct {
	OrgUnitListRes
	PrimaryLegalEntity  *OrgReferenceSummaryRes `json:"primary_legal_entity,omitempty"`
	LocalNote           string                  `json:"local_note"`
	LocalTags           json.RawMessage         `json:"local_tags,omitempty"`
	LocalHandlingStatus string                  `json:"local_handling_status"`
}

type OrgStructureListRes struct {
	OrgBaseRes
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	StructureType string     `json:"structure_type"`
	Status        string     `json:"status"`
	IsDefault     bool       `json:"is_default"`
	ValidFrom     *time.Time `json:"valid_from"`
	ValidTo       *time.Time `json:"valid_to"`
}

type OrgStructureDetailRes struct {
	OrgStructureListRes
}

// OrgReferenceSummaryRes is a source-safe organization reference used by
// details. Id always refers to a Sweet Platform internal object ID.
type OrgReferenceSummaryRes struct {
	Id   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// OrgStructureOrgTreeNodeRes keeps tree occurrence identity separate from the
// business organization identity. Consumers locate a node with StructureNodeId
// and persist OrgUnitId in business records.
type OrgStructureOrgTreeNodeRes struct {
	StructureNodeId int                          `json:"structure_node_id"`
	StructureId     int                          `json:"structure_id"`
	OrgUnitId       int                          `json:"org_unit_id"`
	ParentNodeId    *int                         `json:"parent_node_id"`
	Code            string                       `json:"code"`
	Name            string                       `json:"name"`
	UnitType        string                       `json:"unit_type"`
	Status          string                       `json:"status"`
	NodeStatus      string                       `json:"node_status"`
	Level           int                          `json:"level"`
	Sort            int                          `json:"sort"`
	Disabled        bool                         `json:"disabled"`
	Orphan          bool                         `json:"orphan,omitempty"`
	Children        []OrgStructureOrgTreeNodeRes `json:"children,omitempty"`
}

// OrgStructureNodeListRes intentionally omits Path and source parent data.
// Level is safe display metadata; tree persistence must still use OrgUnitId.
type OrgStructureNodeListRes struct {
	OrgBaseRes
	StructureId  int        `json:"structure_id"`
	OrgUnitId    int        `json:"org_unit_id"`
	ParentNodeId *int       `json:"parent_node_id"`
	Level        int        `json:"level"`
	Sort         int        `json:"sort"`
	Status       string     `json:"status"`
	ValidFrom    *time.Time `json:"valid_from"`
	ValidTo      *time.Time `json:"valid_to"`
}

type OrgStructureNodeDetailRes struct {
	OrgStructureNodeListRes
}

type OrgPositionListRes struct {
	OrgBaseRes
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	OrgUnitId         int        `json:"org_unit_id"`
	PositionType      string     `json:"position_type"`
	JobLevel          string     `json:"job_level"`
	IsManagerPosition bool       `json:"is_manager_position"`
	Status            string     `json:"status"`
	ValidFrom         *time.Time `json:"valid_from"`
	ValidTo           *time.Time `json:"valid_to"`
}

type OrgPositionDetailRes struct {
	OrgPositionListRes
	LocalNote string `json:"local_note"`
}

type OrgEmployeeListRes struct {
	OrgBaseRes
	EmployeeNo           string     `json:"employee_no"`
	Name                 string     `json:"name"`
	EmploymentStatus     string     `json:"employment_status"`
	PrimaryLegalEntityId *int       `json:"primary_legal_entity_id"`
	BoundUserId          *int       `json:"user_id"`
	ValidFrom            *time.Time `json:"valid_from"`
	ValidTo              *time.Time `json:"valid_to"`
}

// OrgEmployeeDetailRes exposes only masked contact values. BoundUserId is the
// linked Sweet Platform account and is not an employee identifier.
type OrgEmployeeDetailRes struct {
	OrgEmployeeListRes
	MobileMasked string          `json:"mobile_masked,omitempty"`
	EmailMasked  string          `json:"email_masked,omitempty"`
	LocalNote    string          `json:"local_note"`
	LocalTags    json.RawMessage `json:"local_tags,omitempty"`
}

type OrgAssignmentListRes struct {
	OrgBaseRes
	EmployeeId     int        `json:"employee_id"`
	LegalEntityId  int        `json:"legal_entity_id"`
	OrgUnitId      int        `json:"org_unit_id"`
	PositionId     *int       `json:"position_id"`
	AssignmentType string     `json:"assignment_type"`
	IsPrimary      bool       `json:"is_primary"`
	IsManager      bool       `json:"is_manager"`
	ValidFrom      *time.Time `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to"`
	Status         string     `json:"status"`
}

type OrgAssignmentDetailRes struct {
	OrgAssignmentListRes
}

type OrgSyncBatchListRes struct {
	OrgBaseRes
	BatchNo      string     `json:"batch_no"`
	ExecutionId  *int       `json:"execution_id"`
	SyncType     string     `json:"sync_type"`
	ObjectScope  string     `json:"object_scope"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	TotalCount   int        `json:"total_count"`
	SuccessCount int        `json:"success_count"`
	FailedCount  int        `json:"failed_count"`
	SkippedCount int        `json:"skipped_count"`
	Status       string     `json:"status"`
	HasError     bool       `json:"has_error"`
}

type OrgSyncBatchDetailRes struct {
	OrgSyncBatchListRes
}

// OrgSyncBatchErrorRes is reserved for a view_error-authorized endpoint.
type OrgSyncBatchErrorRes struct {
	Id           int    `json:"id"`
	ErrorSummary string `json:"error_summary"`
}

type OrgSyncRecordListRes struct {
	OrgBaseRes
	BatchId             int        `json:"batch_id"`
	ExecutionId         *int       `json:"execution_id"`
	ObjectType          string     `json:"object_type"`
	SourceCode          string     `json:"source_code"`
	LocalId             *int       `json:"local_id"`
	Action              string     `json:"action"`
	Status              string     `json:"status"`
	ErrorCode           string     `json:"error_code"`
	DependencyType      string     `json:"dependency_type"`
	RetryCount          int        `json:"retry_count"`
	LastRetryAt         *time.Time `json:"last_retry_at"`
	LocalHandlingStatus string     `json:"local_handling_status"`
	HasError            bool       `json:"has_error"`
}

type OrgSyncRecordDetailRes struct {
	OrgSyncRecordListRes
}

// OrgSyncRecordErrorRes is reserved for a view_error-authorized endpoint.
// Raw source identity remains internal even in this diagnostic response.
type OrgSyncRecordErrorRes struct {
	Id             int    `json:"id"`
	ErrorCode      string `json:"error_code"`
	ErrorMessage   string `json:"error_message"`
	DependencyType string `json:"dependency_type"`
	DependencyKey  string `json:"dependency_key"`
}

func NewOrgLegalEntityListRes(entity model.OrgLegalEntity) OrgLegalEntityListRes {
	return OrgLegalEntityListRes{
		OrgBaseRes:              newOrgBaseRes(entity.Basic),
		Code:                    entity.Code,
		Name:                    entity.Name,
		ShortName:               entity.ShortName,
		EntityType:              entity.EntityType,
		ParentId:                entity.ParentId,
		UnifiedSocialCreditCode: entity.UnifiedSocialCreditCode,
		AccountingCode:          entity.AccountingCode,
		Status:                  entity.Status,
		ValidFrom:               entity.ValidFrom,
		ValidTo:                 entity.ValidTo,
	}
}

func NewOrgLegalEntityDetailRes(entity model.OrgLegalEntity) OrgLegalEntityDetailRes {
	return OrgLegalEntityDetailRes{
		OrgLegalEntityListRes: NewOrgLegalEntityListRes(entity),
		LocalNote:             entity.LocalNote,
		LocalTags:             cloneRawJSON(entity.LocalTags),
		DisplayOrder:          entity.DisplayOrder,
		LocalHandlingStatus:   entity.LocalHandlingStatus,
	}
}

func NewOrgLegalEntityTreeNodeRes(entity model.OrgLegalEntity, disabled bool) OrgLegalEntityTreeNodeRes {
	return OrgLegalEntityTreeNodeRes{
		LegalEntityId: entity.Id,
		Value:         entity.Id,
		Label:         organizationDisplayLabel(entity.Code, entity.Name),
		Code:          entity.Code,
		Name:          entity.Name,
		ShortName:     entity.ShortName,
		EntityType:    entity.EntityType,
		ParentId:      entity.ParentId,
		Status:        entity.Status,
		Disabled:      disabled,
	}
}

func NewOrgLegalEntityOptionRes(entity model.OrgLegalEntity, disabled bool) OrgSelectorOptionRes {
	return OrgSelectorOptionRes{
		Value:    entity.Id,
		Label:    organizationDisplayLabel(entity.Code, entity.Name),
		Code:     entity.Code,
		Name:     entity.Name,
		Disabled: disabled,
	}
}

func NewOrgUnitListRes(unit model.OrgUnit) OrgUnitListRes {
	return OrgUnitListRes{
		OrgBaseRes:           newOrgBaseRes(unit.Basic),
		Code:                 unit.Code,
		Name:                 unit.Name,
		UnitType:             unit.UnitType,
		PrimaryLegalEntityId: unit.PrimaryLegalEntityId,
		Status:               unit.Status,
		ValidFrom:            unit.ValidFrom,
		ValidTo:              unit.ValidTo,
		DisplayOrder:         unit.DisplayOrder,
	}
}

func NewOrgUnitDetailRes(unit model.OrgUnit) OrgUnitDetailRes {
	return OrgUnitDetailRes{
		OrgUnitListRes:      NewOrgUnitListRes(unit),
		LocalNote:           unit.LocalNote,
		LocalTags:           cloneRawJSON(unit.LocalTags),
		LocalHandlingStatus: unit.LocalHandlingStatus,
	}
}

func NewOrgReferenceSummaryRes(id int, code, name string) OrgReferenceSummaryRes {
	return OrgReferenceSummaryRes{Id: id, Code: code, Name: name}
}

func NewOrgStructureListRes(structure model.OrgStructure) OrgStructureListRes {
	return OrgStructureListRes{
		OrgBaseRes:    newOrgBaseRes(structure.Basic),
		Code:          structure.Code,
		Name:          structure.Name,
		StructureType: structure.StructureType,
		Status:        structure.Status,
		IsDefault:     structure.IsDefault,
		ValidFrom:     structure.ValidFrom,
		ValidTo:       structure.ValidTo,
	}
}

func NewOrgStructureDetailRes(structure model.OrgStructure) OrgStructureDetailRes {
	return OrgStructureDetailRes{OrgStructureListRes: NewOrgStructureListRes(structure)}
}

func NewOrgStructureOptionRes(structure model.OrgStructure, disabled bool) OrgSelectorOptionRes {
	return OrgSelectorOptionRes{
		Value:    structure.Id,
		Label:    organizationDisplayLabel(structure.Code, structure.Name),
		Code:     structure.Code,
		Name:     structure.Name,
		Disabled: disabled,
	}
}

func NewOrgUnitOptionRes(unit model.OrgUnit, disabled bool) OrgSelectorOptionRes {
	return OrgSelectorOptionRes{
		Value:    unit.Id,
		Label:    organizationDisplayLabel(unit.Code, unit.Name),
		Code:     unit.Code,
		Name:     unit.Name,
		Disabled: disabled,
	}
}

func NewOrgStructureOrgTreeNodeRes(
	node model.OrgStructureNode,
	unit model.OrgUnit,
	disabled bool,
) OrgStructureOrgTreeNodeRes {
	return OrgStructureOrgTreeNodeRes{
		StructureNodeId: node.Id,
		StructureId:     node.StructureId,
		OrgUnitId:       unit.Id,
		ParentNodeId:    node.ParentNodeId,
		Code:            unit.Code,
		Name:            unit.Name,
		UnitType:        unit.UnitType,
		Status:          unit.Status,
		NodeStatus:      node.Status,
		Level:           node.Level,
		Sort:            node.Sort,
		Disabled:        disabled,
	}
}

func NewOrgStructureNodeListRes(node model.OrgStructureNode) OrgStructureNodeListRes {
	return OrgStructureNodeListRes{
		OrgBaseRes:   newOrgBaseRes(node.Basic),
		StructureId:  node.StructureId,
		OrgUnitId:    node.OrgUnitId,
		ParentNodeId: node.ParentNodeId,
		Level:        node.Level,
		Sort:         node.Sort,
		Status:       node.Status,
		ValidFrom:    node.ValidFrom,
		ValidTo:      node.ValidTo,
	}
}

func NewOrgStructureNodeDetailRes(node model.OrgStructureNode) OrgStructureNodeDetailRes {
	return OrgStructureNodeDetailRes{OrgStructureNodeListRes: NewOrgStructureNodeListRes(node)}
}

func NewOrgPositionListRes(position model.OrgPosition) OrgPositionListRes {
	return OrgPositionListRes{
		OrgBaseRes:        newOrgBaseRes(position.Basic),
		Code:              position.Code,
		Name:              position.Name,
		OrgUnitId:         position.OrgUnitId,
		PositionType:      position.PositionType,
		JobLevel:          position.JobLevel,
		IsManagerPosition: position.IsManagerPosition,
		Status:            position.Status,
		ValidFrom:         position.ValidFrom,
		ValidTo:           position.ValidTo,
	}
}

func NewOrgPositionDetailRes(position model.OrgPosition) OrgPositionDetailRes {
	return OrgPositionDetailRes{
		OrgPositionListRes: NewOrgPositionListRes(position),
		LocalNote:          position.LocalNote,
	}
}

func NewOrgEmployeeListRes(employee model.OrgEmployee) OrgEmployeeListRes {
	return OrgEmployeeListRes{
		OrgBaseRes:           newOrgBaseRes(employee.Basic),
		EmployeeNo:           employee.EmployeeNo,
		Name:                 employee.Name,
		EmploymentStatus:     employee.EmploymentStatus,
		PrimaryLegalEntityId: employee.PrimaryLegalEntityId,
		BoundUserId:          employee.UserId,
		ValidFrom:            employee.ValidFrom,
		ValidTo:              employee.ValidTo,
	}
}

func NewOrgEmployeeDetailRes(employee model.OrgEmployee) OrgEmployeeDetailRes {
	return OrgEmployeeDetailRes{
		OrgEmployeeListRes: NewOrgEmployeeListRes(employee),
		MobileMasked:       maskMobile(employee.Mobile),
		EmailMasked:        maskEmail(employee.Email),
		LocalNote:          employee.LocalNote,
		LocalTags:          cloneRawJSON(employee.LocalTags),
	}
}

func NewOrgAssignmentListRes(assignment model.OrgAssignment) OrgAssignmentListRes {
	return OrgAssignmentListRes{
		OrgBaseRes:     newOrgBaseRes(assignment.Basic),
		EmployeeId:     assignment.EmployeeId,
		LegalEntityId:  assignment.LegalEntityId,
		OrgUnitId:      assignment.OrgUnitId,
		PositionId:     assignment.PositionId,
		AssignmentType: assignment.AssignmentType,
		IsPrimary:      assignment.IsPrimary,
		IsManager:      assignment.IsManager,
		ValidFrom:      assignment.ValidFrom,
		ValidTo:        assignment.ValidTo,
		Status:         assignment.Status,
	}
}

func NewOrgAssignmentDetailRes(assignment model.OrgAssignment) OrgAssignmentDetailRes {
	return OrgAssignmentDetailRes{OrgAssignmentListRes: NewOrgAssignmentListRes(assignment)}
}

func NewOrgSyncBatchListRes(batch model.OrgSyncBatch) OrgSyncBatchListRes {
	return OrgSyncBatchListRes{
		OrgBaseRes:   newOrgBaseRes(batch.Basic),
		BatchNo:      batch.BatchNo,
		ExecutionId:  batch.ExecutionId,
		SyncType:     batch.SyncType,
		ObjectScope:  batch.ObjectScope,
		StartedAt:    batch.StartedAt,
		CompletedAt:  batch.CompletedAt,
		TotalCount:   batch.TotalCount,
		SuccessCount: batch.SuccessCount,
		FailedCount:  batch.FailedCount,
		SkippedCount: batch.SkippedCount,
		Status:       batch.Status,
		HasError:     strings.TrimSpace(batch.ErrorSummary) != "",
	}
}

func NewOrgSyncBatchDetailRes(batch model.OrgSyncBatch) OrgSyncBatchDetailRes {
	return OrgSyncBatchDetailRes{OrgSyncBatchListRes: NewOrgSyncBatchListRes(batch)}
}

func NewOrgSyncBatchErrorRes(batch model.OrgSyncBatch) OrgSyncBatchErrorRes {
	return OrgSyncBatchErrorRes{Id: batch.Id, ErrorSummary: batch.ErrorSummary}
}

func NewOrgSyncRecordListRes(record model.OrgSyncRecord) OrgSyncRecordListRes {
	return OrgSyncRecordListRes{
		OrgBaseRes:          newOrgBaseRes(record.Basic),
		BatchId:             record.BatchId,
		ExecutionId:         record.ExecutionId,
		ObjectType:          record.ObjectType,
		SourceCode:          record.SourceCode,
		LocalId:             record.LocalId,
		Action:              record.Action,
		Status:              record.Status,
		ErrorCode:           record.ErrorCode,
		DependencyType:      record.DependencyType,
		RetryCount:          record.RetryCount,
		LastRetryAt:         record.LastRetryAt,
		LocalHandlingStatus: record.LocalHandlingStatus,
		HasError:            strings.TrimSpace(record.ErrorMessage) != "",
	}
}

func NewOrgSyncRecordDetailRes(record model.OrgSyncRecord) OrgSyncRecordDetailRes {
	return OrgSyncRecordDetailRes{OrgSyncRecordListRes: NewOrgSyncRecordListRes(record)}
}

func NewOrgSyncRecordErrorRes(record model.OrgSyncRecord) OrgSyncRecordErrorRes {
	return OrgSyncRecordErrorRes{
		Id:             record.Id,
		ErrorCode:      record.ErrorCode,
		ErrorMessage:   record.ErrorMessage,
		DependencyType: record.DependencyType,
		DependencyKey:  record.DependencyKey,
	}
}

func newOrgBaseRes(basic model.Basic) OrgBaseRes {
	return OrgBaseRes{
		Id:        basic.Id,
		GmtCreate: basic.GmtCreate,
		GmtModify: basic.GmtModify,
		State:     basic.State,
	}
}

func cloneRawJSON(value []byte) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	cloned := make([]byte, len(value))
	copy(cloned, value)
	return json.RawMessage(cloned)
}

func organizationDisplayLabel(code, name string) string {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	switch {
	case code == "":
		return name
	case name == "":
		return code
	default:
		return code + " - " + name
	}
}

func maskMobile(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) < 7 {
		return "***"
	}
	return string(runes[:3]) + "****" + string(runes[len(runes)-4:])
}

func maskEmail(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	at := strings.LastIndex(value, "@")
	if at <= 0 || at == len(value)-1 {
		return "***"
	}
	local := []rune(value[:at])
	return string(local[:1]) + "***" + value[at:]
}
