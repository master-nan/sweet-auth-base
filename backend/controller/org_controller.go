package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

const (
	orgLegalEntityTableCode = "org_legal_entity"
	orgStructureTableCode   = "org_structure"
	orgUnitTableCode        = "org_unit"
	orgEmployeeTableCode    = "org_employee"
	orgPositionTableCode    = "org_position"
	orgAssignmentTableCode  = "org_assignment"
)

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

// QueryStructures godoc
// @Summary 管理架构列表
// @Description 分页查询管理架构定义
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgStructureQueryReq true "查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/structure/query [post]
func (o *OrgController) QueryStructures(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgStructureQueryReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgStructureTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryStructures(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetStructureDetail godoc
// @Summary 管理架构详情
// @Description 按内部 structure_id 查询管理架构详情
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "管理架构ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/structure/{id} [get]
func (o *OrgController) GetStructureDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "structure_id必须为正整数"))
		return
	}
	var data request.OrgStructureDetailReq
	if err = utils.ValidatorQuery(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetStructureDetail(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// QueryStructureOptions godoc
// @Summary 管理架构选项
// @Description 查询以 structure_id 为 value 的管理架构选项
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgStructureOptionsReq true "选项查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/structure/options [post]
func (o *OrgController) QueryStructureOptions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgStructureOptionsReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgStructureTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryStructureOptions(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// QueryOrgUnits godoc
// @Summary 管理组织列表
// @Description 分页查询管理组织单元
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgUnitQueryReq true "查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/unit/query [post]
func (o *OrgController) QueryOrgUnits(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgUnitQueryReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgUnitTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryOrgUnits(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetOrgUnitDetail godoc
// @Summary 管理组织详情
// @Description 按内部 org_unit_id 查询组织单元详情
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "组织单元ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/unit/{id} [get]
func (o *OrgController) GetOrgUnitDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "org_unit_id必须为正整数"))
		return
	}
	var data request.OrgUnitDetailReq
	if err = utils.ValidatorQuery(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetOrgUnitDetail(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// QueryOrgUnitOptions godoc
// @Summary 管理组织选项
// @Description 查询以 org_unit_id 为 value 的管理组织选项
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgUnitOptionsReq true "选项查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/unit/options [post]
func (o *OrgController) QueryOrgUnitOptions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgUnitOptionsReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgUnitTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryOrgUnitOptions(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetStructureOrgTree godoc
// @Summary 管理组织树
// @Description 使用 org_structure_node.parent_node_id 组装指定管理架构的组织树
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgStructureOrgTreeReq true "树查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/unit/tree [post]
func (o *OrgController) GetStructureOrgTree(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgStructureOrgTreeReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetStructureOrgTree(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result).SetTotal(countStructureOrgTreeNodes(result))
}

// QueryEmployees godoc
// @Summary 企业人员列表
// @Description 分页查询组织镜像中的企业人员
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgEmployeeQueryReq true "查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/employee/query [post]
func (o *OrgController) QueryEmployees(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgEmployeeQueryReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgEmployeeTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryEmployees(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetEmployeeDetail godoc
// @Summary 企业人员详情
// @Description 按内部 employee_id 查询企业人员及安全账号摘要
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "企业人员ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/employee/{id} [get]
func (o *OrgController) GetEmployeeDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "employee_id必须为正整数"))
		return
	}
	var data request.OrgEmployeeDetailReq
	if err = utils.ValidatorQuery(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetEmployeeDetail(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// QueryEmployeeOptions godoc
// @Summary 企业人员选项
// @Description 查询以 employee_id 为 value 的企业人员选项
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgEmployeeOptionsReq true "选项查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/employee/options [post]
func (o *OrgController) QueryEmployeeOptions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgEmployeeOptionsReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgEmployeeTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryEmployeeOptions(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// BindEmployeeUser godoc
// @Summary 绑定企业人员账号
// @Description 将企业人员绑定到明确指定的当前 Sweet Platform 登录账号
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "企业人员ID"
// @Param data body request.OrgEmployeeBindUserReq true "账号绑定参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/employee/{id}/bind-user [post]
func (o *OrgController) BindEmployeeUser(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "employee_id必须为正整数"))
		return
	}
	data := request.OrgEmployeeBindUserReq{EmployeeId: id}
	middleware.SetAuditContext(ctx, middleware.AuditContext{
		Action:       "bind_user",
		ResourceType: "org_employee",
		ResourceCode: "org_employee",
		ResourceID:   strconv.Itoa(id),
	})
	if err = utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.BindEmployeeUser(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	middleware.MarkAccessAuditPersisted(ctx)
	resp.SetData(result)
}

// UnbindEmployeeUser godoc
// @Summary 解绑企业人员账号
// @Description 清除企业人员与当前 Sweet Platform 登录账号的绑定
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "企业人员ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/employee/{id}/unbind-user [post]
func (o *OrgController) UnbindEmployeeUser(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "employee_id必须为正整数"))
		return
	}
	data := request.OrgEmployeeUnbindUserReq{EmployeeId: id}
	middleware.SetAuditContext(ctx, middleware.AuditContext{
		Action:       "unbind_user",
		ResourceType: "org_employee",
		ResourceCode: "org_employee",
		ResourceID:   strconv.Itoa(id),
	})
	if err = utils.ValidateStruct(data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.UnbindEmployeeUser(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	middleware.MarkAccessAuditPersisted(ctx)
	resp.SetData(result)
}

// QueryPositions godoc
// @Summary 岗位列表
// @Description 分页查询组织镜像中的岗位
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgPositionQueryReq true "查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/position/query [post]
func (o *OrgController) QueryPositions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgPositionQueryReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgPositionTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryPositions(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetPositionDetail godoc
// @Summary 岗位详情
// @Description 按内部 position_id 查询岗位详情
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "岗位ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/position/{id} [get]
func (o *OrgController) GetPositionDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "position_id必须为正整数"))
		return
	}
	var data request.OrgPositionDetailReq
	if err = utils.ValidatorQuery(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetPositionDetail(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// QueryPositionOptions godoc
// @Summary 岗位选项
// @Description 查询以 position_id 为 value 的岗位选项
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgPositionOptionsReq true "选项查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/position/options [post]
func (o *OrgController) QueryPositionOptions(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgPositionOptionsReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgPositionTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryPositionOptions(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// QueryAssignments godoc
// @Summary 企业人员任职列表
// @Description 按员工和时间范围查询只读任职镜像
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.OrgAssignmentQueryReq true "查询参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/assignment/query [post]
func (o *OrgController) QueryAssignments(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var data request.OrgAssignmentQueryReq
	if err := utils.ValidatorBody(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := utils.ValidatePagination(data.Page, data.Num); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgAssignmentTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.QueryAssignments(ctx, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetAssignmentDetail godoc
// @Summary 企业人员任职详情
// @Description 按内部 assignment_id 查询只读任职详情
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "任职ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/assignment/{id} [get]
func (o *OrgController) GetAssignmentDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "assignment_id必须为正整数"))
		return
	}
	var data request.OrgAssignmentDetailReq
	if err = utils.ValidatorQuery(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetAssignmentDetail(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// GetEmployeeCurrentAssignmentSummary godoc
// @Summary 企业人员当前任职归属摘要
// @Description 返回员工全部当前有效任职对应的法人、组织和岗位集合
// @Tags 组织主数据
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "企业人员ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/org/employee/{id}/assignments/summary [get]
func (o *OrgController) GetEmployeeCurrentAssignmentSummary(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.WrapParameterError(err, "employee_id必须为正整数"))
		return
	}
	var data request.OrgEmployeeCurrentAssignmentSummaryReq
	if err = utils.ValidatorQuery(ctx, &data, o.translator()); err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := o.organizationTable(orgAssignmentTableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := o.orgService.GetEmployeeCurrentAssignmentSummary(ctx, id, data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (o *OrgController) legalEntityTable() (model.SysTable, error) {
	return o.organizationTable(orgLegalEntityTableCode)
}

func (o *OrgController) organizationTable(tableCode string) (model.SysTable, error) {
	table, err := o.sysTableProvider.GetTableByTableCode(tableCode)
	if err != nil {
		return model.SysTable{}, err
	}
	if table.Id == 0 {
		return model.SysTable{}, myerrors.ErrTableNotFound
	}
	table.TableCode = tableCode
	return table, nil
}

func countStructureOrgTreeNodes(nodes []response.OrgStructureOrgTreeNodeRes) int {
	total := 0
	stack := append([]response.OrgStructureOrgTreeNodeRes(nil), nodes...)
	for len(stack) > 0 {
		index := len(stack) - 1
		node := stack[index]
		stack = stack[:index]
		total++
		stack = append(stack, node.Children...)
	}
	return total
}

func (o *OrgController) translator() ut.Translator {
	if o.translators == nil {
		return nil
	}
	return o.translators["zh"]
}
