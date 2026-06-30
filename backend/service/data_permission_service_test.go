package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/utils"
	"backend/model"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestResolveDataScopeRequiredBindingWithoutRoleDenies(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)

	scope, err := service.ResolveDataScope(model.SysUser{Basic: model.Basic{Id: 99}}, 10, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve data scope: %v", err)
	}
	if scope == nil || !scope.DenyAll {
		t.Fatalf("expected deny all scope without user roles, got %#v", scope)
	}
}

func TestResolveDataScopeSuperAdminBypassesRequiredBinding(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysRole{Basic: model.Basic{Id: 1, State: true}, Name: "super_admin"})
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})

	scope, err := service.ResolveDataScope(model.SysUser{Basic: model.Basic{Id: 7}}, 10, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve data scope: %v", err)
	}
	if scope == nil || !scope.AllowAll || scope.DenyAll || len(scope.Conditions) != 0 {
		t.Fatalf("expected super admin allow all scope, got %#v", scope)
	}
}

func TestResolveDataScopeMergesRoleScopesAndUserIntersection(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 2})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 20, State: true},
		RoleId:        1,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "specified",
		ScopeValues:   `["1","2"]`,
	})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 21, State: true},
		RoleId:        2,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "specified",
		ScopeValues:   `["2","3"]`,
	})
	mustCreate(t, service.db, &model.SysUserDataScopeOverride{
		Basic:         model.Basic{Id: 30, State: true},
		UserId:        7,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "specified",
		ScopeValues:   `["2","4"]`,
		OverrideMode:  "intersect",
	})

	scope, err := service.ResolveDataScope(model.SysUser{Basic: model.Basic{Id: 7}}, 10, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve data scope: %v", err)
	}
	if scope == nil || scope.AllowAll || scope.DenyAll || len(scope.Conditions) != 1 {
		t.Fatalf("expected one constrained scope, got %#v", scope)
	}
	got := scope.Conditions[0]
	if got.Field != "tenant_id" || got.MatchType != "in" || !reflect.DeepEqual(got.Values, []string{"2"}) {
		t.Fatalf("unexpected condition: %#v", got)
	}
}

func TestResolveDataScopeUserDenyOverrideWins(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 20, State: true},
		RoleId:        1,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "all",
	})
	mustCreate(t, service.db, &model.SysUserDataScopeOverride{
		Basic:         model.Basic{Id: 30, State: true},
		UserId:        7,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "none",
		OverrideMode:  "deny",
	})

	scope, err := service.ResolveDataScope(model.SysUser{Basic: model.Basic{Id: 7}}, 10, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve data scope: %v", err)
	}
	if scope == nil || !scope.DenyAll {
		t.Fatalf("expected deny override to win, got %#v", scope)
	}
}

func TestResolveDataScopeUserDimensionStrategyUsesCurrentUserValues(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 20, State: true},
		RoleId:        1,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "user_dimension",
	})
	mustCreate(t, service.db, &model.SysUserDimensionValue{
		Basic:         model.Basic{Id: 30, State: true},
		UserId:        7,
		DimensionCode: "tenant",
		ScopeValues:   `["8","9"]`,
	})
	mustCreate(t, service.db, &model.SysUserDimensionValue{
		Basic:         model.Basic{Id: 31, State: true},
		UserId:        8,
		DimensionCode: "tenant",
		ScopeValues:   `["10"]`,
	})

	scope, err := service.ResolveDataScope(model.SysUser{Basic: model.Basic{Id: 7}}, 10, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve user dimension scope: %v", err)
	}
	if scope == nil || scope.AllowAll || scope.DenyAll || len(scope.Conditions) != 1 {
		t.Fatalf("expected one user dimension condition, got %#v", scope)
	}
	got := scope.Conditions[0]
	if got.Field != "tenant_id" || !reflect.DeepEqual(got.Values, []string{"8", "9"}) {
		t.Fatalf("unexpected user dimension condition: %#v", got)
	}
}

func TestResolveDataScopeUserDimensionStrategyWithoutValuesDenies(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 20, State: true},
		RoleId:        1,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "user_dimension",
	})

	scope, err := service.ResolveDataScope(model.SysUser{Basic: model.Basic{Id: 7}}, 10, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve missing user dimension scope: %v", err)
	}
	if scope == nil || !scope.DenyAll {
		t.Fatalf("expected deny all without user dimension values, got %#v", scope)
	}
}

