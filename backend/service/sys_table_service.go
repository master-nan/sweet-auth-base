/**
 * @Author: Nan
 * @Date: 2024/5/17 上午11:30
 */

package service

import (
	"backend/dto/request"
	"backend/enum"
	myerrors "backend/internal/errors"
	platformmetadata "backend/internal/metadata"
	"backend/internal/security"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SysTableService 管理Table、Field、Relation、Index及其DDL/View生命周期。
// Runtime读取由MetadataRuntimeService负责，低代码菜单发布由LowCodePublicationService负责。
type SysTableService struct {
	sysTableRepo           repository.SysTableRepository
	sysTableFieldRepo      repository.SysTableFieldRepository
	sysTableIndexRepo      repository.SysTableIndexRepository
	sysTableIndexFieldRepo repository.SysTableIndexFieldRepository
	sysTableRelationRepo   repository.SysTableRelationRepository
	sf                     *utils.Snowflake
	metadataRuntime        *MetadataRuntimeService
}

func NewSysTableService(
	sysTableRepo repository.SysTableRepository,
	sysTableFieldRepo repository.SysTableFieldRepository,
	sysTableIndexRepo repository.SysTableIndexRepository,
	sysTableIndexFieldRepo repository.SysTableIndexFieldRepository,
	sysTableRelationRepo repository.SysTableRelationRepository,
	sf *utils.Snowflake,
	metadataRuntime *MetadataRuntimeService,
) *SysTableService {
	return &SysTableService{
		sysTableRepo,
		sysTableFieldRepo,
		sysTableIndexRepo,
		sysTableIndexFieldRepo,
		sysTableRelationRepo,
		sf,
		metadataRuntime,
	}
}

func (s *SysTableService) GetTableById(id int) (model.SysTable, error) {
	return s.metadataRuntime.configTableByID(context.Background(), id)
}

func (s *SysTableService) GetTableByTableCode(code string) (model.SysTable, error) {
	return s.metadataRuntime.configTableByCode(context.Background(), code)
}

func (s *SysTableService) CreateTable(ctx context.Context, req request.TableCreateReq) error {
	tableCode, err := normalizeDBIdentifier("表编码", req.TableCode)
	if err != nil {
		return err
	}
	req.TableCode = tableCode
	if err = validateMetadataTableType(req.TableType); err != nil {
		return err
	}
	masterDetailMode, ok := enum.NormalizeSysMasterDetailMode(string(req.MasterDetailMode))
	if !ok {
		return myerrors.NewParameterError("主子表展示模式不合法")
	}
	req.MasterDetailMode = masterDetailMode
	formOpenMode, ok := enum.NormalizeSysFormOpenMode(string(req.FormOpenMode))
	if !ok {
		return myerrors.NewParameterError("表单打开方式不合法")
	}
	req.FormOpenMode = formOpenMode
	detailOpenMode, ok := enum.NormalizeSysDetailOpenMode(string(req.DetailOpenMode))
	if !ok {
		return myerrors.NewParameterError("详情打开方式不合法")
	}
	req.DetailOpenMode = detailOpenMode
	var data model.SysTable
	table, e := s.GetTableByTableCode(req.TableCode)
	if e != nil {
		return e
	}
	if table.Id != 0 {
		return myerrors.ErrTableExist
	}
	if req.TableType == enum.View {
		req.SQL, err = validateMetadataViewSQL(req.SQL)
		if err != nil {
			return err
		}
	} else if strings.TrimSpace(req.SQL) != "" {
		return myerrors.NewValidationError("普通表不允许配置视图SQL")
	}
	err = copier.Copy(&data, &req)
	if err != nil {
		zap.L().Error("结构体字段映射失败", zap.String("target", "SysTable"), zap.Error(err))
		return err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	data.Id = int(id)

	if req.TableType == enum.View && strings.TrimSpace(req.SQL) != "" {
		data.TableType = enum.View
		data.SQL = req.SQL
		data.ParentId = req.ParentId
		if err := RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
			if s.sysTableRepo.HasPhysicalTable(tx, req.TableCode) {
				return myerrors.ErrTableExist
			}
			if e := s.sysTableRepo.CreateView(tx, req.TableCode, req.SQL); e != nil {
				return e
			}
			columns, e := s.sysTableRepo.FetchTableMetadata(ctx, tx, "public", req.TableCode)
			if e != nil {
				return e
			}
			fields, e := convertColumnsToSysTableFields(req.TableCode, columns)
			if e != nil {
				return e
			}
			for i := range fields {
				fieldId, err := s.sf.GenerateUniqueID()
				if err != nil {
					return err
				}
				fields[i].Id = int(fieldId)
				fields[i].TableId = data.Id
			}
			tableRecord := data
			tableRecord.TableFields = nil
			if e := s.sysTableRepo.Create(tx, &tableRecord); e != nil {
				return e
			}
			if e := s.sysTableFieldRepo.Create(tx, &fields); e != nil {
				return e
			}
			return nil
		}); err != nil {
			return err
		}
		s.RefreshCache(data.Id)
		return nil
	}
	// 自动在sys_table_field中为Basic结构体中的每个字段创建记录
	fields := newBaseTableFields(data.Id)
	for i := range fields {
		fieldId, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		fields[i].Id = int(fieldId)
	}
	return RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if s.sysTableRepo.HasPhysicalTable(tx, data.TableCode) {
			return myerrors.ErrTableExist
		}
		tableRecord := data
		tableRecord.TableFields = nil
		if e := s.sysTableRepo.Create(tx, &tableRecord); e != nil {
			return e
		}
		if e := s.sysTableFieldRepo.Create(tx, &fields); e != nil {
			return e
		}
		// 创建实例
		dynamicModel := s.sysTableRepo.Model(fields)
		if e := s.sysTableRepo.CreateTable(tx, data.TableCode, dynamicModel); e != nil {
			return e
		}
		return s.syncTableFieldComments(tx, data.TableCode, fields)
	})
}

func (s *SysTableService) syncTableFieldComments(tx *gorm.DB, tableCode string, fields []model.SysTableField) error {
	for _, field := range fields {
		if !s.sysTableRepo.HasTableColumn(tx, tableCode, field.FieldCode) {
			continue
		}
		if err := s.sysTableRepo.SetTableColumnComment(tx, tableCode, field.FieldCode, field.FieldName); err != nil {
			return err
		}
	}
	return nil
}

func newBaseTableFields(tableID int) []model.SysTableField {
	fields := []model.SysTableField{
		{TableId: tableID, FieldName: "id", FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true, IsNull: false, InputType: enum.InputNumberInputType, IsSort: true, Sequence: 1, IsListShow: true},
		{TableId: tableID, FieldName: "创建时间", FieldCode: "gmt_create", FieldType: enum.DatetimeFieldType, IsNull: false, InputType: enum.DatetimePickerInputType, IsSort: true, Sequence: 2, IsListShow: true},
		{TableId: tableID, FieldName: "创建者", FieldCode: "gmt_create_user", FieldType: enum.BigIntFieldType, IsNull: false, InputType: enum.InputNumberInputType, Sequence: 3, IsListShow: true},
		{TableId: tableID, FieldName: "修改时间", FieldCode: "gmt_modify", FieldType: enum.DatetimeFieldType, IsNull: false, InputType: enum.DatetimePickerInputType, IsSort: true, Sequence: 4, IsListShow: true},
		{TableId: tableID, FieldName: "修改者", FieldCode: "gmt_modify_user", FieldType: enum.BigIntFieldType, IsNull: false, InputType: enum.InputNumberInputType, Sequence: 5, IsListShow: true},
		{TableId: tableID, FieldName: "删除时间", FieldCode: "gmt_delete", FieldType: enum.DatetimeFieldType, IsNull: true, InputType: enum.DatetimePickerInputType, Sequence: 6},
		{TableId: tableID, FieldName: "删除者", FieldCode: "gmt_delete_user", FieldType: enum.BigIntFieldType, IsNull: true, InputType: enum.InputNumberInputType, Sequence: 7},
		{TableId: tableID, FieldName: "状态", FieldCode: "state", FieldType: enum.BooleanFieldType, IsNull: false, InputType: enum.SelectInputType, IsSort: true, DefaultValue: utils.StringPtr("true"), DictCode: utils.StringPtr("whether"), Sequence: 8, IsListShow: true},
	}
	for i := range fields {
		applyManagedFieldDefaults(&fields[i])
	}
	return fields
}

