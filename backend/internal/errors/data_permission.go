package errors

// Configuration application errors.
const (
	ErrorCodeDataResourceNotFound                 = 120001
	ErrorCodeDataResourceCodeDuplicate            = 120002
	ErrorCodeDataResourceCodeInvalid              = 120003
	ErrorCodeDataResourceReferenced               = 120004
	ErrorCodeDataResourceStateInvalid             = 120005
	ErrorCodeDataResourceOperationDuplicate       = 120006
	ErrorCodeDataResourceOperationInvalid         = 120007
	ErrorCodeDataResourceFieldImmutable           = 120008
	ErrorCodeDataResourcePermissionEnableDenied   = 120009
	ErrorCodeDataResourceOperationReferenced      = 120010
	ErrorCodeDataResourceNameRequired             = 120011
	ErrorCodeDataResourceTypeInvalid              = 120012
	ErrorCodeDataResourceTargetInvalid            = 120013
	ErrorCodeDataResourceOperationNotFound        = 120014
	ErrorCodeDataDimensionNotFound                = 120015
	ErrorCodeDataOwnershipNotFound                = 120016
	ErrorCodeDataOwnershipDuplicate               = 120017
	ErrorCodeDataOwnershipCodeInvalid             = 120018
	ErrorCodeDataOwnershipBindingInvalid          = 120019
	ErrorCodeDataOwnershipRegisteredFieldInvalid  = 120021
	ErrorCodeDataOwnershipValueTypeMismatch       = 120022
	ErrorCodeDataOwnershipReferenced              = 120023
	ErrorCodeDataOwnershipFieldImmutable          = 120024
	ErrorCodeDataOwnershipMetadataFieldNotFound   = 120025
	ErrorCodeDataOwnershipMetadataFieldMismatch   = 120026
	ErrorCodeDataOwnershipMetadataFieldForbidden  = 120027
	ErrorCodeDataOwnershipRegisteredFieldMissing  = 120028
	ErrorCodeDataOwnershipRegisteredResource      = 120029
	ErrorCodeDataOwnershipRegisteredDimension     = 120030
	ErrorCodeDataOwnershipRegisteredOperation     = 120031
	ErrorCodeDataOwnershipMetadataDimension       = 120032
	ErrorCodeDataPolicyNotFound                   = 120033
	ErrorCodeDataPolicyCodeDuplicate              = 120034
	ErrorCodeDataPolicyCodeInvalid                = 120035
	ErrorCodeDataPolicyNameRequired               = 120036
	ErrorCodeDataPolicyFieldImmutable             = 120037
	ErrorCodeDataPolicyStateInvalid               = 120038
	ErrorCodeDataPolicyRuleNotFound               = 120039
	ErrorCodeDataPolicyRuleDuplicate              = 120040
	ErrorCodeDataPolicyRuleOwnershipNotFound      = 120041
	ErrorCodeDataPolicyRuleDimensionMismatch      = 120042
	ErrorCodeDataPolicyRuleScopeSourceInvalid     = 120043
	ErrorCodeDataPolicyRuleRelationInvalid        = 120044
	ErrorCodeDataPolicyRuleOperatorInvalid        = 120045
	ErrorCodeDataPolicyRuleSpecifiedValuesInvalid = 120046
	ErrorCodeDataPolicyRuleCountInvalid           = 120047
	ErrorCodeDataGrantNotFound                    = 120048
	ErrorCodeDataGrantSubjectTypeInvalid          = 120049
	ErrorCodeDataGrantSubjectNotFound             = 120050
	ErrorCodeDataGrantPolicyInvalid               = 120051
	ErrorCodeDataGrantPolicyRuleInvalid           = 120052
	ErrorCodeDataGrantOwnershipMismatch           = 120053
	ErrorCodeDataGrantDuplicate                   = 120054
	ErrorCodeDataGrantExists                      = 120055
	ErrorCodeDataGrantValidityInvalid             = 120056
	ErrorCodeDataGrantCountInvalid                = 120057
	ErrorCodeDataPermissionPreflightFailed        = 120058
)

