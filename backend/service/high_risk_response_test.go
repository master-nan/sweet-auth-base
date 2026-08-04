package service

import (
	"backend/model"
	"encoding/json"
	"testing"

	"gorm.io/datatypes"
)

func TestHighRiskResponseSensitiveFieldsAreNotSerialized(t *testing.T) {
	creatorID := 101
	modifierID := 102
	deletedBy := 103
	internalName := "internal"
	basic := model.Basic{
		Id:         1,
		CreateUser: &creatorID,
		CreateName: &internalName,
		ModifyUser: &modifierID,
		ModifyName: &internalName,
		DeleteUser: &deletedBy,
		State:      true,
	}

	tests := []struct {
		name      string
		value     any
		forbidden []string
	}{
		{
			name: "file storage internals",
			value: fileDetailResponse(model.File{
				Basic:       basic,
				FileName:    "contract.pdf",
				FilePath:    "private/contract.pdf",
				FileMd5:     "secret-digest",
				StorageType: "internal-storage",
			}),
			forbidden: []string{"file_path", "file_md5", "storage_type", "createUser", "modify_user", "delete_user", "gmt_delete", "delete_name"},
		},
		{
			name: "menu associations",
			value: menuListResponse(model.SysMenu{
				Basic: basic,
				Roles: []model.SysRole{{Basic: basic}},
				MenuButtons: []model.SysMenuButton{{
					Basic: basic,
					Roles: []model.SysRole{{Basic: basic}},
				}},
			}),
			forbidden: []string{"roles", "menus", "users", "createUser", "modify_user", "delete_user", "gmt_delete", "delete_name"},
		},
		{
			name: "role user association",
			value: roleListResponse(model.SysRole{
				Basic: basic,
				Users: []model.SysUser{{Basic: basic, Password: "secret", AccessTokens: "token"}},
			}),
			forbidden: []string{"users", "password", "access_tokens", "createUser", "modify_user", "delete_user", "gmt_delete", "delete_name"},
		},
		{
			name: "table field dictionary association",
			value: tableFieldListResponse(model.SysTableField{
				Basic: basic,
				Dict:  model.SysDict{Basic: basic, DictName: "internal dictionary"},
			}),
			forbidden: []string{"dict", "createUser", "modify_user", "delete_user", "gmt_delete", "delete_name"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			var fields map[string]any
			if err := json.Unmarshal(payload, &fields); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			assertForbiddenResponseFields(t, fields, test.forbidden...)
		})
	}
}

func TestHighRiskListAndDetailWhitelists(t *testing.T) {
	report := model.ReportDefinition{
		Basic:        model.Basic{Id: 11, State: true},
		Code:         "monthly_sales",
		Name:         "月度销售",
		QueryConfig:  datatypes.JSON(`{"fields":["amount"]}`),
		LayoutConfig: datatypes.JSON(`{"sheet":{}}`),
	}
	reportList := marshalResponseFields(t, reportDefinitionListResponse(report))
	reportDetail := marshalResponseFields(t, reportDefinitionDetailResponse(report))
	assertMissingResponseFields(t, reportList, "query_config", "layout_config")
	assertPresentResponseFields(t, reportDetail, "query_config", "layout_config")

	table := model.SysTable{
		Basic:       model.Basic{Id: 12, State: true},
		TableName:   "运输订单",
		TableCode:   "transport_order",
		SQL:         "select internal_definition",
		TableFields: []model.SysTableField{{Basic: model.Basic{Id: 13, State: true}, FieldCode: "order_no"}},
	}
	tableList := marshalResponseFields(t, tableListResponse(table))
	tableDetail := marshalResponseFields(t, tableDetailResponse(table))
	assertMissingResponseFields(t, tableList, "sql", "table_fields", "table_relations", "table_indexes")
	assertPresentResponseFields(t, tableDetail, "sql", "table_fields", "table_relations", "table_indexes")
}

func TestHighRiskResponseKeepsFrontendBusinessFields(t *testing.T) {
	menu := menuListResponse(model.SysMenu{
		Basic:     model.Basic{Id: 21, State: true},
		Name:      "system_menu",
		Title:     "菜单管理",
		TableCode: "sys_menu",
		Children: []model.SysMenu{{
			Basic: model.Basic{Id: 22, State: true},
			Name:  "system_role",
		}},
		MenuButtons: []model.SysMenuButton{{
			Basic:        model.Basic{Id: 23, State: true},
			Code:         "system_menu_update",
			Path:         "/admin/menu/:id",
			Method:       "PUT",
			ParamsSchema: "{}",
		}},
	})
	fields := marshalResponseFields(t, menu)
	assertPresentResponseFields(t, fields, "id", "state", "name", "title", "table_code", "children", "menu_buttons")

	buttons, ok := fields["menu_buttons"].([]any)
	if !ok || len(buttons) != 1 {
		t.Fatalf("expected one menu button, got %#v", fields["menu_buttons"])
	}
	button, ok := buttons[0].(map[string]any)
	if !ok {
		t.Fatalf("expected object menu button, got %#v", buttons[0])
	}
	assertPresentResponseFields(t, button, "api_path", "http_method", "params_schema")
}

func marshalResponseFields(t *testing.T, value any) map[string]any {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return fields
}

func assertForbiddenResponseFields(t *testing.T, fields map[string]any, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, exists := fields[name]; exists {
			t.Errorf("response must not contain %q", name)
		}
	}
}

func assertMissingResponseFields(t *testing.T, fields map[string]any, names ...string) {
	t.Helper()
	assertForbiddenResponseFields(t, fields, names...)
}

func assertPresentResponseFields(t *testing.T, fields map[string]any, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, exists := fields[name]; !exists {
			t.Errorf("response must contain %q", name)
		}
	}
}
