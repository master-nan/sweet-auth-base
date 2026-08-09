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
	"sync"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type integrationExecutionAuditWriter struct {
	mu          sync.Mutex
	records     []TransactionalAuditRecord
	subject     audit.AuditSubject
	correlation audit.CorrelationIDs
}

func (w *integrationExecutionAuditWriter) RecordTransactionalAuditContext(
	ctx context.Context,
	_ *gorm.DB,
	record TransactionalAuditRecord,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.records = append(w.records, record)
	w.subject, _ = audit.GetAuditSubject(ctx)
	w.correlation = audit.GetCorrelationIDs(ctx)
	return nil
}

func TestIntegrationExecutionServiceConcurrentIdempotencyConverges(t *testing.T) {
	svc, db, system, definition, _ := newIntegrationExecutionTestSubject(t)
	req := integrationExecutionCreateRequest(system.Id, definition.Id, "concurrent-request")
	const callers = 8
	ids := make(chan int, callers)
	errs := make(chan error, callers)
	var wait sync.WaitGroup
	for index := 0; index < callers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := svc.CreateExecution(context.Background(), req)
			if err != nil {
				errs <- err
				return
			}
			ids <- result.Id
		}()
	}
	wait.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent create error: %v", err)
	}
	unique := map[int]struct{}{}
	for id := range ids {
		unique[id] = struct{}{}
	}
	if len(unique) != 1 {
		t.Fatalf("concurrent execution ids = %v", unique)
	}
	var count int64
	if err := db.Model(&model.IntegrationExecution{}).
		Where("interface_definition_id = ? AND interface_version = ? AND idempotency_scope = ? AND idempotency_key = ?",
			definition.Id, definition.Version, req.IdempotencyScope, req.IdempotencyKey).
		Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("idempotent execution count=%d err=%v", count, err)
	}
}

