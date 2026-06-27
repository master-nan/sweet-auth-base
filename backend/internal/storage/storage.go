/**
 * @Author: Nan
 * @Date: 2026/2/17
 */

package storage

import "io"

// Storage 文件存储接口
// 支持本地存储和云存储（OSS/S3等）的统一抽象
type Storage interface {
	// Save 保存文件到存储
	// path: 存储路径（相对路径，如 "2026/02/17/uuid.png"）
	// reader: 文件内容
	// contentType: 文件MIME类型
	// 返回可访问的URL
	Save(path string, reader io.Reader, contentType string) (url string, err error)

	// Delete 删除存储中的文件
	Delete(path string) error

	// Get 获取文件内容
	Get(path string) (io.ReadCloser, error)

	// GetURL 获取文件的访问URL
	GetURL(path string) string

	// Type 返回存储类型标识（"local" / "oss"）
	Type() string
}
