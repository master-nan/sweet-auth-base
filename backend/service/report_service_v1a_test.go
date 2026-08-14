package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/database"
	"backend/internal/datapermission"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type reportV1ATestEnv struct {
	db  *gorm.DB
	sf  *utils.Snowflake
	svc *ReportService
	ctx *gin.Context
}

func TestReportV1ADesignPreviewAndStatusRules(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "v1a_status_rules", reportStatusDraft, queryConfig, layoutConfig)

	preview, err := env.svc.DesignPreview(env.ctx, report.Id, request.ReportPreviewReq{Query: request.Basic{Page: 1, Num: 20}})
	if err != nil {
		t.Fatalf("design preview draft report: %v", err)
	}
	if preview.Meta.RuntimeType != reportRuntimeDesignPreview || preview.Meta.VersionId != 0 || preview.Meta.VersionNo != 0 {
		t.Fatalf("unexpected design-preview meta: %#v", preview.Meta)
	}
	if !hasReportColumn(preview.Columns, "name") {
		t.Fatalf("design-preview should use draft query_config columns: %#v", preview.Columns)
	}
	designLog := env.lastExecutionLog(t, reportRuntimeDesignPreview)
	assertReportRuntimeLog(t, designLog, reportRuntimeDesignPreview, 0, 0)

	if _, err := env.svc.RunReport(env.ctx, report.Id, request.ReportPreviewReq{Query: request.Basic{Page: 1, Num: 20}}); err == nil || !strings.Contains(err.Error(), "未发布") {
		t.Fatalf("draft report should not run, got %v", err)
	}
	if err := env.svc.UpdateReportDefinitionStatus(env.ctx, report.Id, reportStatusPublished); err == nil || !strings.Contains(err.Error(), "/admin/report/:id/publish") {
		t.Fatalf("status API should reject direct publish, got %v", err)
	}

	disabled := env.createReport(t, "v1a_disabled", reportStatusDisabled, queryConfig, layoutConfig)
	if _, err := env.svc.DesignPreview(env.ctx, disabled.Id, request.ReportPreviewReq{}); err == nil || !strings.Contains(err.Error(), "已停用") {
		t.Fatalf("disabled report should not design-preview, got %v", err)
	}
	if _, err := env.svc.RunReport(env.ctx, disabled.Id, request.ReportPreviewReq{}); err == nil || !strings.Contains(err.Error(), "已停用") {
		t.Fatalf("disabled report should not run, got %v", err)
	}
}

func TestReportV1APublishRunUsesVersionSnapshot(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	nameQuery, nameLayout := reportV1ATableConfig("name")
	report := env.createReport(t, "v1a_snapshot", reportStatusDraft, nameQuery, nameLayout)

	published1, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{ChangeLog: "publish name"})
	if err != nil {
		t.Fatalf("publish report first version: %v", err)
	}
	if published1.VersionNo != 1 || published1.Status != reportVersionPublished {
		t.Fatalf("unexpected first publish response: %#v", published1)
	}
	var stored model.ReportDefinition
	if err := env.db.First(&stored, report.Id).Error; err != nil {
		t.Fatalf("load published report: %v", err)
	}
	if stored.PublishedVersionId != published1.VersionId || normalizeReportStatus(stored.Status) != reportStatusPublished {
		t.Fatalf("published pointer not updated: report=%#v publish=%#v", stored, published1)
	}

	run1, err := env.svc.RunReport(env.ctx, report.Id, request.ReportPreviewReq{Query: request.Basic{Page: 1, Num: 20}})
	if err != nil {
		t.Fatalf("run published report first version: %v", err)
	}
	if run1.Meta.VersionId != published1.VersionId || run1.Meta.VersionNo != 1 || run1.Meta.RuntimeType != reportRuntimeRun {
		t.Fatalf("run should use published version snapshot: %#v", run1.Meta)
	}
	if !hasReportColumn(run1.Columns, "name") || hasReportColumn(run1.Columns, "amount") {
		t.Fatalf("first run should use name snapshot columns: %#v", run1.Columns)
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

	runAfterDraftEdit, err := env.svc.RunReport(env.ctx, report.Id, request.ReportPreviewReq{Query: request.Basic{Page: 1, Num: 20}})
	if err != nil {
		t.Fatalf("run after draft edit: %v", err)
	}
	if !hasReportColumn(runAfterDraftEdit.Columns, "name") || hasReportColumn(runAfterDraftEdit.Columns, "amount") {
		t.Fatalf("run must ignore unpublished draft config: %#v", runAfterDraftEdit.Columns)
	}

	published2, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{ChangeLog: "publish amount"})
	if err != nil {
		t.Fatalf("publish report second version: %v", err)
	}
	if published2.VersionNo != 2 || published2.VersionId == published1.VersionId {
		t.Fatalf("unexpected second publish response: first=%#v second=%#v", published1, published2)
	}
	run2, err := env.svc.RunReport(env.ctx, report.Id, request.ReportPreviewReq{Query: request.Basic{Page: 1, Num: 20}})
	if err != nil {
		t.Fatalf("run second published version: %v", err)
	}
	if run2.Meta.VersionId != published2.VersionId || run2.Meta.VersionNo != 2 {
		t.Fatalf("run should use second version snapshot: %#v", run2.Meta)
	}
	if !hasReportColumn(run2.Columns, "amount") || hasReportColumn(run2.Columns, "name") {
		t.Fatalf("second run should use amount snapshot columns: %#v", run2.Columns)
	}
	var oldVersion model.ReportDefinitionVersion
	if err := env.db.First(&oldVersion, published1.VersionId).Error; err != nil {
		t.Fatalf("load old version: %v", err)
	}
	if oldVersion.Status != reportVersionArchived {
		t.Fatalf("old version should be archived after second publish: %#v", oldVersion)
	}
	runLog := env.lastExecutionLog(t, reportRuntimeRun)
	assertReportRuntimeLog(t, runLog, reportRuntimeRun, published2.VersionId, 2)
}