func (s *SysTableService) UpdateTable(ctx context.Context, req request.TableUpdateReq) error {
	if strings.TrimSpace(req.TableCode) != "" {
		tableCode, err := normalizeDBIdentifier("表编码", req.TableCode)
		if err != nil {
			return err
		}
		req.TableCode = tableCode
	}
	current, err := s.GetTableById(req.Id)
	if err != nil {
		return err
	}
	if current.Id == 0 {
		return myerrors.ErrDataNotFound
	}

	updateReq := req
	if updateReq.TableCode == "" {
		updateReq.TableCode = current.TableCode
	}
	if updateReq.TableCode != current.TableCode {
		return myerrors.NewValidationError("表编码是跨模块稳定标识，创建后不可修改")
	}
	if updateReq.TableType == 0 {
		updateReq.TableType = current.TableType
	}
	if err = validateMetadataTableType(updateReq.TableType); err != nil {
		return err
	}
	if strings.TrimSpace(string(updateReq.MasterDetailMode)) == "" {
		masterDetailMode, _ := enum.NormalizeSysMasterDetailMode(string(current.MasterDetailMode))
		updateReq.MasterDetailMode = masterDetailMode
	} else {
		masterDetailMode, ok := enum.NormalizeSysMasterDetailMode(string(updateReq.MasterDetailMode))
		if !ok {
			return myerrors.NewParameterError("主子表展示模式不合法")
		}
		updateReq.MasterDetailMode = masterDetailMode
	}
	if strings.TrimSpace(string(updateReq.FormOpenMode)) == "" {
		formOpenMode, _ := enum.NormalizeSysFormOpenMode(string(current.FormOpenMode))
		updateReq.FormOpenMode = formOpenMode
	} else {
		formOpenMode, ok := enum.NormalizeSysFormOpenMode(string(updateReq.FormOpenMode))
		if !ok {
			return myerrors.NewParameterError("表单打开方式不合法")
		}
		updateReq.FormOpenMode = formOpenMode
	}
	if strings.TrimSpace(string(updateReq.DetailOpenMode)) == "" {
		detailOpenMode, _ := enum.NormalizeSysDetailOpenMode(string(current.DetailOpenMode))
		updateReq.DetailOpenMode = detailOpenMode
	} else {
		detailOpenMode, ok := enum.NormalizeSysDetailOpenMode(string(updateReq.DetailOpenMode))
		if !ok {
			return myerrors.NewParameterError("详情打开方式不合法")
		}
		updateReq.DetailOpenMode = detailOpenMode
	}
	if updateReq.ParentId == 0 {
		updateReq.ParentId = current.ParentId
	}
	if updateReq.SQL == "" {
		updateReq.SQL = current.SQL
	}

	if updateReq.TableType == enum.View {
		validatedSQL, validationErr := validateMetadataViewSQL(updateReq.SQL)
		if validationErr != nil {
			return validationErr
		}
		updateReq.SQL = validatedSQL
		sqlChanged := strings.TrimSpace(current.SQL) != validatedSQL
		err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
			if sqlChanged {
				if e := s.sysTableRepo.CreateView(tx, updateReq.TableCode, updateReq.SQL); e != nil {
					return e
				}
				if e := s.syncViewTableFields(ctx, tx, current); e != nil {
					return e
				}
			}
			if e := s.sysTableRepo.Update(tx, &updateReq, updateReq.Id); e != nil {
				return e
			}
			return nil
		})
		if err != nil {
			return err
		}
		s.metadataRuntime.deleteCachedTable(current)
		s.RefreshCache(req.Id)
		return nil
	}
	if strings.TrimSpace(updateReq.SQL) != "" {
		return myerrors.NewValidationError("普通表不允许配置视图SQL")
	}

	tx := s.sysTableRepo.DBWithContext(ctx)
	err = s.sysTableRepo.Update(tx, &updateReq, updateReq.Id)
	if err != nil {
		return err
	}
	// 刷新缓存
	s.metadataRuntime.deleteCachedTable(current)
	s.RefreshCache(req.Id)
	return nil
}

func (s *SysTableService) DeleteTableById(ctx context.Context, id int) error {
	err := RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if e := s.sysTableRepo.DeleteById(tx, id); e != nil {
			return e
		}
		// 删除字段元数据
		if e := s.sysTableFieldRepo.DeleteByField(tx, "table_id", id); e != nil {
			return e
		}
		// 查询表所有索引
		tableIndexes, e := s.sysTableIndexRepo.FindListByFieldWithDB(tx, "table_id", id)
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		// 删除索引信息
		if e := s.sysTableIndexRepo.DeleteByField(tx, "table_id", id); e != nil {
			return e
		}
		var indexIDs []int
		for _, index := range tableIndexes {
			indexIDs = append(indexIDs, index.Id)
		}
		// 删除索引中间表信息，需要使用 IN 查询
		if len(indexIDs) > 0 {
			slice := utils.ToInterfaceSlice(indexIDs)
			if e := s.sysTableIndexFieldRepo.DeleteByFieldIn(tx, "index_id", slice); e != nil {
				return e
			}
		}
		// 查询关联表数据
		relations, e := s.sysTableRelationRepo.FindListByFieldWithDB(tx, "table_id", id)
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		for _, relation := range relations {
			// 删除关联关系表
			if e := s.sysTableRelationRepo.DeleteById(tx, relation.Id); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.DeleteCache(id)
	return nil
}

func (s *SysTableService) GetTableFieldById(id int) (model.SysTableField, error) {
	return s.metadataRuntime.configFieldByID(context.Background(), id)
}

func (s *SysTableService) GetTableFieldsByTableId(tableId int) ([]model.SysTableField, error) {
	table, err := s.metadataRuntime.configTableByID(context.Background(), tableId)
	if err != nil {
		return nil, err
	}
	return table.TableFields, nil
}

func (s *SysTableService) CreateTableField(ctx context.Context, req request.TableFieldCreateReq) error {
	fieldCode, err := normalizeDBIdentifier("字段编码", req.FieldCode)
	if err != nil {
		return err
	}
	req.FieldCode = fieldCode
	fields, e := s.GetTableFieldsByTableId(req.TableId)
	if e != nil {
		return e
	}
	for _, field := range fields {
		if field.FieldCode == req.FieldCode {
			return myerrors.ErrTableFieldExist
		}
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	table, err := s.GetTableById(req.TableId)
	if err != nil {
		return err
	}
	if table.TableType == enum.View {
		return myerrors.ErrTableViewFieldNoAdd
	}
	linkageConfig, err := s.normalizeTableFieldLinkageConfig(req.LinkageConfig, table, req.FieldCode)
	if err != nil {
		return err
	}
	req.LinkageConfig = linkageConfig
	var data model.SysTableField
	err = copier.Copy(&data, &req)
	if err != nil {
		zap.L().Error("Error during struct mapping:", zap.Error(err))
		return err
	}
	data.Expression = optionalMetadataString(req.Expression)
	data.LinkageConfig = optionalMetadataString(req.LinkageConfig)
	data.DefaultValue = optionalMetadataString(req.DefaultValue)
	data.DictCode = optionalMetadataString(req.DictCode)
	if err = validateMetadataFieldDefinition(&data, req.Sequence); err != nil {
		return err
	}
	if err = s.validateRelationDisplayContract(data, table); err != nil {
		return err
	}
	data.Id = int(id)
	err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		// 构建SQL类型字符串，包括长度、默认值、是否可为空和备注
		sqlType := buildColumnSQLTypeFromField(data)
		if e := s.sysTableRepo.CreateTableColumn(tx, table.TableCode, data.FieldCode, sqlType); e != nil {
			return e
		}
		if e := s.sysTableFieldRepo.Create(tx, &data); e != nil {
			return e
		}
		return s.syncTableFieldComments(tx, table.TableCode, []model.SysTableField{data})
	})
	if err != nil {
		return err
	}
	s.RefreshCache(table.Id)
	return nil
}

