/**
 * @Author: Nan
 * @Date: 2026/2/17
 */

package storage

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSStorage 阿里云 OSS 存储实现
type OSSStorage struct {
	client   *oss.Client
	bucket   *oss.Bucket
	baseURL  string // CDN/自定义域名访问前缀，如 "https://cdn.example.com"
	basePath string // OSS 存储路径前缀，如 "uploads/"
}

// OSSConfig OSS 配置
type OSSConfig struct {
	Endpoint        string // OSS 端点，如 "oss-cn-hangzhou.aliyuncs.com"
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	BaseURL         string // 外部访问URL前缀
	BasePath        string // OSS内路径前缀
}

func NewOSSStorage(cfg OSSConfig) (*OSSStorage, error) {
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("创建OSS客户端失败: %w", err)
	}

	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("获取OSS Bucket失败: %w", err)
	}

	basePath := cfg.BasePath
	if basePath == "" {
		basePath = "uploads/"
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.%s", cfg.BucketName, cfg.Endpoint)
	}

	return &OSSStorage{
		client:   client,
		bucket:   bucket,
		baseURL:  baseURL,
		basePath: basePath,
	}, nil
}

func (o *OSSStorage) objectKey(path string) string {
	return o.basePath + path
}

func (o *OSSStorage) Save(path string, reader io.Reader, contentType string) (string, error) {
	key := o.objectKey(path)
	options := []oss.Option{
		oss.ContentType(contentType),
	}

	err := o.bucket.PutObject(key, reader, options...)
	if err != nil {
		return "", fmt.Errorf("上传文件到OSS失败: %w", err)
	}

	url := fmt.Sprintf("%s/%s", o.baseURL, key)
	return url, nil
}

func (o *OSSStorage) Delete(path string) error {
	key := o.objectKey(path)
	err := o.bucket.DeleteObject(key)
	if err != nil {
		return fmt.Errorf("删除OSS文件失败: %w", err)
	}
	return nil
}

func (o *OSSStorage) Get(path string) (io.ReadCloser, error) {
	key := o.objectKey(path)
	body, err := o.bucket.GetObject(key)
	if err != nil {
		return nil, fmt.Errorf("获取OSS文件失败: %w", err)
	}
	return body, nil
}

func (o *OSSStorage) GetURL(path string) string {
	key := o.objectKey(path)
	return fmt.Sprintf("%s/%s", o.baseURL, key)
}

func (o *OSSStorage) Type() string {
	return "oss"
}

// InitiateMultipartUpload 发起OSS分片上传
func (o *OSSStorage) InitiateMultipartUpload(path string, contentType string) (string, error) {
	key := o.objectKey(path)
	options := []oss.Option{
		oss.ContentType(contentType),
	}
	result, err := o.bucket.InitiateMultipartUpload(key, options...)
	if err != nil {
		return "", fmt.Errorf("发起OSS分片上传失败: %w", err)
	}
	return result.UploadID, nil
}

// UploadPart 上传单个分片到OSS
func (o *OSSStorage) UploadPart(path string, uploadID string, partNumber int, data []byte) (oss.UploadPart, error) {
	key := o.objectKey(path)
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   o.bucket.BucketName,
		Key:      key,
		UploadID: uploadID,
	}
	part, err := o.bucket.UploadPart(imur, bytes.NewReader(data), int64(len(data)), partNumber)
	if err != nil {
		return oss.UploadPart{}, fmt.Errorf("上传OSS分片失败: %w", err)
	}
	return part, nil
}

// CompleteMultipartUpload 完成OSS分片上传
func (o *OSSStorage) CompleteMultipartUpload(path string, uploadID string, parts []oss.UploadPart) (string, error) {
	key := o.objectKey(path)
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   o.bucket.BucketName,
		Key:      key,
		UploadID: uploadID,
	}
	_, err := o.bucket.CompleteMultipartUpload(imur, parts)
	if err != nil {
		return "", fmt.Errorf("完成OSS分片上传失败: %w", err)
	}
	url := fmt.Sprintf("%s/%s", o.baseURL, key)
	return url, nil
}

// AbortMultipartUpload 取消OSS分片上传
func (o *OSSStorage) AbortMultipartUpload(path string, uploadID string) error {
	key := o.objectKey(path)
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   o.bucket.BucketName,
		Key:      key,
		UploadID: uploadID,
	}
	return o.bucket.AbortMultipartUpload(imur)
}
