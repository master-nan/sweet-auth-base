package controller

import (
	"backend/dto/request"
	"backend/enum"
	apperrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"testing"
)

func TestValidateDataScopeWriteValue(t *testing.T) {
	table := model.SysTable{
		TableCode: "smoke_table",
		TableFields: []model.SysTableField{
			{FieldCode: "scope_id", FieldType: enum.IntFieldType},
			{FieldCode: "name", FieldType: enum.VarcharFieldType},
		},
	}
	scope := &request.DataScope{Conditions: []request.DataScopeCondition{
		{DimensionCode: "scope", Field: "scope_id", MatchType: "in", Values: []string{"1", "3"}},
	}}

	tests := []struct {
		name         string
		data         map[string]interface{}
		requireField bool
		wantErr      bool
	}{
		{name: "allowed integer", data: map[string]interface{}{"scope_id": 1}, requireField: true},
		{name: "allowed string", data: map[string]interface{}{"scope_id": "3"}, requireField: true},
		{name: "missing optional field", data: map[string]interface{}{"name": "updated"}, requireField: false},
		{name: "missing required field", data: map[string]interface{}{"name": "created"}, requireField: true, wantErr: true},
		{name: "out of scope", data: map[string]interface{}{"scope_id": 2}, requireField: false, wantErr: true},
		{name: "invalid typed value", data: map[string]interface{}{"scope_id": "abc"}, requireField: false, wantErr: true},
		{name: "fractional numeric value", data: map[string]interface{}{"scope_id": 1.2}, requireField: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDataScopeWriteValues(table, scope, tt.data, tt.requireField)
			if tt.wantErr {
				if err != apperrors.ErrPermissionDenied {
					t.Fatalf("expected permission denied, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected allowed write value, got %v", err)
			}
		})
	}
}

func TestValidateDataScopeWriteValueRejectsMissingScopeField(t *testing.T) {
	table := model.SysTable{
		TableCode:   "smoke_table",
		TableFields: []model.SysTableField{{FieldCode: "name", FieldType: enum.VarcharFieldType}},
	}
	scope := &request.DataScope{Conditions: []request.DataScopeCondition{
		{DimensionCode: "scope", Field: "scope_id", MatchType: "in", Values: []string{"1"}},
	}}

	if err := validateDataScopeWriteValues(table, scope, map[string]interface{}{"name": "updated"}, false); err != apperrors.ErrPermissionDenied {
		t.Fatalf("expected missing table scope field to be denied, got %v", err)
	}
}

func TestMenuAllowsTableCode(t *testing.T) {
	tests := []struct {
		name      string
		menu      model.SysMenu
		tableCode string
		want      bool
	}{
		{
			name:      "menu uses explicit bound table code",
			menu:      model.SysMenu{PageType: enum.MenuPageTypeLowCode, TableCode: "sys_user"},
			tableCode: "sys_user",
			want:      true,
		},
		{
			name:      "menu name no longer grants table access",
			menu:      model.SysMenu{Name: "sys_menu"},
			tableCode: "sys_menu",
			want:      false,
		},
		{
			name:      "system menu page uses explicit table binding",
			menu:      model.SysMenu{Name: "system_menu", PageType: enum.MenuPageTypeFixed, TableCode: "sys_menu"},
			tableCode: "sys_menu",
			want:      true,
		},
		{
			name:      "system role page without binding is rejected",
			menu:      model.SysMenu{Name: "system_role"},
			tableCode: "sys_role",
			want:      false,
		},
		{
			name:      "audit page uses explicit table binding",
			menu:      model.SysMenu{Name: "system_audit", PageType: enum.MenuPageTypeFixed, TableCode: "access_log"},
			tableCode: "access_log",
			want:      true,
		},
		{
			name:      "audit page without bound option is not special cased",
			menu:      model.SysMenu{Name: "system_audit"},
			tableCode: "access_log",
			want:      false,
		},
		{
			name:      "option no longer grants master detail access",
			menu:      model.SysMenu{Name: "develop_dictionary", Option: "sys_dict,sys_dict_item"},
			tableCode: "sys_dict_item",
			want:      false,
		},
		{
			name:      "generic generalization shell cannot open arbitrary table",
			menu:      model.SysMenu{Name: "develop_generalization"},
			tableCode: "access_log",
			want:      false,
		},
		{
			name:      "wrong explicit option is rejected",
			menu:      model.SysMenu{Name: "sys_menu", Option: "sys_role"},
			tableCode: "sys_menu",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.MenuAllowsTableCode(tt.menu, tt.tableCode); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestAllowQueryWithoutPublishedMenuBlocksProtectedTables(t *testing.T) {
	tests := []struct {
		tableCode string
		want      bool
	}{
		{tableCode: "sys_user", want: false},
		{tableCode: "casbin_rule", want: false},
		{tableCode: "access_log", want: false},
		{tableCode: "demo_customer", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.tableCode, func(t *testing.T) {
			if got := allowQueryWithoutPublishedMenu(tt.tableCode); got != tt.want {
				t.Fatalf("allowQueryWithoutPublishedMenu(%q) = %v, want %v", tt.tableCode, got, tt.want)
			}
		})
	}
}

func TestMenuButtonAllowsReadActionUsesExactEventAction(t *testing.T) {
	tests := []struct {
		name   string
		button model.SysMenuButton
		target string
		want   bool
	}{
		{
			name:   "query",
			button: model.SysMenuButton{Basic: model.Basic{State: true}, MenuId: 10, EventAction: "query"},
			target: "query",
			want:   true,
		},
		{
			name:   "detail",
			button: model.SysMenuButton{Basic: model.Basic{State: true}, MenuId: 10, EventAction: "detail"},
			target: "detail",
			want:   true,
		},
		{
			name:   "old alias rejected",
			button: model.SysMenuButton{Basic: model.Basic{State: true}, MenuId: 10, EventAction: "openDetail"},
			target: "detail",
			want:   false,
		},
		{
			name:   "code is not action",
			button: model.SysMenuButton{Basic: model.Basic{State: true}, MenuId: 10, EventAction: "system_menu_detail"},
			target: "detail",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.MenuButtonAllowsReadAction(tt.button, 10, tt.target); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
