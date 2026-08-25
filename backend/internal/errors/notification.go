package errors

const (
	ErrorCodeNotificationInvalidCategory   = 160001
	ErrorCodeNotificationInvalidLevel      = 160002
	ErrorCodeNotificationInvalidRecipients = 160003
	ErrorCodeNotificationInvalidAction     = 160004
	ErrorCodeNotificationNotVisible        = 160005
	ErrorCodeNotificationPayloadTooLarge   = 160006
	ErrorCodeNotificationDedupConflict     = 160007
	ErrorCodeNotificationRecipientLimit    = 160008
)

var (
	ErrNotificationInvalidCategory   = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeNotificationInvalidCategory, "通知分类不合法")
	ErrNotificationInvalidLevel      = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeNotificationInvalidLevel, "通知级别不合法")
	ErrNotificationInvalidRecipients = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeNotificationInvalidRecipients, "通知收件人不合法")
	ErrNotificationInvalidAction     = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeNotificationInvalidAction, "通知跳转目标不合法")
	ErrNotificationNotVisible        = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeNotificationNotVisible, "通知不存在")
	ErrNotificationPayloadTooLarge   = newApplicationError(KindPayloadTooLarge, CategoryBusiness, ErrorCodeNotificationPayloadTooLarge, "通知内容过大")
	ErrNotificationDedupConflict     = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeNotificationDedupConflict, "通知幂等身份与既有消息事实冲突")
	ErrNotificationRecipientLimit    = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeNotificationRecipientLimit, "通知收件人数超出允许范围")
)
