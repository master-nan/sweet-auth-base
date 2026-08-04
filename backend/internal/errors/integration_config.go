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
)
