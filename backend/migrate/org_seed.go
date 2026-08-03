package main

import (
	"backend/enum"
	"backend/internal/utils"
	"backend/model"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	organizationRootMenuName = "organization"
)

// seedOrganizationFoundation 为现有平台 Seed 流程补充只读组织镜像元数据和功能权限。
func seedOrganizationFoundation(db *gorm.DB, sf *utils.Snowflake) error {
	if err := seedOrganizationDictionaries(db, sf); err != nil {
		return fmt.Errorf("seed organization dictionaries: %w", err)
	}
	if err := seedOrganizationTableMetadata(db, sf); err != nil {
		return fmt.Errorf("seed organization table metadata: %w", err)
	}
	if err := seedOrganizationMenusAndPermissions(db, sf); err != nil {
		return fmt.Errorf("seed organization menus and permissions: %w", err)
	}
	return nil
}

func seedOrganizationDictionaries(db *gorm.DB, sf *utils.Snowflake) error {
	for _, seed := range organizationDictionarySeeds() {
		if err := seedSystemDict(db, sf, seed); err != nil {
			return err
		}
	}
	return seedSystemDict(db, sf, systemDictSeed{
		name: "菜单按钮事件",
		code: "sys_menu_button_event_action",
		items: []systemDictItemSeed{
			{name: "查询", code: "sys_menu_button_event_action_query", value: "query"},
			{name: "详情", code: "sys_menu_button_event_action_detail", value: "detail"},
			{name: "刷新", code: "sys_menu_button_event_action_refresh", value: "refresh"},
			{name: "绑定账号", code: "sys_menu_button_event_action_bind_user", value: "bind_user"},
			{name: "解绑账号", code: "sys_menu_button_event_action_unbind_user", value: "unbind_user"},
			{name: "查看同步", code: "sys_menu_button_event_action_view_sync", value: "view_sync"},
			{name: "重试", code: "sys_menu_button_event_action_retry", value: "retry"},
			{name: "查看错误", code: "sys_menu_button_event_action_view_error", value: "view_error"},
		},
	})
}

