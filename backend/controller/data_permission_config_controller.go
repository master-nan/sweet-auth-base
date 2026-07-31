package controller

import (
	"strconv"

	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

const (
	dataDimensionConfigTableCode  = "sys_data_dimension_definition"
	dataResourceConfigTableCode   = "sys_data_resource"
	dataOwnershipConfigTableCode  = "sys_data_ownership_field"
	dataPolicyConfigTableCode     = "sys_data_policy"
	dataPolicyRuleConfigTableCode = "sys_data_policy_rule"
	dataGrantConfigTableCode      = "sys_data_grant"
)

type dataResourceConfigReader interface {
	CreateResource(*gin.Context, request.DataResourceCreateReq) (response.DataResourceDetailRes, error)
	UpdateResource(*gin.Context, request.DataResourceUpdateReq) (response.DataResourceDetailRes, error)
	GetResource(*gin.Context, int) (response.DataResourceDetailRes, error)
	PageResources(
		*gin.Context,
		request.DataResourceQueryReq,
		model.SysTable,
	) (response.ListResult[response.DataResourceListRes], error)
	ListResourceOperations(*gin.Context, int) ([]response.DataResourceOperationListRes, error)
	ReplaceResourceOperations(
		*gin.Context,
		request.DataResourceOperationBatchReq,
	) ([]response.DataResourceOperationListRes, error)
}

type dataOwnershipConfigReader interface {
	PageDimensions(
		*gin.Context,
		request.DataDimensionDefinitionQueryReq,
		model.SysTable,
	) (response.ListResult[response.DataDimensionDefinitionListRes], error)
	CreateOwnership(
		*gin.Context,
		request.DataOwnershipFieldCreateReq,
	) (response.DataOwnershipFieldDetailRes, error)
	UpdateOwnership(
		*gin.Context,
		request.DataOwnershipFieldUpdateReq,
	) (response.DataOwnershipFieldDetailRes, error)
	GetOwnership(*gin.Context, int) (response.DataOwnershipFieldDetailRes, error)
	PageOwnerships(
		*gin.Context,
		request.DataOwnershipFieldQueryReq,
		model.SysTable,
	) (response.ListResult[response.DataOwnershipFieldListRes], error)
	ListOwnershipsByResource(*gin.Context, int) ([]response.DataOwnershipFieldListRes, error)
}

type dataPolicyConfigReader interface {
	CreatePolicy(*gin.Context, request.DataPolicyCreateReq) (response.DataPolicyDetailRes, error)
	UpdatePolicy(*gin.Context, request.DataPolicyUpdateReq) (response.DataPolicyDetailRes, error)
	GetPolicy(*gin.Context, int) (response.DataPolicyDetailRes, error)
	PagePolicies(
		*gin.Context,
		request.DataPolicyQueryReq,
		model.SysTable,
	) (response.ListResult[response.DataPolicyListRes], error)
	PagePolicyRules(
		*gin.Context,
		request.DataPolicyRuleQueryReq,
		model.SysTable,
	) (response.ListResult[response.DataPolicyRuleListRes], error)
	ReplacePolicyRules(
		*gin.Context,
		request.DataPolicyRuleBatchReq,
	) ([]response.DataPolicyRuleListRes, error)
}

type dataGrantConfigReader interface {
	CreateGrant(*gin.Context, request.DataGrantCreateReq) (response.DataGrantDetailRes, error)
	GetGrant(*gin.Context, int) (response.DataGrantDetailRes, error)
	PageGrants(
		*gin.Context,
		request.DataGrantQueryReq,
		model.SysTable,
	) (response.ListResult[response.DataGrantListRes], error)
}

type dataPermissionConfigPreflightReader interface {
	PreflightResource(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
	PreflightPolicy(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
	PreflightGrant(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
	EnableResource(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
	DisableResource(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
	EnablePolicy(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
	DisablePolicy(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
	EnableGrant(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
	DisableGrant(*gin.Context, int) (response.DataPermissionValidationResultRes, error)
}

// DataPermissionConfigController publishes the reviewed DP-2 configuration
// services without moving configuration rules into the HTTP layer.
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
	result, err := d.ownershipService.PageDimensions(ctx, req, dataDimensionConfigTable())
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) CreateResource(ctx *gin.Context) {
	var req request.DataResourceCreateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.resourceService.CreateResource(ctx, req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryResources(ctx *gin.Context) {
	var req request.DataResourceQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.resourceService.PageResources(ctx, req, dataResourceConfigTable())
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) GetResource(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.resourceService.GetResource(ctx, id)
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
	result, err := d.resourceService.UpdateResource(ctx, req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) ListResourceOperations(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.resourceService.ListResourceOperations(ctx, id)
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
	result, err := d.resourceService.ReplaceResourceOperations(ctx, req)
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
		result, err = d.preflightService.EnableResource(ctx, id)
	} else {
		result, err = d.preflightService.DisableResource(ctx, id)
	}
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryOwnerships(ctx *gin.Context) {
	var req request.DataOwnershipFieldQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.ownershipService.PageOwnerships(ctx, req, dataOwnershipConfigTable())
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) CreateOwnership(ctx *gin.Context) {
	var req request.DataOwnershipFieldCreateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.ownershipService.CreateOwnership(ctx, req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) ListOwnershipsByResource(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.ownershipService.ListOwnershipsByResource(ctx, id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) GetOwnership(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.ownershipService.GetOwnership(ctx, id)
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
	result, err := d.ownershipService.UpdateOwnership(ctx, req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) CreatePolicy(ctx *gin.Context) {
	var req request.DataPolicyCreateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.policyService.CreatePolicy(ctx, req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryPolicies(ctx *gin.Context) {
	var req request.DataPolicyQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.policyService.PagePolicies(ctx, req, dataPolicyConfigTable())
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) GetPolicy(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.policyService.GetPolicy(ctx, id)
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
	result, err := d.policyService.UpdatePolicy(ctx, req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryPolicyRules(ctx *gin.Context) {
	var req request.DataPolicyRuleQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.policyService.PagePolicyRules(ctx, req, dataPolicyRuleConfigTable())
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
	result, err := d.policyService.ReplacePolicyRules(ctx, req)
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
		result, err = d.preflightService.EnablePolicy(ctx, id)
	} else {
		result, err = d.preflightService.DisablePolicy(ctx, id)
	}
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) CreateGrant(ctx *gin.Context) {
	var req request.DataGrantCreateReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.grantService.CreateGrant(ctx, req)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) QueryGrants(ctx *gin.Context) {
	var req request.DataGrantQueryReq
	if !dataPermissionConfigBindBody(d, ctx, &req) {
		return
	}
	result, err := d.grantService.PageGrants(ctx, req, dataGrantConfigTable())
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) GetGrant(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.grantService.GetGrant(ctx, id)
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
		result, err = d.preflightService.EnableGrant(ctx, id)
	} else {
		result, err = d.preflightService.DisableGrant(ctx, id)
	}
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) PreflightResource(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.preflightService.PreflightResource(ctx, id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) PreflightPolicy(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.preflightService.PreflightPolicy(ctx, id)
	d.setResult(ctx, result, err)
}

func (d *DataPermissionConfigController) PreflightGrant(ctx *gin.Context) {
	id, ok := dataPermissionConfigPathID(ctx)
	if !ok {
		return
	}
	result, err := d.preflightService.PreflightGrant(ctx, id)
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

func dataResourceConfigTable() model.SysTable {
	return dataPermissionConfigTable(dataResourceConfigTableCode,
		dataPermissionConfigField("resource_code", enum.VarcharFieldType, true),
		dataPermissionConfigField("name", enum.VarcharFieldType, true),
		dataPermissionConfigField("resource_type", enum.VarcharFieldType, false),
	)
}

func dataDimensionConfigTable() model.SysTable {
	return dataPermissionConfigTable(dataDimensionConfigTableCode,
		dataPermissionConfigField("code", enum.VarcharFieldType, true),
		dataPermissionConfigField("name", enum.VarcharFieldType, true),
		dataPermissionConfigField("category", enum.VarcharFieldType, false),
		dataPermissionConfigField("value_type", enum.VarcharFieldType, false),
	)
}

func dataOwnershipConfigTable() model.SysTable {
	return dataPermissionConfigTable(dataOwnershipConfigTableCode,
		dataPermissionConfigField("resource_id", enum.BigIntFieldType, false),
		dataPermissionConfigField("ownership_code", enum.VarcharFieldType, true),
		dataPermissionConfigField("dimension_id", enum.BigIntFieldType, false),
		dataPermissionConfigField("binding_type", enum.VarcharFieldType, false),
		dataPermissionConfigField("value_type", enum.VarcharFieldType, false),
	)
}

func dataPolicyConfigTable() model.SysTable {
	return dataPermissionConfigTable(dataPolicyConfigTableCode,
		dataPermissionConfigField("code", enum.VarcharFieldType, true),
		dataPermissionConfigField("name", enum.VarcharFieldType, true),
		dataPermissionConfigField("policy_type", enum.VarcharFieldType, false),
	)
}

func dataPolicyRuleConfigTable() model.SysTable {
	return dataPermissionConfigTable(dataPolicyRuleConfigTableCode,
		dataPermissionConfigField("policy_id", enum.BigIntFieldType, false),
		dataPermissionConfigField("ownership_code", enum.VarcharFieldType, true),
		dataPermissionConfigField("sequence", enum.IntFieldType, false),
	)
}

func dataGrantConfigTable() model.SysTable {
	return dataPermissionConfigTable(dataGrantConfigTableCode,
		dataPermissionConfigField("subject_type", enum.VarcharFieldType, true),
		dataPermissionConfigField("subject_id", enum.BigIntFieldType, false),
		dataPermissionConfigField("resource_id", enum.BigIntFieldType, false),
		dataPermissionConfigField("operation", enum.VarcharFieldType, false),
		dataPermissionConfigField("policy_id", enum.BigIntFieldType, false),
	)
}

func dataPermissionConfigTable(
	tableCode string,
	fields ...model.SysTableField,
) model.SysTable {
	return model.SysTable{
		Basic:       model.Basic{State: true},
		TableCode:   tableCode,
		TableFields: fields,
	}
}

func dataPermissionConfigField(
	code string,
	fieldType enum.SysTableFieldType,
	quickSearch bool,
) model.SysTableField {
	return model.SysTableField{
		Basic:            model.Basic{State: true},
		FieldCode:        code,
		FieldType:        fieldType,
		IsListShow:       true,
		IsQuickSearch:    quickSearch,
		IsAdvancedSearch: true,
		IsSort:           true,
	}
}
