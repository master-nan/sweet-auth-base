/**
 * @Author: Nan
 * @Date: 2024/5/17 上午11:30
 */

package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
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
	"hash/fnv"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SysTableService struct {
	sysTableRepo           repository.SysTableRepository
	sysTableFieldRepo      repository.SysTableFieldRepository
	sysTableIndexRepo      repository.SysTableIndexRepository
	sysTableIndexFieldRepo repository.SysTableIndexFieldRepository
	sysTableRelationRepo   repository.SysTableRelationRepository
	sysMenuRepo            repository.SysMenuRepository
	sysMenuButtonRepo      repository.SysMenuButtonRepository
	sysMenuButtonTplRepo   repository.SysMenuButtonTemplateRepository
	sysRoleRepo            repository.SysRoleRepository
	sysRoleMenuRepo        repository.SysRoleMenuRepository
	sysRoleMenuButtonRepo  repository.SysRoleMenuButtonRepository
	sf                     *utils.Snowflake
	metadataRuntime        *MetadataRuntimeService
	serverConfig           *config.Server
}

const lowCodeCrudButtonTemplateScene = "lowcode_crud"

func NewSysTableService(
	sysTableRepo repository.SysTableRepository,
	sysTableFieldRepo repository.SysTableFieldRepository,
	sysTableIndexRepo repository.SysTableIndexRepository,
	sysTableIndexFieldRepo repository.SysTableIndexFieldRepository,
	sysTableRelationRepo repository.SysTableRelationRepository,
	sysMenuRepo repository.SysMenuRepository,
	sysMenuButtonRepo repository.SysMenuButtonRepository,
	sysMenuButtonTplRepo repository.SysMenuButtonTemplateRepository,
	sysRoleRepo repository.SysRoleRepository,
	sysRoleMenuRepo repository.SysRoleMenuRepository,
	sysRoleMenuButtonRepo repository.SysRoleMenuButtonRepository,
	sf *utils.Snowflake,
	metadataRuntime *MetadataRuntimeService,
	serverConfig *config.Server,
) *SysTableService {
	return &SysTableService{
		sysTableRepo,
		sysTableFieldRepo,
		sysTableIndexRepo,
		sysTableIndexFieldRepo,
		sysTableRelationRepo,
		sysMenuRepo,
		sysMenuButtonRepo,
		sysMenuButtonTplRepo,
		sysRoleRepo,
		sysRoleMenuRepo,
		sysRoleMenuButtonRepo,
		sf,
		metadataRuntime,
		serverConfig,
	}
}

func (s *SysTableService) GetTableById(id int) (model.SysTable, error) {
	return s.metadataRuntime.configTableByID(context.Background(), id)
}

