/**
 * @Author: Nan
 * @Date: 2025/2/8 10:28
 */

package errors

import (
	"backend/dto/response"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"strconv"
)

const (
	ErrorCodeGeneric      = 10000
	ErrorCodeParamInvalid = 20003
)

func NewError(statusCode, code int, message string) error {
	return newAdminError(inferErrorCategory(statusCode, code), statusCode, code, message, nil)
}

func NewClassifiedError(category response.ErrorCategory, statusCode, code int, message string) error {
	return newAdminError(category, statusCode, code, message, nil)
}

func WrapError(cause error, category response.ErrorCategory, statusCode, code int, message string) error {
	return newAdminError(category, statusCode, code, message, cause)
}

func NewParameterError(message string) error {
	return NewClassifiedError(
		response.ErrorCategoryParameter,
		http.StatusBadRequest,
		ErrorCodeParamInvalid,
		message,
	)
}

func WrapParameterError(cause error, message string) error {
	return WrapError(
		cause,
		response.ErrorCategoryParameter,
		http.StatusBadRequest,
		ErrorCodeParamInvalid,
		message,
	)
}

func NewPermissionError(statusCode, code int, message string) error {
	return NewClassifiedError(response.ErrorCategoryPermission, statusCode, code, message)
}

func NewBusinessError(statusCode, code int, message string) error {
	return NewClassifiedError(response.ErrorCategoryBusiness, statusCode, code, message)
}

func WrapBusinessError(cause error, statusCode, code int, message string) error {
	return WrapError(cause, response.ErrorCategoryBusiness, statusCode, code, message)
}

func WrapDatabaseError(cause error) error {
	return WrapError(
		cause,
		response.ErrorCategoryDatabase,
		http.StatusInternalServerError,
		ErrorCodeGeneric,
		"系统异常",
	)
}

func WrapSystemError(cause error) error {
	return WrapError(
		cause,
		response.ErrorCategorySystem,
		http.StatusInternalServerError,
		ErrorCodeGeneric,
		"系统异常",
	)
}

func WrapSmsSendFailed(cause error) error {
	return WrapError(cause, response.ErrorCategorySystem, http.StatusBadGateway, 50005, "短信发送失败")
}

func WrapSmsStatusQueryFailed(cause error) error {
	return WrapError(cause, response.ErrorCategorySystem, http.StatusBadGateway, 50010, "短信状态查询失败")
}

func WrapDingTalkRequestFailed(cause error) error {
	return WrapError(cause, response.ErrorCategorySystem, http.StatusBadGateway, 60004, "钉钉服务请求失败")
}

func CategoryOf(err error) response.ErrorCategory {
	var adminErr *response.AdminError
	if stderrors.As(err, &adminErr) {
		if adminErr.Category != "" {
			return adminErr.Category
		}
		return inferErrorCategory(adminErr.StatusCode, adminErr.ErrorCode)
	}
	if isRawParameterError(err) {
		return response.ErrorCategoryParameter
	}
	return response.ErrorCategorySystem
}

// ToClientError 将任意错误转换为稳定的 AdminError 响应契约。
// 布尔值表示该错误是否已被明确分类或识别。
func ToClientError(err error) (*response.AdminError, bool) {
	if err == nil {
		return nil, false
	}

	var adminErr *response.AdminError
	if stderrors.As(err, &adminErr) {
		clientErr := adminErr.ForClient()
		if clientErr.Category == "" {
			clientErr.Category = inferErrorCategory(clientErr.StatusCode, clientErr.ErrorCode)
		}
		return clientErr, true
	}
	if isRawParameterError(err) {
		parameterErr := NewParameterError("参数错误")
		if stderrors.As(parameterErr, &adminErr) {
			return adminErr.ForClient(), true
		}
	}

	internalErr := ErrInternalServer
	if stderrors.As(internalErr, &adminErr) {
		return adminErr.ForClient(), false
	}
	return &response.AdminError{
		StatusCode:   http.StatusInternalServerError,
		ErrorCode:    ErrorCodeGeneric,
		ErrorMessage: "系统异常",
		Success:      false,
		Category:     response.ErrorCategorySystem,
	}, false
}

