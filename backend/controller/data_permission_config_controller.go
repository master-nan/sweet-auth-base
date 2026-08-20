package controller

import (
	"context"
	"strconv"

	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/service"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type dataResourceConfigReader interface {
	CreateResource(context.Context, request.DataResourceCreateReq) (response.DataResourceDetailRes, error)
	UpdateResource(context.Context, request.DataResourceUpdateReq) (response.DataResourceDetailRes, error)
	GetResource(context.Context, int) (response.DataResourceDetailRes, error)
	PageResources(
		context.Context,
		request.DataResourceQueryReq,
	) (response.ListResult[response.DataResourceListRes], error)
	ListResourceOperations(context.Context, int) ([]response.DataResourceOperationListRes, error)
	ReplaceResourceOperations(
		context.Context,
		request.DataResourceOperationBatchReq,
	) ([]response.DataResourceOperationListRes, error)
}

type dataOwnershipConfigReader interface {
	PageDimensions(
		context.Context,
		request.DataDimensionDefinitionQueryReq,
	) (response.ListResult[response.DataDimensionDefinitionListRes], error)
	CreateOwnership(
		context.Context,
		request.DataOwnershipFieldCreateReq,
	) (response.DataOwnershipFieldDetailRes, error)
	UpdateOwnership(
		context.Context,
		request.DataOwnershipFieldUpdateReq,
	) (response.DataOwnershipFieldDetailRes, error)
	GetOwnership(context.Context, int) (response.DataOwnershipFieldDetailRes, error)
	PageOwnerships(
		context.Context,
		request.DataOwnershipFieldQueryReq,
	) (response.ListResult[response.DataOwnershipFieldListRes], error)
	ListOwnershipsByResource(context.Context, int) ([]response.DataOwnershipFieldListRes, error)
}

type dataPolicyConfigReader interface {
	CreatePolicy(context.Context, request.DataPolicyCreateReq) (response.DataPolicyDetailRes, error)
	UpdatePolicy(context.Context, request.DataPolicyUpdateReq) (response.DataPolicyDetailRes, error)
	GetPolicy(context.Context, int) (response.DataPolicyDetailRes, error)
	PagePolicies(
		context.Context,
		request.DataPolicyQueryReq,
	) (response.ListResult[response.DataPolicyListRes], error)
	PagePolicyRules(
		context.Context,
		request.DataPolicyRuleQueryReq,
	) (response.ListResult[response.DataPolicyRuleListRes], error)
	ReplacePolicyRules(
		context.Context,
		request.DataPolicyRuleBatchReq,
	) ([]response.DataPolicyRuleListRes, error)
}

type dataGrantConfigReader interface {
	CreateGrant(context.Context, request.DataGrantCreateReq) (response.DataGrantDetailRes, error)
	GetGrant(context.Context, int) (response.DataGrantDetailRes, error)
	PageGrants(
		context.Context,
		request.DataGrantQueryReq,
	) (response.ListResult[response.DataGrantListRes], error)
}

type dataPermissionConfigPreflightReader interface {
	PreflightResource(context.Context, int) (response.DataPermissionValidationResultRes, error)
	PreflightPolicy(context.Context, int) (response.DataPermissionValidationResultRes, error)
	PreflightGrant(context.Context, int) (response.DataPermissionValidationResultRes, error)
	EnableResource(context.Context, int) (response.DataPermissionValidationResultRes, error)
	DisableResource(context.Context, int) (response.DataPermissionValidationResultRes, error)
	EnablePolicy(context.Context, int) (response.DataPermissionValidationResultRes, error)
	DisablePolicy(context.Context, int) (response.DataPermissionValidationResultRes, error)
	EnableGrant(context.Context, int) (response.DataPermissionValidationResultRes, error)
	DisableGrant(context.Context, int) (response.DataPermissionValidationResultRes, error)
}

// DataPermissionConfigController 开放经过审查的数据权限配置 Service，
// 不将配置规则移入 HTTP 层。
type DataPermissionConfigController struct {
	resourceService  dataResourceConfigReader
	ownershipService dataOwnershipConfigReader
	policyService    dataPolicyConfigReader
	grantService     dataGrantConfigReader
	preflightService dataPermissionConfigPreflightReader
	translators      map[string]ut.Translator
}

func NewDataPermissionConfigController(
	resourceService *service.DataResourceConfigService,
	ownershipService *service.DataOwnershipConfigService,
	policyService *service.DataPolicyConfigService,
	grantService *service.DataGrantConfigService,
	preflightService *service.DataPermissionConfigPreflightService,
	translators map[string]ut.Translator,
) *DataPermissionConfigController {
	return newDataPermissionConfigController(
		resourceService,
		ownershipService,
		policyService,
		grantService,
		preflightService,
		translators,
	)
}