func (s *SysTableService) UpdateTableField(ctx context.Context, req request.TableFieldUpdateReq) error {
	fieldCode, err := normalizeDBIdentifier("字段编码", req.FieldCode)
	if err != nil {
		return err
	}
	req.FieldCode = fieldCode
	candidate := model.SysTableField{
		TableId:            req.TableId,
		FieldName:          req.FieldName,
		FieldCode:          req.FieldCode,
		FieldType:          req.FieldType,
		FieldLength:        req.FieldLength,
		FieldDecimalLength: req.FieldDecimalLength,
		NumericPrecision:   req.NumericPrecision,
		NumericScale:       req.NumericScale,
		LogicalType:        req.LogicalType,
		DisplayFormat:      req.DisplayFormat,
		ListWidth:          req.ListWidth,
		InputType:          req.InputType,
		FormSpan:           req.FormSpan,
		DetailSpan:         req.DetailSpan,
		DefaultValue:       optionalMetadataString(req.DefaultValue),
		DictCode:           optionalMetadataString(req.DictCode),
		IsPrimaryKey:       req.IsPrimaryKey,
		IsIndex:            req.IsIndex,
		IsQuickSearch:      req.IsQuickSearch,
		IsAdvancedSearch:   req.IsAdvancedSearch,
		IsSort:             req.IsSort,
		IsNull:             req.IsNull,
		IsListShow:         req.IsListShow,
		IsInsertShow:       req.IsInsertShow,
		IsUpdateShow:       req.IsUpdateShow,
		Sequence:           uint8(req.Sequence),
		OriginalFieldId:    req.OriginalFieldId,
		Binding:            req.Binding,
		FieldCategory:      req.FieldCategory,
		Expression:         optionalMetadataString(req.Expression),
		LinkageConfig:      optionalMetadataString(req.LinkageConfig),
	}
	if err = validateMetadataFieldDefinition(&candidate, req.Sequence); err != nil {
		return err
	}
	req.NumericPrecision = candidate.NumericPrecision
	req.NumericScale = candidate.NumericScale
	req.LogicalType = candidate.LogicalType
	req.DisplayFormat = candidate.DisplayFormat
	table, err := s.GetTableById(req.TableId)
	if err != nil {
		return err
	}
	if table.Id != 0 {
		linkageConfig, err := s.normalizeTableFieldLinkageConfig(req.LinkageConfig, table, req.FieldCode)
		if err != nil {
			return err
		}
		req.LinkageConfig = linkageConfig
		candidate.LinkageConfig = optionalMetadataString(linkageConfig)
		if err = s.validateRelationDisplayContract(candidate, table); err != nil {
			return err
		}
		fields, e := s.GetTableFieldsByTableId(req.TableId)
		if e != nil {
			return e
		}
		var data model.SysTableField
		for _, field := range fields {
			if field.Id == req.Id {
				diff := cmp.Diff(req, field)
				if diff == "" {
					return myerrors.ErrTableFieldNoChange
				}
				zap.L().Info("变化值：", zap.String("diff", diff))
				data = field
				break
			}
		}
		if table.TableType == enum.View {
			existingDefault := ""
			if data.DefaultValue != nil {
				existingDefault = *data.DefaultValue
			}
			if req.FieldCode != data.FieldCode ||
				req.FieldType != data.FieldType ||
				req.FieldLength != data.FieldLength ||
				req.FieldDecimalLength != data.FieldDecimalLength ||
				req.NumericPrecision != data.NumericPrecision ||
				req.NumericScale != data.NumericScale ||
				req.DefaultValue != existingDefault ||
				req.IsNull != data.IsNull {
				return myerrors.ErrTableViewFieldNoAdd
			}
			if err := s.sysTableFieldRepo.Update(s.sysTableRepo.DBWithContext(ctx), &req, req.Id); err != nil {
				return err
			}
			s.RefreshCache(table.Id)
			return nil
		}
		err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
			// 构建完整的字段SQL类型，避免缺少类型导致SQL错误
			columnNeedsAlter := false
			if req.FieldType != data.FieldType {
				columnNeedsAlter = true
			}
			if req.FieldLength != data.FieldLength || req.FieldDecimalLength != data.FieldDecimalLength ||
				req.NumericPrecision != data.NumericPrecision || req.NumericScale != data.NumericScale {
				columnNeedsAlter = true
			}
			if !cmp.Equal(optionalMetadataString(req.DefaultValue), data.DefaultValue) {
				columnNeedsAlter = true
			}
			if req.IsNull != data.IsNull {
				columnNeedsAlter = true
			}
			if req.FieldName != "" && req.FieldName != data.FieldName {
				columnNeedsAlter = true
			}
			if req.FieldCode != data.FieldCode {
				columnNeedsAlter = true
			}

			if columnNeedsAlter {
				sqlType := buildColumnSQLType(req, data)
				if req.FieldCode == data.FieldCode {
					if e := s.sysTableRepo.ModifyTableColumn(tx, table.TableCode, req.FieldCode, sqlType); e != nil {
						return e
					}
				} else {
					if err := s.sysTableRepo.ChangeTableColumn(tx, table.TableCode, data.FieldCode, req.FieldCode, sqlType); err != nil {
						return err
					}
				}
			}
			if e := s.sysTableFieldRepo.Update(tx, &req, req.Id); e != nil {
				return e
			}
			if _, e := s.sysTableFieldRepo.UpdateFields(tx, req.Id, map[string]any{
				"default_value": optionalMetadataString(req.DefaultValue),
			}); e != nil {
				return e
			}
			return s.syncTableFieldComments(tx, table.TableCode, []model.SysTableField{{
				FieldCode: req.FieldCode,
				FieldName: req.FieldName,
			}})
		})
		if err != nil {
			return err
		}
		s.RefreshCache(table.Id)
		return nil
	}
	return myerrors.ErrDataNotFound
}

func optionalMetadataString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

const maxDBIdentifierLength = 63

func validateMetadataTableType(tableType enum.SysTableType) error {
	if tableType != enum.System && tableType != enum.View {
		return myerrors.NewValidationError("表类型不合法")
	}
	return nil
}

func validateMetadataViewSQL(raw string) (string, error) {
	query, err := platformmetadata.ValidateReadOnlyQuery(raw)
	if errors.Is(err, platformmetadata.ErrReadOnlyQueryEmpty) {
		return "", myerrors.ErrTableViewSQLEmpty
	}
	if err != nil {
		return "", myerrors.NewValidationError("视图仅允许单条SELECT/WITH只读查询")
	}
	return query, nil
}

func normalizeDBIdentifier(name, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", myerrors.NewValidationError(fmt.Sprintf("%s不能为空", name))
	}
	if len(trimmed) > maxDBIdentifierLength {
		return "", myerrors.NewValidationError(fmt.Sprintf("%s长度不能超过%d", name, maxDBIdentifierLength))
	}
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_') {
			return "", myerrors.NewValidationError(fmt.Sprintf("%s只能包含字母、数字、下划线", name))
		}
		if i == 0 && ch >= '0' && ch <= '9' {
			return "", myerrors.NewValidationError(fmt.Sprintf("%s不能以数字开头", name))
		}
	}
	return trimmed, nil
}

func normalizeOptionalDBIdentifier(name, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	return normalizeDBIdentifier(name, trimmed)
}

func validateTableIndexFields(table model.SysTable, indexFields []request.TableIndexFieldReq) ([]request.TableIndexFieldReq, []string, error) {
	if table.Id == 0 {
		return nil, nil, myerrors.ErrDataNotFound
	}
	if len(indexFields) == 0 {
		return nil, nil, myerrors.NewValidationError("索引字段不能为空")
	}
	fieldByID := make(map[int]model.SysTableField, len(table.TableFields))
	for _, field := range table.TableFields {
		fieldByID[field.Id] = field
	}
	seen := make(map[int]struct{}, len(indexFields))
	normalizedFields := make([]request.TableIndexFieldReq, 0, len(indexFields))
	fieldCodeList := make([]string, 0, len(indexFields))
	for _, indexField := range indexFields {
		if indexField.TableId != table.Id {
			return nil, nil, myerrors.NewValidationError("索引字段不属于当前表")
		}
		fieldCode, err := normalizeDBIdentifier("索引字段编码", indexField.FieldCode)
		if err != nil {
			return nil, nil, err
		}
		field, ok := fieldByID[indexField.FieldId]
		if !ok {
			return nil, nil, myerrors.NewValidationError("索引字段不存在")
		}
		if field.FieldCode != fieldCode {
			return nil, nil, myerrors.NewValidationError("索引字段ID和字段编码不匹配")
		}
		if _, ok := seen[indexField.FieldId]; ok {
			return nil, nil, myerrors.NewValidationError("索引字段不能重复")
		}
		seen[indexField.FieldId] = struct{}{}
		normalizedFields = append(normalizedFields, request.TableIndexFieldReq{
			TableId:   table.Id,
			FieldId:   indexField.FieldId,
			FieldCode: fieldCode,
		})
		fieldCodeList = append(fieldCodeList, fieldCode)
	}
	return normalizedFields, fieldCodeList, nil
}

type tableFieldLinkageEnvelope struct {
	Linkage *tableFieldLinkageConfig `json:"linkage"`
}

type tableFieldLinkageConfig struct {
	Enabled       bool              `json:"enabled"`
	Mode          string            `json:"mode"`
	TableCode     string            `json:"tableCode"`
	LabelKey      string            `json:"labelKey"`
	ValueKey      string            `json:"valueKey"`
	ParentKey     string            `json:"parentKey"`
	ChildrenKey   string            `json:"childrenKey"`
	FilterMapping map[string]string `json:"filterMapping"`
	PageSize      int               `json:"pageSize"`
	Selectable    string            `json:"selectable"`
	ShowPath      *bool             `json:"showPath"`
	RootValue     interface{}       `json:"rootValue"`
}

func (s *SysTableService) normalizeTableFieldLinkageConfig(raw string, currentTable model.SysTable, currentFieldCode string) (string, error) {
	return normalizeTableFieldLinkageConfig(raw, currentTable, currentFieldCode, s.resolveLinkageRelatedTable)
}

func (s *SysTableService) resolveLinkageRelatedTable(cfg tableFieldLinkageConfig) (model.SysTable, error) {
	return s.GetTableByTableCode(strings.TrimSpace(cfg.TableCode))
}

