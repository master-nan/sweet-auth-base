package errors

const (
	ErrorCodeUserNotLogin             = 20002
	ErrorCodeCaptchaInvalid           = 20004
	ErrorCodeUserExist                = 20005
	ErrorCodeUserNotExist             = 20006
	ErrorCodePasswordEmpty            = 20007
	ErrorCodePasswordInvalid          = 20008
	ErrorCodePasswordTooShort         = 20009
	ErrorCodePasswordTooSimple        = 20010
	ErrorCodePasswordNotComplexEnough = 20011
	ErrorCodeDictCodeExist            = 20012
	ErrorCodeLoginLocked              = 20014
	ErrorCodeAuthenticationFailed     = 20015
	ErrorCodePasswordChangeRequired   = 20016
	ErrorCodeInvalidRefreshToken      = 30001
	ErrorCodeTokenExpired             = 30002
	ErrorCodeTokenInvalid             = 30003
	ErrorCodeTokenInvalidType         = 30005
	ErrorCodeAppNotFound              = 40001
	ErrorCodeAppUnauthorized          = 40002
	ErrorCodeAppExpired               = 40003
	ErrorCodeAppTokenInvalid          = 40004
	ErrorCodeAppNameExist             = 40005
	ErrorCodeClientNotFound           = 50001
	ErrorCodeSmsTemplateNotFound      = 50002
	ErrorCodeSmsFieldInvalid          = 50004
	ErrorCodeSmsSendFailed            = 50005
	ErrorCodeMobileInvalid            = 50006
	ErrorCodeSmsTemplateExist         = 50008
	ErrorCodeSmsSendTooFrequent       = 50009
	ErrorCodeSmsStatusQueryFailed     = 50010
	ErrorCodeSmsStatusNotFound        = 50011
	ErrorCodeDingTalkSecretNotFound   = 60001
	ErrorCodeDingTalkMsgTypeInvalid   = 60002
	ErrorCodeDingTalkRecipientEmpty   = 60003
	ErrorCodeDingTalkRequestFailed    = 60004
)

var (
	ErrUserNotLogin             = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeUserNotLogin, "用户未登录")
	ErrCaptchaInvalid           = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeCaptchaInvalid, "验证码错误")
	ErrUserExist                = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeUserExist, "用户已存在")
	ErrUserNotExist             = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeUserNotExist, "用户不存在")
	ErrPasswordEmpty            = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodePasswordEmpty, "密码不能为空")
	ErrPasswordInvalid          = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodePasswordInvalid, "密码不能包含空白字符")
	ErrPasswordTooShort         = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodePasswordTooShort, "密码长度不足")
	ErrPasswordTooSimple        = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodePasswordTooSimple, "密码复杂度不足：至少包含字母和数字")
	ErrPasswordNotComplexEnough = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodePasswordNotComplexEnough, "密码复杂度不足：至少包含三类字符（大写/小写/数字/特殊字符）")
	ErrDictCodeExist            = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeDictCodeExist, "存在重复的dictCode")
	ErrLoginLocked              = newApplicationError(KindRateLimited, CategoryBusiness, ErrorCodeLoginLocked, "登录失败次数过多，请稍后再试")
	ErrAuthenticationFailed     = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeAuthenticationFailed, "认证失败")
	ErrPasswordChangeRequired   = newApplicationError(KindForbidden, CategoryPermission, ErrorCodePasswordChangeRequired, "请先修改密码")

	ErrInvalidRefreshToken = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeInvalidRefreshToken, "无效的刷新token")
	ErrTokenExpired        = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeTokenExpired, "token已过期")
	ErrTokenInvalid        = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeTokenInvalid, "无效的token")
	ErrTokenInvalidType    = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeTokenInvalidType, "token类型错误")

	ErrAppNotFound     = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeAppNotFound, "应用不存在")
	ErrAppUnauthorized = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeAppUnauthorized, "应用未授权")
	ErrAppExpired      = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeAppExpired, "应用已过期")
	ErrAppTokenInvalid = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeAppTokenInvalid, "应用token无效")
	ErrAppNameExist    = newApplicationError(KindUnauthenticated, CategoryBusiness, ErrorCodeAppNameExist, "应用名称已存在")

	ErrClientNotFound       = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeClientNotFound, "客户端不存在")
	ErrSmsTemplateNotFound  = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSmsTemplateNotFound, "短信模板不存在")
	ErrSmsFieldInvalid      = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSmsFieldInvalid, "字段不合法")
	ErrSmsSendFailed        = newApplicationError(KindDependencyFailed, CategorySystem, ErrorCodeSmsSendFailed, "短信发送失败")
	ErrMobileInvalid        = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeMobileInvalid, "手机号不合法")
	ErrSmsTemplateExist     = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeSmsTemplateExist, "短信模板已存在")
	ErrSmsSendTooFrequent   = newApplicationError(KindRateLimited, CategoryBusiness, ErrorCodeSmsSendTooFrequent, "短信发送过于频繁，请稍后再试")
	ErrSmsStatusQueryFailed = newApplicationError(KindDependencyFailed, CategorySystem, ErrorCodeSmsStatusQueryFailed, "短信状态查询失败")
	ErrSmsStatusNotFound    = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeSmsStatusNotFound, "短信状态不存在")

	ErrDingTalkSecretNotFound = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeDingTalkSecretNotFound, "钉钉密钥未配置")
	ErrDingTalkMsgTypeInvalid = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeDingTalkMsgTypeInvalid, "钉钉消息类型错误")
	ErrDingTalkRecipientEmpty = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeDingTalkRecipientEmpty, "userid_list, dept_id_list, to_all_user 不能同时为空")
)
