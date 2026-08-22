package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/cache"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	platformmetadata "backend/internal/metadata"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MetadataRuntimeService 是平台表Metadata的唯一运行时读取边界。
// 配置Service可以调用其缓存生命周期方法，但运行时消费者不会取得SysTable模型或管理DTO。
type MetadataRuntimeService struct {
	tables     repository.SysTableRepository
	fields     repository.SysTableFieldRepository
	tableCache *cache.SysTableCache
	fieldCache *cache.SysTableFieldCache
}

// MetadataSecurityReader 只暴露Data Permission校验安全字段所需的窄接口，
// 调用方不能借此取得完整Metadata管理模型。
type MetadataSecurityReader interface {
	FindMetadataSecurityField(context.Context, *gorm.DB, int) (datapermission.MetadataFieldRecord, error)
	HasPhysicalColumn(context.Context, *gorm.DB, int, string) (bool, error)
}

func NewMetadataRuntimeService(
	tables repository.SysTableRepository,
	fields repository.SysTableFieldRepository,
	tableCache *cache.SysTableCache,
	fieldCache *cache.SysTableFieldCache,
) *MetadataRuntimeService {
	return &MetadataRuntimeService{
		tables:     tables,
		fields:     fields,
		tableCache: tableCache,
		fieldCache: fieldCache,
	}
}

func (s *MetadataRuntimeService) GetTable(
	ctx context.Context,
	tableCode string,
) (platformmetadata.TableMetadata, error) {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return platformmetadata.TableMetadata{}, myerrors.ErrParamInvalid
	}
	table, err := s.configTableByCode(ctx, tableCode)
	if err != nil {
		return platformmetadata.TableMetadata{}, err
	}
	if table.Id == 0 || !table.State {
		return platformmetadata.TableMetadata{}, myerrors.ErrDataNotFound
	}
	return platformmetadata.ProjectTable(table), nil
}

func (s *MetadataRuntimeService) GetTableByID(
	ctx context.Context,
	tableID int,
) (platformmetadata.TableMetadata, error) {
	if tableID <= 0 {
		return platformmetadata.TableMetadata{}, myerrors.ErrParamInvalid
	}
	table, err := s.configTableByID(ctx, tableID)
	if err != nil {
		return platformmetadata.TableMetadata{}, err
	}
	if table.Id == 0 || !table.State {
		return platformmetadata.TableMetadata{}, myerrors.ErrDataNotFound
	}
	return platformmetadata.ProjectTable(table), nil
}

func (s *MetadataRuntimeService) GetField(
	ctx context.Context,
	fieldID int,
) (platformmetadata.FieldMetadata, error) {
	field, err := s.configFieldByID(ctx, fieldID)
	if err != nil {
		return platformmetadata.FieldMetadata{}, err
	}
	if !field.State {
		return platformmetadata.FieldMetadata{}, myerrors.ErrDataNotFound
	}
	projected, ok := platformmetadata.ProjectField(field)
	if !ok {
		return platformmetadata.FieldMetadata{}, myerrors.ErrDataNotFound
	}
	return projected, nil
}

func (s *MetadataRuntimeService) GetFields(
	ctx context.Context,
	tableID int,
) ([]platformmetadata.FieldMetadata, error) {
	table, err := s.GetTableByID(ctx, tableID)
	if err != nil {
		return nil, err
	}
	return append([]platformmetadata.FieldMetadata(nil), table.Fields...), nil
}

