package impl

import (
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSysTableFieldRepositoryCreatePersistsFalseBoolDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysTableField{}); err != nil {
		t.Fatalf("migrate sys_table_field: %v", err)
	}

	repo := NewSysTableFieldRepositoryImpl(&database.PrimaryDB{DB: db})
	field := model.SysTableField{
		TableId:          1,
		FieldName:        "名称",
		FieldCode:        "name",
		FieldType:        enum.VarcharFieldType,
		FieldLength:      64,
		InputType:        enum.InputType,
		IsNull:           false,
		IsListShow:       false,
		IsInsertShow:     false,
		IsUpdateShow:     false,
		IsQuickSearch:    false,
		IsAdvancedSearch: false,
		IsSort:           false,
		Sequence:         1,
	}

	if err := repo.Create(db, &field); err != nil {
		t.Fatalf("create sys_table_field: %v", err)
	}

	var got model.SysTableField
	if err := db.First(&got, "field_code = ?", "name").Error; err != nil {
		t.Fatalf("find sys_table_field: %v", err)
	}
	if got.IsNull || got.IsListShow || got.IsInsertShow || got.IsUpdateShow {
		t.Fatalf("false bool defaults were not persisted: %+v", got)
	}
}