func organizationDictionarySeeds() []systemDictSeed {
	return []systemDictSeed{
		{
			name: "法人主体类型",
			code: "org_legal_entity_type",
			items: []systemDictItemSeed{
				{name: "集团", code: "org_legal_entity_type_group", value: "group"},
				{name: "法人公司", code: "org_legal_entity_type_legal_company", value: "legal_company"},
				{name: "分公司", code: "org_legal_entity_type_branch", value: "branch"},
				{name: "内部核算主体", code: "org_legal_entity_type_accounting_unit", value: "accounting_unit"},
			},
		},
		{
			name: "管理组织类型",
			code: "org_unit_type",
			items: []systemDictItemSeed{
				{name: "事业部", code: "org_unit_type_business_unit", value: "business_unit"},
				{name: "区域", code: "org_unit_type_region", value: "region"},
				{name: "中心", code: "org_unit_type_center", value: "center"},
				{name: "部门", code: "org_unit_type_department", value: "department"},
				{name: "团队", code: "org_unit_type_team", value: "team"},
				{name: "项目组", code: "org_unit_type_project_group", value: "project_group"},
			},
		},
		{
			name: "组织架构类型",
			code: "org_structure_type",
			items: []systemDictItemSeed{
				{name: "管理架构", code: "org_structure_type_management", value: "management"},
			},
		},
		{
			name: "岗位类型",
			code: "org_position_type",
			items: []systemDictItemSeed{
				{name: "管理", code: "org_position_type_management", value: "management"},
				{name: "专业", code: "org_position_type_professional", value: "professional"},
				{name: "技术", code: "org_position_type_technical", value: "technical"},
				{name: "运营", code: "org_position_type_operation", value: "operation"},
				{name: "服务", code: "org_position_type_service", value: "service"},
			},
		},
		{
			name: "人员状态",
			code: "org_employment_status",
			items: []systemDictItemSeed{
				{name: "在职", code: "org_employment_status_active", value: "active"},
				{name: "试用", code: "org_employment_status_probation", value: "probation"},
				{name: "暂停", code: "org_employment_status_suspended", value: "suspended"},
				{name: "离职", code: "org_employment_status_resigned", value: "resigned"},
				{name: "退休", code: "org_employment_status_retired", value: "retired"},
			},
		},
		{
			name: "任职类型",
			code: "org_assignment_type",
			items: []systemDictItemSeed{
				{name: "主任职", code: "org_assignment_type_primary", value: "primary"},
				{name: "兼职", code: "org_assignment_type_part_time", value: "part_time"},
				{name: "临时", code: "org_assignment_type_temporary", value: "temporary"},
				{name: "项目", code: "org_assignment_type_project", value: "project"},
			},
		},
		{
			name: "组织对象状态",
			code: "org_object_status",
			items: []systemDictItemSeed{
				{name: "启用", code: "org_object_status_enabled", value: "enabled"},
				{name: "停用", code: "org_object_status_disabled", value: "disabled"},
			},
		},
		{
			name: "组织同步状态",
			code: "org_sync_status",
			items: []systemDictItemSeed{
				{name: "待同步", code: "org_sync_status_pending", value: "pending"},
				{name: "已同步", code: "org_sync_status_synced", value: "synced"},
				{name: "失败", code: "org_sync_status_failed", value: "failed"},
				{name: "等待依赖", code: "org_sync_status_dependency_waiting", value: "dependency_waiting"},
			},
		},
		{
			name: "组织同步类型",
			code: "org_sync_type",
			items: []systemDictItemSeed{
				{name: "全量", code: "org_sync_type_full", value: "full"},
				{name: "增量", code: "org_sync_type_incremental", value: "incremental"},
				{name: "手工重试", code: "org_sync_type_manual_retry", value: "manual_retry"},
			},
		},
		{
			name: "组织同步动作",
			code: "org_sync_action",
			items: []systemDictItemSeed{
				{name: "新增", code: "org_sync_action_insert", value: "insert"},
				{name: "更新", code: "org_sync_action_update", value: "update"},
				{name: "停用", code: "org_sync_action_disable", value: "disable"},
				{name: "删除转停用", code: "org_sync_action_delete_to_disable", value: "delete_to_disable"},
				{name: "跳过", code: "org_sync_action_skip", value: "skip"},
				{name: "无变化", code: "org_sync_action_no_change", value: "no_change"},
			},
		},
		{
			name: "同步记录状态",
			code: "org_sync_record_status",
			items: []systemDictItemSeed{
				{name: "待处理", code: "org_sync_record_status_pending", value: "pending"},
				{name: "处理中", code: "org_sync_record_status_processing", value: "processing"},
				{name: "成功", code: "org_sync_record_status_success", value: "success"},
				{name: "失败", code: "org_sync_record_status_failed", value: "failed"},
				{name: "等待依赖", code: "org_sync_record_status_dependency_waiting", value: "dependency_waiting"},
				{name: "已忽略", code: "org_sync_record_status_ignored", value: "ignored"},
			},
		},
		{
			name: "组织依赖类型",
			code: "org_dependency_type",
			items: []systemDictItemSeed{
				{name: "法人主体", code: "org_dependency_type_legal_entity", value: "legal_entity"},
				{name: "管理组织", code: "org_dependency_type_org_unit", value: "org_unit"},
				{name: "架构节点", code: "org_dependency_type_structure_node", value: "structure_node"},
				{name: "企业人员", code: "org_dependency_type_employee", value: "employee"},
				{name: "岗位", code: "org_dependency_type_position", value: "position"},
				{name: "任职", code: "org_dependency_type_assignment", value: "assignment"},
			},
		},
		{
			name: "账号绑定状态",
			code: "org_user_binding_status",
			items: []systemDictItemSeed{
				{name: "未绑定", code: "org_user_binding_status_unbound", value: "unbound"},
				{name: "已绑定", code: "org_user_binding_status_bound", value: "bound"},
				{name: "冲突", code: "org_user_binding_status_conflict", value: "conflict"},
			},
		},
	}
}

type organizationFieldRule struct {
	name      string
	list      bool
	quick     bool
	advanced  bool
	dictCode  string
	sort      bool
	inputType enum.SysTableFieldInputType
}