var (
	ErrDataResourceNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataResourceNotFound,
		"数据资源不存在",
	)
	ErrDataResourceCodeDuplicate = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataResourceCodeDuplicate,
		"数据资源编码已存在",
	)
	ErrDataResourceCodeInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataResourceCodeInvalid,
		"数据资源编码格式不合法",
	)
	ErrDataResourceReferenced = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataResourceReferenced,
		"数据资源已被配置引用",
	)
	ErrDataResourceStateInvalid = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataResourceStateInvalid,
		"数据资源当前状态不允许执行该操作",
	)
	ErrDataResourceOperationDuplicate = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataResourceOperationDuplicate,
		"数据资源操作已存在",
	)
	ErrDataResourceOperationInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataResourceOperationInvalid,
		"数据资源操作取值不合法",
	)
	ErrDataResourceFieldImmutable = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataResourceFieldImmutable,
		"数据资源身份字段不可修改",
	)
	ErrDataResourcePermissionEnableDenied = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataResourcePermissionEnableDenied,
		"当前阶段不允许启用数据权限",
	)
	ErrDataResourceOperationReferenced = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataResourceOperationReferenced,
		"数据资源操作已被授权引用",
	)
	ErrDataResourceNameRequired = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataResourceNameRequired,
		"数据资源名称不能为空",
	)
	ErrDataResourceTypeInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataResourceTypeInvalid,
		"数据资源类型不合法",
	)
	ErrDataResourceTargetInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataResourceTargetInvalid,
		"数据资源目标配置不合法",
	)
	ErrDataResourceOperationNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataResourceOperationNotFound,
		"数据资源操作不存在",
	)
	ErrDataDimensionNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataDimensionNotFound,
		"数据权限维度不存在或已停用",
	)
	ErrDataOwnershipNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataOwnershipNotFound,
		"数据归属定义不存在",
	)
	ErrDataOwnershipDuplicate = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataOwnershipDuplicate,
		"数据归属编码在当前资源中已存在",
	)
	ErrDataOwnershipCodeInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipCodeInvalid,
		"数据归属编码格式不合法",
	)
	ErrDataOwnershipBindingInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipBindingInvalid,
		"数据归属绑定配置不合法",
	)
	ErrDataOwnershipRegisteredFieldInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipRegisteredFieldInvalid,
		"数据归属注册字段不合法",
	)
	ErrDataOwnershipValueTypeMismatch = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipValueTypeMismatch,
		"数据归属值类型与维度不兼容",
	)
	ErrDataOwnershipReferenced = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataOwnershipReferenced,
		"数据归属定义已被策略引用",
	)
	ErrDataOwnershipFieldImmutable = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataOwnershipFieldImmutable,
		"数据归属身份字段不可修改",
	)
	ErrDataOwnershipMetadataFieldNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataOwnershipMetadataFieldNotFound,
		"数据归属元数据字段不存在或已删除",
	)
	ErrDataOwnershipMetadataFieldMismatch = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipMetadataFieldMismatch,
		"数据归属元数据字段不属于当前资源",
	)
	ErrDataOwnershipMetadataFieldForbidden = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipMetadataFieldForbidden,
		"该元数据字段不能作为数据归属字段",
	)
	ErrDataOwnershipRegisteredFieldMissing = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipRegisteredFieldMissing,
		"数据归属注册项不存在",
	)
	ErrDataOwnershipRegisteredResource = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipRegisteredResource,
		"数据归属注册项与当前资源不匹配",
	)
	ErrDataOwnershipRegisteredDimension = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipRegisteredDimension,
		"数据归属注册项不支持当前维度",
	)
	ErrDataOwnershipRegisteredOperation = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipRegisteredOperation,
		"数据归属注册项不支持当前操作",
	)
	ErrDataOwnershipMetadataDimension = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataOwnershipMetadataDimension,
		"元数据字段与数据权限维度不匹配",
	)
	ErrDataPolicyNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataPolicyNotFound,
		"数据权限策略不存在",
	)
	ErrDataPolicyCodeDuplicate = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPolicyCodeDuplicate,
		"数据权限策略编码已存在",
	)
	ErrDataPolicyCodeInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyCodeInvalid,
		"数据权限策略编码格式不合法",
	)
	ErrDataPolicyNameRequired = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyNameRequired,
		"数据权限策略名称不能为空",
	)
	ErrDataPolicyFieldImmutable = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPolicyFieldImmutable,
		"数据权限策略身份字段不可修改",
	)
	ErrDataPolicyStateInvalid = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPolicyStateInvalid,
		"数据权限策略当前状态不允许执行该操作",
	)
	ErrDataPolicyRuleNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataPolicyRuleNotFound,
		"数据权限策略规则不存在",
	)
	ErrDataPolicyRuleDuplicate = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPolicyRuleDuplicate,
		"数据权限策略规则序号已存在",
	)
	ErrDataPolicyRuleOwnershipNotFound = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyRuleOwnershipNotFound,
		"数据权限策略规则未匹配到有效归属定义",
	)
	ErrDataPolicyRuleDimensionMismatch = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyRuleDimensionMismatch,
		"数据权限策略规则的归属编码与维度不匹配",
	)
	ErrDataPolicyRuleScopeSourceInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyRuleScopeSourceInvalid,
		"数据权限策略规则的范围来源与维度不兼容",
	)
	ErrDataPolicyRuleRelationInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyRuleRelationInvalid,
		"数据权限策略规则的关系类型不合法",
	)
	ErrDataPolicyRuleOperatorInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyRuleOperatorInvalid,
		"数据权限策略规则的操作符不合法",
	)
	ErrDataPolicyRuleSpecifiedValuesInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyRuleSpecifiedValuesInvalid,
		"数据权限策略规则的指定值不合法",
	)
	ErrDataPolicyRuleCountInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPolicyRuleCountInvalid,
		"数据权限策略规则数量超过限制",
	)
	ErrDataGrantNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataGrantNotFound,
		"数据权限授权不存在",
	)
	ErrDataGrantSubjectTypeInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataGrantSubjectTypeInvalid,
		"数据权限授权主体类型不合法",
	)
	ErrDataGrantSubjectNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataGrantSubjectNotFound,
		"数据权限授权主体不存在或已停用",
	)
	ErrDataGrantPolicyInvalid = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataGrantPolicyInvalid,
		"数据权限授权策略不存在、已停用或类型不合法",
	)
	ErrDataGrantPolicyRuleInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataGrantPolicyRuleInvalid,
		"数据权限授权策略规则不完整或不合法",
	)
	ErrDataGrantOwnershipMismatch = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataGrantOwnershipMismatch,
		"数据权限策略与目标资源归属定义不匹配",
	)
	ErrDataGrantDuplicate = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataGrantDuplicate,
		"相同数据权限授权已启用",
	)
	ErrDataGrantExists = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataGrantExists,
		"相同数据权限授权已存在，请恢复原授权",
	)
	ErrDataGrantValidityInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataGrantValidityInvalid,
		"数据权限授权有效期不合法",
	)
	ErrDataGrantCountInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataGrantCountInvalid,
		"批量数据权限授权数量超过限制",
	)
	ErrDataPermissionPreflightFailed = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPermissionPreflightFailed,
		"数据权限配置预检未通过",
	)
)

