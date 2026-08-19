package errors

const (
	ErrorCodeQuerySchemeScopeNotFound    = 150001
	ErrorCodeQuerySchemeScopeForbidden   = 150002
	ErrorCodeQuerySchemeNotFound         = 150003
	ErrorCodeQuerySchemeOwnerForbidden   = 150004
	ErrorCodeQuerySchemeSharedForbidden  = 150005
	ErrorCodeQuerySchemeRevisionConflict = 150006
	ErrorCodeQuerySchemePayloadInvalid   = 150007
	ErrorCodeQuerySchemePayloadTooLarge  = 150008
	ErrorCodeQuerySchemeBindingInvalid   = 150009
	ErrorCodeQuerySchemeMetadataDegraded = 150010
	ErrorCodeQuerySchemeInvalid          = 150011
	ErrorCodeQuerySchemeNameConflict     = 150012
	ErrorCodeQuerySchemeDefaultConflict  = 150013
	ErrorCodeQuerySchemeRoleInvalid      = 150014
	ErrorCodeQuerySchemeTypeInvalid      = 150015
)

var (
	ErrQuerySchemeScopeNotFound    = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeQuerySchemeScopeNotFound, "查询范围不存在")
	ErrQuerySchemeScopeForbidden   = newApplicationError(KindForbidden, CategoryPermission, ErrorCodeQuerySchemeScopeForbidden, "无权访问该查询范围")
	ErrQuerySchemeNotFound         = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeQuerySchemeNotFound, "查询方案不存在")
	ErrQuerySchemeOwnerForbidden   = newApplicationError(KindForbidden, CategoryPermission, ErrorCodeQuerySchemeOwnerForbidden, "无权操作该个人查询方案")
	ErrQuerySchemeSharedForbidden  = newApplicationError(KindForbidden, CategoryPermission, ErrorCodeQuerySchemeSharedForbidden, "无权管理共享查询方案")
	ErrQuerySchemeRevisionConflict = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeQuerySchemeRevisionConflict, "方案已被其他操作更新，请刷新后重试")
	ErrQuerySchemePayloadInvalid   = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeQuerySchemePayloadInvalid, "查询方案内容不合法")
	ErrQuerySchemePayloadTooLarge  = newApplicationError(KindPayloadTooLarge, CategoryBusiness, ErrorCodeQuerySchemePayloadTooLarge, "查询方案内容过大")
	ErrQuerySchemeBindingInvalid   = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeQuerySchemeBindingInvalid, "查询方案动态条件不合法")
	ErrQuerySchemeMetadataDegraded = newApplicationError(KindUnprocessable, CategoryBusiness, ErrorCodeQuerySchemeMetadataDegraded, "查询方案包含已失效字段")
	ErrQuerySchemeInvalid          = newApplicationError(KindUnprocessable, CategoryBusiness, ErrorCodeQuerySchemeInvalid, "查询方案当前不可用")
	ErrQuerySchemeNameConflict     = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeQuerySchemeNameConflict, "查询方案名称已存在")
	ErrQuerySchemeDefaultConflict  = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeQuerySchemeDefaultConflict, "默认查询方案发生冲突")
	ErrQuerySchemeRoleInvalid      = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeQuerySchemeRoleInvalid, "角色范围不合法")
	ErrQuerySchemeTypeInvalid      = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeQuerySchemeTypeInvalid, "查询方案类型不合法")
)
