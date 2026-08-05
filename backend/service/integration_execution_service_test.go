package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/audit"
	"backend/internal/database"
	"backend/internal/datapermission"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

type integrationExecutionAuditWriter struct {
	records     []TransactionalAuditRecord
	subject     audit.AuditSubject
	correlation audit.CorrelationIDs
}

func (w *integrationExecutionAuditWriter) RecordTransactionalAuditContext(
	ctx context.Context,
	_ *gorm.DB,
	record TransactionalAuditRecord,
) error {
	w.records = append(w.records, record)
	w.subject, _ = audit.GetAuditSubject(ctx)
	w.correlation = audit.GetCorrelationIDs(ctx)
	return nil
}

func TestIntegrationExecutionServiceCreateIdempotencyDetailAndPage(t *testing.T) {
	svc, db, system, definition, auditWriter := newIntegrationExecutionTestSubject(t)
	ctx := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(88, "runtime-admin"))
	ctx = audit.WithCorrelationIDs(ctx, audit.CorrelationIDs{RequestID: "request-runtime", TraceID: "trace-runtime"})
	req := integrationExecutionCreateRequest(system.Id, definition.Id, "request-001")

	created, err := svc.CreateExecution(ctx, req)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	if created.ExecutionNo == "" || created.Status != model.IntegrationExecutionStatusCreated || created.Revision != 1 {
		t.Fatalf("unexpected created execution: %+v", created)
	}
	if auditWriter.subject.UserID != 88 || auditWriter.correlation.RequestID != "request-runtime" || auditWriter.correlation.TraceID != "trace-runtime" {
		t.Fatalf("audit context subject=%+v correlation=%+v", auditWriter.subject, auditWriter.correlation)
	}
	if created.ExternalSystem.SystemCode != system.SystemCode ||
		created.Interface.InterfaceCode != definition.InterfaceCode ||
		created.Interface.Version != definition.Version {
		t.Fatalf("execution snapshots were not derived from configuration: %+v", created)
	}

	hit, err := svc.CreateExecution(ctx, req)
	if err != nil || hit.Id != created.Id || len(auditWriter.records) != 1 {
		t.Fatalf("idempotency hit = %+v audits=%d err=%v", hit, len(auditWriter.records), err)
	}
	conflictReq := req
	conflictReq.InputHash = strings.Repeat("b", 64)
	if _, err = svc.CreateExecution(ctx, conflictReq); !errors.Is(err, apperrors.ErrIntegrationExecutionIdempotencyConflict) {
		t.Fatalf("idempotency conflict error = %v", err)
	}

	startedAt := model.Now()
	log := model.IntegrationLog{
		Basic: model.Basic{Id: 901, State: true}, ExecutionID: created.Id, AttemptNo: 1,
		Status: model.IntegrationLogStatusFailed, StartedAt: startedAt,
		ErrorCategory: model.IntegrationErrorCategoryNetwork, ErrorCode: "connection_reset",
		ResultCertainty: model.IntegrationResultCertaintyUnknown,
	}
	testutil.MustCreate(t, db, &log)
	table := integrationExecutionServiceQueryTable()
	detail, err := svc.GetExecution(ctx, created.Id, table, integrationExecutionPermission(t, model.DataPermissionOperationDetail, datapermission.DataScopeDecisionNotApplicable))
	if err != nil || len(detail.Attempts) != 1 || detail.Attempts[0].AttemptNo != 1 {
		t.Fatalf("detail = %+v err=%v", detail, err)
	}

	from := time.Time(created.GmtCreate).Add(-time.Minute)
	to := time.Time(created.GmtCreate).Add(time.Minute)
	page, err := svc.PageExecution(ctx, request.IntegrationExecutionQueryReq{
		Page: 1, Num: 10, Status: model.IntegrationExecutionStatusCreated,
		QuickQuery: &request.QuickQuery{Keyword: "INT-"}, CreatedFrom: &from, CreatedTo: &to,
	}, table, integrationExecutionPermission(t, model.DataPermissionOperationQuery, datapermission.DataScopeDecisionNotApplicable))
	if err != nil || page.Total != 1 || len(page.Data) != 1 || page.Data[0].Id != created.Id {
		t.Fatalf("page = %+v err=%v", page, err)
	}
	denied, err := svc.PageExecution(ctx, request.IntegrationExecutionQueryReq{Page: 1, Num: 10}, table,
		integrationExecutionPermission(t, model.DataPermissionOperationQuery, datapermission.DataScopeDecisionNone))
	if err != nil || denied.Total != 0 || len(denied.Data) != 0 {
		t.Fatalf("deny_all page = %+v err=%v", denied, err)
	}
	if _, err = svc.PageExecution(ctx, request.IntegrationExecutionQueryReq{Page: 1, Num: 10}, table, repository.GeneralizationPermission{}); !errors.Is(err, apperrors.ErrDataPermissionRuntimeRouteConflict) {
		t.Fatalf("invalid permission error = %v", err)
	}

	payload, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal detail: %v", err)
	}
	for _, forbidden := range []string{"authorization", "secret", "ciphertext", "payload", "gmt_delete"} {
		if strings.Contains(strings.ToLower(string(payload)), forbidden) {
			t.Fatalf("detail leaked %q: %s", forbidden, payload)
		}
	}
}

