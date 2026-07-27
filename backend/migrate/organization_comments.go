package main

import (
	"backend/model"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type databaseCommentSpec struct {
	model          any
	tableComment   string
	columnComments map[string]string
}

var platformBasicColumnComments = map[string]string{
	"id":          "主键ID",
	"gmt_create":  "创建时间",
	"create_user": "创建人ID",
	"create_name": "创建人",
	"gmt_modify":  "修改时间",
	"modify_user": "修改人ID",
	"modify_name": "修改人",
	"gmt_delete":  "删除时间",
	"delete_user": "删除人ID",
	"delete_name": "删除人",
}

var organizationSharedColumnComments = map[string]string{
	"state":                 "状态",
	"source_system_code":    "源系统编码",
	"source_id":             "源对象ID",
	"source_code":           "源对象编码",
	"code":                  "编码",
	"name":                  "名称",
	"status":                "状态",
	"valid_from":            "生效时间",
	"valid_to":              "失效时间",
	"source_version":        "源数据版本",
	"source_updated_at":     "源数据更新时间",
	"last_sync_at":          "最后同步时间",
	"source_status":         "源对象状态",
	"source_deleted":        "源对象删除标记",
	"sync_status":           "同步状态",
	"last_error":            "最后同步错误",
	"local_note":            "本地备注",
	"local_tags":            "本地标签",
	"display_order":         "显示顺序",
	"local_handling_status": "本地处理状态",
}

var organizationDatabaseCommentSpecs = []databaseCommentSpec{
	{
		model:        &model.OrgLegalEntity{},
		tableComment: "法人主体主数据镜像",
		columnComments: map[string]string{
			"short_name":                 "简称",
			"entity_type":                "法人主体类型",
			"parent_id":                  "上级法人主体ID",
			"unified_social_credit_code": "统一社会信用代码",
			"accounting_code":            "核算编码",
		},
	},
	{
		model:        &model.OrgUnit{},
		tableComment: "管理组织单元主数据镜像",
		columnComments: map[string]string{
			"unit_type":               "管理组织类型",
			"primary_legal_entity_id": "主要归属法人主体ID",
		},
	},
	{
		model:        &model.OrgStructure{},
		tableComment: "管理组织架构主数据镜像",
		columnComments: map[string]string{
			"structure_type": "管理架构类型",
			"is_default":     "默认架构标记",
		},
	},
	{
		model:        &model.OrgStructureNode{},
		tableComment: "管理组织架构节点关系镜像",
		columnComments: map[string]string{
			"structure_id":     "管理架构ID",
			"org_unit_id":      "管理组织单元ID",
			"parent_node_id":   "父节点ID",
			"source_parent_id": "源父节点ID",
			"path":             "节点路径",
			"level":            "节点层级",
			"sort":             "排序值",
		},
	},
	{
		model:        &model.OrgPosition{},
		tableComment: "岗位主数据镜像",
		columnComments: map[string]string{
			"org_unit_id":         "管理组织单元ID",
			"position_type":       "岗位类型",
			"job_level":           "岗位职级",
			"is_manager_position": "管理岗位标记",
		},
	},
	{
		model:        &model.OrgEmployee{},
		tableComment: "企业人员主数据镜像",
		columnComments: map[string]string{
			"employee_no":             "员工编号",
			"mobile":                  "手机号",
			"email":                   "邮箱",
			"employment_status":       "在职状态",
			"primary_legal_entity_id": "主要归属法人主体ID",
			"user_id":                 "当前应用账号ID",
		},
	},
	{
		model:        &model.OrgAssignment{},
		tableComment: "员工任职关系镜像",
		columnComments: map[string]string{
			"employee_id":     "员工ID",
			"legal_entity_id": "法人主体ID",
			"org_unit_id":     "管理组织单元ID",
			"position_id":     "岗位ID",
			"assignment_type": "任职类型",
			"is_primary":      "主任职标记",
			"is_manager":      "管理任职标记",
		},
	},
	{
		model:        &model.OrgSyncBatch{},
		tableComment: "组织主数据同步批次",
		columnComments: map[string]string{
			"batch_no":      "同步批次号",
			"execution_id":  "集成执行ID",
			"sync_type":     "同步类型",
			"object_scope":  "同步对象范围",
			"started_at":    "开始时间",
			"completed_at":  "完成时间",
			"total_count":   "总记录数",
			"success_count": "成功记录数",
			"failed_count":  "失败记录数",
			"skipped_count": "跳过记录数",
			"error_summary": "错误摘要",
		},
	},
	{
		model:        &model.OrgSyncRecord{},
		tableComment: "组织主数据同步记录",
		columnComments: map[string]string{
			"batch_id":        "同步批次ID",
			"execution_id":    "集成执行ID",
			"object_type":     "对象类型",
			"local_id":        "本地对象ID",
			"action":          "同步动作",
			"error_code":      "错误码",
			"error_message":   "错误信息",
			"dependency_type": "依赖对象类型",
			"dependency_key":  "依赖对象键",
			"retry_count":     "重试次数",
			"last_retry_at":   "最后重试时间",
		},
	},
}

func applyOrganizationDatabaseComments(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, spec := range organizationDatabaseCommentSpecs {
			if err := applyDatabaseCommentSpec(tx, spec); err != nil {
				return err
			}
		}
		return nil
	})
}

