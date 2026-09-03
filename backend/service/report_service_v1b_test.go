package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/reportconfig"
	"backend/model"
	"backend/repository"
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestReportOrderedDatasetJoinsOrdersAndOrientsFromPrimary(t *testing.T) {
	config := reportconfig.Config{Query: reportconfig.QueryConfig{DatasetJoins: []reportconfig.DatasetJoin{
		{
			Id:             "company_region",
			LeftDatasetId:  "region",
			RightDatasetId: "company",
			Conditions: []reportconfig.DatasetJoinCondition{
				{LeftField: "id", RightField: "region_id"},
			},
		},
		{
			Id:             "order_company",
			LeftDatasetId:  "main",
			RightDatasetId: "company",
			Conditions: []reportconfig.DatasetJoinCondition{
				{LeftField: "company_id", RightField: "id"},
			},
		},
	}}}

	joins, err := reportOrderedDatasetJoins(config, "main")
	if err != nil {
		t.Fatalf("order dataset joins: %v", err)
	}
	if len(joins) != 2 || joins[0].Id != "order_company" || joins[1].Id != "company_region" {
		t.Fatalf("unexpected join order: %#v", joins)
	}
	if joins[1].LeftDatasetId != "company" || joins[1].Conditions[0].LeftField != "region_id" || joins[1].RightDatasetId != "region" || joins[1].Conditions[0].RightField != "id" {
		t.Fatalf("second join should point from the connected dataset to the new dataset: %#v", joins[1])
	}
}

func TestReportOrderedDatasetJoinsRejectsInvalidGraphs(t *testing.T) {
	tests := []struct {
		name  string
		joins []reportconfig.DatasetJoin
	}{
		{
			name: "disconnected",
			joins: []reportconfig.DatasetJoin{
				{Id: "main_company", LeftDatasetId: "main", RightDatasetId: "company"},
				{Id: "region_department", LeftDatasetId: "region", RightDatasetId: "department"},
			},
		},
		{
			name: "cycle",
			joins: []reportconfig.DatasetJoin{
				{Id: "main_company", LeftDatasetId: "main", RightDatasetId: "company"},
				{Id: "company_region", LeftDatasetId: "company", RightDatasetId: "region"},
				{Id: "region_main", LeftDatasetId: "region", RightDatasetId: "main"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := reportconfig.Config{Query: reportconfig.QueryConfig{DatasetJoins: tt.joins}}
			if _, err := reportOrderedDatasetJoins(config, "main"); err == nil || !strings.Contains(err.Error(), "断开或循环") {
				t.Fatalf("invalid join graph should be rejected, got %v", err)
			}
		})
	}
}

func TestReportV1BExportStatusRulesAndLogs(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "v1b_status_rules", reportStatusDraft, queryConfig, layoutConfig)

	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{}); err == nil || !strings.Contains(err.Error(), "未发布") {
		t.Fatalf("draft report should not export, got %v", err)
	}
	draftLog := env.lastExecutionLog(t, reportRuntimeExport)
	if draftLog.Success {
		t.Fatalf("draft export failure log should be unsuccessful: %#v", draftLog)
	}
	assertReportRuntimeLog(t, draftLog, reportRuntimeExport, 0, 0)

	disabled := env.createReport(t, "v1b_disabled", reportStatusDisabled, queryConfig, layoutConfig)
	if _, err := env.svc.ExportReport(env.ctx, disabled.Id, request.ReportExportReq{}); err == nil || !strings.Contains(err.Error(), "已停用") {
		t.Fatalf("disabled report should not export, got %v", err)
	}
	disabledLog := env.lastExecutionLog(t, reportRuntimeExport)
	if disabledLog.Success {
		t.Fatalf("disabled export failure log should be unsuccessful: %#v", disabledLog)
	}
}

