package main

import (
	"backend/model"
	"fmt"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

func TestOrganizationFoundationSeedIsIdempotentAndReadOnly(t *testing.T) {
	db := migrateTestDB(t)
	if err := autoMigrateCoreSchema(db); err != nil {
		t.Fatalf("migrate core schema: %v", err)
	}
	sf := newMigrationTestSnowflake(t)
	if err := seedDicts(db, sf); err != nil {
		t.Fatalf("seed platform dictionaries: %v", err)
	}
	if err := seedOrganizationFoundation(db, sf); err != nil {
		t.Fatalf("seed organization foundation: %v", err)
	}

	firstCounts := organizationSeedCountSnapshot(t, db)
	firstKeys := organizationSeedKeySnapshot(t, db)
	customizeOrganizationSeedUserFields(t, db)

	if err := seedOrganizationFoundation(db, sf); err != nil {
		t.Fatalf("seed organization foundation twice: %v", err)
	}
	secondCounts := organizationSeedCountSnapshot(t, db)
	secondKeys := organizationSeedKeySnapshot(t, db)

	if !reflect.DeepEqual(firstCounts, secondCounts) {
		t.Fatalf("organization seed counts changed: first=%#v second=%#v", firstCounts, secondCounts)
	}
	if !reflect.DeepEqual(firstKeys, secondKeys) {
		t.Fatalf("organization seed stable keys changed: first=%#v second=%#v", firstKeys, secondKeys)
	}

	assertOrganizationSeedCatalog(t, db)
	assertOrganizationMetadataReadOnly(t, db)
	assertOrganizationPermissions(t, db)
	assertOrganizationSeedUserFieldsPreserved(t, db)
}

func organizationSeedCountSnapshot(t *testing.T, db *gorm.DB) map[string]int64 {
	t.Helper()
	return map[string]int64{
		"dict": countWhere(t, db, &model.SysDict{}, "dict_code LIKE ?", "org_%"),
		"dict_item": countJoinedRows(
			t,
			db,
			"sys_dict_item",
			"JOIN sys_dict ON sys_dict.id = sys_dict_item.dict_id",
			"sys_dict.dict_code LIKE ?",
			"org_%",
		),
		"table": countWhere(t, db, &model.SysTable{}, "table_code IN ?", organizationTableCodes()),
		"field": countJoinedRows(
			t,
			db,
			"sys_table_field",
			"JOIN sys_table ON sys_table.id = sys_table_field.table_id",
			"sys_table.table_code IN ?",
			organizationTableCodes(),
		),
		"menu": countWhere(
			t,
			db,
			&model.SysMenu{},
			"name = ? OR name LIKE ?",
			organizationRootMenuName,
			"organization_%",
		),
		"button": countJoinedRows(
			t,
			db,
			"sys_menu_button",
			"JOIN sys_menu ON sys_menu.id = sys_menu_button.menu_id",
			"sys_menu.name = ? OR sys_menu.name LIKE ?",
			organizationRootMenuName,
			"organization_%",
		),
		"role_menu": countJoinedRows(
			t,
			db,
			"sys_role_menu",
			"JOIN sys_menu ON sys_menu.id = sys_role_menu.menu_id",
			"sys_menu.name = ? OR sys_menu.name LIKE ?",
			organizationRootMenuName,
			"organization_%",
		),
		"role_button": countJoinedRows(
			t,
			db,
			"sys_role_menu_button",
			"JOIN sys_menu ON sys_menu.id = sys_role_menu_button.menu_id",
			"sys_menu.name = ? OR sys_menu.name LIKE ?",
			organizationRootMenuName,
			"organization_%",
		),
		"casbin": countWhere(t, db, &model.CasbinRule{}, "v0 = ? AND v1 LIKE ?", "super_admin", "/admin/org/%"),
	}
}

func organizationSeedKeySnapshot(t *testing.T, db *gorm.DB) map[string]map[string]int64 {
	t.Helper()
	return map[string]map[string]int64{
		"dict": keyIDSnapshot(
			t,
			db,
			"SELECT dict_code AS seed_key, id FROM sys_dict WHERE dict_code LIKE 'org_%'",
		),
		"dict_item": keyIDSnapshot(
			t,
			db,
			"SELECT sys_dict.dict_code || ':' || sys_dict_item.item_code AS seed_key, sys_dict_item.id "+
				"FROM sys_dict_item JOIN sys_dict ON sys_dict.id = sys_dict_item.dict_id "+
				"WHERE sys_dict.dict_code LIKE 'org_%'",
		),
		"table": keyIDSnapshot(
			t,
			db,
			"SELECT table_code AS seed_key, id FROM sys_table WHERE table_code LIKE 'org_%'",
		),
		"field": keyIDSnapshot(
			t,
			db,
			"SELECT sys_table.table_code || ':' || sys_table_field.field_code AS seed_key, sys_table_field.id "+
				"FROM sys_table_field JOIN sys_table ON sys_table.id = sys_table_field.table_id "+
				"WHERE sys_table.table_code LIKE 'org_%'",
		),
		"menu": keyIDSnapshot(
			t,
			db,
			"SELECT name AS seed_key, id FROM sys_menu "+
				"WHERE name = 'organization' OR name LIKE 'organization_%'",
		),
		"button": keyIDSnapshot(
			t,
			db,
			"SELECT sys_menu.name || ':' || sys_menu_button.code AS seed_key, sys_menu_button.id "+
				"FROM sys_menu_button JOIN sys_menu ON sys_menu.id = sys_menu_button.menu_id "+
				"WHERE sys_menu.name = 'organization' OR sys_menu.name LIKE 'organization_%'",
		),
	}
}

func organizationTableCodes() []string {
	seeds := organizationTableSeeds()
	codes := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		codes = append(codes, seed.code)
	}
	return codes
}

