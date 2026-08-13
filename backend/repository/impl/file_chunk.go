/**
 * @Author: Nan
 * @Date: 2026/2/17
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type FileChunkRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.FileChunk]
}

func NewFileChunkRepositoryImpl(primaryDB *database.PrimaryDB) *FileChunkRepositoryImpl {
	return &FileChunkRepositoryImpl{
		primaryDB.DB,
		NewBasicRepositoryImpl(primaryDB.DB, &model.FileChunk{}),
	}
}

func (f *FileChunkRepositoryImpl) FindByUploadId(ctx context.Context, uploadId string) ([]model.FileChunk, error) {
	var chunks []model.FileChunk
	err := f.db.WithContext(ctx).Where("upload_id = ?", uploadId).Order("chunk_index ASC").Find(&chunks).Error
	return chunks, err
}

func (f *FileChunkRepositoryImpl) FindByUploadIdAndIndex(ctx context.Context, uploadId string, index int) (model.FileChunk, error) {
	var chunk model.FileChunk
	err := f.db.WithContext(ctx).Where("upload_id = ? AND chunk_index = ?", uploadId, index).First(&chunk).Error
	return chunk, err
}

func (f *FileChunkRepositoryImpl) CountUploadedChunks(ctx context.Context, uploadId string) (int64, error) {
	var count int64
	err := f.db.WithContext(ctx).Model(&model.FileChunk{}).Where("upload_id = ? AND uploaded = ?", uploadId, true).Count(&count).Error
	return count, err
}

func (f *FileChunkRepositoryImpl) GetFirstChunk(ctx context.Context, uploadId string) (model.FileChunk, error) {
	var chunk model.FileChunk
	err := f.db.WithContext(ctx).Where("upload_id = ?", uploadId).Order("chunk_index ASC").First(&chunk).Error
	return chunk, err
}

func (f *FileChunkRepositoryImpl) FindUnfinishedUpload(ctx context.Context, fileMd5 string, fileSize int64, fileName string, actorID int, allowAll bool) (model.FileChunk, error) {
	var chunk model.FileChunk
	query := f.db.WithContext(ctx).
		Where("file_md5 = ? AND file_size = ? AND file_name = ? AND merged = ? AND uploaded = ?", fileMd5, fileSize, fileName, false, false).
		Order("gmt_create DESC, id DESC")
	if !allowAll {
		query = query.Where("create_user = ?", actorID)
	}
	err := query.First(&chunk).Error
	return chunk, err
}

func (f *FileChunkRepositoryImpl) MarkUploadMerged(ctx context.Context, uploadId, fileMd5 string) error {
	return f.db.WithContext(ctx).Model(&model.FileChunk{}).
		Where("upload_id = ?", uploadId).
		Updates(map[string]interface{}{"merged": true, "file_md5": fileMd5}).
		Error
}
