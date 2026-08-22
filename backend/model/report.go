package model

import "gorm.io/datatypes"

// ReportDefinition 保存可复用的报表定义。
//
// 标准报表模型通过 query_config/layout_config 保存数据集、参数和工作表布局。
// SourceCode和SourceType是列表与搜索所需的受控冗余，权威配置仍来自主表Dataset。
// SQL 数据集只能通过报表 Service 的只读预览守卫执行。
type ReportDefinition struct {
	Basic
	Code                string         `gorm:"size:128;uniqueIndex:uni_report_definition_code;comment:报表编码" json:"code"`
	Name                string         `gorm:"size:128;comment:报表名称" json:"name"`
	Description         string         `gorm:"size:512;comment:报表说明" json:"description"`
	Category            string         `gorm:"size:128;comment:报表分类" json:"category"`
	Status              string         `gorm:"size:32;default:draft;index:idx_report_definition_status;comment:报表状态（draft:草稿,published:已发布,disabled:已停用）" json:"status"`
	PublishedVersionId  int            `gorm:"index:idx_report_definition_published_version;comment:当前发布版本ID" json:"published_version_id"`
	SourceType          string         `gorm:"size:32;default:table;comment:数据源类型" json:"source_type"`
	SourceCode          string         `gorm:"size:128;index:idx_report_definition_source;comment:数据源表/视图编码" json:"source_code"`
	PermissionMenuId    int            `gorm:"comment:数据权限菜单ID" json:"permission_menu_id"`
	PermissionTableCode string         `gorm:"size:128;comment:数据权限表编码" json:"permission_table_code"`
	QueryConfig         datatypes.JSON `gorm:"type:jsonb;comment:查询配置JSON" json:"query_config"`
	LayoutConfig        datatypes.JSON `gorm:"type:jsonb;comment:布局配置JSON" json:"layout_config"`
	Remark              string         `gorm:"size:256;comment:备注" json:"remark"`
}

type ReportDefinitionVersion struct {
	Basic
	ReportId            int            `gorm:"index:idx_report_definition_version_report;uniqueIndex:uni_report_definition_version_no;comment:报表ID" json:"report_id"`
	VersionNo           int            `gorm:"uniqueIndex:uni_report_definition_version_no;comment:版本号" json:"version_no"`
	ReportCode          string         `gorm:"size:128;comment:报表编码" json:"report_code"`
	ReportName          string         `gorm:"size:128;comment:报表名称" json:"report_name"`
	Description         string         `gorm:"size:512;comment:报表说明" json:"description"`
	Category            string         `gorm:"size:128;comment:报表分类" json:"category"`
	SourceType          string         `gorm:"size:32;comment:数据源类型" json:"source_type"`
	SourceCode          string         `gorm:"size:128;comment:数据源表/视图编码" json:"source_code"`
	PermissionMenuId    int            `gorm:"comment:数据权限菜单ID" json:"permission_menu_id"`
	PermissionTableCode string         `gorm:"size:128;comment:数据权限表编码" json:"permission_table_code"`
	QueryConfig         datatypes.JSON `gorm:"type:jsonb;comment:查询配置JSON快照" json:"query_config"`
	LayoutConfig        datatypes.JSON `gorm:"type:jsonb;comment:布局配置JSON快照" json:"layout_config"`
	Status              string         `gorm:"size:32;default:published;index:idx_report_definition_version_status;comment:版本状态（published:当前发布,archived:历史归档）" json:"status"`
	PublishedAt         CustomTime     `gorm:"type:timestamp;index:idx_report_definition_version_published_at;comment:发布时间" json:"published_at"`
	PublishedBy         int            `gorm:"comment:发布人ID" json:"published_by"`
	PublishedName       string         `gorm:"size:128;comment:发布人名称" json:"published_name"`
	ChangeLog           string         `gorm:"size:512;comment:发布说明" json:"change_log"`
}

type ReportExecutionLog struct {
	Basic
	ReportId     int            `gorm:"index:idx_report_execution_log_report;comment:报表ID" json:"report_id"`
	ReportCode   string         `gorm:"size:128;index:idx_report_execution_log_code;comment:报表编码" json:"report_code"`
	UserId       int            `gorm:"index:idx_report_execution_log_user;comment:执行用户ID" json:"user_id"`
	UserName     string         `gorm:"size:128;comment:执行用户名" json:"user_name"`
	Action       string         `gorm:"size:32;comment:执行动作" json:"action"`
	Params       datatypes.JSON `gorm:"type:jsonb;comment:执行参数JSON" json:"params"`
	Success      bool           `gorm:"default:true;comment:是否成功" json:"success"`
	DurationMs   int64          `gorm:"comment:耗时毫秒" json:"duration_ms"`
	RowCount     int            `gorm:"comment:返回行数" json:"row_count"`
	ErrorMessage string         `gorm:"type:text;comment:错误信息" json:"error_message"`
}
