/**
 * @Author: Nan
 * @Date: 2024/8/5 下午11:55
 */

package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/storage"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FileUploadService 处理普通上传、秒传和分片上传生命周期。
type FileUploadService struct {
	fileRepo      repository.FileRepository
	fileChunkRepo repository.FileChunkRepository
	sf            *utils.Snowflake
	config        *config.Server
	storage       storage.Storage
	staging       *storage.LocalChunkStaging
	locks         *keyedLockPool
}

// NewFileUploadService 创建上传能力服务。
func NewFileUploadService(
	fileRepo repository.FileRepository,
	fileChunkRepo repository.FileChunkRepository,
	sf *utils.Snowflake,
	config *config.Server,
	store storage.Storage,
) *FileUploadService {
	return &FileUploadService{
		fileRepo:      fileRepo,
		fileChunkRepo: fileChunkRepo,
		sf:            sf,
		config:        config,
		storage:       store,
		staging:       storage.NewLocalChunkStaging(config.Upload.Dir),
		locks:         newKeyedLockPool(),
	}
}

// Upload 上传文件（小文件直传）
func (f *FileUploadService) Upload(ctx context.Context, actor FileAccessActor, fileHeader *multipart.FileHeader) (model.File, error) {
	if err := validateUploadSize(fileHeader.Size, f.config.Upload); err != nil {
		return model.File{}, err
	}
	if err := validateUploadExtension(fileHeader.Filename, f.config.Upload); err != nil {
		return model.File{}, err
	}

	src, err := fileHeader.Open()
	if err != nil {
		return model.File{}, myerrors.WrapSystemError(err)
	}
	defer src.Close()

	detectedType, err := detectUploadContentType(src)
	if err != nil {
		return model.File{}, myerrors.WrapSystemError(err)
	}
	if err := validateUploadMimeType(detectedType, f.config.Upload); err != nil {
		return model.File{}, err
	}

	// 计算文件 MD5
	hash := md5.New()
	if _, err := io.Copy(hash, src); err != nil {
		return model.File{}, myerrors.WrapSystemError(err)
	}
	fileMd5 := fmt.Sprintf("%x", hash.Sum(nil))

	// 秒传：检查是否已有相同 MD5 的文件
	existing, err := f.fileRepo.FindByFileMd5(ctx, fileMd5)
	if err == nil && existing.Id != 0 {
		if err := f.ensureReusableFile(actor, existing); err != nil {
			return model.File{}, err
		}
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.File{}, err
	}

	// 重置文件读取位置
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return model.File{}, myerrors.WrapSystemError(err)
	}

	// 生成文件信息
	fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
	fileUuid := uuid.New().String()
	datePath := time.Now().Format("2006/01/02")
	storedName := fileUuid + fileExt
	storagePath := fmt.Sprintf("%s/%s", datePath, storedName)

	// 通过 Storage 接口保存文件
	contentType := normalizedContentType(detectedType)
	if contentType == "" {
		contentType = normalizedContentType(fileHeader.Header.Get("Content-Type"))
	}
	id, err := f.sf.GenerateUniqueID()
	if err != nil {
		return model.File{}, err
	}

	if _, err = f.storage.Save(storagePath, src, contentType); err != nil {
		return model.File{}, myerrors.WrapSystemError(err)
	}

	file := model.File{
		FileName:    fileHeader.Filename,
		FilePath:    storagePath,
		FileType:    contentType,
		FileUrl:     f.fileAccessURL(fileUuid),
		FileSize:    fileHeader.Size,
		FileMd5:     fileMd5,
		FileExt:     fileExt,
		FileUuid:    fileUuid,
		StorageType: f.storage.Type(),
	}
	file.Id = int(id)

	tx := f.fileRepo.DBWithContext(ctx)
	err = f.fileRepo.Create(tx, &file)
	if err != nil {
		if cleanupErr := f.storage.Delete(storagePath); cleanupErr != nil {
			return model.File{}, myerrors.WrapSystemError(errors.Join(err, cleanupErr))
		}
		return model.File{}, err
	}

	return file, nil
}

