package service

import (
	"errors"
	"net/http"
	"strings"

	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const organizationDimensionProviderCode = "organization"

type dimensionDefinitionLookup func(*gin.Context, string) (model.DataDimensionDefinition, error)
type dimensionOrganizationScopeLookup func(
	*gin.Context,
	int,
	string,
) (response.OrgEffectiveOrganizationScopeRes, error)

// DimensionProvider resolves trusted subject facts for one configured
// dimension. It does not read grants, policies, resources, or business data.
type DimensionProvider interface {
	ResolveDimensionValues(
		*gin.Context,
		datapermission.SubjectContext,
		string,
	) (datapermission.DimensionValues, error)
}

type DimensionProviderRuntime struct {
	findDimension       dimensionDefinitionLookup
	resolveOrganization dimensionOrganizationScopeLookup
}

var _ DimensionProvider = (*DimensionProviderRuntime)(nil)

func NewDimensionProviderRuntime(
	dimensionRepo repository.DataDimensionDefinitionRepository,
	organizationProvider OrgPermissionProvider,
) *DimensionProviderRuntime {
	return newDimensionProviderRuntime(
		dimensionRepo.FindByCode,
		organizationProvider.GetEmployeeEffectiveOrganizationScope,
	)
}

func newDimensionProviderRuntime(
	findDimension dimensionDefinitionLookup,
	resolveOrganization dimensionOrganizationScopeLookup,
) *DimensionProviderRuntime {
	return &DimensionProviderRuntime{
		findDimension:       findDimension,
		resolveOrganization: resolveOrganization,
	}
}

func (runtime *DimensionProviderRuntime) ResolveDimensionValues(
	ctx *gin.Context,
	subject datapermission.SubjectContext,
	dimensionCode string,
) (datapermission.DimensionValues, error) {
	if err := subject.Validate(); err != nil {
		return datapermission.DimensionValues{}, err
	}
	if runtime == nil || runtime.findDimension == nil {
		return datapermission.DimensionValues{}, myerrors.ErrDataPermissionDimensionProviderFailed
	}

	code := strings.TrimSpace(dimensionCode)
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
		values = []int{subject.EmployeeId()}
	case datapermission.DimensionCodeLegalEntity, datapermission.DimensionCodeManagementOrg:
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
	return myerrors.WrapBusinessError(
		cause,
		http.StatusServiceUnavailable,
		myerrors.ErrorCodeDataPermissionDimensionProviderFailed,
		"数据权限维度Provider调用失败",
	)
}
