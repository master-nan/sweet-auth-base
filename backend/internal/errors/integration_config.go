package errors

import "net/http"

const (
	ErrorCodeExternalSystemNotFound         = 130001
	ErrorCodeExternalSystemCodeDuplicate    = 130002
	ErrorCodeExternalSystemCodeInvalid      = 130003
	ErrorCodeExternalSystemFieldImmutable   = 130004
	ErrorCodeExternalSystemStatusInvalid    = 130005
	ErrorCodeExternalSystemConfiguration    = 130006
	ErrorCodeExternalSystemBaseURLInvalid   = 130007
	ErrorCodeExternalSystemRevisionConflict = 130008
	ErrorCodeExternalSystemReferenced       = 130009
)

var (
	ErrExternalSystemNotFound         = NewBusinessError(http.StatusNotFound, ErrorCodeExternalSystemNotFound, "外部系统不存在")
	ErrExternalSystemCodeDuplicate    = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemCodeDuplicate, "外部系统编码已存在")
	ErrExternalSystemCodeInvalid      = NewBusinessError(http.StatusBadRequest, ErrorCodeExternalSystemCodeInvalid, "外部系统编码格式不合法")
	ErrExternalSystemFieldImmutable   = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemFieldImmutable, "外部系统身份字段不可修改")
	ErrExternalSystemStatusInvalid    = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemStatusInvalid, "外部系统当前状态不允许执行该操作")
	ErrExternalSystemConfiguration    = NewBusinessError(http.StatusBadRequest, ErrorCodeExternalSystemConfiguration, "外部系统必要配置不完整")
	ErrExternalSystemBaseURLInvalid   = NewBusinessError(http.StatusBadRequest, ErrorCodeExternalSystemBaseURLInvalid, "外部系统基础地址不合法")
	ErrExternalSystemRevisionConflict = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemRevisionConflict, "外部系统已被其他操作修改，请刷新后重试")
	ErrExternalSystemReferenced       = NewBusinessError(http.StatusConflict, ErrorCodeExternalSystemReferenced, "外部系统已被配置引用")
)
