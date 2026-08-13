/**
 * @Author: Nan
 * @Date: 2024/8/5 下午11:53
 */

package repository

import (
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type FileRepository interface {
	BasicRepository[model.File]
	FindByFileUuid(ctx context.Context, uuid string) (model.File, error)
	FindByFileMd5(ctx context.Context, md5 string) (model.File, error)
	DeleteFile(tx *gorm.DB, file model.File) error
}

type FileChunkRepository interface {
	BasicRepository[model.FileChunk]
	FindByUploadId(ctx context.Context, uploadId string) ([]model.FileChunk, error)
	FindByUploadIdAndIndex(ctx context.Context, uploadId string, index int) (model.FileChunk, error)
	CountUploadedChunks(ctx context.Context, uploadId string) (int64, error)
	GetFirstChunk(ctx context.Context, uploadId string) (model.FileChunk, error)
	FindUnfinishedUpload(ctx context.Context, fileMd5 string, fileSize int64, fileName string, actorID int, allowAll bool) (model.FileChunk, error)
	MarkUploadMerged(ctx context.Context, uploadId, fileMd5 string) error
}
