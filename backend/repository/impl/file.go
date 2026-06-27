/**
 * @Author: Nan
 * @Date: 2024/8/5 下午11:53
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FileRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.File]
}

func NewFileRepositoryImpl(primaryDB *database.PrimaryDB) *FileRepositoryImpl {
	return &FileRepositoryImpl{
		primaryDB.DB,
		NewBasicRepositoryImpl(primaryDB.DB, &model.File{}),
	}
}

func (f *FileRepositoryImpl) FindByFileUuid(uuid string) (model.File, error) {
	var file model.File
	err := f.db.Where("file_uuid = ?", uuid).First(&file).Error
	return file, err
}

func (f *FileRepositoryImpl) FindByFileMd5(md5 string) (model.File, error) {
	var file model.File
	err := f.db.Where("file_md5 = ?", md5).First(&file).Error
	return file, err
}

func (f *FileRepositoryImpl) DeleteFile(ctx *gin.Context, file model.File) error {
	return f.ExecuteTx(ctx, func(tx *gorm.DB) error {
		if err := f.Update(tx, map[string]interface{}{
			"file_md5":  deletedFileUniqueValue(file.FileMd5, file.Id),
			"file_uuid": deletedFileUniqueValue(file.FileUuid, file.Id),
		}, file.Id); err != nil {
			return err
		}
		return f.DeleteById(tx, file.Id)
	})
}

func deletedFileUniqueValue(value string, id int) string {
	value = strings.TrimSpace(value)
	suffix := fmt.Sprintf("#deleted-%d", id)
	if strings.HasSuffix(value, suffix) {
		return value
	}
	maxLen := 128 - len(suffix)
	if maxLen < 0 {
		maxLen = 0
	}
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	return value + suffix
}
