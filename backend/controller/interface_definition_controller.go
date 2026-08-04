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

const interfaceDefinitionTableCode = "integration_interface_definition"

type interfaceDefinitionApplication interface {
	Create(context.Context, request.InterfaceDefinitionCreateReq) (response.InterfaceDefinitionDetailRes, error)
	Get(context.Context, int) (response.InterfaceDefinitionDetailRes, error)
	Page(context.Context, request.InterfaceDefinitionQueryReq, model.SysTable) (response.ListResult[response.InterfaceDefinitionListRes], error)
	Update(context.Context, int, request.InterfaceDefinitionUpdateReq) (response.InterfaceDefinitionDetailRes, error)
	CreateVersion(context.Context, int, int) (response.InterfaceDefinitionDetailRes, error)
	Enable(context.Context, int, int) (response.InterfaceDefinitionDetailRes, error)
	Disable(context.Context, int, int) (response.InterfaceDefinitionDetailRes, error)
}

type InterfaceDefinitionController struct {
	service     interfaceDefinitionApplication
	translators map[string]ut.Translator
}

func NewInterfaceDefinitionController(service *service.InterfaceDefinitionService, translators map[string]ut.Translator) *InterfaceDefinitionController {
	return newInterfaceDefinitionController(service, translators)
}

func newInterfaceDefinitionController(service interfaceDefinitionApplication, translators map[string]ut.Translator) *InterfaceDefinitionController {
	return &InterfaceDefinitionController{service: service, translators: translators}
}

func (c *InterfaceDefinitionController) Query(ctx *gin.Context) {
	var req request.InterfaceDefinitionQueryReq
	if !bindInterfaceDefinition(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Page(ctx.Request.Context(), req, interfaceDefinitionQueryTable())
	c.setListResult(ctx, result.Data, result.Total, err)
}

func (c *InterfaceDefinitionController) Detail(ctx *gin.Context) {
	id, ok := interfaceDefinitionPathID(ctx)
	if !ok {
		return
	}
	result, err := c.service.Get(ctx.Request.Context(), id)
	c.setResult(ctx, result, err, false)
}

func (c *InterfaceDefinitionController) Create(ctx *gin.Context) {
	var req request.InterfaceDefinitionCreateReq
	if !bindInterfaceDefinition(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Create(ctx.Request.Context(), req)
	c.setResult(ctx, result, err, true)
}

func (c *InterfaceDefinitionController) Update(ctx *gin.Context) {
	id, ok := interfaceDefinitionPathID(ctx)
	if !ok {
		return
	}
	var req request.InterfaceDefinitionUpdateReq
	if !bindInterfaceDefinition(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Update(ctx.Request.Context(), id, req)
	c.setResult(ctx, result, err, true)
}

func (c *InterfaceDefinitionController) CreateVersion(ctx *gin.Context) {
	id, ok := interfaceDefinitionPathID(ctx)
	if !ok {
		return
	}
	var req request.InterfaceDefinitionVersionReq
	if !bindInterfaceDefinition(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.CreateVersion(ctx.Request.Context(), id, req.Revision)
	c.setResult(ctx, result, err, true)
}

func (c *InterfaceDefinitionController) Enable(ctx *gin.Context)  { c.changeState(ctx, true) }
func (c *InterfaceDefinitionController) Disable(ctx *gin.Context) { c.changeState(ctx, false) }

func (c *InterfaceDefinitionController) changeState(ctx *gin.Context, enable bool) {
	id, ok := interfaceDefinitionPathID(ctx)
	if !ok {
		return
	}
	var req request.InterfaceDefinitionStateReq
	if !bindInterfaceDefinition(ctx, &req, c.translators) {
		return
	}
	var result response.InterfaceDefinitionDetailRes
	var err error
	if enable {
		result, err = c.service.Enable(ctx.Request.Context(), id, req.Revision)
	} else {
		result, err = c.service.Disable(ctx.Request.Context(), id, req.Revision)
	}
	c.setResult(ctx, result, err, true)
}

func bindInterfaceDefinition[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func (c *InterfaceDefinitionController) setResult(ctx *gin.Context, result any, err error, audited bool) {
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

func (c *InterfaceDefinitionController) setListResult(ctx *gin.Context, result any, total int, err error) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(total)
}

func interfaceDefinitionPathID(ctx *gin.Context) (int, bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}

func interfaceDefinitionQueryTable() model.SysTable {
	return model.SysTable{Basic: model.Basic{State: true}, TableCode: interfaceDefinitionTableCode, TableFields: []model.SysTableField{
		interfaceDefinitionQueryField("interface_code", enum.VarcharFieldType, true),
		interfaceDefinitionQueryField("name", enum.VarcharFieldType, true),
		interfaceDefinitionQueryField("version", enum.IntFieldType, false),
		interfaceDefinitionQueryField("protocol", enum.VarcharFieldType, false),
		interfaceDefinitionQueryField("http_method", enum.VarcharFieldType, false),
		interfaceDefinitionQueryField("relative_path", enum.VarcharFieldType, true),
		interfaceDefinitionQueryField("status", enum.VarcharFieldType, false),
		interfaceDefinitionQueryField("gmt_modify", enum.DatetimeFieldType, false),
	}}
}

func interfaceDefinitionQueryField(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
	return model.SysTableField{Basic: model.Basic{State: true}, FieldCode: code, FieldType: fieldType, IsQuickSearch: quick, IsAdvancedSearch: true, IsSort: true}
}
