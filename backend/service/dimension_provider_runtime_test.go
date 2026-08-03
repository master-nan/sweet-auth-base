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
				nil,
			)

			result, err := runtime.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, tt.dimensionCode))
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
		nil,
	)

	result, err := runtime.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, datapermission.DimensionCodeEmployee))
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
		nil,
	)

	result, err := runtime.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, datapermission.DimensionCodeManagementOrg))
	if err != nil {
		t.Fatalf("resolve empty dimension: %v", err)
	}
	if len(result.Values()) != 0 {
		t.Fatalf("empty scope must not expand access: %v", result.Values())
	}
}

func TestDimensionProviderRuntimeResolvesManagementOrgDescendants(t *testing.T) {
	subject := dimensionProviderSubject(t)
	structureCode := "DP-ACCEPTANCE-MGMT"
	requestedRoots := make([]int, 0, 2)
	runtime := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeManagementOrg, model.DataDimensionValueTypeBigint)),
		func(_ *gin.Context, _ int, _ string) (response.OrgEffectiveOrganizationScopeRes, error) {
			return resolvedDimensionScope(subject, []int{21}, []int{21, 11, 11}), nil
		},
		func(_ *gin.Context, code string, rootId int, asOfDate string, includeSelf bool) (response.OrgDescendantsRes, error) {
			if code != structureCode || asOfDate != subject.AsOfDate() || !includeSelf {
				t.Fatalf("unexpected descendant request: code=%s date=%s include_self=%v", code, asOfDate, includeSelf)
			}
			requestedRoots = append(requestedRoots, rootId)
			items := map[int][]response.OrgRelationItemRes{
				11: {
					{OrgUnitId: 11, Distance: 0},
					{OrgUnitId: 12, Distance: 1},
					{OrgUnitId: 13, Distance: 2},
				},
				21: {{OrgUnitId: 21, Distance: 0}},
			}[rootId]
			return response.OrgDescendantsRes{
				StructureCode: structureCode,
				OrgUnitId:     rootId,
				AsOfDate:      subject.AsOfDate(),
				Items:         items,
			}, nil
		},
	)
	request, err := newDimensionProviderRequest(
		datapermission.DimensionCodeManagementOrg,
		model.DataPolicyRelationSelfAndDescendants,
		&structureCode,
	)
	if err != nil {
		t.Fatalf("create descendant request: %v", err)
	}

	result, err := runtime.ResolveDimensionValues(nil, subject, request)
	if err != nil {
		t.Fatalf("resolve descendants: %v", err)
	}
	if got, want := result.BigintValues(), []int64{11, 12, 13, 21}; !reflect.DeepEqual(got, want) {
		t.Fatalf("descendant values: got %v want %v", got, want)
	}
	if got, want := requestedRoots, []int{21, 11}; !reflect.DeepEqual(got, want) {
		t.Fatalf("descendant roots: got %v want %v", got, want)
	}
}

func TestDimensionProviderRuntimeFailsClosedOnInvalidOrganizationTrees(t *testing.T) {
	subject := dimensionProviderSubject(t)
	structureCode := "DP-ACCEPTANCE-MGMT"
	tests := []struct {
		name       string
		dependency func(*gin.Context, string, int, string, bool) (response.OrgDescendantsRes, error)
	}{
		{
			name: "cycle",
			dependency: func(*gin.Context, string, int, string, bool) (response.OrgDescendantsRes, error) {
				return response.OrgDescendantsRes{}, myerrors.ErrOrgStructureCycle
			},
		},
		{
			name: "orphan",
			dependency: func(*gin.Context, string, int, string, bool) (response.OrgDescendantsRes, error) {
				return response.OrgDescendantsRes{}, myerrors.ErrOrgStructureNodeMissing
			},
		},
		{
			name: "invalid organization response",
			dependency: func(_ *gin.Context, _ string, rootId int, _ string, _ bool) (response.OrgDescendantsRes, error) {
				return response.OrgDescendantsRes{
					StructureCode: structureCode,
					OrgUnitId:     rootId,
					AsOfDate:      subject.AsOfDate(),
					Items:         []response.OrgRelationItemRes{{OrgUnitId: 999, Distance: 0}},
				}, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := newDimensionProviderRuntime(
				dimensionLookup(dimensionFixture(datapermission.DimensionCodeManagementOrg, model.DataDimensionValueTypeBigint)),
				func(_ *gin.Context, _ int, _ string) (response.OrgEffectiveOrganizationScopeRes, error) {
					return resolvedDimensionScope(subject, []int{21}, []int{11}), nil
				},
				tt.dependency,
			)
			request, err := newDimensionProviderRequest(
				datapermission.DimensionCodeManagementOrg,
				model.DataPolicyRelationSelfAndDescendants,
				&structureCode,
			)
			if err != nil {
				t.Fatalf("create descendant request: %v", err)
			}
			result, err := runtime.ResolveDimensionValues(nil, subject, request)
			assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionProviderFailed)
			if len(result.Values()) != 0 {
				t.Fatalf("failed tree returned values: %v", result.Values())
			}
		})
	}
}

