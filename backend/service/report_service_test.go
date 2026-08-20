package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/database"
	"backend/internal/reportconfig"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestReportCreateDerivesSourceFromPrimaryDataset(t *testing.T) {
	svc := newReportServiceForConfigTest(t)
	report, err := svc.reportFromCreateReq(request.ReportDefinitionCreateReq{
		Code:         "sales_summary",
		Name:         "销售汇总",
		QueryConfig:  datatypes.JSON([]byte(`{"datasets":[{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true}],"fields":[],"parameters":[]}`)),
		LayoutConfig: datatypes.JSON([]byte(`{"view":"sheet","datasets":[{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true}],"parameters":[],"sheet":{"rows":8,"cols":6,"cells":[]}}`)),
	})
	if err != nil {
		t.Fatalf("create report from request: %v", err)
	}
	if report.SourceType != reportSourceTypeTable || report.SourceCode != "demo_order" {
		t.Fatalf("unexpected source: type=%q code=%q", report.SourceType, report.SourceCode)
	}
	if report.PermissionTableCode != "demo_order" {
		t.Fatalf("permission table = %q, want demo_order", report.PermissionTableCode)
	}
}

func TestReportCreateWithoutPrimaryTableDatasetFails(t *testing.T) {
	svc := newReportServiceForConfigTest(t)
	_, err := svc.reportFromCreateReq(request.ReportDefinitionCreateReq{
		Code:         "sales_summary",
		Name:         "销售汇总",
		QueryConfig:  datatypes.JSON([]byte(`{"datasets":[{"id":"sql_1","name":"SQL","type":"sql","sql":"select 1","primary":true}],"fields":[],"parameters":[]}`)),
		LayoutConfig: datatypes.JSON([]byte(`{"view":"sheet","datasets":[{"id":"sql_1","name":"SQL","type":"sql","sql":"select 1","primary":true}],"parameters":[],"sheet":{"rows":8,"cols":6,"cells":[]}}`)),
	})
	if err == nil || !strings.Contains(err.Error(), "primary table") {
		t.Fatalf("expected primary table dataset error, got %v", err)
	}
}

func TestReportCreateRejectsMissingJoinedTableDataset(t *testing.T) {
	svc := newReportServiceForConfigTest(t)
	_, err := svc.reportFromCreateReq(request.ReportDefinitionCreateReq{
		Code: "sales_summary",
		Name: "销售汇总",
		QueryConfig: datatypes.JSON([]byte(`{
			"datasets":[
				{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true},
				{"id":"company","name":"公司","type":"table","source_code":"missing_company"}
			],
			"dataset_joins":[{"id":"j1","left_dataset_id":"main","left_field":"company_id","right_dataset_id":"company","right_field":"id","join_type":"left"}]
		}`)),
		LayoutConfig: datatypes.JSON([]byte(`{"view":"sheet","sheet":{"rows":8,"cols":6,"cells":[]}}`)),
	})
	if err == nil || !strings.Contains(err.Error(), "数据集表不存在") {
		t.Fatalf("expected missing dataset table error, got %v", err)
	}
}

func TestReportSQLPreviewRejectsDangerousSQL(t *testing.T) {
	tests := []string{
		"delete from demo_order",
		"select * from demo_order; drop table demo_order",
		"with x as (update demo_order set name = 'x' returning *) select * from x",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if _, err := safeReportPreviewSQL(tt); err == nil {
				t.Fatalf("expected dangerous SQL to be rejected")
			}
		})
	}
}

func TestReportConfigParsesTableDataset(t *testing.T) {
	config, err := reportconfig.Parse(
		datatypes.JSON([]byte(`{"datasets":[{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true,"fields":[{"code":"amount","name":"金额","type":"number"}]}],"fields":[{"code":"amount","name":"金额","type":"number"}],"parameters":[{"id":"p1","field":"amount","type":"number","operator":"gte"}]}`)),
		datatypes.JSON([]byte(`{"view":"sheet","datasets":[{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true,"fields":[{"code":"amount","name":"金额","type":"number"}]}],"parameters":[],"sheet":{"rows":8,"cols":6,"cells":[{"id":"A1","row":1,"col":1,"value":"金额","binding":{"type":"field","dataset_id":"main","field":"amount"}}]}}`)),
	)
	if err != nil {
		t.Fatalf("parse report config: %v", err)
	}
	dataset, ok := config.PrimaryTableDataset()
	if !ok {
		t.Fatalf("primary table dataset not found")
	}
	if dataset.Id != "main" || dataset.SourceCode != "demo_order" || len(dataset.Fields) != 1 {
		t.Fatalf("unexpected dataset: %#v", dataset)
	}
	meta := reportConfigDatasetMetadata(config)
	if len(meta) != 1 || len(meta[0].Fields) != 1 || meta[0].Fields[0].Field != "amount" {
		t.Fatalf("unexpected dataset metadata: %#v", meta)
	}
	joins := reportConfigDatasetJoins(config)
	if len(joins) != 0 {
		t.Fatalf("unexpected joins: %#v", joins)
	}
}

