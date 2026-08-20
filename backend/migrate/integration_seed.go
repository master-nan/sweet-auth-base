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

func applyExternalSystemFieldDefaults(tableCode string, field *model.SysTableField) {
	if tableCode != externalSystemTableCode {
		return
	}
	field.IsListShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsSort = true

	switch field.FieldCode {
	case "system_code":
		field.FieldName = "系统编码"
		field.IsListShow = true
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.Sequence = 1
	case "name":
		field.FieldName = "系统名称"
		field.IsListShow = true
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.Sequence = 2
	case "system_type":
		field.FieldName = "系统类型"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.InputType = enum.SelectInputType
		field.DictCode = utils.StringPtr("integration_external_system_type")
		field.Sequence = 3
	case "base_url":
		field.FieldName = "基础地址"
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.Sequence = 4
	case "owner_identifier":
		field.FieldName = "负责人标识"
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.Sequence = 5
	case "owner_name":
		field.FieldName = "负责人"
		field.IsListShow = true
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.Sequence = 6
	case "status":
		field.FieldName = "状态"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.InputType = enum.SelectInputType
		field.DictCode = utils.StringPtr("integration_external_system_status")
		field.Sequence = 7
	case "description":
		field.FieldName = "描述"
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.InputType = enum.TextareaInputType
		field.Sequence = 8
	case "revision":
		field.FieldName = "版本"
	case "gmt_modify":
		field.FieldName = "更新时间"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.InputType = enum.DatetimePickerInputType
		field.Sequence = 9
	}
}

func applyInterfaceDefinitionFieldDefaults(tableCode string, field *model.SysTableField) {
	if tableCode != interfaceDefinitionTableCode {
		return
	}
	field.IsListShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsSort = true
	switch field.FieldCode {
	case "external_system_id":
		field.FieldName = "所属系统"
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.Sequence = 1
	case "interface_code":
		field.FieldName = "接口编码"
		field.IsListShow = true
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.Sequence = 2
	case "name":
		field.FieldName = "接口名称"
		field.IsListShow = true
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.Sequence = 3
	case "version":
		field.FieldName = "版本"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.Sequence = 4
	case "protocol":
		field.FieldName = "协议"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.InputType = enum.SelectInputType
		field.DictCode = utils.StringPtr("integration_interface_protocol")
		field.Sequence = 5
	case "http_method":
		field.FieldName = "HTTP Method"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.InputType = enum.SelectInputType
		field.DictCode = utils.StringPtr("http_method")
		field.Sequence = 6
	case "relative_path":
		field.FieldName = "相对路径"
		field.IsListShow = true
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.Sequence = 7
	case "credential_id":
		field.FieldName = "凭证引用"
	case "timeout_seconds":
		field.FieldName = "超时（秒）"
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.Sequence = 8
	case "response_limit":
		field.FieldName = "响应大小限制（字节）"
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.Sequence = 9
	case "retry_policy_id":
		field.FieldName = "重试策略引用"
	case "status":
		field.FieldName = "状态"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.InputType = enum.SelectInputType
		field.DictCode = utils.StringPtr("integration_interface_definition_status")
		field.Sequence = 10
	case "description":
		field.FieldName = "描述"
		field.IsInsertShow = true
		field.IsUpdateShow = true
		field.InputType = enum.TextareaInputType
		field.Sequence = 11
	case "revision":
		field.FieldName = "修订号"
	case "gmt_modify":
		field.FieldName = "更新时间"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.InputType = enum.DatetimePickerInputType
		field.Sequence = 12
	}
}

