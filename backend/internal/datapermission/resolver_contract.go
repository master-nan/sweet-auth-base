package datapermission

import (
	"context"
	"errors"
	"strings"

	myerrors "backend/internal/errors"
)

// ResolverInput 是单次数据范围解析的不可变身份。
// Resolver 在服务端加载全部 Resource、Grant、Policy、Rule、Ownership 和 Dimension 事实，
// 调用方不能在此注入这些配置对象。
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

// Resolver 将可信身份、配置和 Dimension Provider 事实组合为 DataScopeResult 语义。
// 实现只能使用数据权限配置 Reader 和 Dimension Provider 类型的事实端口。
// SQL、ORM Scope、Adapter、组织表和业务 Repository 均不属于此接口。
type Resolver interface {
	Resolve(context.Context, ResolverInput) (DataScopeResult, error)
}

// ResolverFunc 将函数适配为 Resolver，同时执行已冻结的输入、输出身份和失败关闭契约。
// 它不包含解析算法。
type ResolverFunc func(context.Context, ResolverInput) (DataScopeResult, error)

var _ Resolver = ResolverFunc(nil)

func (resolve ResolverFunc) Resolve(
	ctx context.Context,
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
	var applicationError *myerrors.ApplicationError
	if errors.As(err, &applicationError) {
		return err
	}
	return wrapResolverFailure(err)
}

func wrapResolverFailure(cause error) error {
	return myerrors.WrapApplicationError(
		cause,
		myerrors.KindInternal,
		myerrors.CategoryBusiness,
		myerrors.ErrorCodeDataPermissionResolverFailed,
		"数据权限解析失败",
	)
}