func TestIntegrationExecutionServiceStateMachineAndRevision(t *testing.T) {
	svc, _, system, definition, auditWriter := newIntegrationExecutionTestSubject(t)
	ctx := context.Background()

	created, err := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "success-001"))
	if err != nil {
		t.Fatalf("create success fixture: %v", err)
	}
	if _, err = svc.StartExecution(ctx, created.Id, created.Revision+1); !errors.Is(err, apperrors.ErrIntegrationExecutionRevisionConflict) {
		t.Fatalf("stale start error = %v", err)
	}
	running, err := svc.StartExecution(ctx, created.Id, created.Revision)
	if err != nil || running.Status != model.IntegrationExecutionStatusRunning || running.StartedAt == nil {
		t.Fatalf("start = %+v err=%v", running, err)
	}
	httpStatus := 200
	succeeded, err := svc.CompleteExecution(ctx, running.Id, request.IntegrationExecutionCompleteReq{
		Revision: running.Revision, ResultHTTPStatus: &httpStatus, ResultSizeBytes: 128,
		ResultHash: strings.Repeat("c", 64), ResultSummary: "ok",
	})
	if err != nil || succeeded.Status != model.IntegrationExecutionStatusSucceeded || succeeded.CompletedAt == nil {
		t.Fatalf("complete = %+v err=%v", succeeded, err)
	}
	if _, err = svc.StartExecution(ctx, succeeded.Id, succeeded.Revision); !errors.Is(err, apperrors.ErrIntegrationExecutionStatusInvalid) {
		t.Fatalf("terminal restart error = %v", err)
	}

	retryCreated, _ := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "retry-001"))
	retryRunning, _ := svc.StartExecution(ctx, retryCreated.Id, retryCreated.Revision)
	retryWaiting, err := svc.FailExecution(ctx, retryRunning.Id, request.IntegrationExecutionFailReq{
		Revision: retryRunning.Revision, TargetStatus: model.IntegrationExecutionStatusRetryWaiting,
		ErrorCategory: model.IntegrationErrorCategoryNetwork, ResultSummary: "retryable",
	})
	if err != nil || retryWaiting.Status != model.IntegrationExecutionStatusRetryWaiting || retryWaiting.CompletedAt != nil {
		t.Fatalf("retry waiting = %+v err=%v", retryWaiting, err)
	}
	retryRunning, err = svc.StartExecution(ctx, retryWaiting.Id, retryWaiting.Revision)
	if err != nil || retryRunning.Status != model.IntegrationExecutionStatusRunning {
		t.Fatalf("restart retry = %+v err=%v", retryRunning, err)
	}
	failed, err := svc.FailExecution(ctx, retryRunning.Id, request.IntegrationExecutionFailReq{
		Revision: retryRunning.Revision, TargetStatus: model.IntegrationExecutionStatusFailed,
		ErrorCategory: model.IntegrationErrorCategoryRemote, ResultSummary: "terminal",
	})
	if err != nil || failed.Status != model.IntegrationExecutionStatusFailed || failed.CompletedAt == nil {
		t.Fatalf("fail = %+v err=%v", failed, err)
	}

	cancelCreated, _ := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "cancel-001"))
	cancelled, err := svc.CancelExecution(ctx, cancelCreated.Id, cancelCreated.Revision)
	if err != nil || cancelled.Status != model.IntegrationExecutionStatusCancelled || cancelled.CancelledAt == nil {
		t.Fatalf("cancel = %+v err=%v", cancelled, err)
	}
	runningCancelCreated, _ := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "cancel-002"))
	runningCancel, _ := svc.StartExecution(ctx, runningCancelCreated.Id, runningCancelCreated.Revision)
	if _, err = svc.CancelExecution(ctx, runningCancel.Id, runningCancel.Revision); !errors.Is(err, apperrors.ErrIntegrationExecutionStatusInvalid) {
		t.Fatalf("running cancel error = %v", err)
	}

	retryFailCreated, _ := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "retry-fail"))
	retryFailRunning, _ := svc.StartExecution(ctx, retryFailCreated.Id, retryFailCreated.Revision)
	retryFailWaiting, _ := svc.FailExecution(ctx, retryFailRunning.Id, request.IntegrationExecutionFailReq{
		Revision: retryFailRunning.Revision, TargetStatus: model.IntegrationExecutionStatusRetryWaiting,
		ErrorCategory: model.IntegrationErrorCategoryTimeout,
	})
	if _, err = svc.FailExecution(ctx, retryFailWaiting.Id, request.IntegrationExecutionFailReq{
		Revision: retryFailWaiting.Revision, TargetStatus: model.IntegrationExecutionStatusFailed,
		ErrorCategory: model.IntegrationErrorCategoryTimeout,
	}); err != nil {
		t.Fatalf("retry waiting to failed: %v", err)
	}

	retryCancelCreated, _ := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "retry-cancel"))
	retryCancelRunning, _ := svc.StartExecution(ctx, retryCancelCreated.Id, retryCancelCreated.Revision)
	retryCancelWaiting, _ := svc.FailExecution(ctx, retryCancelRunning.Id, request.IntegrationExecutionFailReq{
		Revision: retryCancelRunning.Revision, TargetStatus: model.IntegrationExecutionStatusRetryWaiting,
		ErrorCategory: model.IntegrationErrorCategoryNetwork,
	})
	if _, err = svc.CancelExecution(ctx, retryCancelWaiting.Id, retryCancelWaiting.Revision); err != nil {
		t.Fatalf("retry waiting to cancelled: %v", err)
	}
	if len(auditWriter.records) != 20 {
		t.Fatalf("audit count = %d, want 20", len(auditWriter.records))
	}
}

