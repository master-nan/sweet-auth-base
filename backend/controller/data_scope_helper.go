package controller

import (
	"backend/dto/request"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
)

func injectQueryDataScope(ctx *gin.Context, dataPermissionService *service.DataPermissionService, data *request.Basic, table model.SysTable) error {
	if dataPermissionService == nil || data == nil {
		return nil
	}
	ctxUser, exists := ctx.Get("user")
	if !exists {
		return nil
	}
	user, ok := ctxUser.(model.SysUser)
	if !ok {
		return nil
	}
	scope, err := dataPermissionService.ResolveDataScopeForTableAction(user, data.MenuId, table, enum.ButtonActionQuery)
	if err != nil {
		return err
	}
	data.DataScope = scope
	return nil
}

func checkRecordDataScopeByTableCode(ctx *gin.Context, sysTableService *service.SysTableService, dataPermissionService *service.DataPermissionService, tableCode string, id int, action enum.SysMenuButtonEventAction) error {
	if sysTableService == nil || dataPermissionService == nil {
		return nil
	}
	table, err := sysTableService.GetTableByTableCode(tableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrParamInvalid
	}
	return checkRecordDataScope(ctx, dataPermissionService, table, id, action)
}

func checkRecordDataScope(ctx *gin.Context, dataPermissionService *service.DataPermissionService, table model.SysTable, id int, action enum.SysMenuButtonEventAction) error {
	if dataPermissionService == nil {
		return nil
	}
	ctxUser, exists := ctx.Get("user")
	if !exists {
		return nil
	}
	user, ok := ctxUser.(model.SysUser)
	if !ok {
		return nil
	}
	return dataPermissionService.CheckRecordDataScope(user, 0, table, id, action)
}