func TestSaveUserDimensionValuesMergesDuplicateDimensions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newDataPermissionServiceForTest(t)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	mustCreate(t, service.db, &model.SysDataDimension{
		Basic:      model.Basic{Id: 40, State: true},
		Code:       "tenant",
		Name:       "租户",
		ValueType:  "number",
		SourceType: "none",
	})

	err := service.SaveUserDimensionValues(ctx, 7, request.UserDimensionValueSaveReq{
		UserId: 7,
		Items: []request.UserDimensionValueItemReq{
			{DimensionCode: "tenant", ScopeValues: []string{"2", "1"}},
			{DimensionCode: "tenant", ScopeValues: []string{"2", "3"}},
		},
	})
	if err != nil {
		t.Fatalf("save duplicate user dimension values: %v", err)
	}
	var items []model.SysUserDimensionValue
	if err := service.db.Where("user_id = ? AND dimension_code = ?", 7, "tenant").Find(&items).Error; err != nil {
		t.Fatalf("query user dimension values: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one merged dimension value row, got %d: %#v", len(items), items)
	}
	if got := decodeStringList(items[0].ScopeValues); !reflect.DeepEqual(got, []string{"1", "2", "3"}) {
		t.Fatalf("expected merged sorted values, got %#v", got)
	}
}

func TestSaveUserDimensionValuesCanRepeatClearAndResave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newDataPermissionServiceForTest(t)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	mustCreate(t, service.db, &model.SysDataDimension{
		Basic:      model.Basic{Id: 40, State: true},
		Code:       "tenant",
		Name:       "租户",
		ValueType:  "number",
		SourceType: "none",
	})

	saveReq := request.UserDimensionValueSaveReq{
		UserId: 7,
		Items: []request.UserDimensionValueItemReq{
			{DimensionCode: "tenant", ScopeValues: []string{"1", "2"}},
		},
	}
	if err := service.SaveUserDimensionValues(ctx, 7, saveReq); err != nil {
		t.Fatalf("initial save user dimension values: %v", err)
	}
	if err := service.SaveUserDimensionValues(ctx, 7, saveReq); err != nil {
		t.Fatalf("repeat save user dimension values should not hit unique index: %v", err)
	}
	if err := service.SaveUserDimensionValues(ctx, 7, request.UserDimensionValueSaveReq{UserId: 7}); err != nil {
		t.Fatalf("clear user dimension values: %v", err)
	}
	if err := service.SaveUserDimensionValues(ctx, 7, request.UserDimensionValueSaveReq{
		UserId: 7,
		Items: []request.UserDimensionValueItemReq{
			{DimensionCode: "tenant", ScopeValues: []string{"3"}},
		},
	}); err != nil {
		t.Fatalf("resave user dimension values after clear should not hit unique index: %v", err)
	}

	var items []model.SysUserDimensionValue
	if err := service.db.Where("user_id = ? AND dimension_code = ?", 7, "tenant").Find(&items).Error; err != nil {
		t.Fatalf("query user dimension values: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one active user dimension row, got %d: %#v", len(items), items)
	}
	if got := decodeStringList(items[0].ScopeValues); !reflect.DeepEqual(got, []string{"3"}) {
		t.Fatalf("expected latest values after resave, got %#v", got)
	}
}

func TestSaveRoleDataScopesDoesNotPersistUnownedMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newDataPermissionServiceForTest(t)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysRole{Basic: model.Basic{Id: 1, State: true}, Name: "operator"})
	mustCreate(t, service.db, &model.SysMenu{
		Basic:     model.Basic{Id: 10, State: true},
		Name:      "demo_order",
		PageType:  enum.MenuPageTypeFixed,
		TableCode: "demo_order",
	})

	err := service.SaveRoleDataScopes(ctx, 1, request.RoleDataPermissionSaveReq{
		RoleId: 1,
		Permissions: []request.RoleDataPermissionItemReq{
			{
				MenuId:        10,
				TableCode:     "demo_order",
				DimensionCode: "tenant",
				Strategy:      "specified",
				ScopeValues:   []string{"8"},
			},
		},
	})
	var count int64
	if queryErr := service.db.Model(&model.SysRoleDataScope{}).Where("role_id = ? AND menu_id = ?", 1, 10).Count(&count).Error; queryErr != nil {
		t.Fatalf("count role data scopes: %v", queryErr)
	}
	if err == nil && count != 0 {
		t.Fatalf("expected unowned menu data scopes to be rejected or left unpersisted, got %d rows", count)
	}
	if err != nil && count != 0 {
		t.Fatalf("save returned error but left partial data scopes: %d rows, err: %v", count, err)
	}
}

func TestResolveDataScopeSpecifiedStrategyNormalizesValues(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 20, State: true},
		RoleId:        1,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "specified",
		ScopeValues:   `["9","8","8"]`,
	})

	scope, err := service.ResolveDataScope(model.SysUser{Basic: model.Basic{Id: 7}}, 10, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve specified scope: %v", err)
	}
	if scope == nil || scope.AllowAll || scope.DenyAll || len(scope.Conditions) != 1 {
		t.Fatalf("expected one specified condition, got %#v", scope)
	}
	if got := scope.Conditions[0].Values; !reflect.DeepEqual(got, []string{"8", "9"}) {
		t.Fatalf("expected sorted unique specified values, got %#v", got)
	}
}

