package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	orgSyncBatchTableCode  = "org_sync_batch"
	orgSyncRecordTableCode = "org_sync_record"
)

func (o *OrgController) QuerySyncBatches(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgSyncBatchQueryReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgSyncBatchTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QuerySyncBatches(ctx.Request.Context(), data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

func (o *OrgController) GetSyncBatchDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := positiveOrganizationID(ctx, "batch_id")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetSyncBatchDetail(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (o *OrgController) GetSyncBatchError(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := positiveOrganizationID(ctx, "batch_id")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetSyncBatchError(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (o *OrgController) QuerySyncRecords(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgSyncRecordQueryReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgSyncRecordTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QuerySyncRecords(ctx.Request.Context(), data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

func (o *OrgController) GetSyncRecordDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := positiveOrganizationID(ctx, "record_id")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetSyncRecordDetail(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (o *OrgController) GetSyncRecordError(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := positiveOrganizationID(ctx, "record_id")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetSyncRecordError(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func positiveOrganizationID(ctx *gin.Context, fieldName string) (int, error) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return 0, myerrors.WrapParameterError(err, fieldName+"必须为正整数")
	}
	if id <= 0 {
		return 0, myerrors.NewParameterError(fieldName + "必须为正整数")
	}
	return id, nil
}
