package errors

import "net/http"

const (
	ErrorCodeDataResourceNotFound               = 120001
	ErrorCodeDataResourceCodeDuplicate          = 120002
	ErrorCodeDataResourceCodeInvalid            = 120003
	ErrorCodeDataResourceReferenced             = 120004
	ErrorCodeDataResourceStateInvalid           = 120005
	ErrorCodeDataResourceOperationDuplicate     = 120006
	ErrorCodeDataResourceOperationInvalid       = 120007
	ErrorCodeDataResourceFieldImmutable         = 120008
	ErrorCodeDataResourcePermissionEnableDenied = 120009
	ErrorCodeDataResourceOperationReferenced    = 120010
	ErrorCodeDataResourceNameRequired           = 120011
	ErrorCodeDataResourceTypeInvalid            = 120012
	ErrorCodeDataResourceTargetInvalid          = 120013
	ErrorCodeDataResourceOperationNotFound      = 120014
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
)
