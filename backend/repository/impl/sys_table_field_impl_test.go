package impl

import (
	"backend/enum"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"testing"
)

func TestSysTableFieldRepositoryCreatePersistsFalseBoolDefaults(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysTableField{})

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
