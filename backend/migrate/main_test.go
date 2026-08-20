package main

import (
	"backend/config"
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/utils"
	"backend/model"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestBootstrapAdminPasswordDefaultsForLocalUse(t *testing.T) {
	t.Setenv("APP_ENV", "docker")
	t.Setenv("APP_REQUIRE_SECURE_CONFIG", "")
	t.Setenv("APP_BOOTSTRAP_ADMIN_PASSWORD", "")

	password, err := bootstrapAdminPassword()
	if err != nil {
		t.Fatalf("expected local bootstrap password default to be allowed: %v", err)
	}
	if password != "admin123" {
		t.Fatalf("unexpected default bootstrap password: %q", password)
	}
}

func TestBootstrapAdminPasswordRejectsDefaultWhenSecureConfigRequired(t *testing.T) {
	t.Setenv("APP_ENV", "docker")
	t.Setenv("APP_REQUIRE_SECURE_CONFIG", "true")
	t.Setenv("APP_BOOTSTRAP_ADMIN_PASSWORD", "")

	if _, err := bootstrapAdminPassword(); err == nil {
		t.Fatal("expected default bootstrap password to be rejected when secure config is required")
	}
}

func TestBootstrapAdminPasswordAllowsStrongValueWhenSecureConfigRequired(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_BOOTSTRAP_ADMIN_PASSWORD", "LocalInit-2026-Strong!")

	password, err := bootstrapAdminPassword()
	if err != nil {
		t.Fatalf("expected strong bootstrap password to be allowed: %v", err)
	}
	if password != "LocalInit-2026-Strong!" {
		t.Fatalf("unexpected bootstrap password: %q", password)
	}
}

func TestBootstrapAdminPasswordRejectsDefaultInPro(t *testing.T) {
	t.Setenv("APP_ENV", "pro")
	t.Setenv("APP_BOOTSTRAP_ADMIN_PASSWORD", "admin123")

	if _, err := bootstrapAdminPassword(); err == nil {
		t.Fatal("expected pro environment to reject default bootstrap password")
	}
}

func TestInsecureBootstrapAdminPassword(t *testing.T) {
	insecure := []string{"", "admin123", "my-admin123", "password", "short"}
	for _, password := range insecure {
		if !insecureBootstrapAdminPassword(password) {
			t.Fatalf("expected %q to be insecure", password)
		}
	}

	if insecureBootstrapAdminPassword("Sufficient-2026-Cred!") {
		t.Fatal("expected strong credential to be allowed")
	}
}

func TestMigrationStepsRegistersPlatformBaselineOrder(t *testing.T) {
	got := migrationStepNames(migrationSteps())
	want := []string{
		"auto_migrate_core_schema",
		"metadata_value_contract",
		"backfill_sys_table_index_field_sequence",
		"query_scheme_schema",
		"integration_configuration_schema",
		"integration_runtime_schema",
		"integration_sync_schema",
		"organization_sync_integrity_schema",
		"data_permission_domain_schema",
		"remove_legacy_data_permission_schema",
		"ensure_sys_menu_option_text",
		"backfill_sys_menu_page_binding",
		"organization_database_comments",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("migration steps = %#v, want %#v", got, want)
	}
}

func TestPlatformSeedStepsRegistersMetadataAndPermissionBaselineOrder(t *testing.T) {
	got := seedStepNames(platformSeedSteps())
	want := []string{
		"sys_configure",
		"sys_dict",
		"application",
		"admin_user",
		"sys_menu_role_button",
		"lowcode_button_templates",
		"menu_button_defaults_repair",
		"sys_table_and_field_metadata",
		"integration_configuration_foundation",
		"query_scheme_foundation",
		"data_permission_dictionary_and_metadata",
		"sys_table_relation_metadata",
		"functional_permission_projection",
		"rebuildable_cache_flush",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("platform seed steps = %#v, want %#v", got, want)
	}
}

func migrationStepNames(steps []migrationStep) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.name)
	}
	return names
}

func seedStepNames(steps []seedStep) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.name)
	}
	return names
}