func TestIntegrationExecutionServiceRejectsRuntimeIncompatibleInterface(t *testing.T) {
	svc, db, system, definition, _ := newIntegrationExecutionTestSubject(t)
	if err := db.Model(&model.InterfaceDefinition{}).Where("id = ?", definition.Id).Update("response_limit", int64(64*1024*1024+1)).Error; err != nil {
		t.Fatalf("prepare incompatible interface: %v", err)
	}
	_, err := svc.CreateExecution(context.Background(), integrationExecutionCreateRequest(system.Id, definition.Id, "runtime-incompatible"))
	if !errors.Is(err, apperrors.ErrIntegrationExecutionRuntimeIncompatible) {
		t.Fatalf("runtime-incompatible execution error = %v", err)
	}
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
	conflictReq.Input.QueryParams = map[string][]string{"page": {"2"}}
	if _, err = svc.CreateExecution(ctx, conflictReq); !errors.Is(err, apperrors.ErrIntegrationExecutionIdempotencyConflict) {
		t.Fatalf("idempotency conflict error = %v", err)
	}
	forgedHashReq := integrationExecutionCreateRequest(system.Id, definition.Id, "forged-hash")
	forgedHashReq.InputHash = strings.Repeat("b", 64)
	if _, err = svc.CreateExecution(ctx, forgedHashReq); !errors.Is(err, apperrors.ErrIntegrationExecutionInputHashMismatch) {
		t.Fatalf("forged input hash error = %v", err)
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
	logTable := table
	logTable.TableCode = "integration_log"
	logTable.TableFields = []model.SysTableField{
		{Basic: model.Basic{Id: 21, State: true}, FieldCode: "execution_id", FieldType: enum.BigIntFieldType, IsAdvancedSearch: true},
		{Basic: model.Basic{Id: 22, State: true}, FieldCode: "attempt_no", FieldType: enum.IntFieldType, IsAdvancedSearch: true, IsSort: true},
		{Basic: model.Basic{Id: 23, State: true}, FieldCode: "status", FieldType: enum.VarcharFieldType, IsAdvancedSearch: true},
	}
	detail, err := svc.GetExecution(ctx, created.Id, table,
		integrationExecutionPermission(t, model.DataPermissionOperationDetail, datapermission.DataScopeDecisionNotApplicable))
	if err != nil || detail.Id != created.Id || detail.CurrentAttempt != 0 {
		t.Fatalf("detail = %+v err=%v", detail, err)
	}
	if detail.InputSummary.SnapshotVersion != model.IntegrationExecutionInputSnapshotVersion ||
		detail.InputSummary.QueryCount != 1 || detail.InputSummary.SizeBytes <= 0 {
		t.Fatalf("input summary = %+v", detail.InputSummary)
	}
	logPage, err := svc.PageLogs(ctx, request.IntegrationLogQueryReq{Page: 1, Num: 10, ExecutionID: created.Id}, logTable,
		integrationExecutionPermission(t, model.DataPermissionOperationQuery, datapermission.DataScopeDecisionNotApplicable))
	if err != nil || logPage.Total != 1 || len(logPage.Data) != 1 || logPage.Data[0].AttemptNo != 1 {
		t.Fatalf("independent log page = %+v err=%v", logPage, err)
	}
	if _, err = svc.GetLog(ctx, log.Id, logTable,
		integrationExecutionPermission(t, model.DataPermissionOperationDetail, datapermission.DataScopeDecisionNone)); !errors.Is(err, apperrors.ErrIntegrationExecutionNotFound) {
		t.Fatalf("denied log detail error = %v", err)
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
	for _, forbidden := range []string{
		"authorization", "secret", "ciphertext", "payload", "gmt_delete", `"attempts":`,
		"http_status", "result_certainty", "request_id", "trace_id",
	} {
		if strings.Contains(strings.ToLower(string(payload)), forbidden) {
			t.Fatalf("detail leaked %q: %s", forbidden, payload)
		}
	}
}

func TestIntegrationExecutionServiceStateMachineAndRevision(t *testing.T) {
	svc, db, system, definition, auditWriter := newIntegrationExecutionTestSubject(t)
	ctx := context.Background()

	created, err := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "cancel-created"))
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	if created.Status != model.IntegrationExecutionStatusCreated || created.CurrentAttempt != 0 {
		t.Fatalf("creation bypassed worker claim: %+v", created)
	}
	if _, err = svc.CancelExecution(ctx, created.Id, created.Revision+1); !errors.Is(err, apperrors.ErrIntegrationExecutionRevisionConflict) {
		t.Fatalf("stale cancel error = %v", err)
	}
	cancelled, err := svc.CancelExecution(ctx, created.Id, created.Revision)
	if err != nil || cancelled.Status != model.IntegrationExecutionStatusCancelled || cancelled.CancelledAt == nil {
		t.Fatalf("cancel created = %+v err=%v", cancelled, err)
	}
	if _, err = svc.CancelExecution(ctx, cancelled.Id, cancelled.Revision); !errors.Is(err, apperrors.ErrIntegrationExecutionStatusInvalid) {
		t.Fatalf("terminal cancel error = %v", err)
	}

	retryWaiting, err := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "cancel-retry"))
	if err != nil {
		t.Fatalf("create retry fixture: %v", err)
	}
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", retryWaiting.Id).Updates(map[string]any{
		"status": model.IntegrationExecutionStatusRetryWaiting, "revision": retryWaiting.Revision + 1,
	}).Error; err != nil {
		t.Fatalf("prepare retry waiting fixture: %v", err)
	}
	retryWaiting.Revision++
	retryCancelled, err := svc.CancelExecution(ctx, retryWaiting.Id, retryWaiting.Revision)
	if err != nil || retryCancelled.Status != model.IntegrationExecutionStatusCancelled {
		t.Fatalf("cancel retry waiting = %+v err=%v", retryCancelled, err)
	}

	running, err := svc.CreateExecution(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "running-reject"))
	if err != nil {
		t.Fatalf("create running fixture: %v", err)
	}
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", running.Id).Updates(map[string]any{
		"status": model.IntegrationExecutionStatusRunning, "revision": running.Revision + 1,
		"lease_owner": "worker-only", "current_attempt": 1,
	}).Error; err != nil {
		t.Fatalf("prepare running fixture: %v", err)
	}
	if _, err = svc.CancelExecution(ctx, running.Id, running.Revision+1); !errors.Is(err, apperrors.ErrIntegrationExecutionStatusInvalid) {
		t.Fatalf("running cancel error = %v", err)
	}
	if len(auditWriter.records) != 5 {
		t.Fatalf("audit count = %d, want 5", len(auditWriter.records))
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

func TestIntegrationExecutionServiceFreezesRetryPolicySnapshot(t *testing.T) {
	svc, db, system, definition, _ := newIntegrationExecutionTestSubject(t)
	policy := model.RetryPolicy{
		Basic: model.Basic{Id: 850, State: true}, PolicyCode: "runtime_retry", PolicyName: "Runtime Retry",
		Version: 1, Status: model.RetryPolicyStatusEnabled, MaxAttempts: 3,
		InitialDelayMs: 5000, MaxDelayMs: 300000, BackoffType: model.RetryBackoffTypeExponential,
		BackoffMultiplier: 2, JitterType: model.RetryJitterTypeFull, JitterRatio: 1,
		RetryWindowMs: 86400000, RetryableErrorCategories: datatypes.JSON([]byte(`["network","timeout","remote"]`)),
		RetryableHTTPStatuses: datatypes.JSON([]byte(`[429,502,503,504]`)), RespectRetryAfter: true, Revision: 1,
	}
	testutil.MustCreate(t, db, &policy)
	if err := db.Model(&model.InterfaceDefinition{}).Where("id = ?", definition.Id).Update("retry_policy_id", policy.Id).Error; err != nil {
		t.Fatalf("attach retry policy: %v", err)
	}
	created, err := svc.CreateExecution(context.Background(), integrationExecutionCreateRequest(system.Id, definition.Id, "retry-snapshot"))
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	if created.RetryPolicy == nil || created.RetryPolicy.PolicyCode != policy.PolicyCode || created.RetryPolicy.PolicyVersion != 1 || created.RetryPolicy.MaxAttempts != 3 {
		t.Fatalf("retry summary=%+v", created.RetryPolicy)
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("load execution: %v", err)
	}
	before := append([]byte(nil), stored.RetryPolicySnapshot...)
	if stored.RetryPolicySnapshotVersion != 1 || stored.RetryPolicyID == nil || *stored.RetryPolicyID != policy.Id {
		t.Fatalf("stored retry snapshot metadata=%+v", stored)
	}
	if stored.RemoteIdempotencyMode != model.InterfaceIdempotencyModeSafeMethod ||
		!strings.Contains(string(stored.RetryPolicySnapshot), `"idempotency_mode":"safe_method"`) {
		t.Fatalf("remote idempotency was not frozen: %+v snapshot=%s", stored, stored.RetryPolicySnapshot)
	}
	if err := db.Model(&model.RetryPolicy{}).Where("id = ?", policy.Id).Updates(map[string]any{"status": model.RetryPolicyStatusDisabled, "max_attempts": 9}).Error; err != nil {
		t.Fatalf("mutate source policy: %v", err)
	}
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("reload execution: %v", err)
	}
	if string(stored.RetryPolicySnapshot) != string(before) || !strings.Contains(string(stored.RetryPolicySnapshot), `"max_attempts":3`) {
		t.Fatalf("execution snapshot drifted: %s", stored.RetryPolicySnapshot)
	}
}

