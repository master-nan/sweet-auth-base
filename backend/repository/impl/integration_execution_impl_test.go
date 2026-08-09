package impl

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	"backend/internal/datapermission"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestIntegrationExecutionRepositoryDomainQueriesAndPagination(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.IntegrationExecution{}, &model.IntegrationLog{})
	primary := &database.PrimaryDB{DB: db}
	executions := NewIntegrationExecutionRepositoryImpl(primary)
	logs := NewIntegrationLogRepositoryImpl(primary)

	fixtures := []model.IntegrationExecution{
		repositoryExecutionFixture(1, "INT-001", model.IntegrationExecutionStatusCreated, "one"),
		repositoryExecutionFixture(2, "INT-002", model.IntegrationExecutionStatusSucceeded, "two"),
		repositoryExecutionFixture(3, "INT-003", model.IntegrationExecutionStatusCreated, "three"),
	}
	for index := range fixtures {
		testutil.MustCreate(t, db, &fixtures[index])
	}

	found, err := executions.FindByIdempotency(db, 20, 1, "repository", "two")
	if err != nil || found.ExecutionNo != "INT-002" {
		t.Fatalf("find by idempotency = %+v err=%v", found, err)
	}
	candidates, err := executions.ListCandidatesByStatus(
		context.Background(),
		[]string{model.IntegrationExecutionStatusCreated},
		1,
	)
	if err != nil || len(candidates) != 1 || candidates[0].ExecutionNo != "INT-001" {
		t.Fatalf("candidates = %+v err=%v", candidates, err)
	}

	basic := request.Basic{Page: 1, Num: 10, Filters: map[string]any{"status": model.IntegrationExecutionStatusCreated}}
	notApplicableExecution := generalizationRepositoryExecution(
		t, model.DataPermissionOperationQuery, datapermission.DataScopeDecisionNotApplicable, nil,
	)
	permission := repository.GeneralizationPermission{AdapterExecution: &notApplicableExecution}
	page, err := executions.GetIntegrationExecutionList(context.Background(), &basic, integrationExecutionQueryTableForTest(), permission)
	if err != nil || page.Total != 2 || len(page.Data) != 2 {
		t.Fatalf("page = %+v err=%v", page, err)
	}
	detailExecution := generalizationRepositoryExecution(
		t, model.DataPermissionOperationDetail, datapermission.DataScopeDecisionNotApplicable, nil,
	)
	if detail, detailErr := executions.FindByIDWithPermission(
		context.Background(), 1, integrationExecutionQueryTableForTest(),
		repository.GeneralizationPermission{AdapterExecution: &detailExecution},
	); detailErr != nil || detail.ExecutionNo != "INT-001" {
		t.Fatalf("permission detail = %+v err=%v", detail, detailErr)
	}

	startedAt := model.Now()
	second := model.IntegrationLog{Basic: model.Basic{Id: 102, State: true}, ExecutionID: 1, AttemptNo: 2, Status: model.IntegrationLogStatusFailed, StartedAt: startedAt, ResultCertainty: model.IntegrationResultCertaintyUnknown}
	first := model.IntegrationLog{Basic: model.Basic{Id: 101, State: true}, ExecutionID: 1, AttemptNo: 1, Status: model.IntegrationLogStatusFailed, StartedAt: startedAt, ResultCertainty: model.IntegrationResultCertaintyUnknown}
	testutil.MustCreate(t, db, &second)
	testutil.MustCreate(t, db, &first)
	logPage, err := logs.GetIntegrationLogList(
		context.Background(),
		request.IntegrationLogQueryReq{Page: 1, Num: 10, ExecutionID: 1, Order: request.Order{Field: "attempt_no", IsAsc: true}},
		integrationLogQueryTableForTest(),
		permission,
	)
	if err != nil || logPage.Total != 2 || len(logPage.Data) != 2 || logPage.Data[0].AttemptNo != 1 || logPage.Data[1].AttemptNo != 2 || logPage.Data[0].Execution.ExecutionNo != "INT-001" {
		t.Fatalf("log page = %+v err=%v", logPage, err)
	}
	mismatchedPage, err := logs.GetIntegrationLogList(
		context.Background(),
		request.IntegrationLogQueryReq{Page: 1, Num: 10, ExecutionID: 2},
		integrationLogQueryTableForTest(),
		permission,
	)
	if err != nil || mismatchedPage.Total != 0 || len(mismatchedPage.Data) != 0 {
		t.Fatalf("mismatched execution log page = %+v err=%v", mismatchedPage, err)
	}
}