func TestPlatformSeedStepsAreIdempotentForFoundationData(t *testing.T) {
	db := migrateTestDB(t)
	if err := autoMigrateCoreSchema(db); err != nil {
		t.Fatalf("migrate core schema: %v", err)
	}
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate data permission schema: %v", err)
	}
	cfg := &config.Server{}
	cfg.Conf.Salt = "seed-idempotency-test-salt"
	sf := newMigrationTestSnowflake(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_BOOTSTRAP_ADMIN_PASSWORD", "Seed-Idempotency-2026!")

	if err := seedAllData(db, cfg, sf); err != nil {
		t.Fatalf("seed platform data: %v", err)
	}
	firstCounts := platformSeedCountSnapshot(t, db)
	firstKeys := platformSeedKeySnapshot(t, db)

	if err := db.Model(&model.SysConfigure{}).
		Where("id = ?", 1).
		Updates(map[string]interface{}{
			"system_name": "用户维护平台名称",
			"system_logo": "custom-logo",
		}).Error; err != nil {
		t.Fatalf("customize sys configure: %v", err)
	}
	if err := db.Model(&model.SysDict{}).
		Where("dict_code = ?", "whether").
		Update("dict_name", "用户维护是否字典").Error; err != nil {
		t.Fatalf("customize dict name: %v", err)
	}
	var whether model.SysDict
	if err := db.Where("dict_code = ?", "whether").First(&whether).Error; err != nil {
		t.Fatalf("query whether dict: %v", err)
	}
	if err := db.Model(&model.SysDictItem{}).
		Where("dict_id = ? AND item_code = ?", whether.Id, "whether_yes").
		Update("item_name", "用户维护是").Error; err != nil {
		t.Fatalf("customize dict item name: %v", err)
	}

	if err := seedAllData(db, cfg, sf); err != nil {
		t.Fatalf("seed platform data twice: %v", err)
	}
	secondCounts := platformSeedCountSnapshot(t, db)
	secondKeys := platformSeedKeySnapshot(t, db)

	if !reflect.DeepEqual(secondCounts, firstCounts) {
		t.Fatalf("platform seed counts changed after second run: first=%#v second=%#v", firstCounts, secondCounts)
	}
	if !reflect.DeepEqual(secondKeys, firstKeys) {
		t.Fatalf("platform seed stable keys changed after second run: first=%#v second=%#v", firstKeys, secondKeys)
	}

	assertNoDuplicateGroups(t, db, "sys_dict", []string{"dict_code"})
	assertNoDuplicateGroups(t, db, "sys_dict_item", []string{"dict_id", "item_code"})
	assertNoDuplicateGroups(t, db, "sys_dict_item", []string{"dict_id", "item_value"})
	assertNoDuplicateGroups(t, db, "sys_table", []string{"table_code"})
	assertNoDuplicateGroups(t, db, "sys_table_field", []string{"table_id", "field_code"})
	assertNoDuplicateGroups(t, db, "sys_menu", []string{"name"})
	assertNoDuplicateGroups(t, db, "sys_menu_button", []string{"menu_id", "code"})
	assertNoDuplicateGroups(t, db, "sys_menu_button_template", []string{"scene", "code_suffix"})
	assertNoDuplicateGroups(t, db, "sys_data_dimension_definition", []string{"code"})
	assertNoDuplicateGroups(t, db, "sys_role_menu", []string{"role_id", "menu_id"})
	assertNoDuplicateGroups(t, db, "sys_role_menu_button", []string{"role_id", "menu_id", "button_id"})
	assertNoDuplicateGroups(t, db, "casbin_rule", []string{"ptype", "v0", "v1", "v2"})
	assertMenuTitle(t, db, "system_data_permission", "router.system.dataPermission")
	assertMenuTitle(t, db, "report_v2_workbench", "router.report.workbench")
	assertSeedUserMaintainedFieldsPreserved(t, db, whether.Id)
}

func assertMenuTitle(t *testing.T, db *gorm.DB, name string, expected string) {
	t.Helper()
	var menu model.SysMenu
	if err := db.Where("name = ?", name).First(&menu).Error; err != nil {
		t.Fatalf("query menu %s: %v", name, err)
	}
	if menu.Title != expected {
		t.Fatalf("menu %s title = %q, want %q", name, menu.Title, expected)
	}
}

