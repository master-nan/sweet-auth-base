package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/middleware"
	"backend/service"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type syncTaskApplication interface {
	CreateSyncTask(context.Context, request.SyncTaskCreateReq) (response.SyncTaskDetailRes, error)
	PageSyncTask(context.Context, request.SyncTaskQueryReq) (response.ListResult[response.SyncTaskListRes], error)
	GetSyncTask(context.Context, int) (response.SyncTaskDetailRes, error)
	GetSyncTaskForEdit(context.Context, int) (response.SyncTaskEditRes, error)
	UpdateDraftSyncTask(context.Context, int, request.SyncTaskUpdateReq) (response.SyncTaskDetailRes, error)
	CreateSyncTaskVersion(context.Context, int, int) (response.SyncTaskDetailRes, error)
	EnableSyncTask(context.Context, int, int) (response.SyncTaskDetailRes, error)
	DisableSyncTask(context.Context, int, int) (response.SyncTaskDetailRes, error)
	RunSyncTask(context.Context, int, int) (response.SyncBatchDetailRes, error)
	ListSyncConsumers(context.Context) []response.SyncConsumerMetadataRes
}

type syncBatchApplication interface {
	PageSyncBatch(context.Context, request.SyncBatchQueryReq) (response.ListResult[response.SyncBatchListRes], error)
	GetSyncBatch(context.Context, int) (response.SyncBatchDetailRes, error)
}

type IntegrationSyncController struct {
	tasks       syncTaskApplication
	batches     syncBatchApplication
	translators map[string]ut.Translator
}

func NewIntegrationSyncController(tasks *service.SyncTaskService, batches *service.SyncBatchService, translators map[string]ut.Translator) *IntegrationSyncController {
	return &IntegrationSyncController{tasks: tasks, batches: batches, translators: translators}
}

func (c *IntegrationSyncController) QueryTasks(ctx *gin.Context) {
	var req request.SyncTaskQueryReq
	if !bindIntegrationSync(ctx, &req, c.translators) {
		return
	}
	result, err := c.tasks.PageSyncTask(ctx.Request.Context(), req)
	c.setList(ctx, result.Data, result.Total, err)
}

func (c *IntegrationSyncController) TaskDetail(ctx *gin.Context) {
	id, ok := integrationSyncPathID(ctx)
	if !ok {
		return
	}
	result, err := c.tasks.GetSyncTask(ctx.Request.Context(), id)
	c.setResult(ctx, result, err, false)
}

func (c *IntegrationSyncController) TaskEdit(ctx *gin.Context) {
	id, ok := integrationSyncPathID(ctx)
	if !ok {
		return
	}
	result, err := c.tasks.GetSyncTaskForEdit(ctx.Request.Context(), id)
	c.setResult(ctx, result, err, false)
}

func (c *IntegrationSyncController) CreateTask(ctx *gin.Context) {
	var req request.SyncTaskCreateReq
	if !bindIntegrationSync(ctx, &req, c.translators) {
		return
	}
	result, err := c.tasks.CreateSyncTask(ctx.Request.Context(), req)
	c.setResult(ctx, result, err, true)
}

func (c *IntegrationSyncController) UpdateTask(ctx *gin.Context) {
	id, ok := integrationSyncPathID(ctx)
	if !ok {
		return
	}
	var req request.SyncTaskUpdateReq
	if !bindIntegrationSync(ctx, &req, c.translators) {
		return
	}
	result, err := c.tasks.UpdateDraftSyncTask(ctx.Request.Context(), id, req)
	c.setResult(ctx, result, err, true)
}

func (c *IntegrationSyncController) CreateTaskVersion(ctx *gin.Context) {
	id, ok := integrationSyncPathID(ctx)
	if !ok {
		return
	}
	var req request.SyncTaskVersionReq
	if !bindIntegrationSync(ctx, &req, c.translators) {
		return
	}
	result, err := c.tasks.CreateSyncTaskVersion(ctx.Request.Context(), id, req.Revision)
	c.setResult(ctx, result, err, true)
}

func (c *IntegrationSyncController) EnableTask(ctx *gin.Context)  { c.changeTaskState(ctx, true) }
func (c *IntegrationSyncController) DisableTask(ctx *gin.Context) { c.changeTaskState(ctx, false) }

func (c *IntegrationSyncController) RunTask(ctx *gin.Context) {
	id, ok := integrationSyncPathID(ctx)
	if !ok {
		return
	}
	var req request.SyncTaskRunReq
	if !bindIntegrationSync(ctx, &req, c.translators) {
		return
	}
	result, err := c.tasks.RunSyncTask(ctx.Request.Context(), id, req.Revision)
	c.setResult(ctx, result, err, true)
}

func (c *IntegrationSyncController) changeTaskState(ctx *gin.Context, enable bool) {
	id, ok := integrationSyncPathID(ctx)
	if !ok {
		return
	}
	var req request.SyncTaskStateReq
	if !bindIntegrationSync(ctx, &req, c.translators) {
		return
	}
	var result response.SyncTaskDetailRes
	var err error
	if enable {
		result, err = c.tasks.EnableSyncTask(ctx.Request.Context(), id, req.Revision)
	} else {
		result, err = c.tasks.DisableSyncTask(ctx.Request.Context(), id, req.Revision)
	}
	c.setResult(ctx, result, err, true)
}

func (c *IntegrationSyncController) ConsumerMetadata(ctx *gin.Context) {
	c.setResult(ctx, c.tasks.ListSyncConsumers(ctx.Request.Context()), nil, false)
}

func (c *IntegrationSyncController) QueryBatches(ctx *gin.Context) {
	var req request.SyncBatchQueryReq
	if !bindIntegrationSync(ctx, &req, c.translators) {
		return
	}
	result, err := c.batches.PageSyncBatch(ctx.Request.Context(), req)
	c.setList(ctx, result.Data, result.Total, err)
}

func (c *IntegrationSyncController) BatchDetail(ctx *gin.Context) {
	id, ok := integrationSyncPathID(ctx)
	if !ok {
		return
	}
	result, err := c.batches.GetSyncBatch(ctx.Request.Context(), id)
	c.setResult(ctx, result, err, false)
}

func bindIntegrationSync[T any](ctx *gin.Context, value *T, translators map[string]ut.Translator) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func integrationSyncPathID(ctx *gin.Context) (int, bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}

func (c *IntegrationSyncController) setResult(ctx *gin.Context, result any, err error, audited bool) {
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

func (c *IntegrationSyncController) setList(ctx *gin.Context, data any, total int, err error) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data).SetTotal(total)
}
