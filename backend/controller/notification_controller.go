package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/service"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type notificationApplication interface {
	UnreadCount(context.Context) (response.NotificationUnreadCountRes, error)
	Recent(context.Context, int) ([]response.NotificationSummaryRes, error)
	Query(context.Context, request.NotificationQueryReq) (response.ListResult[response.NotificationSummaryRes], error)
	Detail(context.Context, int) (response.NotificationDetailRes, error)
	MarkRead(context.Context, int) (response.NotificationDetailRes, error)
	MarkAllRead(context.Context) (response.NotificationMarkAllReadRes, error)
}

type NotificationController struct {
	service     notificationApplication
	translators map[string]ut.Translator
}

func NewNotificationController(
	service *service.NotificationService,
	translators map[string]ut.Translator,
) *NotificationController {
	return &NotificationController{service: service, translators: translators}
}

// UnreadCount godoc
// @Summary 获取当前用户通知未读数
// @Tags 消息通知中心
// @Produce json
// @Success 200 {object} response.Response
// @Router /admin/runtime/notifications/unread-count [get]
func (controller *NotificationController) UnreadCount(ctx *gin.Context) {
	result, err := controller.service.UnreadCount(ctx.Request.Context())
	controller.result(ctx, result, err)
}

// Recent godoc
// @Summary 获取当前用户最近通知
// @Tags 消息通知中心
// @Produce json
// @Param limit query int false "返回条数，默认8，最大10"
// @Success 200 {object} response.Response
// @Router /admin/runtime/notifications/recent [get]
func (controller *NotificationController) Recent(ctx *gin.Context) {
	var req request.NotificationRecentReq
	if !bindNotificationQuery(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.Recent(ctx.Request.Context(), req.Limit)
	controller.result(ctx, result, err)
}

// Query godoc
// @Summary 分页查询当前用户通知
// @Tags 消息通知中心
// @Accept json
// @Produce json
// @Param data body request.NotificationQueryReq true "查询参数"
// @Success 200 {object} response.Response
// @Router /admin/runtime/notifications/query [post]
func (controller *NotificationController) Query(ctx *gin.Context) {
	var req request.NotificationQueryReq
	if !bindNotificationBody(ctx, &req, controller.translators) {
		return
	}
	result, err := controller.service.Query(ctx.Request.Context(), req)
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// Detail godoc
// @Summary 获取当前用户通知详情
// @Tags 消息通知中心
// @Produce json
// @Param id path int true "通知ID"
// @Success 200 {object} response.Response
// @Router /admin/runtime/notifications/{id} [get]
func (controller *NotificationController) Detail(ctx *gin.Context) {
	id, ok := notificationPathId(ctx)
	if !ok {
		return
	}
	result, err := controller.service.Detail(ctx.Request.Context(), id)
	controller.result(ctx, result, err)
}

// MarkRead godoc
// @Summary 标记当前用户的一条通知已读
// @Tags 消息通知中心
// @Produce json
// @Param id path int true "通知ID"
// @Success 200 {object} response.Response
// @Router /admin/runtime/notifications/{id}/read [post]
func (controller *NotificationController) MarkRead(ctx *gin.Context) {
	id, ok := notificationPathId(ctx)
	if !ok {
		return
	}
	result, err := controller.service.MarkRead(ctx.Request.Context(), id)
	controller.result(ctx, result, err)
}

// MarkAllRead godoc
// @Summary 标记当前用户全部通知已读
// @Tags 消息通知中心
// @Produce json
// @Success 200 {object} response.Response
// @Router /admin/runtime/notifications/read-all [post]
func (controller *NotificationController) MarkAllRead(ctx *gin.Context) {
	result, err := controller.service.MarkAllRead(ctx.Request.Context())
	controller.result(ctx, result, err)
}

func bindNotificationBody[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func bindNotificationQuery[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorQuery(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func (controller *NotificationController) result(ctx *gin.Context, result any, err error) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func notificationPathId(ctx *gin.Context) (int, bool) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}
