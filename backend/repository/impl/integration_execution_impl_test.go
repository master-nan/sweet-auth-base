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
	"testing"
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
