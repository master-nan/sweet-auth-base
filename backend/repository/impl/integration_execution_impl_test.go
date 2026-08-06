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
	"testing"
	"time"
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
	items, err := logs.ListByExecutionID(context.Background(), 1)
	if err != nil || len(items) != 2 || items[0].AttemptNo != 1 || items[1].AttemptNo != 2 {
		t.Fatalf("logs = %+v err=%v", items, err)
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

	claimed, err := executions.ClaimCreatedExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
		WorkerID: "worker-a", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{101, 102}, RequestID: "request-1", TraceID: "trace-1",
	})
	if err != nil || len(claimed) != 2 || claimed[0].Attempt.AttemptNo != 1 || claimed[1].Attempt.AttemptNo != 1 {
		t.Fatalf("claim = %+v err=%v", claimed, err)
	}
	otherWorker, err := executions.ClaimCreatedExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
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
	recovered, err := executions.RecoverExpiredExecution(context.Background(), repository.ExpiredExecutionRecovery{ExecutionID: claimed[1].Execution.Id, RecoveredAt: now})
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
	_, err := executions.ClaimCreatedExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
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

func repositoryExecutionFixture(id int, number, status, key string) model.IntegrationExecution {
	return model.IntegrationExecution{
		Basic: model.Basic{Id: id, State: true}, ExecutionNo: number,
		ExternalSystemID: 10, ExternalSystemCode: "repo_system", ExternalSystemName: "Repo System",
		InterfaceDefinitionID: 20, InterfaceCode: "repo_api", InterfaceName: "Repo API", InterfaceVersion: 1,
		TriggerSource: model.IntegrationTriggerSourceManual, Status: status,
		IdempotencyScope: "repository", IdempotencyKey: key,
		InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Revision: 1,
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