func TestResolveDataScopeForTableActionUsesUniqueBoundMenu(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysMenu{
		Basic:     model.Basic{Id: 10, State: true},
		Name:      "demo_order",
		PageType:  enum.MenuPageTypeFixed,
		TableCode: "demo_order",
	})
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 20, State: true},
		RoleId:        1,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "specified",
		ScopeValues:   `["8"]`,
	})

	scope, err := service.ResolveDataScopeForTableAction(model.SysUser{Basic: model.Basic{Id: 7}}, 0, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve data scope by table action: %v", err)
	}
	if scope == nil || scope.AllowAll || scope.DenyAll || len(scope.Conditions) != 1 {
		t.Fatalf("expected scoped result from unique bound menu, got %#v", scope)
	}
	if got := scope.Conditions[0].Values; !reflect.DeepEqual(got, []string{"8"}) {
		t.Fatalf("unexpected scope values: %#v", got)
	}
}

func TestResolveDataScopeForTableActionDeniesAmbiguousBoundMenus(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysDataScopeBinding{
		Basic:         model.Basic{Id: 11, State: true},
		MenuId:        11,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		FieldCode:     "tenant_id",
		MatchType:     "in",
		Required:      true,
		Actions:       `["query"]`,
	})
	mustCreate(t, service.db, &model.SysMenu{
		Basic:     model.Basic{Id: 10, State: true},
		Name:      "demo_order_a",
		PageType:  enum.MenuPageTypeFixed,
		TableCode: "demo_order",
	})
	mustCreate(t, service.db, &model.SysMenu{
		Basic:     model.Basic{Id: 11, State: true},
		Name:      "demo_order_b",
		PageType:  enum.MenuPageTypeFixed,
		TableCode: "demo_order",
	})

	scope, err := service.ResolveDataScopeForTableAction(model.SysUser{Basic: model.Basic{Id: 7}}, 0, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve data scope by table action: %v", err)
	}
	if scope == nil || !scope.DenyAll {
		t.Fatalf("expected deny all for ambiguous bound menus, got %#v", scope)
	}
}

func TestCheckRecordDataScopeDeniesOutsideRecord(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	table.TableFields = append(table.TableFields, model.SysTableField{FieldCode: "gmt_delete", FieldType: enum.DatetimeFieldType})
	if err := service.db.Exec(`CREATE TABLE demo_order (id INTEGER PRIMARY KEY, tenant_id INTEGER, name TEXT, gmt_delete DATETIME)`).Error; err != nil {
		t.Fatalf("create demo table: %v", err)
	}
	if err := service.db.Exec(`INSERT INTO demo_order (id, tenant_id, name) VALUES (1, 8, 'allowed'), (2, 9, 'denied')`).Error; err != nil {
		t.Fatalf("seed demo rows: %v", err)
	}
	seedDataPermissionBinding(t, service.db, true)
	mustCreate(t, service.db, &model.SysMenu{
		Basic:     model.Basic{Id: 10, State: true},
		Name:      "demo_order",
		PageType:  enum.MenuPageTypeFixed,
		TableCode: "demo_order",
	})
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 20, State: true},
		RoleId:        1,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "tenant",
		Strategy:      "specified",
		ScopeValues:   `["8"]`,
	})

	user := model.SysUser{Basic: model.Basic{Id: 7}}
	if err := service.CheckRecordDataScope(user, 10, table, 1, enum.ButtonActionUpdate); err != nil {
		t.Fatalf("expected row 1 to pass: %v", err)
	}
	if err := service.CheckRecordDataScope(user, 10, table, 2, enum.ButtonActionUpdate); err == nil {
		t.Fatalf("expected row 2 to be denied")
	}
}

func TestRecordContainsValueMatchesFileIdExactly(t *testing.T) {
	service, _ := newDataPermissionServiceForTest(t)
	if err := service.db.Exec(`CREATE TABLE demo_file_record (id INTEGER PRIMARY KEY, attachments TEXT)`).Error; err != nil {
		t.Fatalf("create file record table: %v", err)
	}
	if err := service.db.Exec(`INSERT INTO demo_file_record (id, attachments) VALUES (1, '[11,12]')`).Error; err != nil {
		t.Fatalf("seed file record: %v", err)
	}
	table := model.SysTable{
		TableCode: "demo_file_record",
		TableFields: []model.SysTableField{
			{FieldCode: "attachments", InputType: enum.FilePickerInputType},
		},
	}
	matched, err := service.RecordContainsValue(table, 1, "11")
	if err != nil {
		t.Fatalf("contains exact file id: %v", err)
	}
	if !matched {
		t.Fatalf("expected exact id 11 to match")
	}
	matched, err = service.RecordContainsValue(table, 1, "1")
	if err != nil {
		t.Fatalf("contains inexact file id: %v", err)
	}
	if matched {
		t.Fatalf("did not expect id 1 to match [11,12]")
	}
}