func TestReportV1ASQLDatasetRequiresSuperAdmin(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	queryConfig, layoutConfig := reportV1ASQLConfig()
	report := env.createReport(t, "v1a_sql_block", reportStatusDraft, queryConfig, layoutConfig)

	if _, err := env.svc.DesignPreview(env.ctx, report.Id, request.ReportPreviewReq{}); err == nil || !strings.Contains(err.Error(), "无权限") {
		t.Fatalf("non-super-admin should not design-preview SQL dataset, got %v", err)
	}
	if _, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{}); err == nil || !strings.Contains(err.Error(), "无权限") {
		t.Fatalf("non-super-admin should not publish SQL dataset, got %v", err)
	}

	version := env.createVersion(t, report, 1, reportVersionPublished, true, queryConfig, layoutConfig)
	if err := env.db.Model(&model.ReportDefinition{}).
		Where("id = ?", report.Id).
		Updates(map[string]any{
			"status":               reportStatusPublished,
			"state":                true,
			"published_version_id": version.Id,
		}).Error; err != nil {
		t.Fatalf("mark sql report as published: %v", err)
	}
	if _, err := env.svc.RunReport(env.ctx, report.Id, request.ReportPreviewReq{}); err == nil || !strings.Contains(err.Error(), "无权限") {
		t.Fatalf("non-super-admin should not run SQL dataset, got %v", err)
	}
}

func TestReportV1ARunRejectsUnpublishedVersionStatus(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "v1a_archived_version", reportStatusPublished, queryConfig, layoutConfig)
	version := env.createVersion(t, report, 1, reportVersionArchived, true, queryConfig, layoutConfig)
	if err := env.db.Model(&model.ReportDefinition{}).
		Where("id = ?", report.Id).
		Update("published_version_id", version.Id).Error; err != nil {
		t.Fatalf("set archived version pointer: %v", err)
	}

	if _, err := env.svc.RunReport(env.ctx, report.Id, request.ReportPreviewReq{}); err == nil || !strings.Contains(err.Error(), "发布版本状态不可运行") {
		t.Fatalf("run should reject non-published version status, got %v", err)
	}
}

