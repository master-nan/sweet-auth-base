/**
 * @Author: Nan
 * @Date: 2024/5/25 下午2:24
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SysDictRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysDict]
}

func NewSysDictRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysDictRepositoryImpl {
	return &SysDictRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysDict{}),
	}
}

func (i *SysDictRepositoryImpl) GetSysDictList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysDict], error) {
	var repo response.ListResult[model.SysDict]
	var sysDictList []model.SysDict
	total, err := i.PaginateAndCountAsync(basic, &sysDictList, table)
	zap.L().Debug("sysDictList", zap.Any("sysDictList", sysDictList))
	repo.Data = sysDictList
	repo.Total = int(total)
	return repo, err
}

func (i *SysDictRepositoryImpl) GetSysDictByCode(code string) (model.SysDict, error) {
	var sysDict model.SysDict
	err := i.db.Preload("DictItems").Where("dict_code = ?", code).First(&sysDict).Error
	return sysDict, err
}
