/**
 * @Author: Nan
 * @Date: 2024/6/13 下午11:30
 */

package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	queryutil "backend/repository/util"
	"backend/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type GeneralizationController struct {
	generalizationService *service.GeneralizationService
	sysTableService       *service.SysTableService
	sysMenuService        *service.SysMenuService
	dataPermissionService *service.DataPermissionService
	translators           map[string]ut.Translator
}

func NewGeneralizationController(generalizationService *service.GeneralizationService, sysTableService *service.SysTableService, sysMenuService *service.SysMenuService, dataPermissionService *service.DataPermissionService, translators map[string]ut.Translator) *GeneralizationController {
	return &GeneralizationController{
		generalizationService: generalizationService,
		sysTableService:       sysTableService,
		sysMenuService:        sysMenuService,
		dataPermissionService: dataPermissionService,
		translators:           translators,
	}
}

func (gc *GeneralizationController) dataScopeForMenu(ctx *gin.Context, table model.SysTable, menuId int, action enum.SysMenuButtonEventAction) (*request.DataScope, error) {
	user := ctx.MustGet("user").(model.SysUser)
	return gc.dataPermissionService.ResolveDataScope(user, menuId, table, action)
}

// injectDataScope 从 JWT 用户和请求中的 menu_id 注入数据权限范围
func (gc *GeneralizationController) injectDataScope(ctx *gin.Context, data *request.Basic, table model.SysTable, action enum.SysMenuButtonEventAction) error {
	if data.MenuId <= 0 {
		return nil
	}
	scope, err := gc.dataScopeForMenu(ctx, table, data.MenuId, action)
	if err != nil {
		return err
	}
	data.DataScope = scope
	return nil
}

func (gc *GeneralizationController) checkMenuPermission(ctx *gin.Context, menuId int, tableCode string) error {
	if menuId <= 0 {
		return nil
	}
	user := ctx.MustGet("user").(model.SysUser)
	hasPermission, err := gc.sysMenuService.HasUserMenuPermission(user.Id, menuId)
	if err != nil {
		return err
	}
	if !hasPermission {
		return myerrors.ErrPermissionDenied
	}
	menu, err := gc.sysMenuService.GetMenuById(menuId)
	if err != nil {
		return err
	}
	if menu.Id == 0 {
		return myerrors.ErrPermissionDenied
	}
	if menu.IsHidden || !menu.State {
		return myerrors.ErrPermissionDenied
	}
	if !utils.MenuAllowsTableCode(menu, tableCode) {
		return myerrors.ErrPermissionDenied
	}
	if strings.TrimSpace(menu.Option) == "" {
		hasPublishedMenu, err := gc.sysMenuService.HasPublishedTableMenu(tableCode)
		if err != nil {
			return err
		}
		if hasPublishedMenu {
			return myerrors.ErrPermissionDenied
		}
	}
	return nil
}

func (gc *GeneralizationController) resolveQueryMenuId(ctx *gin.Context, requestedMenuId int, tableCode string) (int, error) {
	return gc.resolveLowCodeMenuId(ctx, requestedMenuId, tableCode, enum.ButtonActionQuery, true)
}

func (gc *GeneralizationController) checkButtonActionPermission(ctx *gin.Context, menuId int, tableCode, action string) error {
	if menuId <= 0 {
		return myerrors.ErrPermissionDenied
	}
	if err := gc.checkMenuPermission(ctx, menuId, tableCode); err != nil {
		return err
	}
	user := ctx.MustGet("user").(model.SysUser)
	hasPermission, err := gc.sysMenuService.HasUserMenuButtonAction(user.Id, menuId, action)
	if err != nil {
		return err
	}
	if !hasPermission {
		return myerrors.ErrPermissionDenied
	}
	return nil
}