func newReportV1ATestEnv(t *testing.T, user model.SysUser) *reportV1ATestEnv {
	t.Helper()
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{},
		&model.SysUser{},
		&model.SysRole{},
		&model.SysUserRole{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysTable{},
		&model.SysTableField{},
		&model.SysTableRelation{},
		&model.SysTableIndex{},
		&model.SysTableIndexField{},
		&model.ReportDefinition{},
		&model.ReportDefinitionVersion{},
		&model.ReportExecutionLog{},
	)
	seedReportV1ATable(t, db)
	primaryDB := &database.PrimaryDB{DB: db}
	store := newJSONMemoryCacher()
	permissionRuntime := newLowCodeDataPermissionRuntime(
		func(context.Context, int) ([]model.DataResource, error) { return nil, nil },
		func(context.Context, int) ([]model.DataOwnershipField, error) { return nil, nil },
		func(context.Context, int) (datapermission.SubjectContext, error) {
			return datapermission.SubjectContext{}, nil
		},
		func(context.Context, datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			return datapermission.DataScopeResult{}, nil
		},
		func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
			return datapermission.BuildAdapterExecution(input)
		},
	)
	generalizationService := NewGeneralizationServiceWithDataPermission(
		impl.NewGeneralizationRepositoryImpl(primaryDB),
		sf,
		permissionRuntime,
	)
	metadataRuntime := NewMetadataRuntimeService(
		impl.NewSysTableRepositoryImpl(primaryDB),
		impl.NewSysTableFieldRepositoryImpl(primaryDB),
		cache.NewSysTableCache(store),
		cache.NewSysTableFieldCache(store),
	)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("user", user)
	return &reportV1ATestEnv{
		db: db,
		sf: sf,
		svc: &ReportService{
			reportRepo:            impl.NewReportDefinitionRepositoryImpl(primaryDB),
			reportVersionRepo:     impl.NewReportDefinitionVersionRepositoryImpl(primaryDB),
			reportLogRepo:         impl.NewReportExecutionLogRepositoryImpl(primaryDB),
			generalizationService: generalizationService,
			metadataRuntime:       metadataRuntime,
			sf:                    sf,
		},
		ctx: ctx,
	}
}

func reportV1AUser(superAdmin bool) model.SysUser {
	user := model.SysUser{
		Basic:    model.Basic{Id: 1001, State: true},
		UserName: "report_tester",
	}
	if superAdmin {
		user.Roles = []model.SysRole{{Basic: model.Basic{Id: 1, State: true}, Name: "super_admin"}}
	}
	return user
}

func seedReportV1ATable(t *testing.T, db *gorm.DB) {
	t.Helper()
	table := model.SysTable{
		Basic:     model.Basic{Id: 100, State: true},
		TableName: "订单",
		TableCode: "demo_order",
		TableType: enum.System,
	}
	if err := db.Create(&table).Error; err != nil {
		t.Fatalf("seed sys table: %v", err)
	}
	fields := []model.SysTableField{
		{Basic: model.Basic{Id: 101, State: true}, TableId: table.Id, FieldName: "ID", FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true, IsListShow: true, Sequence: 1},
		{Basic: model.Basic{Id: 102, State: true}, TableId: table.Id, FieldName: "名称", FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true, Sequence: 2},
		{Basic: model.Basic{Id: 103, State: true}, TableId: table.Id, FieldName: "金额", FieldCode: "amount", FieldType: enum.FloatFieldType, IsListShow: true, Sequence: 3},
	}
	if err := db.Create(&fields).Error; err != nil {
		t.Fatalf("seed sys table fields: %v", err)
	}
	if err := db.Exec(`CREATE TABLE demo_order (id integer, name text, amount real)`).Error; err != nil {
		t.Fatalf("create demo_order: %v", err)
	}
	if err := db.Exec(`INSERT INTO demo_order (id, name, amount) VALUES (1, 'alpha', 12.5), (2, 'beta', 30.0)`).Error; err != nil {
		t.Fatalf("seed demo_order: %v", err)
	}
}

func (env *reportV1ATestEnv) createReport(t *testing.T, code string, status string, queryConfig datatypes.JSON, layoutConfig datatypes.JSON) model.ReportDefinition {
	t.Helper()
	id := env.nextID(t)
	report := model.ReportDefinition{
		Basic:               model.Basic{Id: id, State: status != reportStatusDisabled},
		Code:                code,
		Name:                code,
		Status:              status,
		SourceType:          reportSourceTypeTable,
		SourceCode:          "demo_order",
		PermissionTableCode: "demo_order",
		QueryConfig:         queryConfig,
		LayoutConfig:        layoutConfig,
	}
	if err := env.db.Create(&report).Error; err != nil {
		t.Fatalf("create report %s: %v", code, err)
	}
	return report
}

