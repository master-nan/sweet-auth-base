package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/middleware"
	"backend/service"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type QuerySchemeController struct {
	service     querySchemeApplication
	translators map[string]ut.Translator
}

type querySchemeApplication interface {
	GetScopeConfig(context.Context, string) (response.QueryScopeConfigRes, error)
	Available(context.Context, string) ([]response.QuerySchemeSummaryRes, error)
	Resolve(context.Context, int, request.QuerySchemeResolveReq) (response.QuerySchemeResolveRes, error)
	List(context.Context, request.QuerySchemeManagementQueryReq) (response.ListResult[response.QuerySchemeListRes], error)
	Detail(context.Context, int) (response.QuerySchemeDetailRes, error)
	CreatePersonal(context.Context, request.QuerySchemePersonalCreateReq) (response.QuerySchemeDetailRes, error)
	UpdatePersonal(context.Context, int, request.QuerySchemePersonalUpdateReq) (response.QuerySchemeDetailRes, error)
	DeletePersonal(context.Context, int, int) error
	SetPersonalDefault(context.Context, int, request.QuerySchemeDefaultReq) (response.QuerySchemeDetailRes, error)
	CopyToPersonal(context.Context, int, request.QuerySchemeCopyReq) (response.QuerySchemeDetailRes, error)
	CreateShared(context.Context, request.QuerySchemeSharedCreateReq) (response.QuerySchemeDetailRes, error)
	UpdateShared(context.Context, int, request.QuerySchemeSharedUpdateReq) (response.QuerySchemeDetailRes, error)
	DeleteShared(context.Context, int, int) error
	SetSharedEnabled(context.Context, int, request.QuerySchemeEnabledReq) (response.QuerySchemeDetailRes, error)
}

func NewQuerySchemeController(service *service.QuerySchemeService, translators map[string]ut.Translator) *QuerySchemeController {
	return &QuerySchemeController{service: service, translators: translators}
}

// ScopeConfig godoc
// @Summary 获取查询范围运行配置
// @Tags 查询方案中心
// @Produce json
// @Param scope path string true "查询范围编码"
// @Success 200 {object} response.Response
// @Router /admin/runtime/query-scopes/{scope} [get]
func (controller *QuerySchemeController) ScopeConfig(ctx *gin.Context) {
	result, err := controller.service.GetScopeConfig(ctx.Request.Context(), ctx.Param("scope"))
	controller.result(ctx, result, err, false)
}

