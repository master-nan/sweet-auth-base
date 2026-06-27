/**
 * @Author: Nan
 * @Date: 2026/2/17
 */

package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage 本地文件存储实现
type LocalStorage struct {
	// BaseDir 存储根目录（支持绝对路径和相对路径）
	BaseDir string
	// BaseURL 文件访问URL前缀，如 "/files"
	BaseURL string
}

func NewLocalStorage(baseDir string, baseURL string) *LocalStorage {
	if baseDir == "" {
		baseDir = "./uploads"
	}
	if baseURL == "" {
		baseURL = "/files"
	}
	return &LocalStorage{
		BaseDir: baseDir,
		BaseURL: baseURL,
	}
}

func (l *LocalStorage) Save(path string, reader io.Reader, contentType string) (string, error) {
	fullPath, err := l.safeFullPath(path)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, reader); err != nil {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	url := fmt.Sprintf("%s/%s", l.BaseURL, path)
	return url, nil
}

func (l *LocalStorage) Delete(path string) error {
	fullPath, err := l.safeFullPath(path)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

func (l *LocalStorage) Get(path string) (io.ReadCloser, error) {
	fullPath, err := l.safeFullPath(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	return file, nil
}

func (l *LocalStorage) GetURL(path string) string {
	return fmt.Sprintf("%s/%s", l.BaseURL, path)
}

func (l *LocalStorage) Type() string {
	return "local"
}

// GetFullPath 获取文件的完整本地路径（仅本地存储可用）
func (l *LocalStorage) GetFullPath(path string) string {
	fullPath, err := l.safeFullPath(path)
	if err != nil {
		return ""
	}
	return fullPath
}

func (l *LocalStorage) safeFullPath(path string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "." || cleaned == "" || filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("非法文件路径")
	}
	base, err := filepath.Abs(l.BaseDir)
	if err != nil {
		return "", fmt.Errorf("解析存储目录失败: %w", err)
	}
	fullPath, err := filepath.Abs(filepath.Join(base, cleaned))
	if err != nil {
		return "", fmt.Errorf("解析文件路径失败: %w", err)
	}
	rel, err := filepath.Rel(base, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("非法文件路径")
	}
	return fullPath, nil
}