func TestTreeDataScopeExpandsDescendants(t *testing.T) {
	service, table := newDataPermissionServiceForTest(t)
	if err := service.db.Exec(`CREATE TABLE org_tree (id INTEGER PRIMARY KEY, parent_id INTEGER, name TEXT)`).Error; err != nil {
		t.Fatalf("create tree table: %v", err)
	}
	if err := service.db.Exec(`INSERT INTO org_tree (id, parent_id, name) VALUES (1, 0, 'root'), (2, 1, 'child'), (3, 2, 'leaf'), (4, 0, 'other')`).Error; err != nil {
		t.Fatalf("seed tree rows: %v", err)
	}
	mustCreate(t, service.db, &model.SysDataDimension{
		Basic:       model.Basic{Id: 2, State: true},
		Code:        "org",
		Name:        "组织",
		ValueType:   "number",
		SourceType:  "table",
		SourceCode:  "org_tree",
		ValueField:  "id",
		LabelField:  "name",
		ParentField: "parent_id",
	})
	mustCreate(t, service.db, &model.SysDataScopeBinding{
		Basic:         model.Basic{Id: 11, State: true},
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "org",
		FieldCode:     "tenant_id",
		MatchType:     "in",
		Required:      true,
		Actions:       `["query"]`,
	})
	mustCreate(t, service.db, &model.SysUserRole{UserId: 7, RoleId: 1})
	mustCreate(t, service.db, &model.SysRoleDataScope{
		Basic:         model.Basic{Id: 20, State: true},
		RoleId:        1,
		MenuId:        10,
		TableCode:     "demo_order",
		DimensionCode: "org",
		Strategy:      "tree",
		ScopeValues:   `["1"]`,
	})

	scope, err := service.ResolveDataScope(model.SysUser{Basic: model.Basic{Id: 7}}, 10, table, enum.ButtonActionQuery)
	if err != nil {
		t.Fatalf("resolve tree scope: %v", err)
	}
	if scope == nil || scope.DenyAll || len(scope.Conditions) != 1 {
		t.Fatalf("expected one tree condition, got %#v", scope)
	}
	if !reflect.DeepEqual(scope.Conditions[0].Values, []string{"1", "2", "3"}) {
		t.Fatalf("unexpected expanded values: %#v", scope.Conditions[0].Values)
	}
}

func newDataPermissionServiceForTest(t *testing.T) (*DataPermissionService, model.SysTable) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SysDataDimension{},
		&model.SysDataScopeBinding{},
		&model.SysRoleDataScope{},
		&model.SysUserDataScopeOverride{},
		&model.SysUserDimensionValue{},
		&model.SysUserRole{},
		&model.SysRoleMenu{},
		&model.SysRole{},
		&model.SysMenu{},
	); err != nil {
		t.Fatalf("migrate data permission models: %v", err)
	}
	table := model.SysTable{
		Basic:     model.Basic{Id: 1, State: true},
		TableCode: "demo_order",
		TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType},
			{FieldCode: "tenant_id", FieldType: enum.IntFieldType},
			{FieldCode: "name", FieldType: enum.VarcharFieldType},
		},
	}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	return &DataPermissionService{db: db, sf: sf}, table
}

func seedDataPermissionBinding(t *testing.T, db *gorm.DB, required bool) {
	t.Helper()
	mustCreate(t, db, &model.SysDataDimension{
		Basic:      model.Basic{Id: 1, State: true},
		Code:       "tenant",
		Name:       "租户",
		ValueType:  "number",
		SourceType: "none",
	})
	row := map[string]interface{}{
		"id":             10,
		"state":          true,
		"menu_id":        10,
		"table_code":     "demo_order",
		"dimension_code": "tenant",
		"field_code":     "tenant_id",
		"match_type":     "in",
		"required":       required,
		"actions":        `["query","update","delete"]`,
	}
	if err := db.Model(&model.SysDataScopeBinding{}).Create(row).Error; err != nil {
		t.Fatalf("seed data permission binding: %v", err)
	}
}

func mustCreate(t *testing.T, db *gorm.DB, value interface{}) {
	t.Helper()
	if err := db.Create(value).Error; err != nil {
		t.Fatalf("create %T: %v", value, err)
	}
}
