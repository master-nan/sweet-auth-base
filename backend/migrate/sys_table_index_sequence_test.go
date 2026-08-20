package main

import (
	"backend/enum"
	"backend/model"
	"testing"
)

func TestBackfillSysTableIndexFieldSequencePostgreSQLIsIdempotent(t *testing.T) {
	db, cleanup := openQuerySchemePostgresSchema(t)
	defer cleanup()
	if err := db.AutoMigrate(&model.SysTable{}, &model.SysTableField{}, &model.SysTableIndex{}, &model.SysTableIndexField{}); err != nil {
		t.Fatalf("migrate metadata: %v", err)
	}
	table := model.SysTable{Basic: model.Basic{Id: 101, State: true}, TableName: "Backfill", TableCode: "pfcr_index_backfill", TableType: enum.System}
	first := model.SysTableField{Basic: model.Basic{Id: 102, State: true}, TableId: table.Id, FieldName: "First", FieldCode: "first_code", FieldType: enum.VarcharFieldType}
	second := model.SysTableField{Basic: model.Basic{Id: 103, State: true}, TableId: table.Id, FieldName: "Second", FieldCode: "second_code", FieldType: enum.VarcharFieldType}
	index := model.SysTableIndex{Basic: model.Basic{Id: 104, State: true}, TableId: table.Id, IndexName: "idx_pfcr_backfill"}
	for _, value := range []any{&table, &first, &second, &index} {
		if err := db.Create(value).Error; err != nil {
			t.Fatalf("seed metadata: %v", err)
		}
	}
	if err := db.Create(&model.SysTableIndexField{IndexId: index.Id, FieldId: first.Id, Sequence: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SysTableIndexField{IndexId: index.Id, FieldId: second.Id, Sequence: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE pfcr_index_backfill (id bigint PRIMARY KEY, first_code varchar(32), second_code varchar(32))`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_pfcr_backfill ON pfcr_index_backfill (second_code, first_code)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := backfillSysTableIndexFieldSequence(db); err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	if err := backfillSysTableIndexFieldSequence(db); err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	var fields []model.SysTableIndexField
	if err := db.Where("index_id = ?", index.Id).Order("sequence ASC").Find(&fields).Error; err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields[0].FieldId != second.Id || fields[0].Sequence != 1 || fields[1].FieldId != first.Id || fields[1].Sequence != 2 {
		t.Fatalf("backfilled index order = %+v", fields)
	}
}
