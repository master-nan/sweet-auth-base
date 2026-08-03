package errors

import "net/http"

const (
	ErrorCodeDataPermissionRuntimeRouteConflict  = 120117
	ErrorCodeDataPermissionRuntimeFailed         = 120118
	ErrorCodeDataPermissionFilterApplyFailed     = 120119
	ErrorCodeDataPermissionOwnershipUpdateDenied = 120120
)

var (
	ErrDataPermissionRuntimeRouteConflict = NewBusinessError(
		http.StatusConflict,
		ErrorCodeDataPermissionRuntimeRouteConflict,
		"数据权限运行时路由冲突",
	)
	ErrDataPermissionRuntimeFailed = NewBusinessError(
		http.StatusInternalServerError,
		ErrorCodeDataPermissionRuntimeFailed,
		"数据权限运行时处理失败",
	)
	ErrDataPermissionFilterApplyFailed = NewBusinessError(
		http.StatusInternalServerError,
		ErrorCodeDataPermissionFilterApplyFailed,
		"数据权限过滤应用失败",
	)
	ErrDataPermissionOwnershipUpdateDenied = NewPermissionError(
		http.StatusForbidden,
		ErrorCodeDataPermissionOwnershipUpdateDenied,
		"当前阶段不允许通过通用接口修改数据归属字段",
	)
)
