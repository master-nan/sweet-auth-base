package errors

import "net/http"

const (
	ErrorCodeDataPermissionSubjectUserNotFound      = 120059
	ErrorCodeDataPermissionRoleContextMissing       = 120060
	ErrorCodeDataPermissionEmployeeUnbound          = 120061
	ErrorCodeDataPermissionSubjectContextInvalid    = 120062
	ErrorCodeDataScopeDecisionInvalid               = 120063
	ErrorCodeDataScopeResultConditionMismatch       = 120064
	ErrorCodeDataScopeFilterConditionMissing        = 120065
	ErrorCodeDataScopeConditionGroupEmpty           = 120066
	ErrorCodeDataScopeOwnershipCodeInvalid          = 120067
	ErrorCodeDataScopeDimensionInvalid              = 120068
	ErrorCodeDataScopeOperatorInvalid               = 120069
	ErrorCodeDataScopeValueTypeInvalid              = 120070
	ErrorCodeDataScopeValueTypeMismatch             = 120071
	ErrorCodeDataScopeValueCountExceeded            = 120072
	ErrorCodeDataScopeMergeUnsupported              = 120073
	ErrorCodeDataScopeResultIdentityInvalid         = 120074
	ErrorCodeDataScopeComplexityExceeded            = 120075
	ErrorCodeDataPermissionDimensionNotFound        = 120076
	ErrorCodeDataPermissionDimensionUnsupported     = 120077
	ErrorCodeDataPermissionDimensionTypeMismatch    = 120078
	ErrorCodeDataPermissionDimensionProviderFailed  = 120079
	ErrorCodeDataPermissionResolverResourceMissing  = 120080
	ErrorCodeDataPermissionResolverOperationMissing = 120081
	ErrorCodeDataPermissionResolverGrantMissing     = 120082
	ErrorCodeDataPermissionResolverPolicyInvalid    = 120083
	ErrorCodeDataPermissionResolverOwnershipMissing = 120084
	ErrorCodeDataPermissionResolverDimensionFailed  = 120085
	ErrorCodeDataPermissionResolverConfigConflict   = 120086
	ErrorCodeDataPermissionResolverFailed           = 120087
)

var (
	ErrDataPermissionSubjectUserNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataPermissionSubjectUserNotFound,
		"数据权限主体用户不存在",
	)
	ErrDataPermissionRoleContextMissing = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataPermissionRoleContextMissing,
		"数据权限角色上下文缺失",
	)
	ErrDataPermissionEmployeeUnbound = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPermissionEmployeeUnbound,
		"当前账号未绑定企业人员",
	)
	ErrDataPermissionSubjectContextInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPermissionSubjectContextInvalid,
		"数据权限主体上下文不合法",
	)
	ErrDataScopeDecisionInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeDecisionInvalid,
		"数据权限结果决策类型不合法",
	)
	ErrDataScopeResultConditionMismatch = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeResultConditionMismatch,
		"数据权限结果与过滤条件不一致",
	)
	ErrDataScopeFilterConditionMissing = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeFilterConditionMissing,
		"数据权限过滤条件缺失",
	)
	ErrDataScopeConditionGroupEmpty = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeConditionGroupEmpty,
		"数据权限条件组不能为空",
	)
	ErrDataScopeOwnershipCodeInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeOwnershipCodeInvalid,
		"数据权限归属编码不合法",
	)
	ErrDataScopeDimensionInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeDimensionInvalid,
		"数据权限维度不合法",
	)
	ErrDataScopeOperatorInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeOperatorInvalid,
		"数据权限过滤操作符不合法",
	)
	ErrDataScopeValueTypeInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeValueTypeInvalid,
		"数据权限值类型不合法",
	)
	ErrDataScopeValueTypeMismatch = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeValueTypeMismatch,
		"数据权限过滤值类型不一致",
	)
	ErrDataScopeValueCountExceeded = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataScopeValueCountExceeded,
		"数据权限过滤值数量超过限制",
	)
	ErrDataScopeMergeUnsupported = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataScopeMergeUnsupported,
		"当前数据权限结果不支持合并",
	)
	ErrDataScopeResultIdentityInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataScopeResultIdentityInvalid,
		"数据权限结果资源或操作不合法",
	)
	ErrDataScopeComplexityExceeded = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataScopeComplexityExceeded,
		"数据权限结果复杂度超过限制",
	)
	ErrDataPermissionDimensionNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataPermissionDimensionNotFound,
		"数据权限维度不存在",
	)
	ErrDataPermissionDimensionUnsupported = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataPermissionDimensionUnsupported,
		"数据权限维度暂不支持运行时解析",
	)
	ErrDataPermissionDimensionTypeMismatch = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataPermissionDimensionTypeMismatch,
		"数据权限维度值类型不匹配",
	)
	ErrDataPermissionDimensionProviderFailed = NewBusinessError(
		http.StatusServiceUnavailable,
		ErrorCodeDataPermissionDimensionProviderFailed,
		"数据权限维度Provider调用失败",
	)
	ErrDataPermissionResolverResourceMissing = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataPermissionResolverResourceMissing,
		"数据权限解析资源不存在",
	)
	ErrDataPermissionResolverOperationMissing = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataPermissionResolverOperationMissing,
		"数据权限解析操作不存在",
	)
	ErrDataPermissionResolverGrantMissing = NewBusinessError(
		http.StatusForbidden,
		ErrorCodeDataPermissionResolverGrantMissing,
		"数据权限授权不存在",
	)
	ErrDataPermissionResolverPolicyInvalid = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataPermissionResolverPolicyInvalid,
		"数据权限策略无效",
	)
	ErrDataPermissionResolverOwnershipMissing = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataPermissionResolverOwnershipMissing,
		"数据权限归属定义缺失",
	)
	ErrDataPermissionResolverDimensionFailed = NewBusinessError(
		http.StatusServiceUnavailable,
		ErrorCodeDataPermissionResolverDimensionFailed,
		"数据权限维度解析失败",
	)
	ErrDataPermissionResolverConfigConflict = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPermissionResolverConfigConflict,
		"数据权限解析配置冲突",
	)
	ErrDataPermissionResolverFailed = NewBusinessError(
		http.StatusInternalServerError,
		ErrorCodeDataPermissionResolverFailed,
		"数据权限解析失败",
	)
)
