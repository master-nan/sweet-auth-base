package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LocalChunkStaging owns temporary chunk paths. It is staging for the existing
// upload protocol, not a second durable Storage implementation.
type LocalChunkStaging struct {
	baseDir string
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
		chunk, openErr := s.Open(chunkPath)
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
	fullPath, err := s.safeFullPath(relative)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

func (s *LocalChunkStaging) Remove(relative string) error {
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
	fullPath, err := s.safeFullPath(filepath.Join("chunks", uploadID))
	if err != nil {
		return err
	}
	return os.RemoveAll(fullPath)
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