func normalizeTableFieldLinkageConfig(raw string, currentTable model.SysTable, currentFieldCode string, resolveTable func(tableFieldLinkageConfig) (model.SysTable, error)) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	var envelope tableFieldLinkageEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return "", myerrors.NewValidationError("联动配置JSON格式不正确")
	}
	if envelope.Linkage == nil || !envelope.Linkage.Enabled {
		return raw, nil
	}
	cfg := *envelope.Linkage
	switch cfg.Mode {
	case "relation", "cascader":
	default:
		return "", myerrors.NewValidationError("联动配置mode仅支持relation或cascader")
	}
	if strings.TrimSpace(cfg.TableCode) == "" {
		return "", myerrors.NewValidationError("联动配置必须指定tableCode")
	}
	if cfg.PageSize < 0 || cfg.PageSize > 1000 {
		return "", myerrors.NewValidationError("联动配置pageSize必须在0到1000之间")
	}
	relatedTable, err := resolveTable(cfg)
	if err != nil {
		return "", err
	}
	if relatedTable.Id == 0 {
		return "", myerrors.NewValidationError("联动配置关联表不存在")
	}
	currentFields := tableFieldByCode(currentTable.TableFields)
	if strings.TrimSpace(currentFieldCode) != "" {
		currentFields[currentFieldCode] = model.SysTableField{FieldCode: currentFieldCode}
	}
	relatedFields := tableFieldByCode(relatedTable.TableFields)
	if strings.TrimSpace(currentFieldCode) != "" &&
		((relatedTable.Id != 0 && relatedTable.Id == currentTable.Id) ||
			(strings.TrimSpace(relatedTable.TableCode) != "" && relatedTable.TableCode == currentTable.TableCode)) {
		relatedFields[currentFieldCode] = model.SysTableField{FieldCode: currentFieldCode}
	}
	if strings.TrimSpace(cfg.LabelKey) == "" || strings.TrimSpace(cfg.ValueKey) == "" {
		return "", myerrors.NewValidationError("关系展示必须配置labelKey和valueKey")
	}
	if err := validateOptionalLinkageField("labelKey", cfg.LabelKey, relatedFields); err != nil {
		return "", err
	}
	if err := validateOptionalLinkageField("valueKey", cfg.ValueKey, relatedFields); err != nil {
		return "", err
	}
	if err := validateOptionalLinkageField("parentKey", cfg.ParentKey, relatedFields); err != nil {
		return "", err
	}
	valueField := relatedFields[strings.TrimSpace(cfg.ValueKey)]
	labelField := relatedFields[strings.TrimSpace(cfg.LabelKey)]
	if security.IsSensitiveFieldName(valueField.FieldCode) || security.IsSensitiveFieldName(labelField.FieldCode) {
		return "", myerrors.NewValidationError("关系展示不允许使用敏感字段")
	}
	if parentKey := strings.TrimSpace(cfg.ParentKey); parentKey != "" && security.IsSensitiveFieldName(parentKey) {
		return "", myerrors.NewValidationError("关系展示不允许使用敏感父级字段")
	}
	if !relationValueFieldUnique(relatedTable, valueField) {
		return "", myerrors.NewValidationError("关系取值字段必须是主键或单字段唯一索引")
	}
	if cfg.Mode == "cascader" && strings.TrimSpace(cfg.ParentKey) == strings.TrimSpace(cfg.ValueKey) {
		return "", myerrors.NewValidationError("级联配置父级字段不能和取值字段相同")
	}
	for targetField, sourceField := range cfg.FilterMapping {
		targetField = strings.TrimSpace(targetField)
		sourceField = strings.TrimSpace(sourceField)
		if targetField == "" || sourceField == "" {
			return "", myerrors.NewValidationError("联动配置filterMapping不能为空")
		}
		if _, ok := relatedFields[targetField]; !ok {
			return "", myerrors.NewValidationError(fmt.Sprintf("联动配置filterMapping目标字段%s不存在", targetField))
		}
		if security.IsSensitiveFieldName(targetField) {
			return "", myerrors.NewValidationError("关系展示不允许使用敏感过滤字段")
		}
		if _, ok := currentFields[sourceField]; !ok {
			return "", myerrors.NewValidationError(fmt.Sprintf("联动配置filterMapping源字段%s不存在", sourceField))
		}
	}
	if strings.TrimSpace(cfg.TableCode) != "" || strings.TrimSpace(relatedTable.TableCode) == "" {
		return raw, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", myerrors.NewValidationError("联动配置JSON格式不正确")
	}
	linkage, ok := payload["linkage"].(map[string]interface{})
	if !ok {
		return raw, nil
	}
	linkage["tableCode"] = relatedTable.TableCode
	normalized, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}

func validateOptionalLinkageField(name, fieldCode string, fields map[string]model.SysTableField) error {
	fieldCode = strings.TrimSpace(fieldCode)
	if fieldCode == "" {
		return nil
	}
	if _, ok := fields[fieldCode]; !ok {
		return myerrors.NewValidationError(fmt.Sprintf("联动配置%s字段%s不存在", name, fieldCode))
	}
	return nil
}

func tableFieldByCode(fields []model.SysTableField) map[string]model.SysTableField {
	result := make(map[string]model.SysTableField, len(fields))
	for _, field := range fields {
		result[field.FieldCode] = field
	}
	return result
}

func relationValueFieldUnique(table model.SysTable, field model.SysTableField) bool {
	if field.IsPrimaryKey || field.FieldCode == "id" {
		return true
	}
	for _, index := range table.TableIndexes {
		if index.IsUnique && len(index.IndexFields) == 1 && index.IndexFields[0].FieldCode == field.FieldCode {
			return true
		}
	}
	return false
}

func (s *SysTableService) validateRelationDisplayContract(field model.SysTableField, currentTable model.SysTable) error {
	if field.LinkageConfig == nil || strings.TrimSpace(*field.LinkageConfig) == "" {
		return nil
	}
	var envelope tableFieldLinkageEnvelope
	if err := json.Unmarshal([]byte(*field.LinkageConfig), &envelope); err != nil || envelope.Linkage == nil || !envelope.Linkage.Enabled {
		return nil
	}
	target, err := s.resolveLinkageRelatedTable(*envelope.Linkage)
	if err != nil {
		return err
	}
	targetField, ok := tableFieldByCode(target.TableFields)[strings.TrimSpace(envelope.Linkage.ValueKey)]
	if !ok {
		return myerrors.NewValidationError("关系取值字段不存在")
	}
	if !platformmetadata.StorageTypesCompatible(field.FieldType, targetField.FieldType) {
		return myerrors.NewValidationError("关系源字段与目标取值字段类型不兼容")
	}
	return nil
}

func buildColumnSQLType(req request.TableFieldUpdateReq, data model.SysTableField) string {
	fieldType := req.FieldType
	length := req.FieldLength
	decimalLength := req.FieldDecimalLength
	if fieldType == enum.DecimalFieldType {
		length, decimalLength = req.NumericPrecision, req.NumericScale
		if length == 0 {
			length, decimalLength = data.NumericPrecision, data.NumericScale
		}
	}
	if length == 0 {
		length = data.FieldLength
	}
	if decimalLength == 0 {
		decimalLength = data.FieldDecimalLength
	}
	defaultValue := req.DefaultValue
	descriptor, _ := platformmetadata.DescribeStorage(fieldType)
	sqlType := descriptor.SQLType
	// 拼接长度（仅对需要长度的类型，BOOLEAN/TEXT/JSON/DATE/DATETIME/TIME 不接受长度参数）
	if length > 0 && descriptor.AcceptsLength {
		if decimalLength > 0 {
			sqlType += fmt.Sprintf("(%d,%d)", length, decimalLength)
		} else {
			sqlType += fmt.Sprintf("(%d)", length)
		}
	}
	defaultSQL := buildDefaultSQL(fieldType, defaultValue)
	if defaultSQL != "" {
		sqlType += defaultSQL
	}
	if req.IsNull {
		sqlType += " NULL"
	} else {
		sqlType += " NOT NULL"
	}
	return sqlType
}

func buildColumnSQLTypeFromField(field model.SysTableField) string {
	descriptor, _ := platformmetadata.DescribeStorage(field.FieldType)
	sqlType := descriptor.SQLType
	length, decimalLength := field.FieldLength, field.FieldDecimalLength
	if field.FieldType == enum.DecimalFieldType {
		length, decimalLength = field.NumericPrecision, field.NumericScale
	}
	if length > 0 && descriptor.AcceptsLength {
		if field.FieldType == enum.DecimalFieldType {
			sqlType += fmt.Sprintf("(%d,%d)", length, decimalLength)
		} else {
			sqlType += fmt.Sprintf("(%d)", length)
		}
	}
	if field.DefaultValue != nil {
		sqlType += buildDefaultSQL(field.FieldType, *field.DefaultValue)
	}
	if field.IsNull {
		sqlType += " NULL"
	} else {
		sqlType += " NOT NULL"
	}
	return sqlType
}