func TestIntegrationExecutionRepositoryClaimCompleteAndRecover(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.IntegrationExecution{}, &model.IntegrationLog{})
	primary := &database.PrimaryDB{DB: db}
	executions := NewIntegrationExecutionRepositoryImpl(primary)
	now := model.Now()
	created := repositoryExecutionFixture(11, "INT-CLAIM-011", model.IntegrationExecutionStatusCreated, "claim-one")
	second := repositoryExecutionFixture(12, "INT-CLAIM-012", model.IntegrationExecutionStatusCreated, "claim-two")
	testutil.MustCreate(t, db, &created)
	testutil.MustCreate(t, db, &second)

	_, err := executions.CompleteAttemptAndExecution(context.Background(), repository.IntegrationAttemptCompletion{
		ExecutionID: created.Id, AttemptID: 999, AttemptNo: 1,
		WorkerID: "worker-a", ExpectedRevision: created.Revision, ExecutionStatus: model.IntegrationExecutionStatusSucceeded,
		CompletedAt: now, ResultCertainty: model.IntegrationResultCertaintyConfirmed,
	})
	if !errors.Is(err, repository.ErrIntegrationExecutionLeaseLost) {
		t.Fatalf("completion without lease or attempt error = %v", err)
	}

	claimed, err := executions.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
		WorkerID: "worker-a", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{101, 102}, RequestID: "request-1", TraceID: "trace-1",
	})
	if err != nil || len(claimed) != 2 || claimed[0].Attempt.AttemptNo != 1 || claimed[1].Attempt.AttemptNo != 1 {
		t.Fatalf("claim = %+v err=%v", claimed, err)
	}
	otherWorker, err := executions.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
		WorkerID: "worker-b", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{103},
	})
	if err != nil || len(otherWorker) != 0 {
		t.Fatalf("already claimed execution unexpectedly available: %+v err=%v", otherWorker, err)
	}

	completedAt := now.Add(time.Second)
	_, err = executions.CompleteAttemptAndExecution(context.Background(), repository.IntegrationAttemptCompletion{
		ExecutionID: claimed[0].Execution.Id, AttemptID: claimed[0].Attempt.Id, AttemptNo: 1,
		WorkerID: "worker-b", ExpectedRevision: claimed[0].Execution.Revision, ExecutionStatus: model.IntegrationExecutionStatusSucceeded,
		CompletedAt: completedAt, ResultCertainty: model.IntegrationResultCertaintyConfirmed,
	})
	if !errors.Is(err, repository.ErrIntegrationExecutionLeaseLost) {
		t.Fatalf("wrong lease completion error = %v", err)
	}
	completed, err := executions.CompleteAttemptAndExecution(context.Background(), repository.IntegrationAttemptCompletion{
		ExecutionID: claimed[0].Execution.Id, AttemptID: claimed[0].Attempt.Id, AttemptNo: 1,
		WorkerID: "worker-a", ExpectedRevision: claimed[0].Execution.Revision, ExecutionStatus: model.IntegrationExecutionStatusSucceeded,
		CompletedAt: completedAt, ResultCertainty: model.IntegrationResultCertaintyConfirmed, ResultSummary: "ok",
	})
	if err != nil || completed.Status != model.IntegrationExecutionStatusSucceeded || completed.LeaseExpiresAt != nil {
		t.Fatalf("completion = %+v err=%v", completed, err)
	}
	_, err = executions.CompleteAttemptAndExecution(context.Background(), repository.IntegrationAttemptCompletion{
		ExecutionID: claimed[0].Execution.Id, AttemptID: claimed[0].Attempt.Id, AttemptNo: 1,
		WorkerID: "worker-a", ExpectedRevision: completed.Revision, ExecutionStatus: model.IntegrationExecutionStatusSucceeded,
		CompletedAt: completedAt.Add(time.Second), ResultCertainty: model.IntegrationResultCertaintyConfirmed,
	})
	if !errors.Is(err, repository.ErrIntegrationExecutionLeaseLost) && !errors.Is(err, repository.ErrIntegrationAttemptAlreadyCompleted) {
		t.Fatalf("duplicate completion error = %v", err)
	}

	expired := now.Add(-time.Second)
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", claimed[1].Execution.Id).Update("lease_expires_at", expired).Error; err != nil {
		t.Fatalf("expire claimed lease: %v", err)
	}
	recovered, err := executions.RecoverExpiredExecution(context.Background(), repository.ExpiredExecutionRecovery{ExecutionID: claimed[1].Execution.Id})
	if err != nil || !recovered {
		t.Fatalf("recover expired execution = %v err=%v", recovered, err)
	}
	var log model.IntegrationLog
	if err := db.Where("execution_id = ?", claimed[1].Execution.Id).First(&log).Error; err != nil ||
		log.Status != model.IntegrationLogStatusFailed || log.ResultCertainty != model.IntegrationResultCertaintyUnknown {
		t.Fatalf("recovered attempt = %+v err=%v", log, err)
	}
}

