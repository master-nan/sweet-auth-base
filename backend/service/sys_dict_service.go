/**
 * @Author: Nan
 * @Date: 2024/5/23 下午2:59
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
	"errors"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strconv"
)

type SysDictService struct {
	sysDictRepo     repository.SysDictRepository
	sysDictItemRepo repository.SysDictItemRepository
	sf              *utils.Snowflake
	sysDictCache    *cache.SysDictCache
}

func NewSysDictService(sysDictRepo repository.SysDictRepository, sysDictItemRepo repository.SysDictItemRepository, sf *utils.Snowflake, sysDictCache *cache.SysDictCache) *SysDictService {
	return &SysDictService{
		sysDictRepo,
		sysDictItemRepo,
		sf,
		sysDictCache,
	}
}

func (s *SysDictService) GetSysDictById(id int) (model.SysDict, error) {
	data, err := s.sysDictCache.Get(strconv.Itoa(id))
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, cache.ErrCacheMiss) {
		return model.SysDict{}, err
	}
	// 尝试从数据库获取
	dict, err := s.sysDictRepo.WithPreload("DictItems").FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysDict{}, nil
		}
		return model.SysDict{}, err
	}
	// 将结果设置回缓存
	s.sysDictCache.Set(strconv.Itoa(id), dict)
	return dict, nil
}

func (s *SysDictService) GetSysDictList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysDict], error) {
	result, err := s.sysDictRepo.GetSysDictList(basic, table)
	return result, err
}

func (s *SysDictService) GetSysDictByCode(code string) (model.SysDict, error) {
	data, err := s.sysDictCache.Get(code)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, cache.ErrCacheMiss) { // 如果缓存错误不是因为未命中
		return model.SysDict{}, err
	}
	// 尝试从数据库获取
	dict, err := s.sysDictRepo.GetSysDictByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysDict{}, nil
		}
		return model.SysDict{}, err
	}
	// 将结果设置回缓存
	s.sysDictCache.Set(code, dict)
	return dict, nil
}

func (s *SysDictService) CreateSysDict(ctx context.Context, req request.DictCreateReq) (int64, error) {
	var data model.SysDict
	dict, e := s.GetSysDictByCode(req.DictCode)
	if e != nil {
		return 0, e
	}
	if dict.Id != 0 {
		return 0, error2.ErrDictCodeExist
	}
	err := copier.Copy(&data, &req)
	if err != nil {
		zap.L().Error("结构体字段映射失败", zap.String("target", "SysDict"), zap.Error(err))
		return 0, err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, err
	}
	data.Id = int(id)
	tx := s.sysDictRepo.DBWithContext(ctx)
	return id, s.sysDictRepo.Create(tx, &data)
}

func (s *SysDictService) UpdateSysDict(ctx context.Context, req request.DictUpdateReq) error {
	tx := s.sysDictRepo.DBWithContext(ctx)
	err := s.sysDictRepo.Update(tx, req, req.Id)
	if err != nil {
		return err
	}
	data, err := s.GetSysDictById(req.Id)
	if err != nil {
		return err
	}
	if data.Id != 0 {
		s.sysDictCache.Delete(strconv.Itoa(data.Id))
		s.sysDictCache.Delete(data.DictCode)
	}
	return nil
}

func (s *SysDictService) DeleteSysDictById(ctx context.Context, id int) error {
	tx := s.sysDictRepo.DBWithContext(ctx)
	err := s.sysDictRepo.DeleteById(tx, id)
	data, err := s.GetSysDictById(id)
	if err != nil {
		return err
	}
	if data.Id != 0 {
		s.sysDictCache.Delete(strconv.Itoa(data.Id))
		s.sysDictCache.Delete(data.DictCode)
	}
	return err
}

func (s *SysDictService) GetSysDictItemById(id int) (model.SysDictItem, error) {
	data, err := s.sysDictItemRepo.FindById(id)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysDictItem{}, nil
	}
	return data, err
}

func (s *SysDictService) GetSysDictItemsByDictId(id int) ([]model.SysDictItem, error) {
	result, err := s.sysDictItemRepo.GetSysDictItemsByDictId(id)
	return result, err
}

func (s *SysDictService) CreateSysDictItem(ctx context.Context, req request.DictItemCreateReq) error {
	var data model.SysDictItem
	err := copier.Copy(&data, &req)
	if err != nil {
		zap.L().Error("Error during struct mapping:", zap.Error(err))
		return err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	data.Id = int(id)
	tx := s.sysDictRepo.DBWithContext(ctx)
	err = s.sysDictItemRepo.Create(tx, &data)
	if err != nil {
		zap.L().Error("CreateSysDictItem err:", zap.Error(err))
		return err
	}
	dict, err := s.GetSysDictById(req.DictId)
	if err != nil {
		zap.L().Error("CreateSysDictItem err:", zap.Error(err))
		return err
	}
	if dict.Id != 0 {
		s.sysDictCache.Delete(strconv.Itoa(dict.Id))
		s.sysDictCache.Delete(dict.DictCode)
	}
	return nil
}

func (s *SysDictService) UpdateSysDictItem(ctx context.Context, req request.DictItemUpdateReq) error {
	tx := s.sysDictRepo.DBWithContext(ctx)
	err := s.sysDictItemRepo.Update(tx, &req, req.Id)
	if err != nil {
		zap.L().Error("UpdateSysDictItem err:", zap.Error(err))
		return err
	}
	dictItem, err := s.GetSysDictItemById(req.Id)
	if err != nil {
		zap.L().Error("UpdateSysDictItem err:", zap.Error(err))
		return err
	}
	dict, err := s.GetSysDictById(dictItem.DictId)
	if err != nil {
		zap.L().Error("UpdateSysDictItem err:", zap.Error(err))
		return err
	}
	if dict.Id != 0 {
		s.sysDictCache.Delete(strconv.Itoa(dict.Id))
		s.sysDictCache.Delete(dict.DictCode)
	}
	return nil
}

func (s *SysDictService) DeleteSysDictItemById(ctx context.Context, id int) error {
	tx := s.sysDictRepo.DBWithContext(ctx)
	err := s.sysDictItemRepo.DeleteById(tx, id)
	if err != nil {
		zap.L().Error("DeleteSysDictItemById err:", zap.Error(err))
		return err
	}
	dictItem, err := s.GetSysDictItemById(id)
	if err != nil {
		zap.L().Error("DeleteSysDictItemById err:", zap.Error(err))
		return err
	}
	dict, err := s.GetSysDictById(dictItem.DictId)
	if err != nil {
		zap.L().Error("DeleteSysDictItemById err:", zap.Error(err))
		return err
	}
	if dict.Id != 0 {
		s.sysDictCache.Delete(strconv.Itoa(dict.Id))
		s.sysDictCache.Delete(dict.DictCode)
	}
	return nil
}