// Adapter application errors.
const (
	ErrorCodeDataPermissionAdapterInputInvalid      = 120088
	ErrorCodeDataPermissionAdapterTypeUnsupported   = 120089
	ErrorCodeDataPermissionAdapterOwnershipMissing  = 120090
	ErrorCodeDataPermissionAdapterOwnershipMismatch = 120091
	ErrorCodeDataPermissionAdapterExecutionInvalid  = 120092
	ErrorCodeDataPermissionAdapterFailed            = 120093
	ErrorCodeMetadataAdapterResourceTableMissing    = 120094
	ErrorCodeMetadataAdapterTableNotFound           = 120095
	ErrorCodeMetadataAdapterFieldNotFound           = 120096
	ErrorCodeMetadataAdapterFieldResourceMismatch   = 120097
	ErrorCodeMetadataAdapterFieldInactive           = 120098
	ErrorCodeMetadataAdapterFieldTypeUnsupported    = 120099
	ErrorCodeMetadataAdapterFieldTypeDrift          = 120100
	ErrorCodeMetadataAdapterFieldNotFilterable      = 120101
	ErrorCodeMetadataAdapterOperatorUnsupported     = 120102
	ErrorCodeMetadataAdapterValueTypeMismatch       = 120103
	ErrorCodeMetadataAdapterComplexityExceeded      = 120104
	ErrorCodeMetadataAdapterFailed                  = 120105
	ErrorCodeRegisteredAdapterUnregistered          = 120106
	ErrorCodeRegisteredAdapterFieldNotFound         = 120107
	ErrorCodeRegisteredAdapterRegistrationDuplicate = 120108
	ErrorCodeRegisteredAdapterRegistrationConflict  = 120109
	ErrorCodeRegisteredAdapterResourceMismatch      = 120110
	ErrorCodeRegisteredAdapterDimensionUnsupported  = 120111
	ErrorCodeRegisteredAdapterValueTypeUnsupported  = 120112
	ErrorCodeRegisteredAdapterOperationUnsupported  = 120113
	ErrorCodeRegisteredAdapterOperatorUnsupported   = 120114
	ErrorCodeRegisteredAdapterExecutionInvalid      = 120115
	ErrorCodeRegisteredAdapterPartialConversion     = 120116
)

