package service

import (
	"backend/dto/request"
	"backend/dto/response"
	platformmetadata "backend/internal/metadata"
	"backend/internal/querycapability"
	"backend/model"
	"context"
)

func tableFieldListResponse(data model.SysTableField) response.SysTableFieldListRes {
	return response.SysTableFieldListRes{
		BasicRes:           basicResponse(data.Basic),
		TableId:            data.TableId,
		FieldName:          data.FieldName,
		FieldCode:          data.FieldCode,
		FieldType:          data.FieldType,
		FieldLength:        data.FieldLength,
		FieldDecimalLength: data.FieldDecimalLength,
		NumericPrecision:   data.NumericPrecision,
		NumericScale:       data.NumericScale,
		LogicalType:        data.LogicalType,
		DisplayFormat:      data.DisplayFormat,
		ListWidth:          data.ListWidth,
		InputType:          data.InputType,
		FormSpan:           data.FormSpan,
		DetailSpan:         data.DetailSpan,
		DefaultValue:       data.DefaultValue,
		DictCode:           data.DictCode,
		IsPrimaryKey:       data.IsPrimaryKey,
		IsIndex:            data.IsIndex,
		IsQuickSearch:      data.IsQuickSearch,
		IsAdvancedSearch:   data.IsAdvancedSearch,
		IsSort:             data.IsSort,
		IsNull:             data.IsNull,
		IsListShow:         data.IsListShow,
		IsInsertShow:       data.IsInsertShow,
		IsUpdateShow:       data.IsUpdateShow,
		Sequence:           data.Sequence,
		OriginalFieldId:    data.OriginalFieldId,
		Binding:            data.Binding,
		FieldCategory:      data.FieldCategory,
		Expression:         data.Expression,
		Tag:                data.Tag,
		LinkageConfig:      data.LinkageConfig,
	}
}

func tableFieldListResponses(items []model.SysTableField) []response.SysTableFieldListRes {
	result := make([]response.SysTableFieldListRes, 0, len(items))
	for _, item := range items {
		result = append(result, tableFieldListResponse(item))
	}
	return result
}

func tableListResponse(data model.SysTable) response.SysTableListRes {
	return response.SysTableListRes{
		BasicRes:         basicResponse(data.Basic),
		TableName:        data.TableName,
		TableCode:        data.TableCode,
		TableType:        data.TableType,
		MasterDetailMode: data.MasterDetailMode,
		FormOpenMode:     data.FormOpenMode,
		DetailOpenMode:   data.DetailOpenMode,
		ParentId:         data.ParentId,
	}
}

func tableRelationResponse(data model.SysTableRelation) response.SysTableRelationRes {
	return response.SysTableRelationRes{
		BasicRes:       basicResponse(data.Basic),
		TableId:        data.TableId,
		RelatedTableId: data.RelatedTableId,
		ReferenceKey:   data.ReferenceKey,
		ForeignKey:     data.ForeignKey,
		OnDelete:       data.OnDelete,
		OnUpdate:       data.OnUpdate,
		RelationType:   data.RelationType,
		ManyTableCode:  data.ManyTableCode,
	}
}

func tableIndexResponse(data model.SysTableIndex) response.SysTableIndexRes {
	return response.SysTableIndexRes{
		BasicRes:    basicResponse(data.Basic),
		TableId:     data.TableId,
		IndexName:   data.IndexName,
		IsUnique:    data.IsUnique,
		IndexFields: tableFieldListResponses(data.IndexFields),
	}
}

func tableDetailResponse(data model.SysTable) response.SysTableDetailRes {
	relations := make([]response.SysTableRelationRes, 0, len(data.TableRelations))
	for _, item := range data.TableRelations {
		relations = append(relations, tableRelationResponse(item))
	}
	indexes := make([]response.SysTableIndexRes, 0, len(data.TableIndexes))
	for _, item := range data.TableIndexes {
		indexes = append(indexes, tableIndexResponse(item))
	}
	return response.SysTableDetailRes{
		SysTableListRes: tableListResponse(data),
		SQL:             data.SQL,
		TableFields:     tableFieldListResponses(data.TableFields),
		TableRelations:  relations,
		TableIndexes:    indexes,
	}
}

func (s *SysTableService) GetTableByIdResponse(id int) (response.SysTableDetailRes, error) {
	data, err := s.GetTableById(id)
	if err != nil {
		return response.SysTableDetailRes{}, err
	}
	return tableDetailResponse(data), nil
}

func (s *MetadataRuntimeService) GetTableResponse(
	ctx context.Context,
	code string,
) (response.RuntimeTableMetadataRes, error) {
	data, err := s.GetTable(ctx, code)
	if err != nil {
		return response.RuntimeTableMetadataRes{}, err
	}
	return runtimeTableMetadataResponse(data), nil
}

