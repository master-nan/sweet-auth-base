package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestSysTableManyToManyMetadataChangesPreservePostgreSQLData(t *testing.T) {
	db := openSysTableRelationPostgreSQL(t)
	svc := newSysTableRelationPostgreSQLService(t, db)
	mainTable := model.SysTable{Basic: model.Basic{Id: 101, State: true}, TableName: "Orders", TableCode: "pfcr_orders", TableType: enum.System}
	relatedTable := model.SysTable{Basic: model.Basic{Id: 102, State: true}, TableName: "Products", TableCode: "pfcr_products", TableType: enum.System}
	testutil.MustCreate(t, db, &mainTable)
	testutil.MustCreate(t, db, &relatedTable)
	testutil.MustCreate(t, db, &model.SysTableField{Basic: model.Basic{Id: 201, State: true}, TableId: mainTable.Id, FieldName: "Order ID", FieldCode: "order_id", FieldType: enum.BigIntFieldType})
	testutil.MustCreate(t, db, &model.SysTableField{Basic: model.Basic{Id: 202, State: true}, TableId: relatedTable.Id, FieldName: "Product ID", FieldCode: "product_id", FieldType: enum.BigIntFieldType})
	if err := db.Exec(`CREATE TABLE pfcr_order_product (order_id bigint NOT NULL, product_id bigint NOT NULL, PRIMARY KEY (order_id, product_id))`).Error; err != nil {
		t.Fatalf("create join table: %v", err)
	}
	if err := db.Exec(`INSERT INTO pfcr_order_product (order_id, product_id) VALUES (11, 22)`).Error; err != nil {
		t.Fatalf("seed join row: %v", err)
	}
	relation := model.SysTableRelation{
		Basic: model.Basic{Id: 301, State: true}, TableId: mainTable.Id, RelatedTableId: relatedTable.Id,
		ReferenceKey: "order_id", ForeignKey: "product_id", RelationType: enum.ManyToMany,
		ManyTableCode: "pfcr_order_product",
	}
	testutil.MustCreate(t, db, &relation)

	sameStructure := request.TableRelationUpdateReq{
		Id: relation.Id, TableId: mainTable.Id, RelatedTableId: relatedTable.Id,
		ReferenceKey: relation.ReferenceKey, ForeignKey: relation.ForeignKey,
		RelationType: relation.RelationType, ManyTableCode: relation.ManyTableCode,
	}
	if err := svc.UpdateTableRelation(context.Background(), sameStructure); err != nil {
		t.Fatalf("update relation metadata: %v", err)
	}
	assertJoinTableRow(t, db, "pfcr_order_product", 11, 22)

	dangerous := sameStructure
	dangerous.ManyTableCode = "pfcr_order_product_v2"
	if err := svc.UpdateTableRelation(context.Background(), dangerous); !errors.Is(err, apperrors.ErrTableRelationMigration) {
		t.Fatalf("dangerous relation change error = %v", err)
	}
	assertJoinTableRow(t, db, "pfcr_order_product", 11, 22)

	if err := svc.DeleteTableRelationById(context.Background(), relation.Id); err != nil {
		t.Fatalf("delete relation metadata: %v", err)
	}
	assertJoinTableRow(t, db, "pfcr_order_product", 11, 22)
}

func TestSysTableRelationRejectsMissingAndIncompatibleFields(t *testing.T) {
	db := openSysTableRelationPostgreSQL(t)
	svc := newSysTableRelationPostgreSQLService(t, db)
	mainTable := model.SysTable{Basic: model.Basic{Id: 501, State: true}, TableName: "Orders", TableCode: "pfcr_relation_orders", TableType: enum.System}
	relatedTable := model.SysTable{Basic: model.Basic{Id: 502, State: true}, TableName: "Customers", TableCode: "pfcr_relation_customers", TableType: enum.System}
	testutil.MustCreate(t, db, &mainTable)
	testutil.MustCreate(t, db, &relatedTable)
	testutil.MustCreate(t, db, &model.SysTableField{Basic: model.Basic{Id: 503, State: true}, TableId: mainTable.Id, FieldName: "Customer ID", FieldCode: "customer_id", FieldType: enum.BigIntFieldType})
	testutil.MustCreate(t, db, &model.SysTableField{Basic: model.Basic{Id: 504, State: true}, TableId: relatedTable.Id, FieldName: "Customer Code", FieldCode: "customer_code", FieldType: enum.VarcharFieldType})

	base := request.TableRelationCreateReq{
		TableId: mainTable.Id, RelatedTableId: relatedTable.Id,
		ReferenceKey: "customer_id", ForeignKey: "customer_code", RelationType: enum.ManyToOne,
	}
	if err := svc.CreateTableRelation(context.Background(), base); err == nil {
		t.Fatal("expected incompatible relation fields to be rejected")
	}
	base.ForeignKey = "missing_id"
	if err := svc.CreateTableRelation(context.Background(), base); err == nil {
		t.Fatal("expected missing relation field to be rejected")
	}
}

