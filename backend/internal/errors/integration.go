package errors

// Configuration application errors.
const (
	ErrorCodeExternalSystemNotFound          = 130001
	ErrorCodeExternalSystemCodeDuplicate     = 130002
	ErrorCodeExternalSystemCodeInvalid       = 130003
	ErrorCodeExternalSystemFieldImmutable    = 130004
	ErrorCodeExternalSystemStatusInvalid     = 130005
	ErrorCodeExternalSystemConfiguration     = 130006
	ErrorCodeExternalSystemBaseURLInvalid    = 130007
	ErrorCodeExternalSystemRevisionConflict  = 130008
	ErrorCodeExternalSystemReferenced        = 130009
	ErrorCodeInterfaceDefinitionNotFound     = 130101
	ErrorCodeInterfaceCodeDuplicate          = 130102
	ErrorCodeInterfaceCodeInvalid            = 130103
	ErrorCodeInterfaceFieldImmutable         = 130104
	ErrorCodeInterfaceStatusInvalid          = 130105
	ErrorCodeInterfaceConfigurationInvalid   = 130106
	ErrorCodeInterfacePathInvalid            = 130107
	ErrorCodeInterfaceRevisionConflict       = 130108
	ErrorCodeInterfaceExternalSystemInvalid  = 130109
	ErrorCodeInterfaceCredentialInvalid      = 130110
	ErrorCodeInterfaceRetryPolicyInvalid     = 130111
	ErrorCodeInterfaceEnabledVersionConflict = 130112
	ErrorCodeCredentialNotFound              = 130201
	ErrorCodeCredentialCodeDuplicate         = 130202
	ErrorCodeCredentialCodeInvalid           = 130203
	ErrorCodeCredentialFieldImmutable        = 130204
	ErrorCodeCredentialTypeInvalid           = 130205
	ErrorCodeCredentialSecretInvalid         = 130206
	ErrorCodeCredentialStatusInvalid         = 130207
	ErrorCodeCredentialRevisionConflict      = 130208
	ErrorCodeCredentialExternalSystemInvalid = 130209
	ErrorCodeCredentialExpired               = 130210
	ErrorCodeCredentialProtectionFailed      = 130211
	ErrorCodeRetryPolicyNotFound             = 130601
	ErrorCodeRetryPolicyCodeDuplicate        = 130602
	ErrorCodeRetryPolicyCodeInvalid          = 130603
	ErrorCodeRetryPolicyFieldImmutable       = 130604
	ErrorCodeRetryPolicyStatusInvalid        = 130605
	ErrorCodeRetryPolicyConfigurationInvalid = 130606
	ErrorCodeRetryPolicyRevisionConflict     = 130607
	ErrorCodeRetryPolicyEnabledConflict      = 130608
	ErrorCodeRetryPolicyReferenced           = 130609
	ErrorCodeSyncTaskNotFound                = 130401
	ErrorCodeSyncTaskCodeDuplicate           = 130402
	ErrorCodeSyncTaskCodeInvalid             = 130403
	ErrorCodeSyncTaskFieldImmutable          = 130404
	ErrorCodeSyncTaskStatusInvalid           = 130405
	ErrorCodeSyncTaskConfigurationInvalid    = 130406
	ErrorCodeSyncTaskRevisionConflict        = 130407
	ErrorCodeSyncTaskEnabledConflict         = 130408
	ErrorCodeSyncTaskActiveBatch             = 130409
	ErrorCodeSyncInterfaceInvalid            = 130410
	ErrorCodeSyncConsumerNotRegistered       = 130411
	ErrorCodeSyncConsumerIncompatible        = 130412
	ErrorCodeSyncLeaseBudgetInsufficient     = 130413
	ErrorCodeSyncScheduleInvalid             = 130414
	ErrorCodeSyncTimezoneInvalid             = 130415
	ErrorCodeSyncCheckpointInvalid           = 130416
	ErrorCodeSyncInputPlanInvalid            = 130417
	ErrorCodeSyncInputPlanContractMismatch   = 130418
	ErrorCodeSyncBatchNotFound               = 130501
	ErrorCodeSyncSchedulerClaimFailed        = 130502
	ErrorCodeSyncBatchConflict               = 130503
	ErrorCodeSyncBatchStateInvalid           = 130504
	ErrorCodeSyncCheckpointConflict          = 130505
	ErrorCodeSyncExecutionCreateFailed       = 130506
	ErrorCodeSyncBusinessResultPending       = 130507
	ErrorCodeSyncRunnerInvalidConfig         = 130508
	ErrorCodeSyncRunnerAlreadyRunning        = 130509
	ErrorCodeSyncRunnerStartFailed           = 130510
	ErrorCodeSyncRunnerShutdownTimeout       = 130511
	ErrorCodeSyncConsumerDuplicate           = 130512
	ErrorCodeSyncConsumerRegistrationInvalid = 130513
	ErrorCodeSyncConsumptionRequestInvalid   = 130514
	ErrorCodeSyncConsumptionResultInvalid    = 130515
	ErrorCodeSyncConsumerTimeout             = 130516
	ErrorCodeSyncConsumerPanic               = 130517
	ErrorCodeSyncBusinessProcessingFailed    = 130518
)