func (s *MetadataRuntimeService) ListTables(
	ctx context.Context,
) ([]platformmetadata.TableMetadata, error) {
	tables, err := s.tables.ListRuntimeTables(ctx)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	result := make([]platformmetadata.TableMetadata, 0, len(tables))
	for _, table := range tables {
		if !table.State {
			continue
		}
		result = append(result, platformmetadata.ProjectTable(table))
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}

func (s *MetadataRuntimeService) QueryRelationOptions(
	ctx context.Context,
	fieldID int,
	req request.RuntimeRelationOptionsReq,
) (response.ListResult[response.RuntimeRelationOptionRes], error) {
	field, err := s.GetField(ctx, fieldID)
	if err != nil {
		return response.ListResult[response.RuntimeRelationOptionRes]{}, err
	}
	if field.Relation == nil {
		return response.ListResult[response.RuntimeRelationOptionRes]{}, myerrors.NewValidationError("字段未配置运行时关系展示")
	}
	target, err := s.GetTable(ctx, field.Relation.TargetTableCode)
	if err != nil {
		return response.ListResult[response.RuntimeRelationOptionRes]{}, err
	}
	fieldCodes := make(map[string]platformmetadata.FieldMetadata, len(target.Fields))
	for _, targetField := range target.Fields {
		fieldCodes[targetField.Code] = targetField
	}
	if _, ok := fieldCodes[field.Relation.ValueField]; !ok {
		return response.ListResult[response.RuntimeRelationOptionRes]{}, myerrors.NewValidationError("关系取值字段不可用")
	}
	if _, ok := fieldCodes[field.Relation.DisplayField]; !ok {
		return response.ListResult[response.RuntimeRelationOptionRes]{}, myerrors.NewValidationError("关系展示字段不可用")
	}
	page, num := req.Page, req.Num
	if page <= 0 {
		page = 1
	}
	if num <= 0 {
		num = 50
	}
	if num > 200 || len(req.SelectedValues) > 100 || len(strings.TrimSpace(req.Keyword)) > 128 {
		return response.ListResult[response.RuntimeRelationOptionRes]{}, myerrors.ErrParamInvalid
	}
	filters := make(map[string]interface{}, len(field.Relation.FilterMapping))
	for targetField, sourceField := range field.Relation.FilterMapping {
		value, exists := req.SourceValues[sourceField]
		if !exists || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
			continue
		}
		if _, ok := fieldCodes[targetField]; !ok {
			return response.ListResult[response.RuntimeRelationOptionRes]{}, myerrors.NewValidationError("关系过滤字段不可用")
		}
		filters[targetField] = value
	}
	_, hasState := fieldCodes["state"]
	_, hasDeletedAt := fieldCodes["gmt_delete"]
	items, total, err := s.tables.QueryRuntimeRelationOptions(ctx, repository.RuntimeRelationOptionQuery{
		TableCode: target.Code, ValueField: field.Relation.ValueField, DisplayField: field.Relation.DisplayField,
		ParentField: field.Relation.ParentField, Keyword: strings.TrimSpace(req.Keyword), Page: page, Num: num,
		SelectedValues: req.SelectedValues, Filters: filters, HasState: hasState, HasDeletedAt: hasDeletedAt,
	})
	if err != nil {
		return response.ListResult[response.RuntimeRelationOptionRes]{}, myerrors.WrapDatabaseError(err)
	}
	result := make([]response.RuntimeRelationOptionRes, 0, len(items))
	for _, item := range items {
		result = append(result, response.RuntimeRelationOptionRes{Value: item.Value, Label: item.Label, ParentValue: item.ParentValue})
	}
	return response.ListResult[response.RuntimeRelationOptionRes]{Data: result, Total: total}, nil
}

// listConfigTables 仅服务Metadata管理页，持久化模型不会离开service包。
func (s *MetadataRuntimeService) listConfigTables(
	ctx context.Context,
	basic *request.Basic,
) (response.ListResult[model.SysTable], error) {
	queryTable, err := s.configTableByCode(ctx, basic.TableCode)
	if err != nil {
		return response.ListResult[model.SysTable]{}, err
	}
	return s.tables.GetTableList(ctx, basic, queryTable)
}

func (s *MetadataRuntimeService) configTableByID(
	ctx context.Context,
	tableID int,
) (model.SysTable, error) {
	if s == nil || s.tables == nil {
		return model.SysTable{}, myerrors.WrapSystemError(errors.New("metadata runtime is not initialized"))
	}
	key := strconv.Itoa(tableID)
	if s.tableCache != nil {
		data, err := s.tableCache.Get(key)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, cache.ErrCacheMiss) {
			return model.SysTable{}, err
		}
	}
	data, err := s.tables.GetTableById(ctx, tableID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysTable{}, nil
	}
	if err != nil {
		return model.SysTable{}, err
	}
	s.setCachedTable(data)
	return data, nil
}

