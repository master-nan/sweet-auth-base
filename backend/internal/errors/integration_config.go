package errors

import "net/http"

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
	ErrorCodeRetryPolicyNotFound             = 130301
	ErrorCodeRetryPolicyCodeDuplicate        = 130302
	ErrorCodeRetryPolicyCodeInvalid          = 130303
	ErrorCodeRetryPolicyFieldImmutable       = 130304
	ErrorCodeRetryPolicyStatusInvalid        = 130305
	ErrorCodeRetryPolicyConfigurationInvalid = 130306
	ErrorCodeRetryPolicyRevisionConflict     = 130307
	ErrorCodeRetryPolicyEnabledConflict      = 130308
	ErrorCodeRetryPolicyReferenced           = 130309
)

var (
	ErrExternalSystemNotFound          = NewBusinessError(http.StatusNotFound, ErrorCodeExternalSystemNotFound, "外部系统不存在")
	ErrExternalSystemCodeDuplicate     = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemCodeDuplicate, "外部系统编码已存在")
	ErrExternalSystemCodeInvalid       = NewBusinessError(http.StatusBadRequest, ErrorCodeExternalSystemCodeInvalid, "外部系统编码格式不合法")
	ErrExternalSystemFieldImmutable    = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemFieldImmutable, "外部系统身份字段不可修改")
	ErrExternalSystemStatusInvalid     = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemStatusInvalid, "外部系统当前状态不允许执行该操作")
	ErrExternalSystemConfiguration     = NewBusinessError(http.StatusBadRequest, ErrorCodeExternalSystemConfiguration, "外部系统必要配置不完整")
	ErrExternalSystemBaseURLInvalid    = NewBusinessError(http.StatusBadRequest, ErrorCodeExternalSystemBaseURLInvalid, "外部系统基础地址不合法")
	ErrExternalSystemRevisionConflict  = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemRevisionConflict, "外部系统已被其他操作修改，请刷新后重试")
	ErrExternalSystemReferenced        = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemReferenced, "外部系统已被配置引用")
	ErrInterfaceDefinitionNotFound     = NewBusinessError(http.StatusNotFound, ErrorCodeInterfaceDefinitionNotFound, "接口定义不存在")
	ErrInterfaceCodeDuplicate          = NewBusinessError(http.StatusConflict, ErrorCodeInterfaceCodeDuplicate, "接口编码和版本已存在")
	ErrInterfaceCodeInvalid            = NewBusinessError(http.StatusBadRequest, ErrorCodeInterfaceCodeInvalid, "接口编码格式不合法")
	ErrInterfaceFieldImmutable         = NewBusinessError(http.StatusConflict, ErrorCodeInterfaceFieldImmutable, "接口定义身份字段不可修改")
	ErrInterfaceStatusInvalid          = NewBusinessError(http.StatusConflict, ErrorCodeInterfaceStatusInvalid, "接口定义当前状态不允许执行该操作")
	ErrInterfaceConfigurationInvalid   = NewBusinessError(http.StatusBadRequest, ErrorCodeInterfaceConfigurationInvalid, "接口定义必要配置不合法")
	ErrInterfacePathInvalid            = NewBusinessError(http.StatusBadRequest, ErrorCodeInterfacePathInvalid, "接口相对路径不合法")
	ErrInterfaceRevisionConflict       = NewBusinessError(http.StatusConflict, ErrorCodeInterfaceRevisionConflict, "接口定义已被其他操作修改，请刷新后重试")
	ErrInterfaceExternalSystemInvalid  = NewBusinessError(http.StatusConflict, ErrorCodeInterfaceExternalSystemInvalid, "所属外部系统不存在或状态不允许")
	ErrInterfaceCredentialInvalid      = NewBusinessError(http.StatusBadRequest, ErrorCodeInterfaceCredentialInvalid, "接口凭证引用不存在、无效或不属于当前系统")
	ErrInterfaceRetryPolicyInvalid     = NewBusinessError(http.StatusBadRequest, ErrorCodeInterfaceRetryPolicyInvalid, "接口重试策略引用不存在或无效")
	ErrInterfaceEnabledVersionConflict = NewBusinessError(http.StatusConflict, ErrorCodeInterfaceEnabledVersionConflict, "同一接口已有启用版本，请先停用后再启用新版本")
	ErrCredentialNotFound              = NewBusinessError(http.StatusNotFound, ErrorCodeCredentialNotFound, "集成凭证不存在")
	ErrCredentialCodeDuplicate         = NewBusinessError(http.StatusConflict, ErrorCodeCredentialCodeDuplicate, "同一外部系统下凭证编码已存在")
	ErrCredentialCodeInvalid           = NewBusinessError(http.StatusBadRequest, ErrorCodeCredentialCodeInvalid, "凭证编码格式不合法")
	ErrCredentialFieldImmutable        = NewBusinessError(http.StatusConflict, ErrorCodeCredentialFieldImmutable, "凭证身份字段不可修改")
	ErrCredentialTypeInvalid           = NewBusinessError(http.StatusBadRequest, ErrorCodeCredentialTypeInvalid, "凭证类型不受支持")
	ErrCredentialSecretInvalid         = NewBusinessError(http.StatusBadRequest, ErrorCodeCredentialSecretInvalid, "凭证秘密内容不完整或格式不合法")
	ErrCredentialStatusInvalid         = NewBusinessError(http.StatusConflict, ErrorCodeCredentialStatusInvalid, "凭证当前状态不允许执行该操作")
	ErrCredentialRevisionConflict      = NewBusinessError(http.StatusConflict, ErrorCodeCredentialRevisionConflict, "凭证已被其他操作修改，请刷新后重试")
	ErrCredentialExternalSystemInvalid = NewBusinessError(http.StatusConflict, ErrorCodeCredentialExternalSystemInvalid, "所属外部系统不存在或状态不允许")
	ErrCredentialExpired               = NewBusinessError(http.StatusConflict, ErrorCodeCredentialExpired, "凭证已经过期，不能启用")
	ErrCredentialProtectionFailed      = NewBusinessError(http.StatusInternalServerError, ErrorCodeCredentialProtectionFailed, "凭证秘密保护失败")
	ErrRetryPolicyNotFound             = NewBusinessError(http.StatusNotFound, ErrorCodeRetryPolicyNotFound, "重试策略不存在")
	ErrRetryPolicyCodeDuplicate        = NewBusinessError(http.StatusConflict, ErrorCodeRetryPolicyCodeDuplicate, "重试策略编码已存在")
	ErrRetryPolicyCodeInvalid          = NewBusinessError(http.StatusBadRequest, ErrorCodeRetryPolicyCodeInvalid, "重试策略编码格式不合法")
	ErrRetryPolicyFieldImmutable       = NewBusinessError(http.StatusConflict, ErrorCodeRetryPolicyFieldImmutable, "重试策略身份字段不可修改")
	ErrRetryPolicyStatusInvalid        = NewBusinessError(http.StatusConflict, ErrorCodeRetryPolicyStatusInvalid, "重试策略当前状态不允许执行该操作")
	ErrRetryPolicyConfigurationInvalid = NewBusinessError(http.StatusBadRequest, ErrorCodeRetryPolicyConfigurationInvalid, "重试策略参数组合不合法")
	ErrRetryPolicyRevisionConflict     = NewBusinessError(http.StatusConflict, ErrorCodeRetryPolicyRevisionConflict, "重试策略已被其他操作修改，请刷新后重试")
	ErrRetryPolicyEnabledConflict      = NewBusinessError(http.StatusConflict, ErrorCodeRetryPolicyEnabledConflict, "同一策略编码已有启用版本")
	ErrRetryPolicyReferenced           = NewBusinessError(http.StatusConflict, ErrorCodeRetryPolicyReferenced, "重试策略仍被已启用接口引用")
)
