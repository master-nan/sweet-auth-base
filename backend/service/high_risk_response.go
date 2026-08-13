package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"encoding/json"
)

func basicResponse(data model.Basic) response.BasicRes {
	return response.BasicRes{
		Id:         data.Id,
		GmtCreate:  data.GmtCreate,
		CreateName: stringValue(data.CreateName),
		GmtModify:  data.GmtModify,
		ModifyName: stringValue(data.ModifyName),
		State:      data.State,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func reportDefinitionListResponse(data model.ReportDefinition) response.ReportDefinitionListRes {
	return response.ReportDefinitionListRes{
		BasicRes:            basicResponse(data.Basic),
		Code:                data.Code,
		Name:                data.Name,
		Description:         data.Description,
		Category:            data.Category,
		Status:              data.Status,
		PublishedVersionId:  data.PublishedVersionId,
		SourceType:          data.SourceType,
		SourceCode:          data.SourceCode,
		PermissionMenuId:    data.PermissionMenuId,
		PermissionTableCode: data.PermissionTableCode,
		Remark:              data.Remark,
	}
}

func reportDefinitionDetailResponse(data model.ReportDefinition) response.ReportDefinitionDetailRes {
	return response.ReportDefinitionDetailRes{
		ReportDefinitionListRes: reportDefinitionListResponse(data),
		QueryConfig:             json.RawMessage(data.QueryConfig),
		LayoutConfig:            json.RawMessage(data.LayoutConfig),
	}
}

func (s *ReportService) GetReportDefinitionListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.ReportDefinitionListRes], error) {
	result, err := s.GetReportDefinitionList(basic, table)
	if err != nil {
		return response.ListResult[response.ReportDefinitionListRes]{}, err
	}
	items := make([]response.ReportDefinitionListRes, 0, len(result.Data))
	for _, item := range result.Data {
		items = append(items, reportDefinitionListResponse(item))
	}
	return response.ListResult[response.ReportDefinitionListRes]{Data: items, Total: result.Total}, nil
}

func (s *ReportService) GetReportDefinitionByIdResponse(id int) (response.ReportDefinitionDetailRes, error) {
	data, err := s.GetReportDefinitionById(id)
	if err != nil {
		return response.ReportDefinitionDetailRes{}, err
	}
	return reportDefinitionDetailResponse(data), nil
}

