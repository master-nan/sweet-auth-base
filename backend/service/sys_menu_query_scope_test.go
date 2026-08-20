package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestMenuWriteBoundaryDoesNotExposeOrChangeQueryScope(t *testing.T) {
	for _, value := range []any{request.MenuCreateReq{}, request.MenuUpdateReq{}} {
		if _, exists := reflect.TypeOf(value).FieldByName("QueryScopeCode"); exists {
			t.Fatalf("%T exposes query_scope_code as a writable field", value)
		}
	}

	db := testutil.OpenSQLite(t, &model.SysMenu{})
	scopeCode := "system.user.list"
	existing := model.SysMenu{
		Basic: model.Basic{Id: 100, State: true}, Name: "system_user", Title: "用户管理",
		Path: "/system/user", Component: "system/user/Index", PageType: enum.MenuPageTypeFixed,
		TableCode: "sys_user", QueryScopeCode: &scopeCode,
	}
	testutil.MustCreate(t, db, &existing)
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatal(err)
	}
	service := NewSysMenuService(
		impl.NewSysMenuRepositoryImpl(&database.PrimaryDB{DB: db}),
		nil, nil, nil, nil, nil, nil, nil, sf,
	)
	pid, hidden, sequence := 0, false, uint8(1)
	if err := service.UpdateMenu(context.Background(), request.MenuUpdateReq{
		Id: existing.Id, Pid: &pid, Name: existing.Name, Path: existing.Path, Component: existing.Component,
		Title: "用户与账号", IsHidden: &hidden, Sequence: &sequence, PageType: enum.MenuPageTypeFixed,
		TableCode: existing.TableCode,
	}); err != nil {
		t.Fatal(err)
	}
	var updated model.SysMenu
	if err := db.First(&updated, existing.Id).Error; err != nil {
		t.Fatal(err)
	}
	if updated.QueryScopeCode == nil || *updated.QueryScopeCode != scopeCode {
		t.Fatalf("ordinary menu update changed query scope: %#v", updated.QueryScopeCode)
	}

	if err := service.CreateMenu(context.Background(), request.MenuCreateReq{
		Pid: &pid, Name: "ordinary_menu", Path: "/ordinary", Component: "ordinary/Index",
		Title: "普通菜单", IsHidden: &hidden, Sequence: &sequence, PageType: enum.MenuPageTypeFixed,
		TableCode: "ordinary_table",
	}); err != nil {
		t.Fatal(err)
	}
	var created model.SysMenu
	if err := db.Where("name = ?", "ordinary_menu").First(&created).Error; err != nil {
		t.Fatal(err)
	}
	if created.QueryScopeCode != nil {
		t.Fatalf("ordinary menu create assigned query scope: %q", *created.QueryScopeCode)
	}
}

func TestResolvePublishedTableMenuRequiresQueryCapability(t *testing.T) {
	db := testutil.OpenSQLite(t,
		&model.SysMenu{}, &model.SysMenuButton{}, &model.SysRoleMenu{},
		&model.SysRoleMenuButton{}, &model.SysUserRole{}, &model.SysRole{},
	)
	if !db.Migrator().HasColumn(&model.SysRoleMenuButton{}, "MenuId") {
		if err := db.Migrator().AddColumn(&model.SysRoleMenuButton{}, "MenuId"); err != nil {
			t.Fatalf("add role-menu-button menu_id: %v", err)
		}
	}
	primaryDB := &database.PrimaryDB{DB: db}
	service := NewSysMenuService(
		impl.NewSysMenuRepositoryImpl(primaryDB), impl.NewSysRoleMenuRepositoryImpl(primaryDB),
		impl.NewSysRoleRepositoryImpl(primaryDB), impl.NewSysRoleMenuButtonRepositoryImpl(primaryDB),
		impl.NewSysUserRoleRepositoryImpl(primaryDB), impl.NewSysMenuButtonRepositoryImpl(primaryDB),
		nil, nil, nil,
	)
	menu := model.SysMenu{
		Basic: model.Basic{Id: 201, State: true}, Name: "orders", Title: "Orders",
		PageType: enum.MenuPageTypeLowCode, TableCode: "orders", Component: "pages/develop/generalization/Index.vue",
	}
	role := model.SysRole{Basic: model.Basic{Id: 202, State: true}, Name: "orders_reader"}
	testutil.MustCreate(t, db, &menu)
	testutil.MustCreate(t, db, &role)
	testutil.MustCreate(t, db, &model.SysUserRole{UserId: 203, RoleId: role.Id})
	testutil.MustCreate(t, db, &model.SysRoleMenu{RoleId: role.Id, MenuId: menu.Id})

	if _, published, err := service.ResolvePublishedTableMenuId(203, "missing_table", enum.ButtonActionQuery); err != nil || published {
		t.Fatalf("missing published menu = published:%v err:%v", published, err)
	}
	if _, published, err := service.ResolvePublishedTableMenuId(203, menu.TableCode, enum.ButtonActionQuery); !published || !errors.Is(err, apperrors.ErrPermissionDenied) {
		t.Fatalf("menu without query capability = published:%v err:%v", published, err)
	}

	button := model.SysMenuButton{Basic: model.Basic{Id: 204, State: true}, MenuId: menu.Id, Name: "Query", Code: "orders_query", EventAction: string(enum.ButtonActionQuery)}
	testutil.MustCreate(t, db, &button)
	testutil.MustCreate(t, db, &model.SysRoleMenuButton{RoleId: role.Id, MenuId: menu.Id, ButtonId: button.Id})
	menuID, published, err := service.ResolvePublishedTableMenuId(203, menu.TableCode, enum.ButtonActionQuery)
	if err != nil || !published || menuID != menu.Id {
		t.Fatalf("resolved query capability = menu:%d published:%v err:%v", menuID, published, err)
	}
}