var (
	ErrExternalSystemNotFound          = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeExternalSystemNotFound, "外部系统不存在")
	ErrExternalSystemCodeDuplicate     = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeExternalSystemCodeDuplicate, "外部系统编码已存在")
	ErrExternalSystemCodeInvalid       = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeExternalSystemCodeInvalid, "外部系统编码格式不合法")
	ErrExternalSystemFieldImmutable    = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeExternalSystemFieldImmutable, "外部系统身份字段不可修改")
	ErrExternalSystemStatusInvalid     = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeExternalSystemStatusInvalid, "外部系统当前状态不允许执行该操作")
	ErrExternalSystemConfiguration     = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeExternalSystemConfiguration, "外部系统必要配置不完整")
	ErrExternalSystemBaseURLInvalid    = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeExternalSystemBaseURLInvalid, "外部系统基础地址不合法")
	ErrExternalSystemRevisionConflict  = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeExternalSystemRevisionConflict, "外部系统已被其他操作修改，请刷新后重试")
	ErrExternalSystemReferenced        = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeExternalSystemReferenced, "外部系统已被配置引用")
	ErrInterfaceDefinitionNotFound     = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeInterfaceDefinitionNotFound, "接口定义不存在")
	ErrInterfaceCodeDuplicate          = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeInterfaceCodeDuplicate, "接口编码和版本已存在")
	ErrInterfaceCodeInvalid            = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeInterfaceCodeInvalid, "接口编码格式不合法")
	ErrInterfaceFieldImmutable         = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeInterfaceFieldImmutable, "接口定义身份字段不可修改")
	ErrInterfaceStatusInvalid          = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeInterfaceStatusInvalid, "接口定义当前状态不允许执行该操作")
	ErrInterfaceConfigurationInvalid   = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeInterfaceConfigurationInvalid, "接口定义必要配置不合法")
	ErrInterfacePathInvalid            = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeInterfacePathInvalid, "接口相对路径不合法")
	ErrInterfaceRevisionConflict       = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeInterfaceRevisionConflict, "接口定义已被其他操作修改，请刷新后重试")
	ErrInterfaceExternalSystemInvalid  = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeInterfaceExternalSystemInvalid, "所属外部系统不存在或状态不允许")
	ErrInterfaceCredentialInvalid      = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeInterfaceCredentialInvalid, "接口凭证引用不存在、无效或不属于当前系统")
	ErrInterfaceRetryPolicyInvalid     = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeInterfaceRetryPolicyInvalid, "接口重试策略引用不存在或无效")
	ErrInterfaceEnabledVersionConflict = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeInterfaceEnabledVersionConflict, "同一接口已有启用版本，请先停用后再启用新版本")
	ErrCredentialNotFound              = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeCredentialNotFound, "集成凭证不存在")
	ErrCredentialCodeDuplicate         = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeCredentialCodeDuplicate, "同一外部系统下凭证编码已存在")
	ErrCredentialCodeInvalid           = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeCredentialCodeInvalid, "凭证编码格式不合法")
	ErrCredentialFieldImmutable        = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeCredentialFieldImmutable, "凭证身份字段不可修改")
	ErrCredentialTypeInvalid           = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeCredentialTypeInvalid, "凭证类型不受支持")
	ErrCredentialSecretInvalid         = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeCredentialSecretInvalid, "凭证秘密内容不完整或格式不合法")
	ErrCredentialStatusInvalid         = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeCredentialStatusInvalid, "凭证当前状态不允许执行该操作")
	ErrCredentialRevisionConflict      = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeCredentialRevisionConflict, "凭证已被其他操作修改，请刷新后重试")
	ErrCredentialExternalSystemInvalid = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeCredentialExternalSystemInvalid, "所属外部系统不存在或状态不允许")
	ErrCredentialExpired               = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeCredentialExpired, "凭证已经过期，不能启用")
	ErrCredentialProtectionFailed      = newApplicationError(KindInternal, CategoryBusiness, ErrorCodeCredentialProtectionFailed, "凭证秘密保护失败")
	ErrRetryPolicyNotFound             = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeRetryPolicyNotFound, "重试策略不存在")
	ErrRetryPolicyCodeDuplicate        = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeRetryPolicyCodeDuplicate, "重试策略编码已存在")
	ErrRetryPolicyCodeInvalid          = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeRetryPolicyCodeInvalid, "重试策略编码格式不合法")
	ErrRetryPolicyFieldImmutable       = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeRetryPolicyFieldImmutable, "重试策略身份字段不可修改")
	ErrRetryPolicyStatusInvalid        = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeRetryPolicyStatusInvalid, "重试策略当前状态不允许执行该操作")
	ErrRetryPolicyConfigurationInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeRetryPolicyConfigurationInvalid, "重试策略参数组合不合法")
	ErrRetryPolicyRevisionConflict     = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeRetryPolicyRevisionConflict, "重试策略已被其他操作修改，请刷新后重试")
	ErrRetryPolicyEnabledConflict      = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeRetryPolicyEnabledConflict, "同一策略编码已有启用版本")
	ErrRetryPolicyReferenced           = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeRetryPolicyReferenced, "重试策略仍被已启用接口引用")
	ErrSyncTaskNotFound                = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeSyncTaskNotFound, "同步任务不存在")
	ErrSyncTaskCodeDuplicate           = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncTaskCodeDuplicate, "同步任务编码已存在")
	ErrSyncTaskCodeInvalid             = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncTaskCodeInvalid, "同步任务编码格式不合法")
	ErrSyncTaskFieldImmutable          = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncTaskFieldImmutable, "同步任务技术配置不可直接修改")
	ErrSyncTaskStatusInvalid           = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncTaskStatusInvalid, "同步任务当前状态不允许执行该操作")
	ErrSyncTaskConfigurationInvalid    = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncTaskConfigurationInvalid, "同步任务参数组合不合法")
	ErrSyncTaskRevisionConflict        = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncTaskRevisionConflict, "同步任务已被其他操作修改，请刷新后重试")
	ErrSyncTaskEnabledConflict         = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncTaskEnabledConflict, "同一同步任务编码已有启用版本")
	ErrSyncTaskActiveBatch             = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncTaskActiveBatch, "同步任务存在活动批次")
	ErrSyncInterfaceInvalid            = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncInterfaceInvalid, "同步任务接口引用不存在或不兼容")
	ErrSyncConsumerNotRegistered       = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncConsumerNotRegistered, "同步 Consumer 未注册或不可用")
	ErrSyncConsumerIncompatible        = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncConsumerIncompatible, "同步 Consumer 与接口运行契约不兼容")
	ErrSyncLeaseBudgetInsufficient     = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncLeaseBudgetInsufficient, "同步 Consumer 执行时长超出租约安全预算")
	ErrSyncScheduleInvalid             = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncScheduleInvalid, "同步任务 Cron 配置不合法")
	ErrSyncTimezoneInvalid             = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncTimezoneInvalid, "同步任务时区不合法")
	ErrSyncCheckpointInvalid           = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncCheckpointInvalid, "同步任务 Checkpoint 配置不合法")
	ErrSyncInputPlanInvalid            = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncInputPlanInvalid, "同步输入计划格式不合法")
	ErrSyncInputPlanContractMismatch   = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSyncInputPlanContractMismatch, "同步输入计划不满足接口输入契约")
	ErrSyncBatchNotFound               = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeSyncBatchNotFound, "同步批次不存在")
	ErrSyncSchedulerClaimFailed        = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncSchedulerClaimFailed, "同步调度领取失败")
	ErrSyncBatchConflict               = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncBatchConflict, "同步批次已由其他调度实例创建或推进")
	ErrSyncBatchStateInvalid           = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncBatchStateInvalid, "同步批次当前状态不允许协调")
	ErrSyncCheckpointConflict          = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncCheckpointConflict, "同步 Checkpoint 已被其他运行实例推进")
	ErrSyncExecutionCreateFailed       = newApplicationError(KindInternal, CategoryBusiness, ErrorCodeSyncExecutionCreateFailed, "同步 Execution 创建失败")
	ErrSyncBusinessResultPending       = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncBusinessResultPending, "同步业务结果尚未确认")
	ErrSyncRunnerInvalidConfig         = newApplicationError(KindInternal, CategoryBusiness, ErrorCodeSyncRunnerInvalidConfig, "同步 Runner 配置不合法")
	ErrSyncRunnerAlreadyRunning        = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncRunnerAlreadyRunning, "同步 Runner 已在运行")
	ErrSyncRunnerStartFailed           = newApplicationError(KindInternal, CategoryBusiness, ErrorCodeSyncRunnerStartFailed, "同步 Runner 启动失败")
	ErrSyncRunnerShutdownTimeout       = newApplicationError(KindUnavailable, CategoryBusiness, ErrorCodeSyncRunnerShutdownTimeout, "同步 Runner 关闭等待超时")
	ErrSyncConsumerDuplicate           = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeSyncConsumerDuplicate, "同步 Consumer 编码和版本重复")
	ErrSyncConsumerRegistrationInvalid = newApplicationError(KindInternal, CategoryBusiness, ErrorCodeSyncConsumerRegistrationInvalid, "同步 Consumer 注册信息不合法")
	ErrSyncConsumptionRequestInvalid   = newApplicationError(KindInternal, CategoryBusiness, ErrorCodeSyncConsumptionRequestInvalid, "同步 Consumer 请求摘要不合法")
	ErrSyncConsumptionResultInvalid    = newApplicationError(KindInternal, CategoryBusiness, ErrorCodeSyncConsumptionResultInvalid, "同步 Consumer 结果摘要不合法")
	ErrSyncConsumerTimeout             = newApplicationError(KindTimeout, CategoryBusiness, ErrorCodeSyncConsumerTimeout, "同步 Consumer 处理超时")
	ErrSyncConsumerPanic               = newApplicationError(KindInternal, CategoryBusiness, ErrorCodeSyncConsumerPanic, "同步 Consumer 发生内部异常")
	ErrSyncBusinessProcessingFailed    = newApplicationError(KindUnprocessable, CategoryBusiness, ErrorCodeSyncBusinessProcessingFailed, "同步业务处理失败")
)

