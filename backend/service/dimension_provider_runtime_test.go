package service

import (
	"errors"
	"reflect"
	"testing"

	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestDimensionProviderRuntimeResolvesOrganizationDimensions(t *testing.T) {
	subject := dimensionProviderSubject(t)
	tests := []struct {
		name          string
		dimensionCode string
		want          []int64
	}{
		{name: "management organization", dimensionCode: datapermission.DimensionCodeManagementOrg, want: []int64{11, 12}},
		{name: "legal entity", dimensionCode: datapermission.DimensionCodeLegalEntity, want: []int64{21, 22}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			runtime := newDimensionProviderRuntime(
				dimensionLookup(dimensionFixture(tt.dimensionCode, model.DataDimensionValueTypeBigint)),
				func(_ *gin.Context, employeeId int, asOfDate string) (response.OrgEffectiveOrganizationScopeRes, error) {
					calls++
					if employeeId != subject.EmployeeId() || asOfDate != subject.AsOfDate() {
						t.Fatalf("unexpected Organization Provider input: %d %s", employeeId, asOfDate)
					}
					return resolvedDimensionScope(subject, []int{22, 21, 22}, []int{12, 11, 12}), nil
				},
			)

			result, err := runtime.ResolveDimensionValues(nil, subject, tt.dimensionCode)
			if err != nil {
				t.Fatalf("resolve dimension: %v", err)
			}
			if calls != 1 {
				t.Fatalf("Organization Provider calls: got %d want 1", calls)
			}
			if result.DimensionCode() != tt.dimensionCode ||
				result.ValueType() != datapermission.DataScopeValueTypeBigint ||
				!reflect.DeepEqual(result.BigintValues(), tt.want) {
				t.Fatalf("unexpected result: code=%s type=%s values=%v", result.DimensionCode(), result.ValueType(), result.BigintValues())
			}
		})
	}
}

func TestDimensionProviderRuntimeReturnsEmployeeWithoutOrganizationLookup(t *testing.T) {
	subject := dimensionProviderSubject(t)
	runtime := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeEmployee, model.DataDimensionValueTypeBigint)),
		func(_ *gin.Context, _ int, _ string) (response.OrgEffectiveOrganizationScopeRes, error) {
			t.Fatal("employee dimension must not call Organization scope Provider")
			return response.OrgEffectiveOrganizationScopeRes{}, nil
		},
	)

	result, err := runtime.ResolveDimensionValues(nil, subject, datapermission.DimensionCodeEmployee)
	if err != nil {
		t.Fatalf("resolve employee dimension: %v", err)
	}
	if got, want := result.BigintValues(), []int64{int64(subject.EmployeeId())}; !reflect.DeepEqual(got, want) {
		t.Fatalf("employee values: got %v want %v", got, want)
	}
	if result.BigintValues()[0] == int64(subject.UserId()) {
		t.Fatal("employee dimension must not use user_id")
	}
}

func TestDimensionProviderRuntimeReturnsEmptyOrganizationFacts(t *testing.T) {
	subject := dimensionProviderSubject(t)
	runtime := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeManagementOrg, model.DataDimensionValueTypeBigint)),
		func(_ *gin.Context, _ int, _ string) (response.OrgEffectiveOrganizationScopeRes, error) {
			return response.OrgEffectiveOrganizationScopeRes{
				EmployeeId:      subject.EmployeeId(),
				AsOfDate:        subject.AsOfDate(),
				ScopeStatus:     response.OrgEffectiveScopeEmpty,
				AssignmentCount: 0,
				LegalEntityIds:  []int{},
				OrgUnitIds:      []int{},
			}, nil
		},
	)

	result, err := runtime.ResolveDimensionValues(nil, subject, datapermission.DimensionCodeManagementOrg)
	if err != nil {
		t.Fatalf("resolve empty dimension: %v", err)
	}
	if len(result.Values()) != 0 {
		t.Fatalf("empty scope must not expand access: %v", result.Values())
	}
}

func TestDimensionProviderRuntimeRejectsMissingAndUnsupportedDimensions(t *testing.T) {
	subject := dimensionProviderSubject(t)

	missing := newDimensionProviderRuntime(
		func(_ *gin.Context, _ string) (model.DataDimensionDefinition, error) {
			return model.DataDimensionDefinition{}, gorm.ErrRecordNotFound
		},
		nil,
	)
	_, err := missing.ResolveDimensionValues(nil, subject, "missing")
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionNotFound)

	unsupported := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture("warehouse", model.DataDimensionValueTypeBigint)),
		nil,
	)
	_, err = unsupported.ResolveDimensionValues(nil, subject, "warehouse")
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionUnsupported)
}