func organizationField(name string, list, quick, advanced bool) organizationFieldRule {
	return organizationFieldRule{
		name:     name,
		list:     list,
		quick:    quick,
		advanced: advanced,
		sort:     list || advanced,
	}
}

func organizationDictField(name, dictCode string, list, advanced bool) organizationFieldRule {
	rule := organizationField(name, list, false, advanced)
	rule.dictCode = dictCode
	rule.inputType = enum.SelectInputType
	return rule
}

func seedOrganizationTableMetadata(db *gorm.DB, sf *utils.Snowflake) error {
	for _, seed := range organizationTableSeeds() {
		if !db.Migrator().HasTable(seed.code) {
			continue
		}
		table, err := seedSystemTable(db, sf, seed)
		if err != nil {
			return err
		}
		columns, err := db.Migrator().ColumnTypes(seed.code)
		if err != nil {
			return err
		}
		for index, column := range columns {
			field := systemColumnToTableField(seed.code, column, index+1)
			applyOrganizationFieldRule(seed.code, &field)
			if err := seedSystemTableField(db, sf, table, field); err != nil {
				return err
			}
		}
	}
	return nil
}

func organizationTableSeeds() []systemTableMetadataSeed {
	return []systemTableMetadataSeed{
		{code: "org_legal_entity", name: "法人主体"},
		{code: "org_unit", name: "管理组织"},
		{code: "org_structure", name: "管理架构"},
		{code: "org_structure_node", name: "管理架构节点"},
		{code: "org_position", name: "岗位"},
		{code: "org_employee", name: "企业人员"},
		{code: "org_assignment", name: "任职"},
		{code: "org_sync_batch", name: "组织同步批次"},
		{code: "org_sync_record", name: "组织同步记录"},
	}
}

func applyOrganizationFieldRule(tableCode string, field *model.SysTableField) {
	field.IsListShow = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsSort = false

	rule, ok := organizationMetadataFieldRules[tableCode][field.FieldCode]
	if !ok {
		if name, exists := organizationCommonFieldNames[field.FieldCode]; exists {
			field.FieldName = name
		}
		return
	}
	field.FieldName = rule.name
	field.IsListShow = rule.list
	field.IsQuickSearch = rule.quick
	field.IsAdvancedSearch = rule.advanced
	field.IsSort = rule.sort
	if rule.dictCode != "" {
		field.DictCode = utils.StringPtr(rule.dictCode)
	}
	if rule.inputType != 0 {
		field.InputType = rule.inputType
	}
}

var organizationCommonFieldNames = map[string]string{
	"id":                    "ID",
	"source_system_code":    "源系统编码",
	"source_id":             "源对象ID",
	"source_code":           "源对象编码",
	"code":                  "编码",
	"name":                  "名称",
	"status":                "状态",
	"valid_from":            "生效时间",
	"valid_to":              "失效时间",
	"source_version":        "源版本",
	"source_updated_at":     "源更新时间",
	"last_sync_at":          "最近同步时间",
	"source_status":         "源状态",
	"source_deleted":        "源删除标记",
	"sync_status":           "同步状态",
	"last_error":            "最近错误",
	"local_note":            "平台备注",
	"local_tags":            "平台标签",
	"display_order":         "平台显示顺序",
	"local_handling_status": "本地处理状态",
	"gmt_create":            "创建时间",
	"create_user":           "创建人ID",
	"create_name":           "创建人",
	"gmt_modify":            "更新时间",
	"modify_user":           "修改人ID",
	"modify_name":           "修改人",
	"gmt_delete":            "删除时间",
	"delete_user":           "删除人ID",
	"delete_name":           "删除人",
	"state":                 "平台记录状态",
}

