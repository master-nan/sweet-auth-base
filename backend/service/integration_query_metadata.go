package service

import (
	"backend/enum"
	"backend/model"
)

func integrationQueryTable(tableCode string, fields ...model.SysTableField) model.SysTable {
	return model.SysTable{Basic: model.Basic{State: true}, TableCode: tableCode, TableFields: fields}
}

func integrationQueryField(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
	return model.SysTableField{Basic: model.Basic{State: true}, FieldCode: code, FieldType: fieldType, IsQuickSearch: quick, IsAdvancedSearch: true, IsSort: true}
}

func externalSystemQueryTable() model.SysTable {
	return integrationQueryTable("integration_external_system",
		integrationQueryField("system_code", enum.VarcharFieldType, true), integrationQueryField("name", enum.VarcharFieldType, true),
		integrationQueryField("system_type", enum.VarcharFieldType, false), integrationQueryField("owner_identifier", enum.VarcharFieldType, true),
		integrationQueryField("owner_name", enum.VarcharFieldType, true), integrationQueryField("status", enum.VarcharFieldType, false),
		integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func credentialQueryTable() model.SysTable {
	return integrationQueryTable("integration_credential",
		integrationQueryField("credential_code", enum.VarcharFieldType, true), integrationQueryField("name", enum.VarcharFieldType, true),
		integrationQueryField("external_system_id", enum.BigIntFieldType, false),
		integrationQueryField("credential_type", enum.VarcharFieldType, false), integrationQueryField("status", enum.VarcharFieldType, false),
		integrationQueryField("expires_at", enum.DatetimeFieldType, false), integrationQueryField("version", enum.IntFieldType, false),
		integrationQueryField("rotated_at", enum.DatetimeFieldType, false), integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func interfaceDefinitionQueryTable() model.SysTable {
	return integrationQueryTable("integration_interface_definition",
		integrationQueryField("interface_code", enum.VarcharFieldType, true), integrationQueryField("name", enum.VarcharFieldType, true),
		integrationQueryField("external_system_id", enum.BigIntFieldType, false),
		integrationQueryField("version", enum.IntFieldType, false), integrationQueryField("protocol", enum.VarcharFieldType, false),
		integrationQueryField("http_method", enum.VarcharFieldType, false), integrationQueryField("relative_path", enum.VarcharFieldType, true),
		integrationQueryField("status", enum.VarcharFieldType, false), integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func retryPolicyQueryTable() model.SysTable {
	return integrationQueryTable("integration_retry_policy",
		integrationQueryField("policy_code", enum.VarcharFieldType, true), integrationQueryField("policy_name", enum.VarcharFieldType, true),
		integrationQueryField("version", enum.IntFieldType, false), integrationQueryField("status", enum.VarcharFieldType, false),
		integrationQueryField("max_attempts", enum.IntFieldType, false), integrationQueryField("backoff_type", enum.VarcharFieldType, false),
		integrationQueryField("initial_delay_ms", enum.BigIntFieldType, false), integrationQueryField("max_delay_ms", enum.BigIntFieldType, false),
		integrationQueryField("retry_window_ms", enum.BigIntFieldType, false), integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func syncTaskQueryTable() model.SysTable {
	return integrationQueryTable("integration_sync_task",
		integrationQueryField("task_code", enum.VarcharFieldType, true), integrationQueryField("task_name", enum.VarcharFieldType, true),
		integrationQueryField("version", enum.IntFieldType, false), integrationQueryField("status", enum.VarcharFieldType, false),
		integrationQueryField("external_system_id", enum.BigIntFieldType, false), integrationQueryField("interface_definition_id", enum.BigIntFieldType, false),
		integrationQueryField("consumer_code", enum.VarcharFieldType, true), integrationQueryField("schedule_type", enum.VarcharFieldType, false),
		integrationQueryField("checkpoint_mode", enum.VarcharFieldType, false), integrationQueryField("checkpoint_at", enum.DatetimeFieldType, false),
		integrationQueryField("gmt_modify", enum.DatetimeFieldType, false))
}

func syncBatchQueryTable() model.SysTable {
	return integrationQueryTable("integration_sync_batch",
		integrationQueryField("batch_no", enum.VarcharFieldType, true), integrationQueryField("sync_task_id", enum.BigIntFieldType, false),
		integrationQueryField("task_code", enum.VarcharFieldType, true), integrationQueryField("task_name", enum.VarcharFieldType, true),
		integrationQueryField("task_version", enum.IntFieldType, false), integrationQueryField("trigger_type", enum.VarcharFieldType, false),
		integrationQueryField("status", enum.VarcharFieldType, false), integrationQueryField("window_start", enum.DatetimeFieldType, false),
		integrationQueryField("window_end", enum.DatetimeFieldType, false), integrationQueryField("gmt_create", enum.DatetimeFieldType, false),
		integrationQueryField("started_at", enum.DatetimeFieldType, false), integrationQueryField("completed_at", enum.DatetimeFieldType, false))
}
