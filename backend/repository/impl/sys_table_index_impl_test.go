package impl

import (
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"context"
	"testing"
)

func TestGetTableIndexesByTableIdUsesJoinSequence(t *testing.T) {
	db := testutil.OpenSQLite(t)
	if err := db.AutoMigrate(&model.SysTableField{}, &model.SysTableIndex{}, &model.SysTableIndexField{}); err != nil {
		t.Fatalf("migrate index fixtures: %v", err)
	}
	fields := []model.SysTableField{
		{Basic: model.Basic{Id: 11, State: true}, TableId: 1, FieldCode: "first", FieldName: "第一列"},
		{Basic: model.Basic{Id: 12, State: true}, TableId: 1, FieldCode: "second", FieldName: "第二列"},
	}
	index := model.SysTableIndex{
		Basic: model.Basic{Id: 21, State: true}, TableId: 1, IndexName: "idx_ordered_fields",
	}
	links := []model.SysTableIndexField{
		{IndexId: index.Id, FieldId: fields[0].Id, Sequence: 2},
		{IndexId: index.Id, FieldId: fields[1].Id, Sequence: 1},
	}
	if err := db.Create(&fields).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&index).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&links).Error; err != nil {
		t.Fatal(err)
	}

	repository := NewSysTableIndexRepositoryImpl(&database.PrimaryDB{DB: db})
	indexes, err := repository.GetTableIndexesByTableId(context.Background(), 1)
	if err != nil {
		t.Fatalf("load table indexes: %v", err)
	}
	if len(indexes) != 1 || len(indexes[0].IndexFields) != 2 {
		t.Fatalf("unexpected indexes: %+v", indexes)
	}
	if indexes[0].IndexFields[0].FieldCode != "second" || indexes[0].IndexFields[1].FieldCode != "first" {
		t.Fatalf("index fields not ordered by join sequence: %+v", indexes[0].IndexFields)
	}
}
