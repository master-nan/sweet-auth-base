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
	dataResourceConfigTableCode   = "sys_data_resource"
	dataPolicyConfigTableCode     = "sys_data_policy"
	dataPolicyRuleConfigTableCode = "sys_data_policy_rule"
	dataGrantConfigTableCode      = "sys_data_grant"
)

type dataResourceConfigReader interface {
	GetResource(*gin.Context, int) (response.DataResourceDetailRes, error)
	PageResources(
		*gin.Context,
		request.DataResourceQueryReq,
		model.SysTable,
	) (response.ListResult[response.DataResourceListRes], error)
}

type dataOwnershipConfigReader interface {
	GetOwnership(*gin.Context, int) (response.DataOwnershipFieldDetailRes, error)
	ListOwnershipsByResource(*gin.Context, int) ([]response.DataOwnershipFieldListRes, error)
}

type dataPolicyConfigReader interface {
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
}

type dataGrantConfigReader interface {
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
}

// DataPermissionConfigController exposes the reviewed DP-2 read and preflight
// surface. Configuration writes remain behind their service boundary until a
// later task explicitly publishes them.
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

func (d *DataPermissionConfigController) QueryResources(ctx *gin.Context) {
	var req request.DataResourceQueryReq
	if !dataPermissionConfigBindQuery(d, ctx, &req) {
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

func (d *DataPermissionConfigController) QueryPolicies(ctx *gin.Context) {
	var req request.DataPolicyQueryReq
	if !dataPermissionConfigBindQuery(d, ctx, &req) {
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

func (d *DataPermissionConfigController) QueryPolicyRules(ctx *gin.Context) {
	var req request.DataPolicyRuleQueryReq
	if !dataPermissionConfigBindQuery(d, ctx, &req) {
		return
	}
	result, err := d.policyService.PagePolicyRules(ctx, req, dataPolicyRuleConfigTable())
	d.setListResult(ctx, result.Data, result.Total, err)
}

func (d *DataPermissionConfigController) QueryGrants(ctx *gin.Context) {
	var req request.DataGrantQueryReq
	if !dataPermissionConfigBindQuery(d, ctx, &req) {
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

func dataPermissionConfigBindQuery[T any](
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