func TestReportV1BExportUsesPublishedVersionSnapshot(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	nameQuery, nameLayout := reportV1ATableConfig("name")
	report := env.createReport(t, "v1b_snapshot", reportStatusDraft, nameQuery, nameLayout)

	published1, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{ChangeLog: "publish name"})
	if err != nil {
		t.Fatalf("publish report first version: %v", err)
	}
	file1, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{})
	if err != nil {
		t.Fatalf("export first published version: %v", err)
	}
	records1 := parseReportCSV(t, file1.Content)
	assertCSVHeader(t, records1, "名称")
	if file1.RowCount != 2 || len(records1) != 3 {
		t.Fatalf("unexpected first export rows: rowCount=%d records=%#v", file1.RowCount, records1)
	}

	amountQuery, amountLayout := reportV1ATableConfig("amount")
	if err := env.db.Model(&model.ReportDefinition{}).
		Where("id = ?", report.Id).
		Updates(map[string]any{
			"query_config":  amountQuery,
			"layout_config": amountLayout,
		}).Error; err != nil {
		t.Fatalf("modify draft config after publish: %v", err)
	}
	fileAfterDraftEdit, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{})
	if err != nil {
		t.Fatalf("export after draft edit: %v", err)
	}
	recordsAfterDraftEdit := parseReportCSV(t, fileAfterDraftEdit.Content)
	assertCSVHeader(t, recordsAfterDraftEdit, "名称")

	published2, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{ChangeLog: "publish amount"})
	if err != nil {
		t.Fatalf("publish report second version: %v", err)
	}
	file2, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{})
	if err != nil {
		t.Fatalf("export second published version: %v", err)
	}
	records2 := parseReportCSV(t, file2.Content)
	assertCSVHeader(t, records2, "金额")
	if !strings.Contains(file2.FileName, "_v2.csv") {
		t.Fatalf("export file should include version: %s", file2.FileName)
	}
	exportLog := env.lastExecutionLog(t, reportRuntimeExport)
	if !exportLog.Success || exportLog.RowCount != 2 {
		t.Fatalf("export success log should record rows: %#v", exportLog)
	}
	assertReportRuntimeLog(t, exportLog, reportRuntimeExport, published2.VersionId, 2)
	if published1.VersionId == published2.VersionId {
		t.Fatalf("publish versions should differ: first=%#v second=%#v", published1, published2)
	}
}

