/**
 * @Author: Nan
 * @Date: 2024/10/23 21:53
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"gorm.io/gorm"
)

type ApplicationRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.Application]
}

func NewApplicationRepositoryImpl(primaryDB *database.PrimaryDB) *ApplicationRepositoryImpl {
	return &ApplicationRepositoryImpl{
		primaryDB.DB,
		NewBasicRepositoryImpl(primaryDB.DB, &model.Application{}),
	}
}

func (a *ApplicationRepositoryImpl) FindByAppKey(appKey string) (model.Application, error) {
	var application model.Application
	err := a.db.Where("app_key = ?", appKey).First(&application).Error
	return application, err
}

func (a *ApplicationRepositoryImpl) GetApplicationList(basic *request.Basic, table model.SysTable) (response.ListResult[model.Application], error) {
	var repo response.ListResult[model.Application]
	var applicationList []model.Application
	total, err := a.PaginateAndCountAsync(basic, &applicationList, table)
	repo.Data = applicationList
	repo.Total = int(total)
	return repo, err
}

func (a *ApplicationRepositoryImpl) IsAppKeyExists(appKey string) bool {
	var count int64
	a.db.Model(&model.Application{}).Where("app_key = ?", appKey).Count(&count)
	return count > 0
}

func (a *ApplicationRepositoryImpl) IsAppSecretExists(appSecret string) bool {
	var count int64
	a.db.Model(&model.Application{}).Where("app_secret = ?", appSecret).Count(&count)
	return count > 0
}
