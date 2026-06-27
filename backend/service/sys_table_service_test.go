package service

import (
	"errors"
	"testing"

	"backend/enum"
	"backend/internal/database"
	"backend/model"
	"backend/repository/impl"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLowCodeDefaultMenuButtonsUseTemplateData(t *testing.T) {
	buttons := lowCodeDefaultMenuButtons("demo_table", lowCodeDefaultButtonTemplates())
	actual := make(map[string]struct {
		method   string
		path     string
		isButton bool
	}, len(buttons))
	for _, button := range buttons {
		actual[button.Code] = struct {
			method   string
			path     string
			isButton bool
		}{
			method:   button.Method,
			path:     button.Path,
			isButton: button.IsPageButton(),
		}
	}

	expected := map[string]struct {
		method   string
		path     string
		isButton bool
	}{
		"demo_table_query":  {"POST", "/admin/generalization/query/code/:code", false},
		"demo_table_create": {"POST", "/admin/generalization/create", true},
		"demo_table_detail": {"", "", true},
		"demo_table_delete": {"DELETE", "/admin/generalization/delete", true},
	}

	for code, want := range expected {
		got, ok := actual[code]
		if !ok {
			t.Fatalf("default low-code button %s missing", code)
		}
		if got.method != want.method || got.path != want.path || got.isButton != want.isButton {
			t.Fatalf("button %s = %s %s isButton=%v, want %s %s isButton=%v", code, got.method, got.path, got.isButton, want.method, want.path, want.isButton)
		}
	}
}

func TestFindPublishedLowCodeMenuIgnoresSystemMenuOptionBinding(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysMenu{}); err != nil {
		t.Fatalf("migrate menu: %v", err)
	}
	menus := []model.SysMenu{
		{
			Basic:     model.Basic{Id: 205, State: true},
			Name:      "system_user",
			Path:      "user",
			Component: "pages/system/user/Index.vue",
			PageType:  enum.MenuPageTypeFixed,
			TableCode: "sys_user",
			Option:    "sys_user",
		},
		{
			Basic:     model.Basic{Id: 901, State: true},
			Name:      "lowcode_sys_user",
			Path:      "generalization/sys_user",
			Component: "pages/develop/generalization/Index.vue",
			PageType:  enum.MenuPageTypeLowCode,
			TableCode: "sys_user",
			Option:    "sys_user",
		},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	svc := newSysTablePublishTestService(db)
	menu, err := svc.findPublishedLowCodeMenu(db, "sys_user")
	if err != nil {
		t.Fatalf("find low-code menu: %v", err)
	}
	if menu.Id != 901 {
		t.Fatalf("menu id = %d, want low-code menu 901", menu.Id)
	}
}

func TestFindPublishedLowCodeMenuPrefersVisibleEnabledMenu(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysMenu{}); err != nil {
		t.Fatalf("migrate menu: %v", err)
	}
	menus := []model.SysMenu{
		{
			Basic:     model.Basic{Id: 1, State: false},
			Name:      "lowcode_sys_user",
			Component: "pages/develop/generalization/Index.vue",
			PageType:  enum.MenuPageTypeLowCode,
			TableCode: "sys_user",
			Option:    "sys_user",
			IsHidden:  true,
		},
		{
			Basic:     model.Basic{Id: 2, State: true},
			Name:      "lowcode_sys_user",
			Component: "pages/develop/generalization/Index.vue",
			PageType:  enum.MenuPageTypeLowCode,
			TableCode: "sys_user",
			Option:    "sys_user",
		},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	svc := newSysTablePublishTestService(db)
	menu, err := svc.findPublishedLowCodeMenu(db, "sys_user")
	if err != nil {
		t.Fatalf("find low-code menu: %v", err)
	}
	if menu.Id != 2 {
		t.Fatalf("menu id = %d, want visible enabled menu 2", menu.Id)
	}
}

func TestFindPublishedLowCodeMenuDoesNotReturnSystemMenu(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysMenu{}); err != nil {
		t.Fatalf("migrate menu: %v", err)
	}
	menu := model.SysMenu{
		Basic:     model.Basic{Id: 205, State: true},
		Name:      "system_user",
		Path:      "user",
		Component: "pages/system/user/Index.vue",
		PageType:  enum.MenuPageTypeFixed,
		TableCode: "sys_user",
	}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}

	svc := newSysTablePublishTestService(db)
	_, err = svc.findPublishedLowCodeMenu(db, "sys_user")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("err = %v, want record not found", err)
	}
}

