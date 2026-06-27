/**
 * @Author: Nan
 * @Date: 2024/8/5 下午11:47
 */

package model

type File struct {
	Basic
	FileName    string `gorm:"size:256;comment:文件名" json:"file_name"`
	FilePath    string `gorm:"size:512;comment:文件路径(存储相对路径)" json:"file_path"`
	FileType    string `gorm:"size:128;comment:文件类型" json:"file_type"`
	FileUrl     string `gorm:"size:512;comment:文件访问地址" json:"file_url"`
	FileSize    int64  `gorm:"comment:文件大小(字节)" json:"file_size"`
	FileMd5     string `gorm:"size:128;uniqueIndex;comment:文件md5" json:"file_md5"`
	FileExt     string `gorm:"size:32;comment:文件扩展名" json:"file_ext"`
	FileUuid    string `gorm:"size:128;uniqueIndex;comment:文件uuid" json:"file_uuid"`
	StorageType string `gorm:"size:16;default:local;comment:存储类型(local/oss)" json:"storage_type"`
}

// FileChunk 分片上传记录
type FileChunk struct {
	Basic
	UploadId   string `gorm:"size:128;index;comment:上传ID" json:"upload_id"`
	FileName   string `gorm:"size:128;comment:原始文件名" json:"file_name"`
	FileSize   int64  `gorm:"comment:文件总大小" json:"file_size"`
	ChunkSize  int64  `gorm:"comment:分片大小" json:"chunk_size"`
	ChunkCount int    `gorm:"comment:总分片数" json:"chunk_count"`
	ChunkIndex int    `gorm:"comment:当前分片索引(从0开始)" json:"chunk_index"`
	ChunkMd5   string `gorm:"size:128;comment:分片MD5" json:"chunk_md5"`
	ChunkPath  string `gorm:"size:512;comment:分片存储路径" json:"chunk_path"`
	FileMd5    string `gorm:"size:128;comment:完整文件MD5" json:"file_md5"`
	FileType   string `gorm:"size:128;comment:文件MIME类型" json:"file_type"`
	FileExt    string `gorm:"size:128;comment:文件扩展名" json:"file_ext"`
	Uploaded   bool   `gorm:"default:false;comment:分片是否已上传" json:"uploaded"`
	Merged     bool   `gorm:"default:false;comment:是否已合并" json:"merged"`
}
