package errors

import "net/http"

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
	ErrMetadataAdapterResourceTableMissing = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeMetadataAdapterResourceTableMissing,
		"数据权限资源缺少元数据表绑定",
	)
	ErrMetadataAdapterTableNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeMetadataAdapterTableNotFound,
		"数据权限元数据表不存在或不可用",
	)
	ErrMetadataAdapterFieldNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeMetadataAdapterFieldNotFound,
		"数据权限元数据字段不存在",
	)
	ErrMetadataAdapterFieldResourceMismatch = NewBusinessError(
		http.StatusConflict,
		ErrorCodeMetadataAdapterFieldResourceMismatch,
		"数据权限元数据字段不属于当前资源",
	)
	ErrMetadataAdapterFieldInactive = NewBusinessError(
		http.StatusConflict,
		ErrorCodeMetadataAdapterFieldInactive,
		"数据权限元数据字段已停用或删除",
	)
	ErrMetadataAdapterFieldTypeUnsupported = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeMetadataAdapterFieldTypeUnsupported,
		"数据权限元数据字段类型不支持",
	)
	ErrMetadataAdapterFieldTypeDrift = NewBusinessError(
		http.StatusConflict,
		ErrorCodeMetadataAdapterFieldTypeDrift,
		"数据权限元数据字段类型已变化",
	)
	ErrMetadataAdapterFieldNotFilterable = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeMetadataAdapterFieldNotFilterable,
		"数据权限元数据字段不可用于过滤",
	)
	ErrMetadataAdapterOperatorUnsupported = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeMetadataAdapterOperatorUnsupported,
		"数据权限元数据过滤操作不支持",
	)
	ErrMetadataAdapterValueTypeMismatch = NewBusinessError(
		http.StatusConflict,
		ErrorCodeMetadataAdapterValueTypeMismatch,
		"数据权限元数据过滤值类型不匹配",
	)
	ErrMetadataAdapterComplexityExceeded = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeMetadataAdapterComplexityExceeded,
		"数据权限元数据过滤条件超出限制",
	)
	ErrMetadataAdapterFailed = NewBusinessError(
		http.StatusInternalServerError,
		ErrorCodeMetadataAdapterFailed,
		"数据权限元数据字段适配失败",
	)
)