func (f *FileUploadService) UploadResponse(ctx context.Context, actor FileAccessActor, fileHeader *multipart.FileHeader) (response.FileDetailRes, error) {
	file, err := f.Upload(ctx, actor, fileHeader)
	if err != nil {
		return response.FileDetailRes{}, err
	}
	return fileDetailResponse(file), nil
}

// ─── 分片上传 ───────────────────────────────────

// InitChunkUpload 初始化分片上传
func (f *FileUploadService) InitChunkUpload(ctx context.Context, actor FileAccessActor, req request.ChunkUploadInitReq) (request.ChunkUploadInitRes, error) {
	if err := validateUploadSize(req.FileSize, f.config.Upload); err != nil {
		return request.ChunkUploadInitRes{}, err
	}
	if err := validateUploadExtension(req.FileName, f.config.Upload); err != nil {
		return request.ChunkUploadInitRes{}, err
	}
	if strings.TrimSpace(req.FileType) != "" {
		if err := validateUploadMimeType(req.FileType, f.config.Upload); err != nil {
			return request.ChunkUploadInitRes{}, err
		}
	}

	// 秒传检查
	if req.FileMd5 != "" {
		existing, err := f.fileRepo.FindByFileMd5(ctx, req.FileMd5)
		if err == nil && existing.Id != 0 {
			if err := f.ensureReusableFile(actor, existing); err != nil {
				return request.ChunkUploadInitRes{}, err
			}
			return request.ChunkUploadInitRes{
				UploadId:   "",
				FileId:     existing.Id,
				FastUpload: true,
			}, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return request.ChunkUploadInitRes{}, err
		}
	}

	// 计算分片数
	chunkSize := f.config.Upload.ChunkSize
	if chunkSize <= 0 {
		return request.ChunkUploadInitRes{}, myerrors.NewBadRequestError("上传分片大小配置错误")
	}
	chunkSizeBytes := chunkSize << 20
	chunkCount := int(math.Ceil(float64(req.FileSize) / float64(chunkSizeBytes)))
	if chunkCount == 0 {
		chunkCount = 1
	}

	if strings.TrimSpace(req.FileMd5) != "" {
		chunk, err := f.fileChunkRepo.FindUnfinishedUpload(ctx, req.FileMd5, req.FileSize, req.FileName, actor.UserID, actor.IsSuperAdmin)
		if err == nil && chunk.UploadId != "" {
			return request.ChunkUploadInitRes{
				UploadId:   chunk.UploadId,
				ChunkSize:  chunk.ChunkSize,
				ChunkCount: chunk.ChunkCount,
				FastUpload: false,
			}, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return request.ChunkUploadInitRes{}, err
		}
	}

	uploadId := uuid.New().String()

	chunks := make([]model.FileChunk, 0, chunkCount)
	for i := 0; i < chunkCount; i++ {
		id, err := f.sf.GenerateUniqueID()
		if err != nil {
			return request.ChunkUploadInitRes{}, err
		}
		chunk := model.FileChunk{
			UploadId:   uploadId,
			FileName:   req.FileName,
			FileSize:   req.FileSize,
			ChunkSize:  chunkSizeBytes,
			ChunkCount: chunkCount,
			ChunkIndex: i,
			FileMd5:    req.FileMd5,
			FileType:   req.FileType,
			FileExt:    strings.ToLower(filepath.Ext(req.FileName)),
			Uploaded:   false,
			Merged:     false,
		}
		chunk.Id = int(id)
		chunks = append(chunks, chunk)
	}
	if err := RunInTransaction(ctx, f.fileChunkRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		return f.fileChunkRepo.Create(tx, &chunks)
	}); err != nil {
		return request.ChunkUploadInitRes{}, err
	}

	return request.ChunkUploadInitRes{
		UploadId:   uploadId,
		ChunkSize:  chunkSizeBytes,
		ChunkCount: chunkCount,
		FastUpload: false,
	}, nil
}

// validateUploadSize 按 upload 配置校验文件大小。
func validateUploadSize(size int64, cfg config.Upload) error {
	if size <= 0 {
		return myerrors.ErrFileEmpty
	}
	maxSizeMB := cfg.MaxSize
	if maxSizeMB <= 0 {
		return myerrors.NewBadRequestError("上传大小配置错误")
	}
	maxSize := maxSizeMB << 20
	if size > maxSize {
		return myerrors.NewBadRequestError(fmt.Sprintf("文件大小不能超过%dMB", maxSizeMB))
	}
	return nil
}