var organizationMetadataFieldRules = map[string]map[string]organizationFieldRule{
	"org_legal_entity": {
		"source_system_code":         organizationField("源系统编码", false, false, true),
		"source_code":                organizationField("源法人编码", false, true, true),
		"code":                       organizationField("法人编码", true, true, true),
		"name":                       organizationField("法人名称", true, true, true),
		"short_name":                 organizationField("法人简称", true, true, true),
		"entity_type":                organizationDictField("法人类型", "org_legal_entity_type", true, true),
		"parent_id":                  organizationField("上级法人", true, false, true),
		"unified_social_credit_code": organizationField("统一社会信用代码", true, true, true),
		"accounting_code":            organizationField("核算编码", true, true, true),
		"status":                     organizationDictField("状态", "org_object_status", true, true),
		"valid_from":                 organizationField("生效时间", false, false, true),
		"valid_to":                   organizationField("失效时间", false, false, true),
		"local_handling_status":      organizationField("本地处理状态", false, false, true),
	},
	"org_unit": {
		"source_system_code":      organizationField("源系统编码", false, false, true),
		"source_code":             organizationField("源组织编码", false, true, true),
		"code":                    organizationField("组织编码", true, true, true),
		"name":                    organizationField("组织名称", true, true, true),
		"unit_type":               organizationDictField("组织类型", "org_unit_type", true, true),
		"primary_legal_entity_id": organizationField("主法人", true, false, true),
		"status":                  organizationDictField("状态", "org_object_status", true, true),
		"valid_from":              organizationField("生效时间", false, false, true),
		"valid_to":                organizationField("失效时间", false, false, true),
		"display_order":           organizationField("平台显示顺序", false, false, true),
		"local_handling_status":   organizationField("本地处理状态", false, false, true),
	},
	"org_structure": {
		"code":               organizationField("架构编码", true, true, true),
		"name":               organizationField("架构名称", true, true, true),
		"structure_type":     organizationDictField("架构类型", "org_structure_type", true, true),
		"source_system_code": organizationField("源系统编码", false, false, true),
		"status":             organizationDictField("状态", "org_object_status", true, true),
		"is_default":         organizationDictField("默认架构", "whether", true, true),
		"valid_from":         organizationField("生效时间", false, false, true),
		"valid_to":           organizationField("失效时间", false, false, true),
	},
	"org_structure_node": {
		"structure_id":   organizationField("管理架构", true, false, true),
		"org_unit_id":    organizationField("管理组织", true, false, true),
		"parent_node_id": organizationField("上级节点", true, false, true),
		"sort":           organizationField("排序", true, false, true),
		"valid_from":     organizationField("生效时间", false, false, true),
		"valid_to":       organizationField("失效时间", false, false, true),
		"status":         organizationDictField("状态", "org_object_status", true, true),
	},
	"org_position": {
		"source_system_code":  organizationField("源系统编码", false, false, true),
		"source_code":         organizationField("源岗位编码", false, true, true),
		"code":                organizationField("岗位编码", true, true, true),
		"name":                organizationField("岗位名称", true, true, true),
		"org_unit_id":         organizationField("所属管理组织", true, false, true),
		"position_type":       organizationDictField("岗位类型", "org_position_type", true, true),
		"job_level":           organizationField("职级", true, true, true),
		"is_manager_position": organizationDictField("管理岗位", "whether", true, true),
		"status":              organizationDictField("状态", "org_object_status", true, true),
		"valid_from":          organizationField("生效时间", false, false, true),
		"valid_to":            organizationField("失效时间", false, false, true),
	},
	"org_employee": {
		"source_system_code":      organizationField("源系统编码", false, false, true),
		"source_code":             organizationField("源人员编码", false, true, true),
		"employee_no":             organizationField("工号", true, true, true),
		"name":                    organizationField("姓名", true, true, true),
		"employment_status":       organizationDictField("人员状态", "org_employment_status", true, true),
		"primary_legal_entity_id": organizationField("主法人", true, false, true),
		"user_id":                 organizationField("绑定账号", true, false, true),
		"valid_from":              organizationField("生效时间", false, false, true),
		"valid_to":                organizationField("失效时间", false, false, true),
	},
	"org_assignment": {
		"source_system_code": organizationField("源系统编码", false, false, true),
		"employee_id":        organizationField("企业人员", true, false, true),
		"legal_entity_id":    organizationField("任职法人", true, false, true),
		"org_unit_id":        organizationField("任职管理组织", true, false, true),
		"position_id":        organizationField("任职岗位", true, false, true),
		"assignment_type":    organizationDictField("任职类型", "org_assignment_type", true, true),
		"is_primary":         organizationDictField("主任职", "whether", true, true),
		"is_manager":         organizationDictField("负责人任职", "whether", true, true),
		"valid_from":         organizationField("生效时间", true, false, true),
		"valid_to":           organizationField("失效时间", true, false, true),
		"status":             organizationDictField("状态", "org_object_status", true, true),
	},
	"org_sync_batch": {
		"batch_no":      organizationField("批次号", true, true, true),
		"execution_id":  organizationField("集成执行ID", true, false, true),
		"sync_type":     organizationDictField("同步类型", "org_sync_type", true, true),
		"object_scope":  organizationField("对象范围", true, true, true),
		"started_at":    organizationField("开始时间", true, false, true),
		"completed_at":  organizationField("完成时间", true, false, true),
		"total_count":   organizationField("总数", true, false, false),
		"success_count": organizationField("成功数", true, false, false),
		"failed_count":  organizationField("失败数", true, false, false),
		"skipped_count": organizationField("跳过数", true, false, false),
		"status":        organizationDictField("批次状态", "org_sync_record_status", true, true),
		"error_summary": organizationField("错误摘要", false, true, true),
	},
	"org_sync_record": {
		"batch_id":              organizationField("同步批次", true, false, true),
		"execution_id":          organizationField("集成执行ID", true, false, true),
		"object_type":           organizationField("对象类型", true, false, true),
		"source_code":           organizationField("源对象编码", true, true, true),
		"local_id":              organizationField("本地对象ID", true, false, true),
		"action":                organizationDictField("同步动作", "org_sync_action", true, true),
		"status":                organizationDictField("处理状态", "org_sync_record_status", true, true),
		"error_code":            organizationField("错误码", true, true, true),
		"dependency_type":       organizationDictField("依赖类型", "org_dependency_type", true, true),
		"retry_count":           organizationField("重试次数", true, false, true),
		"last_retry_at":         organizationField("最近重试时间", true, false, true),
		"local_handling_status": organizationDictField("本地处理状态", "org_sync_record_status", true, true),
	},
}

