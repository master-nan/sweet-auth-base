package service

import (
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/model"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestSubjectContextBuilderBuildsTrustedContext(t *testing.T) {
	requestedUserIds := make([]int, 0, 3)
	employeeId := 301
	clockCalls := 0
	builder := newSubjectContextBuilder(
		func(userId int) (model.SysUser, error) {
			requestedUserIds = append(requestedUserIds, userId)
			return activeSubjectUser(userId), nil
		},
		func(userId int) ([]model.SysRole, error) {
			requestedUserIds = append(requestedUserIds, userId)
			return []model.SysRole{
				activeSubjectRole(9),
				activeSubjectRole(3),
				activeSubjectRole(9),
				inactiveSubjectRole(5),
			}, nil
		},
		func(_ *gin.Context, userId int) (response.OrgEmployeeContextRes, error) {
			requestedUserIds = append(requestedUserIds, userId)
			return response.NewOrgEmployeeContextRes(userId, &employeeId), nil
		},
		func() time.Time {
			clockCalls++
			return time.Date(2026, time.July, 31, 16, 30, 0, 0, time.UTC)
		},
	)

	context, err := builder.Build(authenticatedSubjectContext(101), 101)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if context.UserId() != 101 || context.EmployeeId() != employeeId {
		t.Fatalf("Build() identity = user %d employee %d", context.UserId(), context.EmployeeId())
	}
	if !reflect.DeepEqual(context.RoleIds(), []int{3, 9}) {
		t.Fatalf("RoleIds() = %v, want [3 9]", context.RoleIds())
	}
	if context.AsOfDate() != "2026-08-01" {
		t.Fatalf("AsOfDate() = %q, want 2026-08-01", context.AsOfDate())
	}
	if !reflect.DeepEqual(requestedUserIds, []int{101, 101, 101}) {
		t.Fatalf("dependency user ids = %v, want trusted user only", requestedUserIds)
	}
	if clockCalls != 1 {
		t.Fatalf("clock calls = %d, want 1", clockCalls)
	}
}

func TestSubjectContextBuilderDoesNotInventRoleContext(t *testing.T) {
	employeeLookupCalled := false
	builder := newSubjectContextBuilder(
		func(userId int) (model.SysUser, error) { return activeSubjectUser(userId), nil },
		func(int) ([]model.SysRole, error) {
			return []model.SysRole{inactiveSubjectRole(1)}, nil
		},
		func(_ *gin.Context, userId int) (response.OrgEmployeeContextRes, error) {
			employeeLookupCalled = true
			employeeId := 301
			return response.NewOrgEmployeeContextRes(userId, &employeeId), nil
		},
		model.Now,
	)

	_, err := builder.Build(authenticatedSubjectContext(101), 101)
	assertSubjectContextBuilderError(t, err, myerrors.ErrorCodeDataPermissionRoleContextMissing)
	if employeeLookupCalled {
		t.Fatal("employee lookup should not run when effective role context is empty")
	}
}

func TestSubjectContextBuilderRejectsUnboundEmployee(t *testing.T) {
	builder := validSubjectContextBuilder(func(_ *gin.Context, userId int) (response.OrgEmployeeContextRes, error) {
		return response.NewOrgEmployeeContextRes(userId, nil), nil
	})

	_, err := builder.Build(authenticatedSubjectContext(101), 101)
	assertSubjectContextBuilderError(t, err, myerrors.ErrorCodeDataPermissionEmployeeUnbound)
}

func TestSubjectContextBuilderRejectsMissingUser(t *testing.T) {
	builder := newSubjectContextBuilder(
		func(int) (model.SysUser, error) { return model.SysUser{}, gorm.ErrRecordNotFound },
		func(int) ([]model.SysRole, error) { return nil, nil },
		func(*gin.Context, int) (response.OrgEmployeeContextRes, error) {
			return response.OrgEmployeeContextRes{}, nil
		},
		model.Now,
	)

	_, err := builder.Build(authenticatedSubjectContext(101), 101)
	assertSubjectContextBuilderError(t, err, myerrors.ErrorCodeDataPermissionSubjectUserNotFound)
}