func tableFieldListResponse(data model.SysTableField) response.SysTableFieldListRes {
	return response.SysTableFieldListRes{
		BasicRes:           basicResponse(data.Basic),
		TableId:            data.TableId,
		FieldName:          data.FieldName,
		FieldCode:          data.FieldCode,
		FieldType:          data.FieldType,
		FieldLength:        data.FieldLength,
		FieldDecimalLength: data.FieldDecimalLength,
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

func (s *SysTableService) GetTableByTableCodeResponse(code string) (response.SysTableDetailRes, error) {
	data, err := s.GetTableByTableCode(code)
	if err != nil {
		return response.SysTableDetailRes{}, err
	}
	return tableDetailResponse(data), nil
}

func (s *SysTableService) GetTableListResponse(basic *request.Basic) (response.ListResult[response.SysTableListRes], error) {
	result, err := s.GetTableList(basic)
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
	data, err := s.GetTableRelationsByTableId(tableId)
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
	data, err := s.GetTableIndexesByTableId(tableId)
	if err != nil {
		return nil, err
	}
	result := make([]response.SysTableIndexRes, 0, len(data))
	for _, item := range data {
		result = append(result, tableIndexResponse(item))
	}
	return result, nil
}

func menuButtonResponse(data model.SysMenuButton) response.SysMenuButtonRes {
	return response.SysMenuButtonRes{
		BasicRes:     basicResponse(data.Basic),
		MenuId:       data.MenuId,
		Name:         data.Name,
		Code:         data.Code,
		Memo:         data.Memo,
		Position:     data.Position,
		EventType:    data.EventType,
		EventAction:  data.EventAction,
		Icon:         data.Icon,
		Color:        data.Color,
		DisplayMode:  data.DisplayMode,
		Sequence:     data.Sequence,
		ApiPath:      data.Path,
		HttpMethod:   data.Method,
		ParamsSchema: data.ParamsSchema,
		ConfirmText:  data.ConfirmText,
		DisableWhen:  data.DisableWhen,
		IsButton:     data.IsButton,
		IsHidden:     data.IsHidden,
		IsDisabled:   data.IsDisabled,
		BeforeHooks:  data.BeforeHooks,
		AfterHooks:   data.AfterHooks,
	}
}

func menuButtonResponses(items []model.SysMenuButton) []response.SysMenuButtonRes {
	result := make([]response.SysMenuButtonRes, 0, len(items))
	for _, item := range items {
		result = append(result, menuButtonResponse(item))
	}
	return result
}

func menuListResponse(data model.SysMenu) response.SysMenuListRes {
	children := make([]response.SysMenuListRes, 0, len(data.Children))
	for _, child := range data.Children {
		children = append(children, menuListResponse(child))
	}
	return response.SysMenuListRes{
		BasicRes:       basicResponse(data.Basic),
		Pid:            data.Pid,
		Name:           data.Name,
		Path:           data.Path,
		Component:      data.Component,
		Title:          data.Title,
		IsHidden:       data.IsHidden,
		Sequence:       data.Sequence,
		PageType:       data.PageType,
		TableCode:      data.TableCode,
		Option:         data.Option,
		Icon:           data.Icon,
		Redirect:       data.Redirect,
		IsUnfold:       data.IsUnfold,
		DetailOpenMode: data.DetailOpenMode,
		MenuButtons:    menuButtonResponses(data.MenuButtons),
		Children:       children,
	}
}

func menuListResponses(items []model.SysMenu) []response.SysMenuListRes {
	result := make([]response.SysMenuListRes, 0, len(items))
	for _, item := range items {
		result = append(result, menuListResponse(item))
	}
	return result
}

func (s *SysMenuService) GetMenuByIdResponse(id int) (response.SysMenuDetailRes, error) {
	data, err := s.GetMenuById(id)
	if err != nil {
		return response.SysMenuDetailRes{}, err
	}
	return response.SysMenuDetailRes{SysMenuListRes: menuListResponse(data)}, nil
}

func (s *SysMenuService) GetMenuTreeResponse() ([]response.SysMenuListRes, error) {
	data, err := s.GetMenuTree()
	if err != nil {
		return nil, err
	}
	return menuListResponses(data), nil
}

func (s *SysMenuService) GetUserMenusResponse(userId int) ([]response.SysMenuListRes, error) {
	data, err := s.GetUserMenus(userId)
	if err != nil {
		return nil, err
	}
	return menuListResponses(data), nil
}

func (s *SysMenuService) GetMenuButtonsByMenuIdResponse(menuId int) ([]response.SysMenuButtonRes, error) {
	data, err := s.GetMenuButtonsByMenuId(menuId)
	if err != nil {
		return nil, err
	}
	return menuButtonResponses(data), nil
}

func roleListResponse(data model.SysRole) response.SysRoleListRes {
	return response.SysRoleListRes{
		BasicRes: basicResponse(data.Basic),
		Name:     data.Name,
		Memo:     data.Memo,
	}
}

func (s *SysRoleService) GetRoleByIdResponse(id int) (response.SysRoleDetailRes, error) {
	data, err := s.GetRoleById(id)
	if err != nil {
		return response.SysRoleDetailRes{}, err
	}
	return response.SysRoleDetailRes{
		SysRoleListRes: roleListResponse(data),
		Menus:          menuListResponses(data.Menus),
		Buttons:        menuButtonResponses(data.Buttons),
	}, nil
}

func (s *SysRoleService) GetRoleListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.SysRoleListRes], error) {
	result, err := s.GetRoleList(basic, table)
	if err != nil {
		return response.ListResult[response.SysRoleListRes]{}, err
	}
	items := make([]response.SysRoleListRes, 0, len(result.Data))
	for _, item := range result.Data {
		items = append(items, roleListResponse(item))
	}
	return response.ListResult[response.SysRoleListRes]{Data: items, Total: result.Total}, nil
}

func (s *SysRoleService) GetRoleMenusResponse(roleId int) ([]response.SysMenuListRes, error) {
	data, err := s.GetRoleMenus(roleId)
	if err != nil {
		return nil, err
	}
	return menuListResponses(data), nil
}

func (s *SysRoleService) GetRoleMenuButtonsResponse(roleId, menuId int) ([]response.SysMenuButtonRes, error) {
	data, err := s.GetRoleMenuButtons(roleId, menuId)
	if err != nil {
		return nil, err
	}
	return menuButtonResponses(data), nil
}
