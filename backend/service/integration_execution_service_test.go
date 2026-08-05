package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestIntegrationExecutionServiceCreateIdempotencyDetailAndPage(t *testing.T) {
	svc, db, system, definition := newIntegrationExecutionTestSubject(t)
	ctx := context.Background()
	req := integrationExecutionCreateRequest(system.Id, definition.Id, "request-001")

	created, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	if created.ExecutionNo == "" || created.Status != model.IntegrationExecutionStatusCreated || created.Revision != 1 {
		t.Fatalf("unexpected created execution: %+v", created)
	}
	if created.ExternalSystem.SystemCode != system.SystemCode ||
		created.Interface.InterfaceCode != definition.InterfaceCode ||
		created.Interface.Version != definition.Version {
		t.Fatalf("execution snapshots were not derived from configuration: %+v", created)
	}

	hit, err := svc.Create(ctx, req)
	if err != nil || hit.Id != created.Id || hit.ExecutionNo != created.ExecutionNo {
		t.Fatalf("idempotency hit = %+v err=%v", hit, err)
	}
	conflictReq := req
	conflictReq.InputHash = strings.Repeat("b", 64)
	if _, err := svc.Create(ctx, conflictReq); !errors.Is(err, apperrors.ErrIntegrationExecutionIdempotencyConflict) {
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
	detail, err := svc.Get(ctx, created.Id)
	if err != nil || len(detail.Attempts) != 1 || detail.Attempts[0].AttemptNo != 1 {
		t.Fatalf("detail = %+v err=%v", detail, err)
	}

	page, err := svc.Page(ctx, request.IntegrationExecutionQueryReq{
		Page: 1, Num: 10, Status: model.IntegrationExecutionStatusCreated,
		QuickQuery: &request.QuickQuery{Keyword: "INT-"},
	}, integrationExecutionServiceQueryTable())
	if err != nil || page.Total != 1 || len(page.Data) != 1 || page.Data[0].Id != created.Id {
		t.Fatalf("page = %+v err=%v", page, err)
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

func TestIntegrationExecutionServiceCancelStateAndRevisionRules(t *testing.T) {
	svc, db, system, definition := newIntegrationExecutionTestSubject(t)
	ctx := context.Background()
	created, err := svc.Create(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "cancel-001"))
	if err != nil {
		t.Fatalf("create cancel fixture: %v", err)
	}

	if _, err := svc.Cancel(ctx, created.Id, created.Revision+1); !errors.Is(err, apperrors.ErrIntegrationExecutionRevisionConflict) {
		t.Fatalf("stale revision error = %v", err)
	}
	cancelled, err := svc.Cancel(ctx, created.Id, created.Revision)
	if err != nil {
		t.Fatalf("cancel created execution: %v", err)
	}
	if cancelled.Status != model.IntegrationExecutionStatusCancelled || cancelled.Revision != created.Revision+1 ||
		cancelled.CancelledAt == nil || cancelled.CompletedAt == nil {
		t.Fatalf("unexpected cancelled execution: %+v", cancelled)
	}
	if _, err := svc.Cancel(ctx, cancelled.Id, cancelled.Revision); !errors.Is(err, apperrors.ErrIntegrationExecutionStatusInvalid) {
		t.Fatalf("terminal cancel error = %v", err)
	}

	retrying, err := svc.Create(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "cancel-002"))
	if err != nil {
		t.Fatalf("create retry fixture: %v", err)
	}
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", retrying.Id).
		Updates(map[string]any{"status": model.IntegrationExecutionStatusRetryWaiting}).Error; err != nil {
		t.Fatalf("set retry waiting: %v", err)
	}
	if _, err := svc.Cancel(ctx, retrying.Id, retrying.Revision); err != nil {
		t.Fatalf("cancel retry waiting execution: %v", err)
	}

	running, err := svc.Create(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "cancel-003"))
	if err != nil {
		t.Fatalf("create running fixture: %v", err)
	}
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", running.Id).
		Updates(map[string]any{"status": model.IntegrationExecutionStatusRunning}).Error; err != nil {
		t.Fatalf("set running: %v", err)
	}
	if _, err := svc.Cancel(ctx, running.Id, running.Revision); !errors.Is(err, apperrors.ErrIntegrationExecutionStatusInvalid) {
		t.Fatalf("running cancel error = %v", err)
	}
}

func TestIntegrationExecutionServiceRejectsInvalidConfiguration(t *testing.T) {
	svc, db, system, definition := newIntegrationExecutionTestSubject(t)
	ctx := context.Background()

	invalidHash := integrationExecutionCreateRequest(system.Id, definition.Id, "invalid-hash")
	invalidHash.InputHash = "not-a-hash"
	if _, err := svc.Create(ctx, invalidHash); !errors.Is(err, apperrors.ErrIntegrationExecutionConfigurationInvalid) {
		t.Fatalf("invalid hash error = %v", err)
	}

	if err := db.Model(&model.ExternalSystem{}).Where("id = ?", system.Id).
		Updates(map[string]any{"status": model.ExternalSystemStatusDisabled}).Error; err != nil {
		t.Fatalf("disable system fixture: %v", err)
	}
	if _, err := svc.Create(ctx, integrationExecutionCreateRequest(system.Id, definition.Id, "disabled-system")); !errors.Is(err, apperrors.ErrIntegrationExecutionConfigurationInvalid) {
		t.Fatalf("disabled system error = %v", err)
	}
}

func newIntegrationExecutionTestSubject(
	t *testing.T,
) (*IntegrationExecutionService, *gorm.DB, model.ExternalSystem, model.InterfaceDefinition) {
	t.Helper()
	db := testutil.OpenSQLite(
		t,
		&model.ExternalSystem{},
		&model.InterfaceDefinition{},
		&model.IntegrationExecution{},
		&model.IntegrationLog{},
	)
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
	return NewIntegrationExecutionService(
		impl.NewIntegrationExecutionRepositoryImpl(primary),
		impl.NewIntegrationLogRepositoryImpl(primary),
		impl.NewExternalSystemRepositoryImpl(primary),
		impl.NewInterfaceDefinitionRepositoryImpl(primary),
		sf,
	), db, system, definition
}

func integrationExecutionCreateRequest(
	systemID int,
	interfaceID int,
	idempotencyKey string,
) request.IntegrationExecutionCreateReq {
	return request.IntegrationExecutionCreateReq{
		ExternalSystemID: systemID, InterfaceDefinitionID: interfaceID,
		TriggerSource:    model.IntegrationTriggerSourceManual,
		IdempotencyScope: "acceptance", IdempotencyKey: idempotencyKey,
		InputHash: strings.Repeat("a", 64),
	}
}

func integrationExecutionServiceQueryTable() model.SysTable {
	return model.SysTable{TableCode: "integration_execution", TableFields: []model.SysTableField{
		{Basic: model.Basic{State: true}, FieldCode: "execution_no", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
		{Basic: model.Basic{State: true}, FieldCode: "external_system_code", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
		{Basic: model.Basic{State: true}, FieldCode: "interface_code", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
		{Basic: model.Basic{State: true}, FieldCode: "external_system_id", FieldType: enum.BigIntFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "interface_definition_id", FieldType: enum.BigIntFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "trigger_source", FieldType: enum.VarcharFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "status", FieldType: enum.VarcharFieldType},
	}}
}