// validateUploadExtension 校验文件扩展名白名单。
func validateUploadExtension(fileName string, cfg config.Upload) error {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	if ext == "" {
		return myerrors.ErrFileExtEmpty
	}
	for _, item := range cfg.AllowedExtensions {
		allowed := strings.ToLower(strings.TrimSpace(item))
		if allowed == "" {
			continue
		}
		if !strings.HasPrefix(allowed, ".") {
			allowed = "." + allowed
		}
		if allowed == ext {
			return nil
		}
	}
	return myerrors.NewBadRequestError(fmt.Sprintf("不允许上传%s类型文件", ext))
}

// validateUploadMimeType 校验上传内容 MIME 白名单。
func validateUploadMimeType(contentType string, cfg config.Upload) error {
	normalized := normalizedContentType(contentType)
	if normalized == "" {
		return nil
	}
	for _, item := range cfg.AllowedMimeTypes {
		if normalizedContentType(item) == normalized {
			return nil
		}
	}
	return myerrors.NewBadRequestError(fmt.Sprintf("不允许上传%s类型文件", normalized))
}

// detectUploadContentType 读取文件头识别 MIME，并把读取位置恢复到开头。
func detectUploadContentType(file multipart.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	return http.DetectContentType(buffer[:n]), nil
}

// normalizedContentType 统一 MIME 格式，去掉 charset 等参数。
func normalizedContentType(contentType string) string {
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

// UploadChunk 上传单个分片
func (f *FileUploadService) UploadChunk(ctx context.Context, actor FileAccessActor, uploadId string, chunkIndex int, fileHeader *multipart.FileHeader) error {
	unlock := f.locks.Lock("chunk:" + uploadId + ":" + fmt.Sprint(chunkIndex))
	defer unlock()

	chunk, err := f.fileChunkRepo.FindByUploadIdAndIndex(ctx, uploadId, chunkIndex)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrChunkNotFound
		}
		return err
	}
	if err := ensureChunkUploadAccess(actor, chunk); err != nil {
		return err
	}

	if chunk.Uploaded {
		return nil // 已上传，幂等
	}
	if err := validateChunkUploadSize(chunk, fileHeader.Size); err != nil {
		return err
	}

	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	chunkPath, chunkMd5, err := f.staging.Write(uploadId, chunkIndex, src)
	if err != nil {
		return myerrors.WrapSystemError(err)
	}

	// 更新分片记录
	updateReq := map[string]interface{}{
		"chunk_path": chunkPath,
		"chunk_md5":  chunkMd5,
		"uploaded":   true,
	}
	return f.fileChunkRepo.Update(f.fileChunkRepo.DBWithContext(ctx), updateReq, chunk.Id)
}

