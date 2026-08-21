package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type ReportController struct {
	reportService *service.ReportService
	translators   map[string]ut.Translator
}

func NewReportController(reportService *service.ReportService, translators map[string]ut.Translator) *ReportController {
	return &ReportController{
		reportService: reportService,
		translators:   translators,
	}
}

func (r *ReportController) QueryReportDefinitions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.Basic
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.Basic](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if data.TableCode == "" {
		data.TableCode = "report_definition"
	}
	table, err := r.reportService.ResolveRuntimeTable(ctx.Request.Context(), data.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.reportService.GetReportDefinitionListResponse(ctx.Request.Context(), reportRequestUser(ctx), &data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

func (r *ReportController) GetReportDefinitionById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	requestedMenuID := 0
	if rawMenuID := ctx.Query("menu_id"); rawMenuID != "" {
		requestedMenuID, err = strconv.Atoi(rawMenuID)
		if err != nil || requestedMenuID <= 0 {
			_ = ctx.Error(myerrors.NewParameterError("menu_id 参数错误"))
			return
		}
	}
	data, err := r.reportService.GetReportDefinitionByIdResponse(ctx.Request.Context(), reportRequestUser(ctx), id, requestedMenuID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

func reportRequestUser(ctx *gin.Context) model.SysUser {
	if ctx == nil {
		return model.SysUser{}
	}
	value, exists := ctx.Get("user")
	if !exists {
		return model.SysUser{}
	}
	user, _ := value.(model.SysUser)
	return user
}

func (r *ReportController) GetReportDataSources(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	data, err := r.reportService.GetDataSources()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data).SetTotal(len(data))
}

func (r *ReportController) InferSQLFields(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.ReportSQLFieldsReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportSQLFieldsReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	fields, err := r.reportService.InferSQLFields(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(fields).SetTotal(len(fields))
}

func (r *ReportController) CreateReportDefinition(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.ReportDefinitionCreateReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportDefinitionCreateReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	id, err := r.reportService.CreateReportDefinition(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(id)
}

func (r *ReportController) UpdateReportDefinition(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportDefinitionUpdateReq
	data.Id = id
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportDefinitionUpdateReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := r.reportService.UpdateReportDefinition(ctx, data); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(id)
}

func (r *ReportController) DeleteReportDefinitionById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := r.reportService.DeleteReportDefinitionById(ctx, id); err != nil {
		_ = ctx.Error(err)
		return
	}
}

func (r *ReportController) UpdateReportDefinitionStatus(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportStatusUpdateReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportStatusUpdateReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := r.reportService.UpdateReportDefinitionStatus(ctx, id, data.Status); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(id)
}

func (r *ReportController) PublishReport(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportPublishReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportPublishReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.reportService.PublishReport(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (r *ReportController) PublishReportMenu(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportPublishMenuReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportPublishMenuReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.reportService.PublishReportAsMenu(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (r *ReportController) UnpublishReportMenu(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.reportService.UnpublishReportMenu(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (r *ReportController) GetReportVersions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.reportService.GetReportVersions(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(len(result))
}

func (r *ReportController) DesignPreviewReport(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportPreviewReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportPreviewReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.reportService.DesignPreview(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(result.Total)
}

func (r *ReportController) RunReport(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportPreviewReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportPreviewReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.reportService.RunReport(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(result.Total)
}

func (r *ReportController) ExportReport(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportExportReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportExportReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	file, err := r.reportService.ExportReport(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	ctx.Header("Content-Type", file.ContentType)
	ctx.Header("Content-Disposition", contentDisposition("attachment", file.FileName))
	ctx.Data(200, file.ContentType, file.Content)
}

func (r *ReportController) PreviewReport(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportPreviewReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportPreviewReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.reportService.Preview(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(result.Total)
}