func TestDimensionProviderRuntimeKeepsNonOrganizationRelationsExact(t *testing.T) {
	subject := dimensionProviderSubject(t)
	structureCode := "DP-ACCEPTANCE-MGMT"
	request := DimensionProviderRequest{
		DimensionCode: datapermission.DimensionCodeLegalEntity,
		Relation:      model.DataPolicyRelationSelfAndDescendants,
		StructureCode: structureCode,
	}
	runtime := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeLegalEntity, model.DataDimensionValueTypeBigint)),
		nil,
		nil,
	)

	result, err := runtime.ResolveDimensionValues(nil, subject, request)
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionUnsupported)
	if len(result.Values()) != 0 {
		t.Fatalf("unsupported relation returned values: %v", result.Values())
	}
}

func TestDimensionProviderRuntimeRejectsMissingAndUnsupportedDimensions(t *testing.T) {
	subject := dimensionProviderSubject(t)

	missing := newDimensionProviderRuntime(
		func(_ *gin.Context, _ string) (model.DataDimensionDefinition, error) {
			return model.DataDimensionDefinition{}, gorm.ErrRecordNotFound
		},
		nil,
		nil,
	)
	_, err := missing.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, "missing"))
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionNotFound)

	unsupported := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture("warehouse", model.DataDimensionValueTypeBigint)),
		nil,
		nil,
	)
	_, err = unsupported.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, "warehouse"))
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionUnsupported)
}

func TestDimensionProviderRuntimeRejectsTypeMismatch(t *testing.T) {
	subject := dimensionProviderSubject(t)
	runtime := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeLegalEntity, model.DataDimensionValueTypeString)),
		nil,
		nil,
	)

	_, err := runtime.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, datapermission.DimensionCodeLegalEntity))
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionTypeMismatch)
}

func TestDimensionProviderRuntimeWrapsDependencyFailures(t *testing.T) {
	subject := dimensionProviderSubject(t)

	dimensionFailure := newDimensionProviderRuntime(
		func(_ *gin.Context, _ string) (model.DataDimensionDefinition, error) {
			return model.DataDimensionDefinition{}, errors.New("database unavailable")
		},
		nil,
		nil,
	)
	_, err := dimensionFailure.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, datapermission.DimensionCodeLegalEntity))
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionProviderFailed)

	organizationFailure := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeLegalEntity, model.DataDimensionValueTypeBigint)),
		func(_ *gin.Context, _ int, _ string) (response.OrgEffectiveOrganizationScopeRes, error) {
			return response.OrgEffectiveOrganizationScopeRes{}, errors.New("organization unavailable")
		},
		nil,
	)
	_, err = organizationFailure.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, datapermission.DimensionCodeLegalEntity))
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionDimensionProviderFailed)
}

func TestDimensionProviderRuntimeRejectsInvalidSubjectAndProviderData(t *testing.T) {
	runtime := newDimensionProviderRuntime(
		dimensionLookup(dimensionFixture(datapermission.DimensionCodeManagementOrg, model.DataDimensionValueTypeBigint)),
		func(_ *gin.Context, _ int, _ string) (response.OrgEffectiveOrganizationScopeRes, error) {
			return response.OrgEffectiveOrganizationScopeRes{}, nil
		},
		nil,
	)
	_, err := runtime.ResolveDimensionValues(nil, datapermission.SubjectContext{}, exactDimensionProviderRequest(t, datapermission.DimensionCodeManagementOrg))
	assertDimensionProviderError(t, err, myerrors.ErrorCodeDataPermissionSubjectUserNotFound)

	subject := dimensionProviderSubject(t)
	_, err = runtime.ResolveDimensionValues(nil, subject, exactDimensionProviderRequest(t, datapermission.DimensionCodeManagementOrg))
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

func exactDimensionProviderRequest(t *testing.T, dimensionCode string) DimensionProviderRequest {
	t.Helper()
	request, err := newDimensionProviderRequest(dimensionCode, model.DataPolicyRelationExact, nil)
	if err != nil {
		t.Fatalf("create exact Dimension Provider request: %v", err)
	}
	return request
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
