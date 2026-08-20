package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"encoding/json"
)

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
