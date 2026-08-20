package service

import (
	"backend/enum"
	"backend/model"
)

func queryMetadataTable(tableCode string, fields ...model.SysTableField) model.SysTable {
	return model.SysTable{Basic: model.Basic{State: true}, TableCode: tableCode, TableFields: fields}
}

func integrationQueryField(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
	return queryMetadataField(code, fieldType, quick, false)
}

func queryMetadataField(code string, fieldType enum.SysTableFieldType, quick, listVisible bool) model.SysTableField {
	return model.SysTableField{Basic: model.Basic{State: true}, FieldCode: code, FieldType: fieldType, IsListShow: listVisible, IsQuickSearch: quick, IsAdvancedSearch: true, IsSort: true}
}

func externalSystemQueryTable() model.SysTable {
	return queryMetadataTable("integration_external_system",
		integrationQueryField("system_code", enum.VarcharFieldType, true), integrationQueryField("name", enum.VarcharFieldType, true),
		integrationQueryField("system_type", enum.VarcharFieldType, false), integrationQueryField("owner_identifier", enum.VarcharFieldType, true),
		integrationQueryField("owner_name", enum.VarcharFieldType, true), integrationQueryField("status", enum.VarcharFieldType, false),
		integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func credentialQueryTable() model.SysTable {
	return queryMetadataTable("integration_credential",
		integrationQueryField("credential_code", enum.VarcharFieldType, true), integrationQueryField("name", enum.VarcharFieldType, true),
		integrationQueryField("external_system_id", enum.BigIntFieldType, false),
		integrationQueryField("credential_type", enum.VarcharFieldType, false), integrationQueryField("status", enum.VarcharFieldType, false),
		integrationQueryField("expires_at", enum.DatetimeFieldType, false), integrationQueryField("version", enum.IntFieldType, false),
		integrationQueryField("rotated_at", enum.DatetimeFieldType, false), integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func interfaceDefinitionQueryTable() model.SysTable {
	return queryMetadataTable("integration_interface_definition",
		integrationQueryField("interface_code", enum.VarcharFieldType, true), integrationQueryField("name", enum.VarcharFieldType, true),
		integrationQueryField("external_system_id", enum.BigIntFieldType, false),
		integrationQueryField("version", enum.IntFieldType, false), integrationQueryField("protocol", enum.VarcharFieldType, false),
		integrationQueryField("http_method", enum.VarcharFieldType, false), integrationQueryField("relative_path", enum.VarcharFieldType, true),
		integrationQueryField("status", enum.VarcharFieldType, false), integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func retryPolicyQueryTable() model.SysTable {
	return queryMetadataTable("integration_retry_policy",
		integrationQueryField("policy_code", enum.VarcharFieldType, true), integrationQueryField("policy_name", enum.VarcharFieldType, true),
		integrationQueryField("version", enum.IntFieldType, false), integrationQueryField("status", enum.VarcharFieldType, false),
		integrationQueryField("max_attempts", enum.IntFieldType, false), integrationQueryField("backoff_type", enum.VarcharFieldType, false),
		integrationQueryField("initial_delay_ms", enum.BigIntFieldType, false), integrationQueryField("max_delay_ms", enum.BigIntFieldType, false),
		integrationQueryField("retry_window_ms", enum.BigIntFieldType, false), integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func syncTaskQueryTable() model.SysTable {
	return queryMetadataTable("integration_sync_task",
		integrationQueryField("task_code", enum.VarcharFieldType, true), integrationQueryField("task_name", enum.VarcharFieldType, true),
		integrationQueryField("version", enum.IntFieldType, false), integrationQueryField("status", enum.VarcharFieldType, false),
		integrationQueryField("external_system_id", enum.BigIntFieldType, false), integrationQueryField("interface_definition_id", enum.BigIntFieldType, false),
		integrationQueryField("consumer_code", enum.VarcharFieldType, true), integrationQueryField("schedule_type", enum.VarcharFieldType, false),
		integrationQueryField("checkpoint_mode", enum.VarcharFieldType, false), integrationQueryField("checkpoint_at", enum.DatetimeFieldType, false),
		integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func syncBatchQueryTable() model.SysTable {
	return queryMetadataTable("integration_sync_batch",
		integrationQueryField("batch_no", enum.VarcharFieldType, true), integrationQueryField("sync_task_id", enum.BigIntFieldType, false),
		integrationQueryField("task_code", enum.VarcharFieldType, true), integrationQueryField("task_name", enum.VarcharFieldType, true),
		integrationQueryField("task_version", enum.IntFieldType, false), integrationQueryField("trigger_type", enum.VarcharFieldType, false),
		integrationQueryField("status", enum.VarcharFieldType, false), integrationQueryField("window_start", enum.DatetimeFieldType, false),
		integrationQueryField("window_end", enum.DatetimeFieldType, false), integrationQueryField("gmt_create", enum.DatetimeFieldType, false),
		integrationQueryField("started_at", enum.DatetimeFieldType, false), integrationQueryField("completed_at", enum.DatetimeFieldType, false))
}

func dataPermissionQueryField(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
	return queryMetadataField(code, fieldType, quick, true)
}

func dataResourceQueryTable() model.SysTable {
	return queryMetadataTable("sys_data_resource", dataPermissionQueryField("resource_code", enum.VarcharFieldType, true), dataPermissionQueryField("name", enum.VarcharFieldType, true), dataPermissionQueryField("resource_type", enum.VarcharFieldType, false))
}

func dataDimensionQueryTable() model.SysTable {
	return queryMetadataTable("sys_data_dimension_definition", dataPermissionQueryField("code", enum.VarcharFieldType, true), dataPermissionQueryField("name", enum.VarcharFieldType, true), dataPermissionQueryField("category", enum.VarcharFieldType, false), dataPermissionQueryField("value_type", enum.VarcharFieldType, false))
}

func dataOwnershipQueryTable() model.SysTable {
	return queryMetadataTable("sys_data_ownership_field", dataPermissionQueryField("resource_id", enum.BigIntFieldType, false), dataPermissionQueryField("ownership_code", enum.VarcharFieldType, true), dataPermissionQueryField("dimension_id", enum.BigIntFieldType, false), dataPermissionQueryField("binding_type", enum.VarcharFieldType, false), dataPermissionQueryField("value_type", enum.VarcharFieldType, false))
}

func dataPolicyQueryTable() model.SysTable {
	return queryMetadataTable("sys_data_policy", dataPermissionQueryField("code", enum.VarcharFieldType, true), dataPermissionQueryField("name", enum.VarcharFieldType, true), dataPermissionQueryField("policy_type", enum.VarcharFieldType, false))
}

func dataPolicyRuleQueryTable() model.SysTable {
	return queryMetadataTable("sys_data_policy_rule", dataPermissionQueryField("policy_id", enum.BigIntFieldType, false), dataPermissionQueryField("ownership_code", enum.VarcharFieldType, true), dataPermissionQueryField("sequence", enum.IntFieldType, false))
}

func dataGrantQueryTable() model.SysTable {
	return queryMetadataTable("sys_data_grant", dataPermissionQueryField("subject_type", enum.VarcharFieldType, true), dataPermissionQueryField("subject_id", enum.BigIntFieldType, false), dataPermissionQueryField("resource_id", enum.BigIntFieldType, false), dataPermissionQueryField("operation", enum.VarcharFieldType, false), dataPermissionQueryField("policy_id", enum.BigIntFieldType, false))
}
