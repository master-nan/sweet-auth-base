package main

import (
	"backend/enum"
	"backend/internal/utils"
	"backend/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	integrationMenuID             = 1200
	externalSystemMenuID          = 1201
	interfaceDefinitionMenuID     = 1202
	credentialMenuID              = 1203
	externalSystemTableCode       = "integration_external_system"
	interfaceDefinitionTableCode  = "integration_interface_definition"
	credentialTableCode           = "integration_credential"
	integrationExecutionTableCode = "integration_execution"
)

func seedIntegrationConfigurationFoundation(db *gorm.DB, sf *utils.Snowflake) error {
	if !db.Migrator().HasTable(&model.ExternalSystem{}) {
		return nil
	}
	root, err := seedMenu(db, sf, directoryMenu(menu(
		integrationMenuID, 0, "integration", "integration", "src/components/Layout/Layout.vue",
		"router.integration.default", "hub", 5,
	)))
	if err != nil {
		return err
	}
	child, err := seedMenu(db, sf, menuWithTable(menu(
		externalSystemMenuID, root.Id, "integration_external_system", "external-system",
		"pages/integration/external-system/Index.vue", "router.integration.externalSystem", "dns", 1,
	), externalSystemTableCode))
	if err != nil {
		return err
	}
	interfaceMenu, err := seedMenu(db, sf, menuWithTable(menu(
		interfaceDefinitionMenuID, root.Id, "integration_interface_definition", "interface-definition",
		"pages/integration/interface-definition/Index.vue", "router.integration.interfaceDefinition", "api", 2,
	), interfaceDefinitionTableCode))
	if err != nil {
		return err
	}
	credentialMenu, err := seedMenu(db, sf, menuWithTable(menu(
		credentialMenuID, root.Id, "integration_credential", "credential",
		"pages/integration/credential/Index.vue", "router.integration.credential", "key", 3,
	), credentialTableCode))
	if err != nil {
		return err
	}

	role, err := seedRole(db, sf)
	if err != nil {
		return err
	}
	for _, item := range []model.SysMenu{root, child, interfaceMenu, credentialMenu} {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysRoleMenu{
			RoleId: role.Id,
			MenuId: item.Id,
		}).Error; err != nil {
			return err
		}
	}

	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(12001, child.Id, "列表查询", "integration_external_system_query", enum.Top, "query", "search", "primary", 90, "/admin/integration/external-system/query", "POST"),
		menuButtonWithAPI(12002, child.Id, "详情", "integration_external_system_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/integration/external-system/:id", "GET"),
		apiPermissionWithAPI(12003, child.Id, "页面元数据", "integration_external_system_metadata", enum.Top, "metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		menuButtonWithAPI(12004, child.Id, "新增", "integration_external_system_create", enum.Top, "create", "add", "primary", 1, "/admin/integration/external-system", "POST"),
		menuButtonWithAPI(12005, child.Id, "编辑", "integration_external_system_update", enum.Line, "update", "edit", "primary", 1, "/admin/integration/external-system/:id", "PUT"),
		menuButtonWithAPI(12006, child.Id, "启用", "integration_external_system_enable", enum.Line, "enable", "play_arrow", "positive", 2, "/admin/integration/external-system/:id/enable", "PUT"),
		menuButtonWithAPI(12007, child.Id, "停用", "integration_external_system_disable", enum.Line, string(enum.ButtonActionDisable), "block", "warning", 3, "/admin/integration/external-system/:id/disable", "PUT"),
		apiPermissionWithAPI(12101, interfaceMenu.Id, "列表查询", "integration_interface_definition_query", enum.Top, "query", "search", "primary", 90, "/admin/integration/interface-definition/query", "POST"),
		menuButtonWithAPI(12102, interfaceMenu.Id, "详情", "integration_interface_definition_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/integration/interface-definition/:id", "GET"),
		apiPermissionWithAPI(12103, interfaceMenu.Id, "页面元数据", "integration_interface_definition_metadata", enum.Top, "metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		menuButtonWithAPI(12104, interfaceMenu.Id, "新增", "integration_interface_definition_create", enum.Top, "create", "add", "primary", 1, "/admin/integration/interface-definition", "POST"),
		menuButtonWithAPI(12105, interfaceMenu.Id, "编辑", "integration_interface_definition_update", enum.Line, "update", "edit", "primary", 1, "/admin/integration/interface-definition/:id", "PUT"),
		menuButtonWithAPI(12106, interfaceMenu.Id, "创建版本", "integration_interface_definition_create_version", enum.Line, "create_version", "content_copy", "primary", 2, "/admin/integration/interface-definition/:id/versions", "POST"),
		menuButtonWithAPI(12107, interfaceMenu.Id, "启用", "integration_interface_definition_enable", enum.Line, "enable", "play_arrow", "positive", 3, "/admin/integration/interface-definition/:id/enable", "PUT"),
		menuButtonWithAPI(12108, interfaceMenu.Id, "停用", "integration_interface_definition_disable", enum.Line, string(enum.ButtonActionDisable), "block", "warning", 4, "/admin/integration/interface-definition/:id/disable", "PUT"),
		apiPermissionWithAPI(12201, credentialMenu.Id, "列表查询", "integration_credential_query", enum.Top, "query", "search", "primary", 90, "/admin/integration/credential/query", "POST"),
		menuButtonWithAPI(12202, credentialMenu.Id, "详情", "integration_credential_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/integration/credential/:id", "GET"),
		apiPermissionWithAPI(12203, credentialMenu.Id, "页面元数据", "integration_credential_metadata", enum.Top, "metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		menuButtonWithAPI(12204, credentialMenu.Id, "新增", "integration_credential_create", enum.Top, "create", "add", "primary", 1, "/admin/integration/credential", "POST"),
		menuButtonWithAPI(12205, credentialMenu.Id, "编辑", "integration_credential_update", enum.Line, "update", "edit", "primary", 1, "/admin/integration/credential/:id", "PUT"),
		menuButtonWithAPI(12206, credentialMenu.Id, "轮换", "integration_credential_rotate", enum.Line, "rotate", "sync", "warning", 2, "/admin/integration/credential/:id/rotate", "POST"),
		menuButtonWithAPI(12207, credentialMenu.Id, "启用", "integration_credential_enable", enum.Line, "enable", "play_arrow", "positive", 3, "/admin/integration/credential/:id/enable", "PUT"),
		menuButtonWithAPI(12208, credentialMenu.Id, "停用", "integration_credential_disable", enum.Line, string(enum.ButtonActionDisable), "block", "warning", 4, "/admin/integration/credential/:id/disable", "PUT"),
		menuButtonWithAPI(12209, credentialMenu.Id, "吊销", "integration_credential_revoke", enum.Line, "revoke", "gpp_bad", "negative", 5, "/admin/integration/credential/:id/revoke", "PUT"),
		apiPermissionWithAPI(12301, root.Id, "执行列表查询", "integration_execution_query", enum.Top, "query_execution", "search", "primary", 101, "/admin/integration/execution/query", "POST"),
		apiPermissionWithAPI(12302, root.Id, "执行详情", "integration_execution_detail", enum.Top, "detail_execution", "visibility", "primary", 102, "/admin/integration/execution/:id", "GET"),
		apiPermissionWithAPI(12303, root.Id, "创建执行", "integration_execution_create", enum.Top, "create_execution", "add", "primary", 103, "/admin/integration/execution", "POST"),
		apiPermissionWithAPI(12304, root.Id, "启动执行", "integration_execution_start", enum.Top, "start_execution", "play_arrow", "primary", 104, "/admin/integration/execution/:id/start", "PUT"),
		apiPermissionWithAPI(12305, root.Id, "完成执行", "integration_execution_complete", enum.Top, "complete_execution", "check_circle", "positive", 105, "/admin/integration/execution/:id/complete", "PUT"),
		apiPermissionWithAPI(12306, root.Id, "标记执行失败", "integration_execution_fail", enum.Top, "fail_execution", "error", "negative", 106, "/admin/integration/execution/:id/fail", "PUT"),
		apiPermissionWithAPI(12307, root.Id, "取消执行", "integration_execution_cancel", enum.Top, "cancel_execution", "cancel", "warning", 107, "/admin/integration/execution/:id/cancel", "PUT"),
	}
	return seedMenuButtons(db, sf, role.Id, role.Name, buttons)
}
