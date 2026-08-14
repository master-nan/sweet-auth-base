package service

import (
	"context"
	"errors"
	"strings"

	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"

	"gorm.io/gorm"
)

const organizationDimensionProviderCode = "organization"

type dimensionDefinitionLookup func(context.Context, string) (model.DataDimensionDefinition, error)
type dimensionOrganizationScopeLookup func(
	context.Context,
	int,
	string,
) (response.OrgEffectiveOrganizationScopeRes, error)
type dimensionOrganizationDescendantsLookup func(
	context.Context,
	string,
	int,
	string,
	bool,
) (response.OrgDescendantsRes, error)

// DimensionProvider 解析指定 Dimension 下可信的主体事实。
// 它不读取 Grant、Policy、Resource 或业务数据。
type DimensionProvider interface {
	ResolveDimensionValues(
		context.Context,
		datapermission.SubjectContext,
		DimensionProviderRequest,
	) (datapermission.DimensionValues, error)
}

type DimensionProviderRuntime struct {
	findDimension         dimensionDefinitionLookup
	resolveOrganization   dimensionOrganizationScopeLookup
	resolveOrgDescendants dimensionOrganizationDescendantsLookup
}

var _ DimensionProvider = (*DimensionProviderRuntime)(nil)

func NewDimensionProviderRuntime(
	dimensionRepo repository.DataDimensionDefinitionRepository,
	organizationProvider OrgPermissionProvider,
) *DimensionProviderRuntime {
	return newDimensionProviderRuntime(
		func(ctx context.Context, code string) (model.DataDimensionDefinition, error) {
			return dimensionRepo.WithContext(ctx).FindByField("code", code)
		},
		func(ctx context.Context, employeeId int, asOfDate string) (response.OrgEffectiveOrganizationScopeRes, error) {
			if ctx == nil {
				ctx = context.Background()
			}
			return organizationProvider.GetEmployeeEffectiveOrganizationScope(ctx, employeeId, asOfDate)
		},
		func(ctx context.Context, structureCode string, orgUnitId int, asOfDate string, includeSelf bool) (response.OrgDescendantsRes, error) {
			if ctx == nil {
				ctx = context.Background()
			}
			return organizationProvider.GetOrgDescendants(ctx, structureCode, orgUnitId, asOfDate, includeSelf)
		},
	)
}

func newDimensionProviderRuntime(
	findDimension dimensionDefinitionLookup,
	resolveOrganization dimensionOrganizationScopeLookup,
	resolveOrgDescendants dimensionOrganizationDescendantsLookup,
) *DimensionProviderRuntime {
	return &DimensionProviderRuntime{
		findDimension:         findDimension,
		resolveOrganization:   resolveOrganization,
		resolveOrgDescendants: resolveOrgDescendants,
	}
}

func (runtime *DimensionProviderRuntime) ResolveDimensionValues(
	ctx context.Context,
	subject datapermission.SubjectContext,
	request DimensionProviderRequest,
) (datapermission.DimensionValues, error) {
	if err := subject.Validate(); err != nil {
		return datapermission.DimensionValues{}, err
	}
	if err := request.validate(); err != nil {
		return datapermission.DimensionValues{}, err
	}
	if runtime == nil || runtime.findDimension == nil {
		return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionProviderFailed
	}

	code := strings.TrimSpace(request.DimensionCode)
	dimension, err := runtime.findDimension(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionNotFound
		}
		return datapermission.DimensionValues{}, wrapDimensionProviderFailure(err)
	}
	if dimension.Id <= 0 || dimension.Code != code {
		return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionProviderFailed
	}
	if !dimension.State || dimension.ProviderCode != organizationDimensionProviderCode {
		return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionUnsupported
	}
	if dimension.ValueType != model.DataDimensionValueTypeBigint {
		return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionTypeMismatch
	}

	var values []int
	switch code {
	case datapermission.DimensionCodeEmployee:
		if request.Relation != model.DataPolicyRelationExact {
			return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionUnsupported
		}
		values = []int{subject.EmployeeId()}
	case datapermission.DimensionCodeLegalEntity, datapermission.DimensionCodeManagementOrg:
		if code == datapermission.DimensionCodeLegalEntity &&
			request.Relation != model.DataPolicyRelationExact {
			return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionUnsupported
		}
		if runtime.resolveOrganization == nil {
			return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionProviderFailed
		}
		scope, scopeErr := runtime.resolveOrganization(
			ctx,
			subject.EmployeeId(),
			subject.AsOfDate(),
		)
		if scopeErr != nil {
			return datapermission.DimensionValues{}, wrapDimensionProviderFailure(scopeErr)
		}
		if err = validateDimensionOrganizationScope(subject, scope); err != nil {
			return datapermission.DimensionValues{}, err
		}
		if code == datapermission.DimensionCodeLegalEntity {
			values = scope.LegalEntityIds
		} else if request.Relation == model.DataPolicyRelationSelfAndDescendants {
			values, err = runtime.resolveManagementOrgDescendants(ctx, subject, request, scope.OrgUnitIds)
			if err != nil {
				return datapermission.DimensionValues{}, err
			}
		} else {
			values = scope.OrgUnitIds
		}
	default:
		return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionUnsupported
	}

	return datapermission.NewDimensionValues(
		code,
		datapermission.DataScopeValueTypeBigint,
		intValuesAsAny(values),
	)
}