// Available godoc
// @Summary 获取当前查询范围可用方案摘要
// @Tags 查询方案中心
// @Produce json
// @Param scope_code query string true "查询范围编码"
// @Success 200 {object} response.Response
// @Router /admin/runtime/query-schemes/available [get]
func (controller *QuerySchemeController) Available(ctx *gin.Context) {
	var req request.QueryScopeReq
	if !bindQuerySchemeQuery(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.Available(ctx.Request.Context(), req.ScopeCode)
	controller.result(ctx, result, err, false)
}

// Resolve godoc
// @Summary 解析查询方案为标准查询
// @Tags 查询方案中心
// @Accept json
// @Produce json
// @Param id path int true "方案ID"
// @Param data body request.QuerySchemeResolveReq true "解析参数"
// @Success 200 {object} response.Response
// @Router /admin/runtime/query-schemes/{id}/resolve [post]
func (controller *QuerySchemeController) Resolve(ctx *gin.Context) {
	id, ok := querySchemePathID(ctx)
	if !ok {
		return
	}
	var req request.QuerySchemeResolveReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.Resolve(ctx.Request.Context(), id, req)
	controller.result(ctx, result, err, false)
}

// Query godoc
// @Summary 分页查询可管理或可使用的查询方案
// @Tags 查询方案中心
// @Accept json
// @Produce json
// @Param data body request.QuerySchemeManagementQueryReq true "查询参数"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/query [post]
func (controller *QuerySchemeController) Query(ctx *gin.Context) {
	var req request.QuerySchemeManagementQueryReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.List(ctx.Request.Context(), req)
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// Detail godoc
// @Summary 获取查询方案详情
// @Tags 查询方案中心
// @Produce json
// @Param id path int true "方案ID"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/{id} [get]
func (controller *QuerySchemeController) Detail(ctx *gin.Context) {
	id, ok := querySchemePathID(ctx)
	if !ok {
		return
	}
	result, err := controller.service.Detail(ctx.Request.Context(), id)
	controller.result(ctx, result, err, false)
}

// CreatePersonal godoc
// @Summary 新建个人查询方案
// @Tags 查询方案中心
// @Accept json
// @Produce json
// @Param data body request.QuerySchemePersonalCreateReq true "个人方案"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/personal [post]
func (controller *QuerySchemeController) CreatePersonal(ctx *gin.Context) {
	var req request.QuerySchemePersonalCreateReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.CreatePersonal(ctx.Request.Context(), req)
	controller.result(ctx, result, err, true)
}

// UpdatePersonal godoc
// @Summary 更新个人查询方案
// @Tags 查询方案中心
// @Accept json
// @Produce json
// @Param id path int true "方案ID"
// @Param data body request.QuerySchemePersonalUpdateReq true "个人方案"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/personal/{id} [put]
func (controller *QuerySchemeController) UpdatePersonal(ctx *gin.Context) {
	id, ok := querySchemePathID(ctx)
	if !ok {
		return
	}
	var req request.QuerySchemePersonalUpdateReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.UpdatePersonal(ctx.Request.Context(), id, req)
	controller.result(ctx, result, err, true)
}

// DeletePersonal godoc
// @Summary 删除个人查询方案
// @Tags 查询方案中心
// @Param id path int true "方案ID"
// @Param revision query int true "修订号"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/personal/{id} [delete]
func (controller *QuerySchemeController) DeletePersonal(ctx *gin.Context) {
	controller.delete(ctx, false)
}

// SetPersonalDefault godoc
// @Summary 设置或取消个人默认方案
// @Tags 查询方案中心
// @Accept json
// @Param id path int true "方案ID"
// @Param data body request.QuerySchemeDefaultReq true "默认状态与修订号"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/personal/{id}/default [put]
func (controller *QuerySchemeController) SetPersonalDefault(ctx *gin.Context) {
	id, ok := querySchemePathID(ctx)
	if !ok {
		return
	}
	var req request.QuerySchemeDefaultReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.SetPersonalDefault(ctx.Request.Context(), id, req)
	controller.result(ctx, result, err, true)
}

// CopyToPersonal godoc
// @Summary 复制可见共享方案为个人方案
// @Tags 查询方案中心
// @Accept json
// @Param id path int true "来源方案ID"
// @Param data body request.QuerySchemeCopyReq true "复制参数"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/{id}/copy-to-personal [post]
func (controller *QuerySchemeController) CopyToPersonal(ctx *gin.Context) {
	id, ok := querySchemePathID(ctx)
	if !ok {
		return
	}
	var req request.QuerySchemeCopyReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.CopyToPersonal(ctx.Request.Context(), id, req)
	controller.result(ctx, result, err, true)
}

// CreateShared godoc
// @Summary 新建共享查询方案
// @Tags 查询方案中心
// @Accept json
// @Param data body request.QuerySchemeSharedCreateReq true "共享方案"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/shared [post]
func (controller *QuerySchemeController) CreateShared(ctx *gin.Context) {
	var req request.QuerySchemeSharedCreateReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.CreateShared(ctx.Request.Context(), req)
	controller.result(ctx, result, err, true)
}

// UpdateShared godoc
// @Summary 更新共享查询方案
// @Tags 查询方案中心
// @Accept json
// @Param id path int true "方案ID"
// @Param data body request.QuerySchemeSharedUpdateReq true "共享方案"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/shared/{id} [put]
func (controller *QuerySchemeController) UpdateShared(ctx *gin.Context) {
	id, ok := querySchemePathID(ctx)
	if !ok {
		return
	}
	var req request.QuerySchemeSharedUpdateReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.UpdateShared(ctx.Request.Context(), id, req)
	controller.result(ctx, result, err, true)
}

// DeleteShared godoc
// @Summary 删除共享查询方案
// @Tags 查询方案中心
// @Param id path int true "方案ID"
// @Param revision query int true "修订号"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/shared/{id} [delete]
func (controller *QuerySchemeController) DeleteShared(ctx *gin.Context) {
	controller.delete(ctx, true)
}

// SetSharedEnabled godoc
// @Summary 启停共享查询方案
// @Tags 查询方案中心
// @Accept json
// @Param id path int true "方案ID"
// @Param data body request.QuerySchemeEnabledReq true "启停参数"
// @Success 200 {object} response.Response
// @Router /admin/query-schemes/shared/{id}/enabled [put]
func (controller *QuerySchemeController) SetSharedEnabled(ctx *gin.Context) {
	id, ok := querySchemePathID(ctx)
	if !ok {
		return
	}
	var req request.QuerySchemeEnabledReq
	if !bindQuerySchemeBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.SetSharedEnabled(ctx.Request.Context(), id, req)
	controller.result(ctx, result, err, true)
}

func (controller *QuerySchemeController) delete(ctx *gin.Context, shared bool) {
	id, ok := querySchemePathID(ctx)
	if !ok {
		return
	}
	var req request.QuerySchemeRevisionReq
	if !bindQuerySchemeQuery(ctx, &req, controller.translators) {
		return
	}
	var err error
	if shared {
		err = controller.service.DeleteShared(ctx.Request.Context(), id, req.Revision)
	} else {
		err = controller.service.DeletePersonal(ctx.Request.Context(), id, req.Revision)
	}
	controller.result(ctx, nil, err, true)
}

func bindQuerySchemeBody[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func bindQuerySchemeQuery[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorQuery(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func (controller *QuerySchemeController) result(ctx *gin.Context, result any, err error, audited bool) {
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

func querySchemePathID(ctx *gin.Context) (int, bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}
