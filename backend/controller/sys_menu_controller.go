/**
 * @Author: Nan
 * @Date: 2024/5/17 上午11:12
 */

package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type MenuController struct {
	sysMenuService *service.SysMenuService
	translators    map[string]ut.Translator
}

func NewMenuController(sysMenuService *service.SysMenuService, translators map[string]ut.Translator) *MenuController {
	return &MenuController{
		sysMenuService,
		translators,
	}
}

// GetMenuById 根据ID获取菜单
// @Summary 根据ID获取菜单
// @Description 根据ID获取菜单
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "菜单ID"
// @Success 200 {object} response.Response
// @Router /admin/menu/{id} [get]
func (m *MenuController) GetMenuById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := m.sysMenuService.GetMenuByIdResponse(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// CreateMenu 创建菜单
// @Summary 创建菜单
// @Description 创建菜单
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.MenuCreateReq true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/menu [post]
func (m *MenuController) CreateMenu(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.MenuCreateReq
	translator := m.translators["zh"]
	err := utils.ValidatorBody[request.MenuCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = m.sysMenuService.CreateMenu(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateMenu 更新菜单
// @Summary 更新菜单
// @Description 更新菜单
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "菜单ID"
// @Param data body request.MenuUpdateReq true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/menu/{id} [put]
func (m *MenuController) UpdateMenu(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.MenuUpdateReq
	data.Id = id
	translator := m.translators["zh"]
	err = utils.ValidatorBody[request.MenuUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = m.sysMenuService.UpdateMenu(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateMenuOrder 更新菜单排序
// @Summary 更新菜单排序
// @Description 批量更新菜单排序
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.MenuOrderUpdateReq true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/menu/order [put]
func (m *MenuController) UpdateMenuOrder(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.MenuOrderUpdateReq
	translator := m.translators["zh"]
	if err := utils.ValidatorBody[request.MenuOrderUpdateReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := m.sysMenuService.UpdateMenuOrder(ctx.Request.Context(), data); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(true)
}

// RefreshMenuCache 刷新菜单缓存
// @Summary 刷新菜单缓存
// @Description 当前菜单读取为实时数据库查询，此接口用于前端统一触发菜单刷新
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} response.Response
// @Router /admin/menu/refresh-cache [post]
func (m *MenuController) RefreshMenuCache(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := m.sysMenuService.RefreshMenuCache(ctx.Request.Context()); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(true)
}

// DeleteMenuById 根据ID删除菜单
// @Summary 根据ID删除菜单
// @Description 根据ID删除菜单
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "菜单ID"
// @Success 200 {object} response.Response
// @Router /admin/menu/{id} [delete]
func (m *MenuController) DeleteMenuById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = m.sysMenuService.DeleteMenuById(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// QueryMenus 查询菜单
// @Summary 查询菜单
// @Description 查询菜单
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.Basic true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/menu/query [post]
func (m *MenuController) QueryMenus(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	result, err := m.sysMenuService.GetMenuTreeResponse()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(len(result))
}

// GetUserMenus 获取用户菜单
// @Summary 获取用户菜单
// @Description 获取用户菜单
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /admin/menu/user/{id} [get]
func (m *MenuController) GetUserMenus(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := m.sysMenuService.GetUserMenusResponse(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// GetMyMenus 获取我的菜单
// @Summary 获取我的菜单
// @Description 获取我的菜单
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} response.Response
// @Router /admin/menu/my [get]
func (m *MenuController) GetMyMenus(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	user := ctx.MustGet("user").(model.SysUser)
	result, err := m.sysMenuService.GetUserMenusResponse(user.Id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// GetMenuButtons 获取菜单按钮列表
// @Summary 获取菜单按钮列表
// @Description 获取菜单按钮列表
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param menuId path int true "菜单ID"
// @Success 200 {object} response.Response
// @Router /admin/menu/buttons/{menuId} [get]
func (m *MenuController) GetMenuButtons(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	menuId, err := strconv.Atoi(ctx.Param("menuId"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	data, err := m.sysMenuService.GetMenuButtonsByMenuIdResponse(menuId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// CreateMenuButton 创建菜单按钮
// @Summary 创建菜单按钮
// @Description 创建菜单按钮
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.MenuButtonCreateReq true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/menu/button [post]
func (m *MenuController) CreateMenuButton(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.MenuButtonCreateReq
	translator := m.translators["zh"]
	err := utils.ValidatorBody[request.MenuButtonCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = m.sysMenuService.CreateMenuButton(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateMenuButton 更新菜单按钮
// @Summary 更新菜单按钮
// @Description 更新菜单按钮
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "菜单按钮ID"
// @Param data body request.MenuButtonUpdateReq true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/menu/button/{id} [put]
func (m *MenuController) UpdateMenuButton(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.MenuButtonUpdateReq
	data.Id = id
	translator := m.translators["zh"]
	err = utils.ValidatorBody[request.MenuButtonUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = m.sysMenuService.UpdateMenuButton(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteMenuButton 删除菜单按钮
// @Summary 删除菜单按钮
// @Description 删除菜单按钮
// @Tags 菜单
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "菜单按钮ID"
// @Success 200 {object} response.Response
// @Router /admin/menu/button/{id} [delete]
func (m *MenuController) DeleteMenuButton(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = m.sysMenuService.DeleteMenuButton(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}
