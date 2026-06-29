package service

import (
	"backend/dto/request"
	"backend/model"
	"reflect"
	"testing"
)

func TestUniquePositiveInts(t *testing.T) {
	got := uniquePositiveInts([]int{0, 2, 2, -1, 3, 2, 4})
	want := []int{2, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFilterAssignableRoleButtons(t *testing.T) {
	buttons := []model.SysMenuButton{
		{Basic: model.Basic{Id: 1, State: true}, MenuId: 10, Code: "query", Path: "/ok", Method: "POST"},
		{Basic: model.Basic{Id: 2, State: true}, MenuId: 11, Code: "foreign", Path: "/foreign", Method: "POST"},
		{Basic: model.Basic{Id: 3, State: false}, MenuId: 10, Code: "stopped", Path: "/stopped", Method: "POST"},
		{Basic: model.Basic{Id: 4, State: true}, MenuId: 10, Code: "disabled", Path: "/disabled", Method: "POST", IsDisabled: true},
	}

	got := filterAssignableRoleButtons(buttons, map[int]bool{10: true})
	if len(got) != 1 || got[0].Id != 1 {
		t.Fatalf("expected only selected active button, got %#v", got)
	}

	got = filterAssignableRoleButtons(buttons, nil)
	if len(got) != 0 {
		t.Fatalf("expected no buttons without selected menus, got %#v", got)
	}
}

func TestAssignedRoleDataScopeRecordsNilPermissionsPreservesLegacyBehavior(t *testing.T) {
	svc := &SysRoleService{}

	records, err := svc.assignedRoleDataScopeRecords(1, map[int]bool{10: true}, nil)
	if err != nil {
		t.Fatalf("expected nil permissions to be accepted: %v", err)
	}
	if records != nil {
		t.Fatalf("expected nil records for legacy request, got %#v", records)
	}
}

func TestAssignedRoleDataScopeRecordsRejectsUnselectedMenu(t *testing.T) {
	dataPermissionService, _ := newDataPermissionServiceForTest(t)
	svc := &SysRoleService{dataPermissionService: dataPermissionService}

	_, err := svc.assignedRoleDataScopeRecords(1, map[int]bool{10: true}, []request.RoleDataPermissionItemReq{
		{MenuId: 20, TableCode: "demo_order", DimensionCode: "tenant", Strategy: "specified", ScopeValues: []string{"1"}},
	})
	if err == nil {
		t.Fatal("expected data permission outside selected menus to be rejected")
	}
}

func TestAssignedRoleDataScopeRecordsBuildsSelectedMenuScopes(t *testing.T) {
	dataPermissionService, _ := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, dataPermissionService.db, true)
	svc := &SysRoleService{dataPermissionService: dataPermissionService}

	records, err := svc.assignedRoleDataScopeRecords(1, map[int]bool{10: true}, []request.RoleDataPermissionItemReq{
		{MenuId: 10, TableCode: "demo_order", DimensionCode: "tenant", Strategy: "specified", ScopeValues: []string{"1"}},
	})
	if err != nil {
		t.Fatalf("expected selected menu data scope to be accepted: %v", err)
	}
	if len(records) != 1 || records[0].RoleId != 1 || records[0].MenuId != 10 || records[0].DimensionCode != "tenant" {
		t.Fatalf("unexpected role data scope records: %#v", records)
	}
}

func TestAssignedRoleDataScopeRecordsAcceptsUserDimensionStrategy(t *testing.T) {
	dataPermissionService, _ := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, dataPermissionService.db, true)
	svc := &SysRoleService{dataPermissionService: dataPermissionService}

	records, err := svc.assignedRoleDataScopeRecords(1, map[int]bool{10: true}, []request.RoleDataPermissionItemReq{
		{MenuId: 10, TableCode: "demo_order", DimensionCode: "tenant", Strategy: "user_dimension"},
	})
	if err != nil {
		t.Fatalf("expected user dimension strategy to be accepted: %v", err)
	}
	if len(records) != 1 || records[0].Strategy != "user_dimension" || records[0].ScopeValues != "[]" {
		t.Fatalf("unexpected user dimension role scope: %#v", records)
	}
}

func TestButtonAPIPolicies(t *testing.T) {
	tests := []struct {
		name   string
		button model.SysMenuButton
		want   []buttonAPIPolicy
	}{
		{
			name:   "explicit api button",
			button: model.SysMenuButton{Path: "/admin/log/access/:id", Method: "get", EventAction: "detail"},
			want:   []buttonAPIPolicy{{Path: "/admin/log/access/:id", Method: "GET"}},
		},
		{
			name:   "generic detail page button without api",
			button: model.SysMenuButton{EventAction: "detail"},
			want:   []buttonAPIPolicy{{Path: "/admin/generalization/detail/code/:code/:id", Method: "GET"}},
		},
		{
			name:   "old detail alias is not a permission action",
			button: model.SysMenuButton{EventAction: "openDetail"},
			want:   []buttonAPIPolicy{},
		},
		{
			name:   "regular navigation button without api",
			button: model.SysMenuButton{EventAction: "navigate"},
			want:   []buttonAPIPolicy{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buttonAPIPolicies(tt.button)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}
