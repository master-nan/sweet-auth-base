/**
 * @Author: Nan
 * @Date: 2024/8/2 上午10:18
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

type RoleController struct {
	sysRoleService  *service.SysRoleService
	sysTableService *service.SysTableService
	translators     map[string]ut.Translator
}

func NewRoleController(sysRoleService *service.SysRoleService, sysTableService *service.SysTableService, translators map[string]ut.Translator) *RoleController {
	return &RoleController{
		sysRoleService,
		sysTableService,
		translators,
	}
}

// QueryRole 查询角色列表
// @Summary 查询角色列表
// @Description 查询角色列表
// @Tags 角色
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.Basic  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/role/query [post]
func (r *RoleController) QueryRole(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.Basic
	translator := r.translators["zh"]
	err := utils.ValidatorBody[request.Basic](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := r.sysTableService.GetTableByTableCode(data.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.sysRoleService.GetRoleList(&data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetRoleById 根据ID获取角色
// @Summary 根据ID获取角色
// @Description 根据ID获取角色
// @Tags 角色
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/role/{id} [get]
func (r *RoleController) GetRoleById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	role, err := r.sysRoleService.GetRoleById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(role)
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 创建角色
// @Tags 角色
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.RoleCreateReq  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/role [post]
func (r *RoleController) CreateRole(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.RoleCreateReq
	translator := r.translators["zh"]
	err := utils.ValidatorBody[request.RoleCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = r.sysRoleService.CreateRole(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateRole 更新角色
// @Summary 更新角色
// @Description 更新角色
// @Tags 角色
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "角色ID"
// @Param data body request.RoleUpdateReq  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/role/{id} [put]
func (r *RoleController) UpdateRole(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.RoleUpdateReq
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.NewBadRequestError(err.Error()))
		return
	}
	data.Id = id
	translator := r.translators["zh"]
	err = utils.ValidatorBody[request.RoleUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = r.sysRoleService.UpdateRole(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteRoleById 根据ID删除角色
// @Summary 根据ID删除角色
// @Description 根据ID删除角色
// @Tags 角色
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/role/{id} [delete]
func (r *RoleController) DeleteRoleById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = r.sysRoleService.DeleteRole(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// GetRoleMenus 获取角色菜单
// @Summary 获取角色菜单
// @Description 获取角色菜单
// @Tags 角色
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/role/{id}/menus [get]
func (r *RoleController) GetRoleMenus(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	roleId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.sysRoleService.GetRoleMenus(roleId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// AssignPermissions 分配角色权限
// @Summary 分配角色权限
// @Description 分配角色权限
// @Tags 角色
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.RoleAssignPermissionsReq  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/role/assign-permissions [post]
func (r *RoleController) AssignPermissions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.RoleAssignPermissionsReq
	translator := r.translators["zh"]
	err := utils.ValidatorBody[request.RoleAssignPermissionsReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = r.sysRoleService.AssignPermissions(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// GetRoleMenuButtons 获取角色菜单按钮权限
// @Summary 获取角色菜单按钮权限
// @Description 获取角色菜单按钮权限
// @Tags 角色
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param roleId path int true "角色ID"
// @Param menuId path int true "菜单ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/role/{roleId}/menu/{menuId}/buttons [get]
func (r *RoleController) GetRoleMenuButtons(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	roleId, err := strconv.Atoi(ctx.Param("roleId"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	menuId, err := strconv.Atoi(ctx.Param("menuId"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := r.sysRoleService.GetRoleMenuButtons(roleId, menuId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}