func (runtime *DimensionProviderRuntime) resolveManagementOrgDescendants(
	ctx context.Context,
	subject datapermission.SubjectContext,
	request DimensionProviderRequest,
	directOrgUnitIds []int,
) ([]int, error) {
	if len(directOrgUnitIds) == 0 {
		return []int{}, nil
	}
	if runtime.resolveOrgDescendants == nil {
		return nil, myerrors.ErrDataPermissionDimensionProviderFailed
	}
	resolved := make(map[int]struct{})
	seenRoots := make(map[int]struct{})
	for _, orgUnitId := range directOrgUnitIds {
		if orgUnitId <= 0 {
			return nil, myerrors.ErrDataPermissionDimensionProviderFailed
		}
		if _, exists := seenRoots[orgUnitId]; exists {
			continue
		}
		seenRoots[orgUnitId] = struct{}{}
		descendants, err := runtime.resolveOrgDescendants(
			ctx,
			request.StructureCode,
			orgUnitId,
			subject.AsOfDate(),
			true,
		)
		if err != nil {
			return nil, wrapDimensionProviderFailure(err)
		}
		if err = validateDimensionOrgDescendants(subject, request, orgUnitId, descendants); err != nil {
			return nil, err
		}
		for _, item := range descendants.Items {
			resolved[item.OrgUnitId] = struct{}{}
		}
	}
	return sortedOrganizationIds(resolved), nil
}

func validateDimensionOrgDescendants(
	subject datapermission.SubjectContext,
	request DimensionProviderRequest,
	rootOrgUnitId int,
	descendants response.OrgDescendantsRes,
) error {
	if descendants.StructureCode != request.StructureCode ||
		descendants.OrgUnitId != rootOrgUnitId ||
		descendants.AsOfDate != subject.AsOfDate() ||
		len(descendants.Items) == 0 {
		return myerrors.ErrDataPermissionDimensionProviderFailed
	}
	seen := make(map[int]struct{}, len(descendants.Items))
	foundSelf := false
	for _, item := range descendants.Items {
		if item.OrgUnitId <= 0 || item.Distance < 0 {
			return myerrors.ErrDataPermissionDimensionProviderFailed
		}
		if _, exists := seen[item.OrgUnitId]; exists {
			return myerrors.ErrDataPermissionDimensionProviderFailed
		}
		seen[item.OrgUnitId] = struct{}{}
		if item.OrgUnitId == rootOrgUnitId {
			if item.Distance != 0 {
				return myerrors.ErrDataPermissionDimensionProviderFailed
			}
			foundSelf = true
		} else if item.Distance == 0 {
			return myerrors.ErrDataPermissionDimensionProviderFailed
		}
	}
	if !foundSelf {
		return myerrors.ErrDataPermissionDimensionProviderFailed
	}
	return nil
}

func validateDimensionOrganizationScope(
	subject datapermission.SubjectContext,
	scope response.OrgEffectiveOrganizationScopeRes,
) error {
	if scope.EmployeeId != subject.EmployeeId() || scope.AsOfDate != subject.AsOfDate() {
		return myerrors.ErrDataPermissionDimensionProviderFailed
	}
	switch scope.ScopeStatus {
	case response.OrgEffectiveScopeEmpty:
		if scope.AssignmentCount != 0 || len(scope.LegalEntityIds) != 0 || len(scope.OrgUnitIds) != 0 {
			return myerrors.ErrDataPermissionDimensionProviderFailed
		}
	case response.OrgEffectiveScopeResolved:
		if scope.AssignmentCount <= 0 {
			return myerrors.ErrDataPermissionDimensionProviderFailed
		}
	default:
		return myerrors.ErrDataPermissionDimensionProviderFailed
	}
	return nil
}

func intValuesAsAny(values []int) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

func wrapDimensionProviderFailure(cause error) error {
	return myerrors.WrapApplicationError(
		cause,
		myerrors.KindUnavailable,
		myerrors.CategoryBusiness,
		myerrors.ErrorCodeDataPermissionDimensionProviderFailed,
		"数据权限维度Provider调用失败",
	)
}