func buildDefaultSQL(fieldType enum.SysTableFieldType, defaultValue string) string {
	if defaultValue == "" {
		return ""
	}

	switch fieldType {
	case enum.BooleanFieldType:
		val := strings.ToLower(strings.TrimSpace(defaultValue))
		if val == "true" || val == "1" {
			return " DEFAULT true"
		}
		if val == "false" || val == "0" {
			return " DEFAULT false"
		}
		return fmt.Sprintf(" DEFAULT '%s'", escapePostgresLiteral(defaultValue))
	case enum.BigIntFieldType, enum.SmallIntFieldType, enum.DecimalFieldType, enum.IntFieldType:
		return fmt.Sprintf(" DEFAULT %s", defaultValue)
	case enum.JsonFieldType:
		return fmt.Sprintf(" DEFAULT '%s'::jsonb", escapePostgresLiteral(defaultValue))
	default:
		return fmt.Sprintf(" DEFAULT '%s'", escapePostgresLiteral(defaultValue))
	}
}

func escapePostgresLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func (s *SysTableService) DeleteTableFieldById(ctx context.Context, id int) error {
	field, err := s.GetTableFieldById(id)
	if err != nil {
		return err
	}
	if field.Id != 0 {
		table, err := s.GetTableById(field.TableId)
		if err != nil {
			return err
		}
		if table.TableType == enum.View {
			return myerrors.ErrTableViewFieldNoDelete
		}
		if table.Id != 0 {
			err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
				if e := s.sysTableFieldRepo.DeleteById(tx, field.Id); e != nil {
					return e
				}
				if e := s.sysTableRepo.DropTableColumn(tx, table.TableCode, field.FieldCode); e != nil {
					return e
				}
				return nil
			})
			if err != nil {
				return err
			}
			// 删除缓存
			s.DeleteCache(table.Id)
			return nil
		}
	}
	return myerrors.ErrDataNotFound
}

func (s *SysTableService) GetTableRelationById(id int) (model.SysTableRelation, error) {
	data, err := s.sysTableRelationRepo.FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysTableRelation{}, nil
		}
		return model.SysTableRelation{}, err
	}
	return data, nil
}

// CreateTableRelation 校验目标表、关系字段和类型兼容性；M:N只创建不存在的中间表。
func (s *SysTableService) CreateTableRelation(ctx context.Context, req request.TableRelationCreateReq) error {
	var err error
	if req.ReferenceKey, err = normalizeDBIdentifier("主表字段", req.ReferenceKey); err != nil {
		return err
	}
	if req.ForeignKey, err = normalizeDBIdentifier("关联表字段", req.ForeignKey); err != nil {
		return err
	}
	if req.ManyTableCode, err = normalizeOptionalDBIdentifier("中间表编码", req.ManyTableCode); err != nil {
		return err
	}
	if err = validateMetadataRelation(req.RelationType, req.ManyTableCode); err != nil {
		return err
	}
	if err = s.validatePhysicalRelationFields(ctx, req.TableId, req.RelatedTableId, req.ReferenceKey, req.ForeignKey); err != nil {
		return err
	}
	err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if req.RelationType == enum.ManyToMany && s.sysTableRepo.HasPhysicalTable(tx, req.ManyTableCode) {
			return myerrors.ErrTableExist
		}
		var data model.SysTableRelation
		e := copier.Copy(&data, &req)
		if e != nil {
			zap.L().Error("Error during struct mapping:", zap.Error(e))
			return e
		}
		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		data.Id = int(id)
		if e := s.sysTableRelationRepo.Create(tx, &data); e != nil {
			return e
		}
		// 如果是多对多 创建对应的表
		if data.RelationType == enum.ManyToMany {
			mainTable, e := s.sysTableRepo.FindByIdWithDB(tx, data.TableId)
			if e != nil {
				return e
			}
			mainTable.TableFields, e = s.sysTableFieldRepo.FindListByFieldWithDB(tx, "table_id", data.TableId)
			if e != nil {
				return e
			}
			relatedTable, e := s.sysTableRepo.FindByIdWithDB(tx, data.RelatedTableId)
			if e != nil {
				return e
			}
			relatedTable.TableFields, e = s.sysTableFieldRepo.FindListByFieldWithDB(tx, "table_id", data.RelatedTableId)
			if e != nil {
				return e
			}
			var referenceKeyField model.SysTableField
			for _, field := range mainTable.TableFields {
				if field.FieldCode == data.ReferenceKey {
					referenceKeyField = field
					referenceKeyField.Tag = utils.StringPtr(`gorm:"primaryKey;autoIncrement:false"`)
				}
			}
			var foreignKeyField model.SysTableField
			for _, field := range relatedTable.TableFields {
				if field.FieldCode == data.ForeignKey {
					foreignKeyField = field
					foreignKeyField.Tag = utils.StringPtr(`gorm:"primaryKey;autoIncrement:false"`)
				}
			}
			if referenceKeyField.Id == 0 || foreignKeyField.Id == 0 {
				return myerrors.ErrDataNotFound
			}

			var relationFields []model.SysTableField
			relationFields = append(relationFields, referenceKeyField, foreignKeyField)
			relationModel := s.sysTableRepo.Model(relationFields)
			return s.sysTableRepo.CreateTable(tx, data.ManyTableCode, relationModel)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.RefreshCache(req.TableId)
	return nil
}

// UpdateTableRelation 只允许不改变物理签名的Metadata更新；
// Relation类型、两端Key或中间表变化必须通过显式Migration处理。
func (s *SysTableService) UpdateTableRelation(ctx context.Context, req request.TableRelationUpdateReq) error {
	oldRelation, err := s.sysTableRelationRepo.FindByIdWithDB(s.sysTableRelationRepo.DBWithContext(ctx), req.Id)
	if err != nil {
		return err
	}
	if oldRelation.Id == 0 {
		return myerrors.ErrDataNotFound
	}
	if req.ReferenceKey, err = normalizeDBIdentifier("主表字段", req.ReferenceKey); err != nil {
		return err
	}
	if req.ForeignKey, err = normalizeDBIdentifier("关联表字段", req.ForeignKey); err != nil {
		return err
	}
	if req.ManyTableCode, err = normalizeOptionalDBIdentifier("中间表编码", req.ManyTableCode); err != nil {
		return err
	}
	if err = validateMetadataRelation(req.RelationType, req.ManyTableCode); err != nil {
		return err
	}
	if err = s.validatePhysicalRelationFields(ctx, req.TableId, req.RelatedTableId, req.ReferenceKey, req.ForeignKey); err != nil {
		return err
	}

	if relationPhysicalSignatureChanged(oldRelation, req) &&
		(oldRelation.RelationType == enum.ManyToMany || req.RelationType == enum.ManyToMany) {
		return myerrors.ErrTableRelationMigration
	}

	err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		return s.sysTableRelationRepo.Update(tx, &req, req.Id)
	})
	if err != nil {
		return err
	}
	s.RefreshCache(req.TableId)
	return nil
}

func relationPhysicalSignatureChanged(old model.SysTableRelation, next request.TableRelationUpdateReq) bool {
	return old.TableId != next.TableId ||
		old.RelatedTableId != next.RelatedTableId ||
		old.RelationType != next.RelationType ||
		old.ReferenceKey != next.ReferenceKey ||
		old.ForeignKey != next.ForeignKey ||
		old.ManyTableCode != next.ManyTableCode
}

func validateMetadataRelation(relationType enum.SysTableRelationType, manyTableCode string) error {
	if relationType < enum.OneToOne || relationType > enum.ManyToMany {
		return myerrors.NewValidationError("关系类型不合法")
	}
	if relationType == enum.ManyToMany && manyTableCode == "" {
		return myerrors.NewValidationError("多对多关系必须指定中间表编码")
	}
	if relationType != enum.ManyToMany && manyTableCode != "" {
		return myerrors.NewValidationError("非多对多关系不允许配置中间表")
	}
	return nil
}

// validatePhysicalRelationFields 确认关系两端字段存在、类型兼容，并满足value field唯一性要求。
func (s *SysTableService) validatePhysicalRelationFields(
	ctx context.Context,
	tableID, relatedTableID int,
	referenceKey, foreignKey string,
) error {
	mainTable, err := s.sysTableRepo.GetTableById(ctx, tableID)
	if err != nil {
		return err
	}
	relatedTable, err := s.sysTableRepo.GetTableById(ctx, relatedTableID)
	if err != nil {
		return err
	}
	referenceField, referenceOK := tableFieldByCode(mainTable.TableFields)[referenceKey]
	foreignField, foreignOK := tableFieldByCode(relatedTable.TableFields)[foreignKey]
	if !referenceOK || !foreignOK {
		return myerrors.NewValidationError("关系两端字段必须存在")
	}
	if !platformmetadata.StorageTypesCompatible(referenceField.FieldType, foreignField.FieldType) {
		return myerrors.NewValidationError("关系两端字段类型不兼容")
	}
	return nil
}