func TestRuntimeRelationOptionsReturnDisplayValues(t *testing.T) {
	db := openSysTableRelationPostgreSQL(t)
	svc := newSysTableRelationPostgreSQLService(t, db)
	source := model.SysTable{Basic: model.Basic{Id: 601, State: true}, TableName: "Orders", TableCode: "pfcr_runtime_orders", TableType: enum.System}
	target := model.SysTable{Basic: model.Basic{Id: 602, State: true}, TableName: "Customers", TableCode: "pfcr_runtime_customers", TableType: enum.System}
	testutil.MustCreate(t, db, &source)
	testutil.MustCreate(t, db, &target)
	linkage := `{"linkage":{"enabled":true,"mode":"relation","tableCode":"pfcr_runtime_customers","labelKey":"customer_name","valueKey":"id"}}`
	testutil.MustCreate(t, db, &model.SysTableField{
		Basic: model.Basic{Id: 603, State: true}, TableId: source.Id, FieldName: "Customer", FieldCode: "customer_id",
		FieldType: enum.BigIntFieldType, LogicalType: enum.LogicalTypeRelation, LinkageConfig: &linkage,
	})
	testutil.MustCreate(t, db, &model.SysTableField{Basic: model.Basic{Id: 604, State: true}, TableId: target.Id, FieldName: "ID", FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true})
	testutil.MustCreate(t, db, &model.SysTableField{Basic: model.Basic{Id: 605, State: true}, TableId: target.Id, FieldName: "Name", FieldCode: "customer_name", FieldType: enum.VarcharFieldType})
	if err := db.Exec(`CREATE TABLE pfcr_runtime_customers (id bigint PRIMARY KEY, customer_name varchar(64) NOT NULL)`).Error; err != nil {
		t.Fatalf("create relation target: %v", err)
	}
	for index := 1; index <= 25; index++ {
		if err := db.Exec(`INSERT INTO pfcr_runtime_customers (id, customer_name) VALUES (?, ?)`, 7100+index, fmt.Sprintf("Customer %02d", index)).Error; err != nil {
			t.Fatalf("seed paged relation target %d: %v", index, err)
		}
	}
	if err := db.Exec(`INSERT INTO pfcr_runtime_customers (id, customer_name) VALUES (7001, 'ZZZ Selected Customer')`).Error; err != nil {
		t.Fatalf("seed relation target: %v", err)
	}

	result, err := svc.metadataRuntime.QueryRelationOptions(context.Background(), 603, request.RuntimeRelationOptionsReq{Page: 1, Num: 20})
	if err != nil {
		t.Fatalf("query runtime relation options: %v", err)
	}
	if result.Total != 26 || len(result.Data) != 20 {
		t.Fatalf("unexpected runtime relation result: %+v", result)
	}
	for _, item := range result.Data {
		if item.Value == "7001" {
			t.Fatalf("selected relation value unexpectedly appeared on first page: %+v", result)
		}
	}

	selected, err := svc.metadataRuntime.QueryRelationOptions(context.Background(), 603, request.RuntimeRelationOptionsReq{
		Page: 1, Num: 20, SelectedValues: []string{"7001"},
	})
	if err != nil {
		t.Fatalf("query selected runtime relation option: %v", err)
	}
	if selected.Total != 1 || len(selected.Data) != 1 || selected.Data[0].Value != "7001" || selected.Data[0].Label != "ZZZ Selected Customer" {
		t.Fatalf("selected relation value was not resolved: %+v", selected)
	}
}