func platformSeedCountSnapshot(t *testing.T, db *gorm.DB) map[string]int64 {
	t.Helper()
	return map[string]int64{
		"sys_dict":                      countRows(t, db, &model.SysDict{}),
		"sys_dict_item":                 countRows(t, db, &model.SysDictItem{}),
		"sys_table":                     countRows(t, db, &model.SysTable{}),
		"sys_table_field":               countRows(t, db, &model.SysTableField{}),
		"sys_menu":                      countRows(t, db, &model.SysMenu{}),
		"sys_menu_button":               countRows(t, db, &model.SysMenuButton{}),
		"sys_menu_button_template":      countRows(t, db, &model.SysMenuButtonTemplate{}),
		"sys_data_dimension_definition": countRows(t, db, &model.DataDimensionDefinition{}),
		"sys_role_menu":                 countRows(t, db, &model.SysRoleMenu{}),
		"sys_role_menu_button":          countRows(t, db, &model.SysRoleMenuButton{}),
		"casbin_rule":                   countRows(t, db, &model.CasbinRule{}),
	}
}

func countRows(t *testing.T, db *gorm.DB, modelValue interface{}) int64 {
	t.Helper()
	var count int64
	if err := db.Model(modelValue).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", modelValue, err)
	}
	return count
}

func platformSeedKeySnapshot(t *testing.T, db *gorm.DB) map[string]map[string]int64 {
	t.Helper()
	return map[string]map[string]int64{
		"sys_dict":                      keyIDSnapshot(t, db, "SELECT dict_code AS seed_key, id FROM sys_dict"),
		"sys_dict_item":                 keyIDSnapshot(t, db, "SELECT CAST(dict_id AS TEXT) || ':' || item_code AS seed_key, id FROM sys_dict_item"),
		"sys_table":                     keyIDSnapshot(t, db, "SELECT table_code AS seed_key, id FROM sys_table"),
		"sys_table_field":               keyIDSnapshot(t, db, "SELECT CAST(table_id AS TEXT) || ':' || field_code AS seed_key, id FROM sys_table_field"),
		"sys_menu":                      keyIDSnapshot(t, db, "SELECT name AS seed_key, id FROM sys_menu"),
		"sys_menu_button":               keyIDSnapshot(t, db, "SELECT CAST(menu_id AS TEXT) || ':' || code AS seed_key, id FROM sys_menu_button"),
		"sys_menu_button_template":      keyIDSnapshot(t, db, "SELECT scene || ':' || code_suffix AS seed_key, id FROM sys_menu_button_template"),
		"sys_data_dimension_definition": keyIDSnapshot(t, db, "SELECT code AS seed_key, id FROM sys_data_dimension_definition"),
	}
}

func keyIDSnapshot(t *testing.T, db *gorm.DB, query string) map[string]int64 {
	t.Helper()
	type row struct {
		SeedKey string `gorm:"column:seed_key"`
		ID      int64  `gorm:"column:id"`
	}
	var rows []row
	if err := db.Raw(query).Scan(&rows).Error; err != nil {
		t.Fatalf("query key snapshot: %v", err)
	}
	snapshot := make(map[string]int64, len(rows))
	for _, item := range rows {
		snapshot[item.SeedKey] = item.ID
	}
	return snapshot
}

func assertNoDuplicateGroups(t *testing.T, db *gorm.DB, table string, columns []string) {
	t.Helper()
	groupBy := strings.Join(columns, ", ")
	query := fmt.Sprintf("SELECT COUNT(*) FROM (SELECT 1 FROM %s GROUP BY %s HAVING COUNT(*) > 1) duplicate_groups", table, groupBy)
	var count int64
	if err := db.Raw(query).Scan(&count).Error; err != nil {
		t.Fatalf("query duplicate groups for %s(%s): %v", table, groupBy, err)
	}
	if count != 0 {
		t.Fatalf("expected no duplicate groups for %s(%s), got %d", table, groupBy, count)
	}
}

func assertSeedUserMaintainedFieldsPreserved(t *testing.T, db *gorm.DB, whetherDictID int) {
	t.Helper()
	var sysConfig model.SysConfigure
	if err := db.First(&sysConfig, 1).Error; err != nil {
		t.Fatalf("query sys configure: %v", err)
	}
	if sysConfig.SystemName != "用户维护平台名称" || sysConfig.SystemLogo != "custom-logo" {
		t.Fatalf("seed overwrote user maintained sys_configure fields: system_name=%q system_logo=%q", sysConfig.SystemName, sysConfig.SystemLogo)
	}

	var dict model.SysDict
	if err := db.Where("dict_code = ?", "whether").First(&dict).Error; err != nil {
		t.Fatalf("query customized dict: %v", err)
	}
	if dict.DictName != "用户维护是否字典" {
		t.Fatalf("seed overwrote user maintained dict name: %q", dict.DictName)
	}

	var item model.SysDictItem
	if err := db.Where("dict_id = ? AND item_code = ?", whetherDictID, "whether_yes").First(&item).Error; err != nil {
		t.Fatalf("query customized dict item: %v", err)
	}
	if item.ItemName != "用户维护是" {
		t.Fatalf("seed overwrote user maintained dict item name: %q", item.ItemName)
	}
}