func TestReportV1BExportRejectsVersionAndSQLPermission(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "v1b_archived_version", reportStatusPublished, queryConfig, layoutConfig)
	version := env.createVersion(t, report, 1, reportVersionArchived, true, queryConfig, layoutConfig)
	if err := env.db.Model(&model.ReportDefinition{}).
		Where("id = ?", report.Id).
		Update("published_version_id", version.Id).Error; err != nil {
		t.Fatalf("set archived version pointer: %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{}); err == nil || !strings.Contains(err.Error(), "发布版本状态不可运行") {
		t.Fatalf("export should reject non-published version status, got %v", err)
	}

	sqlQuery, sqlLayout := reportV1ASQLConfig()
	sqlReport := env.createReport(t, "v1b_sql_block", reportStatusDraft, sqlQuery, sqlLayout)
	sqlVersion := env.createVersion(t, sqlReport, 1, reportVersionPublished, true, sqlQuery, sqlLayout)
	if err := env.db.Model(&model.ReportDefinition{}).
		Where("id = ?", sqlReport.Id).
		Updates(map[string]any{
			"status":               reportStatusPublished,
			"state":                true,
			"published_version_id": sqlVersion.Id,
		}).Error; err != nil {
		t.Fatalf("mark sql report as published: %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, sqlReport.Id, request.ReportExportReq{}); err == nil || !strings.Contains(err.Error(), "无权限") {
		t.Fatalf("non-super-admin should not export SQL dataset, got %v", err)
	}
	sqlLog := env.lastExecutionLog(t, reportRuntimeExport)
	if sqlLog.Success {
		t.Fatalf("sql export failure log should be unsuccessful: %#v", sqlLog)
	}
	assertReportRuntimeLog(t, sqlLog, reportRuntimeExport, sqlVersion.Id, 1)
}

func TestReportV1BExportMaxRowsAndFormatValidation(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "v1b_limits", reportStatusDraft, queryConfig, layoutConfig)
	published, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{})
	if err != nil {
		t.Fatalf("publish report: %v", err)
	}

	if got, err := normalizeReportExportMaxRows(request.ReportExportReq{}); err != nil || got != defaultReportExportMaxRows {
		t.Fatalf("default max rows = %d/%v, want %d", got, err, defaultReportExportMaxRows)
	}
	if got, err := normalizeReportExportMaxRows(request.ReportExportReq{MaxRows: reportIntPtr(0)}); err != nil || got != defaultReportExportMaxRows {
		t.Fatalf("non-positive max rows = %d/%v, want default", got, err)
	}
	if _, err := normalizeReportExportMaxRows(request.ReportExportReq{MaxRows: reportIntPtr(10), MaxRowsAlt: reportIntPtr(20)}); err == nil || !strings.Contains(err.Error(), "max_rows") {
		t.Fatalf("conflicting max rows should fail, got %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{MaxRows: reportIntPtr(maxReportExportRows + 1)}); err == nil || !strings.Contains(err.Error(), "最大限制") {
		t.Fatalf("max rows above hard limit should fail, got %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{Format: "xlsx"}); err == nil || !strings.Contains(err.Error(), "不支持 xlsx") {
		t.Fatalf("xlsx export should fail clearly, got %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{MaxRows: reportIntPtr(1)}); err == nil || !strings.Contains(err.Error(), "导出行数超过系统限制") {
		t.Fatalf("total greater than max rows should fail, got %v", err)
	}
	limitLog := env.lastExecutionLog(t, reportRuntimeExport)
	if limitLog.Success || limitLog.RowCount != 0 {
		t.Fatalf("limit failure log should be unsuccessful with row_count=0: %#v", limitLog)
	}
	assertReportRuntimeLog(t, limitLog, reportRuntimeExport, published.VersionId, 1)
}

func TestReportV1BTableExportCompletesRowsBeyondSingleQueryLimit(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	const totalRows = 6000
	appendDemoOrderRows(t, env, 3, totalRows)
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "v1b_table_export_pages", reportStatusDraft, queryConfig, layoutConfig)
	published, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{})
	if err != nil {
		t.Fatalf("publish table report: %v", err)
	}

	file, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{MaxRows: reportIntPtr(maxReportExportRows)})
	if err != nil {
		t.Fatalf("export table rows beyond single query limit: %v", err)
	}
	records := parseReportCSV(t, file.Content)
	if file.RowCount != totalRows || len(records) != totalRows+1 {
		t.Fatalf("export should include all rows: rowCount=%d csvRecords=%d", file.RowCount, len(records))
	}
	exportLog := env.lastExecutionLog(t, reportRuntimeExport)
	if !exportLog.Success || exportLog.RowCount != totalRows {
		t.Fatalf("export log should record all exported rows: %#v", exportLog)
	}
	assertReportRuntimeLog(t, exportLog, reportRuntimeExport, published.VersionId, 1)
}

func TestReportV1BSQLDatasetSuperAdminExportsPublishedVersion(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	queryConfig, layoutConfig := reportV1ASQLConfig()
	report := env.createReport(t, "v1b_sql_export_success", reportStatusDraft, queryConfig, layoutConfig)
	published, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{})
	if err != nil {
		t.Fatalf("publish sql report as super admin: %v", err)
	}

	file, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{DatasetId: "sql_1"})
	if err != nil {
		t.Fatalf("export sql dataset as super admin: %v", err)
	}
	records := parseReportCSV(t, file.Content)
	if file.RowCount != 2 || len(records) != 3 {
		t.Fatalf("unexpected sql export rows: rowCount=%d records=%#v", file.RowCount, records)
	}
	if len(records[0]) < 2 || records[0][0] != "id" || records[0][1] != "name" {
		t.Fatalf("unexpected sql export header: %#v", records[0])
	}
	if !csvRowsContain(records, "alpha") || !csvRowsContain(records, "beta") {
		t.Fatalf("sql export should contain seeded rows: %#v", records)
	}
	exportLog := env.lastExecutionLog(t, reportRuntimeExport)
	if !exportLog.Success || exportLog.RowCount != 2 {
		t.Fatalf("sql export success log should record rows: %#v", exportLog)
	}
	assertReportRuntimeLog(t, exportLog, reportRuntimeExport, published.VersionId, 1)
}