// Execution application errors.
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
	ErrorCodeIntegrationExecutionCompleteFailed          = 130324
	ErrorCodeIntegrationExecutionResultUnknown           = 130325
	ErrorCodeIntegrationLeaseRecoveryFailed              = 130327
	ErrorCodeIntegrationWorkerAlreadyRunning             = 130329
	ErrorCodeIntegrationWorkerInvalidConfig              = 130330
	ErrorCodeIntegrationWorkerStartFailed                = 130331
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
	ErrorCodeIntegrationLeaseDurationInvalid             = 130355
	ErrorCodeIntegrationLeaseMarginInsufficient          = 130356
	ErrorCodeIntegrationInterfaceRuntimeIncompatible     = 130357
	ErrorCodeIntegrationExecutionRuntimeIncompatible     = 130358
	ErrorCodeIntegrationExecutionInputStorageTooLarge    = 130359
	ErrorCodeIntegrationExecutionInputSizeMismatch       = 130360
	ErrorCodeIntegrationRetrySnapshotInvalid             = 130361
	ErrorCodeIntegrationRetryPolicyInvalid               = 130362
	ErrorCodeIntegrationRetryScheduleInvalid             = 130363
	ErrorCodeIntegrationRetryCancelConflict              = 130369
	ErrorCodeIntegrationRetryAttemptCreateFailed         = 130370
	ErrorCodeIntegrationRetryExecutionCompleteFailed     = 130371
)

