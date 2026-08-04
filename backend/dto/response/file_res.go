package response

// FileDetailRes 是文件接口对外公开的业务信息，不包含物理路径、摘要和存储实现。
type FileDetailRes struct {
	BasicRes
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	FileUrl  string `json:"file_url"`
	FileSize int64  `json:"file_size"`
	FileExt  string `json:"file_ext"`
	FileUuid string `json:"file_uuid"`
}
