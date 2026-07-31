package main

import (
	"backend/model"
	"fmt"
	"testing"

	"gorm.io/gorm"
)

func TestDataPermissionConfigPermissionSeedPublishesUniqueRoutes(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(
		&model.SysMenuButton{},
		&model.SysRoleMenuButton{},
		&model.CasbinRule{},
	); err != nil {
		t.Fatalf("migrate data permission config seed schema: %v", err)
	}

	const (
		menuID   = 305
		roleID   = 1
		roleName = "super_admin"
	)
	sf := newMigrationTestSnowflake(t)
	for run := 1; run <= 2; run++ {
		if err := seedDataPermissionMenuButtons(db, sf, roleID, roleName, menuID); err != nil {
			t.Fatalf("seed data permission config permissions run %d: %v", run, err)
		}
	}

	type expectedPermission struct {
		id       int
		path     string
		method   string
		action   string
		isButton bool
	}
	expected := map[string]expectedPermission{
		"system_data_permission_config_dimension_query": {
			633, "/admin/data-permission/config/dimension/query", "POST", "query_dimension", false,
		},
		"system_data_permission_config_ownership_query": {
			634, "/admin/data-permission/config/ownership/query", "POST", "query_ownership", false,
		},
		"system_data_permission_config_resource_create": {
			635, "/admin/data-permission/config/resource", "POST", "create_resource", true,
		},
		"system_data_permission_config_resource_update": {
			636, "/admin/data-permission/config/resource/:id", "PUT", "update_resource", true,
		},
		"system_data_permission_config_resource_operation_query": {
			637, "/admin/data-permission/config/resource/:id/operations", "GET", "query_operation", false,
		},
		"system_data_permission_config_resource_operation_replace": {
			638, "/admin/data-permission/config/resource/:id/operations", "PUT", "configure_operations", true,
		},
		"system_data_permission_config_resource_permission": {
			639, "/admin/data-permission/config/resource/:id/permission", "PUT", "toggle_permission", true,
		},
		"system_data_permission_config_ownership_create": {
			640, "/admin/data-permission/config/ownership", "POST", "create_ownership", true,
		},
		"system_data_permission_config_ownership_update": {
			641, "/admin/data-permission/config/ownership/:id", "PUT", "update_ownership", true,
		},
		"system_data_permission_config_policy_create": {
			642, "/admin/data-permission/config/policy", "POST", "create_policy", true,
		},
		"system_data_permission_config_policy_update": {
			643, "/admin/data-permission/config/policy/:id", "PUT", "update_policy", true,
		},
		"system_data_permission_config_policy_rule_replace": {
			644, "/admin/data-permission/config/policy/:id/rules", "PUT", "configure_rules", true,
		},
		"system_data_permission_config_policy_state": {
			645, "/admin/data-permission/config/policy/:id/state", "PUT", "toggle_policy", true,
		},
		"system_data_permission_config_grant_create": {
			646, "/admin/data-permission/config/grant", "POST", "create_grant", true,
		},
		"system_data_permission_config_grant_state": {
			647, "/admin/data-permission/config/grant/:id/state", "PUT", "toggle_grant", true,
		},
		"system_data_permission_config_resource_preflight": {
			630, "/admin/data-permission/config/preflight/resource/:id", "GET", "preflight_resource", true,
		},
		"system_data_permission_config_policy_preflight": {
			631, "/admin/data-permission/config/preflight/policy/:id", "GET", "preflight_policy", true,
		},
		"system_data_permission_config_grant_preflight": {
			632, "/admin/data-permission/config/preflight/grant/:id", "GET", "preflight_grant", true,
		},
	}
	for code, want := range expected {
		var button model.SysMenuButton
		if err := db.Where("menu_id = ? AND code = ?", menuID, code).First(&button).Error; err != nil {
			t.Fatalf("query data permission config button %s: %v", code, err)
		}
		if button.Id != want.id || button.Path != want.path || button.Method != want.method ||
			button.EventAction != want.action || button.IsButton != want.isButton {
			t.Errorf("button %s = %#v, want %#v", code, button, want)
		}
		assertPermissionSeedCount(
			t,
			db,
			&model.SysRoleMenuButton{},
			"role_id = ? AND menu_id = ? AND button_id = ?",
			[]interface{}{roleID, menuID, button.Id},
		)
		assertPermissionSeedCount(
			t,
			db,
			&model.CasbinRule{},
			"ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?",
			[]interface{}{"p", roleName, want.path, want.method},
		)
	}

	var existing model.SysMenuButton
	if err := db.Where(
		"menu_id = ? AND code = ?",
		menuID,
		"system_data_permission_config_resource_query",
	).First(&existing).Error; err != nil {
		t.Fatalf("query existing resource permission: %v", err)
	}
	if existing.Id != 621 || existing.IsButton ||
		existing.Path != "/admin/data-permission/config/resource/query" ||
		existing.Method != "POST" {
		t.Fatalf("existing resource query permission changed: %#v", existing)
	}

	var buttons []model.SysMenuButton
	if err := db.Where(
		"menu_id = ? AND path LIKE ?",
		menuID,
		"/admin/data-permission/config/%",
	).Find(&buttons).Error; err != nil {
		t.Fatalf("query data permission config permissions: %v", err)
	}
	routeOwners := make(map[string]string, len(buttons))
	for _, button := range buttons {
		key := fmt.Sprintf("%s %s", button.Method, button.Path)
		if owner, exists := routeOwners[key]; exists {
			t.Errorf("duplicate route ownership %s: %s and %s", key, owner, button.Code)
		}
		routeOwners[key] = button.Code
	}
}

