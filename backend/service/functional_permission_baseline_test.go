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

	enforcer, err := casbin.NewEnforcer("../casbin_model.conf")
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
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
