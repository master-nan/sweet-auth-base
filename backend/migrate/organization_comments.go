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

var organizationBaseColumnComments = map[string]string{
	"id":          "平台内部主键；业务引用使用对应领域ID，不使用外部源系统ID。",
	"gmt_create":  "镜像记录在平台数据库中的创建时间。",
	"create_user": "平台创建操作人账号ID，可空；不表示外部组织源中的人员。",
	"create_name": "平台创建操作人显示名称，可空。",
	"gmt_modify":  "镜像记录在平台数据库中的最后修改时间。",
	"modify_user": "平台最后修改操作人账号ID，可空；不表示外部组织源中的人员。",
	"modify_name": "平台最后修改操作人显示名称，可空。",
	"gmt_delete":  "平台软删除时间；外部组织源的停用或删除应由业务状态字段表达，不等同于本字段。",
	"delete_user": "平台软删除操作人账号ID，可空。",
	"delete_name": "平台软删除操作人显示名称，可空。",
	"state":       "平台记录启用状态；与外部源业务状态及同步状态相互独立。",
}

var organizationDatabaseCommentSpecs = []databaseCommentSpec{
	{
		model:        &model.OrgLegalEntity{},
		tableComment: "法人或核算主体组织主数据镜像，数据来自唯一外部权威组织源；平台仅用于查询、引用和本地扩展，不维护正式法人档案。",
		columnComments: map[string]string{
			"source_system_code":         "权威组织源系统稳定编码；即使当前只有单一权威源，也用于明确来源边界。",
			"source_id":                  "法人或核算主体在权威组织源中的稳定唯一标识，不作为平台业务外键。",
			"source_code":                "法人或核算主体在权威组织源中的业务编码，可空，不作为平台内部主键。",
			"code":                       "平台标准法人或核算主体编码，由权威组织源映射。",
			"name":                       "法人或核算主体标准名称，由权威组织源维护。",
			"short_name":                 "法人或核算主体简称，由权威组织源维护。",
			"entity_type":                "法人或核算主体类型，如法人公司、分公司或内部核算主体。",
			"parent_id":                  "上级法人或核算主体的平台内部ID，仅表达法人层级，不表示管理组织关系。",
			"unified_social_credit_code": "统一社会信用代码；非独立法人或未提供时可为空。",
			"accounting_code":            "财务或核算体系使用的主体编码，由权威组织源提供。",
			"status":                     "平台标准化业务状态，由权威组织源状态映射。",
			"valid_from":                 "法人或核算主体生效时间；为空表示权威源未提供起始时间。",
			"valid_to":                   "法人或核算主体失效时间；为空表示尚无已知结束时间。",
			"source_version":             "权威组织源记录版本，用于增量顺序和陈旧消息判断。",
			"source_updated_at":          "权威组织源记录最后更新时间。",
			"last_sync_at":               "本对象最近一次成功写入或确认镜像的时间。",
			"source_status":              "权威组织源返回的原始状态值，用于追溯状态映射。",
			"source_deleted":             "权威组织源是否已将对象标记删除；镜像记录仍保留以支持历史引用。",
			"sync_status":                "对象最近一次组织业务同步处理状态，不表示接口调用技术状态。",
			"last_error":                 "对象最近一次组织业务同步错误摘要，不等同于统一集成中心接口日志。",
			"local_note":                 "平台本地备注，属于平台扩展字段，组织同步不得覆盖。",
			"local_tags":                 "平台本地标签JSON，属于平台扩展字段，组织同步不得覆盖。",
			"display_order":              "平台本地展示顺序，可空，不改变权威组织源层级。",
			"local_handling_status":      "平台对同步异常或待办事项的本地处理状态，组织同步不得覆盖。",
		},
	},
	{
		model:        &model.OrgUnit{},
		tableComment: "管理组织单元主数据镜像，数据来自唯一外部权威组织源；用于部门、事业部、区域、中心等管理组织引用，平台不维护正式组织调整。",
		columnComments: map[string]string{
			"source_system_code":      "权威组织源系统稳定编码。",
			"source_id":               "管理组织单元在权威组织源中的稳定唯一标识，不作为平台业务外键。",
			"source_code":             "管理组织单元在权威组织源中的业务编码，可空。",
			"code":                    "平台标准管理组织编码，由权威组织源映射。",
			"name":                    "管理组织单元标准名称，由权威组织源维护。",
			"unit_type":               "管理组织单元类型，如事业部、区域、中心、部门、团队或项目组织。",
			"primary_legal_entity_id": "管理组织单元主要归属法人或核算主体的平台内部ID；不等同于管理组织层级。",
			"status":                  "平台标准化业务状态，由权威组织源状态映射。",
			"valid_from":              "管理组织单元生效时间；为空表示权威源未提供起始时间。",
			"valid_to":                "管理组织单元失效时间；为空表示尚无已知结束时间。",
			"source_version":          "权威组织源记录版本，用于增量顺序和陈旧消息判断。",
			"source_updated_at":       "权威组织源记录最后更新时间。",
			"last_sync_at":            "本对象最近一次成功写入或确认镜像的时间。",
			"source_status":           "权威组织源返回的原始状态值，用于追溯状态映射。",
			"source_deleted":          "权威组织源是否已将对象标记删除；镜像记录仍保留以支持历史引用。",
			"sync_status":             "对象最近一次组织业务同步处理状态，不表示接口调用技术状态。",
			"last_error":              "对象最近一次组织业务同步错误摘要，不等同于统一集成中心接口日志。",
			"local_note":              "平台本地备注，属于平台扩展字段，组织同步不得覆盖。",
			"local_tags":              "平台本地标签JSON，属于平台扩展字段，组织同步不得覆盖。",
			"display_order":           "平台本地展示顺序，可空，不改变权威组织源层级。",
			"local_handling_status":   "平台对同步异常或待办事项的本地处理状态，组织同步不得覆盖。",
		},
	},
	{
		model:        &model.OrgStructure{},
		tableComment: "管理组织架构定义镜像，用于区分行政、经营、财务或区域等组织视图；平台只读引用，不用于表达法人层级或本地编辑组织。",
		columnComments: map[string]string{
			"code":               "管理架构稳定编码，用于树查询和选择器的structure_code条件。",
			"name":               "管理架构名称。",
			"structure_type":     "管理架构类型，V1用于管理组织视图分类。",
			"source_system_code": "权威组织源系统稳定编码。",
			"source_id":          "管理架构在权威组织源中的稳定标识，可空。",
			"status":             "平台标准化业务状态，由权威组织源状态映射。",
			"is_default":         "是否为平台默认管理架构；不表示唯一架构。",
			"valid_from":         "管理架构生效时间；为空表示权威源未提供起始时间。",
			"valid_to":           "管理架构失效时间；为空表示尚无已知结束时间。",
			"source_version":     "权威组织源记录版本，用于增量顺序和陈旧消息判断。",
			"last_sync_at":       "本管理架构最近一次成功写入或确认镜像的时间。",
			"sync_status":        "管理架构最近一次组织业务同步处理状态，不表示接口调用技术状态。",
		},
	},
	{
		model:        &model.OrgStructureNode{},
		tableComment: "管理组织架构中的节点关系镜像；节点引用org_unit并表达特定架构内的父子关系，节点本身不是组织实体，业务数据不得持久化节点ID代替org_unit_id。",
		columnComments: map[string]string{
			"structure_id":       "所属管理架构的平台内部ID。",
			"org_unit_id":        "节点引用的管理组织单元平台内部ID；业务组织引用应保存此ID。",
			"parent_node_id":     "同一管理架构中父节点的平台内部ID，可空表示根节点。",
			"source_system_code": "权威组织源系统稳定编码。",
			"source_id":          "架构节点在权威组织源中的稳定唯一标识。",
			"source_parent_id":   "父节点在权威组织源中的标识，用于父节点尚未落库时的依赖定位。",
			"path":               "节点祖先路径缓存，用于高效树查询；不是业务组织标识。",
			"level":              "节点在当前管理架构中的层级缓存，根层级从1开始。",
			"sort":               "节点在同级中的源排序值。",
			"valid_from":         "架构节点关系生效时间；为空表示权威源未提供起始时间。",
			"valid_to":           "架构节点关系失效时间；为空表示尚无已知结束时间。",
			"status":             "架构节点关系的标准化业务状态。",
			"source_deleted":     "权威组织源是否已删除该节点关系；节点记录仍保留以支持历史查询。",
			"sync_status":        "节点关系最近一次组织业务同步处理状态，不表示接口调用技术状态。",
		},
	},
	{
		model:        &model.OrgPosition{},
		tableComment: "岗位主数据镜像，数据来自唯一外部权威组织源；用于平台岗位引用，不等同于系统角色，平台不维护正式岗位调整。",
		columnComments: map[string]string{
			"source_system_code":  "权威组织源系统稳定编码。",
			"source_id":           "岗位在权威组织源中的稳定唯一标识，不作为平台业务外键。",
			"source_code":         "岗位在权威组织源中的业务编码，可空。",
			"code":                "平台标准岗位编码，由权威组织源映射。",
			"name":                "岗位标准名称，由权威组织源维护。",
			"org_unit_id":         "岗位所属管理组织单元的平台内部ID。",
			"position_type":       "岗位类型，如管理岗或专业岗；不表示系统角色。",
			"job_level":           "岗位职级或层级，由权威组织源提供。",
			"is_manager_position": "是否为管理负责人岗位的源镜像标记，不自动产生系统权限。",
			"status":              "平台标准化业务状态，由权威组织源状态映射。",
			"valid_from":          "岗位生效时间；为空表示权威源未提供起始时间。",
			"valid_to":            "岗位失效时间；为空表示尚无已知结束时间。",
			"source_version":      "权威组织源记录版本，用于增量顺序和陈旧消息判断。",
			"last_sync_at":        "本岗位最近一次成功写入或确认镜像的时间。",
			"source_deleted":      "权威组织源是否已将岗位标记删除；镜像记录仍保留以支持历史引用。",
			"sync_status":         "岗位最近一次组织业务同步处理状态，不表示接口调用技术状态。",
			"local_note":          "平台本地备注，属于平台扩展字段，组织同步不得覆盖。",
		},
	},
	{
		model:        &model.OrgEmployee{},
		tableComment: "企业人员主数据镜像，数据来自唯一外部权威组织源；用于平台人员引用和当前应用账号绑定，不作为HR档案维护。",
		columnComments: map[string]string{
			"source_system_code":      "权威组织源系统稳定编码。",
			"source_id":               "企业人员在权威组织源中的稳定唯一标识，不作为平台业务外键。",
			"source_code":             "企业人员在权威组织源中的业务编码，可空。",
			"employee_no":             "企业人员编号，由权威组织源维护。",
			"name":                    "企业人员姓名，由权威组织源维护。",
			"mobile":                  "企业人员手机号，属于敏感联系方式，查询和展示必须遵守脱敏规则。",
			"email":                   "企业人员邮箱，属于敏感联系方式，查询和展示必须遵守脱敏规则。",
			"employment_status":       "人员在企业中的标准化任职状态，如在职或离职；不等同于平台账号状态。",
			"primary_legal_entity_id": "人员主归属法人或核算主体的平台内部ID，仅为源镜像摘要。",
			"valid_from":              "人员主数据生效时间；为空表示权威源未提供起始时间。",
			"valid_to":                "人员主数据失效时间；为空表示尚无已知结束时间。",
			"source_version":          "权威组织源记录版本，用于增量顺序和陈旧消息判断。",
			"source_updated_at":       "权威组织源记录最后更新时间。",
			"last_sync_at":            "本人员最近一次成功写入或确认镜像的时间。",
			"source_deleted":          "权威组织源是否已将人员标记删除；镜像记录仍保留以支持历史引用。",
			"sync_status":             "人员最近一次组织业务同步处理状态，不表示接口调用技术状态。",
			"user_id":                 "当前Sweet Platform应用实例登录账号绑定ID，可空；不是集团统一身份，也不是外部系统账号映射。",
			"local_note":              "平台本地备注，属于平台扩展字段，组织同步不得覆盖。",
			"local_tags":              "平台本地标签JSON，属于平台扩展字段，组织同步不得覆盖。",
		},
	},
	{
		model:        &model.OrgAssignment{},
		tableComment: "人员、法人、管理组织和岗位之间的任职关系镜像，数据来自唯一外部权威组织源；支持多任职及有效期查询，不承担调岗、离职或人事流程。",
		columnComments: map[string]string{
			"source_system_code": "权威组织源系统稳定编码。",
			"source_id":          "任职关系在权威组织源中的稳定唯一标识。",
			"employee_id":        "任职人员的平台内部employee_id，不是登录账号user_id。",
			"legal_entity_id":    "任职所属法人或核算主体的平台内部legal_entity_id。",
			"org_unit_id":        "任职所属管理组织单元的平台内部org_unit_id。",
			"position_id":        "任职岗位的平台内部position_id，可空表示源数据未提供岗位。",
			"assignment_type":    "任职类型，如主职、兼职、临时或项目任职，由权威组织源映射。",
			"is_primary":         "权威组织源提供的主任职标记；平台不得自动选择第一条任职推断主任职。",
			"is_manager":         "是否为该任职组织的管理负责人标记，不自动产生系统角色或权限。",
			"valid_from":         "任职关系生效时间；用于当前、历史和未来任职查询。",
			"valid_to":           "任职关系失效时间；为空表示尚无已知结束时间。",
			"status":             "任职关系的标准化业务状态。",
			"source_version":     "权威组织源记录版本，用于增量顺序和陈旧消息判断。",
			"source_deleted":     "权威组织源是否已删除该任职关系；镜像记录仍保留以支持历史查询。",
			"sync_status":        "任职关系最近一次组织业务同步处理状态，不表示接口调用技术状态。",
		},
	},
	{
		model:        &model.OrgSyncBatch{},
		tableComment: "组织主数据同步批次的业务处理结果记录，用于全量或增量同步监控与对账；不保存通用HTTP请求响应，不等同于统一集成中心技术执行日志。",
		columnComments: map[string]string{
			"batch_no":      "组织业务同步批次号，作为批次稳定业务标识。",
			"execution_id":  "关联统一集成中心技术执行实例ID，可空；本表不复制技术执行日志。",
			"sync_type":     "组织业务同步类型，如全量或增量。",
			"object_scope":  "本批次处理的组织对象范围，如全部或特定对象类型。",
			"started_at":    "组织业务处理开始时间。",
			"completed_at":  "组织业务处理完成时间。",
			"total_count":   "批次计划或实际处理对象总数。",
			"success_count": "批次处理成功对象数。",
			"failed_count":  "批次处理失败对象数。",
			"skipped_count": "批次跳过或无变化对象数。",
			"status":        "组织业务同步批次状态，不表示HTTP调用技术状态。",
			"error_summary": "批次级组织业务错误摘要，不保存完整请求响应载荷。",
		},
	},
	{
		model:        &model.OrgSyncRecord{},
		tableComment: "单个组织对象的同步业务处理结果记录，用于异常定位、业务重试和对账；不等同于HTTP接口日志，也不替代统一集成中心重试记录。",
		columnComments: map[string]string{
			"batch_id":              "所属组织业务同步批次的平台内部ID。",
			"execution_id":          "关联统一集成中心技术执行实例ID，可空。",
			"object_type":           "本记录处理的组织对象类型，如法人、管理组织、人员、岗位或任职。",
			"source_id":             "被处理对象在权威组织源中的稳定标识。",
			"source_code":           "被处理对象在权威组织源中的业务编码，可空。",
			"local_id":              "处理成功后对应组织镜像对象的平台内部ID，可空。",
			"action":                "本次业务处理动作，如新增、更新、停用、跳过或无变化。",
			"status":                "单对象组织业务处理状态，不表示HTTP调用技术状态。",
			"error_code":            "组织同步业务错误码，用于稳定分类和重试判断。",
			"error_message":         "组织同步业务错误摘要；不得写入未脱敏请求、响应或敏感凭证。",
			"dependency_type":       "未满足依赖的组织对象类型，如父组织、人员、岗位或法人。",
			"dependency_key":        "未满足依赖的稳定定位键，通常为源对象标识或业务编码。",
			"retry_count":           "该同步业务记录已发起的业务重试次数；不替代统一集成中心技术重试计数。",
			"last_retry_at":         "最近一次组织业务重试时间。",
			"local_handling_status": "平台对该同步异常的本地处理状态，如待处理、已忽略或已关闭。",
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
		if comment, exists := organizationBaseColumnComments[columnName]; exists {
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