func TestReportV1BSQLDatasetTotalLimitFails(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	queryConfig, layoutConfig := reportV1ASQLConfig()
	report := env.createReport(t, "v1b_sql_export_limit", reportStatusDraft, queryConfig, layoutConfig)
	published, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{})
	if err != nil {
		t.Fatalf("publish sql report as super admin: %v", err)
	}

	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{DatasetId: "sql_1", MaxRows: reportIntPtr(1)}); err == nil || !strings.Contains(err.Error(), "导出行数超过系统限制") {
		t.Fatalf("sql export total greater than max rows should fail, got %v", err)
	}
	exportLog := env.lastExecutionLog(t, reportRuntimeExport)
	if exportLog.Success || exportLog.RowCount != 0 {
		t.Fatalf("sql export limit log should be unsuccessful with row_count=0: %#v", exportLog)
	}
	assertReportRuntimeLog(t, exportLog, reportRuntimeExport, published.VersionId, 1)
}

func TestReportV1BJoinedTableDatasetExportUsesNewRuntime(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	seedReportV1BJoinedTables(t, env)
	queryConfig, layoutConfig := reportV1BJoinedConfig()
	report := env.createReport(t, "v1b_joined_export_success", reportStatusDraft, queryConfig, layoutConfig)
	published, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{})
	if err != nil {
		t.Fatalf("publish joined report: %v", err)
	}
	recorder := newReportJoinedQueryRecorder(1, [][]driver.Value{{"alpha", "Acme"}})
	env.svc.reportRepo = reportDefinitionRepoWithQueryDB{
		ReportDefinitionRepository: env.svc.reportRepo,
		queryDB:                    newReportJoinedQueryDB(t, recorder),
	}

	file, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{})
	if err != nil {
		t.Fatalf("export joined table dataset: %v", err)
	}
	records := parseReportCSV(t, file.Content)
	if file.RowCount != 1 || len(records) != 2 {
		t.Fatalf("joined export should include only scoped row: rowCount=%d records=%#v", file.RowCount, records)
	}
	if len(records[0]) < 2 || records[0][0] != "订单名称" || records[0][1] != "公司名称" {
		t.Fatalf("unexpected joined export header: %#v", records[0])
	}
	if records[1][0] != "alpha" || records[1][1] != "Acme" || csvRowsContain(records, "beta") {
		t.Fatalf("joined export should apply primary table data scope: %#v", records)
	}
	if recorder.hasQueryContaining(`"demo_order"."tenant_id" IN`) {
		t.Fatalf("joined export must not use the removed legacy data-scope filter, queries=%#v", recorder.queries)
	}
	exportLog := env.lastExecutionLog(t, reportRuntimeExport)
	if !exportLog.Success || exportLog.RowCount != 1 {
		t.Fatalf("joined export success log should record scoped row count: %#v", exportLog)
	}
	assertReportRuntimeLog(t, exportLog, reportRuntimeExport, published.VersionId, 1)
}

func TestReportV1BJoinedTableDatasetTotalLimitFails(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	seedReportV1BJoinedTables(t, env)
	queryConfig, layoutConfig := reportV1BJoinedConfig()
	report := env.createReport(t, "v1b_joined_export_limit", reportStatusDraft, queryConfig, layoutConfig)
	published, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{})
	if err != nil {
		t.Fatalf("publish joined report: %v", err)
	}
	recorder := newReportJoinedQueryRecorder(2, [][]driver.Value{{"alpha", "Acme"}})
	env.svc.reportRepo = reportDefinitionRepoWithQueryDB{
		ReportDefinitionRepository: env.svc.reportRepo,
		queryDB:                    newReportJoinedQueryDB(t, recorder),
	}

	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{MaxRows: reportIntPtr(1)}); err == nil || !strings.Contains(err.Error(), "导出行数超过系统限制") {
		t.Fatalf("joined export total greater than max rows should fail, got %v", err)
	}
	exportLog := env.lastExecutionLog(t, reportRuntimeExport)
	if exportLog.Success || exportLog.RowCount != 0 {
		t.Fatalf("joined export limit log should be unsuccessful with row_count=0: %#v", exportLog)
	}
	assertReportRuntimeLog(t, exportLog, reportRuntimeExport, published.VersionId, 1)
}

