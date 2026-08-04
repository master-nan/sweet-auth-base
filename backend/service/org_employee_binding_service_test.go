package service

import (
	"backend/dto/request"
	"backend/dto/response"
	apperrors "backend/internal/errors"
	"backend/internal/test"
	"backend/model"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type testTransactionalAuditWriter struct {
	mu      sync.Mutex
	records []TransactionalAuditRecord
	err     error
}

func (writer *testTransactionalAuditWriter) RecordTransactionalAudit(
	_ *gin.Context,
	tx *gorm.DB,
	record TransactionalAuditRecord,
) error {
	if tx == nil {
		return ErrTransactionDatabaseRequired
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.err != nil {
		return writer.err
	}
	writer.records = append(writer.records, record)
	return nil
}

func (writer *testTransactionalAuditWriter) snapshot() []TransactionalAuditRecord {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return append([]TransactionalAuditRecord(nil), writer.records...)
}

func TestOrgServiceBindEmployeeUser(t *testing.T) {
	t.Run("success records old and new account values", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{}
		orgService, db := newOrgServiceTestSubjectWithAuditWriter(t, auditWriter)
		employee := orgServiceEmployeeFixture(1, "EMP-BIND-1", "绑定人员", "active")
		user := employeeBindingUserFixture(101, "binding_user")
		testutil.MustCreate(t, db, &employee)
		testutil.MustCreate(t, db, &user)

		result, err := orgService.BindEmployeeUser(
			employeeBindingContext(employee.Id),
			request.OrgEmployeeBindUserReq{EmployeeId: employee.Id, UserId: user.Id},
		)
		if err != nil {
			t.Fatalf("bind employee user: %v", err)
		}
		if result.EmployeeId != employee.Id ||
			result.UserId == nil ||
			*result.UserId != user.Id ||
			result.BindingStatus != "bound" ||
			result.BoundAccount == nil ||
			result.BoundAccount.UserName != user.UserName {
			t.Fatalf("unexpected binding response: %+v", result)
		}

		var stored model.OrgEmployee
		if err = db.First(&stored, employee.Id).Error; err != nil {
			t.Fatalf("reload employee: %v", err)
		}
		if stored.UserId == nil || *stored.UserId != user.Id {
			t.Fatalf("stored user_id = %v, want %d", stored.UserId, user.Id)
		}
		records := auditWriter.snapshot()
		if len(records) != 1 {
			t.Fatalf("audit records = %d, want 1", len(records))
		}
		change := records[0].Changes["user_id"]
		if records[0].Action != orgEmployeeBindUserAction ||
			records[0].ResourceCode != "org_employee" ||
			records[0].ResourceId != "1" ||
			change.OldValue != nil ||
			change.NewValue != user.Id {
			t.Fatalf("unexpected bind audit: %+v", records[0])
		}
	})

	t.Run("employee not found", func(t *testing.T) {
		orgService, _ := newOrgServiceTestSubject(t)
		_, err := orgService.BindEmployeeUser(
			employeeBindingContext(999),
			request.OrgEmployeeBindUserReq{EmployeeId: 999, UserId: 101},
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgEmployeeNotFound,
		)
	})

	t.Run("user not found", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		employee := orgServiceEmployeeFixture(2, "EMP-BIND-2", "缺少账号", "active")
		testutil.MustCreate(t, db, &employee)
		_, err := orgService.BindEmployeeUser(
			employeeBindingContext(employee.Id),
			request.OrgEmployeeBindUserReq{EmployeeId: employee.Id, UserId: 999},
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgUserNotFound,
		)
	})

	t.Run("employee already has account", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		firstUser := employeeBindingUserFixture(103, "first_binding_user")
		secondUser := employeeBindingUserFixture(104, "second_binding_user")
		employee := orgServiceEmployeeFixture(3, "EMP-BIND-3", "已有账号", "active")
		employee.UserId = &firstUser.Id
		testutil.MustCreate(t, db, &[]model.SysUser{firstUser, secondUser})
		testutil.MustCreate(t, db, &employee)

		_, err := orgService.BindEmployeeUser(
			employeeBindingContext(employee.Id),
			request.OrgEmployeeBindUserReq{EmployeeId: employee.Id, UserId: secondUser.Id},
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgEmployeeAlreadyBound,
		)
	})

	t.Run("user already belongs to another employee", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		user := employeeBindingUserFixture(105, "already_bound_user")
		owner := orgServiceEmployeeFixture(4, "EMP-BIND-4", "原绑定人员", "active")
		owner.UserId = &user.Id
		target := orgServiceEmployeeFixture(5, "EMP-BIND-5", "目标人员", "active")
		testutil.MustCreate(t, db, &user)
		testutil.MustCreate(t, db, &[]model.OrgEmployee{owner, target})

		_, err := orgService.BindEmployeeUser(
			employeeBindingContext(target.Id),
			request.OrgEmployeeBindUserReq{EmployeeId: target.Id, UserId: user.Id},
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgUserAlreadyBound,
		)
	})
}