func countWhere(t *testing.T, db *gorm.DB, value interface{}, query string, args ...interface{}) int64 {
	t.Helper()
	var count int64
	if err := db.Model(value).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", value, err)
	}
	return count
}

func countJoinedRows(t *testing.T, db *gorm.DB, table, join, query string, args ...interface{}) int64 {
	t.Helper()
	var count int64
	if err := db.Table(table).Joins(join).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count joined %s: %v", table, err)
	}
	return count
}

func assertOrganizationSeedCatalog(t *testing.T, db *gorm.DB) {
	t.Helper()
	if got := countWhere(t, db, &model.SysDict{}, "dict_code LIKE ?", "org_%"); got != int64(len(organizationDictionarySeeds())) {
		t.Fatalf("organization dictionary count = %d, want %d", got, len(organizationDictionarySeeds()))
	}
	if got := countWhere(t, db, &model.SysTable{}, "table_code IN ?", organizationTableCodes()); got != 9 {
		t.Fatalf("organization table metadata count = %d, want 9", got)
	}
	if got := countWhere(t, db, &model.SysMenu{}, "pid <> 0 AND name LIKE ?", "organization_%"); got != 6 {
		t.Fatalf("organization functional menu count = %d, want 6", got)
	}

	for _, code := range organizationTableCodes() {
		var table model.SysTable
		if err := db.Where("table_code = ?", code).First(&table).Error; err != nil {
			t.Fatalf("query %s metadata: %v", code, err)
		}
		columns, err := db.Migrator().ColumnTypes(code)
		if err != nil {
			t.Fatalf("query %s columns: %v", code, err)
		}
		var fieldCount int64
		if err := db.Model(&model.SysTableField{}).Where("table_id = ?", table.Id).Count(&fieldCount).Error; err != nil {
			t.Fatalf("count %s fields: %v", code, err)
		}
		if fieldCount != int64(len(columns)) {
			t.Fatalf("%s metadata fields = %d, physical columns = %d", code, fieldCount, len(columns))
		}
	}

	assertNoDuplicateGroups(t, db, "sys_dict", []string{"dict_code"})
	assertNoDuplicateGroups(t, db, "sys_dict_item", []string{"dict_id", "item_code"})
	assertNoDuplicateGroups(t, db, "sys_table", []string{"table_code"})
	assertNoDuplicateGroups(t, db, "sys_table_field", []string{"table_id", "field_code"})
	assertNoDuplicateGroups(t, db, "sys_menu", []string{"name"})
	assertNoDuplicateGroups(t, db, "sys_menu_button", []string{"menu_id", "code"})
	assertNoDuplicateGroups(t, db, "sys_role_menu", []string{"role_id", "menu_id"})
	assertNoDuplicateGroups(t, db, "sys_role_menu_button", []string{"role_id", "menu_id", "button_id"})
	assertNoDuplicateGroups(t, db, "casbin_rule", []string{"ptype", "v0", "v1", "v2"})
}

