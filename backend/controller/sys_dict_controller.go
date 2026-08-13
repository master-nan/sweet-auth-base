/**
 * @Author: Nan
 * @Date: 2024/5/23 下午2:57
 */

package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type DictController struct {
	sysDictService  *service.SysDictService
	sysTableService *service.SysTableService
	translators     map[string]ut.Translator
}

func NewDictController(sysDictService *service.SysDictService, sysTableService *service.SysTableService, translators map[string]ut.Translator) *DictController {
	return &DictController{
		sysDictService,
		sysTableService,
		translators,
	}
}

// GetSysDictById 根据ID获取字典详情
// @Summary 字典详情
// @Description 根据ID获取字典详情
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "字典ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /dict/id/{id} [get]
func (t *DictController) GetSysDictById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysDictService.GetSysDictById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// GetSysDictByCode 根据CODE获取字典详情
// @Summary 字典详情
// @Description 根据CODE获取字典详情
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param code path string true "字典CODE"
// @Success 200 {object} response.Response "请求成功"
// @Router /dict/code/{code} [get]
func (t *DictController) GetSysDictByCode(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := utils.SanitizeInput(ctx.Param("code"))
	data, err := t.sysDictService.GetSysDictByCode(code)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// QuerySysDict 字典列表
// @Summary 字典列表
// @Description 根据查询条件查询字段列表
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param q body request.Basic false "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response  "内部错误"
// @Router /dict/query [post]
func (t *DictController) QuerySysDict(ctx *gin.Context) {
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
	result, err := t.sysDictService.GetSysDictList(&data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// CreateSysDict 新增字典
// @Summary 新增字典
// @Description 新增字典主体
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param b body request.DictCreateReq  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response  "内部错误"
// @Router /dict [post]
func (t *DictController) CreateSysDict(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.DictCreateReq
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.DictCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	id, err := t.sysDictService.CreateSysDict(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(id)
}

// UpdateSysDict 更新字典
// @Summary 更新字典
// @Description 根据ID更新字典信息
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "字典ID"
// @Success 200 {object} response.Response "请求成功"
// @Failure 400 {object} response.Response "请求错误"
// @Failure 500 {object} response.Response "内部错误"
// @Router /dict/{id} [put]
func (t *DictController) UpdateSysDict(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.DictUpdateReq
	data.Id = id
	translator := t.translators["zh"]
	err = utils.ValidatorBody[request.DictUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = t.sysDictService.UpdateSysDict(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteSysDictById 删除字典
// @Summary 删除字典
// @Description 根据ID删除字典
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "字典ID"
// @Success 200 {object} response.Response "请求成功"
func (t *DictController) DeleteSysDictById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = t.sysDictService.DeleteSysDictById(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// GetSysDictItemsByDictId 根据字典ID获取字典项
// @Summary 字典项
// @Description 根据字典ID获取字典项
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "字典ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /dict/item/{id} [get]
func (t *DictController) GetSysDictItemsByDictId(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := t.sysDictService.GetSysDictItemsByDictId(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// GetSysDictItemById 根据ID获取字典项
// @Summary 字典项
// @Description 根据ID获取字典项
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "字典项ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /dict/item/id/{id} [get]
func (t *DictController) GetSysDictItemById(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysDictService.GetSysDictItemById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// CreateSysDictItem 新增字典项
// @Summary 新增字典项
// @Description 新增字典项主体
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param b body request.DictItemCreateReq  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /dict/item [post]
func (t *DictController) CreateSysDictItem(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.DictItemCreateReq
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.DictItemCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = t.sysDictService.CreateSysDictItem(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateSysDictItem 更新字典项
// @Summary 更新字典项
// @Description 根据ID更新字典项信息
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "字典项ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /dict/item/{id} [put]
func (t *DictController) UpdateSysDictItem(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	translator := t.translators["zh"]
	var data request.DictItemUpdateReq
	data.Id = id
	err = utils.ValidatorBody[request.DictItemUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = t.sysDictService.UpdateSysDictItem(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteSysDictItemById 删除字典项
// @Summary 删除字典项
// @Description 根据ID删除字典项
// @Tags 字典
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "字典项ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /dict/item/{id} [delete]
func (t *DictController) DeleteSysDictItemById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = t.sysDictService.DeleteSysDictItemById(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}