func TestApplyReportParameterValues(t *testing.T) {
	config, err := reportconfig.Parse(
		datatypes.JSON([]byte(`{
			"datasets":[{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true}],
			"parameters":[
				{"id":"customer","dataset_id":"main","field":"customer_name","operator":"like"},
				{"id":"status","dataset_id":"main","field":"status","operator":"eq"},
				{"id":"created","dataset_id":"main","field":"gmt_create","operator":"between"}
			]
		}`)),
		datatypes.JSON([]byte(`{"view":"sheet"}`)),
	)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	query := request.Basic{}
	err = applyReportParameterValues(&query, config, "main", map[string]any{
		"customer": "华南",
		"status":   "done",
		"created":  []any{"2026-07-01 00:00:00", "2026-07-31 23:59:59"},
	})
	if err != nil {
		t.Fatalf("apply parameters: %v", err)
	}
	if query.Filters["status"] != "done" {
		t.Fatalf("status filter not applied: %#v", query.Filters)
	}
	if len(query.Expressions) != 1 || len(query.Expressions[0].Rules) != 2 {
		t.Fatalf("unexpected expressions: %#v", query.Expressions)
	}
	if query.Expressions[0].Rules[0].ExpressionType != enum.Like || query.Expressions[0].Rules[0].Field != "customer_name" {
		t.Fatalf("like parameter not applied: %#v", query.Expressions[0].Rules[0])
	}
	if query.Expressions[0].Rules[1].ExpressionType != enum.Between || query.Expressions[0].Rules[1].Field != "gmt_create" {
		t.Fatalf("between parameter not applied: %#v", query.Expressions[0].Rules[1])
	}
}