func TestSubjectContextBuilderRejectsUntrustedUserOverride(t *testing.T) {
	lookupCalled := false
	builder := newSubjectContextBuilder(
		func(userId int) (model.SysUser, error) {
			lookupCalled = true
			return activeSubjectUser(userId), nil
		},
		func(int) ([]model.SysRole, error) { return []model.SysRole{activeSubjectRole(3)}, nil },
		func(_ *gin.Context, userId int) (response.OrgEmployeeContextRes, error) {
			employeeId := 301
			return response.NewOrgEmployeeContextRes(userId, &employeeId), nil
		},
		model.Now,
	)

	_, err := builder.Build(authenticatedSubjectContext(101), 202)
	assertSubjectContextBuilderError(t, err, myerrors.ErrorCodeDataPermissionSubjectContextInvalid)
	if lookupCalled {
		t.Fatal("dependencies must not run for an untrusted user override")
	}
}

func TestSubjectContextBuilderRejectsBindingDataAnomaly(t *testing.T) {
	tests := []struct {
		name    string
		binding func(*gin.Context, int) (response.OrgEmployeeContextRes, error)
	}{
		{
			name: "binding belongs to another user",
			binding: func(_ *gin.Context, _ int) (response.OrgEmployeeContextRes, error) {
				employeeId := 301
				return response.NewOrgEmployeeContextRes(202, &employeeId), nil
			},
		},
		{
			name: "bound status without employee",
			binding: func(_ *gin.Context, userId int) (response.OrgEmployeeContextRes, error) {
				return response.OrgEmployeeContextRes{
					UserId: userId, BindingStatus: response.OrgEmployeeBindingBound,
				}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validSubjectContextBuilder(tt.binding).
				Build(authenticatedSubjectContext(101), 101)
			assertSubjectContextBuilderError(
				t,
				err,
				myerrors.ErrorCodeDataPermissionSubjectContextInvalid,
			)
		})
	}
}

func TestSubjectContextBuilderWrapsRawDependencyErrors(t *testing.T) {
	builder := newSubjectContextBuilder(
		func(int) (model.SysUser, error) { return model.SysUser{}, errors.New("database unavailable") },
		func(int) ([]model.SysRole, error) { return nil, nil },
		func(*gin.Context, int) (response.OrgEmployeeContextRes, error) {
			return response.OrgEmployeeContextRes{}, nil
		},
		model.Now,
	)

	_, err := builder.Build(authenticatedSubjectContext(101), 101)
	var adminError *response.AdminError
	if !errors.As(err, &adminError) {
		t.Fatalf("Build() error = %T, want AdminError", err)
	}
	if adminError.Category != response.ErrorCategoryDatabase || adminError.ErrorCode != myerrors.ErrorCodeGeneric {
		t.Fatalf("Build() error = %+v, want stable database error", adminError)
	}
}

func validSubjectContextBuilder(
	findEmployee subjectContextEmployeeLookup,
) *SubjectContextBuilder {
	return newSubjectContextBuilder(
		func(userId int) (model.SysUser, error) { return activeSubjectUser(userId), nil },
		func(int) ([]model.SysRole, error) { return []model.SysRole{activeSubjectRole(3)}, nil },
		findEmployee,
		func() time.Time {
			return time.Date(2026, time.August, 1, 9, 0, 0, 0, model.AppLocation())
		},
	)
}

func authenticatedSubjectContext(userId int) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set("id", userId)
	return ctx
}

func activeSubjectUser(userId int) model.SysUser {
	return model.SysUser{Basic: model.Basic{Id: userId, State: true}}
}

func activeSubjectRole(roleId int) model.SysRole {
	return model.SysRole{Basic: model.Basic{Id: roleId, State: true}}
}

func inactiveSubjectRole(roleId int) model.SysRole {
	return model.SysRole{Basic: model.Basic{Id: roleId, State: false}}
}

func assertSubjectContextBuilderError(t *testing.T, err error, errorCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d", errorCode)
	}
	var adminError *response.AdminError
	if !errors.As(err, &adminError) {
		t.Fatalf("error = %T, want AdminError", err)
	}
	if adminError.ErrorCode != errorCode {
		t.Fatalf("error code = %d, want %d", adminError.ErrorCode, errorCode)
	}
}
