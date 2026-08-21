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

// getApplicationByID 根据id获取应用信息
func (a *ApplicationService) getApplicationByID(id int) (model.Application, error) {
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

// GetApplicationForAuthentication bypasses cached snapshots so application
// disablement and secret rotation take effect for already issued AppTokens.
func (a *ApplicationService) GetApplicationForAuthentication(ctx context.Context, id int) (model.Application, error) {
	result, err := a.applicationRepo.FindByIdWithDB(a.applicationRepo.DBWithContext(ctx), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Application{}, nil
		}
		return model.Application{}, err
	}
	return result, nil
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

// CreateApplicationResponse 创建应用并返回只展示一次的凭据。
func (a *ApplicationService) CreateApplicationResponse(ctx context.Context, req request.ApplicationCreateReq) (response.ApplicationSecretRes, error) {
	var data model.Application
	app, err := a.applicationRepo.FindByField("name", req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.ApplicationSecretRes{}, err
	}
	if app.Id != 0 {
		return response.ApplicationSecretRes{}, error2.ErrAppNameExist
	}
	appKey, appSecret, err := a.generateAppKeyAndSecret()
	if err != nil {
		return response.ApplicationSecretRes{}, err
	}
	err = copier.Copy(&data, &req)
	if err != nil {
		return response.ApplicationSecretRes{}, err
	}
	data.AppKey = appKey
	data.AppSecret = appSecret
	id, err := a.sf.GenerateUniqueID()
	if err != nil {
		return response.ApplicationSecretRes{}, err
	}
	data.Id = int(id)
	tx := a.applicationRepo.DBWithContext(ctx)
	if err := a.applicationRepo.Create(tx, &data); err != nil {
		return response.ApplicationSecretRes{}, err
	}
	return applicationSecretResponse(data), nil
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

func (a *ApplicationService) RotateApplicationSecretResponse(ctx context.Context, id int) (response.ApplicationSecretRes, error) {
	app, err := a.applicationRepo.FindById(id)
	if err != nil {
		return response.ApplicationSecretRes{}, err
	}
	appSecret, err := a.generateUniqueAppSecret()
	if err != nil {
		return response.ApplicationSecretRes{}, err
	}
	tx := a.applicationRepo.DBWithContext(ctx)
	data := map[string]interface{}{"app_secret": appSecret}
	if err = a.applicationRepo.WithSelect("app_secret").Update(tx, data, id); err != nil {
		return response.ApplicationSecretRes{}, err
	}
	app.AppSecret = appSecret
	a.RefreshCache(id)
	return applicationSecretResponse(app), nil
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

func applicationResponse(data model.Application) response.ApplicationRes {
	return response.ApplicationRes{
		BasicRes:   response.NewBasicRes(data.Basic),
		Name:       data.Name,
		AppKey:     data.AppKey,
		Expiration: data.Expiration,
		DingKey:    data.DingKey,
		DingAppID:  data.DingAppID,
		Remark:     data.Remark,
	}
}

func applicationSecretResponse(data model.Application) response.ApplicationSecretRes {
	return response.ApplicationSecretRes{
		Id: data.Id, Name: data.Name, AppKey: data.AppKey,
		AppSecret: data.AppSecret, Expiration: data.Expiration,
	}
}

func (s *ApplicationService) GetApplicationByIdResponse(id int) (response.ApplicationRes, error) {
	data, err := s.getApplicationByID(id)
	return applicationResponse(data), err
}

func (s *ApplicationService) GetApplicationListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.ApplicationRes], error) {
	data, err := s.applicationRepo.GetApplicationList(basic, table)
	if err != nil {
		return response.ListResult[response.ApplicationRes]{}, err
	}
	items := make([]response.ApplicationRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, applicationResponse(item))
	}
	return response.ListResult[response.ApplicationRes]{Data: items, Total: data.Total}, nil
}

func (a *ApplicationService) DeleteCache(id int) {
	if a.applicationCache == nil {
		return
	}
	app, _ := a.getApplicationByID(id)
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

func (a *ApplicationService) generateAppKeyAndSecret() (string, string, error) {
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
