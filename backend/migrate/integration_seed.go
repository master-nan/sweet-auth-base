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
	executionMenuID               = 1204
	logMenuID                     = 1205
	retryPolicyMenuID             = 1206
	syncTaskMenuID                = 1207
	syncBatchMenuID               = 1208
	externalSystemTableCode       = "integration_external_system"
	interfaceDefinitionTableCode  = "integration_interface_definition"
	credentialTableCode           = "integration_credential"
	retryPolicyTableCode          = "integration_retry_policy"
	integrationSyncTaskTableCode  = "integration_sync_task"
	integrationSyncBatchTableCode = "integration_sync_batch"
	integrationExecutionTableCode = "integration_execution"
	integrationLogTableCode       = "integration_log"
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
	retryPolicyMenu, err := seedMenu(db, sf, menuWithTable(menu(
		retryPolicyMenuID, root.Id, "integration_retry_policy", "retry-policy",
		"pages/integration/retry-policy/Index.vue", "router.integration.retryPolicy", "autorenew", 4,
	), retryPolicyTableCode))
	if err != nil {
		return err
	}
	syncTaskMenu, err := seedMenu(db, sf, menuWithTable(menu(
		syncTaskMenuID, root.Id, "integration_sync_task", "sync-task",
		"pages/integration/sync-task/Index.vue", "router.integration.syncTask", "sync_alt", 5,
	), integrationSyncTaskTableCode))
	if err != nil {
		return err
	}
	syncBatchMenu, err := seedMenu(db, sf, menuWithTable(menu(
		syncBatchMenuID, root.Id, "integration_sync_batch", "sync-batch",
		"pages/integration/sync-batch/Index.vue", "router.integration.syncBatch", "view_timeline", 6,
	), integrationSyncBatchTableCode))
	if err != nil {
		return err
	}
	executionMenu, err := seedMenu(db, sf, menuWithTable(menu(
		executionMenuID, root.Id, "integration_execution", "execution",
		"pages/integration/execution/Index.vue", "router.integration.execution", "play_circle", 7,
	), integrationExecutionTableCode))
	if err != nil {
		return err
	}
	logMenu, err := seedMenu(db, sf, menuWithTable(menu(
		logMenuID, root.Id, "integration_log", "log",
		"pages/integration/log/Index.vue", "router.integration.log", "history", 8,
	), integrationLogTableCode))
	if err != nil {
		return err
	}

	role, err := seedRole(db, sf)
	if err != nil {
		return err
	}
	for _, item := range []model.SysMenu{root, child, interfaceMenu, credentialMenu, retryPolicyMenu, syncTaskMenu, syncBatchMenu, executionMenu, logMenu} {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysRoleMenu{
			RoleId: role.Id,
			MenuId: item.Id,
		}).Error; err != nil {
			return err
		}
	}
	if err := retireMenuButtons(db, root.Id, []string{
		"integration_execution_query", "integration_execution_detail", "integration_execution_create",
		"integration_execution_cancel",
	}); err != nil {
		return err
	}
	if err := removeDeprecatedIntegrationExecutionCommands(db, root.Id); err != nil {
		return err
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
		apiPermissionWithAPI(12401, retryPolicyMenu.Id, "列表查询", "integration_retry_policy_query", enum.Top, "query", "search", "primary", 90, "/admin/integration/retry-policy/query", "POST"),
		menuButtonWithAPI(12402, retryPolicyMenu.Id, "详情", "integration_retry_policy_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/integration/retry-policy/:id", "GET"),
		apiPermissionWithAPI(12403, retryPolicyMenu.Id, "页面元数据", "integration_retry_policy_metadata", enum.Top, "metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		menuButtonWithAPI(12404, retryPolicyMenu.Id, "新增", "integration_retry_policy_create", enum.Top, "create", "add", "primary", 1, "/admin/integration/retry-policy", "POST"),
		menuButtonWithAPI(12405, retryPolicyMenu.Id, "编辑", "integration_retry_policy_update", enum.Line, "update", "edit", "primary", 1, "/admin/integration/retry-policy/:id", "PUT"),
		menuButtonWithAPI(12406, retryPolicyMenu.Id, "创建版本", "integration_retry_policy_create_version", enum.Line, "create_version", "content_copy", "primary", 2, "/admin/integration/retry-policy/:id/versions", "POST"),
		menuButtonWithAPI(12407, retryPolicyMenu.Id, "启用", "integration_retry_policy_enable", enum.Line, "enable", "play_arrow", "positive", 3, "/admin/integration/retry-policy/:id/enable", "PUT"),
		menuButtonWithAPI(12408, retryPolicyMenu.Id, "停用", "integration_retry_policy_disable", enum.Line, string(enum.ButtonActionDisable), "block", "warning", 4, "/admin/integration/retry-policy/:id/disable", "PUT"),
		apiPermissionWithAPI(12501, syncTaskMenu.Id, "列表查询", "integration_sync_task_query", enum.Top, "query", "search", "primary", 90, "/admin/integration/sync-task/query", "POST"),
		menuButtonWithAPI(12502, syncTaskMenu.Id, "详情", "integration_sync_task_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/integration/sync-task/:id", "GET"),
		apiPermissionWithAPI(12503, syncTaskMenu.Id, "页面元数据", "integration_sync_task_metadata", enum.Top, "metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(12504, syncTaskMenu.Id, "编辑数据", "integration_sync_task_edit_data", enum.Top, "edit_data", "edit", "primary", 93, "/admin/integration/sync-task/:id/edit", "GET"),
		apiPermissionWithAPI(12505, syncTaskMenu.Id, "Consumer元数据", "integration_sync_task_consumer_metadata", enum.Top, "consumer_metadata", "list", "primary", 94, "/admin/integration/sync-task/consumers", "GET"),
		menuButtonWithAPI(12506, syncTaskMenu.Id, "新增", "integration_sync_task_create", enum.Top, "create", "add", "primary", 1, "/admin/integration/sync-task", "POST"),
		menuButtonWithAPI(12507, syncTaskMenu.Id, "编辑", "integration_sync_task_update", enum.Line, "update", "edit", "primary", 1, "/admin/integration/sync-task/:id", "PUT"),
		menuButtonWithAPI(12508, syncTaskMenu.Id, "创建版本", "integration_sync_task_create_version", enum.Line, "create_version", "content_copy", "primary", 2, "/admin/integration/sync-task/:id/versions", "POST"),
		menuButtonWithAPI(12509, syncTaskMenu.Id, "启用", "integration_sync_task_enable", enum.Line, "enable", "play_arrow", "positive", 3, "/admin/integration/sync-task/:id/enable", "PUT"),
		menuButtonWithAPI(12510, syncTaskMenu.Id, "停用", "integration_sync_task_disable", enum.Line, string(enum.ButtonActionDisable), "block", "warning", 4, "/admin/integration/sync-task/:id/disable", "PUT"),
		menuButtonWithAPI(12511, syncTaskMenu.Id, "运行一次", "integration_sync_task_run", enum.Line, "run", "play_circle", "primary", 5, "/admin/integration/sync-task/:id/run", "POST"),
		apiPermissionWithAPI(12601, syncBatchMenu.Id, "列表查询", "integration_sync_batch_query", enum.Top, "query", "search", "primary", 90, "/admin/integration/sync-batch/query", "POST"),
		menuButtonWithAPI(12602, syncBatchMenu.Id, "详情", "integration_sync_batch_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/integration/sync-batch/:id", "GET"),
		apiPermissionWithAPI(12603, syncBatchMenu.Id, "页面元数据", "integration_sync_batch_metadata", enum.Top, "metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(12301, executionMenu.Id, "执行列表查询", "integration_execution_query", enum.Top, "query", "search", "primary", 90, "/admin/integration/execution/query", "POST"),
		menuButtonWithAPI(12302, executionMenu.Id, "执行详情", "integration_execution_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/integration/execution/:id", "GET"),
		apiPermissionWithAPI(12303, root.Id, "提交执行", "integration_execution_create", enum.Top, "create", "add", "primary", 120, "/admin/integration/execution", "POST"),
		apiPermissionWithAPI(12308, logMenu.Id, "调用日志查询", "integration_log_query", enum.Top, "query", "search", "primary", 90, "/admin/integration/log/query", "POST"),
		menuButtonWithAPI(12309, logMenu.Id, "调用日志详情", "integration_log_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/integration/log/:id", "GET"),
		menuButtonWithAPI(12311, executionMenu.Id, "取消执行", "integration_execution_cancel", enum.Line, "cancel", "cancel", "warning", 1, "/admin/integration/execution/:id/cancel", "PUT"),
		apiPermissionWithAPI(12310, root.Id, "Worker状态", "integration_worker_status", enum.Top, "status", "monitor_heart", "primary", 110, "/admin/integration/worker/status", "GET"),
	}
	return seedMenuButtons(db, sf, role.Id, role.Name, buttons)
}

func removeDeprecatedIntegrationExecutionCommands(db *gorm.DB, menuID int) error {
	codes := []string{
		"integration_execution_start",
		"integration_execution_complete",
		"integration_execution_fail",
	}
	paths := []string{
		"/admin/integration/execution/:id/start",
		"/admin/integration/execution/:id/complete",
		"/admin/integration/execution/:id/fail",
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var buttonIDs []int
		if err := tx.Unscoped().Model(&model.SysMenuButton{}).
			Where("menu_id = ? AND code IN ? AND path IN ? AND method = ?", menuID, codes, paths, "PUT").
			Pluck("id", &buttonIDs).Error; err != nil {
			return err
		}
		if len(buttonIDs) > 0 {
			if err := tx.Unscoped().Where("button_id IN ?", buttonIDs).Delete(&model.SysRoleMenuButton{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("id IN ?", buttonIDs).Delete(&model.SysMenuButton{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("ptype = ? AND v1 IN ? AND v2 = ?", "p", paths, "PUT").Delete(&model.CasbinRule{}).Error
	})
}
