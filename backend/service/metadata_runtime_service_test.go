package service

import (
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/database"
	"backend/internal/datapermission"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMetadataRuntimeServiceReadsSafeStableProjectionAndRefreshesCache(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.SysTable{},
		&model.SysTableField{},
		&model.SysTableRelation{},
		&model.SysTableIndex{},
		&model.SysTableIndexField{},
	)
	if err := db.Exec(`CREATE TABLE runtime_order (id integer, name text, access_token text)`).Error; err != nil {
		t.Fatalf("create physical table: %v", err)
	}
	table := model.SysTable{
		Basic:     model.Basic{Id: 101, State: true},
		TableCode: "runtime_order",
		TableName: "运行订单",
		TableType: enum.System,
		SQL:       "SELECT secret_value FROM protected_source",
	}
	testutil.MustCreate(t, db, &table)
	fields := []model.SysTableField{
		metadataRuntimeTestField(202, table.Id, "name", "名称", 2),
		metadataRuntimeTestField(201, table.Id, "id", "ID", 1),
		metadataRuntimeTestField(203, table.Id, "access_token", "访问令牌", 3),
		metadataRuntimeTestField(204, table.Id, "disabled", "停用字段", 4),
	}
	fields[3].State = false
	unsafeTag := `gorm:"column:name;serializer:json"`
	fields[1].Tag = &unsafeTag
	for index := range fields {
		testutil.MustCreate(t, db, &fields[index])
	}
	if err := db.Model(&model.SysTableField{}).Where("id = ?", fields[3].Id).Update("state", false).Error; err != nil {
		t.Fatalf("disable field metadata: %v", err)
	}

	store := newJSONMemoryCacher()
	primaryDB := &database.PrimaryDB{DB: db}
	service := NewMetadataRuntimeService(
		impl.NewSysTableRepositoryImpl(primaryDB),
		impl.NewSysTableFieldRepositoryImpl(primaryDB),
		cache.NewSysTableCache(store),
		cache.NewSysTableFieldCache(store),
	)

	result, err := service.GetTable(context.Background(), table.TableCode)
	if err != nil {
		t.Fatalf("get runtime table: %v", err)
	}
	if len(result.Fields) != 2 || result.Fields[0].Code != "id" || result.Fields[1].Code != "name" {
		t.Fatalf("runtime field projection = %+v", result.Fields)
	}
	if result.Fields[0].ListVisible || !result.Fields[0].SystemManaged {
		t.Fatalf("managed field capabilities leaked: %+v", result.Fields[0])
	}
	runtimeResponse, err := service.GetTableResponse(context.Background(), table.TableCode)
	if err != nil {
		t.Fatalf("get runtime response: %v", err)
	}
	encoded, err := json.Marshal(runtimeResponse)
	if err != nil {
		t.Fatalf("marshal runtime response: %v", err)
	}
	payload := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"protected_source", "access_token", "serializer:json", `"sql"`, `"tag"`, "gmt_create"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("runtime response leaked %q: %s", forbidden, payload)
		}
	}
	if ok, err := service.HasPhysicalColumn(context.Background(), db, table.Id, "name"); err != nil || !ok {
		t.Fatalf("physical column check = %v, %v", ok, err)
	}
	if ok, err := service.HasPhysicalColumn(context.Background(), db, table.Id, "missing_column"); err != nil || ok {
		t.Fatalf("missing physical column check = %v, %v", ok, err)
	}

	if err := db.Model(&model.SysTableField{}).Where("id = ?", fields[1].Id).Update("field_name", "标识").Error; err != nil {
		t.Fatalf("update field metadata: %v", err)
	}
	cached, err := service.GetTable(context.Background(), table.TableCode)
	if err != nil || cached.Fields[0].DisplayName != "ID" {
		t.Fatalf("expected cached metadata before refresh: %+v, %v", cached, err)
	}
	service.Refresh(context.Background(), table.Id)
	refreshed, err := service.GetTable(context.Background(), table.TableCode)
	if err != nil || refreshed.Fields[0].DisplayName != "标识" {
		t.Fatalf("expected refreshed metadata: %+v, %v", refreshed, err)
	}
}

func TestMetadataRuntimeServiceImplementsDataPermissionReader(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysTable{}, &model.SysTableField{})
	table := model.SysTable{Basic: model.Basic{Id: 301, State: true}, TableCode: "permission_order", TableName: "权限订单", TableType: enum.System}
	field := metadataRuntimeTestField(302, table.Id, "owner_org_id", "归属组织", 1)
	field.IsAdvancedSearch = true
	testutil.MustCreate(t, db, &table)
	testutil.MustCreate(t, db, &field)
	primaryDB := &database.PrimaryDB{DB: db}
	service := NewMetadataRuntimeService(
		impl.NewSysTableRepositoryImpl(primaryDB),
		impl.NewSysTableFieldRepositoryImpl(primaryDB),
		nil,
		nil,
	)

	var reader datapermission.MetadataFieldReader = service
	tableRecord, err := reader.FindMetadataTable(context.Background(), table.Id)
	if err != nil || tableRecord.Id != table.Id || !tableRecord.State || tableRecord.Deleted {
		t.Fatalf("table record = %+v, %v", tableRecord, err)
	}
	fieldRecord, err := reader.FindMetadataField(context.Background(), field.Id)
	if err != nil || fieldRecord.Id != field.Id || fieldRecord.TableId != table.Id || !fieldRecord.IsAdvancedSearch {
		t.Fatalf("field record = %+v, %v", fieldRecord, err)
	}
}

func metadataRuntimeTestField(id, tableID int, code, name string, sequence uint8) model.SysTableField {
	return model.SysTableField{
		Basic:            model.Basic{Id: id, State: true},
		TableId:          tableID,
		FieldName:        name,
		FieldCode:        code,
		FieldType:        enum.VarcharFieldType,
		InputType:        enum.InputType,
		Sequence:         sequence,
		FieldCategory:    enum.NormalField,
		IsListShow:       true,
		IsQuickSearch:    true,
		IsAdvancedSearch: true,
	}
}