func seedOrganizationMenusAndPermissions(db *gorm.DB, sf *utils.Snowflake) error {
	root, err := seedMenu(
		db,
		sf,
		directoryMenu(menu(1000, 0, organizationRootMenuName, "organization", "src/components/Layout/Layout.vue", "组织管理", "account_tree", 5)),
	)
	if err != nil {
		return err
	}

	menuSeeds := []model.SysMenu{
		menuWithOption(menuWithTable(menu(1002, root.Id, "organization_structure", "structure", "pages/organization/structure/Index.vue", "组织架构", "lan", 1), "org_unit"), "org_structure,org_structure_node,org_unit,org_legal_entity"),
		menuWithOption(menuWithTable(menu(1003, root.Id, "organization_employee", "employee", "pages/organization/employee/Index.vue", "人员与任职", "badge", 3), "org_employee"), "org_employee,org_assignment"),
		menuWithTable(menu(1004, root.Id, "organization_position", "position", "pages/organization/position/Index.vue", "岗位", "work", 4), "org_position"),
		menuWithOption(menuWithTable(menu(1005, root.Id, "organization_sync_batch", "sync-batch", "pages/organization/sync-batch/Index.vue", "同步批次", "sync", 5), "org_sync_batch"), "org_sync_batch,org_sync_record"),
		menuWithTable(menu(1006, root.Id, "organization_sync_error", "sync-error", "pages/organization/sync-error/Index.vue", "同步异常", "error_outline", 6), "org_sync_record"),
	}

	menuByName := map[string]model.SysMenu{root.Name: root}
	for _, item := range menuSeeds {
		seeded, err := seedMenu(db, sf, item)
		if err != nil {
			return err
		}
		menuByName[seeded.Name] = seeded
	}

	role, err := seedRole(db, sf)
	if err != nil {
		return err
	}
	for _, item := range menuByName {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysRoleMenu{
			RoleId: role.Id,
			MenuId: item.Id,
		}).Error; err != nil {
			return err
		}
	}

	for _, group := range organizationMenuButtons(menuByName) {
		if _, ok := menuByName[group.menuName]; !ok {
			return fmt.Errorf("organization menu %s missing after seed", group.menuName)
		}
		if err := seedMenuButtons(db, sf, role.Id, role.Name, group.buttons); err != nil {
			return err
		}
	}
	return retireLegacyOrganizationLegalEntityMenu(db, menuByName["organization_structure"].Id)
}