func TestIntegrationExecutionRepositoryClaimRollsBackWhenAttemptCreationFails(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.IntegrationExecution{}, &model.IntegrationLog{})
	primary := &database.PrimaryDB{DB: db}
	executions := NewIntegrationExecutionRepositoryImpl(primary)
	created := repositoryExecutionFixture(21, "INT-CLAIM-021", model.IntegrationExecutionStatusCreated, "claim-rollback")
	testutil.MustCreate(t, db, &created)
	// 用已存在的主键模拟 Attempt 写入失败，验证领取状态和 current_attempt 一并回滚。
	existingAttempt := model.IntegrationLog{Basic: model.Basic{Id: 201, State: true}, ExecutionID: 999, AttemptNo: 1,
		Status: model.IntegrationLogStatusFailed, StartedAt: model.Now(), ResultCertainty: model.IntegrationResultCertaintyUnknown}
	testutil.MustCreate(t, db, &existingAttempt)
	now := model.Now()
	_, err := executions.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
		WorkerID: "worker-rollback", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{existingAttempt.Id},
	})
	if err == nil {
		t.Fatal("expected attempt creation failure")
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("load execution after rollback: %v", err)
	}
	if stored.Status != model.IntegrationExecutionStatusCreated || stored.CurrentAttempt != 0 || stored.LeaseOwner != "" || stored.LeaseExpiresAt != nil || stored.Revision != created.Revision {
		t.Fatalf("claim transaction did not roll back: %+v", stored)
	}
}

func TestIntegrationExecutionRepositoryClaimsDueRetryAndClosesUnrunnableCandidates(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.IntegrationExecution{}, &model.IntegrationLog{})
	executions := NewIntegrationExecutionRepositoryImpl(&database.PrimaryDB{DB: db})
	now := model.Now()

	due := repositoryRetryWaitingFixture(31, "INT-RETRY-031", now.Add(-time.Second), now.Add(-time.Minute), 1, 3)
	notDue := repositoryRetryWaitingFixture(32, "INT-RETRY-032", now.Add(time.Hour), now.Add(-time.Minute), 1, 3)
	exhausted := repositoryRetryWaitingFixture(33, "INT-RETRY-033", now.Add(-time.Second), now.Add(-time.Minute), 3, 3)
	expiredWindow := repositoryRetryWaitingFixture(34, "INT-RETRY-034", now.Add(-time.Second), now.Add(-2*time.Minute), 1, 3)
	invalidSnapshot := repositoryRetryWaitingFixture(35, "INT-RETRY-035", now.Add(-time.Second), now.Add(-time.Minute), 1, 3)
	invalidSnapshot.RetryPolicySnapshot = datatypes.JSON([]byte(`{"version":1}`))
	nullSchedule := repositoryRetryWaitingFixture(36, "INT-RETRY-036", now, now.Add(-time.Minute), 1, 3)
	nullSchedule.NextRunAt = nil
	for _, value := range []*model.IntegrationExecution{&due, &notDue, &exhausted, &expiredWindow, &invalidSnapshot, &nullSchedule} {
		testutil.MustCreate(t, db, value)
	}

	claimed, err := executions.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
		WorkerID: "retry-worker", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute),
		AttemptIDs: []int{301, 302, 303, 304, 305, 306},
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim due retry = %+v err=%v", claimed, err)
	}
	if claimed[0].Execution.Id != due.Id || claimed[0].Attempt.AttemptNo != 2 || claimed[0].Execution.CurrentAttempt != 2 ||
		claimed[0].Execution.NextRunAt != nil || claimed[0].Execution.StartedAt == nil || !claimed[0].Execution.StartedAt.Equal(*due.StartedAt) {
		t.Fatalf("unexpected claimed retry: %+v", claimed[0])
	}

	assertRetryExecutionState(t, db, notDue.Id, model.IntegrationExecutionStatusRetryWaiting, 1, "")
	assertRetryExecutionState(t, db, exhausted.Id, model.IntegrationExecutionStatusFailed, 3, "retry_attempts_exhausted")
	assertRetryExecutionState(t, db, expiredWindow.Id, model.IntegrationExecutionStatusFailed, 1, "retry_window_expired")
	assertRetryExecutionState(t, db, invalidSnapshot.Id, model.IntegrationExecutionStatusFailed, 1, "retry_policy_invalid")
	assertRetryExecutionState(t, db, nullSchedule.Id, model.IntegrationExecutionStatusRetryWaiting, 1, "")

	var attemptCount int64
	if err := db.Model(&model.IntegrationLog{}).Count(&attemptCount).Error; err != nil || attemptCount != 1 {
		t.Fatalf("only runnable retry may create attempt: count=%d err=%v", attemptCount, err)
	}
}