var (
	ErrIntegrationExecutionNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeIntegrationExecutionNotFound,
		"集成执行不存在",
	)
	ErrIntegrationExecutionIdempotencyConflict = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionIdempotencyConflict,
		"幂等键已用于不同的集成执行输入",
	)
	ErrIntegrationExecutionStatusInvalid = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionStatusInvalid,
		"集成执行当前状态不允许执行该操作",
	)
	ErrIntegrationExecutionRevisionConflict = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionRevisionConflict,
		"集成执行已被其他操作修改，请刷新后重试",
	)
	ErrIntegrationExecutionConfigurationInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationExecutionConfigurationInvalid,
		"集成执行配置不存在、未启用或输入不合法",
	)
	ErrIntegrationCredentialNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeIntegrationCredentialNotFound,
		"运行时凭证不存在",
	)
	ErrIntegrationCredentialSystemMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationCredentialSystemMismatch,
		"运行时凭证不属于目标外部系统",
	)
	ErrIntegrationCredentialInterfaceMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationCredentialInterfaceMismatch,
		"运行时凭证与接口定义不匹配",
	)
	ErrIntegrationCredentialInactive = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationCredentialInactive,
		"运行时凭证未启用",
	)
	ErrIntegrationCredentialExpired = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationCredentialExpired,
		"运行时凭证已过期",
	)
	ErrIntegrationCredentialRevoked = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationCredentialRevoked,
		"运行时凭证已吊销",
	)
	ErrIntegrationCredentialTypeUnsupported = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationCredentialTypeUnsupported,
		"运行时凭证类型暂不支持",
	)
	ErrIntegrationCredentialSecretMissing = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationCredentialSecretMissing,
		"运行时凭证秘密材料不完整",
	)
	ErrIntegrationCredentialDecryptFailed = newApplicationError(KindInternal, CategorySystem,
		ErrorCodeIntegrationCredentialDecryptFailed,
		"运行时凭证秘密无法安全解析",
	)
	ErrIntegrationCredentialMaterialInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationCredentialMaterialInvalid,
		"运行时凭证材料不合法",
	)
	ErrIntegrationCredentialInjectionInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationCredentialInjectionInvalid,
		"运行时凭证认证注入不合法",
	)
	ErrIntegrationExecutionClaimConflict = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionClaimConflict,
		"集成执行已被其他 Worker 领取",
	)
	ErrIntegrationExecutionLeaseLost = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionLeaseLost,
		"集成执行租约已失效",
	)
	ErrIntegrationAttemptCreateFailed = newApplicationError(KindInternal, CategorySystem,
		ErrorCodeIntegrationAttemptCreateFailed,
		"集成执行尝试创建失败",
	)
	ErrIntegrationAttemptAlreadyCompleted = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationAttemptAlreadyCompleted,
		"集成执行尝试已完成",
	)
	ErrIntegrationConfigurationUnavailable = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationConfigurationUnavailable,
		"集成运行配置不可用",
	)
	ErrIntegrationExecutionCompleteFailed = newApplicationError(KindInternal, CategorySystem,
		ErrorCodeIntegrationExecutionCompleteFailed,
		"集成执行结果保存失败",
	)
	ErrIntegrationExecutionResultUnknown = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionResultUnknown,
		"集成远程调用结果无法确认",
	)
	ErrIntegrationLeaseRecoveryFailed = newApplicationError(KindInternal, CategorySystem,
		ErrorCodeIntegrationLeaseRecoveryFailed,
		"集成执行租约恢复失败",
	)
	ErrIntegrationWorkerAlreadyRunning = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationWorkerAlreadyRunning,
		"集成常驻 Worker 已在运行",
	)
	ErrIntegrationWorkerInvalidConfig = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationWorkerInvalidConfig,
		"集成常驻 Worker 配置不合法",
	)
	ErrIntegrationWorkerStartFailed = newApplicationError(KindInternal, CategorySystem,
		ErrorCodeIntegrationWorkerStartFailed,
		"集成常驻 Worker 启动失败",
	)
	ErrIntegrationWorkerShutdownTimeout = newApplicationError(KindTimeout, CategorySystem,
		ErrorCodeIntegrationWorkerShutdownTimeout,
		"集成常驻 Worker 优雅关闭超时",
	)
	ErrIntegrationWorkerPanicRecovered = newApplicationError(KindInternal, CategorySystem,
		ErrorCodeIntegrationWorkerPanicRecovered,
		"集成常驻 Worker 执行异常已恢复",
	)
	ErrIntegrationExecutionInputMissing = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionInputMissing,
		"集成执行输入快照缺失",
	)
	ErrIntegrationExecutionInputInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationExecutionInputInvalid,
		"集成执行输入不合法",
	)
	ErrIntegrationExecutionInputSemanticTooLarge = newApplicationError(KindPayloadTooLarge, CategoryBusiness,
		ErrorCodeIntegrationExecutionInputSemanticTooLarge,
		"集成执行输入语义大小超过平台限制",
	)
	ErrIntegrationExecutionInputContractMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionInputContractMismatch,
		"集成执行输入与接口契约不匹配",
	)
	ErrIntegrationExecutionInputHashMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionInputHashMismatch,
		"集成执行输入完整性校验失败",
	)
	ErrIntegrationExecutionInputVersionUnsupported = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionInputVersionUnsupported,
		"集成执行输入快照版本不受支持",
	)
	ErrIntegrationExecutionPathParameterMissing = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationExecutionPathParameterMissing,
		"集成执行缺少必填 Path 参数",
	)
	ErrIntegrationExecutionPathParameterUnknown = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationExecutionPathParameterUnknown,
		"集成执行包含未声明的 Path 参数",
	)
	ErrIntegrationExecutionQueryParameterInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationExecutionQueryParameterInvalid,
		"集成执行 Query 参数不合法",
	)
	ErrIntegrationExecutionHeaderNotAllowed = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationExecutionHeaderNotAllowed,
		"集成执行 Header 不在允许范围内",
	)
	ErrIntegrationExecutionBodyInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationExecutionBodyInvalid,
		"集成执行 JSON Body 不合法",
	)
	ErrIntegrationExecutionSensitiveInputRejected = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationExecutionSensitiveInputRejected,
		"集成执行不允许保存敏感输入",
	)
	ErrIntegrationExecutionInputStorageFailed = newApplicationError(KindInternal, CategorySystem,
		ErrorCodeIntegrationExecutionInputStorageFailed,
		"集成执行输入快照保存失败",
	)
	ErrIntegrationExecutionInputLoadFailed = newApplicationError(KindInternal, CategorySystem,
		ErrorCodeIntegrationExecutionInputLoadFailed,
		"集成执行输入快照加载失败",
	)
	ErrIntegrationTimeoutOutOfRange = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationTimeoutOutOfRange,
		"接口请求超时超出平台运行范围",
	)
	ErrIntegrationResponseLimitOutOfRange = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationResponseLimitOutOfRange,
		"接口响应大小限制超出平台运行范围",
	)
	ErrIntegrationLeaseDurationInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationLeaseDurationInvalid,
		"集成执行租约时长不合法",
	)
	ErrIntegrationLeaseMarginInsufficient = newApplicationError(KindInvalidArgument, CategoryBusiness,
		ErrorCodeIntegrationLeaseMarginInsufficient,
		"集成执行租约安全余量不足",
	)
	ErrIntegrationInterfaceRuntimeIncompatible = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationInterfaceRuntimeIncompatible,
		"接口定义与当前集成运行契约不兼容",
	)
	ErrIntegrationExecutionRuntimeIncompatible = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionRuntimeIncompatible,
		"集成执行引用的接口不符合当前运行契约",
	)
	ErrIntegrationExecutionInputStorageTooLarge = newApplicationError(KindPayloadTooLarge, CategoryBusiness,
		ErrorCodeIntegrationExecutionInputStorageTooLarge,
		"集成执行输入存储大小超过平台限制",
	)
	ErrIntegrationExecutionInputSizeMismatch = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationExecutionInputSizeMismatch,
		"集成执行输入大小完整性校验失败",
	)
	ErrIntegrationRetrySnapshotInvalid = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationRetrySnapshotInvalid,
		"集成重试策略快照无效",
	)
	ErrIntegrationRetryPolicyInvalid = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationRetryPolicyInvalid,
		"集成重试策略无效",
	)
	ErrIntegrationRetryScheduleInvalid = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationRetryScheduleInvalid,
		"集成重试调度参数无效",
	)
	ErrIntegrationRetryCancelConflict = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeIntegrationRetryCancelConflict,
		"集成重试任务已被领取，无法取消",
	)
	ErrIntegrationRetryAttemptCreateFailed = newApplicationError(KindInternal, CategoryBusiness,
		ErrorCodeIntegrationRetryAttemptCreateFailed,
		"集成重试调用记录创建失败",
	)
	ErrIntegrationRetryExecutionCompleteFailed = newApplicationError(KindInternal, CategoryBusiness,
		ErrorCodeIntegrationRetryExecutionCompleteFailed,
		"集成重试执行结果收敛失败",
	)
)
