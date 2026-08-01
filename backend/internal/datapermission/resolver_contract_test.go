package datapermission_test

import (
	stderrors "errors"
	"testing"

	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"

	"github.com/gin-gonic/gin"
)

const (
	testResolverResource  = "service:tms.transport_order"
	testResolverOperation = "query"
)

func TestResolverContractConstructionAndNormalOutput(t *testing.T) {
	input := newResolverInput(t)
	var resolver datapermission.Resolver = datapermission.ResolverFunc(
		func(_ *gin.Context, received datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			if received.ResourceCode() != testResolverResource || received.Operation() != testResolverOperation {
				t.Fatalf("unexpected resolver input: %s %s", received.ResourceCode(), received.Operation())
			}
			if received.SubjectContext().EmployeeId() != input.SubjectContext().EmployeeId() {
				t.Fatal("resolver did not receive the trusted SubjectContext")
			}
			return datapermission.NewNoneResult(received.ResourceCode(), received.Operation())
		},
	)

	result, err := resolver.Resolve(nil, input)
	if err != nil {
		t.Fatalf("resolve contract: %v", err)
	}
	if result.ResourceCode() != testResolverResource ||
		result.Operation() != testResolverOperation ||
		result.Decision() != datapermission.DataScopeDecisionNone {
		t.Fatalf("unexpected result: resource=%s operation=%s decision=%s", result.ResourceCode(), result.Operation(), result.Decision())
	}
}

func TestResolverInputRejectsMissingResourceAndOperation(t *testing.T) {
	subject := newResolverSubject(t)

	_, err := datapermission.NewResolverInput(subject, "", testResolverOperation)
	assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionResolverResourceMissing)

	_, err = datapermission.NewResolverInput(subject, testResolverResource, "")
	assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionResolverOperationMissing)

	_, err = datapermission.NewResolverInput(subject, testResolverResource, "unsupported")
	assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionResolverOperationMissing)
}

func TestResolverInputRejectsInvalidSubjectContext(t *testing.T) {
	_, err := datapermission.NewResolverInput(
		datapermission.SubjectContext{},
		testResolverResource,
		testResolverOperation,
	)
	assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionSubjectUserNotFound)
}

func TestResolverContractRejectsResultIdentityConflict(t *testing.T) {
	resolver := datapermission.ResolverFunc(
		func(_ *gin.Context, _ datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			return datapermission.NewNoneResult("service:tms.other_resource", testResolverOperation)
		},
	)

	result, err := resolver.Resolve(nil, newResolverInput(t))
	assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionResolverConfigConflict)
	assertResolverDidNotReturnAccess(t, result)
}

func TestResolverContractFailsClosedOnErrorsAndInvalidResults(t *testing.T) {
	input := newResolverInput(t)

	t.Run("raw resolver failure", func(t *testing.T) {
		resolver := datapermission.ResolverFunc(
			func(_ *gin.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
				all, err := datapermission.NewAllResult(input.ResourceCode(), input.Operation())
				if err != nil {
					t.Fatalf("create all result: %v", err)
				}
				return all, stderrors.New("internal resolver failure")
			},
		)
		result, err := resolver.Resolve(nil, input)
		assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionResolverFailed)
		assertResolverDidNotReturnAccess(t, result)
	})

	t.Run("stable resolver error", func(t *testing.T) {
		resolver := datapermission.ResolverFunc(
			func(_ *gin.Context, _ datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
				return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverGrantMissing
			},
		)
		result, err := resolver.Resolve(nil, input)
		assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionResolverGrantMissing)
		assertResolverDidNotReturnAccess(t, result)
	})

	t.Run("invalid output", func(t *testing.T) {
		resolver := datapermission.ResolverFunc(
			func(_ *gin.Context, _ datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
				return datapermission.DataScopeResult{}, nil
			},
		)
		result, err := resolver.Resolve(nil, input)
		assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionResolverFailed)
		assertResolverDidNotReturnAccess(t, result)
	})

	t.Run("nil contract implementation", func(t *testing.T) {
		var resolver datapermission.Resolver = datapermission.ResolverFunc(nil)
		result, err := resolver.Resolve(nil, input)
		assertResolverErrorCode(t, err, myerrors.ErrorCodeDataPermissionResolverFailed)
		assertResolverDidNotReturnAccess(t, result)
	})
}

func TestResolverStableErrorContract(t *testing.T) {
	tests := []struct {
		err  error
		code int
	}{
		{myerrors.ErrDataPermissionResolverResourceMissing, myerrors.ErrorCodeDataPermissionResolverResourceMissing},
		{myerrors.ErrDataPermissionResolverOperationMissing, myerrors.ErrorCodeDataPermissionResolverOperationMissing},
		{myerrors.ErrDataPermissionResolverGrantMissing, myerrors.ErrorCodeDataPermissionResolverGrantMissing},
		{myerrors.ErrDataPermissionResolverPolicyInvalid, myerrors.ErrorCodeDataPermissionResolverPolicyInvalid},
		{myerrors.ErrDataPermissionResolverOwnershipMissing, myerrors.ErrorCodeDataPermissionResolverOwnershipMissing},
		{myerrors.ErrDataPermissionResolverDimensionFailed, myerrors.ErrorCodeDataPermissionResolverDimensionFailed},
		{myerrors.ErrDataPermissionResolverConfigConflict, myerrors.ErrorCodeDataPermissionResolverConfigConflict},
		{myerrors.ErrDataPermissionResolverFailed, myerrors.ErrorCodeDataPermissionResolverFailed},
	}
	for _, tt := range tests {
		assertResolverErrorCode(t, tt.err, tt.code)
	}
}

func newResolverInput(t *testing.T) datapermission.ResolverInput {
	t.Helper()
	input, err := datapermission.NewResolverInput(
		newResolverSubject(t),
		testResolverResource,
		testResolverOperation,
	)
	if err != nil {
		t.Fatalf("create resolver input: %v", err)
	}
	return input
}

func newResolverSubject(t *testing.T) datapermission.SubjectContext {
	t.Helper()
	employeeId := 301
	subject, err := datapermission.NewSubjectContext(101, []int{7, 3}, &employeeId, "2026-08-01")
	if err != nil {
		t.Fatalf("create subject context: %v", err)
	}
	return subject
}

func assertResolverErrorCode(t *testing.T, err error, code int) {
	t.Helper()
	var adminError *response.AdminError
	if !stderrors.As(err, &adminError) {
		t.Fatalf("expected AdminError, got %T: %v", err, err)
	}
	if adminError.ErrorCode != code {
		t.Fatalf("unexpected error code: got %d want %d", adminError.ErrorCode, code)
	}
}

func assertResolverDidNotReturnAccess(t *testing.T, result datapermission.DataScopeResult) {
	t.Helper()
	if result.Decision() == datapermission.DataScopeDecisionAll {
		t.Fatal("resolver error must never return all")
	}
	if result.Validate() == nil {
		t.Fatalf("resolver error must return no usable result, got decision %q", result.Decision())
	}
}