// MergeChunks 合并分片
func (f *FileUploadService) MergeChunks(ctx context.Context, actor FileAccessActor, uploadId string) (model.File, error) {
	unlock := f.locks.Lock("merge:" + uploadId)
	defer unlock()

	firstChunk, err := f.fileChunkRepo.GetFirstChunk(ctx, uploadId)
	if err != nil {
		return model.File{}, myerrors.ErrUploadNotFound
	}
	if err := ensureChunkUploadAccess(actor, firstChunk); err != nil {
		return model.File{}, err
	}
	if firstChunk.Merged {
		existing, findErr := f.fileRepo.FindByFileMd5(ctx, firstChunk.FileMd5)
		if findErr != nil {
			return model.File{}, myerrors.ErrUploadNotFound
		}
		if err := f.ensureReusableFile(actor, existing); err != nil {
			return model.File{}, err
		}
		if err := f.cleanupChunks(uploadId); err != nil {
			return model.File{}, err
		}
		return existing, nil
	}

	// 检查所有分片是否已上传
	uploadedCount, err := f.fileChunkRepo.CountUploadedChunks(ctx, uploadId)
	if err != nil {
		return model.File{}, err
	}
	if int(uploadedCount) < firstChunk.ChunkCount {
		return model.File{}, myerrors.NewBadRequestError(
			fmt.Sprintf("分片未全部上传，已上传 %d/%d", uploadedCount, firstChunk.ChunkCount),
		)
	}

	chunks, err := f.fileChunkRepo.FindByUploadId(ctx, uploadId)
	if err != nil {
		return model.File{}, err
	}

	fileUuid := uuid.New().String()
	fileExt := firstChunk.FileExt
	datePath := time.Now().Format(time.DateOnly)
	storedName := fileUuid + fileExt
	storagePath := fmt.Sprintf("%s-%s", datePath, storedName)

	chunkPaths := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunkPaths = append(chunkPaths, chunk.ChunkPath)
	}
	merged, err := f.staging.Merge(uploadId, fileUuid+fileExt, chunkPaths)
	if err != nil {
		return model.File{}, myerrors.WrapSystemError(err)
	}
	defer f.staging.Remove(merged.Path)
	fileMd5 := merged.MD5
	if err := validateMergedChunkFile(firstChunk, merged.Size, fileMd5); err != nil {
		return model.File{}, err
	}

	mergedForDetection, err := f.staging.Open(merged.Path)
	if err != nil {
		return model.File{}, myerrors.WrapSystemError(err)
	}
	detectedType, detectErr := detectUploadContentType(mergedForDetection)
	_ = mergedForDetection.Close()
	if detectErr != nil {
		return model.File{}, detectErr
	}
	if err := validateUploadMimeType(detectedType, f.config.Upload); err != nil {
		return model.File{}, err
	}
	contentType := normalizedContentType(detectedType)
	if contentType == "" {
		contentType = normalizedContentType(firstChunk.FileType)
	}

	// 秒传检查（合并后再次检查）
	existing, err := f.fileRepo.FindByFileMd5(ctx, fileMd5)
	if err == nil && existing.Id != 0 {
		if err := f.ensureReusableFile(actor, existing); err != nil {
			return model.File{}, err
		}
		if err := f.fileChunkRepo.MarkUploadMerged(ctx, uploadId, fileMd5); err != nil {
			return model.File{}, err
		}
		if err := f.cleanupChunks(uploadId); err != nil {
			return model.File{}, err
		}
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.File{}, err
	}

	// 通过 Storage 接口保存合并后的文件
	mergedFile, err := f.staging.Open(merged.Path)
	if err != nil {
		return model.File{}, err
	}
	defer mergedFile.Close()

	_, err = f.storage.Save(storagePath, mergedFile, contentType)
	if err != nil {
		return model.File{}, myerrors.WrapSystemError(err)
	}

	fileSize := merged.Size

	id, err := f.sf.GenerateUniqueID()
	if err != nil {
		return model.File{}, err
	}

	file := model.File{
		FileName:    firstChunk.FileName,
		FilePath:    storagePath,
		FileType:    contentType,
		FileUrl:     f.fileAccessURL(fileUuid),
		FileSize:    fileSize,
		FileMd5:     fileMd5,
		FileExt:     fileExt,
		FileUuid:    fileUuid,
		StorageType: f.storage.Type(),
	}
	file.Id = int(id)

	tx := f.fileRepo.DBWithContext(ctx)
	err = f.fileRepo.Create(tx, &file)
	if err != nil {
		if cleanupErr := f.storage.Delete(storagePath); cleanupErr != nil {
			return model.File{}, myerrors.WrapSystemError(errors.Join(err, cleanupErr))
		}
		return model.File{}, err
	}

	if err := f.fileChunkRepo.MarkUploadMerged(ctx, uploadId, fileMd5); err != nil {
		return model.File{}, err
	}

	if err := f.cleanupChunks(uploadId); err != nil {
		return model.File{}, err
	}

	return file, nil
}

func (f *FileUploadService) MergeChunksResponse(ctx context.Context, actor FileAccessActor, uploadId string) (response.FileDetailRes, error) {
	file, err := f.MergeChunks(ctx, actor, uploadId)
	if err != nil {
		return response.FileDetailRes{}, err
	}
	return fileDetailResponse(file), nil
}