func applyDatabaseCommentSpec(db *gorm.DB, spec databaseCommentSpec) error {
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(spec.model); err != nil {
		return fmt.Errorf("parse organization comment model: %w", err)
	}

	comments, err := completeOrganizationColumnComments(statement.Schema.DBNames, spec.columnComments)
	if err != nil {
		return fmt.Errorf("validate comments for %s: %w", statement.Schema.Table, err)
	}
	tableName, err := quotePostgresQualifiedIdentifier(statement.Schema.Table)
	if err != nil {
		return fmt.Errorf("quote organization table %s: %w", statement.Schema.Table, err)
	}
	if err := db.Exec(fmt.Sprintf(
		"COMMENT ON TABLE %s IS %s",
		tableName,
		quotePostgresLiteral(spec.tableComment),
	)).Error; err != nil {
		return fmt.Errorf("comment organization table %s: %w", statement.Schema.Table, err)
	}

	columnNames := make([]string, 0, len(comments))
	for columnName := range comments {
		columnNames = append(columnNames, columnName)
	}
	sort.Strings(columnNames)
	for _, columnName := range columnNames {
		quotedColumn, quoteErr := quotePostgresIdentifier(columnName)
		if quoteErr != nil {
			return fmt.Errorf("quote organization column %s.%s: %w", statement.Schema.Table, columnName, quoteErr)
		}
		sql := fmt.Sprintf(
			"COMMENT ON COLUMN %s.%s IS %s",
			tableName,
			quotedColumn,
			quotePostgresLiteral(comments[columnName]),
		)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("comment organization column %s.%s: %w", statement.Schema.Table, columnName, err)
		}
	}
	return nil
}

func completeOrganizationColumnComments(
	modelColumns []string,
	specificComments map[string]string,
) (map[string]string, error) {
	comments := make(map[string]string, len(modelColumns))
	modelColumnSet := make(map[string]struct{}, len(modelColumns))
	for _, columnName := range modelColumns {
		modelColumnSet[columnName] = struct{}{}
		if comment, exists := platformBasicColumnComments[columnName]; exists {
			comments[columnName] = comment
		}
		if comment, exists := organizationSharedColumnComments[columnName]; exists {
			comments[columnName] = comment
		}
		if comment, exists := specificComments[columnName]; exists {
			comments[columnName] = comment
		}
		if strings.TrimSpace(comments[columnName]) == "" {
			return nil, fmt.Errorf("column %s has no database comment", columnName)
		}
	}
	for columnName := range specificComments {
		if _, exists := modelColumnSet[columnName]; !exists {
			return nil, fmt.Errorf("comment references unknown column %s", columnName)
		}
	}
	return comments, nil
}

func quotePostgresQualifiedIdentifier(value string) (string, error) {
	parts := strings.Split(value, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		identifier, err := quotePostgresIdentifier(part)
		if err != nil {
			return "", err
		}
		quoted = append(quoted, identifier)
	}
	return strings.Join(quoted, "."), nil
}

func quotePostgresIdentifier(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("empty PostgreSQL identifier")
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`, nil
}

func quotePostgresLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