// DeleteTableRelationById 只删除Metadata关系，不自动删除可能包含业务数据的中间表。
func (s *SysTableService) DeleteTableRelationById(ctx context.Context, id int) error {
	relation, err := s.sysTableRelationRepo.FindByIdWithDB(s.sysTableRelationRepo.DBWithContext(ctx), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrDataNotFound
		}
		return err
	}
	if relation.Id == 0 {
		return myerrors.ErrDataNotFound
	}
	err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if e := s.sysTableRelationRepo.DeleteById(tx, id); e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 删除关系后必须清掉主表元数据缓存，否则低代码页面会继续读到旧关系。
	s.DeleteCache(relation.TableId)
	return nil
}

func (s *SysTableService) GetTableIndexById(id int) (model.SysTableIndex, error) {
	data, err := s.sysTableIndexRepo.FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysTableIndex{}, nil
		}
		return model.SysTableIndex{}, err
	}
	return data, nil
}

func (s *SysTableService) CreateTableIndex(ctx context.Context, req request.TableIndexCreateReq) error {
	var err error
	if req.IndexName, err = normalizeDBIdentifier("索引名称", req.IndexName); err != nil {
		return err
	}
	table, err := s.GetTableById(req.TableId)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrDataNotFound
	}
	var fieldCodeList []string
	req.IndexFields, fieldCodeList, err = validateTableIndexFields(table, req.IndexFields)
	if err != nil {
		return err
	}
	err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		var data model.SysTableIndex
		e := copier.Copy(&data, &req)
		if e != nil {
			zap.L().Error("Error during struct mapping:", zap.Error(e))
			return e
		}
		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		data.Id = int(id)
		if e := s.sysTableIndexRepo.Create(tx, &data); e != nil {
			return e
		}
		var indexFields []model.SysTableIndexField
		for position, field := range req.IndexFields {
			indexField := model.SysTableIndexField{
				IndexId:  data.Id,
				FieldId:  field.FieldId,
				Sequence: uint8(position + 1),
			}
			indexFields = append(indexFields, indexField)
		}
		if e := s.sysTableIndexFieldRepo.Create(tx, &indexFields); e != nil {
			return e
		}
		fields := strings.Join(fieldCodeList, ",")
		if e := s.sysTableRepo.CreateTableIndex(tx, req.IsUnique, req.IndexName, table.TableCode, fields); e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.RefreshCache(req.TableId)
	return nil
}

func (s *SysTableService) UpdateTableIndex(ctx context.Context, req request.TableIndexUpdateReq) error {
	oldIndex, err := s.sysTableIndexRepo.FindById(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrDataNotFound
		}
		return err
	}
	if oldIndex.Id == 0 {
		return myerrors.ErrDataNotFound
	}
	if oldIndex.TableId != req.TableId {
		return myerrors.NewValidationError("索引不能切换所属表")
	}
	if req.IndexName, err = normalizeDBIdentifier("索引名称", req.IndexName); err != nil {
		return err
	}
	table, err := s.GetTableById(req.TableId)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrDataNotFound
	}
	var fieldCodeList []string
	req.IndexFields, fieldCodeList, err = validateTableIndexFields(table, req.IndexFields)
	if err != nil {
		return err
	}
	err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		// 删除中间表数据
		if e := s.sysTableIndexFieldRepo.DeleteByField(tx, "index_id", req.Id); e != nil {
			return e
		}
		if e := s.sysTableIndexRepo.Update(tx, &req, req.Id); e != nil {
			return e
		}
		// 使用原索引名称删除表索引
		if e := s.sysTableRepo.DropTableIndex(tx, oldIndex.IndexName, table.TableCode); e != nil {
			return e
		}
		var indexFields []model.SysTableIndexField
		for position, field := range req.IndexFields {
			indexField := model.SysTableIndexField{
				IndexId:  req.Id,
				FieldId:  field.FieldId,
				Sequence: uint8(position + 1),
			}
			indexFields = append(indexFields, indexField)
		}
		// 创建中间表数据
		if e := s.sysTableIndexFieldRepo.Create(tx, &indexFields); e != nil {
			return e
		}
		fields := strings.Join(fieldCodeList, ",")
		// 创建表索引
		if err := s.sysTableRepo.CreateTableIndex(tx, req.IsUnique, req.IndexName, table.TableCode, fields); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.RefreshCache(req.TableId)
	return nil
}

func (s *SysTableService) DeleteTableIndexById(ctx context.Context, id int) error {
	index, e := s.sysTableIndexRepo.FindById(id)
	if e != nil {
		return e
	}
	table, e := s.GetTableById(index.TableId)
	if e != nil {
		return e
	}
	err := RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if e := s.sysTableIndexRepo.DeleteById(tx, id); e != nil {
			return e
		}
		// 使用索引名称删除表索引
		if e := s.sysTableRepo.DropTableIndex(tx, index.IndexName, table.TableCode); e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.DeleteCache(table.Id)
	return nil
}

func (s *SysTableService) InitTable(ctx context.Context, tableCode string) error {
	var err error
	tableCode, err = normalizeDBIdentifier("表编码", tableCode)
	if err != nil {
		return err
	}
	if !s.sysTableRepo.HasPhysicalTable(s.sysTableRepo.DBWithContext(ctx), tableCode) {
		return myerrors.ErrTableNotFound
	}
	columns, err := s.sysTableRepo.FetchTableMetadata(ctx, nil, "public", tableCode)
	if err != nil {
		return err
	}
	tableIndexes, err := s.sysTableRepo.FetchTableIndexMetadata(ctx, nil, "public", tableCode)
	if err != nil {
		return err
	}
	fields, err := convertColumnsToSysTableFields(tableCode, columns)
	if err != nil {
		return err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	table, e := s.GetTableByTableCode(tableCode)
	if e != nil {
		return e
	}
	if table.Id != 0 {
		return myerrors.ErrTableInit
	}
	table = model.SysTable{
		Basic: model.Basic{
			Id: int(id),
		},
		TableName: tableCode,
		TableCode: tableCode,
		TableType: enum.System,
	}
	indexesMap := make(map[string]model.SysTableIndex)
	fieldIDs := make(map[string]int, len(fields))
	var indexes []model.SysTableIndex
	var indexFields []model.SysTableIndexField
	for i := range fields {
		fields[i].TableId = table.Id
		fieldId, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		fields[i].Id = int(fieldId)
		fieldIDs[fields[i].FieldCode] = fields[i].Id
	}
	for _, physicalIndex := range tableIndexes {
		fieldID, exists := fieldIDs[physicalIndex.ColumnName]
		if !exists {
			continue
		}
		index, exists := indexesMap[physicalIndex.IndexName]
		if !exists {
			indexID, err := s.sf.GenerateUniqueID()
			if err != nil {
				return err
			}
			index = model.SysTableIndex{
				Basic:   model.Basic{Id: int(indexID)},
				TableId: table.Id, IndexName: physicalIndex.IndexName, IsUnique: !physicalIndex.NonUnique,
			}
			indexesMap[physicalIndex.IndexName] = index
			indexes = append(indexes, index)
		}
		indexFields = append(indexFields, model.SysTableIndexField{
			IndexId: index.Id, FieldId: fieldID, Sequence: uint8(physicalIndex.OrdinalPosition),
		})
	}
	table.TableFields = fields
	table.TableIndexes = indexes
	return RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if e := s.sysTableRepo.Create(tx, &table); e != nil {
			return e
		}
		if len(indexFields) > 0 {
			if e := s.sysTableIndexFieldRepo.Create(tx, &indexFields); e != nil {
				return e
			}
		}
		return nil
	})
}

