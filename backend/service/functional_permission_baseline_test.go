package service

import (
	"backend/dto/request"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"net/http/httptest"
	"testing"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-gonic/gin"
)

func TestAssignPermissionsPersistsMenuAndButtonGrantsAndBuildsCasbinPolicy(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.SysRole{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.SysRoleDataScope{},
		&model.CasbinRule{},
	)

	role := model.SysRole{
		Basic: model.Basic{Id: 1, State: true},
		Name:  "example_operator",
	}
	menus := []model.SysMenu{
		{Basic: model.Basic{Id: 10, State: true}, Name: "example_page"},
		{Basic: model.Basic{Id: 20, State: true}, Name: "other_page"},
	}
	buttons := []model.SysMenuButton{
		{
			Basic:       model.Basic{Id: 100, State: true},
			MenuId:      10,
			Name:        "修改",
			Code:        "update",
			EventAction: "update",
			Path:        "/admin/example/:id",
			Method:      "PUT",
			IsButton:    true,
		},
		{
			Basic:       model.Basic{Id: 200, State: true},
			MenuId:      20,
			Name:        "删除",
			Code:        "delete",
			EventAction: "delete",
			Path:        "/admin/other/:id",
			Method:      "DELETE",
			IsButton:    true,
		},
	}
	testutil.MustCreate(t, db, &role)
	testutil.MustCreate(t, db, &menus)
	testutil.MustCreate(t, db, &buttons)

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		t.Fatalf("new casbin adapter: %v", err)
	}
	enforcer, err := casbin.NewSyncedEnforcer("../casbin_model.conf", adapter)
	if err != nil {
		t.Fatalf("new synced enforcer: %v", err)
	}
	if err = enforcer.LoadPolicy(); err != nil {
		t.Fatalf("load policy: %v", err)
	}
	primaryDB := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("new snowflake: %v", err)
	}
	dataPermissionService := &DataPermissionService{db: db, sf: sf}
	svc := NewSysRoleService(
		impl.NewSysMenuButtonRepositoryImpl(primaryDB),
		impl.NewSysRoleRepositoryImpl(primaryDB),
		impl.NewSysRoleMenuRepositoryImpl(primaryDB),
		impl.NewSysRoleMenuButtonRepositoryImpl(primaryDB),
		impl.NewCasbinRuleRepositoryImpl(primaryDB, enforcer),
		dataPermissionService,
		sf,
	)
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	err = svc.AssignPermissions(ctx, request.RoleAssignPermissionsReq{
		RoleId:    role.Id,
		MenuIds:   []int{menus[0].Id},
		ButtonIds: []int{buttons[0].Id, buttons[1].Id},
	})
	if err != nil {
		t.Fatalf("assign permissions: %v", err)
	}

	var roleMenus []model.SysRoleMenu
	if err := db.Find(&roleMenus).Error; err != nil {
		t.Fatalf("query role menus: %v", err)
	}
	if len(roleMenus) != 1 || roleMenus[0].MenuId != menus[0].Id {
		t.Fatalf("expected only selected menu grant, got %#v", roleMenus)
	}

	var roleButtons []model.SysRoleMenuButton
	if err := db.Find(&roleButtons).Error; err != nil {
		t.Fatalf("query role buttons: %v", err)
	}
	if len(roleButtons) != 1 || roleButtons[0].ButtonId != buttons[0].Id {
		t.Fatalf("expected only button under selected menu, got %#v", roleButtons)
	}

	testutil.AssertPermissions(
		t,
		enforcer,
		testutil.PermissionCase{
			Name:    "selected button",
			Subject: role.Name,
			Path:    buttons[0].Path,
			Method:  buttons[0].Method,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "button outside selected menu",
			Subject: role.Name,
			Path:    buttons[1].Path,
			Method:  buttons[1].Method,
			Allowed: false,
		},
	)
}

func TestAssignPermissionsPreservesSeededRoutePolicyRepresentedByPermissionButton(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.SysRole{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.SysRoleDataScope{},
		&model.CasbinRule{},
	)

	role := model.SysRole{
		Basic: model.Basic{Id: 1, State: true},
		Name:  "super_admin",
	}
	menu := model.SysMenu{
		Basic: model.Basic{Id: 304, State: true},
		Name:  "develop_dictionary",
	}
	buttons := []model.SysMenuButton{
		{
			Basic:       model.Basic{Id: 455, State: true},
			MenuId:      menu.Id,
			Name:        "字典编码查询",
			Code:        "develop_dictionary_code_query",
			EventAction: "query",
			Path:        "/admin/dict/code/:code",
			Method:      "GET",
			IsButton:    false,
		},
		{
			Basic:       model.Basic{Id: 456, State: true},
			MenuId:      menu.Id,
			Name:        "字典列表",
			Code:        "develop_dictionary_query",
			EventAction: "query",
			Path:        "/admin/dict/query",
			Method:      "POST",
			IsButton:    false,
		},
	}
	testutil.MustCreate(t, db, &role)
	testutil.MustCreate(t, db, &menu)
	testutil.MustCreate(t, db, &buttons)

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		t.Fatalf("new casbin adapter: %v", err)
	}
	enforcer, err := casbin.NewSyncedEnforcer("../casbin_model.conf", adapter)
	if err != nil {
		t.Fatalf("new synced enforcer: %v", err)
	}
	if err = enforcer.LoadPolicy(); err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if _, err = enforcer.AddPolicy(role.Name, "/admin/dict/code/:code", "GET"); err != nil {
		t.Fatalf("seed dictionary code policy: %v", err)
	}

	primaryDB := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("new snowflake: %v", err)
	}
	svc := NewSysRoleService(
		impl.NewSysMenuButtonRepositoryImpl(primaryDB),
		impl.NewSysRoleRepositoryImpl(primaryDB),
		impl.NewSysRoleMenuRepositoryImpl(primaryDB),
		impl.NewSysRoleMenuButtonRepositoryImpl(primaryDB),
		impl.NewCasbinRuleRepositoryImpl(primaryDB, enforcer),
		&DataPermissionService{db: db, sf: sf},
		sf,
	)
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	if err = svc.AssignPermissions(ctx, request.RoleAssignPermissionsReq{
		RoleId:    role.Id,
		MenuIds:   []int{menu.Id},
		ButtonIds: []int{buttons[0].Id, buttons[1].Id},
	}); err != nil {
		t.Fatalf("assign permissions: %v", err)
	}

	allowed, err := enforcer.Enforce(role.Name, "/admin/dict/code/:code", "GET")
	if err != nil {
		t.Fatalf("enforce dictionary code policy: %v", err)
	}
	if !allowed {
		t.Fatal("seeded dictionary code policy was lost after role permission assignment")
	}
}
