package errors

import "net/http"

const (
	ErrorCodeIntegrationExecutionNotFound             = 130301
	ErrorCodeIntegrationExecutionIdempotencyConflict  = 130302
	ErrorCodeIntegrationExecutionStatusInvalid        = 130303
	ErrorCodeIntegrationExecutionRevisionConflict     = 130304
	ErrorCodeIntegrationExecutionConfigurationInvalid = 130305
)

var (
	ErrIntegrationExecutionNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeIntegrationExecutionNotFound,
		"集成执行不存在",
	)
	ErrIntegrationExecutionIdempotencyConflict = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionIdempotencyConflict,
		"幂等键已用于不同的集成执行输入",
	)
	ErrIntegrationExecutionStatusInvalid = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionStatusInvalid,
		"集成执行当前状态不允许执行该操作",
	)
	ErrIntegrationExecutionRevisionConflict = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionRevisionConflict,
		"集成执行已被其他操作修改，请刷新后重试",
	)
	ErrIntegrationExecutionConfigurationInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationExecutionConfigurationInvalid,
		"集成执行配置不存在、未启用或输入不合法",
	)
)