func TestSeedDictsCreatesSystemEnumDictionaries(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysDict{}, &model.SysDictItem{}); err != nil {
		t.Fatalf("migrate dicts: %v", err)
	}
	sf := newMigrationTestSnowflake(t)
	if err := seedDicts(db, sf); err != nil {
		t.Fatalf("seed dicts: %v", err)
	}

	var count int64
	if err := db.Model(&model.SysDict{}).
		Where("dict_code IN ?", []string{"whether", "sys_table_field_type", "sys_table_field_input_type", "sys_menu_button_position", "sys_master_detail_mode", "sys_form_open_mode", "sys_detail_open_mode", "http_method"}).
		Count(&count).Error; err != nil {
		t.Fatalf("count dicts: %v", err)
	}
	if count != 8 {
		t.Fatalf("expected system dictionaries, got %d", count)
	}

	if err := db.Model(&model.SysDictItem{}).Where("item_code = ? AND item_value = ?", "sys_table_field_type_datetime", "7").Count(&count).Error; err != nil {
		t.Fatalf("count field type item: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected datetime field type dict item, got %d", count)
	}

	if err := db.Model(&model.SysDictItem{}).Where("item_code = ? AND item_value = ?", "sys_menu_button_position_detail_top", "6").Count(&count).Error; err != nil {
		t.Fatalf("count detail top position item: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected detail top button position dict item, got %d", count)
	}

	if err := db.Model(&model.SysDictItem{}).Where("item_code = ? AND item_value = ?", "sys_master_detail_mode_stacked", string(enum.MasterDetailStacked)).Count(&count).Error; err != nil {
		t.Fatalf("count master detail mode item: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected stacked master detail mode dict item, got %d", count)
	}

	if err := db.Model(&model.SysDictItem{}).Where("item_code = ? AND item_value = ?", "sys_form_open_mode_page", string(enum.FormOpenPage)).Count(&count).Error; err != nil {
		t.Fatalf("count form open mode item: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected page form open mode dict item, got %d", count)
	}

	if err := db.Model(&model.SysDictItem{}).Where("item_code = ? AND item_value = ?", "sys_detail_open_mode_dialog", string(enum.DetailOpenDialog)).Count(&count).Error; err != nil {
		t.Fatalf("count detail open mode item: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected dialog detail open mode dict item, got %d", count)
	}

	if err := db.Model(&model.SysDictItem{}).Where("item_code = ? AND item_value = ?", "http_method_delete", "DELETE").Count(&count).Error; err != nil {
		t.Fatalf("count http method item: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected http method dict item, got %d", count)
	}

	if err := db.Model(&model.SysDict{}).
		Where("dict_code IN ?", []string{"report_status", "report_category", "report_source_type", "report_kind"}).
		Count(&count).Error; err != nil {
		t.Fatalf("count report dicts: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected report dictionaries, got %d", count)
	}

	for _, item := range []struct {
		code  string
		value string
	}{
		{"report_status_published", "published"},
		{"report_category_finance", "finance"},
		{"report_source_type_sql", "sql"},
		{"report_kind_layout", "layout"},
		{"sys_menu_button_event_action_publish_menu", string(enum.ButtonActionPublishMenu)},
		{"sys_menu_button_event_action_unpublish_menu", string(enum.ButtonActionUnpublishMenu)},
		{"sys_menu_button_event_action_version", string(enum.ButtonActionVersion)},
	} {
		if err := db.Model(&model.SysDictItem{}).Where("item_code = ? AND item_value = ?", item.code, item.value).Count(&count).Error; err != nil {
			t.Fatalf("count dict item %s: %v", item.code, err)
		}
		if count != 1 {
			t.Fatalf("expected dict item %s=%s, got %d", item.code, item.value, count)
		}
	}
}

