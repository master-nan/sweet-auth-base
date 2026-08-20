package service

import (
	"backend/enum"
	"backend/model"
)

func dataPermissionQueryTable(tableCode string, fields ...model.SysTableField) model.SysTable {
	return model.SysTable{Basic: model.Basic{State: true}, TableCode: tableCode, TableFields: fields}
}

func dataPermissionQueryField(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
	return model.SysTableField{Basic: model.Basic{State: true}, FieldCode: code, FieldType: fieldType, IsListShow: true, IsQuickSearch: quick, IsAdvancedSearch: true, IsSort: true}
}

func dataResourceQueryTable() model.SysTable {
	return dataPermissionQueryTable("sys_data_resource", dataPermissionQueryField("resource_code", enum.VarcharFieldType, true), dataPermissionQueryField("name", enum.VarcharFieldType, true), dataPermissionQueryField("resource_type", enum.VarcharFieldType, false))
}
func dataDimensionQueryTable() model.SysTable {
	return dataPermissionQueryTable("sys_data_dimension_definition", dataPermissionQueryField("code", enum.VarcharFieldType, true), dataPermissionQueryField("name", enum.VarcharFieldType, true), dataPermissionQueryField("category", enum.VarcharFieldType, false), dataPermissionQueryField("value_type", enum.VarcharFieldType, false))
}
func dataOwnershipQueryTable() model.SysTable {
	return dataPermissionQueryTable("sys_data_ownership_field", dataPermissionQueryField("resource_id", enum.BigIntFieldType, false), dataPermissionQueryField("ownership_code", enum.VarcharFieldType, true), dataPermissionQueryField("dimension_id", enum.BigIntFieldType, false), dataPermissionQueryField("binding_type", enum.VarcharFieldType, false), dataPermissionQueryField("value_type", enum.VarcharFieldType, false))
}
func dataPolicyQueryTable() model.SysTable {
	return dataPermissionQueryTable("sys_data_policy", dataPermissionQueryField("code", enum.VarcharFieldType, true), dataPermissionQueryField("name", enum.VarcharFieldType, true), dataPermissionQueryField("policy_type", enum.VarcharFieldType, false))
}
func dataPolicyRuleQueryTable() model.SysTable {
	return dataPermissionQueryTable("sys_data_policy_rule", dataPermissionQueryField("policy_id", enum.BigIntFieldType, false), dataPermissionQueryField("ownership_code", enum.VarcharFieldType, true), dataPermissionQueryField("sequence", enum.IntFieldType, false))
}
func dataGrantQueryTable() model.SysTable {
	return dataPermissionQueryTable("sys_data_grant", dataPermissionQueryField("subject_type", enum.VarcharFieldType, true), dataPermissionQueryField("subject_id", enum.BigIntFieldType, false), dataPermissionQueryField("resource_id", enum.BigIntFieldType, false), dataPermissionQueryField("operation", enum.VarcharFieldType, false), dataPermissionQueryField("policy_id", enum.BigIntFieldType, false))
}
