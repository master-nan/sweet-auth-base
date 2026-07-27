package response

import "time"

const (
	OrgEmployeeBindingBound   = "bound"
	OrgEmployeeBindingUnbound = "unbound"

	OrgEffectiveScopeResolved = "resolved"
	OrgEffectiveScopeEmpty    = "empty"
)

// OrgEmployeeContextRes is the minimum account-to-employee contract exposed
// to platform consumers. Account profile and authorization fields stay out.
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

// OrgEffectiveAssignmentRes exposes only organization facts required by
// permission and workflow consumers. Primary-assignment semantics are absent.
type OrgEffectiveAssignmentRes struct {
	AssignmentId  int        `json:"assignment_id"`
	EmployeeId    int        `json:"employee_id"`
	LegalEntityId int        `json:"legal_entity_id"`
	OrgUnitId     int        `json:"org_unit_id"`
	PositionId    *int       `json:"position_id"`
	ValidFrom     *time.Time `json:"valid_from"`
	ValidTo       *time.Time `json:"valid_to"`
}

// OrgEffectiveOrganizationScopeRes aggregates every effective assignment.
// Empty means no effective organization scope and never means unrestricted.
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