func TestSeedAuditMenuButtonsRemovesViewRefreshCapability(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysMenuButton{}, &model.SysRoleMenuButton{}); err != nil {
		t.Fatalf("migrate audit buttons: %v", err)
	}
	legacy := model.SysMenuButton{Basic: model.Basic{Id: 493, State: true}, MenuId: 206, Name: "刷新详情", Code: "system_audit_detail_refresh", Position: enum.DetailTop, EventAction: "refresh", IsButton: true}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed legacy audit refresh: %v", err)
	}
	if err := db.Create(&model.SysRoleMenuButton{RoleId: 1, MenuId: 206, ButtonId: legacy.Id}).Error; err != nil {
		t.Fatalf("seed legacy audit refresh grant: %v", err)
	}
	sf := newMigrationTestSnowflake(t)
	if err := seedAuditMenuButtons(db, sf, 1, "", 206); err != nil {
		t.Fatalf("seed audit buttons: %v", err)
	}

	var count int64
	if err := db.Model(&model.SysMenuButton{}).Where("code = ?", "system_audit_detail_refresh").Count(&count).Error; err != nil {
		t.Fatalf("count retired audit refresh button: %v", err)
	}
	if count != 0 {
		t.Fatalf("audit detail refresh button count = %d, want 0", count)
	}
	if err := db.Model(&model.SysRoleMenuButton{}).Where("button_id = ?", legacy.Id).Count(&count).Error; err != nil {
		t.Fatalf("count retired audit refresh grant: %v", err)
	}
	if count != 0 {
		t.Fatalf("audit detail refresh grant count = %d, want 0", count)
	}
}

func TestRemoveLowCodeViewRefreshConfigurationIsIdempotent(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysMenu{}, &model.SysMenuButton{}, &model.SysRoleMenuButton{}, &model.SysMenuButtonTemplate{}); err != nil {
		t.Fatalf("migrate low-code view refresh fixtures: %v", err)
	}
	menu := model.SysMenu{Basic: model.Basic{Id: 20, State: true}, Name: "lowcode_demo", PageType: enum.MenuPageTypeLowCode, TableCode: "demo"}
	refresh := model.SysMenuButton{Basic: model.Basic{Id: 21, State: true}, MenuId: menu.Id, Name: "刷新", Code: "demo_refresh", EventAction: "refresh", IsButton: true}
	business := model.SysMenuButton{Basic: model.Basic{Id: 22, State: true}, MenuId: menu.Id, Name: "刷新缓存", Code: "demo_refresh_cache", EventAction: "refresh_cache", Path: "/admin/demo/cache", IsButton: true}
	template := model.SysMenuButtonTemplate{Basic: model.Basic{Id: 23, State: true}, Scene: lowCodeCrudButtonTemplateScene, Name: "刷新", CodeSuffix: "_refresh", EventAction: "refresh", IsButton: true}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]model.SysMenuButton{refresh, business}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}

	if err := removeLowCodeViewRefreshConfiguration(db); err != nil {
		t.Fatal(err)
	}
	if err := removeLowCodeViewRefreshConfiguration(db); err != nil {
		t.Fatal(err)
	}

	if got := countWhere(t, db, &model.SysMenuButton{}, "code = ?", refresh.Code); got != 0 {
		t.Fatalf("view refresh button count = %d, want 0", got)
	}
	if got := countWhere(t, db, &model.SysMenuButton{}, "code = ?", business.Code); got != 1 {
		t.Fatalf("business refresh button count = %d, want 1", got)
	}
	if got := countWhere(t, db, &model.SysMenuButtonTemplate{}, "code_suffix = ?", "_refresh"); got != 0 {
		t.Fatalf("view refresh template count = %d, want 0", got)
	}
}

func TestSeedMenuButtonPersistsAPIPermissionAsNonPageButton(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysMenuButton{}, &model.SysRoleMenuButton{}, &model.CasbinRule{}); err != nil {
		t.Fatalf("migrate menu button tables: %v", err)
	}

	sf := newMigrationTestSnowflake(t)
	button := apiPermissionWithAPI(10, 20, "列表查询", "system_user_query", enum.Top, "query", "search", "primary", 90, "/admin/user/query", "POST")
	if err := seedMenuButton(db, sf, 1, "super_admin", button); err != nil {
		t.Fatalf("seed api permission button: %v", err)
	}

	var got model.SysMenuButton
	if err := db.Where("code = ?", "system_user_query").First(&got).Error; err != nil {
		t.Fatalf("query seeded button: %v", err)
	}
	if got.IsButton || got.IsHidden {
		t.Fatalf("api permission should be non-page and visible in metadata, got is_button=%v is_hidden=%v", got.IsButton, got.IsHidden)
	}
	if got.DisplayMode != enum.ButtonDisplayAuto {
		t.Fatalf("api permission should default display_mode to auto, got %q", got.DisplayMode)
	}
}

