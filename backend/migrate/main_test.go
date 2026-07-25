package main

import (
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/utils"
	"backend/model"
	"reflect"
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

func TestMigrationStepsRegistersPlatformBaselineOrder(t *testing.T) {
	got := migrationStepNames(migrationSteps())
	want := []string{
		"auto_migrate_core_schema",
		"ensure_sys_menu_option_text",
		"ensure_data_permission_indexes",
		"backfill_sys_menu_page_binding",
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
		"sys_table_relation_metadata",
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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AccessLog{}); err != nil {
		t.Fatalf("migrate access log: %v", err)
	}
	return db
}
