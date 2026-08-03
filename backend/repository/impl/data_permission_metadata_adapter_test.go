package impl

import (
	"context"
	"errors"
	"testing"

	"backend/enum"
	"backend/internal/database"
	"backend/internal/datapermission"
	testutil "backend/internal/test"
	"backend/model"
)

func TestDataPermissionMetadataReaderLoadsControlledProjection(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysTable{}, &model.SysTableField{})
	table := model.SysTable{
		Basic:     model.Basic{Id: 101, State: true},
		TableName: "运输订单",
		TableCode: "tms_order",
	}
	field := model.SysTableField{
		Basic:            model.Basic{Id: 501, State: true},
		TableId:          table.Id,
		FieldName:        "归属组织",
		FieldCode:        "owner_org_id",
		FieldType:        enum.BigIntFieldType,
		InputType:        enum.InputNumberInputType,
		FieldCategory:    enum.NormalField,
		IsAdvancedSearch: true,
		Sequence:         1,
	}
	testutil.MustCreate(t, db, &table)
	testutil.MustCreate(t, db, &field)

	reader := NewDataPermissionMetadataReaderImpl(&database.PrimaryDB{DB: db})
	tableRecord, err := reader.FindMetadataTable(context.Background(), table.Id)
	if err != nil {
		t.Fatalf("find metadata table: %v", err)
	}
	if tableRecord.Id != table.Id || !tableRecord.State || tableRecord.Deleted {
		t.Fatalf("unexpected table projection: %+v", tableRecord)
	}
	fieldRecord, err := reader.FindMetadataField(context.Background(), field.Id)
	if err != nil {
		t.Fatalf("find metadata field: %v", err)
	}
	if fieldRecord.Id != field.Id || fieldRecord.TableId != table.Id ||
		fieldRecord.FieldCode != field.FieldCode ||
		fieldRecord.FieldType != enum.BigIntFieldType ||
		!fieldRecord.IsAdvancedSearch {
		t.Fatalf("unexpected field projection: %+v", fieldRecord)
	}
}

func TestDataPermissionMetadataReaderExposesSoftDeleteState(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysTable{}, &model.SysTableField{})
	table := model.SysTable{
		Basic:     model.Basic{Id: 102, State: true},
		TableName: "库存",
		TableCode: "inventory",
	}
	field := model.SysTableField{
		Basic:            model.Basic{Id: 502, State: true},
		TableId:          table.Id,
		FieldName:        "法人主体",
		FieldCode:        "legal_entity_id",
		FieldType:        enum.BigIntFieldType,
		InputType:        enum.InputNumberInputType,
		FieldCategory:    enum.NormalField,
		IsAdvancedSearch: true,
		Sequence:         1,
	}
	testutil.MustCreate(t, db, &table)
	testutil.MustCreate(t, db, &field)
	if err := db.Delete(&field).Error; err != nil {
		t.Fatalf("soft delete metadata field: %v", err)
	}
	if err := db.Delete(&table).Error; err != nil {
		t.Fatalf("soft delete metadata table: %v", err)
	}

	reader := NewDataPermissionMetadataReaderImpl(&database.PrimaryDB{DB: db})
	tableRecord, err := reader.FindMetadataTable(context.Background(), table.Id)
	if err != nil || !tableRecord.Deleted {
		t.Fatalf("soft-deleted table projection=%+v err=%v", tableRecord, err)
	}
	fieldRecord, err := reader.FindMetadataField(context.Background(), field.Id)
	if err != nil || !fieldRecord.Deleted {
		t.Fatalf("soft-deleted field projection=%+v err=%v", fieldRecord, err)
	}
}

func TestDataPermissionMetadataReaderReturnsStableNotFoundSentinels(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysTable{}, &model.SysTableField{})
	reader := NewDataPermissionMetadataReaderImpl(&database.PrimaryDB{DB: db})
	_, err := reader.FindMetadataTable(context.Background(), 999)
	if !errors.Is(err, datapermission.ErrMetadataTableRecordNotFound) {
		t.Fatalf("table error=%v, want stable not-found sentinel", err)
	}
	_, err = reader.FindMetadataField(context.Background(), 999)
	if !errors.Is(err, datapermission.ErrMetadataFieldRecordNotFound) {
		t.Fatalf("field error=%v, want stable not-found sentinel", err)
	}
}
