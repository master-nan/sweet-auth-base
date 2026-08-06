package errors

import "net/http"

const (
	ErrorCodeIntegrationExecutionNotFound             = 130301
	ErrorCodeIntegrationExecutionIdempotencyConflict  = 130302
	ErrorCodeIntegrationExecutionStatusInvalid        = 130303
	ErrorCodeIntegrationExecutionRevisionConflict     = 130304
	ErrorCodeIntegrationExecutionConfigurationInvalid = 130305
	ErrorCodeIntegrationCredentialNotFound            = 130306
	ErrorCodeIntegrationCredentialSystemMismatch      = 130307
	ErrorCodeIntegrationCredentialInterfaceMismatch   = 130308
	ErrorCodeIntegrationCredentialInactive            = 130309
	ErrorCodeIntegrationCredentialExpired             = 130310
	ErrorCodeIntegrationCredentialRevoked             = 130311
	ErrorCodeIntegrationCredentialTypeUnsupported     = 130312
	ErrorCodeIntegrationCredentialSecretMissing       = 130313
	ErrorCodeIntegrationCredentialDecryptFailed       = 130314
	ErrorCodeIntegrationCredentialMaterialInvalid     = 130315
	ErrorCodeIntegrationCredentialInjectionInvalid    = 130316
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
	ErrIntegrationCredentialNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeIntegrationCredentialNotFound,
		"运行时凭证不存在",
	)
	ErrIntegrationCredentialSystemMismatch = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationCredentialSystemMismatch,
		"运行时凭证不属于目标外部系统",
	)
	ErrIntegrationCredentialInterfaceMismatch = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationCredentialInterfaceMismatch,
		"运行时凭证与接口定义不匹配",
	)
	ErrIntegrationCredentialInactive = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationCredentialInactive,
		"运行时凭证未启用",
	)
	ErrIntegrationCredentialExpired = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationCredentialExpired,
		"运行时凭证已过期",
	)
	ErrIntegrationCredentialRevoked = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationCredentialRevoked,
		"运行时凭证已吊销",
	)
	ErrIntegrationCredentialTypeUnsupported = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationCredentialTypeUnsupported,
		"运行时凭证类型暂不支持",
	)
	ErrIntegrationCredentialSecretMissing = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationCredentialSecretMissing,
		"运行时凭证秘密材料不完整",
	)
	ErrIntegrationCredentialDecryptFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationCredentialDecryptFailed,
		"运行时凭证秘密无法安全解析",
	)
	ErrIntegrationCredentialMaterialInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationCredentialMaterialInvalid,
		"运行时凭证材料不合法",
	)
	ErrIntegrationCredentialInjectionInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationCredentialInjectionInvalid,
		"运行时凭证认证注入不合法",
	)
)
