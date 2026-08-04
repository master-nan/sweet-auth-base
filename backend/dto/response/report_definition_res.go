package response

import "encoding/json"

// ReportDefinitionListRes 仅包含报表列表和运行入口需要的字段。
type ReportDefinitionListRes struct {
	BasicRes
	Code                string `json:"code"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Category            string `json:"category"`
	Status              string `json:"status"`
	PublishedVersionId  int    `json:"published_version_id"`
	SourceType          string `json:"source_type"`
	SourceCode          string `json:"source_code"`
	PermissionMenuId    int    `json:"permission_menu_id"`
	PermissionTableCode string `json:"permission_table_code"`
	Remark              string `json:"remark"`
}

// ReportDefinitionDetailRes 在列表字段基础上开放报表设计器需要的结构化配置。
type ReportDefinitionDetailRes struct {
	ReportDefinitionListRes
	QueryConfig  json.RawMessage `json:"query_config"`
	LayoutConfig json.RawMessage `json:"layout_config"`
}
