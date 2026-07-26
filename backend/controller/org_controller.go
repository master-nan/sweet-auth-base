package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

const orgLegalEntityTableCode = "org_legal_entity"

type orgTableProvider interface {
	GetTableByTableCode(string) (model.SysTable, error)
}

type OrgController struct {
	orgService       *service.OrgService
	sysTableProvider orgTableProvider
	translators      map[string]ut.Translator
}

func NewOrgController(
	orgService *service.OrgService,
	sysTableService *service.SysTableService,
	translators map[string]ut.Translator,
) *OrgController {
	return &OrgController{
		orgService:       orgService,
		sysTableProvider: sysTableService,
		translators:      translators,
	}
}

// QueryLegalEntities godoc
// @Summary 法人主体列表
// @Description 分页查询当前有效或显式请求的历史法人主体
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgLegalEntityQueryReq true "查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/legal-entity/query [post]
func (o *OrgController) QueryLegalEntities(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgLegalEntityQueryReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.legalEntityTable()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryLegalEntities(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetLegalEntityDetail godoc
// @Summary 法人主体详情
// @Description 按内部 legal_entity_id 查询法人主体详情
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "法人主体ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/legal-entity/{id} [get]
func (o *OrgController) GetLegalEntityDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "legal_entity_id必须为正整数"))
		return
	}
	var data request.OrgLegalEntityDetailReq
	if err = utils.ValidatorQuery(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetLegalEntityDetail(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// GetLegalEntityTree godoc
// @Summary 法人主体树
// @Description 使用 org_legal_entity.parent_id 组装法人主体树
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgLegalEntityTreeReq true "树查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/legal-entity/tree [post]
func (o *OrgController) GetLegalEntityTree(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgLegalEntityTreeReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetLegalEntityTree(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(len(result))
}

// QueryLegalEntityOptions godoc
// @Summary 法人主体选项
// @Description 查询以 legal_entity_id 为 value 的法人主体选项
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgLegalEntityOptionsReq true "选项查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/legal-entity/options [post]
func (o *OrgController) QueryLegalEntityOptions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgLegalEntityOptionsReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.legalEntityTable()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryLegalEntityOptions(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

func (o *OrgController) legalEntityTable() (model.SysTable, error) {
	table, err := o.sysTableProvider.GetTableByTableCode(orgLegalEntityTableCode)
	if err != nil {
		return model.SysTable{}, err
	}
	if table.Id == 0 {
		return model.SysTable{}, myerrors.ErrTableNotFound
	}
	table.TableCode = orgLegalEntityTableCode
	return table, nil
}

func (o *OrgController) translator() ut.Translator {
	if o.translators == nil {
		return nil
	}
	return o.translators["zh"]
}