func TestSeedSystemTableFieldRepairsGeneratedChineseName(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysTable{}, &model.SysTableField{}); err != nil {
		t.Fatalf("migrate system table metadata: %v", err)
	}
	table := model.SysTable{Basic: model.Basic{Id: 10, State: true}, TableName: "菜单按钮", TableCode: "sys_menu_button"}
	if err := db.Create(&table).Error; err != nil {
		t.Fatalf("create system table: %v", err)
	}
	existing := model.SysTableField{
		Basic:        model.Basic{Id: 11, State: true},
		TableId:      table.Id,
		FieldName:    "is button",
		FieldCode:    "is_button",
		FieldType:    enum.BooleanFieldType,
		InputType:    enum.BooleanInputType,
		IsListShow:   true,
		IsInsertShow: true,
		IsUpdateShow: true,
		Sequence:     1,
	}
	if err := db.Select("*").Create(&existing).Error; err != nil {
		t.Fatalf("create existing field: %v", err)
	}

	field := existing
	field.FieldName = systemFieldDisplayName(table.TableCode, field.FieldCode)
	if err := seedSystemTableField(db, newMigrationTestSnowflake(t), table, field); err != nil {
		t.Fatalf("repair system table field: %v", err)
	}

	var got model.SysTableField
	if err := db.First(&got, existing.Id).Error; err != nil {
		t.Fatalf("query repaired field: %v", err)
	}
	if got.FieldName != "是否页面按钮" {
		t.Fatalf("expected field name repaired, got %q", got.FieldName)
	}
}

func TestSeedSystemTableMetadataConfiguresReportDefinition(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysTable{}, &model.SysTableField{}, &model.SysTableIndex{}, &model.SysTableIndexField{}); err != nil {
		t.Fatalf("migrate metadata tables: %v", err)
	}
	if err := db.Table("report_definition").AutoMigrate(&model.ReportDefinition{}); err != nil {
		t.Fatalf("migrate report_definition: %v", err)
	}

	if err := seedSystemTableMetadata(db, newMigrationTestSnowflake(t)); err != nil {
		t.Fatalf("seed system table metadata: %v", err)
	}

	var table model.SysTable
	if err := db.Where("table_code = ?", "report_definition").First(&table).Error; err != nil {
		t.Fatalf("query report_definition sys table: %v", err)
	}

	fields := map[string]model.SysTableField{}
	var result []model.SysTableField
	if err := db.Where("table_id = ?", table.Id).Find(&result).Error; err != nil {
		t.Fatalf("query report_definition fields: %v", err)
	}
	for _, field := range result {
		fields[field.FieldCode] = field
	}

	assertReportField := func(code string, list, advanced, quick bool, dictCode string) {
		t.Helper()
		field, ok := fields[code]
		if !ok {
			t.Fatalf("expected report_definition field %s", code)
		}
		if field.IsListShow != list || field.IsAdvancedSearch != advanced || field.IsQuickSearch != quick {
			t.Fatalf("unexpected field flags for %s: list=%v advanced=%v quick=%v", code, field.IsListShow, field.IsAdvancedSearch, field.IsQuickSearch)
		}
		if dictCode == "" {
			return
		}
		if field.DictCode == nil || *field.DictCode != dictCode {
			t.Fatalf("expected field %s dict %s, got %+v", code, dictCode, field.DictCode)
		}
	}

	assertReportField("name", true, true, true, "")
	assertReportField("code", true, true, true, "")
	assertReportField("category", true, true, true, "report_category")
	assertReportField("status", true, true, false, "report_status")
	assertReportField("source_type", true, true, false, "report_source_type")
	assertReportField("permission_table_code", true, true, true, "")
	assertReportField("gmt_modify", true, true, false, "")
	assertReportField("query_config", false, false, false, "")
	assertReportField("layout_config", false, false, false, "")
}

