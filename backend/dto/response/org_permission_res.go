package response

import "time"

const (
	OrgEmployeeBindingBound   = "bound"
	OrgEmployeeBindingUnbound = "unbound"

	OrgEffectiveScopeResolved = "resolved"
	OrgEffectiveScopeEmpty    = "empty"
)

// OrgEmployeeContextRes 是向平台消费者开放的最小账号与员工关联契约。
// 它不包含账号资料和授权字段。
type OrgEmployeeContextRes struct {
	UserId        int    `json:"user_id"`
	EmployeeId    *int   `json:"employee_id"`
	BindingStatus string `json:"binding_status"`
}

func NewOrgEmployeeContextRes(userId int, employeeId *int) OrgEmployeeContextRes {
	status := OrgEmployeeBindingUnbound
	if employeeId != nil {
		status = OrgEmployeeBindingBound
	}
	return OrgEmployeeContextRes{
		UserId:        userId,
		EmployeeId:    employeeId,
		BindingStatus: status,
	}
}

// OrgEffectiveAssignmentRes 仅开放权限和工作流消费者需要的组织事实。
// 它不包含主任职语义。
type OrgEffectiveAssignmentRes struct {
	AssignmentId  int        `json:"assignment_id"`
	EmployeeId    int        `json:"employee_id"`
	LegalEntityId int        `json:"legal_entity_id"`
	OrgUnitId     int        `json:"org_unit_id"`
	PositionId    *int       `json:"position_id"`
	ValidFrom     *time.Time `json:"valid_from"`
	ValidTo       *time.Time `json:"valid_to"`
}

// OrgEffectiveOrganizationScopeRes 聚合全部有效任职。
// 空集合表示没有有效组织范围，绝不表示不受限制。
type OrgEffectiveOrganizationScopeRes struct {
	EmployeeId      int    `json:"employee_id"`
	AsOfDate        string `json:"as_of_date"`
	ScopeStatus     string `json:"scope_status"`
	AssignmentCount int    `json:"assignment_count"`
	LegalEntityIds  []int  `json:"legal_entity_ids"`
	OrgUnitIds      []int  `json:"org_unit_ids"`
}

type OrgRelationItemRes struct {
	OrgUnitId int `json:"org_unit_id"`
	Distance  int `json:"distance"`
}

type OrgAncestorsRes struct {
	StructureCode string               `json:"structure_code"`
	OrgUnitId     int                  `json:"org_unit_id"`
	AsOfDate      string               `json:"as_of_date"`
	Items         []OrgRelationItemRes `json:"items"`
}

type OrgDescendantsRes struct {
	StructureCode string               `json:"structure_code"`
	OrgUnitId     int                  `json:"org_unit_id"`
	AsOfDate      string               `json:"as_of_date"`
	Items         []OrgRelationItemRes `json:"items"`
}

type OrgDescendantCheckRes struct {
	StructureCode       string `json:"structure_code"`
	AncestorOrgUnitId   int    `json:"ancestor_org_unit_id"`
	DescendantOrgUnitId int    `json:"descendant_org_unit_id"`
	AsOfDate            string `json:"as_of_date"`
	IsDescendant        bool   `json:"is_descendant"`
	Distance            *int   `json:"distance"`
}