func TestIntegrationExecutionServiceRejectsInvalidConfiguration(t *testing.T) {
	svc, db, system, definition, _ := newIntegrationExecutionTestSubject(t)
	ctx := context.Background()

	invalidHash := integrationExecutionCreateRequest(system.Id, definition.Id, "invalid-hash")
	invalidHash.InputHash = "not-a-hash"
	if _, err := svc.CreateExecution(ctx, invalidHash); !errors.Is(err, apperrors.ErrIntegrationExecutionConfigurationInvalid) {
		t.Fatalf("invalid hash error = %v", err)
	}
	from, to := model.Now(), model.Now().Add(-time.Hour)
	if _, err := svc.PageExecution(ctx, request.IntegrationExecutionQueryReq{CreatedFrom: &from, CreatedTo: &to},
		integrationExecutionServiceQueryTable(), integrationExecutionPermission(t, model.DataPermissionOperationQuery, datapermission.DataScopeDecisionNotApplicable)); !errors.Is(err, apperrors.ErrIntegrationExecutionConfigurationInvalid) {
		t.Fatalf("invalid time range error = %v", err)
	}

	if err := db.Model(&model.ExternalSystem{}).Where("id = ?", system.Id).
		Updates(map[string]any{"status": model.ExternalSystemStatusDisabled}).Error; err != nil {
		t.Fatalf("disable system fixture: %v", err)
	}
	if _, err := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "disabled-system")); !errors.Is(err, apperrors.ErrIntegrationExecutionConfigurationInvalid) {
		t.Fatalf("disabled system error = %v", err)
	}
}