func TestSeedReportV2WorkbenchMenuButtons(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysMenuButton{}, &model.SysRoleMenuButton{}, &model.CasbinRule{}); err != nil {
		t.Fatalf("migrate report workbench buttons: %v", err)
	}

	sf := newMigrationTestSnowflake(t)
	if err := seedReportV2WorkbenchMenuButtons(db, sf, 1, "super_admin", 904); err != nil {
		t.Fatalf("seed report workbench buttons: %v", err)
	}
	if err := seedReportV2WorkbenchMenuButtons(db, sf, 1, "super_admin", 904); err != nil {
		t.Fatalf("seed report workbench buttons twice: %v", err)
	}

	expected := map[string]string{
		"report_v2_workbench_query":          string(enum.ButtonActionQuery),
		"report_v2_workbench_create":         string(enum.ButtonActionCreate),
		"report_v2_workbench_refresh":        string(enum.ButtonActionRefresh),
		"report_v2_workbench_design":         string(enum.ButtonActionUpdate),
		"report_v2_workbench_run":            string(enum.ButtonActionRun),
		"report_v2_workbench_publish":        string(enum.ButtonActionPublish),
		"report_v2_workbench_publish_menu":   string(enum.ButtonActionPublishMenu),
		"report_v2_workbench_unpublish_menu": string(enum.ButtonActionUnpublishMenu),
		"report_v2_workbench_version":        string(enum.ButtonActionVersion),
		"report_v2_workbench_disable":        string(enum.ButtonActionDisable),
		"report_v2_workbench_delete":         string(enum.ButtonActionDelete),
	}

	for code, action := range expected {
		var button model.SysMenuButton
		if err := db.Where("menu_id = ? AND code = ?", 904, code).First(&button).Error; err != nil {
			t.Fatalf("query report workbench button %s: %v", code, err)
		}
		if button.EventAction != action {
			t.Fatalf("expected %s action %s, got %s", code, action, button.EventAction)
		}
		var count int64
		if err := db.Model(&model.SysRoleMenuButton{}).Where("role_id = ? AND menu_id = ? AND button_id = ?", 1, 904, button.Id).Count(&count).Error; err != nil {
			t.Fatalf("count report workbench role binding %s: %v", code, err)
		}
		if count != 1 {
			t.Fatalf("expected one role binding for %s, got %d", code, count)
		}
	}

	var count int64
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ? AND code = ?", 904, "report_v2_workbench_run").Count(&count).Error; err != nil {
		t.Fatalf("count report workbench run button: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected idempotent run button seed, got %d", count)
	}
	if err := db.Model(&model.CasbinRule{}).Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", "super_admin", "/admin/report/:id/run", "POST").Count(&count).Error; err != nil {
		t.Fatalf("count run casbin policy: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected run casbin policy, got %d", count)
	}
}

func TestMigrationCachePrefixesOnlyFlushRebuildableBusinessCaches(t *testing.T) {
	prefixes := migrationCachePrefixes()
	required := []string{
		cache.ApplicationCacheKey,
		cache.ConfigureCacheKey,
		cache.DictCacheKey,
		cache.GeneralizationCacheKey,
		cache.MenuButtonCacheKey,
		cache.MenuCacheKey,
		cache.RoleCacheKey,
		cache.RoleMenuButtonCacheKey,
		cache.RoleMenuCacheKey,
		cache.TableCacheKey,
		cache.TableFieldCacheKey,
		cache.UserCacheKey,
		cache.UserRoleCacheKey,
	}
	seen := make(map[string]bool, len(prefixes))
	for _, prefix := range prefixes {
		if seen[prefix] {
			t.Fatalf("duplicate migration cache prefix %q", prefix)
		}
		seen[prefix] = true
	}
	for _, prefix := range required {
		if !seen[prefix] {
			t.Fatalf("expected migration cache prefixes to include %q", prefix)
		}
	}
	for _, forbidden := range []string{
		cache.TokenBlackCacheKey,
		cache.RefreshTokenBlackCacheKey,
		cache.SendCodeCacheKey,
		cache.BlackUserCacheKey,
	} {
		if seen[forbidden] {
			t.Fatalf("migration cache flush must not clear security/session prefix %q", forbidden)
		}
	}
}

