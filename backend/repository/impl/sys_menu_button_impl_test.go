package impl

import (
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSysMenuButtonRepositoryCreatePersistsFalseBoolDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysMenuButton{}); err != nil {
		t.Fatalf("migrate sys_menu_button: %v", err)
	}

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
}
