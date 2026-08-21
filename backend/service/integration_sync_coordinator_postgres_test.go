package service

import (
	"backend/config"
	"backend/internal/audit"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	"backend/internal/integration"
	"backend/internal/organization/hrsync"
	"backend/internal/security"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type syncConsumerTestResolver map[string][]net.IP

func (r syncConsumerTestResolver) LookupIP(_ context.Context, host string) ([]net.IP, error) {
	values, ok := r[host]
	if !ok {
		return nil, fmt.Errorf("host not found")
	}
	return append([]net.IP(nil), values...), nil
}

type syncConsumerTestRoundTripper struct {
	target *url.URL
	base   http.RoundTripper
}

func (r syncConsumerTestRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	copyRequest := request.Clone(request.Context())
	copyURL := *request.URL
	copyURL.Scheme, copyURL.Host = r.target.Scheme, r.target.Host
	copyRequest.URL, copyRequest.Host = &copyURL, ""
	return r.base.RoundTrip(copyRequest)
}

func TestIntegrationSyncPostgreSQLSkipLockedCreatesOneBatch(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	first := seedPostgreSQLSyncCoordinator(t, db, "pg_schedule_once")
	var databaseNow time.Time
	if err := db.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&databaseNow).Error; err != nil {
		t.Fatal(err)
	}
	missedSchedule := databaseNow.UTC().Add(-24 * time.Hour)
	if err := db.Model(&model.IntegrationSyncTask{}).Where("task_code = ?", "pg_schedule_once").Update("next_scheduled_at", missedSchedule).Error; err != nil {
		t.Fatal(err)
	}
	second := newPostgreSQLSyncCoordinator(t, db, integration.SyncBusinessResultSucceeded)
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, coordinator := range []*IntegrationSyncCoordinator{first, second} {
		group.Add(1)
		go func(value *IntegrationSyncCoordinator) {
			defer group.Done()
			<-start
			_, err := value.ScheduleDueTasks(context.Background(), 1)
			results <- err
		}(coordinator)
	}
	close(start)
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("schedule due task: %v", err)
		}
	}
	var count int64
	if err := db.Model(&model.IntegrationSyncBatch{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("batch count=%d err=%v", count, err)
	}
	var task model.IntegrationSyncTask
	if err := db.Where("task_code = ?", "pg_schedule_once").First(&task).Error; err != nil || task.LastScheduledAt == nil || task.NextScheduledAt == nil || !task.LastScheduledAt.Equal(missedSchedule) || !task.NextScheduledAt.After(databaseNow) {
		t.Fatalf("scheduled task=%+v err=%v", task, err)
	}
}

func TestIntegrationSyncPostgreSQLTwoRunnersCreateOneBatch(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	firstCoordinator := seedPostgreSQLSyncCoordinator(t, db, "pg_two_runners")
	secondCoordinator := newPostgreSQLSyncCoordinator(t, db, integration.SyncBusinessResultSucceeded)
	first, err := integration.NewIntegrationSyncRunner(firstCoordinator, integration.SyncRunnerConfig{Enabled: true, RunnerID: "sync-runner-a", PollInterval: time.Second, ScheduleBatchSize: 2, CoordinateBatchSize: 2, ShutdownTimeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	second, err := integration.NewIntegrationSyncRunner(secondCoordinator, integration.SyncRunnerConfig{Enabled: true, RunnerID: "sync-runner-b", PollInterval: time.Second, ScheduleBatchSize: 2, CoordinateBatchSize: 2, ShutdownTimeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := first.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := second.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Stop(context.Background()); _ = second.Stop(context.Background()) })
	waitForPostgreSQLSyncExecution(t, db, 1, 5*time.Second)
	if err := first.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := second.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	var batches int64
	if err := db.Model(&model.IntegrationSyncBatch{}).Count(&batches).Error; err != nil || batches != 1 {
		t.Fatalf("batch count=%d err=%v", batches, err)
	}
	var executions int64
	if err := db.Model(&model.IntegrationExecution{}).Count(&executions).Error; err != nil || executions != 1 {
		t.Fatalf("execution count=%d err=%v", executions, err)
	}
}

func TestIntegrationSyncPostgreSQLManualAndScheduledCompeteForOneActiveBatch(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	coordinator := seedPostgreSQLSyncCoordinator(t, db, "pg_manual_schedule_race")
	primary := &database.PrimaryDB{DB: db}
	registry, err := integration.NewStaticSyncConsumerRegistry(integration.SyncConsumerRegistration{Metadata: integration.SyncConsumerMetadata{
		Code: "test_sync_consumer", Version: 1, Name: "Test", Status: integration.SyncConsumerStatusEnabled,
		ContentTypes: []string{"application/json"}, MaxResponseBytes: 2 << 20, MaxDuration: time.Second,
		CheckpointModes: []string{model.IntegrationSyncCheckpointTimestamp},
	}, Consumer: integration.SyncResultConsumerFunc(func(context.Context, integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
		return integration.NewSyncConsumptionResult(true, "", 1, 0, "")
	})})
	if err != nil {
		t.Fatal(err)
	}
	sf, _ := utils.NewSnowflake(5)
	taskService := NewSyncTaskService(impl.NewIntegrationSyncTaskRepositoryImpl(primary), impl.NewIntegrationSyncBatchRepositoryImpl(primary), impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), impl.NewRetryPolicyRepositoryImpl(primary), registry, sf, &externalSystemAuditWriter{}, &config.Server{})
	var task model.IntegrationSyncTask
	if err := db.Where("task_code = ?", "pg_manual_schedule_race").First(&task).Error; err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, runErr := taskService.RunSyncTask(audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(77, "sync-admin")), task.Id, task.Revision)
		results <- runErr
	}()
	go func() {
		<-start
		_, scheduleErr := coordinator.ScheduleDueTasks(context.Background(), 1)
		results <- scheduleErr
	}()
	close(start)
	firstErr, secondErr := <-results, <-results
	allowed := func(value error) bool {
		return value == nil || errors.Is(value, myerrors.ErrSyncTaskActiveBatch) || errors.Is(value, myerrors.ErrSyncBatchConflict) || errors.Is(value, myerrors.ErrSyncTaskRevisionConflict)
	}
	if !allowed(firstErr) || !allowed(secondErr) || (firstErr != nil && secondErr != nil) {
		t.Fatalf("manual/scheduled results=%v, %v", firstErr, secondErr)
	}
	var batches int64
	if err := db.Model(&model.IntegrationSyncBatch{}).Count(&batches).Error; err != nil || batches != 1 {
		t.Fatalf("active batch count=%d err=%v", batches, err)
	}
}