func (gc *GeneralizationController) resolveLowCodeMenuId(ctx *gin.Context, requestedMenuId int, tableCode string, action enum.SysMenuButtonEventAction, allowNoPublishedMenu bool) (int, error) {
	if requestedMenuId > 0 {
		if err := gc.checkButtonActionPermission(ctx, requestedMenuId, tableCode, string(action)); err != nil {
			return 0, err
		}
		return requestedMenuId, nil
	}
	user := ctx.MustGet("user").(model.SysUser)
	menuId, hasPublishedMenu, err := gc.sysMenuService.ResolvePublishedTableMenuId(user.Id, tableCode, action)
	if err != nil {
		return 0, err
	}
	if hasPublishedMenu {
		return menuId, nil
	}
	if allowNoPublishedMenu {
		return 0, nil
	}
	return 0, myerrors.ErrPermissionDenied
}

func (gc *GeneralizationController) resolveLowCodeMenuIdByActions(ctx *gin.Context, tableCode string, actions ...enum.SysMenuButtonEventAction) (int, error) {
	for _, action := range actions {
		menuId, err := gc.resolveLowCodeMenuId(ctx, 0, tableCode, action, false)
		if err == nil {
			return menuId, nil
		}
		if err != myerrors.ErrPermissionDenied {
			return 0, err
		}
	}
	return 0, myerrors.ErrPermissionDenied
}

func (gc *GeneralizationController) checkCreateDataPermission(ctx *gin.Context, table model.SysTable, menuId int, data map[string]interface{}) error {
	scope, err := gc.dataScopeForMenu(ctx, table, menuId, enum.ButtonActionCreate)
	if err != nil {
		return err
	}
	return validateDataScopeWriteValues(table, scope, data, true)
}

func validateDataScopeWriteValues(table model.SysTable, scope *request.DataScope, data map[string]interface{}, requireField bool) error {
	if scope == nil || scope.AllowAll {
		return nil
	}
	if scope.DenyAll {
		return myerrors.ErrPermissionDenied
	}
	for _, condition := range scope.Conditions {
		if !dataScopeConditionFieldExists(table, condition.Field) {
			return myerrors.ErrPermissionDenied
		}
		value, exists := data[condition.Field]
		if !exists {
			if !requireField {
				continue
			}
			return myerrors.ErrPermissionDenied
		}
		if !queryutil.DataScopeValueAllowed(table, condition, value) {
			return myerrors.ErrPermissionDenied
		}
	}
	return nil
}

func dataScopeConditionFieldExists(table model.SysTable, fieldCode string) bool {
	for _, field := range table.TableFields {
		if field.FieldCode == fieldCode {
			return true
		}
	}
	return false
}

func (gc *GeneralizationController) checkUpdateDataPermission(ctx *gin.Context, table model.SysTable, menuId int, data map[string]interface{}) error {
	scope, err := gc.dataScopeForMenu(ctx, table, menuId, enum.ButtonActionUpdate)
	if err != nil {
		return err
	}
	return validateDataScopeWriteValues(table, scope, data, false)
}

// checkRowDataPermission 检查用户对目标行数据的操作权限。
func (gc *GeneralizationController) checkRowDataPermission(ctx *gin.Context, table model.SysTable, rowId int, menuId int, action enum.SysMenuButtonEventAction) error {
	scope, err := gc.dataScopeForMenu(ctx, table, menuId, action)
	if err != nil {
		return err
	}
	if scope == nil || scope.AllowAll {
		return nil
	}
	if scope.DenyAll {
		return myerrors.ErrPermissionDenied
	}
	ok, err := gc.generalizationService.RowMatchesDataScope(table, rowId, scope)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return myerrors.ErrPermissionDenied
}

