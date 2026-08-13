package response

import (
	"backend/model"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// OrgBaseRes 包含平台记录身份和审计时间戳，不开放来源身份或软删除内部字段。
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

// OrgLegalEntityTreeNodeRes 是法人主体层级的展示节点。
// Value 始终为内部 legal_entity_id；引用的父节点不在可见结果集时设置 Orphan。
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

// OrgSelectorOptionRes 是 Organization 选择器的共享传输格式。
// 持久化值为平台内部 ID，Label 仅用于展示。
type OrgSelectorOptionRes struct {
	Value    int    `json:"value"`
	Label    string `json:"label"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}

// OrgSelectorOptionsRes 是四类业务选择器共享的 Service 响应。
// Controller 将 Items 映射到平台 Response.data，将 Total 映射到 Response.total，
// 不引入第二层 API 包装。
type OrgSelectorOptionsRes struct {
	Items []OrgSelectorOptionRes `json:"items"`
	Total int                    `json:"total"`
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

// OrgReferenceSummaryRes 是详情使用的来源安全组织引用。
// Id 始终指向 Sweet Platform 内部对象 ID。
type OrgReferenceSummaryRes struct {
	Id   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// OrgStructureOrgTreeNodeRes 将树节点身份与业务组织身份分离。
// 消费者使用 StructureNodeId 定位节点，业务记录保存 OrgUnitId。
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

// OrgStructureNodeListRes 有意省略 Path 和来源父级数据。
// Level 是安全的展示元数据，树相关业务持久化仍必须使用 OrgUnitId。
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
	OrgUnit     *OrgReferenceSummaryRes `json:"org_unit,omitempty"`
	LegalEntity *OrgReferenceSummaryRes `json:"legal_entity,omitempty"`
	LocalNote   string                  `json:"local_note"`
}

// OrgBoundUserSummaryRes 是普通组织读取开放的完整账号信息范围。
// 它不包含密码、Token、角色、登录计数及其他安全内部字段。
type OrgBoundUserSummaryRes struct {
	UserId   int    `json:"user_id"`
	UserName string `json:"user_name"`
}

type OrgEmployeeListRes struct {
	OrgBaseRes
	EmployeeNo           string                  `json:"employee_no"`
	Name                 string                  `json:"name"`
	EmploymentStatus     string                  `json:"employment_status"`
	PrimaryLegalEntityId *int                    `json:"primary_legal_entity_id"`
	BoundUserId          *int                    `json:"user_id"`
	BindingStatus        string                  `json:"binding_status"`
	BoundAccount         *OrgBoundUserSummaryRes `json:"bound_account,omitempty"`
	ValidFrom            *time.Time              `json:"valid_from"`
	ValidTo              *time.Time              `json:"valid_to"`
}

// OrgEmployeeDetailRes 仅开放脱敏联系方式。
// BoundUserId 是关联的 Sweet Platform 账号，不是员工标识。
type OrgEmployeeDetailRes struct {
	OrgEmployeeListRes
	MobileMasked string          `json:"mobile_masked,omitempty"`
	EmailMasked  string          `json:"email_masked,omitempty"`
	LocalNote    string          `json:"local_note"`
	LocalTags    json.RawMessage `json:"local_tags,omitempty"`
}

// OrgEmployeeUserBindingRes 是绑定和解绑操作的完整响应范围。
// 账号凭据、角色、权限和安全状态不属于此 DTO。
type OrgEmployeeUserBindingRes struct {
	EmployeeId    int                     `json:"employee_id"`
	UserId        *int                    `json:"user_id"`
	BindingStatus string                  `json:"binding_status"`
	BoundAccount  *OrgBoundUserSummaryRes `json:"bound_account,omitempty"`
}

// OrgEmployeeUserOptionRes 仅开放员工账号显式绑定 UI 所需的账号身份。
type OrgEmployeeUserOptionRes struct {
	Value    int    `json:"value"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
}

type OrgAssignmentListRes struct {
	OrgBaseRes
	EmployeeId     int                     `json:"employee_id"`
	LegalEntityId  int                     `json:"legal_entity_id"`
	OrgUnitId      int                     `json:"org_unit_id"`
	PositionId     *int                    `json:"position_id"`
	AssignmentType string                  `json:"assignment_type"`
	IsPrimary      bool                    `json:"is_primary"`
	IsManager      bool                    `json:"is_manager"`
	ValidFrom      *time.Time              `json:"valid_from"`
	ValidTo        *time.Time              `json:"valid_to"`
	Status         string                  `json:"status"`
	TimeScope      string                  `json:"time_scope"`
	LegalEntity    *OrgReferenceSummaryRes `json:"legal_entity,omitempty"`
	OrgUnit        *OrgReferenceSummaryRes `json:"org_unit,omitempty"`
	Position       *OrgReferenceSummaryRes `json:"position,omitempty"`
}

type OrgAssignmentDetailRes struct {
	OrgAssignmentListRes
}

// OrgEmployeeCurrentAssignmentSummaryRes 聚合全部当前任职。
// 集合中不设置主项或默认项。
type OrgEmployeeCurrentAssignmentSummaryRes struct {
	EmployeeId      int                      `json:"employee_id"`
	AsOfDate        string                   `json:"as_of_date"`
	AssignmentCount int                      `json:"assignment_count"`
	LegalEntities   []OrgReferenceSummaryRes `json:"legal_entities"`
	OrgUnits        []OrgReferenceSummaryRes `json:"org_units"`
	Positions       []OrgReferenceSummaryRes `json:"positions"`
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

// OrgSyncBatchErrorRes 仅用于具有 view_error 权限的接口。
type OrgSyncBatchErrorRes struct {
	Id           int    `json:"id"`
	ErrorSummary string `json:"error_summary"`
}

type OrgSyncRecordListRes struct {
	OrgBaseRes
	BatchId             int        `json:"batch_id"`
	ExecutionId         *int       `json:"execution_id"`
	ObjectType          string     `json:"object_type"`
	SourceSummary       string     `json:"source_summary"`
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

// OrgSyncRecordErrorRes 仅用于具有 view_error 权限的接口。
// 即使在此诊断响应中，原始来源身份仍保持为内部信息。
type OrgSyncRecordErrorRes struct {
	Id                int    `json:"id"`
	ErrorCode         string `json:"error_code"`
	DependencyType    string `json:"dependency_type"`
	DependencySummary string `json:"dependency_summary"`
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

func NewOrgPositionOptionRes(position model.OrgPosition, disabled bool) OrgSelectorOptionRes {
	return OrgSelectorOptionRes{
		Value:    position.Id,
		Label:    organizationDisplayLabel(position.Code, position.Name),
		Code:     position.Code,
		Name:     position.Name,
		Disabled: disabled,
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
		BindingStatus:        employeeBindingStatus(employee.UserId),
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

func NewOrgEmployeeOptionRes(employee model.OrgEmployee, disabled bool) OrgSelectorOptionRes {
	return OrgSelectorOptionRes{
		Value:    employee.Id,
		Label:    organizationDisplayLabel(employee.EmployeeNo, employee.Name),
		Code:     employee.EmployeeNo,
		Name:     employee.Name,
		Disabled: disabled,
	}
}

func NewOrgBoundUserSummaryRes(userId int, userName string) OrgBoundUserSummaryRes {
	return OrgBoundUserSummaryRes{UserId: userId, UserName: userName}
}

func NewOrgEmployeeUserOptionRes(userId int, userName string, disabled bool) OrgEmployeeUserOptionRes {
	return OrgEmployeeUserOptionRes{
		Value:    userId,
		Label:    userName,
		Disabled: disabled,
	}
}

func NewOrgEmployeeUserBindingRes(
	employeeId int,
	account *OrgBoundUserSummaryRes,
) OrgEmployeeUserBindingRes {
	result := OrgEmployeeUserBindingRes{
		EmployeeId:    employeeId,
		BindingStatus: "unbound",
	}
	if account == nil {
		return result
	}
	userId := account.UserId
	result.UserId = &userId
	result.BindingStatus = "bound"
	result.BoundAccount = account
	return result
}

func (r *OrgEmployeeListRes) SetBoundAccount(account OrgBoundUserSummaryRes) {
	if r == nil {
		return
	}
	r.BoundAccount = &account
}

func NewOrgAssignmentListRes(assignment model.OrgAssignment, timeScope string) OrgAssignmentListRes {
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
		TimeScope:      timeScope,
	}
}

func NewOrgAssignmentDetailRes(assignment model.OrgAssignment, timeScope string) OrgAssignmentDetailRes {
	return OrgAssignmentDetailRes{
		OrgAssignmentListRes: NewOrgAssignmentListRes(assignment, timeScope),
	}
}

func (r *OrgAssignmentListRes) SetReferences(
	legalEntity *OrgReferenceSummaryRes,
	orgUnit *OrgReferenceSummaryRes,
	position *OrgReferenceSummaryRes,
) {
	if r == nil {
		return
	}
	r.LegalEntity = legalEntity
	r.OrgUnit = orgUnit
	r.Position = position
}

func NewOrgEmployeeCurrentAssignmentSummaryRes(
	employeeId int,
	asOfDate string,
	assignmentCount int,
	legalEntities []OrgReferenceSummaryRes,
	orgUnits []OrgReferenceSummaryRes,
	positions []OrgReferenceSummaryRes,
) OrgEmployeeCurrentAssignmentSummaryRes {
	return OrgEmployeeCurrentAssignmentSummaryRes{
		EmployeeId:      employeeId,
		AsOfDate:        asOfDate,
		AssignmentCount: assignmentCount,
		LegalEntities:   legalEntities,
		OrgUnits:        orgUnits,
		Positions:       positions,
	}
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
	return OrgSyncBatchErrorRes{Id: batch.Id, ErrorSummary: safeOrganizationReasonCode(batch.ErrorSummary)}
}

func NewOrgSyncRecordListRes(record model.OrgSyncRecord) OrgSyncRecordListRes {
	return OrgSyncRecordListRes{
		OrgBaseRes:          newOrgBaseRes(record.Basic),
		BatchId:             record.BatchId,
		ExecutionId:         record.ExecutionId,
		ObjectType:          record.ObjectType,
		SourceSummary:       safeOrganizationSourceSummary(record.SourceId),
		LocalId:             record.LocalId,
		Action:              record.Action,
		Status:              record.Status,
		ErrorCode:           safeOrganizationReasonCode(record.ErrorCode),
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
		Id:                record.Id,
		ErrorCode:         safeOrganizationReasonCode(record.ErrorCode),
		DependencyType:    record.DependencyType,
		DependencySummary: safeOrganizationSourceSummary(record.DependencyKey),
	}
}

func safeOrganizationReasonCode(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < len("org_sync_")+1 || len(value) > 64 || !strings.HasPrefix(value, "org_sync_") {
		return ""
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' {
			continue
		}
		return ""
	}
	return value
}

// safeOrganizationSourceSummary accepts only values generated by the HR adapter.
// Legacy raw source identifiers and free-form dependency values remain redacted.
func safeOrganizationSourceSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "relation" {
		return value
	}
	if strings.HasPrefix(value, "issue-") {
		if number, err := strconv.Atoi(strings.TrimPrefix(value, "issue-")); err == nil && number > 0 {
			return value
		}
		return ""
	}
	if strings.HasPrefix(value, "assignment-local-") {
		digest := strings.TrimPrefix(value, "assignment-local-")
		if isOrganizationDigest(digest) {
			return value
		}
		return ""
	}
	if isOrganizationDigest(value) {
		return value
	}
	return ""
}

func isOrganizationDigest(value string) bool {
	if len(value) != 24 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
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

func employeeBindingStatus(userId *int) string {
	if userId == nil {
		return "unbound"
	}
	return "bound"
}
