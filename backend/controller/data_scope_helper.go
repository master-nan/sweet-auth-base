package controller

import (
	"backend/dto/request"
	"backend/enum"
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