func (s *MetadataRuntimeService) configTableByCode(
	ctx context.Context,
	tableCode string,
) (model.SysTable, error) {
	if s == nil || s.tables == nil {
		return model.SysTable{}, myerrors.WrapSystemError(errors.New("metadata runtime is not initialized"))
	}
	if s.tableCache != nil {
		data, err := s.tableCache.Get(tableCode)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, cache.ErrCacheMiss) {
			return model.SysTable{}, err
		}
	}
	data, err := s.tables.GetTableByTableCode(ctx, tableCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysTable{}, nil
	}
	if err != nil {
		return model.SysTable{}, err
	}
	s.setCachedTable(data)
	return data, nil
}

func (s *MetadataRuntimeService) configFieldByID(
	ctx context.Context,
	fieldID int,
) (model.SysTableField, error) {
	if s == nil || s.fields == nil {
		return model.SysTableField{}, myerrors.WrapSystemError(errors.New("metadata runtime is not initialized"))
	}
	key := strconv.Itoa(fieldID)
	if s.fieldCache != nil {
		data, err := s.fieldCache.Get(key)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, cache.ErrCacheMiss) {
			return model.SysTableField{}, err
		}
	}
	data, err := s.fields.WithContext(ctx).FindById(fieldID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysTableField{}, nil
	}
	if err != nil {
		return model.SysTableField{}, err
	}
	if s.fieldCache != nil {
		if err = s.fieldCache.Set(key, data); err != nil {
			return model.SysTableField{}, err
		}
	}
	return data, nil
}

// FindMetadataTable和FindMetadataField实现Data Permission所需的MetadataFieldReader，
// 避免Data Permission直接依赖Metadata Repository。
func (s *MetadataRuntimeService) FindMetadataTable(
	ctx context.Context,
	tableID int,
) (datapermission.MetadataTableRecord, error) {
	table, err := s.tables.WithContext(ctx).WithUnscoped().FindById(tableID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return datapermission.MetadataTableRecord{}, datapermission.ErrMetadataTableRecordNotFound
	}
	if err != nil {
		return datapermission.MetadataTableRecord{}, err
	}
	return datapermission.MetadataTableRecord{
		Id:      table.Id,
		State:   table.State,
		Deleted: table.GmtDelete.Valid,
	}, nil
}

func (s *MetadataRuntimeService) FindMetadataField(
	ctx context.Context,
	fieldID int,
) (datapermission.MetadataFieldRecord, error) {
	field, err := s.fields.WithContext(ctx).WithUnscoped().FindById(fieldID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return datapermission.MetadataFieldRecord{}, datapermission.ErrMetadataFieldRecordNotFound
	}
	if err != nil {
		return datapermission.MetadataFieldRecord{}, err
	}
	return dataPermissionFieldRecord(field), nil
}

func (s *MetadataRuntimeService) HasPhysicalColumn(
	ctx context.Context,
	db *gorm.DB,
	tableID int,
	fieldCode string,
) (bool, error) {
	if db == nil {
		db = s.tables.DBWithContext(ctx)
	} else {
		db = db.WithContext(ctx)
	}
	table, err := s.tables.FindMetadataIdentity(db, tableID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if table.Id == 0 || !table.State || table.GmtDelete.Valid {
		return false, nil
	}
	return s.tables.HasTableColumn(db, table.TableCode, fieldCode), nil
}

func (s *MetadataRuntimeService) FindMetadataSecurityField(
	ctx context.Context,
	db *gorm.DB,
	fieldID int,
) (datapermission.MetadataFieldRecord, error) {
	if db == nil {
		db = s.fields.DBWithContext(ctx)
	} else {
		db = db.WithContext(ctx)
	}
	field, err := s.fields.FindMetadataSecurityField(db, fieldID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return datapermission.MetadataFieldRecord{}, datapermission.ErrMetadataFieldRecordNotFound
	}
	if err != nil {
		return datapermission.MetadataFieldRecord{}, err
	}
	return dataPermissionFieldRecord(field), nil
}

func dataPermissionFieldRecord(field model.SysTableField) datapermission.MetadataFieldRecord {
	expression := ""
	if field.Expression != nil {
		expression = *field.Expression
	}
	return datapermission.MetadataFieldRecord{
		Id:               field.Id,
		TableId:          field.TableId,
		State:            field.State,
		Deleted:          field.GmtDelete.Valid,
		FieldCode:        field.FieldCode,
		FieldType:        field.FieldType,
		InputType:        field.InputType,
		FieldCategory:    field.FieldCategory,
		Expression:       expression,
		IsPrimaryKey:     field.IsPrimaryKey,
		IsAdvancedSearch: field.IsAdvancedSearch,
	}
}

func (s *MetadataRuntimeService) Refresh(ctx context.Context, tableID int) {
	if s == nil || tableID <= 0 {
		return
	}
	table, err := s.tables.GetTableById(ctx, tableID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Error("refresh metadata runtime cache failed", zap.Int("table_id", tableID), zap.Error(err))
		}
		return
	}
	s.deleteCachedTableByID(tableID)
	s.deleteCachedTable(table)
	s.setCachedTable(table)
}

