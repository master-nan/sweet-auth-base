package main

import (
	"backend/config"
	"backend/enum"
	"backend/internal/queryscheme"
	"backend/model"
	"os"
	"regexp"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestFunctionalPermissionProjectionUsesButtonMetadataOnly(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(
		&model.SysRole{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.CasbinRule{},
	); err != nil {
		t.Fatalf("migrate permission projection schema: %v", err)
	}

	role := model.SysRole{Basic: model.Basic{Id: 1, State: true}, Name: "operator"}
	menu := model.SysMenu{Basic: model.Basic{Id: 10, State: true}, Name: "demo"}
	buttons := []model.SysMenuButton{
		{
			Basic:       model.Basic{Id: 100, State: true},
			MenuId:      menu.Id,
			Code:        "demo_query",
			Name:        "查询",
			Path:        "/admin/demo/query",
			Method:      "post",
			EventAction: "query",
		},
		{
			Basic:       model.Basic{Id: 101, State: true},
			MenuId:      menu.Id,
			Code:        "demo_query_duplicate",
			Name:        "查询复用",
			Path:        "/admin/demo/query",
			Method:      "POST",
			EventAction: "query",
		},
		{
			Basic:       model.Basic{Id: 102, State: true},
			MenuId:      menu.Id,
			Code:        "demo_detail",
			Name:        "详情",
			EventAction: "detail",
		},
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatalf("create menu: %v", err)
	}
	if err := db.Create(&model.SysRoleMenu{RoleId: role.Id, MenuId: menu.Id}).Error; err != nil {
		t.Fatalf("grant menu: %v", err)
	}
	if err := db.Create(&buttons).Error; err != nil {
		t.Fatalf("create buttons: %v", err)
	}
	for _, button := range buttons {
		if err := db.Create(&model.SysRoleMenuButton{RoleId: role.Id, MenuId: menu.Id, ButtonId: button.Id}).Error; err != nil {
			t.Fatalf("grant button %s: %v", button.Code, err)
		}
	}
	if err := db.Create(&model.CasbinRule{PType: "p", V0: role.Name, V1: "/admin/orphan", V2: "GET"}).Error; err != nil {
		t.Fatalf("create orphan policy: %v", err)
	}
	if err := db.Create(&model.CasbinRule{PType: "p", V0: "direct-user", V1: "/admin/user-route", V2: "GET"}).Error; err != nil {
		t.Fatalf("create non-role policy: %v", err)
	}
	if err := db.Create(&model.CasbinRule{PType: "p", V0: "deleted-role", V1: "/admin/stale-route", V2: "GET"}).Error; err != nil {
		t.Fatalf("create stale role policy: %v", err)
	}
	if err := db.Create(&model.CasbinRule{PType: "g", V0: "operator", V1: "parent-role"}).Error; err != nil {
		t.Fatalf("create grouping policy: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return rebuildFunctionalPermissionPolicies(tx)
	}); err != nil {
		t.Fatalf("rebuild permission projection: %v", err)
	}

	assertPolicyCount(t, db, role.Name, "/admin/demo/query", "POST", 1)
	assertPolicyCount(t, db, role.Name, "/admin/generalization/detail/code/:code/:id", "GET", 1)
	assertPolicyCount(t, db, role.Name, "/admin/orphan", "GET", 0)
	assertPolicyCount(t, db, "direct-user", "/admin/user-route", "GET", 0)
	assertPolicyCount(t, db, "deleted-role", "/admin/stale-route", "GET", 0)
	var groupingCount int64
	if err := db.Model(&model.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ?", "g", role.Name, "parent-role").
		Count(&groupingCount).Error; err != nil {
		t.Fatalf("count grouping policy: %v", err)
	}
	if groupingCount != 1 {
		t.Fatalf("grouping policy count=%d, want 1", groupingCount)
	}
}

func TestCanonicalRuntimeContractMigratesSelectorsAndLowCodeButtons(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(
		&model.SysTableField{},
		&model.DataDimensionDefinition{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysRoleMenuButton{},
	); err != nil {
		t.Fatalf("migrate canonical runtime fixtures: %v", err)
	}
	legacyLinkage := `{"selector_type":"employee_select","selector":{"selector_type":"position_select"}}`
	field := model.SysTableField{Basic: model.Basic{Id: 101, State: true}, TableId: 1, FieldCode: "employee_id", FieldName: "员工", LinkageConfig: &legacyLinkage}
	selector := "legal_entity_select"
	dimension := model.DataDimensionDefinition{Basic: model.Basic{Id: 102, State: true}, Code: "legal_entity", Name: "法人", Category: model.DataDimensionCategoryOrganization, ValueType: model.DataDimensionValueTypeBigint, ProviderCode: "organization", SelectorType: &selector}
	menus := []model.SysMenu{
		{Basic: model.Basic{Id: 201, State: true}, Name: "demo_low_code", PageType: enum.MenuPageTypeLowCode},
		{Basic: model.Basic{Id: 202, State: true}, Name: "system_user", PageType: enum.MenuPageTypeFixed},
	}
	buttons := []model.SysMenuButton{
		{Basic: model.Basic{Id: 301, State: true}, MenuId: 201, Name: "旧查询", Code: "system_demo_query"},
		{Basic: model.Basic{Id: 302, State: true}, MenuId: 201, Name: "新查询", Code: "demo_query"},
		{Basic: model.Basic{Id: 303, State: true}, MenuId: 202, Name: "用户查询", Code: "system_user_query"},
	}
	if err := db.Create(&field).Error; err != nil {
		t.Fatalf("seed selector field: %v", err)
	}
	if err := db.Create(&dimension).Error; err != nil {
		t.Fatalf("seed selector dimension: %v", err)
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&buttons).Error; err != nil {
		t.Fatalf("seed buttons: %v", err)
	}
	if err := db.Exec("INSERT INTO sys_role_menu_button (role_id, button_id) VALUES (?, ?)", 1, 301).Error; err != nil {
		t.Fatalf("seed legacy button grant: %v", err)
	}

	if err := migrateCanonicalRuntimeContract(db); err != nil {
		t.Fatalf("first canonical runtime migration: %v", err)
	}
	if err := migrateCanonicalRuntimeContract(db); err != nil {
		t.Fatalf("second canonical runtime migration: %v", err)
	}
	if err := db.First(&field, field.Id).Error; err != nil {
		t.Fatalf("read canonical field: %v", err)
	}
	if field.LinkageConfig == nil || *field.LinkageConfig != `{"selector":{"selector_type":"position"},"selector_type":"employee"}` {
		t.Fatalf("canonical linkage config=%v", field.LinkageConfig)
	}
	if err := db.First(&dimension, dimension.Id).Error; err != nil {
		t.Fatalf("read canonical dimension: %v", err)
	}
	if dimension.SelectorType == nil || *dimension.SelectorType != "legal_entity" {
		t.Fatalf("canonical dimension selector=%v", dimension.SelectorType)
	}
	var remaining []model.SysMenuButton
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("read canonical buttons: %v", err)
	}
	if len(remaining) != 2 || remaining[0].Id != 302 || remaining[1].Id != 303 {
		t.Fatalf("remaining buttons=%+v", remaining)
	}
	var grantCount int64
	if err := db.Model(&model.SysRoleMenuButton{}).Where("button_id = ?", 301).Count(&grantCount).Error; err != nil || grantCount != 0 {
		t.Fatalf("legacy button grants=%d err=%v", grantCount, err)
	}
}

func TestFunctionalPermissionProjectionSeedIsIdempotentAndCoversStrictRoutes(t *testing.T) {
	db := migrateTestDB(t)
	if err := autoMigrateCoreSchema(db); err != nil {
		t.Fatalf("migrate core schema: %v", err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("migrate integration configuration schema: %v", err)
	}
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate data permission schema: %v", err)
	}
	cfg := &config.Server{}
	cfg.Conf.Salt = "permission-projection-test-salt"
	sf := newMigrationTestSnowflake(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_BOOTSTRAP_ADMIN_PASSWORD", "Permission-Projection-2026!")

	for run := 1; run <= 2; run++ {
		if err := seedAllData(db, cfg, sf); err != nil {
			t.Fatalf("seed platform data run %d: %v", run, err)
		}
	}

	assertNoDuplicateGroups(t, db, "sys_menu_button", []string{"menu_id", "code"})
	assertNoDuplicateGroups(t, db, "sys_menu_button_template", []string{"scene", "code_suffix"})
	assertNoDuplicateGroups(t, db, "sys_role_menu_button", []string{"role_id", "menu_id", "button_id"})
	assertNoDuplicateGroups(t, db, "casbin_rule", []string{"ptype", "v0", "v1", "v2"})
	var scopeCount int64
	if err := db.Model(&model.SysMenu{}).Where("query_scope_code IS NOT NULL AND state = ?", true).Count(&scopeCount).Error; err != nil {
		t.Fatalf("count query scopes: %v", err)
	}
	if scopeCount != int64(len(queryscheme.FixedScopeDeclarations())) {
		t.Fatalf("query scope count=%d, want %d", scopeCount, len(queryscheme.FixedScopeDeclarations()))
	}
	var sharedCapabilityCount int64
	if err := db.Model(&model.SysMenuButton{}).
		Where("event_action = ? AND state = ?", queryscheme.SharedManageCapability, true).
		Count(&sharedCapabilityCount).Error; err != nil {
		t.Fatalf("count query scheme shared capability: %v", err)
	}
	if sharedCapabilityCount != 4 {
		t.Fatalf("shared capability policy projections=%d, want 4", sharedCapabilityCount)
	}

	for _, spec := range filePermissionSpecs {
		assertActivePermissionSource(t, db, spec.path, spec.method)
	}
	assertActivePermissionSource(t, db, "/admin/role/menu/:id", "GET")
	assertActivePermissionSource(t, db, "/admin/role/menu/buttons/:roleId/:menuId", "GET")

	for _, spec := range retiredOrganizationPermissions {
		var count int64
		if err := db.Model(&model.SysMenuButton{}).Where("code = ? AND state = ?", spec.code, true).Count(&count).Error; err != nil {
			t.Fatalf("count retired organization permission %s: %v", spec.code, err)
		}
		if count != 0 {
			t.Fatalf("retired organization permission %s remains active", spec.code)
		}
	}

	allRoutes := adminRoutesFromSource(t)
	for route := range allRoutes {
		if strictCoverageException(route) {
			continue
		}
		var count int64
		if err := db.Model(&model.SysMenuButton{}).
			Where("path = ? AND method = ? AND state = ? AND is_disabled = ?", route.path, route.method, true, false).
			Count(&count).Error; err != nil {
			t.Fatalf("count permission source for %s %s: %v", route.method, route.path, err)
		}
		if count == 0 {
			t.Errorf("strict admin route has no active permission source: %s %s", route.method, route.path)
		}
	}

	var permissionButtons []model.SysMenuButton
	if err := db.Where("path <> '' AND method <> '' AND state = ? AND is_disabled = ?", true, false).
		Find(&permissionButtons).Error; err != nil {
		t.Fatalf("query active permission metadata: %v", err)
	}
	for _, button := range permissionButtons {
		route := auditedRoute{path: button.Path, method: strings.ToUpper(button.Method)}
		if _, exists := allRoutes[route]; !exists {
			t.Errorf("active permission metadata has no router owner: %s %s (%s)", route.method, route.path, button.Code)
		}
	}

}

func TestFunctionalPermissionProjectionRejectsAmbiguousRoleSubjects(t *testing.T) {
	tests := []struct {
		name  string
		roles []model.SysRole
	}{
		{
			name: "duplicate",
			roles: []model.SysRole{
				{Basic: model.Basic{Id: 1}, Name: "operator"},
				{Basic: model.Basic{Id: 2}, Name: "operator"},
			},
		},
		{
			name: "surrounding whitespace",
			roles: []model.SysRole{
				{Basic: model.Basic{Id: 1}, Name: " operator "},
			},
		},
		{
			name: "casbin subject too long",
			roles: []model.SysRole{
				{Basic: model.Basic{Id: 1}, Name: strings.Repeat("角", 101)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateFunctionalPermissionSubjects(tt.roles); err == nil {
				t.Fatal("expected invalid functional permission subject")
			}
		})
	}
}

func TestFunctionalPermissionProjectionIgnoresMismatchedAndDeletedMenuGrants(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(
		&model.SysRole{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.CasbinRule{},
	); err != nil {
		t.Fatalf("migrate permission projection schema: %v", err)
	}

	role := model.SysRole{Basic: model.Basic{Id: 1, State: true}, Name: "operator"}
	activeMenu := model.SysMenu{Basic: model.Basic{Id: 10, State: true}, Name: "active"}
	deletedMenu := model.SysMenu{Basic: model.Basic{Id: 11, State: true}, Name: "deleted"}
	activeButton := model.SysMenuButton{
		Basic:  model.Basic{Id: 100, State: true},
		MenuId: activeMenu.Id,
		Code:   "active_query",
		Path:   "/admin/active",
		Method: "GET",
	}
	deletedButton := model.SysMenuButton{
		Basic:  model.Basic{Id: 101, State: true},
		MenuId: deletedMenu.Id,
		Code:   "deleted_query",
		Path:   "/admin/deleted",
		Method: "GET",
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create(&activeMenu).Error; err != nil {
		t.Fatalf("create active menu: %v", err)
	}
	if err := db.Create(&deletedMenu).Error; err != nil {
		t.Fatalf("create deleted menu: %v", err)
	}
	if err := db.Delete(&deletedMenu).Error; err != nil {
		t.Fatalf("soft delete menu: %v", err)
	}
	if err := db.Create(&[]model.SysMenuButton{activeButton, deletedButton}).Error; err != nil {
		t.Fatalf("create buttons: %v", err)
	}
	if err := db.Create(&[]model.SysRoleMenu{
		{RoleId: role.Id, MenuId: activeMenu.Id},
		{RoleId: role.Id, MenuId: deletedMenu.Id},
	}).Error; err != nil {
		t.Fatalf("grant menus: %v", err)
	}
	if err := db.Create(&[]model.SysRoleMenuButton{
		{RoleId: role.Id, MenuId: deletedMenu.Id, ButtonId: activeButton.Id},
		{RoleId: role.Id, MenuId: deletedMenu.Id, ButtonId: deletedButton.Id},
	}).Error; err != nil {
		t.Fatalf("grant mismatched and deleted menu buttons: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return rebuildFunctionalPermissionPolicies(tx)
	}); err != nil {
		t.Fatalf("rebuild permission projection: %v", err)
	}

	assertPolicyCount(t, db, role.Name, activeButton.Path, activeButton.Method, 0)
	assertPolicyCount(t, db, role.Name, deletedButton.Path, deletedButton.Method, 0)
}

func TestRetireUnimplementedOrganizationPermissionsUsesMenuAndCodeOwnership(t *testing.T) {
	db := migrateTestDB(t)
	if err := autoMigrateCoreSchema(db); err != nil {
		t.Fatalf("migrate retired permission schema: %v", err)
	}

	menus := []model.SysMenu{
		{Basic: model.Basic{Id: 10, State: true}, Name: "organization_structure"},
		{Basic: model.Basic{Id: 11, State: true}, Name: "organization_sync_batch"},
		{Basic: model.Basic{Id: 12, State: true}, Name: "organization_sync_error"},
		{Basic: model.Basic{Id: 13, State: true}, Name: "unrelated"},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("create menus: %v", err)
	}
	target := model.SysMenuButton{
		Basic:  model.Basic{Id: 100, State: true},
		MenuId: 10,
		Code:   "organization_unit_ancestors",
		Path:   "/admin/org/unit/:id/ancestors",
		Method: "GET",
	}
	unrelated := target
	unrelated.Id = 101
	unrelated.MenuId = 13
	if err := db.Create(&[]model.SysMenuButton{target, unrelated}).Error; err != nil {
		t.Fatalf("create buttons: %v", err)
	}
	if err := db.Create(&[]model.SysRoleMenuButton{
		{RoleId: 1, MenuId: target.MenuId, ButtonId: target.Id},
		{RoleId: 1, MenuId: unrelated.MenuId, ButtonId: unrelated.Id},
	}).Error; err != nil {
		t.Fatalf("create role button grants: %v", err)
	}

	if err := retireUnimplementedOrganizationPermissions(db); err != nil {
		t.Fatalf("retire organization permissions: %v", err)
	}

	var targetState bool
	if err := db.Unscoped().Model(&model.SysMenuButton{}).
		Select("state").
		Where("id = ?", target.Id).
		Scan(&targetState).Error; err != nil {
		t.Fatalf("query target state: %v", err)
	}
	if targetState {
		t.Fatal("target organization permission remains active")
	}
	var unrelatedState bool
	if err := db.Model(&model.SysMenuButton{}).
		Select("state").
		Where("id = ?", unrelated.Id).
		Scan(&unrelatedState).Error; err != nil {
		t.Fatalf("query unrelated state: %v", err)
	}
	if !unrelatedState {
		t.Fatal("unrelated permission with same code was retired")
	}
	var unrelatedGrantCount int64
	if err := db.Model(&model.SysRoleMenuButton{}).
		Where("role_id = ? AND menu_id = ? AND button_id = ?", 1, unrelated.MenuId, unrelated.Id).
		Count(&unrelatedGrantCount).Error; err != nil {
		t.Fatalf("count unrelated role grant: %v", err)
	}
	if unrelatedGrantCount != 1 {
		t.Fatalf("unrelated role grant count=%d, want 1", unrelatedGrantCount)
	}
}

func TestLowCodeFilePermissionsRequireExplicitRoleGrant(t *testing.T) {
	db := migrateTestDB(t)
	if err := autoMigrateCoreSchema(db); err != nil {
		t.Fatalf("migrate core schema: %v", err)
	}
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate data permission schema: %v", err)
	}
	cfg := &config.Server{}
	cfg.Conf.Salt = "low-code-file-permission-test-salt"
	sf := newMigrationTestSnowflake(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_BOOTSTRAP_ADMIN_PASSWORD", "LowCode-File-Permission-2026!")
	if err := seedAllData(db, cfg, sf); err != nil {
		t.Fatalf("seed platform data: %v", err)
	}

	role := model.SysRole{Basic: model.Basic{Id: 99, State: true}, Name: "lowcode_operator"}
	menu := model.SysMenu{
		Basic:     model.Basic{Id: 999, State: true},
		Name:      "lowcode_demo",
		Path:      "demo",
		PageType:  enum.MenuPageTypeLowCode,
		TableCode: "demo_table",
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create low-code role: %v", err)
	}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatalf("create low-code menu: %v", err)
	}
	if err := db.Create(&model.SysRoleMenu{RoleId: role.Id, MenuId: menu.Id}).Error; err != nil {
		t.Fatalf("grant low-code menu: %v", err)
	}

	var superAdmin model.SysRole
	if err := db.Where("name = ?", "super_admin").First(&superAdmin).Error; err != nil {
		t.Fatalf("query super admin: %v", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := backfillLowCodeFilePermissions(tx, sf, superAdmin); err != nil {
			return err
		}
		return rebuildFunctionalPermissionPolicies(tx)
	}); err != nil {
		t.Fatalf("backfill low-code file permissions: %v", err)
	}

	for _, spec := range filePermissionSpecs {
		var button model.SysMenuButton
		if err := db.Where("menu_id = ? AND code = ?", menu.Id, menu.TableCode+spec.suffix).First(&button).Error; err != nil {
			t.Fatalf("query low-code file permission %s: %v", spec.suffix, err)
		}
		if button.IsButton {
			t.Fatalf("low-code file permission %s must be API-only", spec.suffix)
		}
		var count int64
		if err := db.Model(&model.SysRoleMenuButton{}).
			Where("role_id = ? AND menu_id = ? AND button_id = ?", role.Id, menu.Id, button.Id).
			Count(&count).Error; err != nil {
			t.Fatalf("count low-code role permission %s: %v", spec.suffix, err)
		}
		if count != 0 {
			t.Fatalf("low-code file permission %s was granted without explicit role assignment", spec.suffix)
		}
		if err := db.Create(&model.SysRoleMenuButton{RoleId: role.Id, MenuId: menu.Id, ButtonId: button.Id}).Error; err != nil {
			t.Fatalf("grant low-code file permission %s: %v", spec.suffix, err)
		}
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return rebuildFunctionalPermissionPolicies(tx)
	}); err != nil {
		t.Fatalf("rebuild explicitly assigned low-code file policies: %v", err)
	}
	for _, spec := range filePermissionSpecs {
		assertPolicyCount(t, db, role.Name, spec.path, spec.method, 1)
	}
}

func assertPolicyCount(t *testing.T, db *gorm.DB, subject, path, method string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", subject, path, method).
		Count(&count).Error; err != nil {
		t.Fatalf("count policy %s %s %s: %v", subject, method, path, err)
	}
	if count != want {
		t.Fatalf("policy %s %s %s count=%d, want %d", subject, method, path, count, want)
	}
}

func assertActivePermissionSource(t *testing.T, db *gorm.DB, path, method string) {
	t.Helper()
	var count int64
	if err := db.Model(&model.SysMenuButton{}).
		Where("path = ? AND method = ? AND state = ? AND is_disabled = ?", path, strings.ToUpper(method), true, false).
		Count(&count).Error; err != nil {
		t.Fatalf("count active permission source %s %s: %v", method, path, err)
	}
	if count == 0 {
		t.Fatalf("active permission source missing for %s %s", method, path)
	}
}

type auditedRoute struct {
	path   string
	method string
}

func adminRoutesFromSource(t *testing.T) map[auditedRoute]struct{} {
	t.Helper()
	content, err := os.ReadFile("../initialize/router.go")
	if err != nil {
		t.Fatalf("read router source: %v", err)
	}
	pattern := regexp.MustCompile(`adminGroup\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`)
	routes := make(map[auditedRoute]struct{})
	for _, match := range pattern.FindAllStringSubmatch(string(content), -1) {
		route := auditedRoute{path: "/admin" + match[2], method: match[1]}
		routes[route] = struct{}{}
	}
	return routes
}

func strictCoverageException(route auditedRoute) bool {
	exceptions := map[auditedRoute]struct{}{
		{path: "/admin/logout", method: "POST"}:                                   {},
		{path: "/admin/user/me", method: "GET"}:                                   {},
		{path: "/admin/menu/my", method: "GET"}:                                   {},
		{path: "/admin/runtime/dict/:code", method: "GET"}:                        {},
		{path: "/admin/runtime/table/:code", method: "GET"}:                       {},
		{path: "/admin/runtime/relation-fields/:fieldId/options", method: "POST"}: {},
		{path: "/admin/runtime/query-scopes/:scope", method: "GET"}:               {},
		{path: "/admin/runtime/query-schemes/available", method: "GET"}:           {},
		{path: "/admin/runtime/query-schemes/:id/resolve", method: "POST"}:        {},
		{path: "/admin/query-schemes/query", method: "POST"}:                      {},
		{path: "/admin/query-schemes/:id", method: "GET"}:                         {},
		{path: "/admin/query-schemes/personal", method: "POST"}:                   {},
		{path: "/admin/query-schemes/personal/:id", method: "PUT"}:                {},
		{path: "/admin/query-schemes/personal/:id", method: "DELETE"}:             {},
		{path: "/admin/query-schemes/personal/:id/default", method: "PUT"}:        {},
		{path: "/admin/query-schemes/:id/copy-to-personal", method: "POST"}:       {},
		{path: "/admin/user/password", method: "POST"}:                            {},
		{path: "/admin/generalization/query/code/:code", method: "POST"}:          {},
		{path: "/admin/generalization/detail/code/:code/:id", method: "GET"}:      {},
		{path: "/admin/generalization/create", method: "POST"}:                    {},
		{path: "/admin/generalization/update", method: "PUT"}:                     {},
		{path: "/admin/generalization/delete", method: "DELETE"}:                  {},
	}
	_, ok := exceptions[route]
	return ok
}