func TestOrgServiceBindEmployeeUserConcurrentConflict(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	user := employeeBindingUserFixture(201, "concurrent_binding_user")
	first := orgServiceEmployeeFixture(11, "EMP-CONCURRENT-1", "并发人员一", "active")
	second := orgServiceEmployeeFixture(12, "EMP-CONCURRENT-2", "并发人员二", "active")
	testutil.MustCreate(t, db, &user)
	testutil.MustCreate(t, db, &[]model.OrgEmployee{first, second})

	start := make(chan struct{})
	results := make(chan error, 2)
	for _, employeeId := range []int{first.Id, second.Id} {
		employeeId := employeeId
		go func() {
			<-start
			_, err := orgService.BindEmployeeUser(
				employeeBindingContext(employeeId),
				request.OrgEmployeeBindUserReq{EmployeeId: employeeId, UserId: user.Id},
			)
			results <- err
		}()
	}
	close(start)

	var successCount int
	var conflictCount int
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			successCount++
			continue
		}
		var adminErr *response.AdminError
		if errors.As(err, &adminErr) &&
			adminErr.ErrorCode == apperrors.ErrorCodeOrgUserAlreadyBound {
			conflictCount++
			continue
		}
		t.Fatalf("unexpected concurrent bind error: %v", err)
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("concurrent results success=%d conflict=%d, want 1/1", successCount, conflictCount)
	}

	var boundCount int64
	if err := db.Model(&model.OrgEmployee{}).
		Where("user_id = ?", user.Id).
		Count(&boundCount).Error; err != nil {
		t.Fatalf("count bound employees: %v", err)
	}
	if boundCount != 1 {
		t.Fatalf("bound employee count = %d, want 1", boundCount)
	}
}

func TestOrgServiceUnbindEmployeeUser(t *testing.T) {
	t.Run("success records old and new values", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{}
		orgService, db := newOrgServiceTestSubjectWithAuditWriter(t, auditWriter)
		user := employeeBindingUserFixture(301, "unbind_user")
		employee := orgServiceEmployeeFixture(21, "EMP-UNBIND-1", "解绑人员", "active")
		employee.UserId = &user.Id
		testutil.MustCreate(t, db, &user)
		testutil.MustCreate(t, db, &employee)

		result, err := orgService.UnbindEmployeeUser(
			employeeBindingContext(employee.Id),
			request.OrgEmployeeUnbindUserReq{EmployeeId: employee.Id},
		)
		if err != nil {
			t.Fatalf("unbind employee user: %v", err)
		}
		if result.EmployeeId != employee.Id ||
			result.UserId != nil ||
			result.BindingStatus != "unbound" ||
			result.BoundAccount != nil {
			t.Fatalf("unexpected unbind response: %+v", result)
		}

		var stored model.OrgEmployee
		if err = db.First(&stored, employee.Id).Error; err != nil {
			t.Fatalf("reload employee: %v", err)
		}
		if stored.UserId != nil {
			t.Fatalf("stored user_id = %v, want nil", stored.UserId)
		}
		records := auditWriter.snapshot()
		if len(records) != 1 {
			t.Fatalf("audit records = %d, want 1", len(records))
		}
		change := records[0].Changes["user_id"]
		if records[0].Action != orgEmployeeUnbindUserAction ||
			change.OldValue != user.Id ||
			change.NewValue != nil {
			t.Fatalf("unexpected unbind audit: %+v", records[0])
		}
	})

	t.Run("already unbound remains idempotent", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		employee := orgServiceEmployeeFixture(22, "EMP-UNBIND-2", "未绑定人员", "active")
		testutil.MustCreate(t, db, &employee)

		result, err := orgService.UnbindEmployeeUser(
			employeeBindingContext(employee.Id),
			request.OrgEmployeeUnbindUserReq{EmployeeId: employee.Id},
		)
		if err != nil {
			t.Fatalf("unbind already unbound employee: %v", err)
		}
		if result.BindingStatus != "unbound" || result.UserId != nil {
			t.Fatalf("unexpected idempotent unbind response: %+v", result)
		}
	})
}