func (s *MetadataRuntimeService) Invalidate(ctx context.Context, tableID int) {
	if s == nil || tableID <= 0 {
		return
	}
	s.deleteCachedTableByID(tableID)
	table, err := s.tables.WithContext(ctx).WithUnscoped().WithPreload("TableFields").FindById(tableID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Error("load deleted metadata for cache invalidation failed", zap.Int("table_id", tableID), zap.Error(err))
		}
		return
	}
	s.deleteCachedTable(table)
}

func (s *MetadataRuntimeService) deleteCachedTableByID(tableID int) {
	if s.tableCache == nil {
		return
	}
	cached, err := s.tableCache.Get(strconv.Itoa(tableID))
	if err != nil {
		if !errors.Is(err, cache.ErrCacheMiss) {
			zap.L().Error("read metadata cache before invalidation failed", zap.Int("table_id", tableID), zap.Error(err))
		}
		return
	}
	s.deleteCachedTable(cached)
}

func (s *MetadataRuntimeService) deleteCachedTable(table model.SysTable) {
	if table.Id == 0 {
		return
	}
	if s.tableCache != nil {
		if err := s.tableCache.Delete(strconv.Itoa(table.Id)); err != nil {
			zap.L().Error("delete metadata cache by id failed", zap.Int("table_id", table.Id), zap.Error(err))
		}
		if strings.TrimSpace(table.TableCode) != "" {
			if err := s.tableCache.Delete(table.TableCode); err != nil {
				zap.L().Error("delete metadata cache by code failed", zap.String("table_code", table.TableCode), zap.Error(err))
			}
		}
	}
	if s.fieldCache == nil {
		return
	}
	for _, field := range table.TableFields {
		if field.Id > 0 {
			if err := s.fieldCache.Delete(strconv.Itoa(field.Id)); err != nil {
				zap.L().Error("delete metadata field cache failed", zap.Int("field_id", field.Id), zap.Error(err))
			}
		}
	}
}

func (s *MetadataRuntimeService) setCachedTable(table model.SysTable) {
	if table.Id == 0 {
		return
	}
	if s.tableCache != nil {
		if err := s.tableCache.Set(strconv.Itoa(table.Id), table); err != nil {
			zap.L().Error("set metadata cache by id failed", zap.Int("table_id", table.Id), zap.Error(err))
		}
		if strings.TrimSpace(table.TableCode) != "" {
			if err := s.tableCache.Set(table.TableCode, table); err != nil {
				zap.L().Error("set metadata cache by code failed", zap.String("table_code", table.TableCode), zap.Error(err))
			}
		}
	}
	if s.fieldCache == nil {
		return
	}
	for _, field := range table.TableFields {
		if field.Id > 0 {
			if err := s.fieldCache.Set(strconv.Itoa(field.Id), field); err != nil {
				zap.L().Error("set metadata field cache failed", zap.Int("field_id", field.Id), zap.Error(err))
			}
		}
	}
}

var _ platformmetadata.RuntimeReader = (*MetadataRuntimeService)(nil)
var _ datapermission.MetadataFieldReader = (*MetadataRuntimeService)(nil)
var _ MetadataSecurityReader = (*MetadataRuntimeService)(nil)
