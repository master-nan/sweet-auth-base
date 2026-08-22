package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LocalChunkStaging 只管理现有上传协议的受控分片暂存目录，
// 不得访问持久文件路径，也不是第二套持久Storage实现。
type LocalChunkStaging struct {
	baseDir string
	mu      sync.RWMutex
}

type MergedChunk struct {
	Path string
	Size int64
	MD5  string
}

func NewLocalChunkStaging(baseDir string) *LocalChunkStaging {
	return &LocalChunkStaging{baseDir: baseDir}
}

func (s *LocalChunkStaging) Write(uploadID string, index int, reader io.Reader) (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 {
		return "", "", fmt.Errorf("invalid chunk index")
	}
	relative := filepath.Join("chunks", uploadID, strconv.Itoa(index))
	fullPath, err := s.safeFullPath(relative)
	if err != nil {
		return "", "", err
	}
	if err = os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", "", err
	}
	destination, err := os.Create(fullPath)
	if err != nil {
		return "", "", err
	}
	hash := md5.New()
	_, copyErr := io.Copy(io.MultiWriter(destination, hash), reader)
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(fullPath)
		if copyErr != nil {
			return "", "", copyErr
		}
		return "", "", closeErr
	}
	return relative, fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (s *LocalChunkStaging) Merge(uploadID, suffix string, chunkPaths []string) (MergedChunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	relative := filepath.Join("chunks", uploadID, "merged-"+suffix)
	fullPath, err := s.safeFullPath(relative)
	if err != nil {
		return MergedChunk{}, err
	}
	if err = os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return MergedChunk{}, err
	}
	destination, err := os.Create(fullPath)
	if err != nil {
		return MergedChunk{}, err
	}
	hash := md5.New()
	writer := io.MultiWriter(destination, hash)
	var size int64
	for _, chunkPath := range chunkPaths {
		chunk, openErr := s.open(chunkPath)
		if openErr != nil {
			_ = destination.Close()
			_ = os.Remove(fullPath)
			return MergedChunk{}, openErr
		}
		written, copyErr := io.Copy(writer, chunk)
		_ = chunk.Close()
		if copyErr != nil {
			_ = destination.Close()
			_ = os.Remove(fullPath)
			return MergedChunk{}, copyErr
		}
		size += written
	}
	if err = destination.Close(); err != nil {
		_ = os.Remove(fullPath)
		return MergedChunk{}, err
	}
	return MergedChunk{Path: relative, Size: size, MD5: fmt.Sprintf("%x", hash.Sum(nil))}, nil
}

func (s *LocalChunkStaging) Open(relative string) (*os.File, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.open(relative)
}

func (s *LocalChunkStaging) open(relative string) (*os.File, error) {
	fullPath, err := s.safeFullPath(relative)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

func (s *LocalChunkStaging) Remove(relative string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fullPath, err := s.safeFullPath(relative)
	if err != nil {
		return err
	}
	if err = os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *LocalChunkStaging) Cleanup(uploadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fullPath, err := s.safeFullPath(filepath.Join("chunks", uploadID))
	if err != nil {
		return err
	}
	return os.RemoveAll(fullPath)
}

// CleanupExpired 只删除受控分片根目录下已过期的Upload Session，
// 根目录外的持久文件不属于清理范围。
func (s *LocalChunkStaging) CleanupExpired(now time.Time, ttl time.Duration) (int, error) {
	if ttl <= 0 {
		return 0, fmt.Errorf("chunk staging TTL must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	root, err := s.safeFullPath("chunks")
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	cutoff := now.Add(-ttl)
	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		directory, pathErr := s.safeFullPath(filepath.Join("chunks", entry.Name()))
		if pathErr != nil {
			return removed, pathErr
		}
		latest, activityErr := latestStagingActivity(directory)
		if activityErr != nil {
			return removed, activityErr
		}
		if !latest.Before(cutoff) {
			continue
		}
		if err := os.RemoveAll(directory); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func latestStagingActivity(root string) (time.Time, error) {
	latest := time.Time{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest, err
}

func (s *LocalChunkStaging) safeFullPath(relative string) (string, error) {
	if strings.TrimSpace(s.baseDir) == "" {
		return "", fmt.Errorf("upload directory is not configured")
	}
	cleaned := filepath.Clean(strings.TrimSpace(relative))
	if cleaned == "." || filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid staging path")
	}
	base, err := filepath.Abs(s.baseDir)
	if err != nil {
		return "", err
	}
	fullPath, err := filepath.Abs(filepath.Join(base, cleaned))
	if err != nil {
		return "", err
	}
	relativeToBase, err := filepath.Rel(base, fullPath)
	if err != nil || relativeToBase == ".." || strings.HasPrefix(relativeToBase, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid staging path")
	}
	return fullPath, nil
}