func TestDimensionProviderRuntimeRejectsTypeMismatch(t *testing.T) {
	subject := dimensionProviderSubject(t)
	runtime := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeLegalEntity, model.DataDimensionValueTypeString)),
		nil,
	)

	_, err := runtime.ResolveDimensionValues(nil, subject, datapermission.DimensionCodeLegalEntity)
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionTypeMismatch)
}

func TestDimensionProviderRuntimeWrapsDependencyFailures(t *testing.T) {
	subject := dimensionProviderSubject(t)

	dimensionFailure := newDimensionProviderRuntime(
		func(_ *gin.Context, _ string) (model.DataDimensionDefinition, error) {
			return model.DataDimensionDefinition{}, errors.New("database unavailable")
		},
		nil,
	)
	_, err := dimensionFailure.ResolveDimensionValues(nil, subject, datapermission.DimensionCodeLegalEntity)
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionProviderFailed)

	organizationFailure := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeLegalEntity, model.DataDimensionValueTypeBigint)),
		func(_ *gin.Context, _ int, _ string) (response.OrgEffectiveOrganizationScopeRes, error) {
			return response.OrgEffectiveOrganizationScopeRes{}, errors.New("organization unavailable")
		},
	)
	_, err = organizationFailure.ResolveDimensionValues(nil, subject, datapermission.DimensionCodeLegalEntity)
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionProviderFailed)
}

func TestDimensionProviderRuntimeRejectsInvalidSubjectAndProviderData(t *testing.T) {
	runtime := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeManagementOrg, model.DataDimensionValueTypeBigint)),
		func(_ *gin.Context, _ int, _ string) (response.OrgEffectiveOrganizationScopeRes, error) {
			return response.OrgEffectiveOrganizationScopeRes{}, nil
		},
	)
	_, err := runtime.ResolveDimensionValues(nil, datapermission.SubjectContext{}, datapermission.DimensionCodeManagementOrg)
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionSubjectUserNotFound)

	subject := dimensionProviderSubject(t)
	_, err = runtime.ResolveDimensionValues(nil, subject, datapermission.DimensionCodeManagementOrg)
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionProviderFailed)
}

func dimensionProviderSubject(t *testing.T) datapermission.SubjectContext {
	t.Helper()
	employeeId := 501
	subject, err := datapermission.NewSubjectContext(1001, []int{9, 3}, &employeeId, "2026-08-01")
	if err != nil {
		t.Fatalf("create subject context: %v", err)
	}
	return subject
}

func dimensionLookup(
	dimension model.DataDimensionDefinition,
) dimensionDefinitionLookup {
	return func(_ *gin.Context, code string) (model.DataDimensionDefinition, error) {
		if dimension.Code != code {
			return model.DataDimensionDefinition{}, gorm.ErrRecordNotFound
		}
		return dimension, nil
	}
}

func dimensionFixture(code, valueType string) model.DataDimensionDefinition {
	return model.DataDimensionDefinition{
		Basic:        model.Basic{Id: 9001, State: true},
		Code:         code,
		ValueType:    valueType,
		ProviderCode: organizationDimensionProviderCode,
	}
}

func resolvedDimensionScope(
	subject datapermission.SubjectContext,
	legalEntityIds []int,
	orgUnitIds []int,
) response.OrgEffectiveOrganizationScopeRes {
	return response.OrgEffectiveOrganizationScopeRes{
		EmployeeId:      subject.EmployeeId(),
		AsOfDate:        subject.AsOfDate(),
		ScopeStatus:     response.OrgEffectiveScopeResolved,
		AssignmentCount: 3,
		LegalEntityIds:  legalEntityIds,
		OrgUnitIds:      orgUnitIds,
	}
}

func assertDimensionProviderError(t *testing.T, err error, code int) {
	t.Helper()
	var adminError *response.AdminError
	if !errors.As(err, &adminError) {
		t.Fatalf("expected AdminError, got %T: %v", err, err)
	}
	if adminError.ErrorCode != code {
		t.Fatalf("unexpected error code: got %d want %d", adminError.ErrorCode, code)
	}
}