func TestReportV1BCSVSafety(t *testing.T) {
	content, err := buildReportCSV([]response.ReportPreviewColumn{
		{Field: "name", Label: "=危险表头"},
		{Field: "amount", Label: "金额"},
	}, []map[string]interface{}{
		{"name": "+SUM(1,1)", "amount": "-10"},
		{"name": "@cmd", "amount": 20},
	})
	if err != nil {
		t.Fatalf("build csv: %v", err)
	}
	records := parseReportCSV(t, content)
	if records[0][0] != "'=危险表头" {
		t.Fatalf("header should be formula-safe: %#v", records[0])
	}
	if records[1][0] != "'+SUM(1,1)" || records[1][1] != "'-10" || records[2][0] != "'@cmd" {
		t.Fatalf("data cells should be formula-safe: %#v", records)
	}
}

func parseReportCSV(t *testing.T, content []byte) [][]string {
	t.Helper()
	content = bytes.TrimPrefix(content, []byte("\xEF\xBB\xBF"))
	records, err := csv.NewReader(bytes.NewReader(content)).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	return records
}

func assertCSVHeader(t *testing.T, records [][]string, want string) {
	t.Helper()
	if len(records) == 0 || len(records[0]) == 0 {
		t.Fatalf("csv header missing: %#v", records)
	}
	if records[0][0] != want {
		t.Fatalf("csv header = %q, want %q; records=%#v", records[0][0], want, records)
	}
}

func reportIntPtr(value int) *int {
	return &value
}

func appendDemoOrderRows(t *testing.T, env *reportV1ATestEnv, from int, to int) {
	t.Helper()
	if from > to {
		return
	}
	rows := make([]map[string]any, 0, to-from+1)
	for id := from; id <= to; id++ {
		rows = append(rows, map[string]any{
			"id":     id,
			"name":   fmt.Sprintf("bulk-%04d", id),
			"amount": float64(id),
		})
	}
	if err := env.db.Table("demo_order").CreateInBatches(rows, 500).Error; err != nil {
		t.Fatalf("append demo_order rows: %v", err)
	}
}

func seedReportV1BJoinedTables(t *testing.T, env *reportV1ATestEnv) {
	t.Helper()
	var orderTable model.SysTable
	if err := env.db.Where("table_code = ?", "demo_order").First(&orderTable).Error; err != nil {
		t.Fatalf("load demo_order metadata: %v", err)
	}
	orderFields := []model.SysTableField{
		{Basic: model.Basic{Id: env.nextID(t), State: true}, TableId: orderTable.Id, FieldName: "公司ID", FieldCode: "company_id", FieldType: enum.BigIntFieldType, IsListShow: true, Sequence: 4},
		{Basic: model.Basic{Id: env.nextID(t), State: true}, TableId: orderTable.Id, FieldName: "租户ID", FieldCode: "tenant_id", FieldType: enum.BigIntFieldType, IsListShow: true, Sequence: 5},
	}
	if err := env.db.Create(&orderFields).Error; err != nil {
		t.Fatalf("seed demo_order joined fields: %v", err)
	}
	if err := env.db.Exec(`ALTER TABLE demo_order ADD COLUMN company_id integer`).Error; err != nil {
		t.Fatalf("add demo_order company_id: %v", err)
	}
	if err := env.db.Exec(`ALTER TABLE demo_order ADD COLUMN tenant_id integer`).Error; err != nil {
		t.Fatalf("add demo_order tenant_id: %v", err)
	}
	if err := env.db.Exec(`UPDATE demo_order SET company_id = 10, tenant_id = 8 WHERE id = 1`).Error; err != nil {
		t.Fatalf("scope allowed demo_order row: %v", err)
	}
	if err := env.db.Exec(`UPDATE demo_order SET company_id = 20, tenant_id = 9 WHERE id = 2`).Error; err != nil {
		t.Fatalf("scope denied demo_order row: %v", err)
	}
	companyTable := model.SysTable{
		Basic:     model.Basic{Id: env.nextID(t), State: true},
		TableName: "公司",
		TableCode: "demo_company",
		TableType: enum.System,
	}
	if err := env.db.Create(&companyTable).Error; err != nil {
		t.Fatalf("seed demo_company table: %v", err)
	}
	companyFields := []model.SysTableField{
		{Basic: model.Basic{Id: env.nextID(t), State: true}, TableId: companyTable.Id, FieldName: "ID", FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true, IsListShow: true, Sequence: 1},
		{Basic: model.Basic{Id: env.nextID(t), State: true}, TableId: companyTable.Id, FieldName: "公司名称", FieldCode: "company_name", FieldType: enum.VarcharFieldType, IsListShow: true, Sequence: 2},
	}
	if err := env.db.Create(&companyFields).Error; err != nil {
		t.Fatalf("seed demo_company fields: %v", err)
	}
	if err := env.db.Exec(`CREATE TABLE demo_company (id integer, company_name text)`).Error; err != nil {
		t.Fatalf("create demo_company: %v", err)
	}
	if err := env.db.Exec(`INSERT INTO demo_company (id, company_name) VALUES (10, 'Acme'), (20, 'BlockedCo')`).Error; err != nil {
		t.Fatalf("seed demo_company rows: %v", err)
	}
}

