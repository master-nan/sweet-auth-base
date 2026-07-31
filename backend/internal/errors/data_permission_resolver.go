package errors

import "net/http"

const (
	ErrorCodeDataPermissionSubjectUserNotFound   = 120059
	ErrorCodeDataPermissionRoleContextMissing    = 120060
	ErrorCodeDataPermissionEmployeeUnbound       = 120061
	ErrorCodeDataPermissionSubjectContextInvalid = 120062
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
)