func applyCredentialFieldDefaults(tableCode string, field *model.SysTableField) {
	if tableCode != credentialTableCode {
		return
	}
	field.IsListShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsSort = true
	switch field.FieldCode {
	case "external_system_id":
		field.FieldName = "所属系统"
		field.IsAdvancedSearch = true
		field.Sequence = 1
	case "credential_code":
		field.FieldName = "凭证编码"
		field.IsListShow = true
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.Sequence = 2
	case "name":
		field.FieldName = "凭证名称"
		field.IsListShow = true
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
		field.Sequence = 3
	case "credential_type":
		field.FieldName = "凭证类型"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.InputType = enum.SelectInputType
		field.DictCode = utils.StringPtr("integration_credential_type")
		field.Sequence = 4
	case "status":
		field.FieldName = "状态"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.InputType = enum.SelectInputType
		field.DictCode = utils.StringPtr("integration_credential_status")
		field.Sequence = 5
	case "expires_at":
		field.FieldName = "有效期"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.InputType = enum.DatetimePickerInputType
		field.Sequence = 6
	case "version":
		field.FieldName = "秘密版本"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.Sequence = 7
	case "rotated_at":
		field.FieldName = "轮换时间"
		field.IsListShow = true
		field.IsAdvancedSearch = true
		field.InputType = enum.DatetimePickerInputType
		field.Sequence = 8
	case "description":
		field.FieldName = "描述"
		field.Sequence = 9
	case "gmt_modify":
		field.FieldName = "更新时间"
		field.IsAdvancedSearch = true
		field.InputType = enum.DatetimePickerInputType
		field.Sequence = 10
	case "secret_storage_ref", "secret_ciphertext", "secret_nonce", "secret_fingerprint":
		field.IsSort = false
	}
}

func applyRetryPolicyFieldDefaults(tableCode string, field *model.SysTableField) {
	if tableCode != retryPolicyTableCode {
		return
	}
	field.IsListShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsSort = true
	switch field.FieldCode {
	case "policy_code":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "策略编码", true, true, true, 1
	case "policy_name":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "策略名称", true, true, true, 2
	case "version":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "版本", true, true, 3
	case "status":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "状态", true, true, enum.SelectInputType, utils.StringPtr("integration_retry_policy_status"), 4
	case "max_attempts":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "最大尝试次数", true, true, 5
	case "backoff_type":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "退避方式", true, true, enum.SelectInputType, utils.StringPtr("integration_retry_backoff_type"), 6
	case "initial_delay_ms":
		field.FieldName, field.IsListShow, field.Sequence = "初始延迟（毫秒）", true, 7
	case "max_delay_ms":
		field.FieldName, field.IsListShow, field.Sequence = "最大延迟（毫秒）", true, 8
	case "retry_window_ms":
		field.FieldName, field.IsListShow, field.Sequence = "重试窗口（毫秒）", true, 9
	case "gmt_modify":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.Sequence = "更新时间", true, true, enum.DatetimePickerInputType, 10
	case "retryable_error_categories", "retryable_http_statuses", "description", "respect_retry_after", "backoff_multiplier", "jitter_type", "jitter_ratio":
		field.IsSort = false
	}
}

func applyIntegrationSyncTaskFieldDefaults(tableCode string, field *model.SysTableField) {
	if tableCode != integrationSyncTaskTableCode {
		return
	}
	field.IsListShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsSort = true
	switch field.FieldCode {
	case "task_code":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "任务编码", true, true, true, 1
	case "task_name":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "任务名称", true, true, true, 2
	case "version":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "版本", true, true, 3
	case "status":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "状态", true, true, enum.SelectInputType, utils.StringPtr("integration_sync_task_status"), 4
	case "external_system_id":
		field.FieldName, field.IsAdvancedSearch, field.Sequence = "外部系统", true, 5
	case "interface_definition_id":
		field.FieldName, field.IsAdvancedSearch, field.Sequence = "接口定义", true, 6
	case "consumer_code":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.Sequence = "Consumer", true, true, 7
	case "schedule_type":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "调度方式", true, true, 8
	case "checkpoint_mode":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "Checkpoint模式", true, true, 9
	case "checkpoint_at":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "当前Checkpoint", true, enum.DatetimePickerInputType, 10
	case "gmt_modify":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.Sequence = "更新时间", true, true, enum.DatetimePickerInputType, 11
	case "input_plan", "description", "cron_expression", "timezone", "next_scheduled_at", "last_scheduled_at", "initial_checkpoint_at":
		field.IsSort = false
	}
}

