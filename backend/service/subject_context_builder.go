package service

import (
	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type subjectContextUserLookup func(int) (model.SysUser, error)
type subjectContextRoleLookup func(int) ([]model.SysRole, error)
type subjectContextEmployeeLookup func(*gin.Context, int) (response.OrgEmployeeContextRes, error)

// SubjectContextBuilder 仅从服务端可信来源生成 Resolver 主体身份。
// 它不接受角色、员工或日期覆盖值。
type SubjectContextBuilder struct {
	findUser     subjectContextUserLookup
	findRoles    subjectContextRoleLookup
	findEmployee subjectContextEmployeeLookup
	currentTime  func() time.Time
}

func NewSubjectContextBuilder(
	userRepo repository.SysUserRepository,
	userRoleRepo repository.SysUserRoleRepository,
	orgProvider OrgPermissionProvider,
) *SubjectContextBuilder {
	return newSubjectContextBuilder(
		userRepo.FindById,
		userRoleRepo.GetUserRoles,
		func(ctx *gin.Context, userId int) (response.OrgEmployeeContextRes, error) {
			requestContext := context.Background()
			if ctx != nil && ctx.Request != nil {
				requestContext = ctx.Request.Context()
			}
			return orgProvider.GetEmployeeByUser(requestContext, userId)
		},
		model.Now,
	)
}

func newSubjectContextBuilder(
	findUser subjectContextUserLookup,
	findRoles subjectContextRoleLookup,
	findEmployee subjectContextEmployeeLookup,
	currentTime func() time.Time,
) *SubjectContextBuilder {
	return &SubjectContextBuilder{
		findUser:     findUser,
		findRoles:    findRoles,
		findEmployee: findEmployee,
		currentTime:  currentTime,
	}
}

// Build 校验 userId 是否为 AuthHandler 设置的身份，再从服务端 Repository 和 Provider 生成其余字段。
func (builder *SubjectContextBuilder) Build(
	ctx *gin.Context,
	userId int,
) (datapermission.SubjectContext, error) {
	if err := validateTrustedSubjectUser(ctx, userId); err != nil {
		return datapermission.SubjectContext{}, err
	}
	if builder == nil || builder.findUser == nil || builder.findRoles == nil ||
		builder.findEmployee == nil || builder.currentTime == nil {
		return datapermission.SubjectContext{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}

	user, err := builder.findUser(userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datapermission.SubjectContext{}, myerrors.ErrDataPermissionSubjectUserNotFound
		}
		return datapermission.SubjectContext{}, wrapSubjectContextDependencyError(err)
	}
	if user.Id == 0 {
		return datapermission.SubjectContext{}, myerrors.ErrDataPermissionSubjectUserNotFound
	}
	if user.Id != userId || !user.State || user.GmtDelete.Valid {
		return datapermission.SubjectContext{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}

	roles, err := builder.findRoles(userId)
	if err != nil {
		return datapermission.SubjectContext{}, wrapSubjectContextDependencyError(err)
	}
	roleIds, err := effectiveSubjectRoleIds(roles)
	if err != nil {
		return datapermission.SubjectContext{}, err
	}
	if len(roleIds) == 0 {
		return datapermission.SubjectContext{}, myerrors.ErrDataPermissionRoleContextMissing
	}

	employee, err := builder.findEmployee(ctx, userId)
	if err != nil {
		if errors.Is(err, myerrors.ErrOrgUserNotFound) {
			return datapermission.SubjectContext{}, myerrors.ErrDataPermissionSubjectContextInvalid
		}
		return datapermission.SubjectContext{}, wrapSubjectContextDependencyError(err)
	}
	if employee.UserId != userId {
		return datapermission.SubjectContext{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}
	if employee.BindingStatus == response.OrgEmployeeBindingUnbound && employee.EmployeeId == nil {
		return datapermission.SubjectContext{}, myerrors.ErrDataPermissionEmployeeUnbound
	}
	if employee.BindingStatus != response.OrgEmployeeBindingBound ||
		employee.EmployeeId == nil || *employee.EmployeeId <= 0 {
		return datapermission.SubjectContext{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}

	now := builder.currentTime()
	if now.IsZero() {
		return datapermission.SubjectContext{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}
	asOfDate := now.In(model.AppLocation()).Format(time.DateOnly)
	return datapermission.NewSubjectContext(userId, roleIds, employee.EmployeeId, asOfDate)
}

func validateTrustedSubjectUser(ctx *gin.Context, userId int) error {
	if userId <= 0 {
		return myerrors.ErrDataPermissionSubjectUserNotFound
	}
	if ctx == nil {
		return myerrors.ErrDataPermissionSubjectContextInvalid
	}
	trustedUser, exists := ctx.Get("id")
	trustedUserId, valid := trustedUser.(int)
	if !exists || !valid || trustedUserId <= 0 || trustedUserId != userId {
		return myerrors.ErrDataPermissionSubjectContextInvalid
	}
	return nil
}

func effectiveSubjectRoleIds(roles []model.SysRole) ([]int, error) {
	roleIds := make([]int, 0, len(roles))
	for _, role := range roles {
		if role.GmtDelete.Valid || !role.State {
			continue
		}
		if role.Id <= 0 {
			return nil, myerrors.ErrDataPermissionSubjectContextInvalid
		}
		roleIds = append(roleIds, role.Id)
	}
	return roleIds, nil
}

func wrapSubjectContextDependencyError(err error) error {
	var adminError *response.AdminError
	if errors.As(err, &adminError) {
		return err
	}
	return myerrors.WrapDatabaseError(err)
}
