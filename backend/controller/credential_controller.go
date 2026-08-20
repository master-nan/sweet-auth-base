package controller

import (
	"backend/dto/request"
	"backend/dto/response"
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

type credentialApplication interface {
	Create(context.Context, request.CredentialCreateReq) (response.CredentialDetailRes, error)
	Get(context.Context, int) (response.CredentialDetailRes, error)
	Page(context.Context, request.CredentialQueryReq) (response.ListResult[response.CredentialListRes], error)
	Update(context.Context, int, request.CredentialUpdateReq) (response.CredentialDetailRes, error)
	Rotate(context.Context, int, request.CredentialRotateReq) (response.CredentialDetailRes, error)
	Enable(context.Context, int, int) (response.CredentialDetailRes, error)
	Disable(context.Context, int, int) (response.CredentialDetailRes, error)
	Revoke(context.Context, int, int) (response.CredentialDetailRes, error)
}

type CredentialController struct {
	service     credentialApplication
	translators map[string]ut.Translator
}

func NewCredentialController(service *service.CredentialService, translators map[string]ut.Translator) *CredentialController {
	return &CredentialController{service: service, translators: translators}
}

func (c *CredentialController) Query(ctx *gin.Context) {
	var req request.CredentialQueryReq
	if !bindCredential(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Page(ctx.Request.Context(), req)
	c.setListResult(ctx, result.Data, result.Total, err)
}

func (c *CredentialController) Detail(ctx *gin.Context) {
	id, ok := credentialPathID(ctx)
	if !ok {
		return
	}
	result, err := c.service.Get(ctx.Request.Context(), id)
	c.setResult(ctx, result, err, false)
}

func (c *CredentialController) Create(ctx *gin.Context) {
	var req request.CredentialCreateReq
	if !bindCredential(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Create(ctx.Request.Context(), req)
	c.setResult(ctx, result, err, true)
}

func (c *CredentialController) Update(ctx *gin.Context) {
	id, ok := credentialPathID(ctx)
	if !ok {
		return
	}
	var req request.CredentialUpdateReq
	if !bindCredential(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Update(ctx.Request.Context(), id, req)
	c.setResult(ctx, result, err, true)
}

func (c *CredentialController) Rotate(ctx *gin.Context) {
	id, ok := credentialPathID(ctx)
	if !ok {
		return
	}
	var req request.CredentialRotateReq
	if !bindCredential(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.Rotate(ctx.Request.Context(), id, req)
	c.setResult(ctx, result, err, true)
}

func (c *CredentialController) Enable(ctx *gin.Context) {
	c.changeState(ctx, model.CredentialStatusActive)
}
func (c *CredentialController) Disable(ctx *gin.Context) {
	c.changeState(ctx, model.CredentialStatusDisabled)
}
func (c *CredentialController) Revoke(ctx *gin.Context) {
	c.changeState(ctx, model.CredentialStatusRevoked)
}

func (c *CredentialController) changeState(ctx *gin.Context, target string) {
	id, ok := credentialPathID(ctx)
	if !ok {
		return
	}
	var req request.CredentialStateReq
	if !bindCredential(ctx, &req, c.translators) {
		return
	}
	var result response.CredentialDetailRes
	var err error
	switch target {
	case model.CredentialStatusActive:
		result, err = c.service.Enable(ctx.Request.Context(), id, req.Revision)
	case model.CredentialStatusDisabled:
		result, err = c.service.Disable(ctx.Request.Context(), id, req.Revision)
	default:
		result, err = c.service.Revoke(ctx.Request.Context(), id, req.Revision)
	}
	c.setResult(ctx, result, err, true)
}

func bindCredential[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func (c *CredentialController) setResult(ctx *gin.Context, result any, err error, audited bool) {
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

func (c *CredentialController) setListResult(ctx *gin.Context, result any, total int, err error) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(total)
}

func credentialPathID(ctx *gin.Context) (int, bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}