func assertOrganizationMetadataReadOnly(t *testing.T, db *gorm.DB) {
	t.Helper()
	var fields []model.SysTableField
	if err := db.Table("sys_table_field").
		Select("sys_table_field.*").
		Joins("JOIN sys_table ON sys_table.id = sys_table_field.table_id").
		Where("sys_table.table_code IN ?", organizationTableCodes()).
		Find(&fields).Error; err != nil {
		t.Fatalf("query organization metadata fields: %v", err)
	}
	for _, field := range fields {
		if field.IsInsertShow || field.IsUpdateShow {
			t.Fatalf("organization field %d:%s unexpectedly editable", field.TableId, field.FieldCode)
		}
	}

	hiddenFields := map[string][]string{
		"org_legal_entity":   {"source_id", "source_version", "source_updated_at", "last_sync_at", "source_status", "source_deleted", "sync_status"},
		"org_unit":           {"source_id", "source_version", "source_updated_at", "last_sync_at", "source_status", "source_deleted", "sync_status"},
		"org_structure":      {"source_id", "source_version", "last_sync_at", "sync_status"},
		"org_structure_node": {"source_id", "source_parent_id", "path", "level", "source_deleted", "sync_status"},
		"org_position":       {"source_id", "source_version", "last_sync_at", "source_deleted", "sync_status"},
		"org_employee":       {"source_id", "source_version", "source_updated_at", "last_sync_at", "source_deleted", "sync_status", "mobile", "email"},
		"org_assignment":     {"source_id", "source_version", "source_deleted", "sync_status"},
		"org_sync_record":    {"source_id", "dependency_key", "error_message"},
	}
	for tableCode, fieldCodes := range hiddenFields {
		for _, fieldCode := range fieldCodes {
			field := organizationMetadataField(t, db, tableCode, fieldCode)
			if field.IsListShow || field.IsQuickSearch || field.IsAdvancedSearch {
				t.Fatalf("%s.%s unexpectedly exposed in list/query metadata", tableCode, fieldCode)
			}
		}
	}

	userID := organizationMetadataField(t, db, "org_employee", "user_id")
	if userID.FieldName != "绑定账号" || !userID.IsListShow || !userID.IsAdvancedSearch {
		t.Fatalf("org_employee.user_id metadata does not preserve binding semantics: %#v", userID)
	}
}

func organizationMetadataField(t *testing.T, db *gorm.DB, tableCode, fieldCode string) model.SysTableField {
	t.Helper()
	var field model.SysTableField
	if err := db.Table("sys_table_field").
		Select("sys_table_field.*").
		Joins("JOIN sys_table ON sys_table.id = sys_table_field.table_id").
		Where("sys_table.table_code = ? AND sys_table_field.field_code = ?", tableCode, fieldCode).
		First(&field).Error; err != nil {
		t.Fatalf("query metadata field %s.%s: %v", tableCode, fieldCode, err)
	}
	return field
}