func TestIntegrationSyncPostgreSQLDueQueryUsesUTCInNonUTCSession(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	seedPostgreSQLSyncCoordinator(t, db, "pg_schedule_timezone")
	var databaseNow time.Time
	if err := db.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&databaseNow).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.IntegrationSyncTask{}).Where("task_code = ?", "pg_schedule_timezone").Update("next_scheduled_at", databaseNow.UTC().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	defer tx.Rollback()
	if err := tx.Exec("SET LOCAL TIME ZONE 'Asia/Shanghai'").Error; err != nil {
		t.Fatal(err)
	}
	repository := impl.NewIntegrationSyncBatchRepositoryImpl(&database.PrimaryDB{DB: db})
	values, err := repository.FindScheduledCandidates(tx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 0 {
		t.Fatalf("future UTC schedule was selected in non-UTC session: %+v", values)
	}
}

func TestIntegrationSyncPostgreSQLRunnerSequentialE2E(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	coordinator := seedPostgreSQLSyncCoordinator(t, db, "pg_runner_e2e")
	runner, err := integration.NewIntegrationSyncRunner(coordinator, integration.SyncRunnerConfig{
		Enabled: true, RunnerID: "pg-sync-runner", PollInterval: time.Second,
		ScheduleBatchSize: 4, CoordinateBatchSize: 4, ShutdownTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := runner.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runner.Stop(context.Background()) })

	first := waitForPostgreSQLSyncExecution(t, db, 1, 5*time.Second)
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	markSyncExecution(t, db, first.Id, model.IntegrationExecutionStatusSucceeded)
	runner, err = integration.NewIntegrationSyncRunner(coordinator, integration.SyncRunnerConfig{
		Enabled: true, RunnerID: "pg-sync-runner-restarted", PollInterval: time.Second,
		ScheduleBatchSize: 4, CoordinateBatchSize: 4, ShutdownTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Start(ctx); err != nil {
		t.Fatal(err)
	}
	second := waitForPostgreSQLSyncExecution(t, db, 2, 5*time.Second)
	if second.SyncWindowStart == nil || first.SyncWindowEnd == nil || !second.SyncWindowStart.Equal(*first.SyncWindowEnd) {
		t.Fatalf("non-contiguous slices first=%+v second=%+v", first, second)
	}
	markSyncExecution(t, db, second.Id, model.IntegrationExecutionStatusSucceeded)
	waitForPostgreSQLSyncBatchStatus(t, db, model.IntegrationSyncBatchStatusSucceeded, 5*time.Second)
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	var batch model.IntegrationSyncBatch
	if err := db.First(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if batch.ExecutionCount != 2 || batch.TechnicalSuccessCount != 2 || batch.CheckpointAfter == nil || !batch.CheckpointAfter.Equal(*batch.WindowEnd) {
		t.Fatalf("final batch=%+v", batch)
	}
	var task model.IntegrationSyncTask
	if err := db.First(&task, batch.SyncTaskID).Error; err != nil || task.CheckpointAt == nil || !task.CheckpointAt.Equal(*batch.WindowEnd) {
		t.Fatalf("final task=%+v err=%v", task, err)
	}
}

func TestIntegrationSyncPostgreSQLRunnerTransportRetryConsumerCheckpointE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "")
}

func TestIntegrationSyncPostgreSQLConsumerFailureStopsCheckpointE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, true, "")
}

func TestIntegrationSyncPostgreSQLLowerBoundOnlyOrganizationConsumerE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "success")
}

func TestIntegrationSyncPostgreSQLOrganizationDeferredReplayE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "deferred_replay")
}

func TestIntegrationSyncPostgreSQLOrganizationCycleStopsCheckpointE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "cycle")
}

func TestIntegrationSyncPostgreSQLPositionConsumerE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "position_success")
}

func TestIntegrationSyncPostgreSQLPositionDeferredReplayE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "position_deferred_replay")
}

func TestIntegrationSyncPostgreSQLEmployeeConsumerRetryPartitionAndCheckpointE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "employee_success")
}

func TestIntegrationSyncPostgreSQLEmployeeBusinessFailureDoesNotRetryE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "employee_conflict")
}

func TestIntegrationSyncPostgreSQLResignedEmployeeRetryAssignmentCheckpointE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "resigned_success")
}

func TestIntegrationSyncPostgreSQLResignedEmployeeDoesNotAllowImplicitRehireE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "employee_rehire_conflict")
}

func TestIntegrationSyncPostgreSQLResignedEmployeeMissingDependencyE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "resigned_missing")
}

func TestIntegrationSyncPostgreSQLResignedEmployeeAssignmentPeriodRollbackE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "resigned_period_conflict")
}

func TestIntegrationSyncPostgreSQLStaleResignationNoopE2E(t *testing.T) {
	runIntegrationSyncPostgreSQLTransportConsumerE2E(t, false, "resigned_stale")
}