func (env *reportV1ATestEnv) createVersion(t *testing.T, report model.ReportDefinition, versionNo int, status string, state bool, queryConfig datatypes.JSON, layoutConfig datatypes.JSON) model.ReportDefinitionVersion {
	t.Helper()
	version := model.ReportDefinitionVersion{
		Basic:               model.Basic{Id: env.nextID(t), State: state},
		ReportId:            report.Id,
		VersionNo:           versionNo,
		ReportCode:          report.Code,
		ReportName:          report.Name,
		SourceType:          report.SourceType,
		SourceCode:          report.SourceCode,
		PermissionTableCode: report.PermissionTableCode,
		QueryConfig:         queryConfig,
		LayoutConfig:        layoutConfig,
		Status:              status,
		PublishedAt:         model.CustomTime(model.Now()),
		PublishedBy:         reportV1AUser(false).Id,
		PublishedName:       reportV1AUser(false).UserName,
	}
	if err := env.db.Create(&version).Error; err != nil {
		t.Fatalf("create report version: %v", err)
	}
	return version
}

func (env *reportV1ATestEnv) nextID(t *testing.T) int {
	t.Helper()
	id, err := env.sf.GenerateUniqueID()
	if err != nil {
		t.Fatalf("generate id: %v", err)
	}
	return int(id)
}

func (env *reportV1ATestEnv) lastExecutionLog(t *testing.T, action string) model.ReportExecutionLog {
	t.Helper()
	var log model.ReportExecutionLog
	if err := env.db.Where("action = ?", action).Order("id DESC").First(&log).Error; err != nil {
		t.Fatalf("load last execution log action=%s: %v", action, err)
	}
	return log
}

func assertReportRuntimeLog(t *testing.T, log model.ReportExecutionLog, runtimeType string, versionId int, versionNo int) {
	t.Helper()
	if log.Action != runtimeType {
		t.Fatalf("unexpected log action: %#v", log)
	}
	var payload struct {
		Runtime struct {
			RuntimeType string `json:"runtime_type"`
			VersionId   int    `json:"version_id"`
			VersionNo   int    `json:"version_no"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal(log.Params, &payload); err != nil {
		t.Fatalf("unmarshal execution log params: %v; raw=%s", err, string(log.Params))
	}
	if payload.Runtime.RuntimeType != runtimeType || payload.Runtime.VersionId != versionId || payload.Runtime.VersionNo != versionNo {
		t.Fatalf("unexpected runtime params: %#v raw=%s", payload.Runtime, string(log.Params))
	}
}

func reportV1ATableConfig(field string) (datatypes.JSON, datatypes.JSON) {
	label := map[string]string{
		"name":   "名称",
		"amount": "金额",
	}[field]
	fieldType := map[string]string{
		"name":   "string",
		"amount": "number",
	}[field]
	queryConfig := datatypes.JSON([]byte(fmt.Sprintf(`{
		"datasets":[{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true,"fields":[{"code":"%s","name":"%s","type":"%s"}]}],
		"fields":[{"code":"%s","name":"%s","type":"%s"}],
		"parameters":[]
	}`, field, label, fieldType, field, label, fieldType)))
	layoutConfig := datatypes.JSON([]byte(fmt.Sprintf(`{
		"view":"sheet",
		"datasets":[{"id":"main","name":"订单","type":"table","source_code":"demo_order","primary":true,"fields":[{"code":"%s","name":"%s","type":"%s"}]}],
		"parameters":[],
		"sheet":{"rows":8,"cols":6,"cells":[{"id":"A1","row":1,"col":1,"value":"%s","binding":{"type":"field","dataset_id":"main","field":"%s"}}]}
	}`, field, label, fieldType, label, field)))
	return queryConfig, layoutConfig
}

func reportV1ASQLConfig() (datatypes.JSON, datatypes.JSON) {
	queryConfig := datatypes.JSON([]byte(`{
		"datasets":[{"id":"sql_1","name":"SQL","type":"sql","sql":"select id, name from demo_order","primary":true,"fields":[{"code":"name","name":"名称","type":"string"}]}],
		"fields":[{"code":"name","name":"名称","type":"string"}],
		"parameters":[]
	}`))
	layoutConfig := datatypes.JSON([]byte(`{
		"view":"sheet",
		"datasets":[{"id":"sql_1","name":"SQL","type":"sql","sql":"select id, name from demo_order","primary":true,"fields":[{"code":"name","name":"名称","type":"string"}]}],
		"parameters":[],
		"sheet":{"rows":8,"cols":6,"cells":[{"id":"A1","row":1,"col":1,"value":"名称","binding":{"type":"field","dataset_id":"sql_1","field":"name"}}]}
	}`))
	return queryConfig, layoutConfig
}