type organizationMenuButtonSeed struct {
	menuName string
	buttons  []model.SysMenuButton
}

func organizationMenuButtons(menuByName map[string]model.SysMenu) []organizationMenuButtonSeed {
	structureMenu := menuByName["organization_structure"].Id
	employeeMenu := menuByName["organization_employee"].Id
	positionMenu := menuByName["organization_position"].Id
	syncBatchMenu := menuByName["organization_sync_batch"].Id
	syncErrorMenu := menuByName["organization_sync_error"].Id

	return []organizationMenuButtonSeed{
		{menuName: "organization_structure", buttons: []model.SysMenuButton{
			apiPermissionWithAPI(800, structureMenu, "法人查询", "organization_legal_entity_query", enum.Top, "query", "search", "primary", 0, "/admin/org/legal-entity/query", "POST"),
			apiPermissionWithAPI(801, structureMenu, "法人树查询", "organization_legal_entity_tree", enum.Top, "query", "account_tree", "primary", 1, "/admin/org/legal-entity/tree", "POST"),
			apiPermissionWithAPI(802, structureMenu, "法人选项查询", "organization_legal_entity_options", enum.Top, "query", "list", "primary", 2, "/admin/org/legal-entity/options", "POST"),
			apiPermissionWithAPI(803, structureMenu, "法人详情", "organization_legal_entity_detail", enum.Line, "detail", "visibility", "primary", 3, "/admin/org/legal-entity/:id", "GET"),
			apiPermissionWithAPI(810, structureMenu, "管理组织查询", "organization_unit_query", enum.Top, "query", "search", "primary", 0, "/admin/org/unit/query", "POST"),
			apiPermissionWithAPI(811, structureMenu, "管理组织树查询", "organization_unit_tree", enum.Top, "query", "account_tree", "primary", 1, "/admin/org/unit/tree", "POST"),
			apiPermissionWithAPI(812, structureMenu, "管理组织选项查询", "organization_unit_options", enum.Top, "query", "list", "primary", 2, "/admin/org/unit/options", "POST"),
			apiPermissionWithAPI(813, structureMenu, "管理组织详情", "organization_unit_detail", enum.Line, "detail", "visibility", "primary", 4, "/admin/org/unit/:id", "GET"),
			menuButton(816, structureMenu, "刷新", "organization_structure_refresh", enum.Top, "refresh", "refresh", "primary", 3),
			menuButton(817, structureMenu, "查看同步", "organization_structure_view_sync", enum.Line, "view_sync", "sync", "primary", 2),
			apiPermissionWithAPI(860, structureMenu, "管理架构查询", "organization_structure_query", enum.Top, "query", "search", "primary", 5, "/admin/org/structure/query", "POST"),
			apiPermissionWithAPI(861, structureMenu, "管理架构选项查询", "organization_structure_options", enum.Top, "query", "list", "primary", 6, "/admin/org/structure/options", "POST"),
			apiPermissionWithAPI(862, structureMenu, "管理架构详情", "organization_structure_detail", enum.Line, "detail", "visibility", "primary", 7, "/admin/org/structure/:id", "GET"),
		}},
		{menuName: "organization_employee", buttons: []model.SysMenuButton{
			apiPermissionWithAPI(820, employeeMenu, "人员查询", "organization_employee_query", enum.Top, "query", "search", "primary", 0, "/admin/org/employee/query", "POST"),
			apiPermissionWithAPI(821, employeeMenu, "人员选项查询", "organization_employee_options", enum.Top, "query", "list", "primary", 1, "/admin/org/employee/options", "POST"),
			apiPermissionWithAPI(865, employeeMenu, "可绑定账号选项", "organization_employee_user_options", enum.Line, "bind_user", "person_search", "primary", 8, "/admin/org/employee/user-options", "POST"),
			menuButtonWithAPI(822, employeeMenu, "详情", "organization_employee_detail", enum.Line, "detail", "visibility", "primary", 1, "/admin/org/employee/:id", "GET"),
			organizationDetailButton(menuButtonWithAPI(823, employeeMenu, "绑定账号", "organization_employee_bind_user", enum.DetailTop, "bind_user", "link", "primary", 2, "/admin/org/employee/:id/bind-user", "POST"), `{"field":"row.user_id","op":"not_empty"}`),
			organizationDetailButton(menuButtonWithAPI(824, employeeMenu, "解绑账号", "organization_employee_unbind_user", enum.DetailTop, "unbind_user", "link_off", "warning", 3, "/admin/org/employee/:id/unbind-user", "POST"), `{"field":"row.user_id","op":"empty"}`),
			menuButton(825, employeeMenu, "刷新", "organization_employee_refresh", enum.Top, "refresh", "refresh", "primary", 2),
			menuButton(826, employeeMenu, "查看同步", "organization_employee_view_sync", enum.DetailBottom, "view_sync", "sync", "primary", 4),
			apiPermissionWithAPI(827, employeeMenu, "任职查询", "organization_assignment_query", enum.Line, "query", "work_history", "primary", 5, "/admin/org/assignment/query", "POST"),
			apiPermissionWithAPI(828, employeeMenu, "任职详情", "organization_assignment_detail", enum.Line, "detail", "visibility", "primary", 6, "/admin/org/assignment/:id", "GET"),
			apiPermissionWithAPI(829, employeeMenu, "当前任职摘要", "organization_assignment_summary", enum.Line, "query", "summarize", "primary", 7, "/admin/org/employee/:id/assignments/summary", "GET"),
		}},
		{menuName: "organization_position", buttons: []model.SysMenuButton{
			apiPermissionWithAPI(830, positionMenu, "岗位查询", "organization_position_query", enum.Top, "query", "search", "primary", 0, "/admin/org/position/query", "POST"),
			apiPermissionWithAPI(831, positionMenu, "岗位选项查询", "organization_position_options", enum.Top, "query", "list", "primary", 1, "/admin/org/position/options", "POST"),
			menuButtonWithAPI(832, positionMenu, "详情", "organization_position_detail", enum.Line, "detail", "visibility", "primary", 1, "/admin/org/position/:id", "GET"),
			menuButton(833, positionMenu, "刷新", "organization_position_refresh", enum.Top, "refresh", "refresh", "primary", 2),
			menuButton(834, positionMenu, "查看同步", "organization_position_view_sync", enum.DetailBottom, "view_sync", "sync", "primary", 2),
		}},
		{menuName: "organization_sync_batch", buttons: []model.SysMenuButton{
			apiPermissionWithAPI(840, syncBatchMenu, "同步批次查询", "organization_sync_batch_query", enum.Top, "query", "search", "primary", 0, "/admin/org/sync/batch/query", "POST"),
			menuButtonWithAPI(841, syncBatchMenu, "详情", "organization_sync_batch_detail", enum.Line, "detail", "visibility", "primary", 1, "/admin/org/sync/batch/:id", "GET"),
			menuButton(842, syncBatchMenu, "刷新", "organization_sync_batch_refresh", enum.Top, "refresh", "refresh", "primary", 1),
			organizationDetailButton(menuButtonWithAPI(843, syncBatchMenu, "查看错误", "organization_sync_batch_view_error", enum.DetailTop, "view_error", "error_outline", "negative", 2, "/admin/org/sync/batch/:id/error", "GET"), `{"field":"row.has_error","op":"falsy"}`),
		}},
		{menuName: "organization_sync_error", buttons: []model.SysMenuButton{
			apiPermissionWithAPI(850, syncErrorMenu, "同步异常查询", "organization_sync_error_query", enum.Top, "query", "search", "primary", 0, "/admin/org/sync/record/query", "POST"),
			menuButtonWithAPI(851, syncErrorMenu, "详情", "organization_sync_error_detail", enum.Line, "detail", "visibility", "primary", 1, "/admin/org/sync/record/:id", "GET"),
			menuButton(854, syncErrorMenu, "刷新", "organization_sync_error_refresh", enum.Top, "refresh", "refresh", "primary", 1),
			organizationDetailButton(menuButtonWithAPI(855, syncErrorMenu, "查看错误", "organization_sync_error_view_error", enum.DetailTop, "view_error", "error_outline", "negative", 2, "/admin/org/sync/record/:id/error", "GET"), `{"field":"row.has_error","op":"falsy"}`),
		}},
	}
}