func TestIntegrationExecutionRepositoryRetryAttemptNumbersRemainMonotonic(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.IntegrationExecution{}, &model.IntegrationLog{})
	executions := NewIntegrationExecutionRepositoryImpl(&database.PrimaryDB{DB: db})
	now := model.Now()
	execution := repositoryRetryWaitingFixture(41, "INT-RETRY-041", now.Add(-time.Second), now.Add(-time.Minute), 1, 3)
	firstAttempt := model.IntegrationLog{
		Basic: model.Basic{Id: 401, State: true}, ExecutionID: execution.Id, AttemptNo: 1,
		Status: model.IntegrationLogStatusFailed, StartedAt: *execution.StartedAt,
		ResultCertainty: model.IntegrationResultCertaintyConfirmed,
	}
	testutil.MustCreate(t, db, &execution)
	testutil.MustCreate(t, db, &firstAttempt)

	claimed, err := executions.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
		WorkerID: "retry-worker", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{402},
	})
	if err != nil || len(claimed) != 1 || claimed[0].Attempt.AttemptNo != 2 {
		t.Fatalf("claim attempt 2 = %+v err=%v", claimed, err)
	}
	nextRunAt := now.Add(-time.Millisecond)
	completedAt := now.Add(time.Millisecond)
	completed, err := executions.CompleteAttemptAndExecution(context.Background(), repository.IntegrationAttemptCompletion{
		ExecutionID: execution.Id, AttemptID: claimed[0].Attempt.Id, AttemptNo: 2,
		WorkerID: "retry-worker", ExpectedRevision: claimed[0].Execution.Revision,
		ExecutionStatus: model.IntegrationExecutionStatusRetryWaiting, CompletedAt: completedAt,
		ErrorCategory: model.IntegrationErrorCategoryRemote, ResultCertainty: model.IntegrationResultCertaintyConfirmed,
		Retryable: true, RetryReasonCode: "retry_allowed", RetryDelayMs: 1000,
		RetryScheduledAt: &nextRunAt, RetryAfterSource: "local",
	})
	if err != nil || completed.Status != model.IntegrationExecutionStatusRetryWaiting || completed.NextRunAt == nil {
		t.Fatalf("schedule attempt 3 = %+v err=%v", completed, err)
	}
	claimed, err = executions.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
		WorkerID: "retry-worker", StartedAt: now.Add(2 * time.Millisecond), LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{403},
	})
	if err != nil || len(claimed) != 1 || claimed[0].Attempt.AttemptNo != 3 || claimed[0].Execution.CurrentAttempt != 3 {
		t.Fatalf("claim attempt 3 = %+v err=%v", claimed, err)
	}
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", execution.Id).
		Updates(map[string]any{"status": model.IntegrationExecutionStatusRetryWaiting, "next_run_at": now.Add(-time.Second), "lease_owner": "", "lease_expires_at": nil}).Error; err != nil {
		t.Fatalf("prepare exhausted retry: %v", err)
	}
	if err := db.Model(&model.IntegrationLog{}).Where("id = ?", claimed[0].Attempt.Id).
		Updates(map[string]any{"status": model.IntegrationLogStatusFailed, "ended_at": now.Add(3 * time.Millisecond)}).Error; err != nil {
		t.Fatalf("complete exhausted attempt fixture: %v", err)
	}
	claimed, err = executions.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
		WorkerID: "retry-worker", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{404},
	})
	if err != nil || len(claimed) != 0 {
		t.Fatalf("attempt 4 must not be created: %+v err=%v", claimed, err)
	}
	assertRetryExecutionState(t, db, execution.Id, model.IntegrationExecutionStatusFailed, 3, "retry_attempts_exhausted")
	var count int64
	if err := db.Model(&model.IntegrationLog{}).Where("execution_id = ?", execution.Id).Count(&count).Error; err != nil || count != 3 {
		t.Fatalf("attempt history count=%d err=%v", count, err)
	}
}

