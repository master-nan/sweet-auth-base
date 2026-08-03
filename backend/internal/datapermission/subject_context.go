package datapermission

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	myerrors "backend/internal/errors"
)

// SubjectContext 是单次数据权限解析的不可变身份输入。
// 它不包含组织范围和权限决策。
type SubjectContext struct {
	userId     int
	roleIds    []int
	employeeId int
	asOfDate   string
}

// NewSubjectContext 创建经过校验且结果确定的 Resolver 主体。
func NewSubjectContext(
	userId int,
	roleIds []int,
	employeeId *int,
	asOfDate string,
) (SubjectContext, error) {
	context := SubjectContext{
		userId:   userId,
		roleIds:  normalizeRoleIds(roleIds),
		asOfDate: strings.TrimSpace(asOfDate),
	}
	if employeeId != nil {
		context.employeeId = *employeeId
	}

	if userId <= 0 {
		return SubjectContext{}, myerrors.ErrDataPermissionSubjectUserNotFound
	}
	if len(roleIds) == 0 {
		return SubjectContext{}, myerrors.ErrDataPermissionRoleContextMissing
	}
	if employeeId == nil {
		return SubjectContext{}, myerrors.ErrDataPermissionEmployeeUnbound
	}
	if err := context.Validate(); err != nil {
		return SubjectContext{}, err
	}
	return context, nil
}

// Validate 仅校验固有身份和日期不变量，不查询用户、角色、员工、组织或权限。
func (context SubjectContext) Validate() error {
	if context.userId <= 0 {
		return myerrors.ErrDataPermissionSubjectUserNotFound
	}
	if len(context.roleIds) == 0 {
		return myerrors.ErrDataPermissionRoleContextMissing
	}
	for _, roleId := range context.roleIds {
		if roleId <= 0 {
			return myerrors.ErrDataPermissionSubjectContextInvalid
		}
	}
	if context.employeeId <= 0 {
		return myerrors.ErrDataPermissionSubjectContextInvalid
	}
	if _, err := time.Parse(time.DateOnly, context.asOfDate); err != nil {
		return myerrors.ErrDataPermissionSubjectContextInvalid
	}
	return nil
}

func (context SubjectContext) UserId() int {
	return context.userId
}

func (context SubjectContext) RoleIds() []int {
	return append([]int(nil), context.roleIds...)
}

func (context SubjectContext) EmployeeId() int {
	return context.employeeId
}

func (context SubjectContext) AsOfDate() string {
	return context.asOfDate
}

func (context SubjectContext) MarshalJSON() ([]byte, error) {
	if err := context.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		UserId     int    `json:"user_id"`
		RoleIds    []int  `json:"role_ids"`
		EmployeeId int    `json:"employee_id"`
		AsOfDate   string `json:"as_of_date"`
	}{
		UserId:     context.userId,
		RoleIds:    context.RoleIds(),
		EmployeeId: context.employeeId,
		AsOfDate:   context.asOfDate,
	})
}

func normalizeRoleIds(roleIds []int) []int {
	normalized := append([]int(nil), roleIds...)
	sort.Ints(normalized)
	if len(normalized) < 2 {
		return normalized
	}

	writeIndex := 1
	for readIndex := 1; readIndex < len(normalized); readIndex++ {
		if normalized[readIndex] == normalized[writeIndex-1] {
			continue
		}
		normalized[writeIndex] = normalized[readIndex]
		writeIndex++
	}
	return normalized[:writeIndex]
}
