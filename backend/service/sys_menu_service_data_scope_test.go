package service

import (
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	"backend/repository/impl"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestFilterGrantedMenuButtons(t *testing.T) {
	buttons := []model.SysMenuButton{
		{Basic: model.Basic{Id: 1, State: true}, MenuId: 10, Code: "query", IsButton: false, IsHidden: true},
		{Basic: model.Basic{Id: 2, State: true}, MenuId: 10, Code: "create", IsButton: true},
		{Basic: model.Basic{Id: 3, State: false}, MenuId: 10, Code: "update"},
		{Basic: model.Basic{Id: 4, State: true}, MenuId: 10, Code: "delete", IsDisabled: true},
	}

	got := filterGrantedMenuButtons(buttons, map[int]bool{1: true, 3: true, 4: true})
	if len(got) != 1 || got[0].Id != 1 {
		t.Fatalf("expected only granted active API permission, got %#v", got)
	}

	got = filterGrantedMenuButtons(buttons, nil)
	if len(got) != 0 {
		t.Fatalf("expected no buttons without grants, got %#v", got)
	}
}

func TestMenuButtonAllowsAction(t *testing.T) {
	tests := []struct {
		name   string
		button model.SysMenuButton
		want   bool
	}{
		{
			name:   "api permission is valid permission point",
			button: model.SysMenuButton{Basic: model.Basic{State: true}, MenuId: 10, EventAction: "query", IsButton: false, IsHidden: true},
			want:   true,
		},
		{
			name:   "unsupported event action",
			button: model.SysMenuButton{Basic: model.Basic{State: true}, MenuId: 10, EventAction: "lowcode:search"},
			want:   false,
		},
		{
			name:   "wrong menu",
			button: model.SysMenuButton{Basic: model.Basic{State: true}, MenuId: 11, EventAction: "query"},
			want:   false,
		},
		{
			name:   "disabled",
			button: model.SysMenuButton{Basic: model.Basic{State: true}, MenuId: 10, EventAction: "query", IsDisabled: true},
			want:   false,
		},
		{
			name:   "stopped",
			button: model.SysMenuButton{Basic: model.Basic{State: false}, MenuId: 10, EventAction: "query"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := menuButtonAllowsAction(tt.button, 10, "query")
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestNormalizeAndValidateMenuButton(t *testing.T) {
	normalMenu := model.SysMenu{Basic: model.Basic{Id: 1, State: true}, Name: "system_menu"}
	button := model.SysMenuButton{
		Basic:       model.Basic{State: true},
		MenuId:      1,
		Code:        " system_menu_query ",
		EventAction: " query ",
		Path:        " /admin/menu/query ",
		Method:      " post ",
		DisplayMode: " ICON_TEXT ",
	}
	if err := normalizeAndValidateMenuButton(&button, normalMenu); err != nil {
		t.Fatalf("expected normal admin api button to be valid: %v", err)
	}
	if button.Code != "system_menu_query" || button.EventAction != "query" || button.Path != "/admin/menu/query" || button.Method != "POST" {
		t.Fatalf("button was not normalized: %#v", button)
	}
	if button.DisplayMode != enum.ButtonDisplayIconText {
		t.Fatalf("button display mode was not normalized: %#v", button.DisplayMode)
	}

	defaultDisplay := model.SysMenuButton{Code: "default_display", EventAction: "query"}
	if err := normalizeAndValidateMenuButton(&defaultDisplay, normalMenu); err != nil {
		t.Fatalf("expected default display mode to be valid: %v", err)
	}
	if defaultDisplay.DisplayMode != enum.ButtonDisplayAuto {
		t.Fatalf("expected empty display mode to default to auto, got %#v", defaultDisplay.DisplayMode)
	}

	invalidDisplay := model.SysMenuButton{Code: "bad_display", EventAction: "query", DisplayMode: "huge"}
	if err := normalizeAndValidateMenuButton(&invalidDisplay, normalMenu); err == nil {
		t.Fatalf("expected invalid display mode to be rejected")
	}

	invalidMethod := model.SysMenuButton{Code: "bad_method", EventAction: "query", Path: "/admin/menu/query", Method: "PATCH"}
	if err := normalizeAndValidateMenuButton(&invalidMethod, normalMenu); err == nil {
		t.Fatalf("expected invalid method to be rejected")
	}

	missingPair := model.SysMenuButton{Code: "missing_pair", EventAction: "query", Path: "/admin/menu/query"}
	if err := normalizeAndValidateMenuButton(&missingPair, normalMenu); err == nil {
		t.Fatalf("expected incomplete api config to be rejected")
	}

	externalURL := model.SysMenuButton{Code: "external", EventAction: "query", Path: "https://example.com/admin/menu/query", Method: "POST"}
	if err := normalizeAndValidateMenuButton(&externalURL, normalMenu); err == nil {
		t.Fatalf("expected external api path to be rejected")
	}

	navigate := model.SysMenuButton{Code: "go_detail", EventAction: "navigate", Path: "/dashboard"}
	if err := normalizeAndValidateMenuButton(&navigate, normalMenu); err != nil {
		t.Fatalf("expected frontend navigate button to be valid: %v", err)
	}
}

func TestValidateMenuButtonRuntimeConfig(t *testing.T) {
	normalMenu := model.SysMenu{Basic: model.Basic{Id: 1, State: true}, Name: "system_menu"}

	valid := model.SysMenuButton{
		EventAction:  "custom",
		Code:         "runtime_valid",
		ParamsSchema: `{"type":"object","properties":{"reason":{"type":"string","enum":["approve","reject"]}},"required":["reason"]}`,
		DisableWhen:  `{"all":[{"field":"row.status","op":"eq","value":"done"},{"field":"selection.length","op":"gt","value":0}]}`,
		BeforeHooks:  `[" requireSelection ","requireSelection","requireRow"]`,
		AfterHooks:   `["refresh","clearSelection"]`,
	}
	if err := normalizeAndValidateMenuButton(&valid, normalMenu); err != nil {
		t.Fatalf("expected runtime config to be valid: %v", err)
	}
	if valid.BeforeHooks != `["requireSelection","requireRow"]` {
		t.Fatalf("expected hooks to be trimmed and deduplicated, got %s", valid.BeforeHooks)
	}

	tests := []struct {
		name   string
		button model.SysMenuButton
	}{
		{
			name:   "params schema invalid json",
			button: model.SysMenuButton{Code: "bad_params_json", ParamsSchema: `{"fields":`},
		},
		{
			name:   "params schema invalid field",
			button: model.SysMenuButton{Code: "bad_params_field", ParamsSchema: `[{"field_code":"bad-field"}]`},
		},
		{
			name:   "disable condition invalid op",
			button: model.SysMenuButton{Code: "bad_disable_op", DisableWhen: `{"field":"row.status","op":"exec","value":"x"}`},
		},
		{
			name:   "disable condition invalid field",
			button: model.SysMenuButton{Code: "bad_disable_field", DisableWhen: `{"field":"row.status;drop","op":"eq","value":"x"}`},
		},
		{
			name:   "hooks are not string array",
			button: model.SysMenuButton{Code: "bad_hooks", BeforeHooks: `{"hook":"refresh"}`},
		},
		{
			name:   "hook name invalid",
			button: model.SysMenuButton{Code: "bad_hook_name", AfterHooks: `["refresh()"]`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := normalizeAndValidateMenuButton(&tt.button, normalMenu); err == nil {
				t.Fatalf("expected runtime config to be rejected")
			}
		})
	}
}

func TestValidateLowCodeMenuButtonConfig(t *testing.T) {
	lowCodeMenu := model.SysMenu{
		Basic:     model.Basic{Id: 10, State: true},
		Name:      "lowcode_smoke",
		Component: "pages/develop/generalization/Index.vue",
		PageType:  enum.MenuPageTypeLowCode,
		TableCode: "smoke_table",
	}

	createButton := model.SysMenuButton{
		Code:        "smoke_table_create",
		EventAction: "create",
		Path:        "/admin/generalization/create",
		Method:      "POST",
	}
	if err := normalizeAndValidateMenuButton(&createButton, lowCodeMenu); err != nil {
		t.Fatalf("expected low-code create button to be valid: %v", err)
	}

	wrongAPI := model.SysMenuButton{
		Code:        "smoke_table_create",
		EventAction: "create",
		Path:        "/admin/user/reset_password/:id",
		Method:      "POST",
	}
	if err := normalizeAndValidateMenuButton(&wrongAPI, lowCodeMenu); err == nil {
		t.Fatalf("expected low-code create button with unrelated api to be rejected")
	}

	refreshWithAPI := model.SysMenuButton{
		Code:        "smoke_table_refresh",
		EventAction: "refresh",
		Path:        "/admin/generalization/query/code/:code",
		Method:      "POST",
	}
	if err := normalizeAndValidateMenuButton(&refreshWithAPI, lowCodeMenu); err == nil {
		t.Fatalf("expected low-code refresh api to be rejected")
	}

	customExport := model.SysMenuButton{
		Code:        "smoke_table_export",
		EventAction: "export",
		Path:        "/admin/generalization/query/code/:code",
		Method:      "POST",
	}
	if err := normalizeAndValidateMenuButton(&customExport, lowCodeMenu); err != nil {
		t.Fatalf("expected low-code custom admin api button to be valid: %v", err)
	}
}

func TestMenuButtonUpdateUsesScalarColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysMenu{}, &model.SysRole{}, &model.SysMenuButton{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	original := model.SysMenuButton{
		Basic:       model.Basic{Id: 438, State: true},
		MenuId:      203,
		Name:        "编辑",
		Code:        "system_menu_button_update",
		Position:    enum.Line,
		EventAction: "update_button",
		Icon:        "edit",
		Color:       "primary",
		DisplayMode: enum.ButtonDisplayAuto,
		Sequence:    5,
		Path:        "/admin/menu/button/:id",
		Method:      "PUT",
		IsButton:    false,
		IsHidden:    true,
		IsDisabled:  true,
	}
	if err := db.Create(&original).Error; err != nil {
		t.Fatalf("create button: %v", err)
	}

	repo := impl.NewSysMenuButtonRepositoryImpl(&database.PrimaryDB{DB: db})
	update := model.SysMenuButton{
		MenuId:      203,
		Name:        "编辑按钮",
		Code:        "system_menu_button_update",
		Position:    enum.Line,
		EventAction: "update_button",
		Icon:        "edit",
		Color:       "primary",
		DisplayMode: enum.ButtonDisplayAuto,
		Sequence:    5,
		Path:        "/admin/menu/button/:id",
		Method:      "PUT",
		IsButton:    true,
		IsHidden:    false,
		IsDisabled:  false,
	}
	if err := repo.Update(db, menuButtonUpdateMap(update), 438); err != nil {
		t.Fatalf("update button with scalar map: %v", err)
	}

	var got model.SysMenuButton
	if err := db.First(&got, 438).Error; err != nil {
		t.Fatalf("query updated button: %v", err)
	}
	if got.Name != "编辑按钮" || got.Path != "/admin/menu/button/:id" || got.Method != "PUT" {
		t.Fatalf("button was not updated correctly: %#v", got)
	}
	if !got.IsButton || got.IsHidden || got.IsDisabled {
		t.Fatalf("expected button bool fields to be persisted, got isButton=%v hidden=%v disabled=%v", got.IsButton, got.IsHidden, got.IsDisabled)
	}
	if !got.State {
		t.Fatalf("state should be preserved when editing button config")
	}
}

func TestOrphanRolePolicyCleanups(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysRole{}, &model.SysMenuButton{}, &model.SysRoleMenuButton{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&model.SysRole{Basic: model.Basic{Id: 1, State: true}, Name: "smoke_role"}).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	buttons := []model.SysMenuButton{
		{Basic: model.Basic{Id: 10, State: true}, MenuId: 20, Code: "first", Path: "/admin/smoke", Method: "POST"},
		{Basic: model.Basic{Id: 11, State: true}, MenuId: 20, Code: "second", Path: "/admin/smoke", Method: "POST"},
	}
	if err := db.Create(&buttons).Error; err != nil {
		t.Fatalf("create buttons: %v", err)
	}
	roleButtons := []model.SysRoleMenuButton{
		{RoleId: 1, MenuId: 20, ButtonId: 10},
		{RoleId: 1, MenuId: 20, ButtonId: 11},
	}
	if err := db.Create(&roleButtons).Error; err != nil {
		t.Fatalf("create role buttons: %v", err)
	}
	primaryDB := &database.PrimaryDB{DB: db}
	service := &SysMenuService{
		sysRoleRepo:           impl.NewSysRoleRepositoryImpl(primaryDB),
		sysRoleMenuButtonRepo: impl.NewSysRoleMenuButtonRepositoryImpl(primaryDB),
	}

	candidates := map[buttonPolicyKey]struct{}{{RoleID: 1, Path: "/admin/smoke", Method: "POST"}: struct{}{}}
	if err := db.Where("button_id = ?", 10).Delete(&model.SysRoleMenuButton{}).Error; err != nil {
		t.Fatalf("delete first role button: %v", err)
	}
	if err := db.Where("id = ?", 10).Delete(&model.SysMenuButton{}).Error; err != nil {
		t.Fatalf("delete first button: %v", err)
	}
	cleanups, err := service.orphanRolePolicyCleanups(db, candidates)
	if err != nil {
		t.Fatalf("orphan cleanups with shared route: %v", err)
	}
	if len(cleanups) != 0 {
		t.Fatalf("expected shared route policy to remain, got %#v", cleanups)
	}

	if err := db.Where("button_id = ?", 11).Delete(&model.SysRoleMenuButton{}).Error; err != nil {
		t.Fatalf("delete second role button: %v", err)
	}
	if err := db.Where("id = ?", 11).Delete(&model.SysMenuButton{}).Error; err != nil {
		t.Fatalf("delete second button: %v", err)
	}
	cleanups, err = service.orphanRolePolicyCleanups(db, candidates)
	if err != nil {
		t.Fatalf("orphan cleanups without shared route: %v", err)
	}
	if len(cleanups) != 1 || cleanups[0].RoleName != "smoke_role" || cleanups[0].Path != "/admin/smoke" || cleanups[0].Method != "POST" {
		t.Fatalf("expected one cleanup for smoke role policy, got %#v", cleanups)
	}
}