func (s *SysTableService) GetTableList(basic *request.Basic) (response.ListResult[model.SysTable], error) {
	return s.metadataRuntime.listConfigTables(context.Background(), basic)
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
		return fmt.Errorf("主子表展示模式不合法: %s", req.MasterDetailMode)
	}
	req.MasterDetailMode = masterDetailMode
	formOpenMode, ok := enum.NormalizeSysFormOpenMode(string(req.FormOpenMode))
	if !ok {
		return fmt.Errorf("表单打开方式不合法: %s", req.FormOpenMode)
	}
	req.FormOpenMode = formOpenMode
	detailOpenMode, ok := enum.NormalizeSysDetailOpenMode(string(req.DetailOpenMode))
	if !ok {
		return fmt.Errorf("详情打开方式不合法: %s", req.DetailOpenMode)
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
		return myerrors.NewBadRequestError("普通表不允许配置视图SQL")
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
		if err := s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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
			fields := convertColumnsToSysTableFields(req.TableCode, columns)
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
	return s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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
		return nil
	})
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
		return myerrors.NewBadRequestError("表编码是跨模块稳定标识，创建后不可修改")
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
			return fmt.Errorf("主子表展示模式不合法: %s", updateReq.MasterDetailMode)
		}
		updateReq.MasterDetailMode = masterDetailMode
	}
	if strings.TrimSpace(string(updateReq.FormOpenMode)) == "" {
		formOpenMode, _ := enum.NormalizeSysFormOpenMode(string(current.FormOpenMode))
		updateReq.FormOpenMode = formOpenMode
	} else {
		formOpenMode, ok := enum.NormalizeSysFormOpenMode(string(updateReq.FormOpenMode))
		if !ok {
			return fmt.Errorf("表单打开方式不合法: %s", updateReq.FormOpenMode)
		}
		updateReq.FormOpenMode = formOpenMode
	}
	if strings.TrimSpace(string(updateReq.DetailOpenMode)) == "" {
		detailOpenMode, _ := enum.NormalizeSysDetailOpenMode(string(current.DetailOpenMode))
		updateReq.DetailOpenMode = detailOpenMode
	} else {
		detailOpenMode, ok := enum.NormalizeSysDetailOpenMode(string(updateReq.DetailOpenMode))
		if !ok {
			return fmt.Errorf("详情打开方式不合法: %s", updateReq.DetailOpenMode)
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
		err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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
		return myerrors.NewBadRequestError("普通表不允许配置视图SQL")
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
	err := s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
		if e := s.sysTableRepo.DeleteById(tx, id); e != nil {
			return e
		}
		// 删除字段元数据
		if e := s.sysTableFieldRepo.DeleteByField(tx, "table_id", id); e != nil {
			return e
		}
		// 查询表所有索引
		tableIndexes, e := s.sysTableIndexRepo.GetTableIndexesByTableId(ctx, id)
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
		relations, e := s.sysTableRelationRepo.GetTableRelationsByTableId(ctx, id)
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		for _, relation := range relations {
			// 删除关联关系表
			if e := s.sysTableRelationRepo.DeleteById(tx, relation.Id); e != nil {
				return e
			}
			if relation.RelationType == enum.ManyToMany {
				// 删除多对多中间表
				if e := s.sysTableRepo.DropTable(tx, relation.ManyTableCode); e != nil {
					return e
				}
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
	data.Id = int(id)
	err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
		// 构建SQL类型字符串，包括长度、默认值、是否可为空和备注
		sqlType := buildColumnSQLTypeFromField(data)
		if e := s.sysTableRepo.CreateTableColumn(tx, table.TableCode, data.FieldCode, sqlType); e != nil {
			return e
		}
		if e := s.sysTableFieldRepo.Create(tx, &data); e != nil {
			return e
		}
		return nil
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
		err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
			// 构建完整的字段SQL类型，避免缺少类型导致SQL错误
			columnNeedsAlter := false
			if req.FieldType != data.FieldType {
				columnNeedsAlter = true
			}
			if req.FieldLength != data.FieldLength || req.FieldDecimalLength != data.FieldDecimalLength {
				columnNeedsAlter = true
			}
			if req.DefaultValue != "" {
				if data.DefaultValue == nil || req.DefaultValue != *data.DefaultValue {
					columnNeedsAlter = true
				}
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
			return nil
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
		return myerrors.NewBadRequestError("表类型不合法")
	}
	return nil
}

func validateMetadataViewSQL(raw string) (string, error) {
	query, err := platformmetadata.ValidateReadOnlyQuery(raw)
	if errors.Is(err, platformmetadata.ErrReadOnlyQueryEmpty) {
		return "", myerrors.ErrTableViewSQLEmpty
	}
	if err != nil {
		return "", myerrors.NewBadRequestError("视图仅允许单条SELECT/WITH只读查询")
	}
	return query, nil
}

func normalizeDBIdentifier(name, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", myerrors.NewBadRequestError(fmt.Sprintf("%s不能为空", name))
	}
	if len(trimmed) > maxDBIdentifierLength {
		return "", myerrors.NewBadRequestError(fmt.Sprintf("%s长度不能超过%d", name, maxDBIdentifierLength))
	}
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		if !isDBIdentifierChar(ch) {
			return "", myerrors.NewBadRequestError(fmt.Sprintf("%s只能包含字母、数字、下划线", name))
		}
		if i == 0 && ch >= '0' && ch <= '9' {
			return "", myerrors.NewBadRequestError(fmt.Sprintf("%s不能以数字开头", name))
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

func isDBIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

func validateTableIndexFields(table model.SysTable, indexFields []request.TableIndexFieldReq) ([]request.TableIndexFieldReq, []string, error) {
	if table.Id == 0 {
		return nil, nil, myerrors.ErrDataNotFound
	}
	if len(indexFields) == 0 {
		return nil, nil, myerrors.NewBadRequestError("索引字段不能为空")
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
			return nil, nil, myerrors.NewBadRequestError("索引字段不属于当前表")
		}
		fieldCode, err := normalizeDBIdentifier("索引字段编码", indexField.FieldCode)
		if err != nil {
			return nil, nil, err
		}
		field, ok := fieldByID[indexField.FieldId]
		if !ok {
			return nil, nil, myerrors.NewBadRequestError("索引字段不存在")
		}
		if field.FieldCode != fieldCode {
			return nil, nil, myerrors.NewBadRequestError("索引字段ID和字段编码不匹配")
		}
		if _, ok := seen[indexField.FieldId]; ok {
			return nil, nil, myerrors.NewBadRequestError("索引字段不能重复")
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

func (s *SysTableService) validateTableFieldLinkageConfig(raw string, currentTable model.SysTable, currentFieldCode string) error {
	_, err := s.normalizeTableFieldLinkageConfig(raw, currentTable, currentFieldCode)
	return err
}

func (s *SysTableService) resolveLinkageRelatedTable(cfg tableFieldLinkageConfig) (model.SysTable, error) {
	return s.GetTableByTableCode(strings.TrimSpace(cfg.TableCode))
}

func validateTableFieldLinkageConfig(raw string, currentTable model.SysTable, currentFieldCode string, resolveTable func(tableFieldLinkageConfig) (model.SysTable, error)) error {
	_, err := normalizeTableFieldLinkageConfig(raw, currentTable, currentFieldCode, resolveTable)
	return err
}

func normalizeTableFieldLinkageConfig(raw string, currentTable model.SysTable, currentFieldCode string, resolveTable func(tableFieldLinkageConfig) (model.SysTable, error)) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	var envelope tableFieldLinkageEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return "", myerrors.NewBadRequestError("联动配置JSON格式不正确")
	}
	if envelope.Linkage == nil || !envelope.Linkage.Enabled {
		return raw, nil
	}
	cfg := *envelope.Linkage
	switch cfg.Mode {
	case "relation", "cascader":
	default:
		return "", myerrors.NewBadRequestError("联动配置mode仅支持relation或cascader")
	}
	if strings.TrimSpace(cfg.TableCode) == "" {
		return "", myerrors.NewBadRequestError("联动配置必须指定tableCode")
	}
	if cfg.PageSize < 0 || cfg.PageSize > 1000 {
		return "", myerrors.NewBadRequestError("联动配置pageSize必须在0到1000之间")
	}
	relatedTable, err := resolveTable(cfg)
	if err != nil {
		return "", err
	}
	if relatedTable.Id == 0 {
		return "", myerrors.NewBadRequestError("联动配置关联表不存在")
	}
	currentFields := tableFieldCodeSet(currentTable.TableFields)
	if strings.TrimSpace(currentFieldCode) != "" {
		currentFields[currentFieldCode] = struct{}{}
	}
	relatedFields := tableFieldCodeSet(relatedTable.TableFields)
	if strings.TrimSpace(currentFieldCode) != "" &&
		((relatedTable.Id != 0 && relatedTable.Id == currentTable.Id) ||
			(strings.TrimSpace(relatedTable.TableCode) != "" && relatedTable.TableCode == currentTable.TableCode)) {
		relatedFields[currentFieldCode] = struct{}{}
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
	if cfg.Mode == "cascader" && strings.TrimSpace(cfg.ParentKey) == strings.TrimSpace(cfg.ValueKey) {
		return "", myerrors.NewBadRequestError("级联配置父级字段不能和取值字段相同")
	}
	for targetField, sourceField := range cfg.FilterMapping {
		targetField = strings.TrimSpace(targetField)
		sourceField = strings.TrimSpace(sourceField)
		if targetField == "" || sourceField == "" {
			return "", myerrors.NewBadRequestError("联动配置filterMapping不能为空")
		}
		if _, ok := relatedFields[targetField]; !ok {
			return "", myerrors.NewBadRequestError(fmt.Sprintf("联动配置filterMapping目标字段%s不存在", targetField))
		}
		if _, ok := currentFields[sourceField]; !ok {
			return "", myerrors.NewBadRequestError(fmt.Sprintf("联动配置filterMapping源字段%s不存在", sourceField))
		}
	}
	if strings.TrimSpace(cfg.TableCode) != "" || strings.TrimSpace(relatedTable.TableCode) == "" {
		return raw, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", myerrors.NewBadRequestError("联动配置JSON格式不正确")
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

func tableFieldCodeSet(fields []model.SysTableField) map[string]struct{} {
	result := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		result[field.FieldCode] = struct{}{}
	}
	return result
}

func validateOptionalLinkageField(name, fieldCode string, fields map[string]struct{}) error {
	fieldCode = strings.TrimSpace(fieldCode)
	if fieldCode == "" {
		return nil
	}
	if _, ok := fields[fieldCode]; !ok {
		return myerrors.NewBadRequestError(fmt.Sprintf("联动配置%s字段%s不存在", name, fieldCode))
	}
	return nil
}

func buildColumnSQLType(req request.TableFieldUpdateReq, data model.SysTableField) string {
	fieldType := req.FieldType
	length := req.FieldLength
	decimalLength := req.FieldDecimalLength
	if length == 0 {
		length = data.FieldLength
	}
	if decimalLength == 0 {
		decimalLength = data.FieldDecimalLength
	}
	defaultValue := req.DefaultValue
	if defaultValue == "" && data.DefaultValue != nil {
		defaultValue = *data.DefaultValue
	}
	sqlType := utils.SqlTypeFromFieldType(fieldType)
	// 拼接长度（仅对需要长度的类型，BOOLEAN/TEXT/JSON/DATE/DATETIME/TIME 不接受长度参数）
	if length > 0 && typeAcceptsLength(fieldType) {
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
	sqlType := utils.SqlTypeFromFieldType(field.FieldType)
	if field.FieldLength > 0 && typeAcceptsLength(field.FieldType) {
		if field.FieldDecimalLength > 0 {
			sqlType += fmt.Sprintf("(%d,%d)", field.FieldLength, field.FieldDecimalLength)
		} else {
			sqlType += fmt.Sprintf("(%d)", field.FieldLength)
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

// typeAcceptsLength 判断字段类型是否接受长度参数
func typeAcceptsLength(ft enum.SysTableFieldType) bool {
	switch ft {
	case enum.VarcharFieldType, enum.FloatFieldType:
		return true
	default:
		return false
	}
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
	case enum.BigIntFieldType, enum.TinyintFieldType, enum.FloatFieldType, enum.IntFieldType:
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
			err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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

func (s *SysTableService) GetTableRelationsByTableId(tableId int) ([]model.SysTableRelation, error) {
	return s.sysTableRelationRepo.GetTableRelationsByTableId(context.Background(), tableId)
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
	err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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
			mainTable, e := s.GetTableById(data.TableId)
			if e != nil {
				return e
			}
			relatedTable, e := s.GetTableById(data.RelatedTableId)
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
			// 先删除再创建
			if e := s.sysTableRepo.DropTable(tx, data.ManyTableCode); e != nil {
				return e
			}
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

func (s *SysTableService) UpdateTableRelation(ctx context.Context, req request.TableRelationUpdateReq) error {
	oldRelation, err := s.sysTableRelationRepo.FindById(req.Id)
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

	err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
		newJoinTable := req.RelationType == enum.ManyToMany &&
			(oldRelation.RelationType != enum.ManyToMany || oldRelation.ManyTableCode != req.ManyTableCode)
		if newJoinTable && s.sysTableRepo.HasPhysicalTable(tx, req.ManyTableCode) {
			return myerrors.ErrTableExist
		}
		if e := s.sysTableRelationRepo.Update(tx, &req, req.Id); e != nil {
			return e
		}
		// 如果旧关系是多对多且中间表存在，而新关系不再是多对多，则删除中间表
		if oldRelation.RelationType == enum.ManyToMany && oldRelation.ManyTableCode != "" {
			if req.RelationType != enum.ManyToMany || oldRelation.ManyTableCode != req.ManyTableCode {
				if e := s.sysTableRepo.DropTable(tx, oldRelation.ManyTableCode); e != nil {
					return e
				}
			}
		}
		// 如果新关系是多对多，重建中间表
		if req.RelationType == enum.ManyToMany && req.ManyTableCode != "" {
			mainTable, e := s.GetTableById(req.TableId)
			if e != nil {
				return e
			}
			relatedTable, e := s.GetTableById(req.RelatedTableId)
			if e != nil {
				return e
			}
			var referenceKeyField model.SysTableField
			for _, field := range mainTable.TableFields {
				if field.FieldCode == req.ReferenceKey {
					referenceKeyField = field
					referenceKeyField.Tag = utils.StringPtr(`gorm:"primaryKey;autoIncrement:false"`)
				}
			}
			var foreignKeyField model.SysTableField
			for _, field := range relatedTable.TableFields {
				if field.FieldCode == req.ForeignKey {
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
			if e := s.sysTableRepo.DropTable(tx, req.ManyTableCode); e != nil {
				return e
			}
			return s.sysTableRepo.CreateTable(tx, req.ManyTableCode, relationModel)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.RefreshCache(req.TableId)
	return nil
}

func validateMetadataRelation(relationType enum.SysTableRelationType, manyTableCode string) error {
	if relationType < enum.OneToOne || relationType > enum.ManyToMany {
		return myerrors.NewBadRequestError("关系类型不合法")
	}
	if relationType == enum.ManyToMany && manyTableCode == "" {
		return myerrors.NewBadRequestError("多对多关系必须指定中间表编码")
	}
	if relationType != enum.ManyToMany && manyTableCode != "" {
		return myerrors.NewBadRequestError("非多对多关系不允许配置中间表")
	}
	return nil
}

func (s *SysTableService) DeleteTableRelationById(ctx context.Context, id int) error {
	relation, err := s.sysTableRelationRepo.FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrDataNotFound
		}
		return err
	}
	if relation.Id == 0 {
		return myerrors.ErrDataNotFound
	}
	err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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

func (s *SysTableService) GetTableIndexesByTableId(tableId int) ([]model.SysTableIndex, error) {
	return s.sysTableIndexRepo.GetTableIndexesByTableId(context.Background(), tableId)
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
	err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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
		for _, field := range req.IndexFields {
			indexField := model.SysTableIndexField{
				IndexId: data.Id,
				FieldId: field.FieldId,
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
		return myerrors.NewBadRequestError("索引不能切换所属表")
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
	err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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
		for _, field := range req.IndexFields {
			indexField := model.SysTableIndexField{
				IndexId: req.Id,
				FieldId: field.FieldId,
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
	err := s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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
	fields := convertColumnsToSysTableFields(tableCode, columns)
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
	var indexes []model.SysTableIndex
	var indexFields []model.SysTableIndexField
	for i := range fields {
		fields[i].TableId = table.Id
		fieldId, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		fields[i].Id = int(fieldId)
		for j := range tableIndexes {
			if tableIndexes[j].ColumnName == fields[i].FieldCode {
				indexId, err := s.sf.GenerateUniqueID()
				if err != nil {
					return err
				}
				if _, exists := indexesMap[tableIndexes[j].IndexName]; !exists {
					indexesMap[tableIndexes[j].IndexName] = model.SysTableIndex{
						Basic: model.Basic{
							Id: int(indexId),
						},
						TableId:   table.Id,
						IndexName: tableIndexes[j].IndexName,
						IsUnique:  !tableIndexes[j].NonUnique,
					}
					indexes = append(indexes, indexesMap[tableIndexes[j].IndexName])
				} else {
					indexId = int64(indexesMap[tableIndexes[j].IndexName].Id)
				}
				indexFields = append(indexFields, model.SysTableIndexField{
					IndexId: int(indexId),
					FieldId: fields[i].Id,
				})
			}
		}
	}
	table.TableFields = fields
	table.TableIndexes = indexes
	return s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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

	fields := convertColumnsToSysTableFields(table.TableCode, columns)
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

	if len(missing) == 0 && len(toUpdate) == 0 {
		return nil
	}

	selectRepo := s.sysTableFieldRepo.WithSelect(
		"is_list_show",
		"is_insert_show",
		"is_update_show",
		"is_quick_search",
		"is_advanced_search",
	)
	err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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

	s.RefreshCache(table.Id)
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

	existingIndexFieldSet := make(map[string]bool)
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
			existingIndexFieldSet[key] = true
		}
	}

	newIndexes := make([]model.SysTableIndex, 0)
	newIndexFields := make([]model.SysTableIndexField, 0)

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
		if existingIndexFieldSet[key] {
			continue
		}
		existingIndexFieldSet[key] = true
		newIndexFields = append(newIndexFields, model.SysTableIndexField{
			IndexId: indexId,
			FieldId: fieldId,
		})
	}

	if len(newIndexes) == 0 && len(newIndexFields) == 0 {
		return nil
	}

	err = s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
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
		return nil
	})
	if err == nil {
		s.RefreshCache(table.Id)
	}
	return err
}

// PublishTableAsMenu 将元数据表发布成一个侧边栏菜单。
// 这个方法只做业务编排：确认开发管理父菜单、创建或恢复低代码菜单、
// 补齐默认按钮/接口权限，并把权限默认授给 super_admin。具体增删改查
// 放在 repository 层，避免 service 直接拼数据库语句。
func (s *SysTableService) PublishTableAsMenu(ctx context.Context, tableCode string, parentID int) error {
	table, err := s.GetTableByTableCode(tableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrTableNotFound
	}
	return s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
		if err := s.ensureTableCanPublishLowCode(tx, table); err != nil {
			return err
		}
		targetParentID, err := s.resolvePublishParentMenu(tx, parentID)
		if err != nil {
			return err
		}
		menuID, err := s.ensureLowCodeMenu(tx, table, targetParentID)
		if err != nil {
			return err
		}
		if err := s.cleanupLegacyLowCodeMenuButtons(tx, menuID); err != nil {
			return err
		}
		if err := s.hideDuplicateLowCodeMenus(tx, table.TableCode, menuID); err != nil {
			return err
		}
		buttonIDs, err := s.ensureDefaultCrudButtons(tx, table.TableCode, menuID)
		if err != nil {
			return err
		}
		return s.ensureSuperAdminMenuPermissions(tx, menuID, buttonIDs)
	})
}

// ensureTableCanPublishLowCode 判断当前表是否允许发布成低代码菜单。
// 判断依据来自菜单数据本身：如果同一 table_code 已经被系统固定页面绑定，
// 说明这个表由定制页面负责交互，不应该再发布一份通用 CRUD 页面。
func (s *SysTableService) ensureTableCanPublishLowCode(tx *gorm.DB, table model.SysTable) error {
	menus, err := s.sysMenuRepo.FindFixedMenusByTableCode(tx, table.TableCode)
	if err != nil {
		return err
	}
	if len(menus) == 0 {
		return nil
	}
	menu := menus[0]
	title := strings.TrimSpace(menu.Title)
	if title == "" {
		title = menu.Name
	}
	return myerrors.NewBadRequestError(fmt.Sprintf("表 %s 已绑定固定菜单 %s，不能发布成低代码页面", table.TableCode, title))
}

// resolvePublishParentMenu 解析低代码页面发布目录。
// 前端传 parent_id 时按用户选择的菜单挂载；不传时才使用“开发管理”作为默认目录。
func (s *SysTableService) resolvePublishParentMenu(tx *gorm.DB, parentID int) (int, error) {
	if parentID <= 0 {
		return s.ensureDevelopMenu(tx)
	}
	menu, err := s.sysMenuRepo.FindById(parentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, myerrors.NewBadRequestError("发布目录不存在")
		}
		return 0, err
	}
	if menu.IsHidden || !menu.State {
		return 0, myerrors.NewBadRequestError("发布目录不可用")
	}
	if !isLowCodePublishParentMenu(menu) {
		return 0, myerrors.NewBadRequestError("低代码页面只能发布到目录菜单下")
	}
	return menu.Id, nil
}

// isLowCodePublishParentMenu 判断菜单是否能作为低代码发布父级。
// 低代码发布只能挂在目录菜单下，不能挂到固定功能页或另一个低代码页面下面。
// 老数据如果还没有 page_type，但本身没有绑定数据表，也按历史目录菜单兼容处理。
func isLowCodePublishParentMenu(menu model.SysMenu) bool {
	if menu.IsHidden || !menu.State || strings.TrimSpace(menu.TableCode) != "" {
		return false
	}
	if menu.PageType == "" {
		return true
	}
	return menu.PageType == enum.MenuPageTypeDirectory
}

// UnpublishTableMenu 取消发布低代码菜单。
// 取消发布不会物理删除菜单，而是隐藏并停用菜单，同时清理菜单、按钮和数据权限授权，
// 防止旧页签继续拿这个菜单上下文访问通用页面。
func (s *SysTableService) UnpublishTableMenu(ctx context.Context, tableCode string) error {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return myerrors.ErrParamInvalid
	}
	table, err := s.GetTableByTableCode(tableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrTableNotFound
	}
	return s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
		menus, err := s.findPublishedLowCodeMenus(tx, table.TableCode)
		if err != nil {
			return err
		}
		if len(menus) == 0 {
			return nil
		}
		menuIDs := make([]int, 0, len(menus))
		for _, menu := range menus {
			menuIDs = append(menuIDs, menu.Id)
		}
		if err := s.sysMenuRepo.HideMenusByIds(tx, menuIDs); err != nil {
			return err
		}
		if err := s.sysRoleMenuRepo.DeleteByMenuIds(tx, menuIDs); err != nil {
			return err
		}
		if err := s.sysRoleMenuButtonRepo.DeleteByMenuIds(tx, menuIDs); err != nil {
			return err
		}
		return nil
	})
}

// ensureDevelopMenu 确保“开发管理”父菜单存在。
// 发布低代码页面时需要把生成菜单挂在开发管理下面；如果老库没有这条菜单，
// 这里按系统固定菜单补一条，避免发布流程中断。
func (s *SysTableService) ensureDevelopMenu(tx *gorm.DB) (int, error) {
	menu, err := s.sysMenuRepo.FindByField("name", "develop")
	if err == nil {
		return menu.Id, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	menu = model.SysMenu{
		Basic:     model.Basic{Id: 300, State: true},
		Pid:       0,
		Name:      "develop",
		Path:      "develop",
		Component: "src/components/Layout/Layout.vue",
		Title:     "router.develop.default",
		Sequence:  3,
		PageType:  enum.MenuPageTypeDirectory,
		Icon:      utils.StringPtr("developer_mode"),
	}
	return menu.Id, s.sysMenuRepo.Create(tx, &menu)
}

// ensureLowCodeMenu 创建或恢复指定表对应的低代码菜单。
// 如果菜单已经存在，只恢复发布所需的路由、父级、状态等字段，保留用户在菜单管理中
// 调整过的标题、图标和排序；如果不存在，生成新菜单 id 并创建默认菜单。
// 这里不会处理按钮权限，按钮由 ensureDefaultCrudButtons 负责。
func (s *SysTableService) ensureLowCodeMenu(tx *gorm.DB, table model.SysTable, parentID int) (int, error) {
	name := lowCodeMenuName(table.TableCode)
	menu, err := s.findPublishedLowCodeMenu(tx, table.TableCode)
	if err == nil {
		update := map[string]any{
			"pid":        parentID,
			"name":       name,
			"path":       "generalization/" + table.TableCode,
			"component":  "pages/develop/generalization/Index.vue",
			"page_type":  enum.MenuPageTypeLowCode,
			"table_code": table.TableCode,
			"option":     "",
			"is_hidden":  false,
			"state":      true,
			"gmt_delete": nil,
		}
		return menu.Id, s.sysMenuRepo.UpdateMenuFields(tx, menu.Id, update)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, err
	}
	menu = model.SysMenu{
		Basic:     model.Basic{Id: int(id), State: true},
		Pid:       parentID,
		Name:      name,
		Path:      "generalization/" + table.TableCode,
		Component: "pages/develop/generalization/Index.vue",
		Title:     table.TableName,
		Sequence:  uint8(30 + (table.Id % 100)),
		Icon:      utils.StringPtr("dynamic_form"),
		PageType:  enum.MenuPageTypeLowCode,
		TableCode: table.TableCode,
		IsHidden:  false,
	}
	return menu.Id, s.sysMenuRepo.Create(tx, &menu)
}

// cleanupLegacyLowCodeMenuButtons 清理早期自动生成的 system_ 前缀按钮。
// 老版本低代码发布用 system_ 前缀生成按钮，后续改成按表编码生成。
// 重新发布时先移除旧按钮和对应授权，避免权限分配页出现两套重复按钮。
func (s *SysTableService) cleanupLegacyLowCodeMenuButtons(tx *gorm.DB, menuID int) error {
	legacyButtons, err := s.sysMenuButtonRepo.FindLegacyLowCodeButtons(tx, menuID)
	if err != nil {
		return err
	}
	if len(legacyButtons) == 0 {
		return nil
	}
	buttonIDs := make([]int, 0, len(legacyButtons))
	for _, button := range legacyButtons {
		buttonIDs = append(buttonIDs, button.Id)
	}
	if err := s.sysRoleMenuButtonRepo.DeleteByButtonIds(tx, buttonIDs); err != nil {
		return err
	}
	return s.sysMenuButtonRepo.DeleteByIds(tx, buttonIDs)
}

// hideDuplicateLowCodeMenus 隐藏同一张表历史上发布出的重复菜单。
// 发布入口多次迭代后可能存在“同一 table_code 多个菜单”的历史数据；
// 这里只保留当前菜单，其余全部停用，并清掉旧菜单的菜单授权和按钮授权。
func (s *SysTableService) hideDuplicateLowCodeMenus(tx *gorm.DB, tableCode string, keepMenuID int) error {
	menus, err := s.findPublishedLowCodeMenus(tx, tableCode)
	if err != nil {
		return err
	}
	duplicateIDs := make([]int, 0, len(menus))
	for _, menu := range menus {
		if menu.Id != keepMenuID {
			duplicateIDs = append(duplicateIDs, menu.Id)
		}
	}
	if len(duplicateIDs) == 0 {
		return nil
	}
	if err := s.sysMenuRepo.HideMenusByIds(tx, duplicateIDs); err != nil {
		return err
	}
	if err := s.sysRoleMenuRepo.DeleteByMenuIds(tx, duplicateIDs); err != nil {
		return err
	}
	if err := s.sysRoleMenuButtonRepo.DeleteByMenuIds(tx, duplicateIDs); err != nil {
		return err
	}
	return nil
}

// findPublishedLowCodeMenu 获取指定表当前可用的低代码菜单。
// 查询范围只认真正的低代码通用页面菜单，不会把绑定同一 table_code 的系统菜单误认为发布菜单。
func (s *SysTableService) findPublishedLowCodeMenu(tx *gorm.DB, tableCode string) (model.SysMenu, error) {
	menus, err := s.findPublishedLowCodeMenus(tx, tableCode)
	if err != nil {
		return model.SysMenu{}, err
	}
	if len(menus) == 0 {
		return model.SysMenu{}, gorm.ErrRecordNotFound
	}
	return menus[0], nil
}

// findPublishedLowCodeMenus 获取指定表历史发布过的低代码菜单。
// 查询只依赖 page_type 和 table_code，避免固定页面和低代码页面互相误判。
func (s *SysTableService) findPublishedLowCodeMenus(tx *gorm.DB, tableCode string) ([]model.SysMenu, error) {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return nil, nil
	}
	return s.sysMenuRepo.FindPublishedLowCodeMenus(tx, tableCode)
}

func lowCodeMenuName(tableCode string) string {
	raw := strings.TrimSpace(tableCode)
	name := "lowcode_" + raw
	if len(name) <= 32 {
		return name
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(raw))
	hash := fmt.Sprintf("%08x", hasher.Sum32())
	prefixLen := 32 - len("lowcode_") - len("_") - len(hash)
	if prefixLen < 1 {
		prefixLen = 1
	}
	prefix := raw
	if len(prefix) > prefixLen {
		prefix = prefix[:prefixLen]
	}
	return "lowcode_" + prefix + "_" + hash
}

// ensureDefaultCrudButtons 补齐低代码通用页面的默认按钮和接口权限。
// 发布低代码菜单时，页面至少需要查询、新增、详情、编辑、删除、刷新以及文件上传/预览相关接口权限。
// 这个方法只负责“按模板保证存在并更新基础属性”，不负责判断业务按钮应该有哪些；
// 标记完成、审核、调整状态这类业务动作应该由菜单按钮配置维护，不应该继续塞进默认模板。
// 所有数据库写入都委托给 SysMenuButtonRepository，service 层只做模板编排和事务流程控制。
func (s *SysTableService) ensureDefaultCrudButtons(tx *gorm.DB, tableCode string, menuID int) ([]int, error) {
	templates, err := s.sysMenuButtonTplRepo.FindEnabledByScene(lowCodeCrudButtonTemplateScene)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, myerrors.NewBadRequestError("低代码默认按钮模板未初始化")
	}
	defaults := lowCodeDefaultMenuButtons(tableCode, templates)
	buttonIDs := make([]int, 0, len(defaults))
	for _, item := range defaults {
		button, err := s.sysMenuButtonRepo.FindByMenuIdAndCode(tx, menuID, item.Code)
		if err == nil {
			updates := map[string]any{
				"name":          item.Name,
				"memo":          item.Memo,
				"position":      item.Position,
				"event_type":    item.EventType,
				"event_action":  item.EventAction,
				"icon":          item.Icon,
				"color":         item.Color,
				"display_mode":  item.DisplayMode,
				"sequence":      item.Sequence,
				"path":          item.Path,
				"method":        strings.ToUpper(item.Method),
				"params_schema": item.ParamsSchema,
				"confirm_text":  item.ConfirmText,
				"disable_when":  item.DisableWhen,
				"before_hooks":  item.BeforeHooks,
				"after_hooks":   item.AfterHooks,
				"is_button":     item.IsButton,
				"is_hidden":     item.IsHidden,
				"is_disabled":   item.IsDisabled,
				"state":         true,
			}
			if err := s.sysMenuButtonRepo.UpdateMenuButtonFields(tx, button.Id, updates); err != nil {
				return nil, err
			}
			buttonIDs = append(buttonIDs, button.Id)
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return nil, err
		}
		item.Basic = model.Basic{Id: int(id), State: true}
		item.MenuId = menuID
		if err := s.sysMenuButtonRepo.Create(tx, &item); err != nil {
			return nil, err
		}
		buttonIDs = append(buttonIDs, item.Id)
	}
	return buttonIDs, nil
}

// lowCodeDefaultMenuButtons 把数据库里的低代码按钮模板套到当前 tableCode。
// 模板只描述“这类低代码页面默认需要哪些按钮或接口权限”，实际按钮编码仍按当前表编码生成。
func lowCodeDefaultMenuButtons(tableCode string, templates []model.SysMenuButtonTemplate) []model.SysMenuButton {
	buttons := make([]model.SysMenuButton, 0, len(templates))
	for _, template := range templates {
		displayMode, ok := enum.NormalizeSysMenuButtonDisplayMode(string(template.DisplayMode))
		if !ok {
			displayMode = enum.ButtonDisplayAuto
		}
		buttons = append(buttons, model.SysMenuButton{
			Name:         template.Name,
			Code:         tableCode + template.CodeSuffix,
			Memo:         template.Memo,
			Position:     template.Position,
			EventType:    template.EventType,
			EventAction:  template.EventAction,
			Icon:         template.Icon,
			Color:        template.Color,
			DisplayMode:  displayMode,
			Sequence:     template.Sequence,
			Path:         template.Path,
			Method:       strings.ToUpper(template.Method),
			ParamsSchema: template.ParamsSchema,
			ConfirmText:  template.ConfirmText,
			DisableWhen:  template.DisableWhen,
			IsButton:     template.IsButton,
			IsHidden:     false,
			IsDisabled:   template.IsDisabled,
			BeforeHooks:  template.BeforeHooks,
			AfterHooks:   template.AfterHooks,
		})
	}
	return buttons
}

// ensureSuperAdminMenuPermissions 给 super_admin 补齐新发布菜单和默认按钮权限。
// 如果库里没有 super_admin 角色，直接跳过；有这个角色时用忽略冲突插入，
// 避免重复发布同一张表时报唯一键冲突。
func (s *SysTableService) ensureSuperAdminMenuPermissions(tx *gorm.DB, menuID int, buttonIDs []int) error {
	role, err := s.sysRoleRepo.FindByField("name", "super_admin")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if err := s.sysRoleMenuRepo.CreateIfNotExists(tx, model.SysRoleMenu{RoleId: role.Id, MenuId: menuID}); err != nil {
		return err
	}
	for _, buttonID := range buttonIDs {
		if err := s.sysRoleMenuButtonRepo.CreateIfNotExists(tx, model.SysRoleMenuButton{RoleId: role.Id, MenuId: menuID, ButtonId: buttonID}); err != nil {
			return err
		}
	}
	return nil
}

// SyncViewTableFields 视图字段元数据全量对齐（增删改）
func (s *SysTableService) SyncViewTableFields(ctx context.Context, table model.SysTable) error {
	return s.sysTableRepo.ExecuteTx(ctx, func(tx *gorm.DB) error {
		return s.syncViewTableFields(ctx, tx, table)
	})
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

	newFields := convertColumnsToSysTableFields(table.TableCode, columns)
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

func convertColumnsToSysTableFields(tableCode string, columns []model.TableColumnMate) []model.SysTableField {
	var fields []model.SysTableField
	for _, column := range columns {
		field := model.SysTableField{
			FieldCode:          column.ColumnName,              // 通常 FieldCode 会是数据库的真实列名
			FieldDecimalLength: int(column.NumericScale.Int64), // 根据需要设置
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
				field.FieldType = enum.TinyintFieldType
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
		case "numeric", "decimal", "double precision", "float", "float4", "float8", "real":
			field.FieldType = enum.FloatFieldType
			field.InputType = enum.InputNumberInputType
			field.FieldLength = int(column.NumericPrecision.Int64)
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
	return fields
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
