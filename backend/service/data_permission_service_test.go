package service

import (
	"backend/enum"
	"backend/internal/utils"
	"backend/model"
	"reflect"
	"testing"

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
		&model.SysUserRole{},
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
