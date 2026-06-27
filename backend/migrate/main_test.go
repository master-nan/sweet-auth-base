package main

import (
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/utils"
	"backend/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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
}

func TestSeedAuditMenuButtonsIncludesDetailRefresh(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysMenuButton{}, &model.SysRoleMenuButton{}); err != nil {
		t.Fatalf("migrate audit buttons: %v", err)
	}
	sf := newMigrationTestSnowflake(t)
	if err := seedAuditMenuButtons(db, sf, 1, "", 206); err != nil {
		t.Fatalf("seed audit buttons: %v", err)
	}

	var button model.SysMenuButton
	if err := db.Where("code = ? AND position = ?", "system_audit_detail_refresh", enum.DetailTop).First(&button).Error; err != nil {
		t.Fatalf("query audit detail refresh button: %v", err)
	}
	if button.EventAction != "refresh" || button.Path != "" || button.Method != "" {
		t.Fatalf("unexpected audit detail refresh button: %+v", button)
	}

	var count int64
	if err := db.Model(&model.SysRoleMenuButton{}).Where("role_id = ? AND menu_id = ? AND button_id = ?", 1, 206, button.Id).Count(&count).Error; err != nil {
		t.Fatalf("count role button binding: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected super admin binding for audit detail refresh, got %d", count)
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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AccessLog{}); err != nil {
		t.Fatalf("migrate access log: %v", err)
	}
	return db
}
