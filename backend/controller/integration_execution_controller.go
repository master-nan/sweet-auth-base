package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/middleware"
	"backend/model"
	"backend/repository"
	"backend/service"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

const integrationExecutionTableCode = "integration_execution"
const integrationLogTableCode = "integration_log"

type integrationExecutionApplication interface {
	CreateExecution(context.Context, request.IntegrationExecutionCreateReq) (response.IntegrationExecutionDetailRes, error)
	CancelExecution(context.Context, int, int) (response.IntegrationExecutionDetailRes, error)
	GetExecution(context.Context, int, model.SysTable, repository.GeneralizationPermission) (response.IntegrationExecutionDetailRes, error)
	PageExecution(context.Context, request.IntegrationExecutionQueryReq, model.SysTable, repository.GeneralizationPermission) (response.ListResult[response.IntegrationExecutionListRes], error)
	GetLog(context.Context, int, model.SysTable, repository.GeneralizationPermission) (response.IntegrationLogDetailRes, error)
	PageLogs(context.Context, request.IntegrationLogQueryReq, model.SysTable, repository.GeneralizationPermission) (response.ListResult[response.IntegrationLogListRes], error)
}

type integrationExecutionTableProvider interface {
	GetTableByTableCode(string) (model.SysTable, error)
}

type integrationExecutionPermissionResolver interface {
	ResolveDataPermission(*gin.Context, model.SysTable, string) (repository.GeneralizationPermission, error)
}

type IntegrationExecutionController struct {
	service            integrationExecutionApplication
	tableProvider      integrationExecutionTableProvider
	permissionResolver integrationExecutionPermissionResolver
	translators        map[string]ut.Translator
}

func NewIntegrationExecutionController(
	executionService *service.IntegrationExecutionService,
	tableService *service.SysTableService,
	generalizationService *service.GeneralizationService,
	translators map[string]ut.Translator,
) *IntegrationExecutionController {
	return newIntegrationExecutionController(executionService, tableService, generalizationService, translators)
}

func newIntegrationExecutionController(
	executionService integrationExecutionApplication,
	tableProvider integrationExecutionTableProvider,
	permissionResolver integrationExecutionPermissionResolver,
	translators map[string]ut.Translator,
) *IntegrationExecutionController {
	return &IntegrationExecutionController{
		service: executionService, tableProvider: tableProvider,
		permissionResolver: permissionResolver, translators: translators,
	}
}

func (c *IntegrationExecutionController) Query(ctx *gin.Context) {
	var req request.IntegrationExecutionQueryReq
	if !bindIntegrationExecution(ctx, &req, c.translators) {
		return
	}
	table, permission, ok := c.resolveReadPermission(ctx, model.DataPermissionOperationQuery)
	if !ok {
		return
	}
	result, err := c.service.PageExecution(ctx.Request.Context(), req, table, permission)
	c.setListResult(ctx, result.Data, result.Total, err)
}

func (c *IntegrationExecutionController) Detail(ctx *gin.Context) {
	id, ok := integrationExecutionPathID(ctx)
	if !ok {
		return
	}
	table, permission, ok := c.resolveReadPermission(ctx, model.DataPermissionOperationDetail)
	if !ok {
		return
	}
	result, err := c.service.GetExecution(ctx.Request.Context(), id, table, permission)
	c.setResult(ctx, result, err, false)
}

func (c *IntegrationExecutionController) QueryLogs(ctx *gin.Context) {
	var req request.IntegrationLogQueryReq
	if !bindIntegrationExecution(ctx, &req, c.translators) {
		return
	}
	table, permission, ok := c.resolvePermissionForTable(ctx, integrationLogTableCode, model.DataPermissionOperationQuery)
	if !ok {
		return
	}
	result, err := c.service.PageLogs(ctx.Request.Context(), req, table, permission)
	c.setListResult(ctx, result.Data, result.Total, err)
}

func (c *IntegrationExecutionController) LogDetail(ctx *gin.Context) {
	id, ok := integrationExecutionPathID(ctx)
	if !ok {
		return
	}
	table, permission, ok := c.resolvePermissionForTable(ctx, integrationLogTableCode, model.DataPermissionOperationDetail)
	if !ok {
		return
	}
	result, err := c.service.GetLog(ctx.Request.Context(), id, table, permission)
	c.setResult(ctx, result, err, false)
}

func (c *IntegrationExecutionController) Create(ctx *gin.Context) {
	var req request.IntegrationExecutionCreateReq
	if !bindIntegrationExecution(ctx, &req, c.translators) {
		return
	}
	result, err := c.service.CreateExecution(ctx.Request.Context(), req)
	c.setResult(ctx, result, err, true)
}

func (c *IntegrationExecutionController) Cancel(ctx *gin.Context) {
	c.changeState(ctx, func(requestContext context.Context, id int, req request.IntegrationExecutionStateReq) (response.IntegrationExecutionDetailRes, error) {
		return c.service.CancelExecution(requestContext, id, req.Revision)
	})
}

func (c *IntegrationExecutionController) changeState(
	ctx *gin.Context,
	command func(context.Context, int, request.IntegrationExecutionStateReq) (response.IntegrationExecutionDetailRes, error),
) {
	id, ok := integrationExecutionPathID(ctx)
	if !ok {
		return
	}
	var req request.IntegrationExecutionStateReq
	if !bindIntegrationExecution(ctx, &req, c.translators) {
		return
	}
	result, err := command(ctx.Request.Context(), id, req)
	c.setResult(ctx, result, err, true)
}

func (c *IntegrationExecutionController) resolveReadPermission(
	ctx *gin.Context,
	operation string,
) (model.SysTable, repository.GeneralizationPermission, bool) {
	table, err := c.tableProvider.GetTableByTableCode(integrationExecutionTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return model.SysTable{}, repository.GeneralizationPermission{}, false
	}
	permission, err := c.permissionResolver.ResolveDataPermission(ctx, table, operation)
	if err != nil {
		_ = ctx.Error(err)
		return model.SysTable{}, repository.GeneralizationPermission{}, false
	}
	return table, permission, true
}

func (c *IntegrationExecutionController) resolvePermissionForTable(
	ctx *gin.Context,
	tableCode string,
	operation string,
) (model.SysTable, repository.GeneralizationPermission, bool) {
	table, err := c.tableProvider.GetTableByTableCode(tableCode)
	if err != nil {
		_ = ctx.Error(err)
		return model.SysTable{}, repository.GeneralizationPermission{}, false
	}
	permission, err := c.permissionResolver.ResolveDataPermission(ctx, table, operation)
	if err != nil {
		_ = ctx.Error(err)
		return model.SysTable{}, repository.GeneralizationPermission{}, false
	}
	return table, permission, true
}

func bindIntegrationExecution[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func (c *IntegrationExecutionController) setResult(ctx *gin.Context, result any, err error, audited bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if audited {
		middleware.MarkAccessAuditPersisted(ctx)
	}
	resp.SetData(result)
}

func (c *IntegrationExecutionController) setListResult(ctx *gin.Context, result any, total int, err error) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(total)
}

func integrationExecutionPathID(ctx *gin.Context) (int, bool) {
	ctx.Set("response", response.NewResponse())
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}
