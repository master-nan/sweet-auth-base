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

func (s *SysDictService) getSysDictByID(id int) (model.SysDict, error) {
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

func (s *SysDictService) getSysDictList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysDict], error) {
	result, err := s.sysDictRepo.GetSysDictList(basic, table)
	return result, err
}

func (s *SysDictService) getSysDictByCode(code string) (model.SysDict, error) {
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
	dict, e := s.getSysDictByCode(req.DictCode)
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
	data, err := s.getSysDictByID(req.Id)
	if err != nil {
		return err
	}
	tx := s.sysDictRepo.DBWithContext(ctx)
	if err := s.sysDictRepo.Update(tx, req, req.Id); err != nil {
		return err
	}
	s.invalidateDictCache(data)
	return nil
}

func (s *SysDictService) DeleteSysDictById(ctx context.Context, id int) error {
	data, err := s.getSysDictByID(id)
	if err != nil {
		return err
	}
	tx := s.sysDictRepo.DBWithContext(ctx)
	if err := s.sysDictRepo.DeleteById(tx, id); err != nil {
		return err
	}
	s.invalidateDictCache(data)
	return nil
}

func (s *SysDictService) getSysDictItemByID(id int) (model.SysDictItem, error) {
	data, err := s.sysDictItemRepo.FindById(id)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysDictItem{}, nil
	}
	return data, err
}

func (s *SysDictService) getSysDictItemsByDictID(id int) ([]model.SysDictItem, error) {
	result, err := s.sysDictItemRepo.GetSysDictItemsByDictId(id)
	return result, err
}

func (s *SysDictService) CreateSysDictItem(ctx context.Context, req request.DictItemCreateReq) error {
	dict, err := s.getSysDictByID(req.DictId)
	if err != nil {
		return err
	}
	var data model.SysDictItem
	err = copier.Copy(&data, &req)
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
	s.invalidateDictCache(dict)
	return nil
}

func (s *SysDictService) UpdateSysDictItem(ctx context.Context, req request.DictItemUpdateReq) error {
	dictItem, err := s.getSysDictItemByID(req.Id)
	if err != nil {
		return err
	}
	dict, err := s.getSysDictByID(dictItem.DictId)
	if err != nil {
		return err
	}
	tx := s.sysDictRepo.DBWithContext(ctx)
	err = s.sysDictItemRepo.Update(tx, &req, req.Id)
	if err != nil {
		zap.L().Error("UpdateSysDictItem err:", zap.Error(err))
		return err
	}
	s.invalidateDictCache(dict)
	return nil
}

func (s *SysDictService) DeleteSysDictItemById(ctx context.Context, id int) error {
	dictItem, err := s.getSysDictItemByID(id)
	if err != nil {
		return err
	}
	dict, err := s.getSysDictByID(dictItem.DictId)
	if err != nil {
		return err
	}
	tx := s.sysDictRepo.DBWithContext(ctx)
	err = s.sysDictItemRepo.DeleteById(tx, id)
	if err != nil {
		zap.L().Error("DeleteSysDictItemById err:", zap.Error(err))
		return err
	}
	s.invalidateDictCache(dict)
	return nil
}

func (s *SysDictService) invalidateDictCache(dict model.SysDict) {
	if dict.Id == 0 {
		return
	}
	_ = s.sysDictCache.Delete(strconv.Itoa(dict.Id))
	_ = s.sysDictCache.Delete(dict.DictCode)
}

func sysDictItemResponse(data model.SysDictItem) response.SysDictItemRes {
	return response.SysDictItemRes{
		BasicRes:  response.NewBasicRes(data.Basic),
		DictId:    data.DictId,
		ItemName:  data.ItemName,
		ItemCode:  data.ItemCode,
		ItemValue: data.ItemValue,
	}
}

func sysDictItemResponses(items []model.SysDictItem) []response.SysDictItemRes {
	result := make([]response.SysDictItemRes, 0, len(items))
	for _, item := range items {
		result = append(result, sysDictItemResponse(item))
	}
	return result
}

func sysDictResponse(data model.SysDict) response.SysDictRes {
	return response.SysDictRes{
		BasicRes:  response.NewBasicRes(data.Basic),
		DictName:  data.DictName,
		DictCode:  data.DictCode,
		DictItems: sysDictItemResponses(data.DictItems),
	}
}

func (s *SysDictService) GetSysDictByIdResponse(id int) (response.SysDictRes, error) {
	data, err := s.getSysDictByID(id)
	return sysDictResponse(data), err
}

func (s *SysDictService) GetSysDictByCodeResponse(code string) (response.SysDictRes, error) {
	data, err := s.getSysDictByCode(code)
	return sysDictResponse(data), err
}

func (s *SysDictService) GetRuntimeDictByCodeResponse(code string) (response.RuntimeDictRes, error) {
	data, err := s.getSysDictByCode(code)
	if err != nil {
		return response.RuntimeDictRes{}, err
	}
	return runtimeDictResponse(data), nil
}

func runtimeDictResponse(data model.SysDict) response.RuntimeDictRes {
	items := make([]response.RuntimeDictItemRes, 0, len(data.DictItems))
	for _, item := range data.DictItems {
		if !item.State {
			continue
		}
		items = append(items, response.RuntimeDictItemRes{
			ItemName:  item.ItemName,
			ItemCode:  item.ItemCode,
			ItemValue: item.ItemValue,
		})
	}
	return response.RuntimeDictRes{DictName: data.DictName, DictCode: data.DictCode, DictItems: items}
}

func (s *SysDictService) GetSysDictListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.SysDictRes], error) {
	data, err := s.getSysDictList(basic, table)
	if err != nil {
		return response.ListResult[response.SysDictRes]{}, err
	}
	items := make([]response.SysDictRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, sysDictResponse(item))
	}
	return response.ListResult[response.SysDictRes]{Data: items, Total: data.Total}, nil
}

func (s *SysDictService) GetSysDictItemByIdResponse(id int) (response.SysDictItemRes, error) {
	data, err := s.getSysDictItemByID(id)
	return sysDictItemResponse(data), err
}

func (s *SysDictService) GetSysDictItemsByDictIdResponse(id int) ([]response.SysDictItemRes, error) {
	data, err := s.getSysDictItemsByDictID(id)
	return sysDictItemResponses(data), err
}