func newDataPermissionConfigController(
	resourceService dataResourceConfigReader,
	ownershipService dataOwnershipConfigReader,
	policyService dataPolicyConfigReader,
	grantService dataGrantConfigReader,
	preflightService dataPermissionConfigPreflightReader,
	translators map[string]ut.Translator,
) *DataPermissionConfigController {
	return &DataPermissionConfigController{
		resourceService:  resourceService,
		ownershipService: ownershipService,
		policyService:    policyService,
		grantService:     grantService,
		preflightService: preflightService,
		translators:      translators,
	}
}

func (d *DataPermissionConfigController) QueryDimensions(ctx *gin.Context) {
	var req request.DataDimensionDefinitionQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.ownershipService.PageDimensions(ctx.Request.Context(), req)
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) CreateResource(ctx *gin.Context) {
	var req request.DataResourceCreateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.resourceService.CreateResource(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryResources(ctx *gin.Context) {
	var req request.DataResourceQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.resourceService.PageResources(ctx.Request.Context(), req)
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) GetResource(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.resourceService.GetResource(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) UpdateResource(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	req := request.DataResourceUpdateReq{Id: id}
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	req.Id = id
	result, err := d.resourceService.UpdateResource(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) ListResourceOperations(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.resourceService.ListResourceOperations(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) ReplaceResourceOperations(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	req := request.DataResourceOperationBatchReq{ResourceId: id}
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	req.ResourceId = id
	result, err := d.resourceService.ReplaceResourceOperations(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) SetResourcePermission(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	var req request.DataResourcePermissionStateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	var (
		result response.DataPermissionValidationResultRes
		err    error
	)
	if *req.PermissionEnabled {
		result, err = d.preflightService.EnableResource(ctx.Request.Context(), id)
	} else {
		result, err = d.preflightService.DisableResource(ctx.Request.Context(), id)
	}
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryOwnerships(ctx *gin.Context) {
	var req request.DataOwnershipFieldQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.ownershipService.PageOwnerships(ctx.Request.Context(), req)
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) CreateOwnership(ctx *gin.Context) {
	var req request.DataOwnershipFieldCreateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.ownershipService.CreateOwnership(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) ListOwnershipsByResource(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.ownershipService.ListOwnershipsByResource(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) GetOwnership(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.ownershipService.GetOwnership(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) UpdateOwnership(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	req := request.DataOwnershipFieldUpdateReq{Id: id}
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	req.Id = id
	result, err := d.ownershipService.UpdateOwnership(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) CreatePolicy(ctx *gin.Context) {
	var req request.DataPolicyCreateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.policyService.CreatePolicy(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryPolicies(ctx *gin.Context) {
	var req request.DataPolicyQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.policyService.PagePolicies(ctx.Request.Context(), req)
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) GetPolicy(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.policyService.GetPolicy(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) UpdatePolicy(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	req := request.DataPolicyUpdateReq{Id: id}
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	req.Id = id
	result, err := d.policyService.UpdatePolicy(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryPolicyRules(ctx *gin.Context) {
	var req request.DataPolicyRuleQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.policyService.PagePolicyRules(ctx.Request.Context(), req)
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) ReplacePolicyRules(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	req := request.DataPolicyRuleBatchReq{PolicyId: id}
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	req.PolicyId = id
	result, err := d.policyService.ReplacePolicyRules(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) SetPolicyState(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	var req request.DataPermissionConfigStateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	var (
		result response.DataPermissionValidationResultRes
		err    error
	)
	if *req.State {
		result, err = d.preflightService.EnablePolicy(ctx.Request.Context(), id)
	} else {
		result, err = d.preflightService.DisablePolicy(ctx.Request.Context(), id)
	}
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) CreateGrant(ctx *gin.Context) {
	var req request.DataGrantCreateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.grantService.CreateGrant(ctx.Request.Context(), req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryGrants(ctx *gin.Context) {
	var req request.DataGrantQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.grantService.PageGrants(ctx.Request.Context(), req)
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) GetGrant(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.grantService.GetGrant(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) SetGrantState(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	var req request.DataPermissionConfigStateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	var (
		result response.DataPermissionValidationResultRes
		err    error
	)
	if *req.State {
		result, err = d.preflightService.EnableGrant(ctx.Request.Context(), id)
	} else {
		result, err = d.preflightService.DisableGrant(ctx.Request.Context(), id)
	}
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) PreflightResource(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.preflightService.PreflightResource(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) PreflightPolicy(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.preflightService.PreflightPolicy(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) PreflightGrant(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.preflightService.PreflightGrant(ctx.Request.Context(), id)
	d.setResult(ctx, result, err)
}

func dataPermissionConfigBindBody[T any](
	d *DataPermissionConfigController,
	ctx *gin.Context,
	value *T,
) bool {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err := utils.ValidatorBody(ctx, value, d.translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return false
	}
	return true
}

func (d *DataPermissionConfigController) setResult(
	ctx *gin.Context,
	result any,
	err error,
) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (d *DataPermissionConfigController) setListResult(
	ctx *gin.Context,
	result any,
	total int,
	err error,
) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(total)
}

func dataPermissionConfigPathID(ctx *gin.Context) (int, bool) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return 0, false
	}
	return id, true
}
