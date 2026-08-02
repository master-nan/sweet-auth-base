package errors

import "net/http"

const (
	ErrorCodeDataPermissionAdapterInputInvalid      = 120088
	ErrorCodeDataPermissionAdapterTypeUnsupported   = 120089
	ErrorCodeDataPermissionAdapterOwnershipMissing  = 120090
	ErrorCodeDataPermissionAdapterOwnershipMismatch = 120091
	ErrorCodeDataPermissionAdapterExecutionInvalid  = 120092
	ErrorCodeDataPermissionAdapterFailed            = 120093
)

var (
	ErrDataPermissionAdapterInputInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeDataPermissionAdapterInputInvalid,
		"数据权限适配器输入不合法",
	)
	ErrDataPermissionAdapterTypeUnsupported = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataPermissionAdapterTypeUnsupported,
		"数据权限适配器类型不支持",
	)
	ErrDataPermissionAdapterOwnershipMissing = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeDataPermissionAdapterOwnershipMissing,
		"数据权限适配器归属定义缺失",
	)
	ErrDataPermissionAdapterOwnershipMismatch = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPermissionAdapterOwnershipMismatch,
		"数据权限适配器归属定义不匹配",
	)
	ErrDataPermissionAdapterExecutionInvalid = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPermissionAdapterExecutionInvalid,
		"数据权限适配器执行结果不合法",
	)
	ErrDataPermissionAdapterFailed = NewBusinessError(
		http.StatusInternalServerError,
		ErrorCodeDataPermissionAdapterFailed,
		"数据权限适配失败",
	)
)
