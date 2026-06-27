/**
 * @Author: Nan
 * @Date: 2024/8/5 下午11:53
 */

package repository

import (
	"backend/model"

	"github.com/gin-gonic/gin"
)

type FileRepository interface {
	BasicRepository[model.File]
	FindByFileUuid(uuid string) (model.File, error)
	FindByFileMd5(md5 string) (model.File, error)
	DeleteFile(ctx *gin.Context, file model.File) error
}

type FileChunkRepository interface {
	BasicRepository[model.FileChunk]
	FindByUploadId(uploadId string) ([]model.FileChunk, error)
	FindByUploadIdAndIndex(uploadId string, index int) (model.FileChunk, error)
	CountUploadedChunks(uploadId string) (int64, error)
	GetFirstChunk(uploadId string) (model.FileChunk, error)
	FindUnfinishedUpload(fileMd5 string, fileSize int64, fileName string) (model.FileChunk, error)
	MarkUploadMerged(uploadId string) error
}
