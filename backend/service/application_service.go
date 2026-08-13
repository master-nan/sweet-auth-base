/**
 * @Author: Nan
 * @Date: 2024/10/23 21:56
 */

package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/cache"
	error2 "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type ApplicationService struct {
	applicationRepo  repository.ApplicationRepository
	applicationCache *cache.ApplicationCache
	sf               *utils.Snowflake
}

func NewApplicationService(applicationRepo repository.ApplicationRepository, applicationCache *cache.ApplicationCache, sf *utils.Snowflake) *ApplicationService {
	return &ApplicationService{
		applicationRepo,
		applicationCache,
		sf,
	}
}

// GetApplicationById 根据id获取应用信息
func (a *ApplicationService) GetApplicationById(id int) (model.Application, error) {
	if a.applicationCache != nil {
		data, err := a.applicationCache.Get(strconv.Itoa(id))
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, cache.ErrCacheMiss) {
			return model.Application{}, err
		}
	}
	// 尝试从数据库获取
	result, err := a.applicationRepo.FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Application{}, nil
		}
		return model.Application{}, err
	}
	return result, nil
}

// GetApplicationList 获取应用列表
func (a *ApplicationService) GetApplicationList(basic *request.Basic, table model.SysTable) (response.ListResult[model.Application], error) {
	result, err := a.applicationRepo.GetApplicationList(basic, table)
	return result, err
}

// GetApplicationByAppKey 根据appKey获取应用信息
func (a *ApplicationService) GetApplicationByAppKey(appKey string) (model.Application, error) {
	var data model.Application
	var err error
	if a.applicationCache != nil {
		data, err = a.applicationCache.Get(appKey)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, cache.ErrCacheMiss) {
			return model.Application{}, err
		}
	}
	// 尝试从数据库获取
	data, err = a.applicationRepo.FindByAppKey(appKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Application{}, nil
		}
		return model.Application{}, err
	}
	if a.applicationCache != nil {
		_ = a.applicationCache.Set(appKey, data)
		_ = a.applicationCache.Set(strconv.Itoa(data.Id), data)
	}
	return data, nil
}

// CreateApplication 创建应用
func (a *ApplicationService) CreateApplication(ctx context.Context, req request.ApplicationCreateReq) (model.Application, error) {
	var data model.Application
	app, err := a.applicationRepo.FindByField("name", req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Application{}, err
	}
	if app.Id != 0 {
		return model.Application{}, error2.ErrAppNameExist
	}
	appKey, appSecret, err := a.GenerateAppKeyAndSecret()
	if err != nil {
		return model.Application{}, err
	}
	err = copier.Copy(&data, &req)
	if err != nil {
		return model.Application{}, err
	}
	data.AppKey = appKey
	data.AppSecret = appSecret
	id, err := a.sf.GenerateUniqueID()
	if err != nil {
		return model.Application{}, err
	}
	data.Id = int(id)
	tx := a.applicationRepo.DBWithContext(ctx)
	if err := a.applicationRepo.Create(tx, &data); err != nil {
		return model.Application{}, err
	}
	return data, nil
}

// UpdateApplication 更新应用
func (a *ApplicationService) UpdateApplication(ctx context.Context, req request.ApplicationUpdateReq) error {
	if strings.TrimSpace(req.DingSecret) == "" {
		existing, err := a.applicationRepo.FindById(req.Id)
		if err != nil {
			return err
		}
		req.DingSecret = existing.DingSecret
	}
	tx := a.applicationRepo.DBWithContext(ctx)
	err := a.applicationRepo.Update(tx, &req, req.Id)
	if err != nil {
		return err
	}
	a.RefreshCache(req.Id)
	return nil
}

func (a *ApplicationService) RotateApplicationSecret(ctx context.Context, id int) (model.Application, error) {
	app, err := a.applicationRepo.FindById(id)
	if err != nil {
		return model.Application{}, err
	}
	appSecret, err := a.generateUniqueAppSecret()
	if err != nil {
		return model.Application{}, err
	}
	tx := a.applicationRepo.DBWithContext(ctx)
	data := map[string]interface{}{"app_secret": appSecret}
	if err = a.applicationRepo.WithSelect("app_secret").Update(tx, data, id); err != nil {
		return model.Application{}, err
	}
	app.AppSecret = appSecret
	a.RefreshCache(id)
	return app, nil
}

// DeleteApplicationById 根据id删除应用
func (a *ApplicationService) DeleteApplicationById(ctx context.Context, id int) error {
	app, findErr := a.applicationRepo.FindById(id)
	tx := a.applicationRepo.DBWithContext(ctx)
	err := a.applicationRepo.DeleteById(tx, id)
	if err != nil {
		return err
	}
	if findErr == nil {
		a.deleteCacheForApplication(app)
	} else {
		a.deleteCacheKey(strconv.Itoa(id))
	}
	return nil
}

// RefreshCache 刷新缓存
func (a *ApplicationService) RefreshCache(id int) {
	if a.applicationCache == nil {
		return
	}
	app, err := a.applicationRepo.FindById(id)
	if err == nil {
		if app.Id != 0 {
			_ = a.applicationCache.Set(strconv.Itoa(app.Id), app)
			_ = a.applicationCache.Set(app.AppKey, app)
		}
	}
}

func (a *ApplicationService) DeleteCache(id int) {
	if a.applicationCache == nil {
		return
	}
	app, _ := a.GetApplicationById(id)
	a.deleteCacheForApplication(app)
}

func (a *ApplicationService) deleteCacheForApplication(app model.Application) {
	if app.Id == 0 {
		return
	}
	a.deleteCacheKey(strconv.Itoa(app.Id))
	a.deleteCacheKey(app.AppKey)
}

func (a *ApplicationService) deleteCacheKey(key string) {
	if a.applicationCache == nil || strings.TrimSpace(key) == "" {
		return
	}
	_ = a.applicationCache.Delete(key)
}

// GenerateAppKeyAndSecret 生成appKey和appSecret
func (a *ApplicationService) GenerateAppKeyAndSecret() (string, string, error) {
	var apiKey string
	var err error
	for {
		apiKey, err = utils.GenerateSecretKey(32)
		if err != nil {
			return "", "", err
		}
		if !a.applicationRepo.IsAppKeyExists(apiKey) {
			break
		}
	}
	apiSecret, err := a.generateUniqueAppSecret()
	if err != nil {
		return "", "", err
	}
	return apiKey, apiSecret, nil
}

func (a *ApplicationService) generateUniqueAppSecret() (string, error) {
	for {
		apiSecret, err := utils.GenerateSecretKey(32)
		if err != nil {
			return "", err
		}
		if !a.applicationRepo.IsAppSecretExists(apiSecret) {
			return apiSecret, nil
		}
	}
}