func TestReportSQLParameterWhere(t *testing.T) {
	config, err := reportconfig.Parse(
		datatypes.JSON([]byte(`{
			"datasets":[{"id":"sql_1","name":"SQL","type":"sql","sql":"select id, customer_name, amount from demo_order"}],
			"parameters":[
				{"id":"customer","dataset_id":"sql_1","field":"customer_name","operator":"like"},
				{"id":"amount","dataset_id":"sql_1","field":"amount","operator":"gte"}
			]
		}`)),
		datatypes.JSON([]byte(`{"view":"sheet"}`)),
	)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	where, args, err := reportSQLParameterWhere(config, "sql_1", map[string]any{
		"customer": "华南",
		"amount":   100,
	})
	if err != nil {
		t.Fatalf("build sql where: %v", err)
	}
	if where != `CAST("customer_name" AS TEXT) ILIKE ? AND "amount" >= ?` {
		t.Fatalf("unexpected where: %s", where)
	}
	if len(args) != 2 || args[0] != "%华南%" || args[1] != 100 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestReportJoinedParameterLikeCastsNumericField(t *testing.T) {
	config, err := reportconfig.Parse(
		datatypes.JSON([]byte(`{
			"datasets":[{"id":"main","name":"按钮","type":"table","source_code":"sys_menu_button","primary":true}],
			"parameters":[{"id":"id","dataset_id":"main","field":"id","operator":"like"}]
		}`)),
		datatypes.JSON([]byte(`{"view":"sheet"}`)),
	)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	table := model.SysTable{
		TableCode: "sys_menu_button",
		TableFields: []model.SysTableField{
			{FieldName: "ID", FieldCode: "id", FieldType: enum.BigIntFieldType, IsListShow: true},
		},
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	query := db.Table(table.TableCode)
	query, err = reportApplyJoinedParameters(query, config, "main", table.TableCode, map[string]model.SysTable{"main": table}, reportDatasetAliases(config, "main", table.TableCode), map[string]any{"id": "33"})
	if err != nil {
		t.Fatalf("apply joined params: %v", err)
	}
	stmt := query.Find(&[]map[string]any{}).Statement
	if !strings.Contains(stmt.SQL.String(), "CAST(") || !strings.Contains(stmt.SQL.String(), "ILIKE") {
		t.Fatalf("numeric like should cast field to text: %s", stmt.SQL.String())
	}
}

func TestReportJoinedPreviewSelectionsUsesSheetBindings(t *testing.T) {
	primaryTable := model.SysTable{
		TableCode: "demo_order",
		TableFields: []model.SysTableField{
			{FieldName: "订单号", FieldCode: "order_no", FieldType: enum.VarcharFieldType, IsListShow: true},
			{FieldName: "公司ID", FieldCode: "company_id", FieldType: enum.BigIntFieldType, IsListShow: true},
		},
	}
	companyTable := model.SysTable{
		TableCode: "demo_company",
		TableFields: []model.SysTableField{
			{FieldName: "ID", FieldCode: "id", FieldType: enum.BigIntFieldType, IsListShow: true},
			{FieldName: "公司名称", FieldCode: "company_name", FieldType: enum.VarcharFieldType, IsListShow: true},
		},
	}
	config, err := reportconfig.Parse(
		datatypes.JSON([]byte(`{
			"datasets":[
				{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true},
				{"id":"company","name":"公司","type":"table","source_code":"demo_company"}
			],
			"dataset_joins":[{"id":"j1","left_dataset_id":"main","left_field":"company_id","right_dataset_id":"company","right_field":"id","join_type":"left"}]
		}`)),
		datatypes.JSON([]byte(`{
			"view":"sheet",
			"sheet":{"rows":8,"cols":6,"cells":[
				{"id":"A1","row":1,"col":1,"value":"订单号","binding":{"type":"field","dataset_id":"main","field":"order_no"}},
				{"id":"B1","row":1,"col":2,"value":"公司名称","binding":{"type":"group","dataset_id":"company","field":"company_name"}}
			]}
		}`)),
	)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	aliases := reportDatasetAliases(config, "main", "demo_order")
	joinSQL, err := reportJoinSQL(config.DatasetJoins()[0], "main", "demo_order", map[string]reportconfig.Dataset{
		"main":    config.Datasets()[0],
		"company": config.Datasets()[1],
	}, map[string]model.SysTable{
		"main":    primaryTable,
		"company": companyTable,
	}, aliases)
	if err != nil {
		t.Fatalf("build join sql: %v", err)
	}
	if !strings.Contains(joinSQL, `LEFT JOIN "demo_company" AS "rds_2"`) || !strings.Contains(joinSQL, `"demo_order"."company_id" = "rds_2"."id"`) {
		t.Fatalf("unexpected join sql: %s", joinSQL)
	}
	selections, columns, err := reportJoinedPreviewSelections(config, "main", primaryTable, map[string]model.SysTable{
		"main":    primaryTable,
		"company": companyTable,
	}, aliases)
	if err != nil {
		t.Fatalf("build selections: %v", err)
	}
	if len(selections) != 2 || len(columns) != 2 {
		t.Fatalf("unexpected selections=%#v columns=%#v", selections, columns)
	}
	if columns[1].Field != "company__company_name" || columns[1].Label != "公司名称" {
		t.Fatalf("joined field column not mapped: %#v", columns[1])
	}
}

func TestReportDataSourceColumnsIncludesSyntheticID(t *testing.T) {
	table := model.SysTable{
		TableCode: "sys_menu",
		TableFields: []model.SysTableField{
			{FieldName: "父菜单ID", FieldCode: "pid", FieldType: enum.BigIntFieldType, IsListShow: true},
			{FieldName: "显示标题", FieldCode: "title", FieldType: enum.VarcharFieldType, IsListShow: true},
		},
	}
	columns := reportDataSourceColumns(table)
	if len(columns) == 0 || columns[0].Field != "id" {
		t.Fatalf("synthetic id column missing: %#v", columns)
	}
	if _, ok := reportFindTableField(table, "id"); !ok {
		t.Fatal("synthetic id should be accepted by table field lookup")
	}
}

func TestReportDataSourceColumnsIncludeNonListFields(t *testing.T) {
	table := model.SysTable{
		TableCode: "sys_menu_button",
		TableFields: []model.SysTableField{
			{FieldName: "按钮名称", FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true},
			{FieldName: "菜单ID", FieldCode: "menu_id", FieldType: enum.BigIntFieldType, IsListShow: false},
		},
	}
	columns := reportDataSourceColumns(table)
	if !hasReportColumn(columns, "menu_id") {
		t.Fatalf("report dataset fields should include non-list menu_id: %#v", columns)
	}
	if !hasReportColumn(reportPreviewColumns(table), "menu_id") {
		t.Fatalf("report preview fallback should include non-list menu_id")
	}
	config, err := reportconfig.Parse(
		datatypes.JSON([]byte(`{"datasets":[{"id":"main","name":"按钮","type":"table","source_code":"sys_menu_button","primary":true}]}`)),
		datatypes.JSON([]byte(`{"view":"sheet"}`)),
	)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	_, previewColumns, err := reportJoinedPreviewSelections(config, "main", table, map[string]model.SysTable{"main": table}, reportDatasetAliases(config, "main", table.TableCode))
	if err != nil {
		t.Fatalf("build preview selections: %v", err)
	}
	if !hasReportColumn(previewColumns, "main__menu_id") {
		t.Fatalf("joined preview fallback should include non-list menu_id: %#v", previewColumns)
	}
}

func TestReportTableWithPreviewFieldsEnablesConfiguredNonListFields(t *testing.T) {
	table := model.SysTable{
		TableCode: "sys_menu_button",
		TableFields: []model.SysTableField{
			{FieldName: "按钮名称", FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true},
			{FieldName: "菜单ID", FieldCode: "menu_id", FieldType: enum.BigIntFieldType, IsListShow: false},
		},
	}
	config, err := reportconfig.Parse(
		datatypes.JSON([]byte(`{"datasets":[{"id":"main","name":"按钮","type":"table","source_code":"sys_menu_button","primary":true,"fields":[{"name":"菜单ID","code":"menu_id"}]}]}`)),
		datatypes.JSON([]byte(`{"view":"sheet","sheet":{"cells":[{"row":1,"col":1,"binding":{"type":"group","dataset_id":"main","field":"menu_id"}}]}}`)),
	)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	result := reportTableWithPreviewFields(table, config, "main")
	field, ok := reportFindTableField(result, "menu_id")
	if !ok || !field.IsListShow {
		t.Fatalf("configured non-list field should be selected in report preview: %#v", result.TableFields)
	}
}

func TestInferSQLFields(t *testing.T) {
	svc := newReportServiceForConfigTest(t)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	fields, err := svc.InferSQLFields(ctx, request.ReportSQLFieldsReq{
		SQL: "select id, name, amount from demo_order",
	})
	if err != nil {
		t.Fatalf("infer sql fields: %v", err)
	}
	for _, code := range []string{"id", "name", "amount"} {
		if !hasReportColumn(fields, code) {
			t.Fatalf("missing inferred column %s in %#v", code, fields)
		}
	}
}

func hasReportColumn(columns []response.ReportPreviewColumn, field string) bool {
	for _, column := range columns {
		if column.Field == field {
			return true
		}
	}
	return false
}

func newReportServiceForConfigTest(t *testing.T) *ReportService {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.SysTable{}, &model.SysTableField{}, &model.SysTableRelation{}, &model.SysTableIndex{}, &model.SysTableIndexField{})
	table := model.SysTable{
		Basic:     model.Basic{Id: 1, State: true},
		TableName: "订单",
		TableCode: "demo_order",
		TableType: enum.System,
	}
	if err := db.Create(&table).Error; err != nil {
		t.Fatalf("seed table: %v", err)
	}
	fields := []model.SysTableField{
		{Basic: model.Basic{Id: 10, State: true}, TableId: table.Id, FieldName: "ID", FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true, IsListShow: true, Sequence: 1},
		{Basic: model.Basic{Id: 11, State: true}, TableId: table.Id, FieldName: "名称", FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true, Sequence: 2},
		{Basic: model.Basic{Id: 12, State: true}, TableId: table.Id, FieldName: "金额", FieldCode: "amount", FieldType: enum.DecimalFieldType, NumericPrecision: 18, NumericScale: 2, IsListShow: true, Sequence: 3},
	}
	if err := db.Create(&fields).Error; err != nil {
		t.Fatalf("seed fields: %v", err)
	}
	if err := db.Exec(`CREATE TABLE demo_order (id integer, name text, amount real)`).Error; err != nil {
		t.Fatalf("create physical demo_order: %v", err)
	}
	primaryDB := &database.PrimaryDB{DB: db}
	store := newJSONMemoryCacher()
	metadataRuntime := NewMetadataRuntimeService(
		impl.NewSysTableRepositoryImpl(primaryDB),
		impl.NewSysTableFieldRepositoryImpl(primaryDB),
		cache.NewSysTableCache(store),
		cache.NewSysTableFieldCache(store),
	)
	return &ReportService{
		reportRepo:      impl.NewReportDefinitionRepositoryImpl(primaryDB),
		metadataRuntime: metadataRuntime,
	}
}