func TestIntegrationExecutionServiceGeneratesRemoteIdempotencyKey(t *testing.T) {
	svc, db, system, definition, _ := newIntegrationExecutionTestSubject(t)
	policy := model.RetryPolicy{
		Basic: model.Basic{Id: 851, State: true}, PolicyCode: "remote_key_retry", PolicyName: "Remote Key Retry",
		Version: 1, Status: model.RetryPolicyStatusEnabled, MaxAttempts: 2,
		InitialDelayMs: 1000, MaxDelayMs: 2000, BackoffType: model.RetryBackoffTypeFixed, BackoffMultiplier: 1,
		JitterType: model.RetryJitterTypeFull, JitterRatio: 1, RetryWindowMs: 60000,
		RetryableErrorCategories: datatypes.JSON([]byte(`["remote"]`)), RetryableHTTPStatuses: datatypes.JSON([]byte(`[503]`)), Revision: 1,
	}
	testutil.MustCreate(t, db, &policy)
	if err := validateRetryPolicyConfiguration(policy); err != nil {
		t.Fatalf("retry policy fixture invalid: %v", err)
	}
	if err := db.Model(&model.InterfaceDefinition{}).Where("id = ?", definition.Id).Updates(map[string]any{
		"http_method": model.InterfaceMethodPOST, "retry_policy_id": policy.Id,
		"idempotency_mode": model.InterfaceIdempotencyModeRemoteKeyHeader, "remote_idempotency_header": "Idempotency-Key",
	}).Error; err != nil {
		t.Fatalf("prepare remote key interface: %v", err)
	}
	created, err := svc.CreateExecution(context.Background(), integrationExecutionCreateRequest(system.Id, definition.Id, "remote-key"))
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("load execution: %v", err)
	}
	if stored.RemoteIdempotencyMode != model.InterfaceIdempotencyModeRemoteKeyHeader || len(stored.RemoteIdempotencyKey) != 64 ||
		strings.Contains(string(stored.RetryPolicySnapshot), stored.RemoteIdempotencyKey) {
		t.Fatalf("remote key freeze=%+v snapshot=%s", stored, stored.RetryPolicySnapshot)
	}
}

func newIntegrationExecutionTestSubject(
	t *testing.T,
) (*IntegrationExecutionService, *gorm.DB, model.ExternalSystem, model.InterfaceDefinition, *integrationExecutionAuditWriter) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.RetryPolicy{}, &model.InterfaceDefinition{}, &model.IntegrationExecution{}, &model.IntegrationLog{})
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
		InputContract: datatypes.JSON([]byte(`{"version":1,"parameters":[{"code":"page","location":"query","data_type":"integer","required":false,"max_length":8,"allow_multiple":false,"sensitive":false}]}`)),
		Status:        model.InterfaceDefinitionStatusEnabled, IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod, Revision: 1,
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
		impl.NewRetryPolicyRepositoryImpl(primary),
		sf,
		auditWriter,
	), db, system, definition, auditWriter
}

func integrationExecutionCreateRequest(systemID int, interfaceID int, idempotencyKey string) request.IntegrationExecutionCreateReq {
	return request.IntegrationExecutionCreateReq{
		ExternalSystemID: systemID, InterfaceDefinitionID: interfaceID,
		TriggerSource: model.IntegrationTriggerSourceManual, IdempotencyScope: "acceptance",
		IdempotencyKey: idempotencyKey,
		Input:          request.IntegrationExecutionInputReq{QueryParams: map[string][]string{"page": {"1"}}},
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
