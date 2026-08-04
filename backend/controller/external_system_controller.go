package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

const externalSystemTableCode = "integration_external_system"

type externalSystemApplication interface {
	Create(context.Context, request.ExternalSystemCreateReq) (response.ExternalSystemDetailRes, error)
	Get(context.Context, int) (response.ExternalSystemDetailRes, error)
	Page(context.Context, request.ExternalSystemQueryReq, model.SysTable) (response.ListResult[response.ExternalSystemListRes], error)
	Update(context.Context, int, request.ExternalSystemUpdateReq) (response.ExternalSystemDetailRes, error)
	Enable(context.Context, int, int) (response.ExternalSystemDetailRes, error)
	Disable(context.Context, int, int) (response.ExternalSystemDetailRes, error)
}

type ExternalSystemController struct {
	service     externalSystemApplication
	translators map[string]ut.Translator
}

func NewExternalSystemController(
	service *service.ExternalSystemService,
	translators map[string]ut.Translator,
) *ExternalSystemController {
	return newExternalSystemController(service, translators)
}

func newExternalSystemController(
	service externalSystemApplication,
	translators map[string]ut.Translator,
) *ExternalSystemController {
	return &ExternalSystemController{service: service, translators: translators}
}

func (c *ExternalSystemController) Query(ctx *gin.Context) {
	var req request.ExternalSystemQueryReq
	if !bindExternalSystem(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Page(ctx.Request.Context(), req, externalSystemQueryTable())
	c.setListResult(ctx, result.Data, result.Total, err)
}

func (c *ExternalSystemController) Detail(ctx *gin.Context) {
	id, ok := externalSystemPathID(ctx)
	if !ok {
		return
	}
	result, err := c.service.Get(ctx.Request.Context(), id)
	c.setResult(ctx, result, err, false)
}

func (c *ExternalSystemController) Create(ctx *gin.Context) {
	var req request.ExternalSystemCreateReq
	if !bindExternalSystem(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Create(ctx.Request.Context(), req)
	c.setResult(ctx, result, err, true)
}

func (c *ExternalSystemController) Update(ctx *gin.Context) {
	id, ok := externalSystemPathID(ctx)
	if !ok {
		return
	}
	var req request.ExternalSystemUpdateReq
	if !bindExternalSystem(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Update(ctx.Request.Context(), id, req)
	c.setResult(ctx, result, err, true)
}

func (c *ExternalSystemController) Enable(ctx *gin.Context) {
	c.changeState(ctx, true)
}

func (c *ExternalSystemController) Disable(ctx *gin.Context) {
	c.changeState(ctx, false)
}

func (c *ExternalSystemController) changeState(ctx *gin.Context, enable bool) {
	id, ok := externalSystemPathID(ctx)
	if !ok {
		return
	}
	var req request.ExternalSystemStateReq
	if !bindExternalSystem(ctx, &req, c.translators) {
		return
	}
	var (
		result response.ExternalSystemDetailRes
		err    error
	)
	if enable {
		result, err = c.service.Enable(ctx.Request.Context(), id, req.Revision)
	} else {
		result, err = c.service.Disable(ctx.Request.Context(), id, req.Revision)
	}
	c.setResult(ctx, result, err, true)
}

func bindExternalSystem[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func (c *ExternalSystemController) setResult(ctx *gin.Context, result any, err error, audited bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if audited {
		middleware.MarkAccessAuditPersisted(ctx)
	}
	resp.SetData(result)
}

func (c *ExternalSystemController) setListResult(ctx *gin.Context, result any, total int, err error) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(total)
}

func externalSystemPathID(ctx *gin.Context) (int, bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}

func externalSystemQueryTable() model.SysTable {
	return model.SysTable{
		Basic:     model.Basic{State: true},
		TableCode: externalSystemTableCode,
		TableFields: []model.SysTableField{
			externalSystemQueryField("system_code", enum.VarcharFieldType, true),
			externalSystemQueryField("name", enum.VarcharFieldType, true),
			externalSystemQueryField("system_type", enum.VarcharFieldType, false),
			externalSystemQueryField("owner_identifier", enum.VarcharFieldType, true),
			externalSystemQueryField("owner_name", enum.VarcharFieldType, true),
			externalSystemQueryField("status", enum.VarcharFieldType, false),
			externalSystemQueryField("gmt_modify", enum.DatetimeFieldType, false),
		},
	}
}

func externalSystemQueryField(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
	return model.SysTableField{
		Basic:            model.Basic{State: true},
		FieldCode:        code,
		FieldType:        fieldType,
		IsQuickSearch:    quick,
		IsAdvancedSearch: true,
		IsSort:           true,
	}
}
