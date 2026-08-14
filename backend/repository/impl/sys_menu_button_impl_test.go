package impl

import (
	"backend/enum"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"testing"
)

func TestSysMenuButtonRepositoryCreatePersistsFalseBoolDefaults(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysMenuButton{})

	repo := NewSysMenuButtonRepositoryImpl(&database.PrimaryDB{DB: db})
	button := model.SysMenuButton{
		Basic:       model.Basic{Id: 1, State: true},
		MenuId:      10,
		Name:        "列表查询",
		Code:        "system_user_query",
		Position:    enum.Top,
		EventAction: "query",
		Path:        "/admin/user/query",
		Method:      "POST",
		IsButton:    false,
		IsHidden:    false,
	}
	if err := repo.Create(db, &button); err != nil {
		t.Fatalf("create menu button: %v", err)
	}

	var got model.SysMenuButton
	if err := db.First(&got, button.Id).Error; err != nil {
		t.Fatalf("query menu button: %v", err)
	}
	if got.IsButton || got.IsHidden {
		t.Fatalf("expected non-page api permission, got is_button=%v is_hidden=%v", got.IsButton, got.IsHidden)
	}
	if got.DisplayMode != enum.ButtonDisplayAuto {
		t.Fatalf("expected default display mode auto, got %q", got.DisplayMode)
	}
}
