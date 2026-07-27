package main

import (
	"backend/model"
	"testing"

	"gorm.io/gorm"
)

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
