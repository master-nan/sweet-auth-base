package main

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	repositoryimpl "backend/repository/impl"
	"context"
	"testing"
)

func TestMetadataValueContractPostgres16AndDecimalRoundTrip(t *testing.T) {
	db, cleanup := openQuerySchemePostgresSchema(t)
	defer cleanup()
	if err := db.AutoMigrate(&model.SysDict{}, &model.SysTable{}, &model.SysTableField{}); err != nil {
		t.Fatalf("auto migrate metadata prerequisites: %v", err)
	}
	if err := migrateMetadataValueContract(db); err != nil {
		t.Fatalf("first metadata value migration: %v", err)
	}
	if err := migrateMetadataValueContract(db); err != nil {
		t.Fatalf("second metadata value migration: %v", err)
	}

	table := model.SysTable{Basic: model.Basic{Id: 8101, State: true}, TableName: "精确数值测试", TableCode: "decimal_round_trip"}
	if err := db.Create(&table).Error; err != nil {
		t.Fatalf("create table metadata: %v", err)
	}
	validSmall := "32767"
	validDecimal := "12345678901234567890.1234567890"
	metadataRows := []model.SysTableField{
		{Basic: model.Basic{Id: 8201, State: true}, TableId: table.Id, FieldName: "小整数", FieldCode: "small_value", FieldType: enum.SmallIntFieldType, DefaultValue: &validSmall},
		{Basic: model.Basic{Id: 8202, State: true}, TableId: table.Id, FieldName: "精确数值", FieldCode: "amount", FieldType: enum.DecimalFieldType, NumericPrecision: 30, NumericScale: 10, DefaultValue: &validDecimal},
	}
	if err := db.Create(&metadataRows).Error; err != nil {
		t.Fatalf("create valid metadata: %v", err)
	}
	badSmall := "32768"
	invalidSmall := model.SysTableField{Basic: model.Basic{Id: 8203, State: true}, TableId: table.Id, FieldName: "越界", FieldCode: "bad_small", FieldType: enum.SmallIntFieldType, DefaultValue: &badSmall}
	if err := db.Create(&invalidSmall).Error; err == nil {
		t.Fatal("PostgreSQL must reject an out-of-range SmallInt default")
	}
	invalidShape := model.SysTableField{Basic: model.Basic{Id: 8204, State: true}, TableId: table.Id, FieldName: "错误精度", FieldCode: "bad_decimal", FieldType: enum.DecimalFieldType, NumericPrecision: 4, NumericScale: 5}
	if err := db.Create(&invalidShape).Error; err == nil {
		t.Fatal("PostgreSQL must reject an invalid numeric precision/scale")
	}

	if err := db.Exec(`CREATE TABLE decimal_round_trip (
		id bigint PRIMARY KEY,
		large_value numeric(30,10) NOT NULL,
		money_value numeric(30,2) NOT NULL,
		weight_value numeric(30,12) NOT NULL,
		ratio_value numeric(30,18) NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create decimal round-trip table: %v", err)
	}
	runtimeTable := model.SysTable{TableCode: "decimal_round_trip", TableFields: []model.SysTableField{
		{FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true, IsListShow: true},
		{FieldCode: "large_value", FieldType: enum.DecimalFieldType, NumericPrecision: 30, NumericScale: 10, IsListShow: true},
		{FieldCode: "money_value", FieldType: enum.DecimalFieldType, NumericPrecision: 30, NumericScale: 2, IsListShow: true},
		{FieldCode: "weight_value", FieldType: enum.DecimalFieldType, NumericPrecision: 30, NumericScale: 12, IsListShow: true},
		{FieldCode: "ratio_value", FieldType: enum.DecimalFieldType, NumericPrecision: 30, NumericScale: 18, IsListShow: true},
	}}
	repository := repositoryimpl.NewGeneralizationRepositoryImpl(&database.PrimaryDB{DB: db})
	values := map[string]interface{}{
		"id": 1, "large_value": "12345678901234567890.1234567890", "money_value": "99999999999999999999.99",
		"weight_value": "9876543210.123456789012", "ratio_value": "0.123456789012345678",
	}
	if err := repository.Create(context.Background(), runtimeTable, values); err != nil {
		t.Fatalf("generalization create exact decimals: %v", err)
	}
	detail, err := repository.GetById(context.Background(), runtimeTable, 1)
	if err != nil {
		t.Fatalf("generalization detail exact decimals: %v", err)
	}
	for field, want := range map[string]string{
		"large_value": "12345678901234567890.1234567890", "money_value": "99999999999999999999.99",
		"weight_value": "9876543210.123456789012", "ratio_value": "0.123456789012345678",
	} {
		if got := detail[field]; got != want {
			t.Fatalf("detail %s=%#v (%T), want exact string %q", field, got, got, want)
		}
	}
	list, err := repository.Query(context.Background(), &request.Basic{Page: 1, Num: 10}, runtimeTable)
	if err != nil || len(list.Data) != 1 {
		t.Fatalf("generalization list=%+v err=%v", list, err)
	}
	if got := list.Data[0]["large_value"]; got != "12345678901234567890.1234567890" {
		t.Fatalf("list Decimal=%#v (%T)", got, got)
	}
}