func reportV1BJoinedConfig() (datatypes.JSON, datatypes.JSON) {
	queryConfig := datatypes.JSON([]byte(`{
		"datasets":[
			{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true,"fields":[
				{"code":"name","name":"订单名称","type":"string"},
				{"code":"company_id","name":"公司ID","type":"number"},
				{"code":"tenant_id","name":"租户ID","type":"number"}
			]},
			{"id":"company","name":"公司","type":"table","source_code":"demo_company","fields":[
				{"code":"company_name","name":"公司名称","type":"string"}
			]}
		],
		"dataset_joins":[{"id":"j1","left_dataset_id":"main","right_dataset_id":"company","join_type":"left","conditions":[{"left_field":"company_id","right_field":"id"}]}],
		"fields":[],
		"parameters":[]
	}`))
	layoutConfig := datatypes.JSON([]byte(`{
		"view":"sheet",
		"datasets":[
			{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true},
			{"id":"company","name":"公司","type":"table","source_code":"demo_company"}
		],
		"dataset_joins":[{"id":"j1","left_dataset_id":"main","right_dataset_id":"company","join_type":"left","conditions":[{"left_field":"company_id","right_field":"id"}]}],
		"parameters":[],
		"sheet":{"rows":8,"cols":6,"cells":[
			{"id":"A1","row":1,"col":1,"value":"订单名称","binding":{"type":"field","dataset_id":"main","field":"name"}},
			{"id":"B1","row":1,"col":2,"value":"公司名称","binding":{"type":"group","dataset_id":"company","field":"company_name"}}
		]}
	}`))
	return queryConfig, layoutConfig
}

func csvRowsContain(records [][]string, value string) bool {
	for _, record := range records {
		for _, cell := range record {
			if cell == value {
				return true
			}
		}
	}
	return false
}

type reportDefinitionRepoWithQueryDB struct {
	repository.ReportDefinitionRepository
	queryDB *gorm.DB
}

func (r reportDefinitionRepoWithQueryDB) DBWithContext(context.Context) *gorm.DB {
	return r.queryDB
}

type reportJoinedQueryRecorder struct {
	total     int64
	rows      [][]driver.Value
	queries   []string
	args      [][]driver.NamedValue
	deadlines []time.Time
}

func newReportJoinedQueryRecorder(total int64, rows [][]driver.Value) *reportJoinedQueryRecorder {
	return &reportJoinedQueryRecorder{total: total, rows: rows}
}

func (r *reportJoinedQueryRecorder) record(query string, args []driver.NamedValue) {
	r.queries = append(r.queries, query)
	r.args = append(r.args, args)
}

func (r *reportJoinedQueryRecorder) hasQueryContaining(fragment string) bool {
	for _, query := range r.queries {
		if strings.Contains(query, fragment) {
			return true
		}
	}
	return false
}

var (
	reportJoinedDriverOnce      sync.Once
	reportJoinedDriverRegistry  = map[string]*reportJoinedQueryRecorder{}
	reportJoinedDriverRegistryM sync.Mutex
)