// SyncTableFields 同步已有表结构到 sys_table_field。
// 已存在的字段只修正敏感字段和系统托管字段的展示开关，不覆盖输入类型、字典、名称等页面配置。
func (s *SysTableService) SyncTableFields(ctx context.Context, tableCode string) error {
	var err error
	tableCode, err = normalizeDBIdentifier("表编码", tableCode)
	if err != nil {
		return err
	}
	columns, err := s.sysTableRepo.FetchTableMetadata(ctx, nil, "public", tableCode)
	if err != nil {
		return err
	}

	table, err := s.GetTableByTableCode(tableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrTableNotFound
	}

	existingFields, err := s.GetTableFieldsByTableId(table.Id)
	if err != nil {
		return err
	}
	fieldMap := make(map[string]model.SysTableField)
	for _, field := range existingFields {
		fieldMap[field.FieldCode] = field
	}

	fields, err := convertColumnsToSysTableFields(table.TableCode, columns)
	if err != nil {
		return err
	}
	var missing []model.SysTableField
	var toUpdate []struct {
		Id    int
		Field model.SysTableField
	}
	for i := range fields {
		if existing, ok := fieldMap[fields[i].FieldCode]; ok {
			if patch, changed := fieldVisibilityPatch(existing); changed {
				toUpdate = append(toUpdate, struct {
					Id    int
					Field model.SysTableField
				}{Id: existing.Id, Field: patch})
			}
			continue
		}
		fieldId, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		fields[i].Id = int(fieldId)
		fields[i].TableId = table.Id
		missing = append(missing, fields[i])
	}

	changed := len(missing) > 0 || len(toUpdate) > 0
	if changed {
		selectRepo := s.sysTableFieldRepo.WithSelect(
			"is_list_show",
			"is_insert_show",
			"is_update_show",
			"is_quick_search",
			"is_advanced_search",
		)
		err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
			if len(missing) > 0 {
				if e := s.sysTableFieldRepo.Create(tx, &missing); e != nil {
					return e
				}
			}
			for _, item := range toUpdate {
				if e := selectRepo.Update(tx, &item.Field, item.Id); e != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	commentFields := append(append([]model.SysTableField{}, existingFields...), missing...)
	if err := s.syncTableFieldComments(s.sysTableRepo.DBWithContext(ctx), table.TableCode, commentFields); err != nil {
		return err
	}
	if changed {
		s.RefreshCache(table.Id)
	}
	return nil
}

func fieldVisibilityPatch(existing model.SysTableField) (model.SysTableField, bool) {
	patched := existing
	applySensitiveFieldDefaults(&patched)
	applyManagedFieldDefaults(&patched)
	changed := patched.IsListShow != existing.IsListShow ||
		patched.IsInsertShow != existing.IsInsertShow ||
		patched.IsUpdateShow != existing.IsUpdateShow ||
		patched.IsQuickSearch != existing.IsQuickSearch ||
		patched.IsAdvancedSearch != existing.IsAdvancedSearch
	if !changed {
		return model.SysTableField{}, false
	}
	return model.SysTableField{
		IsListShow:       patched.IsListShow,
		IsInsertShow:     patched.IsInsertShow,
		IsUpdateShow:     patched.IsUpdateShow,
		IsQuickSearch:    patched.IsQuickSearch,
		IsAdvancedSearch: patched.IsAdvancedSearch,
	}, true
}

// SyncTableIndexes 同步已有表索引到 sys_table_index 与 sys_table_index_field（仅补充缺失索引）
func (s *SysTableService) SyncTableIndexes(ctx context.Context, tableCode string) error {
	var err error
	tableCode, err = normalizeDBIdentifier("表编码", tableCode)
	if err != nil {
		return err
	}
	indexes, err := s.sysTableRepo.FetchTableIndexMetadata(ctx, nil, "public", tableCode)
	if err != nil {
		return err
	}

	table, err := s.GetTableByTableCode(tableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrTableNotFound
	}

	fields, err := s.GetTableFieldsByTableId(table.Id)
	if err != nil {
		return err
	}
	fieldMap := make(map[string]int)
	for _, field := range fields {
		fieldMap[field.FieldCode] = field.Id
	}

	existingIndexes, err := s.sysTableIndexRepo.GetTableIndexesByTableId(ctx, table.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	indexIdMap := make(map[string]int)
	indexIds := make([]int, 0, len(existingIndexes))
	for _, index := range existingIndexes {
		indexIdMap[index.IndexName] = index.Id
		indexIds = append(indexIds, index.Id)
	}

	existingIndexFieldsByKey := make(map[string]model.SysTableIndexField)
	if len(indexIds) > 0 {
		var existingIndexFields []model.SysTableIndexField
		err = s.sysTableIndexFieldRepo.DBWithContext(ctx).
			Where("index_id in ?", indexIds).
			Find(&existingIndexFields).Error
		if err != nil {
			return err
		}
		for _, item := range existingIndexFields {
			key := fmt.Sprintf("%d:%d", item.IndexId, item.FieldId)
			existingIndexFieldsByKey[key] = item
		}
	}

	newIndexes := make([]model.SysTableIndex, 0)
	newIndexFields := make([]model.SysTableIndexField, 0)
	sequenceUpdates := make([]model.SysTableIndexField, 0)

	for _, idx := range indexes {
		fieldId, exists := fieldMap[idx.ColumnName]
		if !exists {
			continue
		}

		indexId, ok := indexIdMap[idx.IndexName]
		if !ok {
			newId, err := s.sf.GenerateUniqueID()
			if err != nil {
				return err
			}
			indexId = int(newId)
			indexIdMap[idx.IndexName] = indexId
			newIndexes = append(newIndexes, model.SysTableIndex{
				Basic: model.Basic{
					Id: indexId,
				},
				TableId:   table.Id,
				IndexName: idx.IndexName,
				IsUnique:  !idx.NonUnique,
			})
		}

		key := fmt.Sprintf("%d:%d", indexId, fieldId)
		sequence := uint8(idx.OrdinalPosition)
		if existing, exists := existingIndexFieldsByKey[key]; exists {
			if existing.Sequence != sequence {
				existing.Sequence = sequence
				sequenceUpdates = append(sequenceUpdates, existing)
			}
			continue
		}
		existingIndexFieldsByKey[key] = model.SysTableIndexField{}
		newIndexFields = append(newIndexFields, model.SysTableIndexField{
			IndexId: indexId, FieldId: fieldId, Sequence: sequence,
		})
	}

	if len(newIndexes) == 0 && len(newIndexFields) == 0 && len(sequenceUpdates) == 0 {
		return nil
	}

	err = RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if len(newIndexes) > 0 {
			if e := s.sysTableIndexRepo.Create(tx, &newIndexes); e != nil {
				return e
			}
		}
		if len(newIndexFields) > 0 {
			if e := s.sysTableIndexFieldRepo.Create(tx, &newIndexFields); e != nil {
				return e
			}
		}
		for _, item := range sequenceUpdates {
			if e := s.sysTableIndexFieldRepo.UpdateSequence(tx, item.IndexId, item.FieldId, item.Sequence); e != nil {
				return e
			}
		}
		return nil
	})
	if err == nil {
		s.RefreshCache(table.Id)
	}
	return err
}

func (s *SysTableService) syncViewTableFields(ctx context.Context, tx *gorm.DB, table model.SysTable) error {
	columns, err := s.sysTableRepo.FetchTableMetadata(ctx, tx, "public", table.TableCode)
	if err != nil {
		return err
	}

	existingFields, err := s.sysTableFieldRepo.FindListByFieldWithDB(tx, "table_id", table.Id)
	if err != nil {
		return err
	}

	newFields, err := convertColumnsToSysTableFields(table.TableCode, columns)
	if err != nil {
		return err
	}
	newFieldMap := make(map[string]model.SysTableField)
	for i := range newFields {
		newFields[i].TableId = table.Id
		newFields[i].IsInsertShow = false
		newFields[i].IsUpdateShow = false
		newFieldMap[newFields[i].FieldCode] = newFields[i]
	}

	existingMap := make(map[string]model.SysTableField)
	for _, field := range existingFields {
		existingMap[field.FieldCode] = field
	}

	var toCreate []model.SysTableField
	var toUpdate []struct {
		Id    int
		Field model.SysTableField
	}
	var toDelete []int

	for code, newField := range newFieldMap {
		if oldField, ok := existingMap[code]; ok {
			existingDefault := ""
			if oldField.DefaultValue != nil {
				existingDefault = *oldField.DefaultValue
			}
			newDefault := ""
			if newField.DefaultValue != nil {
				newDefault = *newField.DefaultValue
			}
			existingDictCode := ""
			if oldField.DictCode != nil {
				existingDictCode = *oldField.DictCode
			}
			newDictCode := ""
			if newField.DictCode != nil {
				newDictCode = *newField.DictCode
			}
			if oldField.FieldName != newField.FieldName ||
				oldField.FieldType != newField.FieldType ||
				oldField.FieldLength != newField.FieldLength ||
				oldField.FieldDecimalLength != newField.FieldDecimalLength ||
				oldField.IsNull != newField.IsNull ||
				oldField.IsPrimaryKey != newField.IsPrimaryKey ||
				existingDefault != newDefault ||
				existingDictCode != newDictCode ||
				oldField.InputType != newField.InputType ||
				oldField.Sequence != newField.Sequence ||
				oldField.IsListShow != newField.IsListShow ||
				oldField.IsInsertShow != newField.IsInsertShow ||
				oldField.IsUpdateShow != newField.IsUpdateShow {
				updateField := model.SysTableField{
					FieldName:          newField.FieldName,
					FieldType:          newField.FieldType,
					FieldLength:        newField.FieldLength,
					FieldDecimalLength: newField.FieldDecimalLength,
					IsNull:             newField.IsNull,
					IsPrimaryKey:       newField.IsPrimaryKey,
					DefaultValue:       newField.DefaultValue,
					DictCode:           newField.DictCode,
					InputType:          newField.InputType,
					Sequence:           newField.Sequence,
					IsListShow:         newField.IsListShow,
					IsInsertShow:       false,
					IsUpdateShow:       false,
				}
				toUpdate = append(toUpdate, struct {
					Id    int
					Field model.SysTableField
				}{Id: oldField.Id, Field: updateField})
			}
		} else {
			fieldId, err := s.sf.GenerateUniqueID()
			if err != nil {
				return err
			}
			newField.Id = int(fieldId)
			toCreate = append(toCreate, newField)
		}
	}

	for code, oldField := range existingMap {
		if _, ok := newFieldMap[code]; !ok {
			toDelete = append(toDelete, oldField.Id)
		}
	}

	if len(toCreate) == 0 && len(toUpdate) == 0 && len(toDelete) == 0 {
		return nil
	}

	selectRepo := s.sysTableFieldRepo.WithSelect(
		"field_name",
		"field_type",
		"field_length",
		"field_decimal_length",
		"is_null",
		"is_primary_key",
		"default_value",
		"dict_code",
		"input_type",
		"sequence",
		"is_list_show",
		"is_insert_show",
		"is_update_show",
	)

	if len(toCreate) > 0 {
		if e := s.sysTableFieldRepo.Create(tx, &toCreate); e != nil {
			return e
		}
	}
	for _, item := range toUpdate {
		if e := selectRepo.Update(tx, &item.Field, item.Id); e != nil {
			return e
		}
	}
	for _, id := range toDelete {
		if e := s.sysTableFieldRepo.DeleteById(tx, id); e != nil {
			return e
		}
	}
	return nil
}

// RefreshCache 刷新表元数据缓存。
// 表元数据有两种常用读取方式：按 id 读取、按 table_code 读取，所以缓存会同时保存两份 key。
// 字段缓存按字段 id 保存，刷新表时也会同步刷新字段缓存，避免修改字段后通用页面仍拿旧配置。
func (s *SysTableService) RefreshCache(originId int) {
	s.metadataRuntime.Refresh(context.Background(), originId)
}

func (s *SysTableService) DeleteCache(tableId int) {
	s.metadataRuntime.Invalidate(context.Background(), tableId)
}

func cleanColumnDisplayName(name string) string {
	text := strings.TrimSpace(name)
	if len(text) < 2 {
		return text
	}
	first := text[:1]
	last := text[len(text)-1:]
	if (first == "'" || first == `"` || first == "`") && first == last {
		cleaned := strings.TrimSpace(text[1 : len(text)-1])
		if cleaned != "" {
			return cleaned
		}
	}
	return text
}

func convertColumnsToSysTableFields(tableCode string, columns []model.TableColumnMate) ([]model.SysTableField, error) {
	var fields []model.SysTableField
	for _, column := range columns {
		field := model.SysTableField{
			FieldCode:          column.ColumnName,              // 通常 FieldCode 会是数据库的真实列名
			FieldDecimalLength: int(column.NumericScale.Int64), // 根据需要设置
			NumericPrecision:   int(column.NumericPrecision.Int64),
			NumericScale:       int(column.NumericScale.Int64),
			IsNull:             column.IsNullable == "YES",
			IsPrimaryKey:       column.ColumnKey == "PRI",
			IsQuickSearch:      false,
			IsAdvancedSearch:   false,
			IsSort:             true,
			IsListShow:         true,
			IsInsertShow:       false,
			IsUpdateShow:       false,
			Sequence:           uint8(column.OrdinalPosition),
			OriginalFieldId:    0,
			FieldLength:        0,
			FieldCategory:      enum.NormalField,
			Binding:            "required", // 根据实际逻辑调整
			InputType:          enum.InputType,
		}
		if column.ColumnComment != "" {
			field.FieldName = cleanColumnDisplayName(column.ColumnComment)
		} else {
			field.FieldName = column.ColumnName
		}
		dataType := strings.ToLower(strings.TrimSpace(column.DataType))
		columnType := strings.ToLower(strings.TrimSpace(column.ColumnType))
		switch dataType {
		case "bigint", "int8":
			field.FieldType = enum.BigIntFieldType
			field.InputType = enum.InputNumberInputType
		case "int", "integer", "int4", "mediumint":
			field.FieldType = enum.IntFieldType
			field.InputType = enum.InputNumberInputType
		case "smallint", "tinyint", "int2":
			if columnType == "tinyint(1)" {
				field.FieldType = enum.BooleanFieldType
				field.InputType = enum.BooleanInputType
			} else {
				field.FieldType = enum.SmallIntFieldType
				field.InputType = enum.InputNumberInputType
				field.FieldLength = int(column.NumericPrecision.Int64)
			}
		case "varchar", "character varying", "character", "char", "bpchar":
			field.FieldType = enum.VarcharFieldType
			field.FieldLength = int(column.CharacterMaximumLength.Int64)
		case "text", "mediumtext", "longtext":
			field.FieldType = enum.TextFieldType
			field.InputType = enum.TextareaInputType
			field.FieldLength = int(column.CharacterMaximumLength.Int64)
		case "boolean", "bool":
			field.FieldType = enum.BooleanFieldType
			field.InputType = enum.BooleanInputType
		case "date":
			field.FieldType = enum.DateFieldType
			field.InputType = enum.DatePickerInputType
		case "datetime", "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
			field.FieldType = enum.DatetimeFieldType
			field.InputType = enum.DatetimePickerInputType
		case "time", "time without time zone", "time with time zone", "timetz":
			field.FieldType = enum.TimeFieldType
			field.InputType = enum.TimePickerInputType
		case "numeric", "decimal":
			field.FieldType = enum.DecimalFieldType
			field.InputType = enum.InputNumberInputType
			field.NumericPrecision = int(column.NumericPrecision.Int64)
			field.NumericScale = int(column.NumericScale.Int64)
		case "double precision", "float", "float4", "float8", "real":
			return nil, myerrors.NewValidationError("近似浮点列不能导入Metadata，请先通过显式Migration转换为numeric")
		case "json", "jsonb":
			field.FieldType = enum.JsonFieldType
			field.InputType = enum.JsonInputType
		default:
			field.FieldType = enum.VarcharFieldType
			if column.CharacterMaximumLength.Valid {
				field.FieldLength = int(column.CharacterMaximumLength.Int64)
			} else {
				field.FieldLength = int(column.NumericPrecision.Int64)
			}
		}
		// 检查DefaultValue是否有值
		if column.ColumnDefault.Valid {
			field.DefaultValue = &column.ColumnDefault.String
		}
		applySystemFieldDefaults(tableCode, &field)
		applySensitiveFieldDefaults(&field)
		applyManagedFieldDefaults(&field)
		fields = append(fields, field)
	}
	return fields, nil
}

func applySystemFieldDefaults(tableCode string, field *model.SysTableField) {
	dictCode := systemFieldDictCode(tableCode, field.FieldCode, field.FieldType)
	if dictCode == "" {
		return
	}
	field.DictCode = utils.StringPtr(dictCode)
	field.InputType = enum.SelectInputType
}

func systemFieldDictCode(tableCode, fieldCode string, fieldType enum.SysTableFieldType) string {
	switch fieldCode {
	case "master_detail_mode":
		if tableCode == "sys_table" {
			return "sys_master_detail_mode"
		}
	case "form_open_mode":
		if tableCode == "sys_table" {
			return "sys_form_open_mode"
		}
	case "detail_open_mode":
		if tableCode == "sys_table" {
			return "sys_detail_open_mode"
		}
	case "table_type":
		return "sys_table_type"
	case "field_type":
		return "sys_table_field_type"
	case "logical_type":
		return "sys_table_field_logical_type"
	case "display_format":
		return "sys_table_field_display_format"
	case "input_type":
		return "sys_table_field_input_type"
	case "field_category":
		return "sys_table_field_category"
	case "relation_type":
		return "sys_table_relation_type"
	case "position":
		if tableCode == "sys_menu_button" || tableCode == "sys_menu_button_template" {
			return "sys_menu_button_position"
		}
	case "display_mode":
		if tableCode == "sys_menu_button" || tableCode == "sys_menu_button_template" {
			return "sys_menu_button_display_mode"
		}
	case "event_action":
		if tableCode == "sys_menu_button" || tableCode == "sys_menu_button_template" {
			return "sys_menu_button_event_action"
		}
	case "method", "http_method":
		return "http_method"
	case "state", "success":
		return "whether"
	}
	if isBooleanLikeField(fieldCode, fieldType) {
		return "whether"
	}
	return ""
}

func isBooleanLikeField(fieldCode string, fieldType enum.SysTableFieldType) bool {
	if fieldType == enum.BooleanFieldType {
		return true
	}
	switch fieldCode {
	case "enabled", "disabled":
		return true
	}
	return strings.HasPrefix(fieldCode, "is_") ||
		strings.HasPrefix(fieldCode, "has_") ||
		strings.HasPrefix(fieldCode, "can_") ||
		strings.HasPrefix(fieldCode, "allow_") ||
		strings.HasPrefix(fieldCode, "enable_") ||
		strings.HasPrefix(fieldCode, "disable_")
}

func applySensitiveFieldDefaults(field *model.SysTableField) {
	if !security.IsSensitiveFieldName(field.FieldCode) {
		return
	}
	field.IsListShow = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
}

func applyManagedFieldDefaults(field *model.SysTableField) {
	if !security.IsManagedMetadataField(field.FieldCode) {
		return
	}
	field.IsListShow = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
}
