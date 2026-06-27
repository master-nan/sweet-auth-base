/**
 * @Author: Nan
 * @Date: 2026/2/17
 */

package request

// ChunkUploadInitReq 初始化分片上传请求
type ChunkUploadInitReq struct {
	FileName string `json:"file_name" binding:"required"` // 文件名
	FileSize int64  `json:"file_size" binding:"required"` // 文件总大小（字节）
	FileMd5  string `json:"file_md5"`                     // 完整文件MD5（用于秒传检测）
	FileType string `json:"file_type"`                    // 文件MIME类型
}

// ChunkUploadInitRes 初始化分片上传响应
type ChunkUploadInitRes struct {
	UploadId   string `json:"upload_id"`            // 上传ID（秒传时为空）
	FileId     int    `json:"file_id,omitempty"`    // 秒传时返回已有文件ID
	ChunkSize  int64  `json:"chunk_size,omitempty"` // 分片大小（字节）
	ChunkCount int    `json:"chunk_count"`          // 总分片数（秒传时为0）
	FastUpload bool   `json:"fast_upload"`          // 是否秒传
}

// ChunkUploadProgressRes 分片上传进度响应
type ChunkUploadProgressRes struct {
	UploadId        string `json:"upload_id"`
	ChunkCount      int    `json:"chunk_count"`
	UploadedCount   int    `json:"uploaded_count"`
	UploadedIndexes []int  `json:"uploaded_indexes"` // 已上传的分片索引列表（用于断点续传）
}
