/**
 * @Author: Nan
 * @Date: 2024/5/17 上午11:12
 */

package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	ut "github.com/go-playground/universal-translator"
)

type TableController struct {
	sysTableService *service.SysTableService
	sysMenuService  *service.SysMenuService
	translators     map[string]ut.Translator
}

func NewTableController(sysTableService *service.SysTableService, sysMenuService *service.SysMenuService, translators map[string]ut.Translator) *TableController {
	return &TableController{
		sysTableService,
		sysMenuService,
		translators,
	}
}

// GetTableByID 根据ID获取表
// @Summary 根据ID获取表
// @Description 根据ID获取表
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表ID"
// @Success 200 {object} response.Response
// @Router /admin/table/{id} [get]
func (t *TableController) GetTableByID(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysTableService.GetTableById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// GetTableByCode 根据表编码获取表
// @Summary 根据表编码获取表
// @Description 根据表编码获取表
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param code path string true "表编码"
// @Success 200 {object} response.Response
// @Router /admin/table/code/{code} [get]
func (t *TableController) GetTableByCode(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := utils.SanitizeInput(ctx.Param("code"))
	data, err := t.sysTableService.GetTableByTableCode(code)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// QueryTable 查询表列表
// @Summary 查询表列表
// @Description 查询表列表
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.Basic  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table/query [post]
func (t *TableController) QueryTable(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.Basic
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.Basic](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := t.sysTableService.GetTableList(&data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// CreateTable 创建表
// @Summary 创建表
// @Description 创建表
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.TableCreateReq  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table [post]
func (t *TableController) CreateTable(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.TableCreateReq
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.TableCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "create")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.CreateTable(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateTable 更新表
// @Summary 更新表
// @Description 更新表
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表ID"
// @Param data body request.TableUpdateReq  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table/{id} [put]
func (t *TableController) UpdateTable(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.NewBadRequestError(err.Error()))
		return
	}
	var data request.TableUpdateReq
	data.Id = id
	translator := t.translators["zh"]
	err = utils.ValidatorBody[request.TableUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "update")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.UpdateTable(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteTableById 根据ID删除表
// @Summary 根据ID删除表
// @Description 根据ID删除表
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表ID"
// @Success 200 {object} response.Response
// @Router /admin/table/{id} [delete]
func (t *TableController) DeleteTableById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "delete")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.DeleteTableById(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// GetTableFieldsByTableId 根据表ID获取表字段
// @Summary 根据表ID获取表字段
// @Description 根据表ID获取表字段
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表ID"
// @Success 200 {object} response.Response
// @Router /admin/table/fields/{id} [get]
func (t *TableController) GetTableFieldsByTableId(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysTableService.GetTableFieldsByTableId(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
	resp.SetTotal(len(data))
}

// GetTableFieldById 根据ID获取表字段
// @Summary 根据ID获取表字段
// @Description 根据ID获取表字段
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表字段ID"
// @Success 200 {object} response.Response
// @Router /admin/table/field/{id} [get]
func (t *TableController) GetTableFieldById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysTableService.GetTableFieldById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// CreateTableField 创建表字段
// @Summary 创建表字段
// @Description 创建表字段
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.TableFieldCreateReq  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table/field [post]
func (t *TableController) CreateTableField(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.TableFieldCreateReq
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.TableFieldCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.CreateTableField(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateTableField 更新表字段
// @Summary 更新表字段
// @Description 更新表字段
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表字段ID"
// @Param data body request.TableFieldUpdateReq  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table/field/{id} [put]
func (t *TableController) UpdateTableField(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.NewBadRequestError(err.Error()))
		return
	}
	var data request.TableFieldUpdateReq
	data.Id = id
	translator := t.translators["zh"]
	err = utils.ValidatorBody[request.TableFieldUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.UpdateTableField(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteTableFieldById 根据ID删除表字段
// @Summary 根据ID删除表字段
// @Description 根据ID删除表字段
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表字段ID"
// @Success 200 {object} response.Response
// @Router /admin/table/field/{id} [delete]
func (t *TableController) DeleteTableFieldById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.DeleteTableFieldById(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// GetTableRelationsByTableId 根据表ID获取表关系
// @Summary 根据表ID获取表关系
// @Description 根据表ID获取表关系
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表ID"
// @Success 200 {object} response.Response
// @Router /admin/table/relations/{id} [get]
func (t *TableController) GetTableRelationsByTableId(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysTableService.GetTableRelationsByTableId(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetTotal(len(data))
	resp.SetData(data)
}

// GetTableRelationById 根据ID获取表关系
// @Summary 根据ID获取表关系
// @Description 根据ID获取表关系
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表关系ID"
// @Success 200 {object} response.Response
// @Router /admin/table/relation/{id} [get]
func (t *TableController) GetTableRelationById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysTableService.GetTableRelationById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// CreateTableRelation 创建表关系
// @Summary 创建表关系
// @Description 创建表关系
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.TableRelationCreateReq  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table/relation [post]
func (t *TableController) CreateTableRelation(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.TableRelationCreateReq
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.TableRelationCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.CreateTableRelation(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateTableRelation 更新表关系
// @Summary 更新表关系
// @Description 更新表关系
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表关系ID"
// @Param data body request.TableRelationUpdateReq  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table/relation/{id} [put]
func (t *TableController) UpdateTableRelation(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.NewBadRequestError(err.Error()))
		return
	}
	var data request.TableRelationUpdateReq
	data.Id = id
	translator := t.translators["zh"]
	err = utils.ValidatorBody[request.TableRelationUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.UpdateTableRelation(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteTableRelationById 根据ID删除表关系
// @Summary 根据ID删除表关系
// @Description 根据ID删除表关系
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表关系ID"
// @Success 200 {object} response.Response
// @Router /admin/table/relation/{id} [delete]
func (t *TableController) DeleteTableRelationById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.DeleteTableRelationById(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// GetTableIndexById 根据ID获取表索引
// @Summary 根据ID获取表索引
// @Description 根据ID获取表索引
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表索引ID"
// @Success 200 {object} response.Response
// @Router /admin/table/index/{id} [get]
func (t *TableController) GetTableIndexById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysTableService.GetTableIndexById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// GetTableIndexesByTableId 根据表ID获取表索引
// @Summary 根据表ID获取表索引
// @Description 根据表ID获取表索引
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表ID"
// @Success 200 {object} response.Response
// @Router /admin/table/indexes/{id} [get]
func (t *TableController) GetTableIndexesByTableId(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := t.sysTableService.GetTableIndexesByTableId(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(data)
}

// CreateTableIndex 创建表索引
// @Summary 创建表索引
// @Description 创建表索引
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.TableIndexCreateReq  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table/index [post]
func (t *TableController) CreateTableIndex(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.TableIndexCreateReq
	translator := t.translators["zh"]
	err := utils.ValidatorBody[request.TableIndexCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.CreateTableIndex(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateTableIndex 更新表索引
// @Summary 更新表索引
// @Description 更新表索引
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表索引ID"
// @Param data body request.TableIndexUpdateReq  true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/table/index/{id} [put]
func (t *TableController) UpdateTableIndex(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.NewBadRequestError(err.Error()))
		return
	}
	var data request.TableIndexUpdateReq
	data.Id = id
	translator := t.translators["zh"]
	err = utils.ValidatorBody[request.TableIndexUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.UpdateTableIndex(ctx, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteTableIndexById 根据ID删除表索引
// @Summary 根据ID删除表索引
// @Description 根据ID删除表索引
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表索引ID"
// @Success 200 {object} response.Response
// @Router /admin/table/index/{id} [delete]
func (t *TableController) DeleteTableIndexById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.DeleteTableIndexById(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// DeleteTableIndexByTableId 根据表ID删除表索引
// @Summary 根据表ID删除表索引
// @Description 根据表ID删除表索引
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "表ID"
// @Success 200 {object} response.Response
// @Router /admin/table/index/table/{id} [delete]
func (t *TableController) DeleteTableIndexByTableId(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.DeleteTableIndexByTableId(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// InitTable 初始化表
// @Summary 初始化表
// @Description 初始化表
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param code path string true "表编码"
// @Success 200 {object} response.Response
// @Router /admin/table/init/{code} [post]
func (t *TableController) InitTable(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := utils.SanitizeInput(ctx.Param("code"))
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "init_meta")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.InitTable(ctx, code)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// SyncTableFields 同步表字段元数据
// @Summary 同步表字段元数据
// @Description 将数据库表结构差异同步到 sys_table_field
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param code path string true "表编码"
// @Success 200 {object} response.Response
// @Router /admin/table/sync/{code} [post]
func (t *TableController) SyncTableFields(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := utils.SanitizeInput(ctx.Param("code"))
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "sync_fields")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.SyncTableFields(ctx, code)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// SyncTableIndexes 同步表索引元数据
// @Summary 同步表索引元数据
// @Description 将数据库表索引同步到 sys_table_index 与 sys_table_index_field
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param code path string true "表编码"
// @Success 200 {object} response.Response
// @Router /admin/table/sync/index/{code} [post]
func (t *TableController) SyncTableIndexes(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := utils.SanitizeInput(ctx.Param("code"))
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "field_manager")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.SyncTableIndexes(ctx, code)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// PublishTable 发布通用CRUD菜单
// @Summary 发布通用CRUD菜单
// @Description 为表配置生成侧边栏菜单和默认CRUD按钮
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param code path string true "表编码"
// @Param data body request.TablePublishReq false "发布目录"
// @Success 200 {object} response.Response
// @Router /admin/table/publish/{code} [post]
func (t *TableController) PublishTable(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := utils.SanitizeInput(ctx.Param("code"))
	var data request.TablePublishReq
	if ctx.Request.ContentLength != 0 {
		if err := ctx.ShouldBindBodyWith(&data, binding.JSON); err != nil {
			if err != io.EOF {
				_ = ctx.Error(err)
				return
			}
		}
	}
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "publish")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.PublishTableAsMenu(ctx, code, data.ParentId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UnpublishTable 下线通用CRUD菜单
// @Summary 下线通用CRUD菜单
// @Description 隐藏表配置生成的低代码菜单并撤销角色授权
// @Tags 表
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param code path string true "表编码"
// @Success 200 {object} response.Response
// @Router /admin/table/unpublish/{code} [post]
func (t *TableController) UnpublishTable(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := utils.SanitizeInput(ctx.Param("code"))
	user := ctx.MustGet("user").(model.SysUser)
	allowed, err := t.sysMenuService.HasUserMenuButtonActionByMenuName(user.Id, "develop_database", "unpublish")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if !allowed {
		_ = ctx.Error(myerrors.ErrPermissionDenied)
		return
	}
	err = t.sysTableService.UnpublishTableMenu(ctx, code)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}
