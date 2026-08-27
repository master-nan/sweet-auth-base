package service

import (
	"backend/internal/database"
	"backend/internal/integration"
	"backend/internal/organization/hrsync"
	"backend/internal/security"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type organizationHRAcceptanceTask struct {
	taskCode     string
	consumerCode string
	path         string
}

func TestOrganizationHRPostgreSQLFullInitializationOrderE2E(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	tasks := []organizationHRAcceptanceTask{
		{"accept_hr_legal_entity", hrsync.ConsumerCodeLegalEntity, "/legal-entities"},
		{"accept_hr_legal_department", hrsync.ConsumerCodeLegalDepartment, "/legal-departments"},
		{"accept_hr_management_company", hrsync.ConsumerCodeManagementCompany, "/management-companies"},
		{"accept_hr_management_department", hrsync.ConsumerCodeManagementDepartment, "/management-departments"},
		{"accept_hr_position", hrsync.ConsumerCodePosition, "/positions"},
		{"accept_hr_employee", hrsync.ConsumerCodeEmployee, "/employees"},
		{"accept_hr_resigned_employee", hrsync.ConsumerCodeResignedEmployee, "/resigned-employees"},
	}

	var legalCalls atomic.Int32
	var callMu sync.Mutex
	callOrder := make([]string, 0, len(tasks)+1)
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		callMu.Lock()
		callOrder = append(callOrder, request.URL.Path)
		callMu.Unlock()
		if request.Header.Get("Authorization") != "Bearer hr-acceptance-token" ||
			request.URL.Query().Get("changed_since") == "" || len(request.URL.Query()) != 1 {
			t.Errorf("unsafe or invalid HR request: path=%s query=%s", request.URL.Path, request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/legal-entities" && legalCalls.Add(1) == 1 {
			writer.WriteHeader(http.StatusServiceUnavailable)
			_, _ = writer.Write([]byte(`{"success":false,"data":[]}`))
			return
		}
		lower, err := time.Parse(time.RFC3339Nano, request.URL.Query().Get("changed_since"))
		if err != nil {
			t.Errorf("parse changed_since: %v", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		changed := lower.UTC().Add(10 * time.Minute).Format("2006-01-02T15:04:05")
		data := organizationHRAcceptanceResponse(request.URL.Path, changed, lower)
		if data == nil {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": data})
	}))
	defer tlsServer.Close()

	primary := &database.PrimaryDB{DB: db}
	snowflake, err := utils.NewSnowflake(13)
	if err != nil {
		t.Fatal(err)
	}
	domain := NewOrganizationHRSyncService(impl.NewOrganizationHRSyncRepositoryImpl(primary), snowflake)
	sourceContract, err := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	registrations, err := hrsync.EnabledConsumerRegistrations(domain, sourceContract)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := integration.NewStaticSyncConsumerRegistry(registrations...)
	if err != nil {
		t.Fatal(err)
	}

	transport := newOrganizationHRAcceptanceTransport(t, tlsServer)
	protector, err := security.NewCredentialSecretProtectorWithKey("organization-hr-acceptance-master-key")
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := json.Marshal(map[string]string{"token": "hr-acceptance-token"})
	envelope, err := protector.Seal(secret)
	if err != nil {
		t.Fatal(err)
	}
	system := model.ExternalSystem{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SystemCode: "hr_acceptance", Name: "HR Acceptance",
		SystemType: model.ExternalSystemTypeHR, BaseURL: "https://hr-acceptance.test", OwnerIdentifier: "acceptance",
		OwnerName: "Acceptance", Status: model.ExternalSystemStatusEnabled, Revision: 1,
	}
	credential := model.Credential{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExternalSystemID: system.Id,
		CredentialCode: "hr_acceptance_token", Name: "HR Acceptance Token", CredentialType: model.CredentialTypeBearerToken,
		Status: model.CredentialStatusActive, SecretStorageRef: envelope.StorageRef, SecretCiphertext: envelope.Ciphertext,
		SecretNonce: envelope.Nonce, SecretFingerprint: envelope.Fingerprint, Version: 1, Revision: 1,
	}
	retryPolicy := model.RetryPolicy{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, PolicyCode: "hr_acceptance_retry", PolicyName: "HR Acceptance Retry",
		Version: 1, Status: model.RetryPolicyStatusEnabled, MaxAttempts: 2, InitialDelayMs: 1000, MaxDelayMs: 1000,
		BackoffType: model.RetryBackoffTypeFixed, BackoffMultiplier: 1, JitterType: model.RetryJitterTypeNone,
		JitterRatio: 0, RetryWindowMs: 60000, RetryableErrorCategories: datatypes.JSON([]byte(`["network","remote","timeout"]`)),
		RetryableHTTPStatuses: datatypes.JSON([]byte(`[429,502,503,504]`)), RespectRetryAfter: true, Revision: 1,
	}
	if _, err := integration.BuildRetryPolicySnapshot(retryPolicy, integration.RetryPolicySnapshotOptions{IdempotencyMode: integration.RemoteIdempotencySafeMethod}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{&system, &credential, &retryPolicy} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&model.RetryPolicy{}).Where("id = ?", retryPolicy.Id).Update("jitter_ratio", 0).Error; err != nil {
		t.Fatal(err)
	}
	var storedRetryPolicy model.RetryPolicy
	if err := db.First(&storedRetryPolicy, retryPolicy.Id).Error; err != nil {
		t.Fatal(err)
	}
	if err := validateRetryPolicyConfiguration(storedRetryPolicy); err != nil {
		t.Fatalf("stored retry fixture is invalid: %+v err=%v", storedRetryPolicy, err)
	}

	var databaseNow time.Time
	if err := db.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&databaseNow).Error; err != nil {
		t.Fatal(err)
	}
	checkpoint := databaseNow.UTC().Add(-2 * time.Hour).Truncate(time.Second)
	farFuture := databaseNow.UTC().Add(24 * time.Hour)
	planRaw, _ := json.Marshal(integration.SyncExecutionInputPlan{
		Version: integration.SyncExecutionInputPlanVersionV2, WindowMode: integration.SyncWindowModeLowerBoundOnly,
		StaticInput:        integration.ExecutionInputValues{},
		WindowStartBinding: &integration.SyncWindowBinding{Location: "query", Code: "changed_since", Format: "rfc3339"},
	})
	contractRaw, _ := json.Marshal(integration.InterfaceInputContract{Version: 1, Parameters: []integration.InputParameterDefinition{
		{Code: "changed_since", Location: "query", DataType: "string", Required: true, MaxLength: 64},
	}})
	for _, item := range tasks {
		definition := model.InterfaceDefinition{
			Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExternalSystemID: system.Id, InterfaceCode: item.taskCode,
			Name: item.taskCode, Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET,
			RelativePath: item.path, CredentialID: &credential.Id, TimeoutSeconds: 5, ResponseLimit: 1 << 20,
			InputContract: contractRaw, RetryPolicyID: &retryPolicy.Id, IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod,
			Status: model.InterfaceDefinitionStatusEnabled, Revision: 1,
		}
		task := model.IntegrationSyncTask{
			Basic: model.Basic{Id: nextSyncTestID(), State: true}, TaskCode: item.taskCode, TaskName: item.taskCode,
			Version: 1, Status: model.IntegrationSyncTaskStatusEnabled, ExternalSystemID: system.Id, InterfaceDefinitionID: definition.Id,
			ConsumerCode: item.consumerCode, ConsumerVersion: 1, ScheduleType: model.IntegrationSyncScheduleCron,
			CronExpression: "0 0 * * *", Timezone: "UTC", NextScheduledAt: &farFuture,
			CheckpointMode: model.IntegrationSyncCheckpointTimestamp, InitialCheckpointAt: &checkpoint, CheckpointAt: &checkpoint,
			LookbackSeconds: 0, WindowSliceSeconds: 14400, InputPlan: datatypes.JSON(planRaw), Revision: 1,
		}
		if err := db.Create(&definition).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&task).Error; err != nil {
			t.Fatal(err)
		}
	}

	executions := impl.NewIntegrationExecutionRepositoryImpl(primary)
	systems := impl.NewExternalSystemRepositoryImpl(primary)
	interfaces := impl.NewInterfaceDefinitionRepositoryImpl(primary)
	credentials := impl.NewCredentialRepositoryImpl(primary)
	batches := impl.NewIntegrationSyncBatchRepositoryImpl(primary)
	taskRepository := impl.NewIntegrationSyncTaskRepositoryImpl(primary)
	executionService := NewIntegrationExecutionService(executions, impl.NewIntegrationLogRepositoryImpl(primary), systems, interfaces,
		impl.NewRetryPolicyRepositoryImpl(primary), snowflake, &integrationExecutionAuditWriter{})
	coordinator := NewIntegrationSyncCoordinator(taskRepository, batches, executions, systems, interfaces, executionService,
		integration.NewPersistedSyncBusinessResultProvider(), snowflake)
	syncRunner, err := integration.NewIntegrationSyncRunner(coordinator, integration.SyncRunnerConfig{
		Enabled: true, RunnerID: "hr-acceptance-sync", PollInterval: time.Second, ScheduleBatchSize: 8,
		CoordinateBatchSize: 8, ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	provider := integration.NewCredentialProvider(credentials, interfaces, protector)
	guard, _ := integration.NewInMemoryConcurrencyGuard(4, 4, 4)
	leaseDuration := 10 * time.Minute
	engine, err := integration.NewIntegrationExecutionEngine(executions, systems, interfaces, credentials, batches, provider, transport, guard, registry,
		snowflake, integration.ExecutionEngineOptions{WorkerID: "hr-acceptance-worker", LeaseDuration: leaseDuration, BatchSize: 4})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := integration.NewIntegrationWorkerRunner(engine, integration.WorkerRunnerConfig{
		Enabled: true, WorkerID: "hr-acceptance-worker", PollInterval: time.Second, ClaimBatchSize: 4, InstanceConcurrency: 4,
		LeaseRecoveryInterval: 10 * time.Second, ShutdownTimeout: 5 * time.Second, LeaseDuration: leaseDuration,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := worker.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := syncRunner.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syncRunner.Stop(context.Background()); _ = worker.Stop(context.Background()) })

	for index, item := range tasks {
		if index == len(tasks)-1 {
			seedOrganizationHRAcceptanceAssignment(t, db, checkpoint)
		}
		if err := db.Model(&model.IntegrationSyncTask{}).Where("task_code = ?", item.taskCode).
			Update("next_scheduled_at", gorm.Expr("CURRENT_TIMESTAMP - INTERVAL '1 minute'")).Error; err != nil {
			t.Fatal(err)
		}
		waitForOrganizationHRAcceptanceBatch(t, db, item.taskCode, 30*time.Second)
	}
	if err := syncRunner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	assertOrganizationHRAcceptanceState(t, db, checkpoint)
	callMu.Lock()
	gotOrder := append([]string(nil), callOrder...)
	callMu.Unlock()
	wantOrder := []string{"/legal-entities", "/legal-entities", "/legal-departments", "/management-companies", "/management-departments", "/positions", "/employees", "/resigned-employees"}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("HR initialization request order=%v want=%v", gotOrder, wantOrder)
	}
}

func organizationHRAcceptanceResponse(path, changed string, lower time.Time) []map[string]any {
	switch path {
	case "/legal-entities":
		return []map[string]any{{"zjkid_ignore": "legal-root", "id": "legal-record", "pk_corp": "LEGAL-ROOT", "name": "Legal Root", "shortname": "Legal", "fatherpkzjkid_ignore": "", "isenable": 1, "changeTime": changed}}
	case "/management-companies":
		return []map[string]any{{"zjkid_ignore": "management-root", "id": "management-record", "pk_corp": "MANAGEMENT-ROOT", "name": "Management Root", "fatherpkzjkid_ignore": "", "isenable": 1, "changeTime": changed, "level": 1}}
	case "/management-departments":
		return []map[string]any{
			{"zjkid_ignore": "management-child", "id": "management-child-record", "code": "MANAGEMENT-CHILD", "name": "Management Child", "pk_fathedeptzjkid_ignore": "management-parent", "isenable": 1, "changeTime": changed, "ilevel": 3, "disorder": "2"},
			{"zjkid_ignore": "management-parent", "id": "management-parent-record", "code": "MANAGEMENT-PARENT", "name": "Management Parent", "pk_fathedeptzjkid_ignore": "management-root", "isenable": 1, "changeTime": changed, "ilevel": 2, "disorder": "1"},
		}
	case "/legal-departments":
		return []map[string]any{
			{"zjkid_ignore": "legal-child", "id": "legal-child-record", "code": "LEGAL-CHILD", "name": "Legal Child", "pk_fathedeptzjkid_ignore": "legal-department-root", "orgidzjkid_ignore": "legal-root", "isenable": 1, "changeTime": changed, "ilevel": 2, "disorder": "2"},
			{"zjkid_ignore": "legal-department-root", "id": "legal-department-root-record", "code": "LEGAL-DEPARTMENT-ROOT", "name": "Legal Department Root", "pk_fathedeptzjkid_ignore": "", "orgidzjkid_ignore": "legal-root", "isenable": 1, "changeTime": changed, "ilevel": 1, "disorder": "1"},
		}
	case "/positions":
		return []map[string]any{{"postidzjkid_ignore": "position-root", "postCode": "POSITION-ROOT", "postname": "Position", "deptidzjkid_ignore": "management-child", "posLevel": "", "isenable": 1, "changeTime": changed}}
	case "/employees":
		return []map[string]any{{"psnidzjkid_ignore": "employee-root", "jhcode": "EMPLOYEE-ROOT", "name": "Acceptance Employee", "mobile": "", "email": "", "isenable": 1, "changeTime": changed, "sendpost": "[]"}}
	case "/resigned-employees":
		return []map[string]any{{"psnidzjkid_ignore": "employee-root", "changeTime": lower.UTC().Add(20 * time.Minute).Format("2006-01-02T15:04:05"), "lzdate": lower.UTC().Format("2006-01-02")}}
	default:
		return nil
	}
}

func newOrganizationHRAcceptanceTransport(t *testing.T, server *httptest.Server) integration.TransportClient {
	t.Helper()
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := integration.NewEndpointPolicy(false, nil, syncConsumerTestResolver{"hr-acceptance.test": {net.ParseIP("93.184.216.34")}})
	if err != nil {
		t.Fatal(err)
	}
	transport, err := integration.NewHTTPTransportClient(policy, integration.TransportClientOptions{
		RoundTripper: syncConsumerTestRoundTripper{target: target, base: server.Client().Transport},
	})
	if err != nil {
		t.Fatal(err)
	}
	return transport
}

func waitForOrganizationHRAcceptanceBatch(t *testing.T, db *gorm.DB, taskCode string, timeout time.Duration) {
	t.Helper()
	if testutil.Eventually(timeout, 100*time.Millisecond, func() bool {
		var batch model.IntegrationSyncBatch
		err := db.Where("task_code = ?", taskCode).Order("id DESC").First(&batch).Error
		if err == nil && batch.Status == model.IntegrationSyncBatchStatusSucceeded {
			return true
		}
		if err == nil && batch.Status == model.IntegrationSyncBatchStatusFailed {
			var execution model.IntegrationExecution
			var records []model.OrgSyncRecord
			_ = db.Where("sync_batch_id = ?", batch.Id).Order("id DESC").First(&execution).Error
			_ = db.Where("execution_id = ?", execution.Id).Order("id ASC").Find(&records).Error
			t.Fatalf("task %s failed: reason=%s summary=%s execution_status=%s execution_error=%s retry_reason=%s result=%s business_reason=%s records=%+v", taskCode, batch.ReasonCode, batch.ResultSummary, execution.Status, execution.ErrorCategory, execution.RetryReasonCode, execution.ResultSummary, execution.SyncBusinessReasonCode, records)
		}
		return false
	}) {
		return
	}
	t.Fatalf("task %s did not complete in %s", taskCode, timeout)
}

func seedOrganizationHRAcceptanceAssignment(t *testing.T, db *gorm.DB, checkpoint time.Time) {
	t.Helper()
	var legal model.OrgLegalEntity
	var unit model.OrgUnit
	var position model.OrgPosition
	var employee model.OrgEmployee
	for value, query := range map[any]string{
		&legal: "source_id = 'legal-root'", &unit: "source_id = 'management_unit:management-child'",
		&position: "source_id = 'position-root'", &employee: "source_id = 'employee-root'",
	} {
		if err := db.Where(query).First(value).Error; err != nil {
			t.Fatalf("load generated dependency %T: %v", value, err)
		}
	}
	validFrom := checkpoint.Add(-7 * 24 * time.Hour)
	assignment := model.OrgAssignment{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: "existing_business_fact",
		SourceId: "preexisting-assignment", EmployeeId: employee.Id, LegalEntityId: legal.Id, OrgUnitId: unit.Id,
		PositionId: &position.Id, AssignmentType: "secondary", ValidFrom: &validFrom, Status: "enabled",
		SourceDeleted: false, SyncStatus: "synced",
	}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
}

func assertOrganizationHRAcceptanceState(t *testing.T, db *gorm.DB, originalCheckpoint time.Time) {
	t.Helper()
	counts := []struct {
		model any
		want  int64
	}{
		{&model.OrgLegalEntity{}, 1}, {&model.OrgUnit{}, 5}, {&model.OrgStructure{}, 2},
		{&model.OrgStructureNode{}, 5}, {&model.OrgPosition{}, 1}, {&model.OrgEmployee{}, 1},
		{&model.OrgAssignment{}, 1}, {&model.OrgSyncBatch{}, 7}, {&model.IntegrationSyncBatch{}, 7},
		{&model.IntegrationExecution{}, 7}, {&model.IntegrationLog{}, 8},
	}
	for _, item := range counts {
		var got int64
		if err := db.Model(item.model).Count(&got).Error; err != nil || got != item.want {
			t.Fatalf("acceptance count %T=%d want=%d err=%v", item.model, got, item.want, err)
		}
	}
	var employee model.OrgEmployee
	if err := db.Where("source_id = ?", "employee-root").First(&employee).Error; err != nil || employee.EmploymentStatus != "resigned" || employee.ValidTo == nil || employee.UserId != nil || employee.SourceDeleted {
		t.Fatalf("accepted employee=%+v err=%v", employee, err)
	}
	var assignment model.OrgAssignment
	if err := db.Where("employee_id = ?", employee.Id).First(&assignment).Error; err != nil || assignment.Status != "disabled" || assignment.ValidTo == nil || assignment.SourceDeleted {
		t.Fatalf("accepted assignment=%+v err=%v", assignment, err)
	}
	var executions []model.IntegrationExecution
	if err := db.Order("id ASC").Find(&executions).Error; err != nil {
		t.Fatal(err)
	}
	for index, execution := range executions {
		wantAttempt := 1
		if index == 0 {
			wantAttempt = 2
		}
		if execution.Status != model.IntegrationExecutionStatusSucceeded || execution.SyncBusinessStatus != model.IntegrationSyncBusinessStatusSucceeded || execution.CurrentAttempt != wantAttempt {
			t.Fatalf("accepted execution[%d]=%+v", index, execution)
		}
	}
	var taskRows []model.IntegrationSyncTask
	if err := db.Order("id ASC").Find(&taskRows).Error; err != nil || len(taskRows) != 7 {
		t.Fatalf("accepted tasks=%d err=%v", len(taskRows), err)
	}
	for _, task := range taskRows {
		if task.CheckpointAt == nil || !task.CheckpointAt.After(originalCheckpoint) {
			t.Fatalf("checkpoint was not advanced for %s: %v", task.TaskCode, task.CheckpointAt)
		}
	}

	var management, legal model.OrgStructure
	if err := db.Where("code = ?", "hr_management").First(&management).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("code = ?", "hr_legal").First(&legal).Error; err != nil || management.Id == legal.Id {
		t.Fatalf("independent structures management=%+v legal=%+v err=%v", management, legal, err)
	}
	var managementChild, managementParent, managementRoot, legalChild, legalRoot model.OrgStructureNode
	findNode := func(target *model.OrgStructureNode, structureID int, sourceID string) {
		if err := db.Where("structure_id = ? AND source_id LIKE ?", structureID, "%"+sourceID).First(target).Error; err != nil {
			t.Fatalf("find structure node %s: %v", sourceID, err)
		}
	}
	findNode(&managementChild, management.Id, "management-child")
	findNode(&managementParent, management.Id, "management-parent")
	findNode(&managementRoot, management.Id, "management-root")
	findNode(&legalChild, legal.Id, "legal-child")
	findNode(&legalRoot, legal.Id, "legal-department-root")
	if managementChild.ParentNodeId == nil || *managementChild.ParentNodeId != managementParent.Id ||
		managementParent.ParentNodeId == nil || *managementParent.ParentNodeId != managementRoot.Id ||
		legalChild.ParentNodeId == nil || *legalChild.ParentNodeId != legalRoot.Id {
		t.Fatalf("structure relations management=%+v/%+v/%+v legal=%+v/%+v", managementChild, managementParent, managementRoot, legalChild, legalRoot)
	}
	for _, forbiddenColumn := range []string{"response_body", "source_dto", "payload"} {
		if db.Migrator().HasColumn(&model.OrgSyncRecord{}, forbiddenColumn) || db.Migrator().HasColumn(&model.OrgSyncBatch{}, forbiddenColumn) {
			t.Fatalf("forbidden response persistence column exists: %s", forbiddenColumn)
		}
	}
}
