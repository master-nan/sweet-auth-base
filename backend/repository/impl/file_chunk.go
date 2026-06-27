/**
 * @Author: Nan
 * @Date: 2026/2/17
 */

package impl

import (
	"backend/internal/database"
	"backend/model"

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

func (f *FileChunkRepositoryImpl) FindByUploadId(uploadId string) ([]model.FileChunk, error) {
	var chunks []model.FileChunk
	err := f.db.Where("upload_id = ?", uploadId).Order("chunk_index ASC").Find(&chunks).Error
	return chunks, err
}

func (f *FileChunkRepositoryImpl) FindByUploadIdAndIndex(uploadId string, index int) (model.FileChunk, error) {
	var chunk model.FileChunk
	err := f.db.Where("upload_id = ? AND chunk_index = ?", uploadId, index).First(&chunk).Error
	return chunk, err
}

func (f *FileChunkRepositoryImpl) CountUploadedChunks(uploadId string) (int64, error) {
	var count int64
	err := f.db.Model(&model.FileChunk{}).Where("upload_id = ? AND uploaded = ?", uploadId, true).Count(&count).Error
	return count, err
}

func (f *FileChunkRepositoryImpl) GetFirstChunk(uploadId string) (model.FileChunk, error) {
	var chunk model.FileChunk
	err := f.db.Where("upload_id = ?", uploadId).First(&chunk).Error
	return chunk, err
}

func (f *FileChunkRepositoryImpl) FindUnfinishedUpload(fileMd5 string, fileSize int64, fileName string) (model.FileChunk, error) {
	var chunk model.FileChunk
	err := f.db.
		Where("file_md5 = ? AND file_size = ? AND file_name = ? AND merged = ? AND uploaded = ?", fileMd5, fileSize, fileName, false, false).
		Order("gmt_create DESC, id DESC").
		First(&chunk).Error
	return chunk, err
}

func (f *FileChunkRepositoryImpl) MarkUploadMerged(uploadId string) error {
	return f.db.Model(&model.FileChunk{}).
		Where("upload_id = ?", uploadId).
		Update("merged", true).
		Error
}
