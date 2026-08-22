package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"strconv"
)

// Kind 描述稳定的应用失败类别，不与HTTP状态码直接耦合。
type Kind string

const (
	KindInvalidArgument  Kind = "invalid_argument"
	KindUnauthenticated  Kind = "unauthenticated"
	KindForbidden        Kind = "forbidden"
	KindNotFound         Kind = "not_found"
	KindConflict         Kind = "conflict"
	KindUnprocessable    Kind = "unprocessable"
	KindPayloadTooLarge  Kind = "payload_too_large"
	KindRateLimited      Kind = "rate_limited"
	KindDependencyFailed Kind = "dependency_failed"
	KindUnavailable      Kind = "unavailable"
	KindTimeout          Kind = "timeout"
	KindInternal         Kind = "internal"
)

// Category 区分应用失败所属边界，不暴露内部技术原因。
type Category string

const (
	CategoryParameter  Category = "parameter"
	CategoryPermission Category = "permission"
	CategoryBusiness   Category = "business"
	CategoryDatabase   Category = "database"
	CategorySystem     Category = "system"
)

// ApplicationError 是跨应用边界传递的稳定错误；Cause保持私有，仅用于内部诊断。
type ApplicationError struct {
	Kind        Kind
	Code        int
	SafeMessage string
	Category    Category
	cause       error
}

func (e *ApplicationError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ErrorCode: %d, ErrorMessage: %s", e.Code, e.SafeMessage)
}

func (e *ApplicationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newApplicationError(kind Kind, category Category, code int, message string) error {
	return &ApplicationError{Kind: kind, Category: category, Code: code, SafeMessage: message}
}

// WrapApplicationError 在应用边界把技术原因转换为稳定错误。
func WrapApplicationError(cause error, kind Kind, category Category, code int, message string) error {
	return &ApplicationError{Kind: kind, Category: category, Code: code, SafeMessage: message, cause: cause}
}

func NewParameterError(message string) error {
	return newApplicationError(KindInvalidArgument, CategoryParameter, ErrorCodeParamInvalid, message)
}

func WrapParameterError(cause error, message string) error {
	return WrapApplicationError(cause, KindInvalidArgument, CategoryParameter, ErrorCodeParamInvalid, message)
}

func WrapDatabaseError(cause error) error {
	return WrapApplicationError(cause, KindInternal, CategoryDatabase, ErrorCodeGeneric, "系统异常")
}

func WrapSystemError(cause error) error {
	return WrapApplicationError(cause, KindInternal, CategorySystem, ErrorCodeGeneric, "系统异常")
}

func WrapSmsSendFailed(cause error) error {
	return WrapApplicationError(cause, KindDependencyFailed, CategorySystem, ErrorCodeSmsSendFailed, "短信发送失败")
}

func WrapSmsStatusQueryFailed(cause error) error {
	return WrapApplicationError(cause, KindDependencyFailed, CategorySystem, ErrorCodeSmsStatusQueryFailed, "短信状态查询失败")
}

func WrapDingTalkRequestFailed(cause error) error {
	return WrapApplicationError(cause, KindDependencyFailed, CategorySystem, ErrorCodeDingTalkRequestFailed, "钉钉服务请求失败")
}

func NewValidationError(message string) error {
	return newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeGeneric, message)
}

func AsApplicationError(err error) (*ApplicationError, bool) {
	var applicationErr *ApplicationError
	if !stderrors.As(err, &applicationErr) {
		return nil, false
	}
	return applicationErr, true
}

// Classify 返回稳定Application Error；未知技术失败统一转换为内部错误且不泄漏Cause。
func Classify(err error) (*ApplicationError, bool) {
	if err == nil {
		return nil, false
	}
	if applicationErr, ok := AsApplicationError(err); ok {
		return applicationErr, true
	}
	if IsParameterParsingError(err) {
		applicationErr, _ := AsApplicationError(NewParameterError("参数错误"))
		return applicationErr, true
	}
	applicationErr, _ := AsApplicationError(ErrInternalServer)
	return applicationErr, false
}

func CategoryOf(err error) Category {
	if applicationErr, ok := AsApplicationError(err); ok {
		return applicationErr.Category
	}
	if IsParameterParsingError(err) {
		return CategoryParameter
	}
	return CategorySystem
}

func KindOf(err error) Kind {
	if applicationErr, ok := AsApplicationError(err); ok {
		return applicationErr.Kind
	}
	if IsParameterParsingError(err) {
		return KindInvalidArgument
	}
	return KindInternal
}

func SafeMessageOf(err error) string {
	applicationErr, _ := Classify(err)
	if applicationErr == nil {
		return ""
	}
	return applicationErr.SafeMessage
}

// IsParameterParsingError 只识别Adapter层的参数解码错误。
func IsParameterParsingError(err error) bool {
	if err == nil {
		return false
	}
	if stderrors.Is(err, io.EOF) {
		return true
	}
	var syntaxErr *json.SyntaxError
	if stderrors.As(err, &syntaxErr) {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	if stderrors.As(err, &typeErr) {
		return true
	}
	var numberErr *strconv.NumError
	return stderrors.As(err, &numberErr)
}
