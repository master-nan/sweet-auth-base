package errors

import "net/http"

const (
	ErrorCodeIntegrationExecutionNotFound                = 130301
	ErrorCodeIntegrationExecutionIdempotencyConflict     = 130302
	ErrorCodeIntegrationExecutionStatusInvalid           = 130303
	ErrorCodeIntegrationExecutionRevisionConflict        = 130304
	ErrorCodeIntegrationExecutionConfigurationInvalid    = 130305
	ErrorCodeIntegrationCredentialNotFound               = 130306
	ErrorCodeIntegrationCredentialSystemMismatch         = 130307
	ErrorCodeIntegrationCredentialInterfaceMismatch      = 130308
	ErrorCodeIntegrationCredentialInactive               = 130309
	ErrorCodeIntegrationCredentialExpired                = 130310
	ErrorCodeIntegrationCredentialRevoked                = 130311
	ErrorCodeIntegrationCredentialTypeUnsupported        = 130312
	ErrorCodeIntegrationCredentialSecretMissing          = 130313
	ErrorCodeIntegrationCredentialDecryptFailed          = 130314
	ErrorCodeIntegrationCredentialMaterialInvalid        = 130315
	ErrorCodeIntegrationCredentialInjectionInvalid       = 130316
	ErrorCodeIntegrationExecutionClaimConflict           = 130317
	ErrorCodeIntegrationExecutionLeaseLost               = 130318
	ErrorCodeIntegrationAttemptCreateFailed              = 130319
	ErrorCodeIntegrationAttemptAlreadyCompleted          = 130320
	ErrorCodeIntegrationConfigurationUnavailable         = 130321
	ErrorCodeIntegrationCredentialResolutionFailed       = 130322
	ErrorCodeIntegrationTransportFailed                  = 130323
	ErrorCodeIntegrationExecutionCompleteFailed          = 130324
	ErrorCodeIntegrationExecutionResultUnknown           = 130325
	ErrorCodeIntegrationConcurrencyLimitReached          = 130326
	ErrorCodeIntegrationLeaseRecoveryFailed              = 130327
	ErrorCodeIntegrationWorkerDisabled                   = 130328
	ErrorCodeIntegrationWorkerAlreadyRunning             = 130329
	ErrorCodeIntegrationWorkerInvalidConfig              = 130330
	ErrorCodeIntegrationWorkerStartFailed                = 130331
	ErrorCodeIntegrationWorkerPollFailed                 = 130332
	ErrorCodeIntegrationWorkerClaimFailed                = 130333
	ErrorCodeIntegrationWorkerExecutionFailed            = 130334
	ErrorCodeIntegrationWorkerRecoveryFailed             = 130335
	ErrorCodeIntegrationWorkerShutdownTimeout            = 130336
	ErrorCodeIntegrationWorkerPanicRecovered             = 130337
	ErrorCodeIntegrationExecutionInputMissing            = 130338
	ErrorCodeIntegrationExecutionInputInvalid            = 130339
	ErrorCodeIntegrationExecutionInputSemanticTooLarge   = 130340
	ErrorCodeIntegrationExecutionInputContractMismatch   = 130341
	ErrorCodeIntegrationExecutionInputHashMismatch       = 130342
	ErrorCodeIntegrationExecutionInputVersionUnsupported = 130343
	ErrorCodeIntegrationExecutionPathParameterMissing    = 130344
	ErrorCodeIntegrationExecutionPathParameterUnknown    = 130345
	ErrorCodeIntegrationExecutionQueryParameterInvalid   = 130346
	ErrorCodeIntegrationExecutionHeaderNotAllowed        = 130347
	ErrorCodeIntegrationExecutionBodyInvalid             = 130348
	ErrorCodeIntegrationExecutionSensitiveInputRejected  = 130349
	ErrorCodeIntegrationExecutionInputStorageFailed      = 130350
	ErrorCodeIntegrationExecutionInputLoadFailed         = 130351
	ErrorCodeIntegrationTimeoutOutOfRange                = 130352
	ErrorCodeIntegrationResponseLimitOutOfRange          = 130353
	ErrorCodeIntegrationRuntimeContractInvalid           = 130354
	ErrorCodeIntegrationLeaseDurationInvalid             = 130355
	ErrorCodeIntegrationLeaseMarginInsufficient          = 130356
	ErrorCodeIntegrationInterfaceRuntimeIncompatible     = 130357
	ErrorCodeIntegrationExecutionRuntimeIncompatible     = 130358
	ErrorCodeIntegrationExecutionInputStorageTooLarge    = 130359
	ErrorCodeIntegrationExecutionInputSizeMismatch       = 130360
	ErrorCodeIntegrationRetrySnapshotInvalid             = 130361
	ErrorCodeIntegrationRetryPolicyInvalid               = 130362
	ErrorCodeIntegrationRetryScheduleInvalid             = 130363
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
	ErrIntegrationExecutionClaimConflict = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionClaimConflict,
		"集成执行已被其他 Worker 领取",
	)
	ErrIntegrationExecutionLeaseLost = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionLeaseLost,
		"集成执行租约已失效",
	)
	ErrIntegrationAttemptCreateFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationAttemptCreateFailed,
		"集成执行尝试创建失败",
	)
	ErrIntegrationAttemptAlreadyCompleted = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationAttemptAlreadyCompleted,
		"集成执行尝试已完成",
	)
	ErrIntegrationConfigurationUnavailable = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationConfigurationUnavailable,
		"集成运行配置不可用",
	)
	ErrIntegrationCredentialResolutionFailed = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationCredentialResolutionFailed,
		"集成运行凭证解析失败",
	)
	ErrIntegrationTransportFailed = NewError(
		http.StatusBadGateway,
		ErrorCodeIntegrationTransportFailed,
		"集成远程调用失败",
	)
	ErrIntegrationExecutionCompleteFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationExecutionCompleteFailed,
		"集成执行结果保存失败",
	)
	ErrIntegrationExecutionResultUnknown = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionResultUnknown,
		"集成远程调用结果无法确认",
	)
	ErrIntegrationConcurrencyLimitReached = NewBusinessError(
		http.StatusTooManyRequests,
		ErrorCodeIntegrationConcurrencyLimitReached,
		"集成执行并发已达上限",
	)
	ErrIntegrationLeaseRecoveryFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationLeaseRecoveryFailed,
		"集成执行租约恢复失败",
	)
	ErrIntegrationWorkerDisabled = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationWorkerDisabled,
		"集成常驻 Worker 未启用",
	)
	ErrIntegrationWorkerAlreadyRunning = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationWorkerAlreadyRunning,
		"集成常驻 Worker 已在运行",
	)
	ErrIntegrationWorkerInvalidConfig = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationWorkerInvalidConfig,
		"集成常驻 Worker 配置不合法",
	)
	ErrIntegrationWorkerStartFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationWorkerStartFailed,
		"集成常驻 Worker 启动失败",
	)
	ErrIntegrationWorkerPollFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationWorkerPollFailed,
		"集成常驻 Worker 轮询失败",
	)
	ErrIntegrationWorkerClaimFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationWorkerClaimFailed,
		"集成常驻 Worker 领取执行失败",
	)
	ErrIntegrationWorkerExecutionFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationWorkerExecutionFailed,
		"集成常驻 Worker 执行失败",
	)
	ErrIntegrationWorkerRecoveryFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationWorkerRecoveryFailed,
		"集成常驻 Worker 租约恢复失败",
	)
	ErrIntegrationWorkerShutdownTimeout = NewError(
		http.StatusGatewayTimeout,
		ErrorCodeIntegrationWorkerShutdownTimeout,
		"集成常驻 Worker 优雅关闭超时",
	)
	ErrIntegrationWorkerPanicRecovered = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationWorkerPanicRecovered,
		"集成常驻 Worker 执行异常已恢复",
	)
	ErrIntegrationExecutionInputMissing = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionInputMissing,
		"集成执行输入快照缺失",
	)
	ErrIntegrationExecutionInputInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationExecutionInputInvalid,
		"集成执行输入不合法",
	)
	ErrIntegrationExecutionInputSemanticTooLarge = NewBusinessError(
		http.StatusRequestEntityTooLarge,
		ErrorCodeIntegrationExecutionInputSemanticTooLarge,
		"集成执行输入语义大小超过平台限制",
	)
	ErrIntegrationExecutionInputContractMismatch = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionInputContractMismatch,
		"集成执行输入与接口契约不匹配",
	)
	ErrIntegrationExecutionInputHashMismatch = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionInputHashMismatch,
		"集成执行输入完整性校验失败",
	)
	ErrIntegrationExecutionInputVersionUnsupported = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionInputVersionUnsupported,
		"集成执行输入快照版本不受支持",
	)
	ErrIntegrationExecutionPathParameterMissing = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationExecutionPathParameterMissing,
		"集成执行缺少必填 Path 参数",
	)
	ErrIntegrationExecutionPathParameterUnknown = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationExecutionPathParameterUnknown,
		"集成执行包含未声明的 Path 参数",
	)
	ErrIntegrationExecutionQueryParameterInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationExecutionQueryParameterInvalid,
		"集成执行 Query 参数不合法",
	)
	ErrIntegrationExecutionHeaderNotAllowed = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationExecutionHeaderNotAllowed,
		"集成执行 Header 不在允许范围内",
	)
	ErrIntegrationExecutionBodyInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationExecutionBodyInvalid,
		"集成执行 JSON Body 不合法",
	)
	ErrIntegrationExecutionSensitiveInputRejected = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationExecutionSensitiveInputRejected,
		"集成执行不允许保存敏感输入",
	)
	ErrIntegrationExecutionInputStorageFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationExecutionInputStorageFailed,
		"集成执行输入快照保存失败",
	)
	ErrIntegrationExecutionInputLoadFailed = NewError(
		http.StatusInternalServerError,
		ErrorCodeIntegrationExecutionInputLoadFailed,
		"集成执行输入快照加载失败",
	)
	ErrIntegrationTimeoutOutOfRange = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationTimeoutOutOfRange,
		"接口请求超时超出平台运行范围",
	)
	ErrIntegrationResponseLimitOutOfRange = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationResponseLimitOutOfRange,
		"接口响应大小限制超出平台运行范围",
	)
	ErrIntegrationRuntimeContractInvalid = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationRuntimeContractInvalid,
		"集成运行参数契约不合法",
	)
	ErrIntegrationLeaseDurationInvalid = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationLeaseDurationInvalid,
		"集成执行租约时长不合法",
	)
	ErrIntegrationLeaseMarginInsufficient = NewBusinessError(
		http.StatusBadRequest,
		ErrorCodeIntegrationLeaseMarginInsufficient,
		"集成执行租约安全余量不足",
	)
	ErrIntegrationInterfaceRuntimeIncompatible = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationInterfaceRuntimeIncompatible,
		"接口定义与当前集成运行契约不兼容",
	)
	ErrIntegrationExecutionRuntimeIncompatible = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionRuntimeIncompatible,
		"集成执行引用的接口不符合当前运行契约",
	)
	ErrIntegrationExecutionInputStorageTooLarge = NewBusinessError(
		http.StatusRequestEntityTooLarge,
		ErrorCodeIntegrationExecutionInputStorageTooLarge,
		"集成执行输入存储大小超过平台限制",
	)
	ErrIntegrationExecutionInputSizeMismatch = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationExecutionInputSizeMismatch,
		"集成执行输入大小完整性校验失败",
	)
	ErrIntegrationRetrySnapshotInvalid = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationRetrySnapshotInvalid,
		"集成重试策略快照无效",
	)
	ErrIntegrationRetryPolicyInvalid = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationRetryPolicyInvalid,
		"集成重试策略无效",
	)
	ErrIntegrationRetryScheduleInvalid = NewBusinessError(
		http.StatusConflict,
		ErrorCodeIntegrationRetryScheduleInvalid,
		"集成重试调度参数无效",
	)
)
