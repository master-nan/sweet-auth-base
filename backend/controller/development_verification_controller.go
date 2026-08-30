package controller

import (
	"backend/dto/response"
	"backend/service"
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

type developmentVerificationApplication interface {
	Statuses(context.Context) ([]response.DevelopmentVerificationStatusRes, error)
	Prepare(context.Context, string) (response.DevelopmentVerificationPrepareRes, error)
	Cleanup(context.Context, string) (response.DevelopmentVerificationStatusRes, error)
}

type DevelopmentVerificationController struct {
	service developmentVerificationApplication
}

func NewDevelopmentVerificationController(
	service *service.DevelopmentVerificationService,
) *DevelopmentVerificationController {
	return &DevelopmentVerificationController{service: service}
}

func (controller *DevelopmentVerificationController) Statuses(ctx *gin.Context) {
	result, err := controller.service.Statuses(ctx.Request.Context())
	controller.result(ctx, result, err)
}

func (controller *DevelopmentVerificationController) Prepare(ctx *gin.Context) {
	result, err := controller.service.Prepare(ctx.Request.Context(), strings.TrimSpace(ctx.Param("scenario")))
	controller.result(ctx, result, err)
}

func (controller *DevelopmentVerificationController) Cleanup(ctx *gin.Context) {
	result, err := controller.service.Cleanup(ctx.Request.Context(), strings.TrimSpace(ctx.Param("scenario")))
	controller.result(ctx, result, err)
}

func (controller *DevelopmentVerificationController) result(ctx *gin.Context, result any, err error) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}
