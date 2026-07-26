package service

import (
	"backend/dto/request"
	"backend/internal/database"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"net/http/httptest"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAssignPermissionsPersistsMenuAndButtonGrantsAndBuildsCasbinPolicy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SysRole{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.SysRoleDataScope{},
	); err != nil {
		t.Fatalf("migrate functional permission models: %v", err)
	}

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
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&buttons).Error; err != nil {
		t.Fatalf("seed buttons: %v", err)
	}

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

	allowed, err := enforcer.Enforce(role.Name, buttons[0].Path, buttons[0].Method)
	if err != nil {
		t.Fatalf("enforce selected button policy: %v", err)
	}
	if !allowed {
		t.Fatal("expected selected button API policy to be granted")
	}
	allowed, err = enforcer.Enforce(role.Name, buttons[1].Path, buttons[1].Method)
	if err != nil {
		t.Fatalf("enforce unselected button policy: %v", err)
	}
	if allowed {
		t.Fatal("expected button outside selected menu to remain denied")
	}
}
