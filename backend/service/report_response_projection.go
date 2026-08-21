package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/model"
	"context"
	"encoding/json"
)

func reportDefinitionListResponse(data model.ReportDefinition) response.ReportDefinitionListRes {
	return response.ReportDefinitionListRes{
		BasicRes:            response.NewBasicRes(data.Basic),
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

func (s *ReportService) GetReportDefinitionListResponse(ctx context.Context, user model.SysUser, basic *request.Basic, table model.SysTable) (response.ListResult[response.ReportDefinitionListRes], error) {
	result, err := s.GetAuthorizedReportDefinitionList(ctx, user, basic, table)
	if err != nil {
		return response.ListResult[response.ReportDefinitionListRes]{}, err
	}
	items := make([]response.ReportDefinitionListRes, 0, len(result.Data))
	for _, item := range result.Data {
		items = append(items, reportDefinitionListResponse(item))
	}
	return response.ListResult[response.ReportDefinitionListRes]{Data: items, Total: result.Total}, nil
}

func (s *ReportService) GetReportDefinitionByIdResponse(ctx context.Context, user model.SysUser, id int, requestedMenuID int) (response.ReportDefinitionDetailRes, error) {
	data, err := s.GetReportDefinitionByIdWithContext(ctx, id)
	if err != nil {
		return response.ReportDefinitionDetailRes{}, err
	}
	if data.Id == 0 {
		return response.ReportDefinitionDetailRes{}, myerrors.ErrDataNotFound
	}
	if err := s.AuthorizeReportDetail(ctx, user, data, requestedMenuID); err != nil {
		return response.ReportDefinitionDetailRes{}, err
	}
	return reportDefinitionDetailResponse(data), nil
}