// GetUploadProgressForUser 获取当前用户可访问的分片上传进度。
func (f *FileUploadService) GetUploadProgressForUser(ctx context.Context, actor FileAccessActor, uploadId string) (request.ChunkUploadProgressRes, error) {
	firstChunk, err := f.fileChunkRepo.GetFirstChunk(ctx, uploadId)
	if err != nil {
		return request.ChunkUploadProgressRes{}, myerrors.ErrUploadNotFound
	}
	if err := ensureChunkUploadAccess(actor, firstChunk); err != nil {
		return request.ChunkUploadProgressRes{}, err
	}
	chunks, err := f.fileChunkRepo.FindByUploadId(ctx, firstChunk.UploadId)
	if err != nil {
		return request.ChunkUploadProgressRes{}, err
	}

	uploadedIndexes := make([]int, 0)
	for _, c := range chunks {
		if c.Uploaded {
			uploadedIndexes = append(uploadedIndexes, c.ChunkIndex)
		}
	}

	return request.ChunkUploadProgressRes{
		UploadId:        firstChunk.UploadId,
		ChunkCount:      firstChunk.ChunkCount,
		UploadedCount:   len(uploadedIndexes),
		UploadedIndexes: uploadedIndexes,
	}, nil
}

// ensureFileReuseAccess 校验秒传复用已有文件时是否为本人文件或超管。
func ensureFileReuseAccess(actor FileAccessActor, file model.File) error {
	if actor.IsSuperAdmin {
		return nil
	}
	if actor.UserID > 0 && file.CreateUser != nil && *file.CreateUser == actor.UserID {
		return nil
	}
	return myerrors.ErrPermissionDenied
}

func (f *FileUploadService) ensureReusableFile(actor FileAccessActor, file model.File) error {
	if err := ensureFileReuseAccess(actor, file); err != nil {
		return err
	}
	reader, err := f.storage.Get(file.FilePath)
	if err != nil {
		return myerrors.ErrFileNotFound
	}
	defer reader.Close()
	hash := md5.New()
	size, err := io.Copy(hash, reader)
	if err != nil {
		return myerrors.WrapSystemError(err)
	}
	if size != file.FileSize || !strings.EqualFold(fmt.Sprintf("%x", hash.Sum(nil)), file.FileMd5) {
		return myerrors.ErrMergedFileMD5Invalid
	}
	return nil
}

func ensureChunkUploadAccess(actor FileAccessActor, chunk model.FileChunk) error {
	if actor.IsSuperAdmin {
		return nil
	}
	if actor.UserID > 0 && chunk.CreateUser != nil && *chunk.CreateUser == actor.UserID {
		return nil
	}
	return myerrors.ErrPermissionDenied
}

// validateChunkUploadSize 校验分片大小，最后一个分片允许小于标准分片大小。
func validateChunkUploadSize(chunk model.FileChunk, size int64) error {
	if size <= 0 {
		return myerrors.ErrChunkEmpty
	}
	expectedMax := chunk.ChunkSize
	if remaining := chunk.FileSize - int64(chunk.ChunkIndex)*chunk.ChunkSize; remaining > 0 && remaining < expectedMax {
		expectedMax = remaining
	}
	if expectedMax > 0 && size > expectedMax {
		return myerrors.ErrChunkOversize
	}
	return nil
}

// validateMergedChunkFile 校验合并后的文件大小和 MD5。
func validateMergedChunkFile(firstChunk model.FileChunk, size int64, fileMd5 string) error {
	if firstChunk.FileSize > 0 && size != firstChunk.FileSize {
		return myerrors.ErrMergedFileSizeMismatch
	}
	if strings.TrimSpace(firstChunk.FileMd5) != "" && !strings.EqualFold(fileMd5, firstChunk.FileMd5) {
		return myerrors.ErrMergedFileMD5Invalid
	}
	return nil
}

// cleanupChunks 清理分片临时目录。
func (f *FileUploadService) cleanupChunks(uploadId string) error {
	if err := f.staging.Cleanup(uploadId); err != nil {
		return myerrors.WrapSystemError(err)
	}
	return nil
}

// fileAccessURL 生成文件公开访问 URL。
func (f *FileUploadService) fileAccessURL(fileUuid string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(f.config.Upload.BaseURL), "/")
	return fmt.Sprintf("%s/%s", baseURL, fileUuid)
}