var (
	ErrDataPermissionAdapterInputInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPermissionAdapterInputInvalid,
		"数据权限适配器输入不合法",
	)
	ErrDataPermissionAdapterTypeUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataPermissionAdapterTypeUnsupported,
		"数据权限适配器类型不支持",
	)
	ErrDataPermissionAdapterOwnershipMissing = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataPermissionAdapterOwnershipMissing,
		"数据权限适配器归属定义缺失",
	)
	ErrDataPermissionAdapterOwnershipMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPermissionAdapterOwnershipMismatch,
		"数据权限适配器归属定义不匹配",
	)
	ErrDataPermissionAdapterExecutionInvalid = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPermissionAdapterExecutionInvalid,
		"数据权限适配器执行结果不合法",
	)
	ErrDataPermissionAdapterFailed = newApplicationError(KindInternal, CategoryBusiness,
		ErrorCodeDataPermissionAdapterFailed,
		"数据权限适配失败",
	)
	ErrMetadataAdapterResourceTableMissing = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeMetadataAdapterResourceTableMissing,
		"数据权限资源缺少元数据表绑定",
	)
	ErrMetadataAdapterTableNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeMetadataAdapterTableNotFound,
		"数据权限元数据表不存在或不可用",
	)
	ErrMetadataAdapterFieldNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeMetadataAdapterFieldNotFound,
		"数据权限元数据字段不存在",
	)
	ErrMetadataAdapterFieldResourceMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeMetadataAdapterFieldResourceMismatch,
		"数据权限元数据字段不属于当前资源",
	)
	ErrMetadataAdapterFieldInactive = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeMetadataAdapterFieldInactive,
		"数据权限元数据字段已停用或删除",
	)
	ErrMetadataAdapterFieldTypeUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeMetadataAdapterFieldTypeUnsupported,
		"数据权限元数据字段类型不支持",
	)
	ErrMetadataAdapterFieldTypeDrift = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeMetadataAdapterFieldTypeDrift,
		"数据权限元数据字段类型已变化",
	)
	ErrMetadataAdapterFieldNotFilterable = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeMetadataAdapterFieldNotFilterable,
		"数据权限元数据字段不可用于过滤",
	)
	ErrMetadataAdapterOperatorUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeMetadataAdapterOperatorUnsupported,
		"数据权限元数据过滤操作不支持",
	)
	ErrMetadataAdapterValueTypeMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeMetadataAdapterValueTypeMismatch,
		"数据权限元数据过滤值类型不匹配",
	)
	ErrMetadataAdapterComplexityExceeded = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeMetadataAdapterComplexityExceeded,
		"数据权限元数据过滤条件超出限制",
	)
	ErrMetadataAdapterFailed = newApplicationError(KindInternal, CategoryBusiness,
		ErrorCodeMetadataAdapterFailed,
		"数据权限元数据字段适配失败",
	)
	ErrRegisteredAdapterUnregistered = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeRegisteredAdapterUnregistered,
		"数据权限注册适配器未注册",
	)
	ErrRegisteredAdapterFieldNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeRegisteredAdapterFieldNotFound,
		"数据权限注册字段不存在",
	)
	ErrRegisteredAdapterRegistrationDuplicate = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeRegisteredAdapterRegistrationDuplicate,
		"数据权限注册字段重复注册",
	)
	ErrRegisteredAdapterRegistrationConflict = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeRegisteredAdapterRegistrationConflict,
		"数据权限注册字段配置冲突",
	)
	ErrRegisteredAdapterResourceMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeRegisteredAdapterResourceMismatch,
		"数据权限注册字段不属于当前资源",
	)
	ErrRegisteredAdapterDimensionUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeRegisteredAdapterDimensionUnsupported,
		"数据权限注册字段不支持当前维度",
	)
	ErrRegisteredAdapterValueTypeUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeRegisteredAdapterValueTypeUnsupported,
		"数据权限注册字段值类型不支持",
	)
	ErrRegisteredAdapterOperationUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeRegisteredAdapterOperationUnsupported,
		"数据权限注册字段不支持当前操作",
	)
	ErrRegisteredAdapterOperatorUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeRegisteredAdapterOperatorUnsupported,
		"数据权限注册字段不支持当前操作符",
	)
	ErrRegisteredAdapterExecutionInvalid = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeRegisteredAdapterExecutionInvalid,
		"数据权限注册字段执行描述不合法",
	)
	ErrRegisteredAdapterPartialConversion = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeRegisteredAdapterPartialConversion,
		"数据权限注册字段条件转换失败",
	)
)