func TestIsLowCodePublishParentMenuOnlyAllowsDirectory(t *testing.T) {
	cases := []struct {
		name string
		menu model.SysMenu
		want bool
	}{
		{
			name: "directory menu",
			menu: model.SysMenu{
				Basic:    model.Basic{State: true},
				PageType: enum.MenuPageTypeDirectory,
			},
			want: true,
		},
		{
			name: "legacy directory menu without page type",
			menu: model.SysMenu{
				Basic: model.Basic{State: true},
			},
			want: true,
		},
		{
			name: "fixed page",
			menu: model.SysMenu{
				Basic:     model.Basic{State: true},
				PageType:  enum.MenuPageTypeFixed,
				TableCode: "sys_user",
			},
		},
		{
			name: "low-code page",
			menu: model.SysMenu{
				Basic:     model.Basic{State: true},
				PageType:  enum.MenuPageTypeLowCode,
				TableCode: "demo_file_page",
			},
		},
		{
			name: "hidden directory",
			menu: model.SysMenu{
				Basic:    model.Basic{State: true},
				PageType: enum.MenuPageTypeDirectory,
				IsHidden: true,
			},
		},
		{
			name: "disabled directory",
			menu: model.SysMenu{
				Basic:    model.Basic{State: false},
				PageType: enum.MenuPageTypeDirectory,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLowCodePublishParentMenu(tc.menu); got != tc.want {
				t.Fatalf("isLowCodePublishParentMenu() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHideDuplicateLowCodeMenusRevokesDuplicateGrants(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysMenu{}, &model.SysRoleMenu{}, &model.SysRoleMenuButton{}, &model.SysDataScopeBinding{}, &model.SysRoleDataScope{}, &model.SysUserDataScopeOverride{}); err != nil {
		t.Fatalf("migrate models: %v", err)
	}
	menus := []model.SysMenu{
		{
			Basic:     model.Basic{Id: 1, State: true},
			Name:      "lowcode_sys_user",
			Component: "pages/develop/generalization/Index.vue",
			PageType:  enum.MenuPageTypeLowCode,
			TableCode: "sys_user",
		},
		{
			Basic:     model.Basic{Id: 2, State: true},
			Name:      "lowcode_sys_user",
			Component: "pages/develop/generalization/Index.vue",
			PageType:  enum.MenuPageTypeLowCode,
			TableCode: "sys_user",
		},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&model.SysRoleMenu{RoleId: 1, MenuId: 2}).Error; err != nil {
		t.Fatalf("seed role menu: %v", err)
	}
	if err := db.Create(&model.SysRoleMenuButton{RoleId: 1, MenuId: 2, ButtonId: 9}).Error; err != nil {
		t.Fatalf("seed role button: %v", err)
	}
	if err := db.Create(&model.SysDataScopeBinding{Basic: model.Basic{Id: 10, State: true}, MenuId: 2, TableCode: "sys_user", DimensionCode: "tenant", FieldCode: "tenant_id"}).Error; err != nil {
		t.Fatalf("seed data permission binding: %v", err)
	}
	if err := db.Create(&model.SysRoleDataScope{Basic: model.Basic{Id: 11, State: true}, RoleId: 1, MenuId: 2, TableCode: "sys_user", DimensionCode: "tenant", Strategy: "specified", ScopeValues: "[\"1\"]"}).Error; err != nil {
		t.Fatalf("seed role data permission: %v", err)
	}
	if err := db.Create(&model.SysUserDataScopeOverride{Basic: model.Basic{Id: 12, State: true}, UserId: 1, MenuId: 2, TableCode: "sys_user", DimensionCode: "tenant", Strategy: "specified", ScopeValues: "[\"1\"]"}).Error; err != nil {
		t.Fatalf("seed user data permission override: %v", err)
	}

	svc := newSysTablePublishTestService(db)
	if err := svc.hideDuplicateLowCodeMenus(db, "sys_user", 1); err != nil {
		t.Fatalf("hide duplicate menus: %v", err)
	}

	var duplicate model.SysMenu
	if err := db.First(&duplicate, 2).Error; err != nil {
		t.Fatalf("query duplicate menu: %v", err)
	}
	if duplicate.State || !duplicate.IsHidden {
		t.Fatalf("duplicate state=%v hidden=%v, want hidden inactive", duplicate.State, duplicate.IsHidden)
	}
	var grantCount int64
	if err := db.Model(&model.SysRoleMenu{}).Where("menu_id = ?", 2).Count(&grantCount).Error; err != nil {
		t.Fatalf("count role menu: %v", err)
	}
	if grantCount != 0 {
		t.Fatalf("duplicate role menu grants = %d, want 0", grantCount)
	}
	if err := db.Model(&model.SysRoleMenuButton{}).Where("menu_id = ?", 2).Count(&grantCount).Error; err != nil {
		t.Fatalf("count role button: %v", err)
	}
	if grantCount != 0 {
		t.Fatalf("duplicate button grants = %d, want 0", grantCount)
	}
	if err := db.Model(&model.SysDataScopeBinding{}).Where("menu_id = ?", 2).Count(&grantCount).Error; err != nil {
		t.Fatalf("count data permission bindings: %v", err)
	}
	if grantCount != 0 {
		t.Fatalf("duplicate data permission bindings = %d, want 0", grantCount)
	}
	if err := db.Model(&model.SysRoleDataScope{}).Where("menu_id = ?", 2).Count(&grantCount).Error; err != nil {
		t.Fatalf("count role data permissions: %v", err)
	}
	if grantCount != 0 {
		t.Fatalf("duplicate role data permissions = %d, want 0", grantCount)
	}
	if err := db.Model(&model.SysUserDataScopeOverride{}).Where("menu_id = ?", 2).Count(&grantCount).Error; err != nil {
		t.Fatalf("count user data permission overrides: %v", err)
	}
	if grantCount != 0 {
		t.Fatalf("duplicate user data permission overrides = %d, want 0", grantCount)
	}
}

func TestCleanupLegacyLowCodeMenuButtonsRemovesSystemButtonsOnly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysMenuButton{}, &model.SysRoleMenuButton{}); err != nil {
		t.Fatalf("migrate buttons: %v", err)
	}
	buttons := []model.SysMenuButton{
		{Basic: model.Basic{Id: 1, State: true}, MenuId: 900, Name: "旧查询", Code: "system_user_query", Position: enum.Top, EventAction: "query"},
		{Basic: model.Basic{Id: 2, State: true}, MenuId: 900, Name: "新查询", Code: "sys_user_query", Position: enum.Top, EventAction: "query"},
		{Basic: model.Basic{Id: 3, State: true}, MenuId: 900, Name: "自定义", Code: "demo_ticket_mark_done", Position: enum.Line, EventAction: "custom"},
	}
	if err := db.Create(&buttons).Error; err != nil {
		t.Fatalf("seed buttons: %v", err)
	}
	grants := []model.SysRoleMenuButton{
		{RoleId: 1, ButtonId: 1},
		{RoleId: 1, ButtonId: 2},
	}
	if err := db.Create(&grants).Error; err != nil {
		t.Fatalf("seed grants: %v", err)
	}

	svc := newSysTablePublishTestService(db)
	if err := svc.cleanupLegacyLowCodeMenuButtons(db, 900); err != nil {
		t.Fatalf("cleanup legacy buttons: %v", err)
	}

	var remaining []model.SysMenuButton
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("query remaining buttons: %v", err)
	}
	if len(remaining) != 2 || remaining[0].Code != "sys_user_query" || remaining[1].Code != "demo_ticket_mark_done" {
		t.Fatalf("remaining buttons = %#v, want new low-code and custom buttons", remaining)
	}
	var grantCount int64
	if err := db.Model(&model.SysRoleMenuButton{}).Where("button_id = ?", 1).Count(&grantCount).Error; err != nil {
		t.Fatalf("count deleted grants: %v", err)
	}
	if grantCount != 0 {
		t.Fatalf("legacy grant count = %d, want 0", grantCount)
	}
}

func newSysTablePublishTestService(db *gorm.DB) *SysTableService {
	primaryDB := &database.PrimaryDB{DB: db}
	return &SysTableService{
		sysMenuRepo:           impl.NewSysMenuRepositoryImpl(primaryDB),
		sysMenuButtonRepo:     impl.NewSysMenuButtonRepositoryImpl(primaryDB),
		sysMenuButtonTplRepo:  impl.NewSysMenuButtonTemplateRepositoryImpl(primaryDB),
		sysRoleRepo:           impl.NewSysRoleRepositoryImpl(primaryDB),
		sysRoleMenuRepo:       impl.NewSysRoleMenuRepositoryImpl(primaryDB),
		sysRoleMenuButtonRepo: impl.NewSysRoleMenuButtonRepositoryImpl(primaryDB),
		dataPermissionService: &DataPermissionService{db: db},
	}
}

func lowCodeDefaultButtonTemplates() []model.SysMenuButtonTemplate {
	return []model.SysMenuButtonTemplate{
		{Name: "列表查询", CodeSuffix: "_query", Position: enum.Top, EventAction: "query", Icon: "search", Color: "primary", Sequence: 0, Path: "/admin/generalization/query/code/:code", Method: "POST", IsButton: false},
		{Name: "详情", CodeSuffix: "_detail", Position: enum.Line, EventAction: "detail", Icon: "visibility", Color: "primary", IsButton: true},
		{Name: "新增", CodeSuffix: "_create", Position: enum.Top, EventAction: "create", Icon: "add", Color: "primary", Sequence: 1, Path: "/admin/generalization/create", Method: "POST", IsButton: true},
		{Name: "删除", CodeSuffix: "_delete", Position: enum.Line, EventAction: "delete", Icon: "delete", Color: "negative", Sequence: 2, Path: "/admin/generalization/delete", Method: "DELETE", IsButton: true},
	}
}
