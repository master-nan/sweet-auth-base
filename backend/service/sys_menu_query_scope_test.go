package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
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