func (gc *GeneralizationController) Query(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := gc.sysTableService.GetTableById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if table.Id == 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.Basic
	translator := gc.translators["zh"]
	if err := utils.ValidatorBody[request.Basic](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	data.TableCode = table.TableCode
	menuId, err := gc.resolveQueryMenuId(ctx, data.MenuId, table.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data.MenuId = menuId
	if err := gc.injectDataScope(ctx, &data, table, enum.ButtonActionQuery); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := gc.generalizationService.Query(&data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

func (gc *GeneralizationController) QueryByCode(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := strings.TrimSpace(ctx.Param("code"))
	if code == "" {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.Basic
	translator := gc.translators["zh"]
	if err := utils.ValidatorBody[request.Basic](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := gc.sysTableService.GetTableByTableCode(code)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if table.Id == 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	data.TableCode = table.TableCode
	menuId, err := gc.resolveQueryMenuId(ctx, data.MenuId, table.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data.MenuId = menuId
	if err := gc.injectDataScope(ctx, &data, table, enum.ButtonActionQuery); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := gc.generalizationService.Query(&data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// DetailByCode 读取低代码记录详情；详情查看要求 detail 权限，编辑取表单初值可使用 update 权限
func (gc *GeneralizationController) DetailByCode(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := strings.TrimSpace(ctx.Param("code"))
	if code == "" {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	table, err := gc.sysTableService.GetTableByTableCode(code)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if table.Id == 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	requestedMenuId := 0
	if rawMenuId := strings.TrimSpace(ctx.Query("menu_id")); rawMenuId != "" {
		requestedMenuId, err = strconv.Atoi(rawMenuId)
		if err != nil || requestedMenuId <= 0 {
			_ = ctx.Error(myerrors.ErrParamInvalid)
			return
		}
	}
	menuId, err := gc.resolveLowCodeMenuId(ctx, requestedMenuId, table.TableCode, enum.ButtonActionDetail, false)
	if err != nil {
		menuId, err = gc.resolveLowCodeMenuId(ctx, requestedMenuId, table.TableCode, enum.ButtonActionUpdate, false)
	}
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := gc.checkRowDataPermission(ctx, table, id, menuId, enum.ButtonActionDetail); err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := gc.generalizationService.GetById(table, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// Create 通用新增
func (gc *GeneralizationController) Create(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.GeneralizationCreateReq
	translator := gc.translators["zh"]
	if err := utils.ValidatorBody[request.GeneralizationCreateReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := gc.sysTableService.GetTableByTableCode(data.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if table.Id == 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	menuId, err := gc.resolveLowCodeMenuId(ctx, data.MenuId, table.TableCode, enum.ButtonActionCreate, false)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data.MenuId = menuId
	if err := gc.checkCreateDataPermission(ctx, table, data.MenuId, data.Data); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := gc.generalizationService.Create(ctx, table, data.Data); err != nil {
		_ = ctx.Error(err)
		return
	}
}

// Update 通用更新
func (gc *GeneralizationController) Update(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.GeneralizationUpdateReq
	translator := gc.translators["zh"]
	if err := utils.ValidatorBody[request.GeneralizationUpdateReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := gc.sysTableService.GetTableByTableCode(data.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if table.Id == 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	menuId, err := gc.resolveLowCodeMenuId(ctx, data.MenuId, table.TableCode, enum.ButtonActionUpdate, false)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data.MenuId = menuId
	if err := gc.checkRowDataPermission(ctx, table, data.Id, data.MenuId, enum.ButtonActionUpdate); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := gc.checkUpdateDataPermission(ctx, table, data.MenuId, data.Data); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := gc.generalizationService.Update(ctx, table, data.Id, data.Data); err != nil {
		_ = ctx.Error(err)
		return
	}
}

// Delete 通用删除
func (gc *GeneralizationController) Delete(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.GeneralizationDeleteReq
	translator := gc.translators["zh"]
	if err := utils.ValidatorBody[request.GeneralizationDeleteReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := gc.sysTableService.GetTableByTableCode(data.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if table.Id == 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	menuId, err := gc.resolveLowCodeMenuId(ctx, data.MenuId, table.TableCode, enum.ButtonActionDelete, false)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data.MenuId = menuId
	if err := gc.checkRowDataPermission(ctx, table, data.Id, data.MenuId, enum.ButtonActionDelete); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := gc.generalizationService.Delete(ctx, table, data.Id); err != nil {
		_ = ctx.Error(err)
		return
	}
}