func newReportJoinedQueryDB(t *testing.T, recorder *reportJoinedQueryRecorder) *gorm.DB {
	t.Helper()
	const driverName = "report_joined_export_test"
	reportJoinedDriverOnce.Do(func() {
		sql.Register(driverName, reportJoinedDriver{})
	})
	dsn := fmt.Sprintf("%s_%p", t.Name(), recorder)
	reportJoinedDriverRegistryM.Lock()
	reportJoinedDriverRegistry[dsn] = recorder
	reportJoinedDriverRegistryM.Unlock()
	t.Cleanup(func() {
		reportJoinedDriverRegistryM.Lock()
		delete(reportJoinedDriverRegistry, dsn)
		reportJoinedDriverRegistryM.Unlock()
	})
	db, err := gorm.Open(sqlite.Dialector{DriverName: driverName, DSN: dsn}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open joined query test db: %v", err)
	}
	return db
}

type reportJoinedDriver struct{}

func (reportJoinedDriver) Open(name string) (driver.Conn, error) {
	reportJoinedDriverRegistryM.Lock()
	recorder := reportJoinedDriverRegistry[name]
	reportJoinedDriverRegistryM.Unlock()
	return &reportJoinedConn{recorder: recorder}, nil
}

type reportJoinedConn struct {
	recorder *reportJoinedQueryRecorder
}

func (c *reportJoinedConn) Prepare(query string) (driver.Stmt, error) {
	return &reportJoinedStmt{conn: c, query: query}, nil
}

func (c *reportJoinedConn) PrepareContext(_ context.Context, query string) (driver.Stmt, error) {
	return &reportJoinedStmt{conn: c, query: query}, nil
}

func (c *reportJoinedConn) Close() error {
	return nil
}

func (c *reportJoinedConn) Begin() (driver.Tx, error) {
	return reportJoinedTx{}, nil
}

func (c *reportJoinedConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if deadline, ok := ctx.Deadline(); ok && c.recorder != nil {
		c.recorder.deadlines = append(c.recorder.deadlines, deadline)
	}
	return c.query(query, args)
}

func (c *reportJoinedConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (c *reportJoinedConn) query(query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.recorder != nil {
		c.recorder.record(query, args)
	}
	lower := strings.ToLower(query)
	if strings.Contains(lower, "sqlite_version") {
		return &reportJoinedRows{columns: []string{"sqlite_version()"}, values: [][]driver.Value{{"3.45.0"}}}, nil
	}
	if strings.Contains(lower, "set_config") {
		value := driver.Value("0")
		if len(args) > 0 {
			value = args[0].Value
		}
		return &reportJoinedRows{columns: []string{"set_config"}, values: [][]driver.Value{{value}}}, nil
	}
	if strings.Contains(lower, "count(") {
		total := int64(0)
		if c.recorder != nil {
			total = c.recorder.total
		}
		return &reportJoinedRows{columns: []string{"count"}, values: [][]driver.Value{{total}}}, nil
	}
	rows := [][]driver.Value(nil)
	if c.recorder != nil {
		rows = c.recorder.rows
	}
	return &reportJoinedRows{columns: []string{"main__name", "company__company_name"}, values: rows}, nil
}

type reportJoinedStmt struct {
	conn  *reportJoinedConn
	query string
}

func (s *reportJoinedStmt) Close() error {
	return nil
}

func (s *reportJoinedStmt) NumInput() int {
	return -1
}

func (s *reportJoinedStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (s *reportJoinedStmt) Query(args []driver.Value) (driver.Rows, error) {
	named := make([]driver.NamedValue, 0, len(args))
	for index, value := range args {
		named = append(named, driver.NamedValue{Ordinal: index + 1, Value: value})
	}
	return s.conn.query(s.query, named)
}

type reportJoinedRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *reportJoinedRows) Columns() []string {
	return r.columns
}

func (r *reportJoinedRows) Close() error {
	return nil
}

func (r *reportJoinedRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

type reportJoinedTx struct{}

func (reportJoinedTx) Commit() error {
	return nil
}

func (reportJoinedTx) Rollback() error {
	return nil
}
