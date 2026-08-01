package datapermission

import (
	"errors"
	"net/http"
	"strings"

	"backend/dto/response"
	myerrors "backend/internal/errors"

	"github.com/gin-gonic/gin"
)

// ResolverInput is the immutable identity of one data-scope resolution. The
// Resolver loads all Resource, Grant, Policy, Rule, Ownership, and Dimension
// facts server-side; callers cannot inject those configuration objects here.
type ResolverInput struct {
	subjectContext SubjectContext
	resourceCode   string
	operation      string
}

func NewResolverInput(
	subjectContext SubjectContext,
	resourceCode string,
	operation string,
) (ResolverInput, error) {
	input := ResolverInput{
		subjectContext: subjectContext,
		resourceCode:   strings.TrimSpace(resourceCode),
		operation:      strings.ToLower(strings.TrimSpace(operation)),
	}
	if err := input.Validate(); err != nil {
		return ResolverInput{}, err
	}
	return input, nil
}

func (input ResolverInput) Validate() error {
	if err := input.subjectContext.Validate(); err != nil {
		return err
	}
	if !dataScopeResourceCodePattern.MatchString(input.resourceCode) {
		return myerrors.ErrDataPermissionResolverResourceMissing
	}
	if _, exists := dataScopeOperations[input.operation]; !exists {
		return myerrors.ErrDataPermissionResolverOperationMissing
	}
	return nil
}

func (input ResolverInput) SubjectContext() SubjectContext {
	return input.subjectContext
}

func (input ResolverInput) ResourceCode() string {
	return input.resourceCode
}

func (input ResolverInput) Operation() string {
	return input.operation
}

// Resolver combines trusted identity, configuration, and Dimension Provider
// facts into DataScopeResult semantics. Implementations may use only data-
// permission configuration readers and DimensionProvider-style fact ports.
// SQL, ORM scopes, adapters, organization tables, and business repositories
// remain outside this interface.
type Resolver interface {
	Resolve(*gin.Context, ResolverInput) (DataScopeResult, error)
}

// ResolverFunc adapts a function to Resolver while enforcing the frozen input,
// output identity, and fail-closed contract. It contains no parsing algorithm.
type ResolverFunc func(*gin.Context, ResolverInput) (DataScopeResult, error)

var _ Resolver = ResolverFunc(nil)

func (resolve ResolverFunc) Resolve(
	ctx *gin.Context,
	input ResolverInput,
) (DataScopeResult, error) {
	if err := input.Validate(); err != nil {
		return DataScopeResult{}, err
	}
	if resolve == nil {
		return DataScopeResult{}, myerrors.ErrDataPermissionResolverFailed
	}

	result, err := resolve(ctx, input)
	if err != nil {
		return DataScopeResult{}, normalizeResolverError(err)
	}
	if err = result.Validate(); err != nil {
		return DataScopeResult{}, wrapResolverFailure(err)
	}
	if result.ResourceCode() != input.resourceCode || result.Operation() != input.operation {
		return DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}
	return result, nil
}

func normalizeResolverError(err error) error {
	var adminError *response.AdminError
	if errors.As(err, &adminError) {
		return err
	}
	return wrapResolverFailure(err)
}

func wrapResolverFailure(cause error) error {
	return myerrors.WrapBusinessError(
		cause,
		http.StatusInternalServerError,
		myerrors.ErrorCodeDataPermissionResolverFailed,
		"数据权限解析失败",
	)
}