func TestOrgServiceEmployeeUserBindingRollsBackWhenAuditFails(t *testing.T) {
	t.Run("bind", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{err: errors.New("audit storage failed")}
		orgService, db := newOrgServiceTestSubjectWithAuditWriter(t, auditWriter)
		user := employeeBindingUserFixture(401, "rollback_bind_user")
		employee := orgServiceEmployeeFixture(31, "EMP-ROLLBACK-1", "绑定回滚人员", "active")
		testutil.MustCreate(t, db, &user)
		testutil.MustCreate(t, db, &employee)

		_, err := orgService.BindEmployeeUser(
			employeeBindingContext(employee.Id),
			request.OrgEmployeeBindUserReq{EmployeeId: employee.Id, UserId: user.Id},
		)
		if err == nil {
			t.Fatal("expected bind audit failure")
		}
		assertEmployeeBindingUserId(t, db, employee.Id, nil)
	})

	t.Run("unbind", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{err: errors.New("audit storage failed")}
		orgService, db := newOrgServiceTestSubjectWithAuditWriter(t, auditWriter)
		user := employeeBindingUserFixture(402, "rollback_unbind_user")
		employee := orgServiceEmployeeFixture(32, "EMP-ROLLBACK-2", "解绑回滚人员", "active")
		employee.UserId = &user.Id
		testutil.MustCreate(t, db, &user)
		testutil.MustCreate(t, db, &employee)

		_, err := orgService.UnbindEmployeeUser(
			employeeBindingContext(employee.Id),
			request.OrgEmployeeUnbindUserReq{EmployeeId: employee.Id},
		)
		if err == nil {
			t.Fatal("expected unbind audit failure")
		}
		assertEmployeeBindingUserId(t, db, employee.Id, &user.Id)
	})
}

func employeeBindingContext(employeeId int) *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/admin/org/employee/"+strconv.Itoa(employeeId)+"/bind-user",
		nil,
	)
	ctx.Set("user", model.SysUser{
		Basic:    model.Basic{Id: 9001},
		UserName: "binding_operator",
	})
	ctx.Set(transactionalAuditRequestIDContextKey, "request-binding-test")
	ctx.Set(transactionalAuditTraceIDContextKey, "trace-binding-test")
	return ctx
}

func employeeBindingUserFixture(id int, userName string) model.SysUser {
	return model.SysUser{
		Basic:        model.Basic{Id: id, State: true},
		UserName:     userName,
		Password:     "password-must-not-leak",
		AccessTokens: "token-must-not-leak",
	}
}

func assertEmployeeBindingUserId(
	t *testing.T,
	db *gorm.DB,
	employeeId int,
	want *int,
) {
	t.Helper()
	var employee model.OrgEmployee
	if err := db.First(&employee, employeeId).Error; err != nil {
		t.Fatalf("reload employee %d: %v", employeeId, err)
	}
	if want == nil {
		if employee.UserId != nil {
			t.Fatalf("employee user_id = %v, want nil", employee.UserId)
		}
		return
	}
	if employee.UserId == nil || *employee.UserId != *want {
		t.Fatalf("employee user_id = %v, want %d", employee.UserId, *want)
	}
}