func newIntegrationExecutionTestSubject(
	t *testing.T,
) (*IntegrationExecutionService, *gorm.DB, model.ExternalSystem, model.InterfaceDefinition, *integrationExecutionAuditWriter) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.InterfaceDefinition{}, &model.IntegrationExecution{}, &model.IntegrationLog{})
	system := model.ExternalSystem{
		Basic: model.Basic{Id: 100, State: true}, SystemCode: "runtime_hr", Name: "Runtime HR",
		SystemType: model.ExternalSystemTypeHR, BaseURL: "https://hr.example.com",
		OwnerIdentifier: "owner-runtime", OwnerName: "运行负责人",
		Status: model.ExternalSystemStatusEnabled, Revision: 1,
	}
	testutil.MustCreate(t, db, &system)
	definition := model.InterfaceDefinition{
		Basic: model.Basic{Id: 200, State: true}, ExternalSystemID: system.Id,
		InterfaceCode: "organization_list", Name: "组织列表", Version: 1,
		Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET,
		RelativePath: "/api/organizations", TimeoutSeconds: 30, ResponseLimit: 1024,
		Status: model.InterfaceDefinitionStatusEnabled, Revision: 1,
	}
	testutil.MustCreate(t, db, &definition)
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	primary := &database.PrimaryDB{DB: db}
	auditWriter := &integrationExecutionAuditWriter{}
	return NewIntegrationExecutionService(
		impl.NewIntegrationExecutionRepositoryImpl(primary),
		impl.NewIntegrationLogRepositoryImpl(primary),
		impl.NewExternalSystemRepositoryImpl(primary),
		impl.NewInterfaceDefinitionRepositoryImpl(primary),
		sf,
		auditWriter,
	), db, system, definition, auditWriter
}

func integrationExecutionCreateRequest(systemID int, interfaceID int, idempotencyKey string) request.IntegrationExecutionCreateReq {
	return request.IntegrationExecutionCreateReq{
		ExternalSystemID: systemID, InterfaceDefinitionID: interfaceID,
		TriggerSource: model.IntegrationTriggerSourceManual, IdempotencyScope: "acceptance",
		IdempotencyKey: idempotencyKey, InputHash: strings.Repeat("a", 64),
	}
}

func integrationExecutionServiceQueryTable() model.SysTable {
	return model.SysTable{Basic: model.Basic{Id: 501, State: true}, TableCode: "integration_execution", TableFields: []model.SysTableField{
		{Basic: model.Basic{Id: 1, State: true}, FieldCode: "execution_no", FieldType: enum.VarcharFieldType, IsQuickSearch: true, IsAdvancedSearch: true},
		{Basic: model.Basic{Id: 2, State: true}, FieldCode: "external_system_code", FieldType: enum.VarcharFieldType, IsQuickSearch: true, IsAdvancedSearch: true},
		{Basic: model.Basic{Id: 3, State: true}, FieldCode: "interface_code", FieldType: enum.VarcharFieldType, IsQuickSearch: true, IsAdvancedSearch: true},
		{Basic: model.Basic{Id: 4, State: true}, FieldCode: "external_system_id", FieldType: enum.BigIntFieldType, IsAdvancedSearch: true},
		{Basic: model.Basic{Id: 5, State: true}, FieldCode: "interface_definition_id", FieldType: enum.BigIntFieldType, IsAdvancedSearch: true},
		{Basic: model.Basic{Id: 6, State: true}, FieldCode: "trigger_source", FieldType: enum.VarcharFieldType, IsAdvancedSearch: true},
		{Basic: model.Basic{Id: 7, State: true}, FieldCode: "status", FieldType: enum.VarcharFieldType, IsAdvancedSearch: true},
		{Basic: model.Basic{Id: 8, State: true}, FieldCode: "gmt_create", FieldType: enum.DatetimeFieldType, IsAdvancedSearch: true},
	}}
}

func integrationExecutionPermission(
	t *testing.T,
	operation string,
	decision datapermission.DataScopeDecision,
) repository.GeneralizationPermission {
	t.Helper()
	resource, err := datapermission.NewAdapterResourceContext(datapermission.AdapterResourceContextInput{
		ResourceCode: "integration_execution", Operation: operation,
		AdapterCode: "metadata_filter", TableId: 501,
	})
	if err != nil {
		t.Fatalf("create resource context: %v", err)
	}
	result, err := datapermission.NewDataScopeResult(datapermission.DataScopeResultInput{
		ResourceCode: resource.ResourceCode(), Operation: operation, Decision: decision,
	})
	if err != nil {
		t.Fatalf("create scope result: %v", err)
	}
	input, err := datapermission.NewAdapterInput(resource, result, nil)
	if err != nil {
		t.Fatalf("create adapter input: %v", err)
	}
	execution, err := datapermission.BuildAdapterExecution(input)
	if err != nil {
		t.Fatalf("build adapter execution: %v", err)
	}
	return repository.GeneralizationPermission{AdapterExecution: &execution}
}
