/**
 * @Author: Nan
 * @Date: 2024/7/20 上午10:27
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/database"
	"backend/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AccessLogRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.AccessLog]
}

func NewAccessLogRepositoryImpl(PrimaryDB *database.PrimaryDB) *AccessLogRepositoryImpl {
	return &AccessLogRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.AccessLog{}),
	}
}

func (a *AccessLogRepositoryImpl) GetAccessLogList(ctx *gin.Context, basic *request.Basic) (response.ListResult[model.AccessLog], error) {
	var repo response.ListResult[model.AccessLog]
	var accessLogList []model.AccessLog
	total, err := a.
		WithContext(ctx).
		WithSelect(accessLogListColumns()...).
		PaginateAndCountAsync(basic, &accessLogList, accessLogTable())
	repo.Data = accessLogList
	repo.Total = int(total)
	return repo, err
}

func accessLogListColumns() []string {
	return []string{
		"id",
		"gmt_create",
		"user_id",
		"user_name",
		"method",
		"ip",
		"url",
		"action",
		"resource_type",
		"resource_code",
		"resource_id",
		"menu_id",
		"status_code",
		"success",
		"duration_ms",
		"state",
	}
}

func accessLogTable() model.SysTable {
	return model.SysTable{
		TableCode: "access_log",
		TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true, IsListShow: true, IsSort: true},
			{FieldCode: "gmt_create", FieldType: enum.DatetimeFieldType, IsListShow: true, IsSort: true},
			{FieldCode: "user_id", FieldType: enum.BigIntFieldType, IsListShow: true, IsSort: true},
			{FieldCode: "user_name", FieldType: enum.VarcharFieldType, IsListShow: true, IsQuickSearch: true},
			{FieldCode: "method", FieldType: enum.VarcharFieldType, IsListShow: true},
			{FieldCode: "ip", FieldType: enum.VarcharFieldType, IsListShow: true, IsQuickSearch: true},
			{FieldCode: "url", FieldType: enum.VarcharFieldType, IsListShow: true, IsQuickSearch: true},
			{FieldCode: "action", FieldType: enum.VarcharFieldType, IsListShow: true, IsQuickSearch: true},
			{FieldCode: "resource_type", FieldType: enum.VarcharFieldType, IsListShow: true},
			{FieldCode: "resource_code", FieldType: enum.VarcharFieldType, IsListShow: true, IsQuickSearch: true},
			{FieldCode: "resource_id", FieldType: enum.VarcharFieldType, IsListShow: true},
			{FieldCode: "menu_id", FieldType: enum.BigIntFieldType, IsListShow: true},
			{FieldCode: "status_code", FieldType: enum.IntFieldType, IsListShow: true, IsSort: true},
			{FieldCode: "success", FieldType: enum.BooleanFieldType, IsListShow: true, IsSort: true},
			{FieldCode: "duration_ms", FieldType: enum.BigIntFieldType, IsListShow: true, IsSort: true},
			{FieldCode: "state", FieldType: enum.BooleanFieldType, IsListShow: true},
		},
	}
}