func runtimeTableMetadataResponse(data platformmetadata.TableMetadata) response.RuntimeTableMetadataRes {
	fields := make([]response.RuntimeFieldMetadataRes, 0, len(data.Fields))
	for _, field := range data.Fields {
		fields = append(fields, response.RuntimeFieldMetadataRes{
			Id: field.ID, TableId: field.TableID, FieldName: field.DisplayName, FieldCode: field.Code,
			FieldType: field.StorageType, LogicalType: field.LogicalType, InputType: field.UIComponent,
			DisplayFormat: field.DisplayFormat,
			FieldLength:   field.Length, FieldDecimalLength: field.DecimalLength,
			NumericPrecision: field.NumericPrecision, NumericScale: field.NumericScale, ListWidth: field.ListWidth,
			AllowedOperators: querycapability.AllowedMetadataOperators(field),
			FormSpan:         field.FormSpan, DetailSpan: field.DetailSpan,
			DefaultValue: field.DefaultValue, DictCode: field.DictionaryCode,
			IsPrimaryKey: field.PrimaryKey, IsIndex: field.Indexed,
			IsQuickSearch: field.QuickQuery, IsAdvancedSearch: field.AdvancedQuery, IsSort: field.Sortable,
			IsNull: field.Nullable, IsListShow: field.ListVisible,
			IsInsertShow: field.InsertVisible, IsUpdateShow: field.UpdateVisible,
			Sequence: field.Sequence, OriginalFieldId: field.OriginalFieldID,
			Binding: field.Binding, FieldCategory: field.Category,
			Expression: field.RelationExpression, LinkageConfig: field.LinkageConfig,
			SystemManaged: field.SystemManaged,
		})
		if field.Relation != nil {
			fields[len(fields)-1].Relation = &response.RuntimeRelationDisplayRes{
				TargetTableCode: field.Relation.TargetTableCode, ValueField: field.Relation.ValueField,
				DisplayField: field.Relation.DisplayField, ParentField: field.Relation.ParentField,
			}
		}
	}
	relations := make([]response.RuntimeRelationRes, 0, len(data.Relations))
	for _, relation := range data.Relations {
		relations = append(relations, response.RuntimeRelationRes{
			Id: relation.ID, TableId: relation.TableID, RelatedTableId: relation.RelatedTableID,
			ReferenceKey: relation.ReferenceKey, ForeignKey: relation.ForeignKey,
			RelationType: relation.RelationType, ManyTableCode: relation.ManyTableCode,
		})
	}
	return response.RuntimeTableMetadataRes{
		Id: data.ID, TableName: data.Name, TableCode: data.Code, TableType: data.TableType,
		MasterDetailMode: data.MasterDetailMode, FormOpenMode: data.FormOpenMode,
		DetailOpenMode: data.DetailOpenMode, TableFields: fields, TableRelations: relations,
	}
}

func (s *SysTableService) GetTableListResponse(basic *request.Basic) (response.ListResult[response.SysTableListRes], error) {
	result, err := s.metadataRuntime.listConfigTables(context.Background(), basic)
	if err != nil {
		return response.ListResult[response.SysTableListRes]{}, err
	}
	items := make([]response.SysTableListRes, 0, len(result.Data))
	for _, item := range result.Data {
		items = append(items, tableListResponse(item))
	}
	return response.ListResult[response.SysTableListRes]{Data: items, Total: result.Total}, nil
}

func (s *SysTableService) GetTableFieldByIdResponse(id int) (response.SysTableFieldDetailRes, error) {
	data, err := s.GetTableFieldById(id)
	if err != nil {
		return response.SysTableFieldDetailRes{}, err
	}
	return response.SysTableFieldDetailRes{SysTableFieldListRes: tableFieldListResponse(data)}, nil
}

func (s *SysTableService) GetTableFieldsByTableIdResponse(tableId int) ([]response.SysTableFieldListRes, error) {
	data, err := s.GetTableFieldsByTableId(tableId)
	if err != nil {
		return nil, err
	}
	return tableFieldListResponses(data), nil
}

func (s *SysTableService) GetTableRelationsByTableIdResponse(tableId int) ([]response.SysTableRelationRes, error) {
	data, err := s.sysTableRelationRepo.GetTableRelationsByTableId(context.Background(), tableId)
	if err != nil {
		return nil, err
	}
	result := make([]response.SysTableRelationRes, 0, len(data))
	for _, item := range data {
		result = append(result, tableRelationResponse(item))
	}
	return result, nil
}

func (s *SysTableService) GetTableRelationByIdResponse(id int) (response.SysTableRelationRes, error) {
	data, err := s.GetTableRelationById(id)
	if err != nil {
		return response.SysTableRelationRes{}, err
	}
	return tableRelationResponse(data), nil
}

func (s *SysTableService) GetTableIndexByIdResponse(id int) (response.SysTableIndexRes, error) {
	data, err := s.GetTableIndexById(id)
	if err != nil {
		return response.SysTableIndexRes{}, err
	}
	return tableIndexResponse(data), nil
}

func (s *SysTableService) GetTableIndexesByTableIdResponse(tableId int) ([]response.SysTableIndexRes, error) {
	data, err := s.sysTableIndexRepo.GetTableIndexesByTableId(context.Background(), tableId)
	if err != nil {
		return nil, err
	}
	result := make([]response.SysTableIndexRes, 0, len(data))
	for _, item := range data {
		result = append(result, tableIndexResponse(item))
	}
	return result, nil
}
