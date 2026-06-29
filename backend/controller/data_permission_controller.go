package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type DataPermissionController struct {
	dataPermissionService *service.DataPermissionService
	translators           map[string]ut.Translator
}

func NewDataPermissionController(dataPermissionService *service.DataPermissionService, translators map[string]ut.Translator) *DataPermissionController {
	return &DataPermissionController{dataPermissionService: dataPermissionService, translators: translators}
}

func (d *DataPermissionController) QueryDimensions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.Basic
	translator := d.translators["zh"]
	if err := utils.ValidatorBody[request.Basic](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, total, err := d.dataPermissionService.QueryDimensions(&data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(int(total))
}

func (d *DataPermissionController) GetDimensionById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	result, err := d.dataPermissionService.GetDimensionById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (d *DataPermissionController) CreateDimension(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.DataPermissionDimensionCreateReq
	translator := d.translators["zh"]
	if err := utils.ValidatorBody[request.DataPermissionDimensionCreateReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := d.dataPermissionService.CreateDimension(ctx, data); err != nil {
		_ = ctx.Error(err)
		return
	}
}

func (d *DataPermissionController) UpdateDimension(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.DataPermissionDimensionUpdateReq
	data.Id = id
	translator := d.translators["zh"]
	if err := utils.ValidatorBody[request.DataPermissionDimensionUpdateReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := d.dataPermissionService.UpdateDimension(ctx, data); err != nil {
		_ = ctx.Error(err)
		return
	}
}

func (d *DataPermissionController) DeleteDimension(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	if err := d.dataPermissionService.DeleteDimension(ctx, id); err != nil {
		_ = ctx.Error(err)
		return
	}
}

func (d *DataPermissionController) GetMenuBindings(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	menuId, err := strconv.Atoi(ctx.Param("menuId"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	result, err := d.dataPermissionService.GetMenuBindings(menuId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (d *DataPermissionController) SaveMenuBindings(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	menuId, err := strconv.Atoi(ctx.Param("menuId"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.DataPermissionBindingSaveReq
	data.MenuId = menuId
	translator := d.translators["zh"]
	if err := utils.ValidatorBody[request.DataPermissionBindingSaveReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := d.dataPermissionService.SaveMenuBindings(ctx, menuId, data); err != nil {
		_ = ctx.Error(err)
		return
	}
}

func (d *DataPermissionController) GetDimensionOptions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	result, err := d.dataPermissionService.DimensionOptions(ctx.Param("code"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (d *DataPermissionController) GetRoleDataScopes(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	roleId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	result, err := d.dataPermissionService.GetRoleDataScopes(roleId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (d *DataPermissionController) SaveRoleDataScopes(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	roleId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.RoleDataPermissionSaveReq
	data.RoleId = roleId
	translator := d.translators["zh"]
	if err := utils.ValidatorBody[request.RoleDataPermissionSaveReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := d.dataPermissionService.SaveRoleDataScopes(ctx, roleId, data); err != nil {
		_ = ctx.Error(err)
		return
	}
}

func (d *DataPermissionController) GetUserOverrides(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	userId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	result, err := d.dataPermissionService.GetUserOverrides(userId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (d *DataPermissionController) SaveUserOverrides(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	userId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.UserDataPermissionOverrideSaveReq
	data.UserId = userId
	translator := d.translators["zh"]
	if err := utils.ValidatorBody[request.UserDataPermissionOverrideSaveReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := d.dataPermissionService.SaveUserOverrides(ctx, userId, data); err != nil {
		_ = ctx.Error(err)
		return
	}
}

func (d *DataPermissionController) GetUserDimensionValues(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	userId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	result, err := d.dataPermissionService.GetUserDimensionValues(userId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (d *DataPermissionController) SaveUserDimensionValues(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	userId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.UserDimensionValueSaveReq
	data.UserId = userId
	translator := d.translators["zh"]
	if err := utils.ValidatorBody[request.UserDimensionValueSaveReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := d.dataPermissionService.SaveUserDimensionValues(ctx, userId, data); err != nil {
		_ = ctx.Error(err)
		return
	}
}

func (d *DataPermissionController) DebugDataScope(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	user := ctx.MustGet("user").(model.SysUser)
	menuId := 0
	if rawMenuId := strings.TrimSpace(ctx.Query("menu_id")); rawMenuId != "" {
		parsed, err := strconv.Atoi(rawMenuId)
		if err != nil || parsed < 0 {
			_ = ctx.Error(myerrors.ErrParamInvalid)
			return
		}
		menuId = parsed
	}
	action := enum.ButtonActionQuery
	if rawAction := strings.TrimSpace(ctx.Query("action")); rawAction != "" {
		normalized, ok := enum.NormalizeSysMenuButtonEventAction(rawAction)
		if !ok {
			_ = ctx.Error(myerrors.ErrParamInvalid)
			return
		}
		action = normalized
	}
	result, err := d.dataPermissionService.DebugDataScope(user, menuId, ctx.Query("table_code"), action)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}