func TestSysTablePostgreSQLDefaultClearAndCompositeIndexOrder(t *testing.T) {
	db := openSysTableRelationPostgreSQL(t)
	svc := newSysTableRelationPostgreSQLService(t, db)
	table := model.SysTable{Basic: model.Basic{Id: 401, State: true}, TableName: "Index target", TableCode: "pfcr_index_target", TableType: enum.System}
	defaultValue := "pending"
	codeField := model.SysTableField{
		Basic: model.Basic{Id: 402, State: true}, TableId: table.Id, FieldName: "Code", FieldCode: "code",
		FieldType: enum.VarcharFieldType, FieldLength: 64, InputType: enum.InputType,
		DefaultValue: &defaultValue, IsNull: false, IsAdvancedSearch: true, IsSort: true,
		IsListShow: true, IsInsertShow: true, IsUpdateShow: true, Sequence: 1, FieldCategory: enum.NormalField,
	}
	nameField := model.SysTableField{
		Basic: model.Basic{Id: 403, State: true}, TableId: table.Id, FieldName: "Name", FieldCode: "name",
		FieldType: enum.VarcharFieldType, FieldLength: 64, InputType: enum.InputType,
		IsNull: false, IsAdvancedSearch: true, IsSort: true,
		IsListShow: true, IsInsertShow: true, IsUpdateShow: true, Sequence: 2, FieldCategory: enum.NormalField,
	}
	testutil.MustCreate(t, db, &table)
	testutil.MustCreate(t, db, &codeField)
	testutil.MustCreate(t, db, &nameField)
	if err := db.Exec(`CREATE TABLE pfcr_index_target (id bigint PRIMARY KEY, code varchar(64) NOT NULL DEFAULT 'pending', name varchar(64) NOT NULL)`).Error; err != nil {
		t.Fatalf("create physical table: %v", err)
	}
	if err := db.Exec(`CREATE INDEX idx_pfcr_name_code ON pfcr_index_target (name, code)`).Error; err != nil {
		t.Fatalf("create physical index: %v", err)
	}
	if err := svc.UpdateTableField(context.Background(), request.TableFieldUpdateReq{
		Id: codeField.Id, TableId: table.Id, FieldName: codeField.FieldName, FieldCode: codeField.FieldCode,
		FieldType: codeField.FieldType, FieldLength: codeField.FieldLength, InputType: codeField.InputType,
		IsNull: codeField.IsNull, IsAdvancedSearch: codeField.IsAdvancedSearch, IsSort: codeField.IsSort,
		IsListShow: codeField.IsListShow, IsInsertShow: codeField.IsInsertShow, IsUpdateShow: codeField.IsUpdateShow,
		Sequence: int(codeField.Sequence), FieldCategory: codeField.FieldCategory,
	}); err != nil {
		t.Fatalf("clear column default: %v", err)
	}
	var physicalDefault *string
	if err := db.Raw(`SELECT column_default FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'pfcr_index_target' AND column_name = 'code'`).Scan(&physicalDefault).Error; err != nil {
		t.Fatalf("read physical default: %v", err)
	}
	if physicalDefault != nil {
		t.Fatalf("physical default = %q, want NULL", *physicalDefault)
	}
	var storedField model.SysTableField
	if err := db.First(&storedField, codeField.Id).Error; err != nil {
		t.Fatalf("reload field metadata: %v", err)
	}
	if storedField.DefaultValue != nil {
		t.Fatalf("metadata default = %q, want nil", *storedField.DefaultValue)
	}

	var schemaName string
	if err := db.Raw("SELECT current_schema()").Scan(&schemaName).Error; err != nil {
		t.Fatalf("read current schema: %v", err)
	}
	metadata, err := impl.NewSysTableRepositoryImpl(&database.PrimaryDB{DB: db}).FetchTableIndexMetadata(context.Background(), db, schemaName, table.TableCode)
	if err != nil {
		t.Fatalf("read composite index metadata: %v", err)
	}
	if len(metadata) != 2 || metadata[0].ColumnName != "name" || metadata[0].OrdinalPosition != 1 || metadata[1].ColumnName != "code" || metadata[1].OrdinalPosition != 2 {
		t.Fatalf("composite index order = %+v", metadata)
	}
}

func assertJoinTableRow(t *testing.T, db *gorm.DB, table string, orderID, productID int64) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where("order_id = ? AND product_id = ?", orderID, productID).Count(&count).Error; err != nil {
		t.Fatalf("query join row: %v", err)
	}
	if count != 1 {
		t.Fatalf("join row count = %d, want 1", count)
	}
}

func newSysTableRelationPostgreSQLService(t *testing.T, db *gorm.DB) *SysTableService {
	t.Helper()
	primaryDB := &database.PrimaryDB{DB: db}
	tableRepo := impl.NewSysTableRepositoryImpl(primaryDB)
	fieldRepo := impl.NewSysTableFieldRepositoryImpl(primaryDB)
	store := newJSONMemoryCacher()
	metadataRuntime := NewMetadataRuntimeService(
		tableRepo, fieldRepo, cache.NewSysTableCache(store), cache.NewSysTableFieldCache(store),
	)
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	return NewSysTableService(
		tableRepo, fieldRepo, impl.NewSysTableIndexRepositoryImpl(primaryDB),
		impl.NewSysTableIndexFieldRepositoryImpl(primaryDB), impl.NewSysTableRelationRepositoryImpl(primaryDB),
		sf, metadataRuntime,
	)
}

func openSysTableRelationPostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := testutil.OpenPostgres(t, postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("sys_table_relation_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf("CREATE SCHEMA %q", schemaName)).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schemaName)).Error })
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	parsed.RawQuery = query.Encode()
	db, err := testutil.OpenPostgres(t, postgres.Open(parsed.String()), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated schema: %v", err)
	}
	if err := db.AutoMigrate(&model.SysTable{}, &model.SysTableField{}, &model.SysTableRelation{}, &model.SysTableIndex{}, &model.SysTableIndexField{}); err != nil {
		t.Fatalf("migrate metadata: %v", err)
	}
	return db
}