func TestSeedSystemTableRelations(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysTable{}, &model.SysTableRelation{}); err != nil {
		t.Fatalf("migrate system table relation metadata: %v", err)
	}
	tables := []model.SysTable{
		{Basic: model.Basic{Id: 10, State: true}, TableName: "字典", TableCode: "sys_dict"},
		{Basic: model.Basic{Id: 11, State: true}, TableName: "字典项", TableCode: "sys_dict_item"},
		{Basic: model.Basic{Id: 12, State: true}, TableName: "数据表", TableCode: "sys_table"},
		{Basic: model.Basic{Id: 13, State: true}, TableName: "数据字段", TableCode: "sys_table_field"},
		{Basic: model.Basic{Id: 14, State: true}, TableName: "菜单", TableCode: "sys_menu"},
		{Basic: model.Basic{Id: 15, State: true}, TableName: "菜单按钮", TableCode: "sys_menu_button"},
	}
	if err := db.Create(&tables).Error; err != nil {
		t.Fatalf("seed system table metadata: %v", err)
	}
	sf := newMigrationTestSnowflake(t)

	if err := seedSystemTableRelations(db, sf); err != nil {
		t.Fatalf("seed system table relations: %v", err)
	}
	if err := seedSystemTableRelations(db, sf); err != nil {
		t.Fatalf("repeat seed system table relations: %v", err)
	}

	var count int64
	if err := db.Model(&model.SysTableRelation{}).Count(&count).Error; err != nil {
		t.Fatalf("count system table relations: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 system table relations, got %d", count)
	}

	var relation model.SysTableRelation
	if err := db.Where("table_id = ? AND related_table_id = ?", 10, 11).First(&relation).Error; err != nil {
		t.Fatalf("query dict relation: %v", err)
	}
	if relation.ReferenceKey != "id" || relation.ForeignKey != "dict_id" || relation.RelationType != enum.OneToMany {
		t.Fatalf("unexpected dict relation: %#v", relation)
	}
}

func TestSystemColumnToTableFieldKeepsUnboundedTextLengthEmpty(t *testing.T) {
	field := systemColumnToTableField("access_log", migrationTestColumn{
		name:         "response",
		databaseType: "text",
		length:       9223372036854775807,
		hasLength:    true,
		nullable:     true,
		hasNullable:  true,
	}, 1)

	if field.FieldType != enum.TextFieldType {
		t.Fatalf("expected text field type, got %v", field.FieldType)
	}
	if field.FieldLength != 0 {
		t.Fatalf("text fields should not store unbounded database length, got %d", field.FieldLength)
	}
	if field.InputType != enum.TextareaInputType {
		t.Fatalf("expected textarea input type, got %v", field.InputType)
	}
}

func TestSystemColumnToTableFieldKeepsVarcharLength(t *testing.T) {
	field := systemColumnToTableField("application", migrationTestColumn{
		name:         "name",
		databaseType: "varchar",
		length:       255,
		hasLength:    true,
		nullable:     false,
		hasNullable:  true,
	}, 1)

	if field.FieldType != enum.VarcharFieldType || field.FieldLength != 255 {
		t.Fatalf("unexpected varchar metadata: type=%v length=%d", field.FieldType, field.FieldLength)
	}
	if field.Binding != "required" {
		t.Fatalf("not-null varchar should be required, got %q", field.Binding)
	}
}

type migrationTestColumn struct {
	name         string
	databaseType string
	length       int64
	hasLength    bool
	precision    int64
	scale        int64
	hasDecimal   bool
	nullable     bool
	hasNullable  bool
}

func (c migrationTestColumn) Name() string {
	return c.name
}

func (c migrationTestColumn) DatabaseTypeName() string {
	return c.databaseType
}

func (c migrationTestColumn) ColumnType() (string, bool) {
	return c.databaseType, c.databaseType != ""
}

func (c migrationTestColumn) PrimaryKey() (bool, bool) {
	return false, false
}

func (c migrationTestColumn) AutoIncrement() (bool, bool) {
	return false, false
}

func (c migrationTestColumn) Length() (int64, bool) {
	return c.length, c.hasLength
}

func (c migrationTestColumn) DecimalSize() (int64, int64, bool) {
	return c.precision, c.scale, c.hasDecimal
}

func (c migrationTestColumn) Nullable() (bool, bool) {
	return c.nullable, c.hasNullable
}

func (c migrationTestColumn) Unique() (bool, bool) {
	return false, false
}

func (c migrationTestColumn) ScanType() reflect.Type {
	return reflect.TypeOf("")
}

func (c migrationTestColumn) Comment() (string, bool) {
	return "", false
}

func (c migrationTestColumn) DefaultValue() (string, bool) {
	return "", false
}

func newMigrationTestSnowflake(t *testing.T) *utils.Snowflake {
	t.Helper()
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("new snowflake: %v", err)
	}
	return sf
}

func migrateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AccessLog{}); err != nil {
		t.Fatalf("migrate access log: %v", err)
	}
	return db
}