func newAdminError(
	category response.ErrorCategory,
	statusCode int,
	code int,
	message string,
	cause error,
) *response.AdminError {
	return &response.AdminError{
		StatusCode:   statusCode,
		ErrorCode:    code,
		ErrorMessage: message,
		Success:      false,
		Category:     category,
		Cause:        cause,
	}
}

func NewBadRequestError(msg string) error {
	return NewBusinessError(http.StatusBadRequest, ErrorCodeGeneric, msg)
}

func inferErrorCategory(statusCode, code int) response.ErrorCategory {
	switch {
	case code == ErrorCodeParamInvalid:
		return response.ErrorCategoryParameter
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return response.ErrorCategoryPermission
	case statusCode >= http.StatusInternalServerError:
		return response.ErrorCategorySystem
	default:
		return response.ErrorCategoryBusiness
	}
}

func isRawParameterError(err error) bool {
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

var (
	ErrInternalServer = NewError(http.StatusInternalServerError, 10000, "系统异常")

	ErrUserNotFound             = NewError(http.StatusBadRequest, 20001, "用户名或密码错误")
	ErrUserNotLogin             = NewError(http.StatusUnauthorized, 20002, "用户未登录")
	ErrParamInvalid             = NewError(http.StatusBadRequest, 20003, "参数错误")
	ErrCaptchaInvalid           = NewError(http.StatusBadRequest, 20004, "验证码错误")
	ErrUserExist                = NewError(http.StatusBadRequest, 20005, "用户已存在")
	ErrUserNotExist             = NewError(http.StatusBadRequest, 20006, "用户不存在")
	ErrPasswordEmpty            = NewError(http.StatusBadRequest, 20007, "密码不能为空")
	ErrPasswordInvalid          = NewError(http.StatusBadRequest, 20008, "密码不能包含空白字符")
	ErrPasswordTooShort         = NewError(http.StatusBadRequest, 20009, "密码长度不足")
	ErrPasswordTooSimple        = NewError(http.StatusBadRequest, 20010, "密码复杂度不足：至少包含字母和数字")
	ErrPasswordNotComplexEnough = NewError(http.StatusBadRequest, 20011, "密码复杂度不足：至少包含三类字符（大写/小写/数字/特殊字符）")
	ErrDictCodeExist            = NewError(http.StatusBadRequest, 20012, "存在重复的dictCode")
	ErrDataNotFound             = NewError(http.StatusBadRequest, 20013, "操作的数据不存在")
	ErrLoginLocked              = NewError(http.StatusTooManyRequests, 20014, "登录失败次数过多，请稍后再试")
	ErrAuthenticationFailed     = NewError(http.StatusUnauthorized, 20015, "认证失败")
	ErrPasswordChangeRequired   = NewError(http.StatusForbidden, 20016, "请先修改密码")

	ErrInvalidRefreshToken = NewError(http.StatusUnauthorized, 30001, "无效的刷新token")
	ErrTokenExpired        = NewError(http.StatusUnauthorized, 30002, "token已过期")
	ErrTokenInvalid        = NewError(http.StatusUnauthorized, 30003, "无效的token")
	ErrTokenNotActive      = NewError(http.StatusUnauthorized, 30004, "token未激活")
	ErrTokenInvalidType    = NewError(http.StatusUnauthorized, 30005, "token类型错误")
	ErrPermissionDenied    = NewError(http.StatusForbidden, 30006, "无权限访问")

	ErrAppNotFound     = NewError(http.StatusUnauthorized, 40001, "应用不存在")
	ErrAppUnauthorized = NewError(http.StatusUnauthorized, 40002, "应用未授权")
	ErrAppExpired      = NewError(http.StatusUnauthorized, 40003, "应用已过期")
	ErrAppTokenInvalid = NewError(http.StatusUnauthorized, 40004, "应用token无效")
	ErrAppNameExist    = NewBusinessError(http.StatusUnauthorized, 40005, "应用名称已存在")

	ErrClientNotFound       = NewError(http.StatusBadRequest, 50001, "客户端不存在")
	ErrSmsTemplateNotFound  = NewError(http.StatusBadRequest, 50002, "短信模板不存在")
	ErrSmsMobileNotFound    = NewError(http.StatusBadRequest, 50003, "手机号不存在")
	ErrSmsFieldInvalid      = NewError(http.StatusBadRequest, 50004, "字段不合法")
	ErrSmsSendFailed        = NewError(http.StatusBadGateway, 50005, "短信发送失败")
	ErrMobileInvalid        = NewError(http.StatusBadRequest, 50006, "手机号不合法")
	ErrCodeInvalid          = NewError(http.StatusBadRequest, 50007, "验证码错误或者已过期")
	ErrSmsTemplateExist     = NewError(http.StatusBadRequest, 50008, "短信模板已存在")
	ErrSmsSendTooFrequent   = NewError(http.StatusTooManyRequests, 50009, "短信发送过于频繁，请稍后再试")
	ErrSmsStatusQueryFailed = NewError(http.StatusBadGateway, 50010, "短信状态查询失败")
	ErrSmsStatusNotFound    = NewError(http.StatusNotFound, 50011, "短信状态不存在")

	ErrDingTalkSecretNotFound = NewError(http.StatusBadRequest, 60001, "钉钉密钥未配置")
	ErrDingTalkMsgTypeInvalid = NewError(http.StatusBadRequest, 60002, "钉钉消息类型错误")
	ErrDingTalkRecipientEmpty = NewError(http.StatusBadRequest, 60003, "userid_list, dept_id_list, to_all_user 不能同时为空")
	ErrDingTalkRequestFailed  = NewError(http.StatusBadGateway, 60004, "钉钉服务请求失败")

	ErrFileNotFound               = NewError(http.StatusNotFound, 70001, "文件不存在")
	ErrFileEmpty                  = NewError(http.StatusBadRequest, 70002, "文件不能为空")
	ErrFileExtEmpty               = NewError(http.StatusBadRequest, 70003, "文件扩展名不能为空")
	ErrChunkNotFound              = NewError(http.StatusBadRequest, 70004, "分片记录不存在")
	ErrUploadNotFound             = NewError(http.StatusBadRequest, 70005, "上传记录不存在")
	ErrChunkEmpty                 = NewError(http.StatusBadRequest, 70006, "分片不能为空")
	ErrChunkOversize              = NewError(http.StatusBadRequest, 70007, "分片大小超过初始化范围")
	ErrMergedFileSizeMismatch     = NewError(http.StatusBadRequest, 70008, "合并文件大小与初始化信息不一致")
	ErrMergedFileMD5Invalid       = NewError(http.StatusBadRequest, 70009, "文件MD5校验失败")
	ErrFileAccessSignatureInvalid = NewError(http.StatusUnauthorized, 70010, "文件访问签名无效")
	ErrFileAccessSignatureExpired = NewError(http.StatusUnauthorized, 70011, "文件访问签名已过期")
	ErrFileAccessPurposeMismatch  = NewError(http.StatusForbidden, 70012, "文件访问用途不匹配")
	ErrFileAccessPurposeMissing   = NewError(http.StatusUnauthorized, 70013, "文件访问签名缺少用途")

	ErrTableExist             = NewError(http.StatusBadRequest, 90001, "表已存在")
	ErrTableFieldExist        = NewError(http.StatusBadRequest, 90002, "字段已存在")
	ErrTableFieldNoChange     = NewError(http.StatusBadRequest, 90003, "字段无变化，无需更新")
	ErrTableInit              = NewError(http.StatusBadRequest, 90004, "表已初始化，请勿重复操作")
	ErrTableViewSQLEmpty      = NewError(http.StatusBadRequest, 90005, "视图类型视图SQL不能为空")
	ErrTableViewFieldNoAdd    = NewError(http.StatusBadRequest, 90006, "视图字段不可新增，请修改视图SQL后同步字段")
	ErrTableNotFound          = NewError(http.StatusBadRequest, 90007, "表不存在，请先初始化表元数据")
	ErrTableViewFieldNoDelete = NewError(http.StatusBadRequest, 90008, "视图字段不可删除，请修改视图SQL后同步字段")
)
