package errors

const (
	ErrorCodeFileNotFound               = 70001
	ErrorCodeFileEmpty                  = 70002
	ErrorCodeFileExtEmpty               = 70003
	ErrorCodeChunkNotFound              = 70004
	ErrorCodeUploadNotFound             = 70005
	ErrorCodeChunkEmpty                 = 70006
	ErrorCodeChunkOversize              = 70007
	ErrorCodeMergedFileSizeMismatch     = 70008
	ErrorCodeMergedFileMD5Invalid       = 70009
	ErrorCodeFileAccessSignatureInvalid = 70010
	ErrorCodeFileAccessSignatureExpired = 70011
	ErrorCodeFileAccessPurposeMismatch  = 70012
	ErrorCodeFileAccessPurposeMissing   = 70013
)

var (
	ErrFileNotFound               = newApplicationError(KindNotFound, CategoryBusiness, ErrorCodeFileNotFound, "文件不存在")
	ErrFileEmpty                  = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeFileEmpty, "文件不能为空")
	ErrFileExtEmpty               = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeFileExtEmpty, "文件扩展名不能为空")
	ErrChunkNotFound              = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeChunkNotFound, "分片记录不存在")
	ErrUploadNotFound             = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeUploadNotFound, "上传记录不存在")
	ErrChunkEmpty                 = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeChunkEmpty, "分片不能为空")
	ErrChunkOversize              = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeChunkOversize, "分片大小超过初始化范围")
	ErrMergedFileSizeMismatch     = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeMergedFileSizeMismatch, "合并文件大小与初始化信息不一致")
	ErrMergedFileMD5Invalid       = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeMergedFileMD5Invalid, "文件MD5校验失败")
	ErrFileAccessSignatureInvalid = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeFileAccessSignatureInvalid, "文件访问签名无效")
	ErrFileAccessSignatureExpired = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeFileAccessSignatureExpired, "文件访问签名已过期")
	ErrFileAccessPurposeMismatch  = newApplicationError(KindForbidden, CategoryPermission, ErrorCodeFileAccessPurposeMismatch, "文件访问用途不匹配")
	ErrFileAccessPurposeMissing   = newApplicationError(KindUnauthenticated, CategoryPermission, ErrorCodeFileAccessPurposeMissing, "文件访问签名缺少用途")
)
