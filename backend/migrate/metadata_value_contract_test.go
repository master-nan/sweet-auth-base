package main

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	repositoryimpl "backend/repository/impl"
	"context"
	"fmt"
	"testing"
)

func TestMetadataValueContractPostgres16AndDecimalRoundTrip(t *testing.T) {
	db, cleanup := openQuerySchemePostgresSchema(t)
	defer cleanup()
	if err := db.AutoMigrate(&model.SysDict{}, &model.SysDictItem{}, &model.SysTable{}, &model.SysTableField{}); err != nil {
		t.Fatalf("auto migrate metadata prerequisites: %v", err)
	}
	table := model.SysTable{Basic: model.Basic{Id: 8101, State: true}, TableName: "精确数值测试", TableCode: "decimal_round_trip"}
	if err := db.Create(&table).Error; err != nil {
		t.Fatalf("create table metadata: %v", err)
	}
	dict := model.SysDict{Basic: model.Basic{Id: 8110, State: true}, DictName: "字段类型", DictCode: "sys_table_field_type"}
	if err := db.Create(&dict).Error; err != nil {
		t.Fatalf("create field type dict: %v", err)
	}
	legacyItems := []model.SysDictItem{
		{Basic: model.Basic{Id: 8111, State: true}, DictId: dict.Id, ItemName: "removed numeric", ItemCode: "sys_table_field_type_float", ItemValue: "2"},
		{Basic: model.Basic{Id: 8112, State: true}, DictId: dict.Id, ItemName: "removed integer", ItemCode: "sys_table_field_type_tinyint", ItemValue: "9"},
		{Basic: model.Basic{Id: 8117, State: true}, DictId: dict.Id, ItemName: "previous smallint", ItemCode: "sys_table_field_type_smallint", ItemValue: "12"},
		{Basic: model.Basic{Id: 8118, State: true}, DictId: dict.Id, ItemName: "previous decimal", ItemCode: "sys_table_field_type_decimal", ItemValue: "13"},
	}
	if err := db.Create(&legacyItems).Error; err != nil {
		t.Fatalf("create removed field type items: %v", err)
	}
	legacyDecimal := model.SysTableField{Basic: model.Basic{Id: 8113, State: true}, TableId: table.Id, FieldName: "旧小数", FieldCode: "legacy_decimal", FieldType: enum.DecimalFieldType, FieldLength: 18, FieldDecimalLength: 4}
	legacySmallInt := model.SysTableField{Basic: model.Basic{Id: 8114, State: true}, TableId: table.Id, FieldName: "旧小整数", FieldCode: "legacy_smallint", FieldType: enum.SmallIntFieldType}
	legacyNarrowDecimal := model.SysTableField{Basic: model.Basic{Id: 8115, State: true}, TableId: table.Id, FieldName: "旧低精度小数", FieldCode: "legacy_narrow_decimal", FieldType: enum.DecimalFieldType, FieldLength: 4}
	legacyScaleZeroDecimal := model.SysTableField{Basic: model.Basic{Id: 8116, State: true}, TableId: table.Id, FieldName: "零位小数", FieldCode: "scale_zero_decimal", FieldType: enum.DecimalFieldType, NumericPrecision: 8, NumericScale: 0}
	previousDecimal := model.SysTableField{Basic: model.Basic{Id: 8119, State: true}, TableId: table.Id, FieldName: "过渡小数", FieldCode: "previous_decimal", FieldType: enum.SysTableFieldType(historicalDecimalFieldType), NumericPrecision: 20, NumericScale: 6}
	previousSmallInt := model.SysTableField{Basic: model.Basic{Id: 8120, State: true}, TableId: table.Id, FieldName: "过渡小整数", FieldCode: "previous_smallint", FieldType: enum.SysTableFieldType(historicalSmallIntFieldType)}
	if err := db.Create(&[]model.SysTableField{legacyDecimal, legacySmallInt, legacyNarrowDecimal, legacyScaleZeroDecimal, previousDecimal, previousSmallInt}).Error; err != nil {
		t.Fatalf("create removed metadata rows: %v", err)
	}
	if err := db.Delete(&legacySmallInt).Error; err != nil {
		t.Fatalf("soft delete removed small integer metadata: %v", err)
	}
	if err := migrateMetadataValueContract(db); err != nil {
		t.Fatalf("first metadata value migration: %v", err)
	}
	if err := migrateMetadataValueContract(db); err != nil {
		t.Fatalf("second metadata value migration: %v", err)
	}
	var migrated []model.SysTableField
	if err := db.Unscoped().Where("id IN ?", []int{legacyDecimal.Id, legacySmallInt.Id, legacyNarrowDecimal.Id, legacyScaleZeroDecimal.Id, previousDecimal.Id, previousSmallInt.Id}).Order("id").Find(&migrated).Error; err != nil {
		t.Fatalf("read migrated metadata: %v", err)
	}
	if len(migrated) != 6 || migrated[0].FieldType != enum.DecimalFieldType || migrated[0].NumericPrecision != 18 || migrated[0].NumericScale != 4 || migrated[1].FieldType != enum.SmallIntFieldType || migrated[2].NumericPrecision != 4 || migrated[2].NumericScale != 4 || migrated[3].NumericPrecision != 8 || migrated[3].NumericScale != 0 || migrated[4].FieldType != enum.DecimalFieldType || migrated[4].NumericPrecision != 20 || migrated[4].NumericScale != 6 || migrated[5].FieldType != enum.SmallIntFieldType {
		t.Fatalf("removed metadata was not migrated: %+v", migrated)
	}
	if err := seedDicts(db, newMigrationTestSnowflake(t)); err != nil {
		t.Fatalf("seed canonical dictionaries: %v", err)
	}
	var removedItems int64
	if err := db.Unscoped().Model(&model.SysDictItem{}).Where("item_code IN ?", []string{"sys_table_field_type_float", "sys_table_field_type_tinyint"}).Count(&removedItems).Error; err != nil || removedItems != 0 {
		t.Fatalf("removed field type items count=%d err=%v", removedItems, err)
	}
	var canonicalItems []model.SysDictItem
	if err := db.Where("dict_id = ?", dict.Id).Order("item_value::integer").Find(&canonicalItems).Error; err != nil {
		t.Fatalf("query canonical field type items: %v", err)
	}
	if len(canonicalItems) != 11 {
		t.Fatalf("canonical field type item count=%d", len(canonicalItems))
	}
	for index, item := range canonicalItems {
		if item.ItemValue != fmt.Sprintf("%d", index+1) {
			t.Fatalf("canonical item[%d]=%+v", index, item)
		}
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
	for index, removedType := range []enum.SysTableFieldType{historicalSmallIntFieldType, historicalDecimalFieldType} {
		field := model.SysTableField{Basic: model.Basic{Id: 8210 + index, State: true}, TableId: table.Id, FieldName: "已移除类型", FieldCode: fmt.Sprintf("removed_%d", removedType), FieldType: removedType}
		if err := db.Create(&field).Error; err == nil {
			t.Fatalf("PostgreSQL must reject removed storage type %d", removedType)
		}
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

func TestMetadataValueContractReplacesPreviousCanonicalCheck(t *testing.T) {
	db, cleanup := openQuerySchemePostgresSchema(t)
	defer cleanup()
	if err := db.AutoMigrate(&model.SysDict{}, &model.SysDictItem{}, &model.SysTable{}, &model.SysTableField{}); err != nil {
		t.Fatalf("auto migrate metadata prerequisites: %v", err)
	}
	table := model.SysTable{Basic: model.Basic{Id: 8301, State: true}, TableName: "过渡编号", TableCode: "previous_field_types"}
	if err := db.Create(&table).Error; err != nil {
		t.Fatalf("create table metadata: %v", err)
	}
	fields := []model.SysTableField{
		{Basic: model.Basic{Id: 8302, State: true}, TableId: table.Id, FieldName: "小整数", FieldCode: "small_value", FieldType: enum.SysTableFieldType(historicalSmallIntFieldType)},
		{Basic: model.Basic{Id: 8303, State: true}, TableId: table.Id, FieldName: "精确数值", FieldCode: "amount", FieldType: enum.SysTableFieldType(historicalDecimalFieldType), NumericPrecision: 18, NumericScale: 0},
	}
	if err := db.Create(&fields).Error; err != nil {
		t.Fatalf("create previous canonical rows: %v", err)
	}
	if err := db.Exec(`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_storage_type CHECK
		(field_type IN (1,3,4,5,6,7,8,10,11,12,13))`).Error; err != nil {
		t.Fatalf("add previous canonical check: %v", err)
	}
	if err := migrateMetadataValueContract(db); err != nil {
		t.Fatalf("migrate through previous canonical check: %v", err)
	}
	var migrated []model.SysTableField
	if err := db.Where("id IN ?", []int{8302, 8303}).Order("id").Find(&migrated).Error; err != nil {
		t.Fatalf("read canonical rows: %v", err)
	}
	if len(migrated) != 2 || migrated[0].FieldType != enum.SmallIntFieldType || migrated[1].FieldType != enum.DecimalFieldType || migrated[1].NumericScale != 0 {
		t.Fatalf("canonical rows=%+v", migrated)
	}
}
