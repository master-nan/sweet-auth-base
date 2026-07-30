package errors

import "net/http"

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
	ErrorCodeDataOwnershipMetadataFieldInvalid    = 120020
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
)

var (
	ErrDataResourceNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataResourceNotFound,
		"数据资源不存在",
	)
	ErrDataResourceCodeDuplicate = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataResourceCodeDuplicate,
		"数据资源编码已存在",
	)
	ErrDataResourceCodeInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataResourceCodeInvalid,
		"数据资源编码格式不合法",
	)
	ErrDataResourceReferenced = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataResourceReferenced,
		"数据资源已被配置引用",
	)
	ErrDataResourceStateInvalid = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataResourceStateInvalid,
		"数据资源当前状态不允许执行该操作",
	)
	ErrDataResourceOperationDuplicate = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataResourceOperationDuplicate,
		"数据资源操作已存在",
	)
	ErrDataResourceOperationInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataResourceOperationInvalid,
		"数据资源操作取值不合法",
	)
	ErrDataResourceFieldImmutable = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataResourceFieldImmutable,
		"数据资源身份字段不可修改",
	)
	ErrDataResourcePermissionEnableDenied = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataResourcePermissionEnableDenied,
		"当前阶段不允许启用数据权限",
	)
	ErrDataResourceOperationReferenced = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataResourceOperationReferenced,
		"数据资源操作已被授权引用",
	)
	ErrDataResourceNameRequired = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataResourceNameRequired,
		"数据资源名称不能为空",
	)
	ErrDataResourceTypeInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataResourceTypeInvalid,
		"数据资源类型不合法",
	)
	ErrDataResourceTargetInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataResourceTargetInvalid,
		"数据资源目标配置不合法",
	)
	ErrDataResourceOperationNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataResourceOperationNotFound,
		"数据资源操作不存在",
	)
	ErrDataDimensionNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataDimensionNotFound,
		"数据权限维度不存在或已停用",
	)
	ErrDataOwnershipNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataOwnershipNotFound,
		"数据归属定义不存在",
	)
	ErrDataOwnershipDuplicate = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataOwnershipDuplicate,
		"数据归属编码在当前资源中已存在",
	)
	ErrDataOwnershipCodeInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipCodeInvalid,
		"数据归属编码格式不合法",
	)
	ErrDataOwnershipBindingInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipBindingInvalid,
		"数据归属绑定配置不合法",
	)
	ErrDataOwnershipMetadataFieldInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipMetadataFieldInvalid,
		"数据归属元数据字段不存在或不可用",
	)
	ErrDataOwnershipRegisteredFieldInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipRegisteredFieldInvalid,
		"数据归属注册字段不合法",
	)
	ErrDataOwnershipValueTypeMismatch = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipValueTypeMismatch,
		"数据归属值类型与维度不兼容",
	)
	ErrDataOwnershipReferenced = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataOwnershipReferenced,
		"数据归属定义已被策略引用",
	)
	ErrDataOwnershipFieldImmutable = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataOwnershipFieldImmutable,
		"数据归属身份字段不可修改",
	)
	ErrDataOwnershipMetadataFieldNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataOwnershipMetadataFieldNotFound,
		"数据归属元数据字段不存在或已删除",
	)
	ErrDataOwnershipMetadataFieldMismatch = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipMetadataFieldMismatch,
		"数据归属元数据字段不属于当前资源",
	)
	ErrDataOwnershipMetadataFieldForbidden = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipMetadataFieldForbidden,
		"该元数据字段不能作为数据归属字段",
	)
	ErrDataOwnershipRegisteredFieldMissing = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipRegisteredFieldMissing,
		"数据归属注册项不存在",
	)
	ErrDataOwnershipRegisteredResource = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipRegisteredResource,
		"数据归属注册项与当前资源不匹配",
	)
	ErrDataOwnershipRegisteredDimension = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipRegisteredDimension,
		"数据归属注册项不支持当前维度",
	)
	ErrDataOwnershipRegisteredOperation = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipRegisteredOperation,
		"数据归属注册项不支持当前操作",
	)
	ErrDataOwnershipMetadataDimension = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataOwnershipMetadataDimension,
		"元数据字段与数据权限维度不匹配",
	)
	ErrDataPolicyNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataPolicyNotFound,
		"数据权限策略不存在",
	)
	ErrDataPolicyCodeDuplicate = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPolicyCodeDuplicate,
		"数据权限策略编码已存在",
	)
	ErrDataPolicyCodeInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyCodeInvalid,
		"数据权限策略编码格式不合法",
	)
	ErrDataPolicyNameRequired = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyNameRequired,
		"数据权限策略名称不能为空",
	)
	ErrDataPolicyFieldImmutable = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPolicyFieldImmutable,
		"数据权限策略身份字段不可修改",
	)
	ErrDataPolicyStateInvalid = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPolicyStateInvalid,
		"数据权限策略当前状态不允许执行该操作",
	)
	ErrDataPolicyRuleNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeDataPolicyRuleNotFound,
		"数据权限策略规则不存在",
	)
	ErrDataPolicyRuleDuplicate = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPolicyRuleDuplicate,
		"数据权限策略规则序号已存在",
	)
	ErrDataPolicyRuleOwnershipNotFound = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyRuleOwnershipNotFound,
		"数据权限策略规则未匹配到有效归属定义",
	)
	ErrDataPolicyRuleDimensionMismatch = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyRuleDimensionMismatch,
		"数据权限策略规则的归属编码与维度不匹配",
	)
	ErrDataPolicyRuleScopeSourceInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyRuleScopeSourceInvalid,
		"数据权限策略规则的范围来源与维度不兼容",
	)
	ErrDataPolicyRuleRelationInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyRuleRelationInvalid,
		"数据权限策略规则的关系类型不合法",
	)
	ErrDataPolicyRuleOperatorInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyRuleOperatorInvalid,
		"数据权限策略规则的操作符不合法",
	)
	ErrDataPolicyRuleSpecifiedValuesInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyRuleSpecifiedValuesInvalid,
		"数据权限策略规则的指定值不合法",
	)
	ErrDataPolicyRuleCountInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPolicyRuleCountInvalid,
		"数据权限策略规则数量超过限制",
	)
)