func applyIntegrationSyncBatchFieldDefaults(tableCode string, field *model.SysTableField) {
	if tableCode != integrationSyncBatchTableCode {
		return
	}
	field.IsListShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsSort = true
	switch field.FieldCode {
	case "batch_no":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "批次编号", true, true, true, 1
	case "task_code":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "任务编码", true, true, true, 2
	case "task_name":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.Sequence = "任务名称", true, true, 3
	case "task_version":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "任务版本", true, true, 4
	case "trigger_type":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "触发类型", true, true, 5
	case "status":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "状态", true, true, enum.SelectInputType, utils.StringPtr("integration_sync_batch_status"), 6
	case "window_start":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "窗口开始", true, enum.DatetimePickerInputType, 7
	case "window_end":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "窗口结束", true, enum.DatetimePickerInputType, 8
	case "current_slice_no":
		field.FieldName, field.IsListShow, field.Sequence = "当前切片", true, 9
	case "planned_slice_count":
		field.FieldName, field.IsListShow, field.Sequence = "计划切片", true, 10
	case "execution_count":
		field.FieldName, field.IsListShow, field.Sequence = "执行数", true, 11
	case "started_at":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "开始时间", true, enum.DatetimePickerInputType, 12
	case "completed_at":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "结束时间", true, enum.DatetimePickerInputType, 13
	case "result_summary", "trigger_key", "triggered_by_user_name":
		field.IsSort = false
	}
}

func applyIntegrationExecutionFieldDefaults(tableCode string, field *model.SysTableField) {
	if tableCode != integrationExecutionTableCode {
		return
	}
	field.IsListShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsSort = true
	switch field.FieldCode {
	case "execution_no":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "执行编号", true, true, true, 1
	case "external_system_id":
		field.FieldName, field.IsAdvancedSearch, field.Sequence = "外部系统", true, 2
	case "external_system_code":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "系统编码", true, true, true, 3
	case "external_system_name":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.Sequence = "系统名称", true, true, 4
	case "interface_definition_id":
		field.FieldName, field.IsAdvancedSearch, field.Sequence = "接口定义", true, 5
	case "interface_code":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.IsAdvancedSearch, field.Sequence = "接口编码", true, true, true, 6
	case "interface_name":
		field.FieldName, field.IsListShow, field.IsQuickSearch, field.Sequence = "接口名称", true, true, 7
	case "interface_version":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "接口版本", true, true, 8
	case "trigger_source":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "触发来源", true, true, enum.SelectInputType, utils.StringPtr("integration_trigger_source"), 9
	case "status":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "状态", true, true, enum.SelectInputType, utils.StringPtr("integration_execution_status"), 10
	case "current_attempt":
		field.FieldName, field.IsListShow, field.Sequence = "当前尝试", true, 11
	case "error_category":
		field.FieldName, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "错误分类", true, enum.SelectInputType, utils.StringPtr("integration_error_category"), 12
	case "gmt_create":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.Sequence = "创建时间", true, true, enum.DatetimePickerInputType, 13
	case "gmt_modify":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "更新时间", true, enum.DatetimePickerInputType, 14
	case "started_at":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "开始时间", true, enum.DatetimePickerInputType, 15
	case "completed_at":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "完成时间", true, enum.DatetimePickerInputType, 16
	case "idempotency_scope", "idempotency_key", "input_hash", "result_hash", "result_summary":
		field.IsSort = false
	}
}

func applyIntegrationLogFieldDefaults(tableCode string, field *model.SysTableField) {
	if tableCode != integrationLogTableCode {
		return
	}
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
	field.IsSort = true
	switch field.FieldCode {
	case "execution_id":
		field.FieldName, field.IsAdvancedSearch, field.Sequence = "执行记录", true, 1
	case "attempt_no":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "Attempt", true, true, 2
	case "status":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "状态", true, true, enum.SelectInputType, utils.StringPtr("integration_log_status"), 3
	case "http_status":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.Sequence = "HTTP状态", true, true, 4
	case "error_category":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "错误分类", true, true, enum.SelectInputType, utils.StringPtr("integration_error_category"), 5
	case "result_certainty":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.DictCode, field.Sequence = "结果确定性", true, true, enum.SelectInputType, utils.StringPtr("integration_result_certainty"), 6
	case "started_at":
		field.FieldName, field.IsListShow, field.IsAdvancedSearch, field.InputType, field.Sequence = "开始时间", true, true, enum.DatetimePickerInputType, 7
	case "ended_at":
		field.FieldName, field.IsListShow, field.InputType, field.Sequence = "结束时间", true, enum.DatetimePickerInputType, 8
	case "duration_ms":
		field.FieldName, field.IsListShow, field.Sequence = "耗时（毫秒）", true, 9
	case "result_summary", "result_hash", "request_id", "trace_id", "worker_id", "response_content_type", "credential_code", "credential_version", "credential_fingerprint_summary":
		field.IsSort = false
	}
}
