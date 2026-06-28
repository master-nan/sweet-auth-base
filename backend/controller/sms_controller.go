/**
 * @Author: Nan
 * @Date: 2025/3/1 15:51
 */

package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/utils"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type SmsController struct {
	smsService            *service.SmsService
	sysTableService       *service.SysTableService
	dataPermissionService *service.DataPermissionService
	translators           map[string]ut.Translator
}

func NewSmsController(smsService *service.SmsService, sysTableService *service.SysTableService, dataPermissionService *service.DataPermissionService, translators map[string]ut.Translator) *SmsController {
	return &SmsController{
		smsService,
		sysTableService,
		dataPermissionService,
		translators,
	}
}

// QuerySmsTemplate 获取短信模板列表
// @Summary 短信模板列表
// @Description 获取短信模板列表
// @Tags 短信
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param b body request.Basic  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/sms/template/query [post]
func (s *SmsController) QuerySmsTemplate(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.Basic
	translator := s.translators["zh"]
	err := utils.ValidatorBody[request.Basic](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := s.sysTableService.GetTableByTableCode(data.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := injectQueryDataScope(ctx, s.dataPermissionService, &data, table); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := s.smsService.GetSmsTemplateList(&data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetSmsTemplateById 根据ID获取短信模板详情
// @Summary 短信模板详情
// @Description 根据ID获取短信模板详情
// @Tags 短信
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "模板ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/sms/template/{id} [get]
func (s *SmsController) GetSmsTemplateById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := checkRecordDataScopeByTableCode(ctx, s.sysTableService, s.dataPermissionService, "sms_template", id, enum.ButtonActionDetail); err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := s.smsService.GetSmsTemplateById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// CreateSmsTemplate 创建短信模板
// @Summary 创建短信模板
// @Description 创建短信模板
// @Tags 短信
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param b body request.SmsTemplateCreateReq true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/sms/template [post]
func (s *SmsController) CreateSmsTemplate(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.SmsTemplateCreateReq
	translator := s.translators["zh"]
	err := utils.ValidatorBody[request.SmsTemplateCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	id, err := s.smsService.CreateSmsTemplate(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(id)
}

// UpdateSmsTemplate 更新短信模板
// @Summary 更新短信模板
// @Description 更新短信模板
// @Tags 短信
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "模板ID"
// @Param b body request.SmsTemplateUpdateReq true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/sms/template/{id} [put]
func (s *SmsController) UpdateSmsTemplate(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.SmsTemplateUpdateReq
	data.Id = id
	translator := s.translators["zh"]
	err = utils.ValidatorBody[request.SmsTemplateUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := checkRecordDataScopeByTableCode(ctx, s.sysTableService, s.dataPermissionService, "sms_template", id, enum.ButtonActionUpdate); err != nil {
		_ = ctx.Error(err)
		return
	}
	err = s.smsService.UpdateSmsTemplate(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(id)
}

// DeleteSmsTemplateById 删除短信模板
// @Summary 删除短信模板
// @Description 删除短信模板
// @Tags 短信
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "模板ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/sms/template/{id} [delete]
func (s *SmsController) DeleteSmsTemplateById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := checkRecordDataScopeByTableCode(ctx, s.sysTableService, s.dataPermissionService, "sms_template", id, enum.ButtonActionDelete); err != nil {
		_ = ctx.Error(err)
		return
	}
	err = s.smsService.DeleteSmsTemplateById(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}
