package errors

const (
	ErrorCodeGeneric          = 10000
	ErrorCodeParamInvalid     = 20003
	ErrorCodeDataNotFound     = 20013
	ErrorCodePermissionDenied = 30006
)

var (
	ErrInternalServer   = newApplicationError(KindInternal, CategorySystem, ErrorCodeGeneric, "系统异常")
	ErrParamInvalid     = newApplicationError(KindInvalidArgument, CategoryParameter, ErrorCodeParamInvalid, "参数错误")
	ErrDataNotFound     = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeDataNotFound, "操作的数据不存在")
	ErrPermissionDenied = newApplicationError(KindForbidden, CategoryPermission, ErrorCodePermissionDenied, "无权限访问")
)