// Resolution application errors.
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
	ErrDataPermissionSubjectUserNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataPermissionSubjectUserNotFound,
		"数据权限主体用户不存在",
	)
	ErrDataPermissionRoleContextMissing = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataPermissionRoleContextMissing,
		"数据权限角色上下文缺失",
	)
	ErrDataPermissionEmployeeUnbound = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPermissionEmployeeUnbound,
		"当前账号未绑定企业人员",
	)
	ErrDataPermissionSubjectContextInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataPermissionSubjectContextInvalid,
		"数据权限主体上下文不合法",
	)
	ErrDataScopeDecisionInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeDecisionInvalid,
		"数据权限结果决策类型不合法",
	)
	ErrDataScopeResultConditionMismatch = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeResultConditionMismatch,
		"数据权限结果与过滤条件不一致",
	)
	ErrDataScopeFilterConditionMissing = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeFilterConditionMissing,
		"数据权限过滤条件缺失",
	)
	ErrDataScopeConditionGroupEmpty = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeConditionGroupEmpty,
		"数据权限条件组不能为空",
	)
	ErrDataScopeOwnershipCodeInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeOwnershipCodeInvalid,
		"数据权限归属编码不合法",
	)
	ErrDataScopeDimensionInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeDimensionInvalid,
		"数据权限维度不合法",
	)
	ErrDataScopeOperatorInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeOperatorInvalid,
		"数据权限过滤操作符不合法",
	)
	ErrDataScopeValueTypeInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeValueTypeInvalid,
		"数据权限值类型不合法",
	)
	ErrDataScopeValueTypeMismatch = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeValueTypeMismatch,
		"数据权限过滤值类型不一致",
	)
	ErrDataScopeValueCountExceeded = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataScopeValueCountExceeded,
		"数据权限过滤值数量超过限制",
	)
	ErrDataScopeMergeUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataScopeMergeUnsupported,
		"当前数据权限结果不支持合并",
	)
	ErrDataScopeResultIdentityInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeDataScopeResultIdentityInvalid,
		"数据权限结果资源或操作不合法",
	)
	ErrDataScopeComplexityExceeded = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataScopeComplexityExceeded,
		"数据权限结果复杂度超过限制",
	)
	ErrDataPermissionDimensionNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataPermissionDimensionNotFound,
		"数据权限维度不存在",
	)
	ErrDataPermissionDimensionUnsupported = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataPermissionDimensionUnsupported,
		"数据权限维度暂不支持运行时解析",
	)
	ErrDataPermissionDimensionTypeMismatch = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataPermissionDimensionTypeMismatch,
		"数据权限维度值类型不匹配",
	)
	ErrDataPermissionDimensionProviderFailed = newApplicationError(KindUnavailable, CategoryBusiness,
		ErrorCodeDataPermissionDimensionProviderFailed,
		"数据权限维度Provider调用失败",
	)
	ErrDataPermissionResolverResourceMissing = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataPermissionResolverResourceMissing,
		"数据权限解析资源不存在",
	)
	ErrDataPermissionResolverOperationMissing = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeDataPermissionResolverOperationMissing,
		"数据权限解析操作不存在",
	)
	ErrDataPermissionResolverGrantMissing = newApplicationError(KindForbidden, CategoryBusiness,
		ErrorCodeDataPermissionResolverGrantMissing,
		"数据权限授权不存在",
	)
	ErrDataPermissionResolverPolicyInvalid = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataPermissionResolverPolicyInvalid,
		"数据权限策略无效",
	)
	ErrDataPermissionResolverOwnershipMissing = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeDataPermissionResolverOwnershipMissing,
		"数据权限归属定义缺失",
	)
	ErrDataPermissionResolverDimensionFailed = newApplicationError(KindUnavailable, CategoryBusiness,
		ErrorCodeDataPermissionResolverDimensionFailed,
		"数据权限维度解析失败",
	)
	ErrDataPermissionResolverConfigConflict = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPermissionResolverConfigConflict,
		"数据权限解析配置冲突",
	)
	ErrDataPermissionResolverFailed = newApplicationError(KindInternal, CategoryBusiness,
		ErrorCodeDataPermissionResolverFailed,
		"数据权限解析失败",
	)
)

// Runtime application errors.
const (
	ErrorCodeDataPermissionRuntimeRouteConflict  = 120117
	ErrorCodeDataPermissionRuntimeFailed         = 120118
	ErrorCodeDataPermissionFilterApplyFailed     = 120119
	ErrorCodeDataPermissionOwnershipUpdateDenied = 120120
)

var (
	ErrDataPermissionRuntimeRouteConflict = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeDataPermissionRuntimeRouteConflict,
		"数据权限运行时路由冲突",
	)
	ErrDataPermissionRuntimeFailed = newApplicationError(KindInternal, CategoryBusiness,
		ErrorCodeDataPermissionRuntimeFailed,
		"数据权限运行时处理失败",
	)
	ErrDataPermissionFilterApplyFailed = newApplicationError(KindInternal, CategoryBusiness,
		ErrorCodeDataPermissionFilterApplyFailed,
		"数据权限过滤应用失败",
	)
	ErrDataPermissionOwnershipUpdateDenied = newApplicationError(KindForbidden, CategoryPermission,
		ErrorCodeDataPermissionOwnershipUpdateDenied,
		"当前阶段不允许通过通用接口修改数据归属字段",
	)
)
