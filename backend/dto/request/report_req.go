package request

import "gorm.io/datatypes"

type ReportDefinitionCreateReq struct {
	Code                string         `json:"code" binding:"required" example:"sales_summary"`
	Name                string         `json:"name" binding:"required" example:"销售汇总"`
	Description         string         `json:"description" example:"按组织和日期查看销售汇总"`
	Category            string         `json:"category" example:"经营分析"`
	SourceType          string         `json:"source_type" example:"table"`
	SourceCode          string         `json:"source_code" example:"tms_waybill"`
	PermissionMenuId    int            `json:"permission_menu_id" example:"0"`
	PermissionTableCode string         `json:"permission_table_code" example:"tms_waybill"`
	QueryConfig         datatypes.JSON `json:"query_config" binding:"omitempty,non_empty_json"`
	LayoutConfig        datatypes.JSON `json:"layout_config" binding:"omitempty,non_empty_json"`
	Remark              string         `json:"remark"`
	State               bool           `json:"state"`
}

type ReportDefinitionUpdateReq struct {
	Id                  int            `json:"id" example:"1"`
	Code                string         `json:"code" binding:"required" example:"sales_summary"`
	Name                string         `json:"name" binding:"required" example:"销售汇总"`
	Description         string         `json:"description" example:"按组织和日期查看销售汇总"`
	Category            string         `json:"category" example:"经营分析"`
	SourceType          string         `json:"source_type" example:"table"`
	SourceCode          string         `json:"source_code" example:"tms_waybill"`
	PermissionMenuId    int            `json:"permission_menu_id" example:"0"`
	PermissionTableCode string         `json:"permission_table_code" example:"tms_waybill"`
	QueryConfig         datatypes.JSON `json:"query_config" binding:"omitempty,non_empty_json"`
	LayoutConfig        datatypes.JSON `json:"layout_config" binding:"omitempty,non_empty_json"`
	Remark              string         `json:"remark"`
	State               bool           `json:"state"`
}

type ReportPreviewReq struct {
	MenuId     int            `json:"menu_id" example:"401"`
	DatasetId  string         `json:"dataset_id" example:"main"`
	Parameters map[string]any `json:"parameters"`
	Query      Basic          `json:"query"`
}

type ReportSQLFieldsReq struct {
	SQL string `json:"sql" binding:"required" example:"select id, name from sys_user"`
}