func assertOrganizationPermissions(t *testing.T, db *gorm.DB) {
	t.Helper()
	allowedActions := map[string]struct{}{
		"query":       {},
		"detail":      {},
		"refresh":     {},
		"bind_user":   {},
		"unbind_user": {},
		"view_sync":   {},
		"retry":       {},
		"view_error":  {},
	}
	var buttons []model.SysMenuButton
	if err := db.Table("sys_menu_button").
		Select("sys_menu_button.*").
		Joins("JOIN sys_menu ON sys_menu.id = sys_menu_button.menu_id").
		Where("sys_menu.name LIKE ?", "organization_%").
		Find(&buttons).Error; err != nil {
		t.Fatalf("query organization buttons: %v", err)
	}
	if len(buttons) == 0 {
		t.Fatal("organization buttons were not seeded")
	}
	actionCounts := make(map[string]int, len(allowedActions))
	for _, button := range buttons {
		if _, ok := allowedActions[button.EventAction]; !ok {
			t.Fatalf("organization button %s uses forbidden action %q", button.Code, button.EventAction)
		}
		actionCounts[button.EventAction]++
	}
	for action := range allowedActions {
		if actionCounts[action] == 0 {
			t.Fatalf("organization permission action %s was not seeded", action)
		}
	}
	for _, forbidden := range []string{"create", "update", "delete"} {
		if got := countJoinedRows(
			t,
			db,
			"sys_menu_button",
			"JOIN sys_menu ON sys_menu.id = sys_menu_button.menu_id",
			"sys_menu.name LIKE ? AND sys_menu_button.event_action = ?",
			"organization_%",
			forbidden,
		); got != 0 {
			t.Fatalf("organization seed contains %d forbidden %s buttons", got, forbidden)
		}
	}

	var role model.SysRole
	if err := db.Where("name = ?", "super_admin").First(&role).Error; err != nil {
		t.Fatalf("query super_admin role: %v", err)
	}
	menuCount := countWhere(
		t,
		db,
		&model.SysMenu{},
		"name = ? OR name LIKE ?",
		organizationRootMenuName,
		"organization_%",
	)
	roleMenuCount := countJoinedRows(
		t,
		db,
		"sys_role_menu",
		"JOIN sys_menu ON sys_menu.id = sys_role_menu.menu_id",
		"sys_role_menu.role_id = ? AND (sys_menu.name = ? OR sys_menu.name LIKE ?)",
		role.Id,
		organizationRootMenuName,
		"organization_%",
	)
	if roleMenuCount != menuCount {
		t.Fatalf("super_admin organization menus = %d, want %d", roleMenuCount, menuCount)
	}
	roleButtonCount := countJoinedRows(
		t,
		db,
		"sys_role_menu_button",
		"JOIN sys_menu ON sys_menu.id = sys_role_menu_button.menu_id",
		"sys_role_menu_button.role_id = ? AND sys_menu.name LIKE ?",
		role.Id,
		"organization_%",
	)
	if roleButtonCount != int64(len(buttons)) {
		t.Fatalf("super_admin organization buttons = %d, want %d", roleButtonCount, len(buttons))
	}
	if got := countWhere(t, db, &model.CasbinRule{}, "v0 = ? AND v1 LIKE ?", role.Name, "/admin/org/%"); got == 0 {
		t.Fatal("super_admin organization Casbin policies were not seeded")
	}
	for _, permission := range []struct {
		path   string
		method string
		action string
	}{
		{path: "/admin/org/structure/query", method: "POST", action: "query"},
		{path: "/admin/org/structure/options", method: "POST", action: "query"},
		{path: "/admin/org/structure/:id", method: "GET", action: "detail"},
		{path: "/admin/org/unit/query", method: "POST", action: "query"},
		{path: "/admin/org/unit/options", method: "POST", action: "query"},
		{path: "/admin/org/unit/tree", method: "POST", action: "query"},
		{path: "/admin/org/unit/:id", method: "GET", action: "detail"},
		{path: "/admin/org/assignment/query", method: "POST", action: "query"},
		{path: "/admin/org/assignment/:id", method: "GET", action: "detail"},
		{path: "/admin/org/employee/:id/assignments/summary", method: "GET", action: "query"},
		{path: "/admin/org/employee/:id/bind-user", method: "POST", action: "bind_user"},
		{path: "/admin/org/employee/:id/unbind-user", method: "POST", action: "unbind_user"},
	} {
		if got := countWhere(
			t,
			db,
			&model.SysMenuButton{},
			"path = ? AND method = ? AND event_action = ?",
			permission.path,
			permission.method,
			permission.action,
		); got != 1 {
			t.Fatalf(
				"organization permission %s %s count = %d, want 1",
				permission.method,
				permission.path,
				got,
			)
		}
		if got := countWhere(
			t,
			db,
			&model.CasbinRule{},
			"v0 = ? AND v1 = ? AND v2 = ?",
			role.Name,
			permission.path,
			permission.method,
		); got != 1 {
			t.Fatalf(
				"organization Casbin policy %s %s count = %d, want 1",
				permission.method,
				permission.path,
				got,
			)
		}
	}
}