func repositoryRetryWaitingFixture(id int, number string, nextRunAt, startedAt time.Time, currentAttempt, maxAttempts int) model.IntegrationExecution {
	value := repositoryExecutionFixture(id, number, model.IntegrationExecutionStatusRetryWaiting, number)
	value.StartedAt = &startedAt
	value.LastAttemptAt = &startedAt
	value.NextRunAt = &nextRunAt
	value.CurrentAttempt = currentAttempt
	value.RetryPolicySnapshotVersion = 1
	value.RetryPolicySnapshot = datatypes.JSON([]byte(fmt.Sprintf(
		`{"version":1,"policy_code":"repository_retry","policy_version":1,"max_attempts":%d,"initial_delay_ms":1000,"max_delay_ms":1000,"backoff_type":"fixed","backoff_multiplier":"1","jitter_type":"none","jitter_ratio":"0","retry_window_ms":60000,"retryable_error_categories":["network","remote","timeout"],"retryable_http_statuses":[429,502,503,504],"respect_retry_after":true,"idempotency_mode":"safe_method","remote_idempotency_header":""}`,
		maxAttempts,
	)))
	return value
}

func assertRetryExecutionState(t *testing.T, db *gorm.DB, id int, status string, currentAttempt int, reason string) {
	t.Helper()
	var stored model.IntegrationExecution
	if err := db.First(&stored, id).Error; err != nil {
		t.Fatalf("load retry execution %d: %v", id, err)
	}
	if stored.Status != status || stored.CurrentAttempt != currentAttempt || stored.RetryReasonCode != reason {
		t.Fatalf("retry execution %d = %+v", id, stored)
	}
	if status != model.IntegrationExecutionStatusRetryWaiting && stored.NextRunAt != nil {
		t.Fatalf("terminal retry execution %d retained next_run_at", id)
	}
}

func repositoryExecutionFixture(id int, number, status, key string) model.IntegrationExecution {
	snapshot := datatypes.JSON([]byte(`{"version":1,"path_params":{},"query_params":{},"headers":{}}`))
	return model.IntegrationExecution{
		Basic: model.Basic{Id: id, State: true}, ExecutionNo: number,
		ExternalSystemID: 10, ExternalSystemCode: "repo_system", ExternalSystemName: "Repo System",
		InterfaceDefinitionID: 20, InterfaceCode: "repo_api", InterfaceName: "Repo API", InterfaceVersion: 1,
		TriggerSource: model.IntegrationTriggerSourceManual, Status: status,
		IdempotencyScope: "repository", IdempotencyKey: key,
		InputHash:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		InputSnapshot: snapshot, InputSnapshotVersion: model.IntegrationExecutionInputSnapshotVersion,
		InputSnapshotSize: len(snapshot), Revision: 1,
	}
}

func integrationExecutionQueryTableForTest() model.SysTable {
	return model.SysTable{TableCode: "integration_execution", TableFields: []model.SysTableField{
		{Basic: model.Basic{State: true}, FieldCode: "execution_no", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
		{Basic: model.Basic{State: true}, FieldCode: "external_system_id", FieldType: enum.BigIntFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "interface_definition_id", FieldType: enum.BigIntFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "trigger_source", FieldType: enum.VarcharFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "status", FieldType: enum.VarcharFieldType},
	}}
}

func integrationLogQueryTableForTest() model.SysTable {
	return model.SysTable{TableCode: "integration_log", TableFields: []model.SysTableField{
		{Basic: model.Basic{State: true}, FieldCode: "execution_id", FieldType: enum.BigIntFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "attempt_no", FieldType: enum.IntFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "status", FieldType: enum.VarcharFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "started_at", FieldType: enum.DatetimeFieldType},
	}}
}