func organizationDetailButton(button model.SysMenuButton, disableWhen string) model.SysMenuButton {
	button.DisableWhen = disableWhen
	return button
}

func retireLegacyOrganizationLegalEntityMenu(db *gorm.DB, structureMenuID int) error {
	var legacyMenu model.SysMenu
	err := db.Where("name = ?", "organization_legal_entity").First(&legacyMenu).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var structureMenu model.SysMenu
		if err := tx.Where("id = ?", structureMenuID).First(&structureMenu).Error; err != nil {
			return err
		}

		var roleMenus []model.SysRoleMenu
		if err := tx.Where("menu_id = ?", legacyMenu.Id).Find(&roleMenus).Error; err != nil {
			return err
		}
		for _, roleMenu := range roleMenus {
			for _, menuID := range []int{structureMenu.Pid, structureMenuID} {
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(
					&model.SysRoleMenu{RoleId: roleMenu.RoleId, MenuId: menuID},
				).Error; err != nil {
					return err
				}
			}
		}

		type legacyButtonGrant struct {
			RoleID int
			Code   string
		}
		var legacyGrants []legacyButtonGrant
		if err := tx.Table("sys_role_menu_button").
			Select("sys_role_menu_button.role_id, sys_menu_button.code").
			Joins("JOIN sys_menu_button ON sys_menu_button.id = sys_role_menu_button.button_id").
			Where("sys_role_menu_button.menu_id = ?", legacyMenu.Id).
			Scan(&legacyGrants).Error; err != nil {
			return err
		}

		replacementCodeByLegacyCode := map[string]string{
			"organization_legal_entity_query":     "organization_legal_entity_query",
			"organization_legal_entity_tree":      "organization_legal_entity_tree",
			"organization_legal_entity_options":   "organization_legal_entity_options",
			"organization_legal_entity_detail":    "organization_legal_entity_detail",
			"organization_legal_entity_refresh":   "organization_structure_refresh",
			"organization_legal_entity_view_sync": "organization_structure_view_sync",
		}
		replacementCodes := make([]string, 0, len(replacementCodeByLegacyCode))
		for _, code := range replacementCodeByLegacyCode {
			replacementCodes = append(replacementCodes, code)
		}

		var replacementButtons []model.SysMenuButton
		if err := tx.Where(
			"menu_id = ? AND code IN ?",
			structureMenuID,
			replacementCodes,
		).Find(&replacementButtons).Error; err != nil {
			return err
		}
		replacementByCode := make(map[string]model.SysMenuButton, len(replacementButtons))
		for _, button := range replacementButtons {
			replacementByCode[button.Code] = button
		}
		for _, grant := range legacyGrants {
			replacementCode, ok := replacementCodeByLegacyCode[grant.Code]
			if !ok {
				continue
			}
			button, ok := replacementByCode[replacementCode]
			if !ok {
				continue
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(
				&model.SysRoleMenuButton{
					RoleId:   grant.RoleID,
					MenuId:   structureMenuID,
					ButtonId: button.Id,
				},
			).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("menu_id = ?", legacyMenu.Id).
			Delete(&model.SysRoleMenuButton{}).Error; err != nil {
			return err
		}
		if err := tx.Where("menu_id = ?", legacyMenu.Id).
			Delete(&model.SysRoleMenu{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.SysMenuButton{}).
			Where("menu_id = ?", legacyMenu.Id).
			Updates(map[string]interface{}{
				"is_button":   false,
				"is_hidden":   true,
				"is_disabled": true,
				"state":       false,
				"gmt_modify":  model.Now(),
			}).Error; err != nil {
			return err
		}
		return tx.Model(&model.SysMenu{}).
			Where("id = ?", legacyMenu.Id).
			Updates(map[string]interface{}{
				"is_hidden":  true,
				"state":      false,
				"gmt_modify": model.Now(),
			}).Error
	})
}