func customizeOrganizationSeedUserFields(t *testing.T, db *gorm.DB) {
	t.Helper()
	var dict model.SysDict
	if err := db.Where("dict_code = ?", "org_object_status").First(&dict).Error; err != nil {
		t.Fatalf("query organization status dict: %v", err)
	}
	if err := db.Model(&model.SysDict{}).Where("id = ?", dict.Id).Update("dict_name", "用户维护组织状态").Error; err != nil {
		t.Fatalf("customize organization dict name: %v", err)
	}
	if err := db.Model(&model.SysDictItem{}).
		Where("dict_id = ? AND item_code = ?", dict.Id, "org_object_status_enabled").
		Update("item_name", "用户维护启用").Error; err != nil {
		t.Fatalf("customize organization dict item: %v", err)
	}

	field := organizationMetadataField(t, db, "org_employee", "name")
	tag := "organization-display-name"
	if err := db.Model(&model.SysTableField{}).Where("id = ?", field.Id).Update("tag", tag).Error; err != nil {
		t.Fatalf("customize organization field tag: %v", err)
	}
}

func assertOrganizationSeedUserFieldsPreserved(t *testing.T, db *gorm.DB) {
	t.Helper()
	var dict model.SysDict
	if err := db.Where("dict_code = ?", "org_object_status").First(&dict).Error; err != nil {
		t.Fatalf("query organization status dict: %v", err)
	}
	if dict.DictName != "用户维护组织状态" {
		t.Fatalf("organization dict name overwritten: %q", dict.DictName)
	}
	var item model.SysDictItem
	if err := db.Where("dict_id = ? AND item_code = ?", dict.Id, "org_object_status_enabled").First(&item).Error; err != nil {
		t.Fatalf("query organization status item: %v", err)
	}
	if item.ItemName != "用户维护启用" {
		t.Fatalf("organization dict item name overwritten: %q", item.ItemName)
	}
	field := organizationMetadataField(t, db, "org_employee", "name")
	if field.Tag == nil || *field.Tag != "organization-display-name" {
		t.Fatalf("organization field extension tag overwritten: %v", field.Tag)
	}
}

func TestOrganizationFoundationSeedUsesStableBusinessKeys(t *testing.T) {
	for _, seed := range organizationDictionarySeeds() {
		if seed.code == "" {
			t.Fatal("organization dictionary has empty code")
		}
		itemCodes := make(map[string]struct{}, len(seed.items))
		for _, item := range seed.items {
			if item.code == "" || item.value == "" {
				t.Fatalf("dictionary %s contains unstable item: %#v", seed.code, item)
			}
			if _, exists := itemCodes[item.code]; exists {
				t.Fatalf("dictionary %s contains duplicate item code %s", seed.code, item.code)
			}
			itemCodes[item.code] = struct{}{}
		}
	}
	for _, seed := range organizationTableSeeds() {
		if seed.code == "" || seed.name == "" {
			t.Fatalf("organization table seed is incomplete: %s", fmt.Sprintf("%#v", seed))
		}
	}
}
