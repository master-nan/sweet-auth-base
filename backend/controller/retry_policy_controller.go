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

const retryPolicyTableCode = "integration_retry_policy"

type retryPolicyApplication interface {
	CreateRetryPolicy(context.Context, request.RetryPolicyCreateReq) (response.RetryPolicyDetailRes, error)
	PageRetryPolicy(context.Context, request.RetryPolicyQueryReq, model.SysTable) (response.ListResult[response.RetryPolicyListRes], error)
	GetRetryPolicy(context.Context, int) (response.RetryPolicyDetailRes, error)
	UpdateDraftRetryPolicy(context.Context, int, request.RetryPolicyUpdateReq) (response.RetryPolicyDetailRes, error)
	CreateRetryPolicyVersion(context.Context, int, int) (response.RetryPolicyDetailRes, error)
	EnableRetryPolicy(context.Context, int, int) (response.RetryPolicyDetailRes, error)
	DisableRetryPolicy(context.Context, int, int) (response.RetryPolicyDetailRes, error)
}

type RetryPolicyController struct {
	service     retryPolicyApplication
	translators map[string]ut.Translator
}

func NewRetryPolicyController(service *service.RetryPolicyService, translators map[string]ut.Translator) *RetryPolicyController {
	return &RetryPolicyController{service: service, translators: translators}
}

func (c *RetryPolicyController) Query(ctx *gin.Context) {
	var req request.RetryPolicyQueryReq
	if !bindRetryPolicy(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.PageRetryPolicy(ctx.Request.Context(), req, retryPolicyQueryTable())
	c.setListResult(ctx, result.Data, result.Total, err)
}

func (c *RetryPolicyController) Detail(ctx *gin.Context) {
	id, ok := retryPolicyPathID(ctx)
	if !ok {
		return
	}
	result, err := c.service.GetRetryPolicy(ctx.Request.Context(), id)
	c.setResult(ctx, result, err, false)
}

func (c *RetryPolicyController) Create(ctx *gin.Context) {
	var req request.RetryPolicyCreateReq
	if !bindRetryPolicy(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.CreateRetryPolicy(ctx.Request.Context(), req)
	c.setResult(ctx, result, err, true)
}

func (c *RetryPolicyController) Update(ctx *gin.Context) {
	id, ok := retryPolicyPathID(ctx)
	if !ok {
		return
	}
	var req request.RetryPolicyUpdateReq
	if !bindRetryPolicy(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.UpdateDraftRetryPolicy(ctx.Request.Context(), id, req)
	c.setResult(ctx, result, err, true)
}

func (c *RetryPolicyController) CreateVersion(ctx *gin.Context) {
	id, ok := retryPolicyPathID(ctx)
	if !ok {
		return
	}
	var req request.RetryPolicyVersionReq
	if !bindRetryPolicy(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.CreateRetryPolicyVersion(ctx.Request.Context(), id, req.Revision)
	c.setResult(ctx, result, err, true)
}

func (c *RetryPolicyController) Enable(ctx *gin.Context) {
	c.changeState(ctx, true)
}

func (c *RetryPolicyController) Disable(ctx *gin.Context) {
	c.changeState(ctx, false)
}

func (c *RetryPolicyController) changeState(ctx *gin.Context, enable bool) {
	id, ok := retryPolicyPathID(ctx)
	if !ok {
		return
	}
	var req request.RetryPolicyStateReq
	if !bindRetryPolicy(ctx, &req, c.translators) {
		return
	}
	var result response.RetryPolicyDetailRes
	var err error
	if enable {
		result, err = c.service.EnableRetryPolicy(ctx.Request.Context(), id, req.Revision)
	} else {
		result, err = c.service.DisableRetryPolicy(ctx.Request.Context(), id, req.Revision)
	}
	c.setResult(ctx, result, err, true)
}

func bindRetryPolicy[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func (c *RetryPolicyController) setResult(ctx *gin.Context, result any, err error, audited bool) {
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

func (c *RetryPolicyController) setListResult(ctx *gin.Context, result any, total int, err error) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(total)
}

func retryPolicyPathID(ctx *gin.Context) (int, bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}

func retryPolicyQueryTable() model.SysTable {
	return model.SysTable{Basic: model.Basic{State: true}, TableCode: retryPolicyTableCode, TableFields: []model.SysTableField{
		retryPolicyQueryField("policy_code", enum.VarcharFieldType, true),
		retryPolicyQueryField("policy_name", enum.VarcharFieldType, true),
		retryPolicyQueryField("version", enum.IntFieldType, false),
		retryPolicyQueryField("status", enum.VarcharFieldType, false),
		retryPolicyQueryField("max_attempts", enum.IntFieldType, false),
		retryPolicyQueryField("backoff_type", enum.VarcharFieldType, false),
		retryPolicyQueryField("initial_delay_ms", enum.BigIntFieldType, false),
		retryPolicyQueryField("max_delay_ms", enum.BigIntFieldType, false),
		retryPolicyQueryField("retry_window_ms", enum.BigIntFieldType, false),
		retryPolicyQueryField("gmt_modify", enum.DatetimeFieldType, false),
	}}
}

func retryPolicyQueryField(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
	return model.SysTableField{Basic: model.Basic{State: true}, FieldCode: code, FieldType: fieldType, IsQuickSearch: quick, IsAdvancedSearch: true, IsSort: true}
}