func TestDictionaryCodePermissionSeedIsIdempotentAndRoleAssignable(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(
		&model.SysMenuButton{},
		&model.SysRoleMenuButton{},
		&model.CasbinRule{},
	); err != nil {
		t.Fatalf("migrate permission seed schema: %v", err)
	}

	sf := newMigrationTestSnowflake(t)
	for run := 1; run <= 2; run++ {
		if err := seedDictionaryMenuButtons(db, sf, 1, "super_admin", 304); err != nil {
			t.Fatalf("seed dictionary permissions run %d: %v", run, err)
		}
	}

	var button model.SysMenuButton
	if err := db.Where(
		"menu_id = ? AND code = ?",
		304,
		"develop_dictionary_code_query",
	).First(&button).Error; err != nil {
		t.Fatalf("query dictionary code permission button: %v", err)
	}
	if button.Path != "/admin/dict/code/:code" || button.Method != "GET" {
		t.Fatalf("unexpected dictionary code permission route: path=%q method=%q", button.Path, button.Method)
	}
	if button.IsButton {
		t.Fatal("dictionary code permission must remain API-only")
	}

	assertPermissionSeedCount(
		t,
		db,
		&model.SysMenuButton{},
		"menu_id = ? AND code = ?",
		[]interface{}{304, "develop_dictionary_code_query"},
	)
	assertPermissionSeedCount(
		t,
		db,
		&model.SysRoleMenuButton{},
		"role_id = ? AND menu_id = ? AND button_id = ?",
		[]interface{}{1, 304, button.Id},
	)
	assertPermissionSeedCount(
		t,
		db,
		&model.CasbinRule{},
		"ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?",
		[]interface{}{"p", "super_admin", "/admin/dict/code/:code", "GET"},
	)
}

func assertPermissionSeedCount(
	t *testing.T,
	db *gorm.DB,
	modelValue interface{},
	query string,
	args []interface{},
) {
	t.Helper()
	var count int64
	if err := db.Model(modelValue).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count permission seed %T: %v", modelValue, err)
	}
	if count != 1 {
		t.Fatalf("permission seed %T count=%d, want 1", modelValue, count)
	}
}
