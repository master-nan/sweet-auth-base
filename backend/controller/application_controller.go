/**
 * @Author: Nan
 * @Date: 2024/10/24 12:48
 */

package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/utils"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type ApplicationController struct {
	applicationService *service.ApplicationService
	sysTableService    *service.SysTableService
	translators        map[string]ut.Translator
}

func NewApplicationController(applicationService *service.ApplicationService, sysTableService *service.SysTableService, translators map[string]ut.Translator) *ApplicationController {
	return &ApplicationController{
		applicationService,
		sysTableService,
		translators,
	}
}

// GetApplicationById 根据ID获取应用详情
// @Summary 应用详情
// @Description 根据ID获取应用详情
// @Tags 应用
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/application/{id} [get]
func (t *ApplicationController) GetApplicationById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.applicationService.GetApplicationByIdResponse(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// QueryApplication 获取应用列表
// @Summary 应用列表
// @Description 获取应用列表
// @Tags 应用
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/application/query [post]
func (t *ApplicationController) QueryApplication(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.Basic
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.Basic](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := t.sysTableService.GetTableByTableCode(data.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := t.applicationService.GetApplicationListResponse(&data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// CreateApplication 创建应用
// @Summary 创建应用
// @Description 创建应用
// @Tags 应用
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param b body request.ApplicationCreateReq  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/application [post]
func (t *ApplicationController) CreateApplication(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.ApplicationCreateReq
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.ApplicationCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	application, err := t.applicationService.CreateApplicationResponse(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(application)
}

// UpdateApplication 更新应用
// @Summary 更新应用
// @Description 更新应用
// @Tags 应用
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "应用ID"
// @Param b body request.ApplicationUpdateReq  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/application/update/{id} [put]
func (t *ApplicationController) UpdateApplication(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ApplicationUpdateReq
	data.Id = id
	translator := t.translators["zh"]
	err = utils.ValidatorBody[request.ApplicationUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = t.applicationService.UpdateApplication(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// RotateApplicationSecret 轮换应用密钥
// @Summary 轮换应用密钥
// @Description 轮换应用 app_secret，响应仅本次返回新密钥
// @Tags 应用
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/application/{id}/rotate-secret [post]
func (t *ApplicationController) RotateApplicationSecret(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	application, err := t.applicationService.RotateApplicationSecretResponse(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(application)
}

// DeleteApplicationById 删除应用
// @Summary 删除应用
// @Description 删除应用
// @Tags 应用
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/application/delete/{id} [delete]
func (t *ApplicationController) DeleteApplicationById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = t.applicationService.DeleteApplicationById(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}