func runIntegrationSyncPostgreSQLTransportConsumerE2E(t *testing.T, failSecondConsumer bool, organizationScenario string) {
	t.Helper()
	db := openSyncCoordinatorPostgreSQL(t)
	var httpCalls atomic.Int32
	var organizationRepaired atomic.Bool
	lowerBoundOnly := organizationScenario != ""
	positionScenario := strings.HasPrefix(organizationScenario, "position_")
	employeeScenario := strings.HasPrefix(organizationScenario, "employee_")
	resignedScenario := strings.HasPrefix(organizationScenario, "resigned_")
	resignedBusinessFailure := organizationScenario == "resigned_missing" || organizationScenario == "resigned_period_conflict"
	var resignationEventTime atomic.Value
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		call := httpCalls.Add(1)
		if lowerBoundOnly {
			expectedParameters := 1
			if employeeScenario {
				expectedParameters = 2
			}
			if request.URL.Path != "/employees" || request.URL.Query().Get("changed_since") == "" || request.URL.Query().Get("updated_from") != "" || request.URL.Query().Get("updated_to") != "" || len(request.URL.Query()) != expectedParameters ||
				(employeeScenario && request.URL.Query().Get("company_partition") != "approved-company-test") {
				t.Errorf("unexpected lower-bound request: %s", request.URL.String())
			}
		} else if request.URL.Path != "/employees" || request.URL.Query().Get("updated_from") == "" || request.URL.Query().Get("updated_to") == "" {
			t.Errorf("unexpected bounded sync request: %s", request.URL.String())
		}
		if request.Header.Get("Authorization") != "Bearer sync-consumer-token" {
			t.Errorf("credential was not injected")
		}
		writer.Header().Set("Content-Type", "application/json")
		if call == 1 && organizationScenario != "employee_rehire_conflict" && !resignedBusinessFailure {
			writer.WriteHeader(http.StatusServiceUnavailable)
		}
		if lowerBoundOnly {
			lower, parseErr := time.Parse(time.RFC3339Nano, request.URL.Query().Get("changed_since"))
			if parseErr != nil {
				t.Errorf("parse changed_since: %v", parseErr)
			}
			if organizationScenario == "deferred_replay" && !organizationRepaired.Load() {
				body, _ := json.Marshal(map[string]any{"success": true, "data": []map[string]any{
					{"zjkid_ignore": "deferred-stable", "code": "DEFERRED-STABLE", "name": "Stable", "pk_fathedeptzjkid_ignore": "", "isenable": 1, "changeTime": lower.Add(15 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
					{"zjkid_ignore": "deferred-child", "code": "DEFERRED-CHILD", "name": "Deferred", "pk_fathedeptzjkid_ignore": "deferred-parent", "isenable": 1, "changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
				}})
				_, _ = writer.Write(body)
				return
			}
			if resignedScenario {
				eventTime := lower.Add(20 * time.Minute).UTC()
				if existing := resignationEventTime.Load(); existing != nil {
					eventTime = existing.(time.Time)
				} else if call > 1 {
					resignationEventTime.Store(eventTime)
				}
				sourceID := "employee-resigned-e2e"
				if organizationScenario == "resigned_missing" {
					sourceID = "employee-resigned-missing"
				}
				body, _ := json.Marshal(map[string]any{"success": true, "data": []map[string]any{
					{"psnidzjkid_ignore": sourceID, "changeTime": eventTime.Format("2006-01-02T15:04:05"), "lzdate": checkpointSafeResignationDate(eventTime)},
					{"psnidzjkid_ignore": "employee-resigned-future", "changeTime": lower.Add(71 * time.Minute).UTC().Format("2006-01-02T15:04:05"), "lzdate": checkpointSafeResignationDate(eventTime)},
				}})
				_, _ = writer.Write(body)
				return
			}
			if positionScenario {
				if organizationScenario == "position_deferred_replay" && !organizationRepaired.Load() {
					body, _ := json.Marshal(map[string]any{"success": true, "data": []map[string]any{{
						"postidzjkid_ignore": "position-deferred", "postCode": "POST-DEFERRED", "postname": "待补组织岗位",
						"deptidzjkid_ignore": "position-dept-missing", "posLevel": "", "isenable": 1,
						"changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05"),
					}}})
					_, _ = writer.Write(body)
					return
				}
				data := []map[string]any{
					{"postidzjkid_ignore": "position-a", "postCode": "POST-A", "postname": "同名岗位", "deptidzjkid_ignore": "position-dept-a", "posLevel": "", "isenable": 1, "changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
					{"postidzjkid_ignore": "position-b", "postCode": "POST-B", "postname": "同名岗位", "deptidzjkid_ignore": "position-dept-b", "posLevel": "L2", "isenable": 1, "changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
					{"postidzjkid_ignore": "position-future", "postCode": "POST-FUTURE", "postname": "未来岗位", "deptidzjkid_ignore": "position-dept-a", "posLevel": "", "isenable": 1, "changeTime": lower.Add(71 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
				}
				if organizationScenario == "position_deferred_replay" {
					data = []map[string]any{{"postidzjkid_ignore": "position-deferred", "postCode": "POST-DEFERRED", "postname": "待补组织岗位", "deptidzjkid_ignore": "position-dept-missing", "posLevel": "", "isenable": 1, "changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05")}}
				} else if call > 2 {
					data[0]["isenable"] = 0
				}
				body, _ := json.Marshal(map[string]any{"success": true, "data": data})
				_, _ = writer.Write(body)
				return
			}
			if employeeScenario {
				status := 1
				name := "员工测试"
				email := any(nil)
				if call > 2 {
					status = 2
					name = "员工测试更新"
					email = "employee-e2e@example.invalid"
				}
				current := map[string]any{
					"psnidzjkid_ignore": "employee-e2e", "jhcode": "EMP-E2E-001", "name": name,
					"mobile": nil, "email": email, "isenable": status, "sendpost": "[]",
					"changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05"),
				}
				duplicate := current
				if organizationScenario == "employee_conflict" {
					duplicate = make(map[string]any, len(current))
					for key, value := range current {
						duplicate[key] = value
					}
					duplicate["name"] = "同版本冲突"
				}
				body, _ := json.Marshal(map[string]any{"success": true, "data": []map[string]any{
					current, duplicate,
					{"psnidzjkid_ignore": fmt.Sprintf("employee-future-%d", lower.Unix()), "jhcode": fmt.Sprintf("EMP-FUTURE-%d", lower.Unix()), "name": "未来员工", "isenable": 1, "sendpost": "[]", "changeTime": lower.Add(71 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
				}})
				_, _ = writer.Write(body)
				return
			}
			if organizationScenario == "cycle" {
				body, _ := json.Marshal(map[string]any{"success": true, "data": []map[string]any{
					{"zjkid_ignore": "cycle-a", "code": "CYCLE-A", "name": "A", "pk_fathedeptzjkid_ignore": "cycle-b", "isenable": 1, "changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
					{"zjkid_ignore": "cycle-b", "code": "CYCLE-B", "name": "B", "pk_fathedeptzjkid_ignore": "cycle-a", "isenable": 1, "changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
				}})
				_, _ = writer.Write(body)
				return
			}
			parentID := fmt.Sprintf("parent-%d", lower.Unix())
			data := []map[string]any{
				{"zjkid_ignore": fmt.Sprintf("child-%d", lower.Unix()), "code": fmt.Sprintf("CHILD-%d", lower.Unix()), "name": "Child", "pk_fathedeptzjkid_ignore": parentID, "isenable": 1, "changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
				{"zjkid_ignore": parentID, "code": fmt.Sprintf("PARENT-%d", lower.Unix()), "name": "Parent", "pk_fathedeptzjkid_ignore": "", "isenable": 1, "changeTime": lower.Add(15 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
				{"zjkid_ignore": "lookback-shared", "code": "LOOKBACK-SHARED", "name": "Replay", "pk_fathedeptzjkid_ignore": "", "isenable": 1, "changeTime": lower.UTC().Format("2006-01-02T15:04:05")},
				{"zjkid_ignore": fmt.Sprintf("future-%d", lower.Unix()), "code": fmt.Sprintf("FUTURE-%d", lower.Unix()), "name": "Future", "pk_fathedeptzjkid_ignore": "", "isenable": 1, "changeTime": lower.Add(71 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
			}
			if organizationScenario == "deferred_replay" {
				data = append(data,
					map[string]any{"zjkid_ignore": "deferred-stable", "code": "DEFERRED-STABLE", "name": "Stable", "pk_fathedeptzjkid_ignore": "", "isenable": 1, "changeTime": lower.Add(15 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
					map[string]any{"zjkid_ignore": "deferred-parent", "code": "DEFERRED-PARENT", "name": "Repaired Parent", "pk_fathedeptzjkid_ignore": "", "isenable": 1, "changeTime": lower.Add(15 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
					map[string]any{"zjkid_ignore": "deferred-child", "code": "DEFERRED-CHILD", "name": "Deferred", "pk_fathedeptzjkid_ignore": "deferred-parent", "isenable": 1, "changeTime": lower.Add(20 * time.Minute).UTC().Format("2006-01-02T15:04:05")},
				)
			}
			body, _ := json.Marshal(map[string]any{"success": true, "data": data})
			_, _ = writer.Write(body)
			return
		}
		_, _ = writer.Write([]byte(`{"employees":[{"id":"10001"}]}`))
	}))
	defer tlsServer.Close()
	target, err := url.Parse(tlsServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := integration.NewEndpointPolicy(false, nil, syncConsumerTestResolver{"sync-consumer.test": {net.ParseIP("93.184.216.34")}})
	if err != nil {
		t.Fatal(err)
	}
	transport, err := integration.NewHTTPTransportClient(policy, integration.TransportClientOptions{RoundTripper: syncConsumerTestRoundTripper{target: target, base: tlsServer.Client().Transport}})
	if err != nil {
		t.Fatal(err)
	}
	consumerCalled := make(chan integration.SyncConsumptionRequest, 2)
	primary := &database.PrimaryDB{DB: db}
	snowflake, _ := utils.NewSnowflake(4)
	consumerCode := "test_sync_consumer"
	consumer := integration.SyncResultConsumer(integration.SyncResultConsumerFunc(func(_ context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
		consumerCalled <- request
		if failSecondConsumer && request.SliceNo() == 2 {
			return integration.NewSyncConsumptionResult(false, "business_validation_failed", 0, 1, "")
		}
		return integration.NewSyncConsumptionResult(true, "", 1, 0, "ORG-SYNC-1")
	}))
	if lowerBoundOnly {
		contract, contractErr := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
		if contractErr != nil {
			t.Fatal(contractErr)
		}
		domain := NewOrganizationHRSyncService(impl.NewOrganizationHRSyncRepositoryImpl(primary), snowflake)
		formalConsumer := integration.SyncResultConsumer(hrsync.NewManagementDepartmentConsumer(domain, contract))
		consumerCode = hrsync.ConsumerCodeManagementDepartment
		if positionScenario {
			formalConsumer = hrsync.NewPositionConsumer(domain, contract)
			consumerCode = hrsync.ConsumerCodePosition
		} else if employeeScenario {
			formalConsumer = hrsync.NewEmployeeConsumer(domain, contract)
			consumerCode = hrsync.ConsumerCodeEmployee
		} else if resignedScenario {
			formalConsumer = hrsync.NewResignedEmployeeConsumer(domain, contract)
			consumerCode = hrsync.ConsumerCodeResignedEmployee
		}
		consumer = integration.SyncResultConsumerFunc(func(ctx context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
			consumerCalled <- request
			return formalConsumer.Consume(ctx, request)
		})
	}
	registry, err := integration.NewStaticSyncConsumerRegistry(integration.SyncConsumerRegistration{
		Metadata: integration.SyncConsumerMetadata{Code: consumerCode, Version: 1, Name: "Test Consumer", Status: integration.SyncConsumerStatusEnabled,
			ContentTypes: []string{"application/json"}, MaxResponseBytes: 1 << 20, MaxDuration: time.Second,
			CheckpointModes: []string{model.IntegrationSyncCheckpointTimestamp}},
		Consumer: consumer,
	})
	if err != nil {
		t.Fatal(err)
	}

	protector, err := security.NewCredentialSecretProtectorWithKey("sync-consumer-e2e-master-key")
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := json.Marshal(map[string]string{"token": "sync-consumer-token"})
	envelope, err := protector.Seal(secret)
	if err != nil {
		t.Fatal(err)
	}
	system := model.ExternalSystem{Basic: model.Basic{Id: nextSyncTestID(), State: true}, SystemCode: "sync_consumer_e2e", Name: "Sync Consumer E2E",
		SystemType: model.ExternalSystemTypeHR, BaseURL: "https://sync-consumer.test", OwnerIdentifier: "ops", OwnerName: "Ops",
		Status: model.ExternalSystemStatusEnabled, Revision: 1}
	credential := model.Credential{Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExternalSystemID: system.Id,
		CredentialCode: "sync_consumer_token", Name: "Sync Consumer Token", CredentialType: model.CredentialTypeBearerToken,
		Status: model.CredentialStatusActive, SecretStorageRef: envelope.StorageRef, SecretCiphertext: envelope.Ciphertext,
		SecretNonce: envelope.Nonce, SecretFingerprint: envelope.Fingerprint, Version: 1, Revision: 1}
	parameters := []integration.InputParameterDefinition{{Code: "updated_from", Location: "query", DataType: "string", Required: true, MaxLength: 64}, {Code: "updated_to", Location: "query", DataType: "string", Required: true, MaxLength: 64}}
	if lowerBoundOnly {
		parameters = []integration.InputParameterDefinition{{Code: "changed_since", Location: "query", DataType: "string", Required: true, MaxLength: 64}}
		if employeeScenario {
			parameters = append(parameters, integration.InputParameterDefinition{Code: "company_partition", Location: "query", DataType: "string", Required: true, MaxLength: 64})
		}
	}
	contract, _ := json.Marshal(integration.InterfaceInputContract{Version: 1, Parameters: parameters})
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExternalSystemID: system.Id,
		InterfaceCode: "employees", Name: "Employees", Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET,
		RelativePath: "/employees", CredentialID: &credential.Id, TimeoutSeconds: 5, ResponseLimit: 1 << 20, InputContract: contract,
		IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	retryPolicy := model.RetryPolicy{Basic: model.Basic{Id: nextSyncTestID(), State: true}, PolicyCode: "sync_consumer_retry", PolicyName: "Sync Consumer Retry", Version: 1, Status: model.RetryPolicyStatusEnabled,
		MaxAttempts: 2, InitialDelayMs: 1000, MaxDelayMs: 1000, BackoffType: model.RetryBackoffTypeFixed, BackoffMultiplier: 1,
		JitterType: model.RetryJitterTypeNone, JitterRatio: 0, RetryWindowMs: 60000,
		RetryableErrorCategories: datatypes.JSON([]byte(`["network","remote","timeout"]`)), RetryableHTTPStatuses: datatypes.JSON([]byte(`[429,502,503,504]`)), RespectRetryAfter: true, Revision: 1}
	if _, err := integration.BuildRetryPolicySnapshot(retryPolicy, integration.RetryPolicySnapshotOptions{IdempotencyMode: integration.RemoteIdempotencySafeMethod}); err != nil {
		t.Fatalf("retry fixture is invalid: %v", err)
	}
	definition.RetryPolicyID = &retryPolicy.Id
	var databaseNow time.Time
	if err := db.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&databaseNow).Error; err != nil {
		t.Fatal(err)
	}
	checkpoint, due := databaseNow.UTC().Add(-90*time.Minute), databaseNow.UTC().Add(-time.Minute)
	inputPlan := integration.SyncExecutionInputPlan{Version: 1, StaticInput: integration.ExecutionInputValues{}, WindowStartBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_from", Format: "rfc3339"}, WindowEndBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}}
	lookback := 0
	if lowerBoundOnly {
		staticInput := integration.ExecutionInputValues{}
		if employeeScenario {
			staticInput.QueryParams = map[string][]string{"company_partition": {"approved-company-test"}}
		}
		inputPlan = integration.SyncExecutionInputPlan{Version: integration.SyncExecutionInputPlanVersionV2, WindowMode: integration.SyncWindowModeLowerBoundOnly, StaticInput: staticInput, WindowStartBinding: &integration.SyncWindowBinding{Location: "query", Code: "changed_since", Format: "rfc3339"}}
		lookback = 600
	}
	plan, _ := json.Marshal(inputPlan)
	task := model.IntegrationSyncTask{Basic: model.Basic{Id: nextSyncTestID(), State: true}, TaskCode: "sync_consumer_e2e", TaskName: "Sync Consumer E2E",
		Version: 1, Status: model.IntegrationSyncTaskStatusEnabled, ExternalSystemID: system.Id, InterfaceDefinitionID: definition.Id,
		ConsumerCode: consumerCode, ConsumerVersion: 1, ScheduleType: model.IntegrationSyncScheduleCron, CronExpression: "* * * * *",
		Timezone: "UTC", NextScheduledAt: &due, CheckpointMode: model.IntegrationSyncCheckpointTimestamp,
		InitialCheckpointAt: &checkpoint, CheckpointAt: &checkpoint, LookbackSeconds: lookback, WindowSliceSeconds: 3600, InputPlan: datatypes.JSON(plan), Revision: 1}
	for _, value := range []any{&system, &credential, &retryPolicy, &definition, &task} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	if positionScenario {
		for _, unit := range []model.OrgUnit{
			organizationHRPositionUnit(nextSyncTestID(), "position-dept-a", "POSITION-DEPT-A"),
			organizationHRPositionUnit(nextSyncTestID(), "position-dept-b", "POSITION-DEPT-B"),
		} {
			if err := db.Create(&unit).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	if resignedScenario || organizationScenario == "employee_rehire_conflict" {
		seedPostgreSQLResignationFixture(t, db, organizationScenario)
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

	executions := impl.NewIntegrationExecutionRepositoryImpl(primary)
	systems := impl.NewExternalSystemRepositoryImpl(primary)
	interfaces := impl.NewInterfaceDefinitionRepositoryImpl(primary)
	credentials := impl.NewCredentialRepositoryImpl(primary)
	batches := impl.NewIntegrationSyncBatchRepositoryImpl(primary)
	tasks := impl.NewIntegrationSyncTaskRepositoryImpl(primary)
	executionService := NewIntegrationExecutionService(executions, impl.NewIntegrationLogRepositoryImpl(primary), systems, interfaces,
		impl.NewRetryPolicyRepositoryImpl(primary), snowflake, &integrationExecutionAuditWriter{})
	coordinator := NewIntegrationSyncCoordinator(tasks, batches, executions, systems, interfaces, executionService,
		integration.NewPersistedSyncBusinessResultProvider(), snowflake)
	syncRunner, err := integration.NewIntegrationSyncRunner(coordinator, integration.SyncRunnerConfig{Enabled: true, RunnerID: "sync-consumer-runner",
		PollInterval: time.Second, ScheduleBatchSize: 2, CoordinateBatchSize: 2, ShutdownTimeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	provider := integration.NewCredentialProvider(credentials, interfaces, protector)
	guard, _ := integration.NewInMemoryConcurrencyGuard(2, 2, 2)
	engine, err := integration.NewIntegrationExecutionEngine(executions, systems, interfaces, credentials, batches, provider, transport, guard, registry,
		snowflake, integration.ExecutionEngineOptions{WorkerID: "sync-consumer-worker", LeaseDuration: integration.IntegrationDefaultLeaseDuration, BatchSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := integration.NewIntegrationWorkerRunner(engine, integration.WorkerRunnerConfig{Enabled: true, WorkerID: "sync-consumer-worker",
		PollInterval: time.Second, ClaimBatchSize: 2, InstanceConcurrency: 2, LeaseRecoveryInterval: 10 * time.Second,
		ShutdownTimeout: 3 * time.Second, LeaseDuration: integration.IntegrationDefaultLeaseDuration})
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
	expectedBatchStatus := model.IntegrationSyncBatchStatusSucceeded
	if failSecondConsumer || organizationScenario == "deferred_replay" || organizationScenario == "cycle" || organizationScenario == "position_deferred_replay" || organizationScenario == "employee_conflict" || organizationScenario == "employee_rehire_conflict" || resignedBusinessFailure {
		expectedBatchStatus = model.IntegrationSyncBatchStatusFailed
	}
	waitForPostgreSQLSyncBatchStatus(t, db, expectedBatchStatus, 20*time.Second)
	if organizationScenario == "cycle" {
		assertOrganizationFailedCheckpoint(t, db, task.Id, checkpoint, hrsync.ReasonParentCycle)
		if err := syncRunner.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := worker.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		return
	}
	if organizationScenario == "deferred_replay" {
		assertOrganizationFailedCheckpoint(t, db, task.Id, checkpoint, hrsync.ReasonParentUnresolved)
		var stableUnits int64
		if err := db.Model(&model.OrgUnit{}).Where("source_id = ?", "management_unit:deferred-stable").Count(&stableUnits).Error; err != nil || stableUnits != 1 {
			t.Fatalf("successful object before replay count=%d err=%v", stableUnits, err)
		}
		organizationRepaired.Store(true)
		replayScheduledFor := checkpoint.Add(30 * time.Minute)
		if err := db.Model(&model.IntegrationSyncTask{}).Where("id = ?", task.Id).Update("next_scheduled_at", replayScheduledFor).Error; err != nil {
			t.Fatal(err)
		}
		waitForPostgreSQLSyncBatchOrdinalStatus(t, db, 2, model.IntegrationSyncBatchStatusSucceeded, 40*time.Second)
		if err := syncRunner.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := worker.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		var refreshed model.IntegrationSyncTask
		if err := db.First(&refreshed, task.Id).Error; err != nil || refreshed.CheckpointAt == nil || !refreshed.CheckpointAt.After(checkpoint) {
			t.Fatalf("repaired checkpoint=%v err=%v", refreshed.CheckpointAt, err)
		}
		var units, deferredChildren int64
		if err := db.Model(&model.OrgUnit{}).Count(&units).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgUnit{}).Where("source_id = ?", "management_unit:deferred-child").Count(&deferredChildren).Error; err != nil || deferredChildren != 1 {
			t.Fatalf("replayed child count=%d err=%v", deferredChildren, err)
		}
		if err := db.Model(&model.OrgUnit{}).Where("source_id = ?", "management_unit:deferred-stable").Count(&stableUnits).Error; err != nil || stableUnits != 1 {
			t.Fatalf("successful replay idempotency count=%d err=%v", stableUnits, err)
		}
		if units < 2 {
			t.Fatalf("repaired units=%d", units)
		}
		return
	}
	if organizationScenario == "position_deferred_replay" {
		assertOrganizationFailedCheckpoint(t, db, task.Id, checkpoint, hrsync.ReasonReferenceMissing)
		var failedExecution model.IntegrationExecution
		if err := db.Order("id ASC").First(&failedExecution).Error; err != nil || failedExecution.Status != model.IntegrationExecutionStatusFailed || failedExecution.CurrentAttempt != 2 {
			t.Fatalf("position business failure retried unexpectedly: execution=%+v err=%v", failedExecution, err)
		}
		var failedAttempts int64
		if err := db.Model(&model.IntegrationLog{}).Where("execution_id = ?", failedExecution.Id).Count(&failedAttempts).Error; err != nil || failedAttempts != 2 {
			t.Fatalf("position business failure attempts=%d err=%v", failedAttempts, err)
		}
		organizationRepaired.Store(true)
		unit := organizationHRPositionUnit(nextSyncTestID(), "position-dept-missing", "POSITION-DEPT-MISSING")
		if err := db.Create(&unit).Error; err != nil {
			t.Fatal(err)
		}
		replayScheduledFor := checkpoint.Add(30 * time.Minute)
		if err := db.Model(&model.IntegrationSyncTask{}).Where("id = ?", task.Id).Update("next_scheduled_at", replayScheduledFor).Error; err != nil {
			t.Fatal(err)
		}
		waitForPostgreSQLSyncBatchOrdinalStatus(t, db, 2, model.IntegrationSyncBatchStatusSucceeded, 40*time.Second)
		if err := syncRunner.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := worker.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		var refreshed model.IntegrationSyncTask
		if err := db.First(&refreshed, task.Id).Error; err != nil || refreshed.CheckpointAt == nil || !refreshed.CheckpointAt.After(checkpoint) {
			t.Fatalf("repaired position checkpoint=%v err=%v", refreshed.CheckpointAt, err)
		}
		var positions int64
		if err := db.Model(&model.OrgPosition{}).Where("source_id = ?", "position-deferred").Count(&positions).Error; err != nil || positions != 1 {
			t.Fatalf("replayed position count=%d err=%v", positions, err)
		}
		return
	}
	if organizationScenario == "employee_conflict" || organizationScenario == "employee_rehire_conflict" {
		reason := hrsync.ReasonSourceIDConflict
		if organizationScenario == "employee_rehire_conflict" {
			reason = hrsync.ReasonEmploymentStateConflict
		}
		assertOrganizationFailedCheckpoint(t, db, task.Id, checkpoint, reason)
		if err := syncRunner.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := worker.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		var failedExecution model.IntegrationExecution
		expectedAttempts := 2
		if organizationScenario == "employee_rehire_conflict" {
			expectedAttempts = 1
		}
		if err := db.Order("id ASC").First(&failedExecution).Error; err != nil || failedExecution.Status != model.IntegrationExecutionStatusFailed || failedExecution.CurrentAttempt != expectedAttempts {
			t.Fatalf("employee business failure execution=%+v err=%v", failedExecution, err)
		}
		var attempts, employees int64
		if err := db.Model(&model.IntegrationLog{}).Where("execution_id = ?", failedExecution.Id).Count(&attempts).Error; err != nil || attempts != int64(expectedAttempts) {
			t.Fatalf("employee business failure attempts=%d err=%v", attempts, err)
		}
		if err := db.Model(&model.OrgEmployee{}).Count(&employees).Error; err != nil || employees != 0 || httpCalls.Load() != int32(expectedAttempts) {
			if organizationScenario != "employee_rehire_conflict" || err != nil || employees != 1 || httpCalls.Load() != int32(expectedAttempts) {
				t.Fatalf("employee business failure employees=%d http=%d err=%v", employees, httpCalls.Load(), err)
			}
		}
		return
	}
	if resignedBusinessFailure {
		reason := hrsync.ReasonReferenceMissing
		if organizationScenario == "resigned_period_conflict" {
			reason = hrsync.ReasonAssignmentPeriodInvalid
		}
		assertOrganizationFailedCheckpoint(t, db, task.Id, checkpoint, reason)
		if err := syncRunner.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := worker.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		var failedExecution model.IntegrationExecution
		if err := db.Order("id ASC").First(&failedExecution).Error; err != nil || failedExecution.Status != model.IntegrationExecutionStatusFailed || failedExecution.CurrentAttempt != 1 || httpCalls.Load() != 1 {
			t.Fatalf("resignation business failure execution=%+v http=%d err=%v", failedExecution, httpCalls.Load(), err)
		}
		if organizationScenario == "resigned_period_conflict" {
			assertPostgreSQLResignationFixtureUnchanged(t, db)
		}
		return
	}
	if err := syncRunner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	for sliceNo := 1; sliceNo <= 2; sliceNo++ {
		select {
		case request := <-consumerCalled:
			if request.ExecutionNo() == "" || request.SyncBatchNo() == "" || request.TaskCode() != task.TaskCode || request.TaskVersion() != task.Version ||
				request.SliceNo() != sliceNo || (!lowerBoundOnly && !strings.Contains(string(request.Body()), "10001")) {
				t.Fatalf("consumer request=%s", request.String())
			}
		default:
			t.Fatalf("consumer was not called for slice %d", sliceNo)
		}
	}
	var executionRows []model.IntegrationExecution
	if err := db.Where("sync_batch_id IS NOT NULL").Order("sync_slice_no ASC").Find(&executionRows).Error; err != nil {
		t.Fatal(err)
	}
	if len(executionRows) != 2 || executionRows[0].Status != model.IntegrationExecutionStatusSucceeded || executionRows[0].CurrentAttempt != 2 || executionRows[0].SyncBusinessStatus != model.IntegrationSyncBusinessStatusSucceeded || httpCalls.Load() != 3 {
		t.Fatalf("executions=%+v http_calls=%d", executionRows, httpCalls.Load())
	}
	var refreshedTask model.IntegrationSyncTask
	if err := db.First(&refreshedTask, task.Id).Error; err != nil || refreshedTask.CheckpointAt == nil {
		t.Fatalf("checkpoint=%+v err=%v", refreshedTask.CheckpointAt, err)
	}
	if failSecondConsumer {
		if executionRows[1].Status != model.IntegrationExecutionStatusFailed || executionRows[1].SyncBusinessStatus != model.IntegrationSyncBusinessStatusFailed || executionRows[1].CurrentAttempt != 1 || executionRows[0].SyncWindowEnd == nil || !refreshedTask.CheckpointAt.Equal(*executionRows[0].SyncWindowEnd) {
			t.Fatalf("consumer failure boundary executions=%+v checkpoint=%v", executionRows, refreshedTask.CheckpointAt)
		}
	} else if executionRows[1].Status != model.IntegrationExecutionStatusSucceeded || executionRows[1].SyncBusinessStatus != model.IntegrationSyncBusinessStatusSucceeded || executionRows[1].CurrentAttempt != 1 || executionRows[1].SyncWindowEnd == nil || !refreshedTask.CheckpointAt.Equal(*executionRows[1].SyncWindowEnd) {
		t.Fatalf("successful checkpoint boundary executions=%+v checkpoint=%v", executionRows, refreshedTask.CheckpointAt)
	}
	if positionScenario {
		var positions []model.OrgPosition
		if err := db.Order("source_id ASC").Find(&positions).Error; err != nil || len(positions) != 2 {
			t.Fatalf("position e2e rows=%+v err=%v", positions, err)
		}
		if positions[0].Name != positions[1].Name || positions[0].OrgUnitId == positions[1].OrgUnitId || positions[0].Status != "disabled" || positions[0].SourceDeleted || positions[0].IsManagerPosition {
			t.Fatalf("position e2e semantics=%+v", positions)
		}
		var futureCount, recordCount, batchCount int64
		if err := db.Model(&model.OrgPosition{}).Where("source_id = ?", "position-future").Count(&futureCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgSyncRecord{}).Count(&recordCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgSyncBatch{}).Count(&batchCount).Error; err != nil {
			t.Fatal(err)
		}
		if futureCount != 0 || recordCount != 4 || batchCount != 2 {
			t.Fatalf("position e2e filter: future=%d records=%d batches=%d", futureCount, recordCount, batchCount)
		}
	} else if employeeScenario {
		var employees []model.OrgEmployee
		if err := db.Find(&employees).Error; err != nil || len(employees) != 1 {
			t.Fatalf("employee e2e rows=%+v err=%v", employees, err)
		}
		employee := employees[0]
		if employee.EmployeeNo != "EMP-E2E-001" || employee.Name != "员工测试更新" || employee.Mobile != "" || employee.Email != "employee-e2e@example.invalid" || employee.EmploymentStatus != "suspended" || employee.UserId != nil || employee.PrimaryLegalEntityId != nil || employee.SourceDeleted {
			t.Fatalf("employee e2e semantics=%+v", employee)
		}
		var futureCount, recordCount, batchCount, assignmentCount int64
		if err := db.Model(&model.OrgEmployee{}).Where("source_id LIKE ?", "employee-future-%").Count(&futureCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgSyncRecord{}).Count(&recordCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgSyncBatch{}).Count(&batchCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgAssignment{}).Count(&assignmentCount).Error; err != nil {
			t.Fatal(err)
		}
		if futureCount != 0 || recordCount != 2 || batchCount != 2 || assignmentCount != 0 {
			t.Fatalf("employee e2e filter: future=%d records=%d batches=%d assignments=%d", futureCount, recordCount, batchCount, assignmentCount)
		}
	} else if resignedScenario {
		var employee model.OrgEmployee
		if err := db.Where("source_id = ?", "employee-resigned-e2e").First(&employee).Error; err != nil || employee.UserId != nil || employee.SourceDeleted {
			t.Fatalf("resigned employee e2e=%+v err=%v", employee, err)
		}
		var assignment model.OrgAssignment
		if organizationScenario != "resigned_stale" {
			if employee.EmploymentStatus != "resigned" || employee.ValidTo == nil {
				t.Fatalf("resigned employee state=%+v", employee)
			}
			if err := db.Where("employee_id = ?", employee.Id).First(&assignment).Error; err != nil || assignment.Status != "disabled" || assignment.ValidTo == nil || assignment.SourceDeleted {
				t.Fatalf("resigned assignment e2e=%+v err=%v", assignment, err)
			}
		} else if employee.EmploymentStatus != "active" || employee.ValidTo != nil {
			t.Fatalf("stale resignation changed employee=%+v", employee)
		}
		var futureCount, recordCount, batchCount int64
		if err := db.Model(&model.OrgEmployee{}).Where("source_id = ?", "employee-resigned-future").Count(&futureCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgSyncRecord{}).Count(&recordCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgSyncBatch{}).Count(&batchCount).Error; err != nil {
			t.Fatal(err)
		}
		expectedRecords := int64(4)
		if organizationScenario == "resigned_stale" {
			expectedRecords = 2
		}
		if futureCount != 0 || recordCount != expectedRecords || batchCount != 2 {
			t.Fatalf("resignation e2e filter: future=%d records=%d batches=%d", futureCount, recordCount, batchCount)
		}
	} else if lowerBoundOnly {
		var futureCount, recordCount, batchCount, unitCount int64
		if err := db.Model(&model.OrgUnit{}).Where("code LIKE ?", "FUTURE-%").Count(&futureCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgSyncRecord{}).Count(&recordCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgSyncBatch{}).Count(&batchCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&model.OrgUnit{}).Count(&unitCount).Error; err != nil {
			t.Fatal(err)
		}
		if futureCount != 0 || recordCount != 6 || batchCount != 2 || unitCount != 5 {
			t.Fatalf("organization filter result: future=%d records=%d batches=%d units=%d", futureCount, recordCount, batchCount, unitCount)
		}
		for _, forbiddenColumn := range []string{"response_body", "source_dto", "payload"} {
			if db.Migrator().HasColumn(&model.OrgSyncRecord{}, forbiddenColumn) || db.Migrator().HasColumn(&model.OrgSyncBatch{}, forbiddenColumn) {
				t.Fatalf("Organization sync persisted forbidden body column: %s", forbiddenColumn)
			}
		}
	}
}

func checkpointSafeResignationDate(lower time.Time) string {
	return lower.UTC().Add(-24 * time.Hour).Format("2006-01-02")
}

func seedPostgreSQLResignationFixture(t *testing.T, db *gorm.DB, scenario string) {
	t.Helper()
	var task model.IntegrationSyncTask
	if err := db.Order("id DESC").First(&task).Error; err != nil || task.CheckpointAt == nil {
		t.Fatalf("resignation task checkpoint=%v err=%v", task.CheckpointAt, err)
	}
	if scenario == "resigned_missing" {
		return
	}
	withAssignment := scenario == "resigned_success" || scenario == "resigned_period_conflict"
	rawSourceID := "employee-e2e"
	status := "resigned"
	validTo := task.CheckpointAt.Add(-24 * time.Hour)
	if strings.HasPrefix(scenario, "resigned_") {
		rawSourceID = "employee-resigned-e2e"
		status = "active"
	}
	employee := model.OrgEmployee{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: rawSourceID, EmployeeNo: "EMP-RESIGNED-E2E", Name: "Existing Employee", EmploymentStatus: status,
		SourceVersion: task.CheckpointAt.UTC().Format(time.RFC3339Nano), SourceUpdatedAt: task.CheckpointAt,
		LastSyncAt: task.CheckpointAt, SourceDeleted: false, SyncStatus: "synced",
	}
	if scenario == "employee_rehire_conflict" {
		employee.ValidTo = &validTo
	}
	if scenario == "resigned_stale" {
		newer := task.CheckpointAt.Add(4 * time.Hour)
		employee.SourceVersion = newer.UTC().Format(time.RFC3339Nano)
		employee.SourceUpdatedAt = &newer
	}
	if err := db.Create(&employee).Error; err != nil {
		t.Fatal(err)
	}
	if !withAssignment {
		return
	}
	legal := model.OrgLegalEntity{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: "legal-resigned-e2e", Code: "LEGAL-RESIGNED-E2E", Name: "Legal", EntityType: "legal_company",
		Status: "enabled", SourceDeleted: false, SyncStatus: "synced",
	}
	unit := model.OrgUnit{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: "management_unit:resigned-e2e", Code: "UNIT-RESIGNED-E2E", Name: "Unit", UnitType: "department",
		Status: "enabled", SourceDeleted: false, SyncStatus: "synced",
	}
	if err := db.Create(&legal).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&unit).Error; err != nil {
		t.Fatal(err)
	}
	validFrom := task.CheckpointAt.Add(-30 * 24 * time.Hour)
	if scenario == "resigned_period_conflict" {
		validFrom = task.CheckpointAt.Add(24 * time.Hour)
	}
	assignment := model.OrgAssignment{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: "assignment-resigned-e2e", EmployeeId: employee.Id, LegalEntityId: legal.Id, OrgUnitId: unit.Id,
		AssignmentType: "secondary", ValidFrom: &validFrom, Status: "enabled", SourceDeleted: false, SyncStatus: "synced",
	}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
}

func assertPostgreSQLResignationFixtureUnchanged(t *testing.T, db *gorm.DB) {
	t.Helper()
	var employee model.OrgEmployee
	if err := db.Where("source_id = ?", "employee-resigned-e2e").First(&employee).Error; err != nil || employee.EmploymentStatus != "active" || employee.ValidTo != nil {
		t.Fatalf("employee changed despite assignment failure=%+v err=%v", employee, err)
	}
	var assignment model.OrgAssignment
	if err := db.Where("employee_id = ?", employee.Id).First(&assignment).Error; err != nil || assignment.Status != "enabled" || assignment.ValidTo != nil {
		t.Fatalf("assignment changed despite period failure=%+v err=%v", assignment, err)
	}
}

func openSyncCoordinatorPostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := openPostgresTestDB(t, postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schemaName := fmt.Sprintf("integration_sync_coordinator_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf("CREATE SCHEMA %q", schemaName)).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schemaName)).Error })
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	query.Set("TimeZone", "UTC")
	parsed.RawQuery = query.Encode()
	db, err := openPostgresTestDB(t, postgres.Open(parsed.String()), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ExternalSystem{}, &model.Credential{}, &model.RetryPolicy{}, &model.InterfaceDefinition{}, &model.IntegrationSyncTask{}, &model.IntegrationSyncBatch{}, &model.IntegrationExecution{}, &model.IntegrationLog{}, &model.OrgLegalEntity{}, &model.OrgUnit{}, &model.OrgStructure{}, &model.OrgStructureNode{}, &model.OrgPosition{}, &model.OrgEmployee{}, &model.OrgAssignment{}, &model.OrgSyncBatch{}, &model.OrgSyncRecord{}); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE UNIQUE INDEX uni_integration_sync_batch_active ON integration_sync_batch (task_code) WHERE status IN ('created','running') AND gmt_delete IS NULL`,
		`CREATE UNIQUE INDEX uni_integration_sync_batch_scheduled ON integration_sync_batch (task_code, scheduled_for) WHERE trigger_type = 'scheduled' AND gmt_delete IS NULL`,
		`CREATE UNIQUE INDEX uni_integration_execution_sync_slice ON integration_execution (sync_batch_id, sync_slice_no) WHERE sync_batch_id IS NOT NULL AND gmt_delete IS NULL`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func seedPostgreSQLSyncCoordinator(t *testing.T, db *gorm.DB, taskCode string) *IntegrationSyncCoordinator {
	t.Helper()
	system := model.ExternalSystem{Basic: model.Basic{Id: nextSyncTestID(), State: true}, SystemCode: taskCode, Name: "PG Sync", SystemType: model.ExternalSystemTypeHR, BaseURL: "https://example.com", OwnerIdentifier: "ops", OwnerName: "Ops", Status: model.ExternalSystemStatusEnabled, Revision: 1}
	contract, _ := json.Marshal(integration.InterfaceInputContract{Version: 1, Parameters: []integration.InputParameterDefinition{
		{Code: "updated_from", Location: "query", DataType: "string", Required: true, MaxLength: 64},
		{Code: "updated_to", Location: "query", DataType: "string", Required: true, MaxLength: 64},
	}})
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExternalSystemID: system.Id, InterfaceCode: taskCode, Name: "PG Sync", Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET, RelativePath: "/employees", TimeoutSeconds: 30, ResponseLimit: 1024 * 1024, InputContract: contract, IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	var databaseNow time.Time
	if err := db.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&databaseNow).Error; err != nil {
		t.Fatal(err)
	}
	databaseNow = databaseNow.UTC()
	checkpoint := databaseNow.Add(-2 * time.Hour)
	due := databaseNow.Add(-time.Minute)
	planRaw, _ := json.Marshal(integration.SyncExecutionInputPlan{Version: 1, StaticInput: integration.ExecutionInputValues{}, WindowStartBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_from", Format: "rfc3339"}, WindowEndBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}})
	task := model.IntegrationSyncTask{Basic: model.Basic{Id: nextSyncTestID(), State: true}, TaskCode: taskCode, TaskName: "PG Sync", Version: 1, Status: model.IntegrationSyncTaskStatusEnabled, ExternalSystemID: system.Id, InterfaceDefinitionID: definition.Id, ConsumerCode: "test_sync_consumer", ConsumerVersion: 1, ScheduleType: model.IntegrationSyncScheduleCron, CronExpression: "* * * * *", Timezone: "UTC", NextScheduledAt: &due, CheckpointMode: model.IntegrationSyncCheckpointTimestamp, InitialCheckpointAt: &checkpoint, CheckpointAt: &checkpoint, LookbackSeconds: 60, WindowSliceSeconds: 3600, InputPlan: datatypes.JSON(planRaw), Revision: 1}
	if err := db.Create(&system).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&definition).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	return newPostgreSQLSyncCoordinator(t, db, integration.SyncBusinessResultSucceeded)
}

func newPostgreSQLSyncCoordinator(t *testing.T, db *gorm.DB, businessStatus string) *IntegrationSyncCoordinator {
	t.Helper()
	primary := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(3)
	if err != nil {
		t.Fatal(err)
	}
	executionRepo := impl.NewIntegrationExecutionRepositoryImpl(primary)
	application := NewIntegrationExecutionService(executionRepo, impl.NewIntegrationLogRepositoryImpl(primary), impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), impl.NewRetryPolicyRepositoryImpl(primary), sf, &integrationExecutionAuditWriter{})
	return NewIntegrationSyncCoordinator(impl.NewIntegrationSyncTaskRepositoryImpl(primary), impl.NewIntegrationSyncBatchRepositoryImpl(primary), executionRepo, impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), application, &syncBusinessResultStub{status: businessStatus}, sf)
}

func waitForPostgreSQLSyncExecution(t *testing.T, db *gorm.DB, sliceNo int, timeout time.Duration) model.IntegrationExecution {
	t.Helper()
	var value model.IntegrationExecution
	if testutil.Eventually(timeout, 25*time.Millisecond, func() bool {
		return db.Where("sync_slice_no = ?", sliceNo).First(&value).Error == nil
	}) {
		return value
	}
	t.Fatalf("slice %d was not created", sliceNo)
	return model.IntegrationExecution{}
}

func waitForPostgreSQLSyncBatchStatus(t *testing.T, db *gorm.DB, status string, timeout time.Duration) {
	t.Helper()
	if testutil.Eventually(timeout, 25*time.Millisecond, func() bool {
		var value model.IntegrationSyncBatch
		return db.First(&value).Error == nil && value.Status == status
	}) {
		return
	}
	var batch model.IntegrationSyncBatch
	var executions []model.IntegrationExecution
	var attempts []model.IntegrationLog
	_ = db.First(&batch).Error
	_ = db.Order("sync_slice_no ASC").Find(&executions).Error
	_ = db.Order("execution_id ASC, attempt_no ASC").Find(&attempts).Error
	t.Fatalf("batch did not reach %s: batch=%+v executions=%+v attempts=%+v", status, batch, executions, attempts)
}

func waitForPostgreSQLSyncBatchOrdinalStatus(t *testing.T, db *gorm.DB, ordinal int, status string, timeout time.Duration) {
	t.Helper()
	if testutil.Eventually(timeout, 25*time.Millisecond, func() bool {
		var values []model.IntegrationSyncBatch
		return db.Order("id ASC").Find(&values).Error == nil && len(values) >= ordinal && values[ordinal-1].Status == status
	}) {
		return
	}
	var values []model.IntegrationSyncBatch
	_ = db.Order("id ASC").Find(&values).Error
	t.Fatalf("batch %d did not reach %s: %+v", ordinal, status, values)
}

func assertOrganizationFailedCheckpoint(t *testing.T, db *gorm.DB, taskID int, initial time.Time, reason hrsync.ReasonCode) {
	t.Helper()
	var task model.IntegrationSyncTask
	if err := db.First(&task, taskID).Error; err != nil || task.CheckpointAt == nil || !task.CheckpointAt.Equal(initial) {
		t.Fatalf("failed checkpoint advanced: checkpoint=%v initial=%v err=%v", task.CheckpointAt, initial, err)
	}
	var records int64
	if err := db.Model(&model.OrgSyncRecord{}).Where("error_code = ?", reason).Count(&records).Error; err != nil || records == 0 {
		t.Fatalf("reason %s records=%d err=%v", reason, records, err)
	}
}

var syncTestIDMu sync.Mutex
var syncTestIDValue = 910000

func nextSyncTestID() int {
	syncTestIDMu.Lock()
	defer syncTestIDMu.Unlock()
	syncTestIDValue++
	return syncTestIDValue
}
