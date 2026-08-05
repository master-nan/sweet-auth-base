package main

import (
	testutil "backend/internal/test"
	"backend/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestIntegrationRuntimeSchemaIsIdempotentAndUnique(t *testing.T) {
	db := migrateTestDB(t)
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("migrate integration configuration: %v", err)
	}
	if err := migrateIntegrationRuntimeSchema(db); err != nil {
		t.Fatalf("first runtime migration: %v", err)
	}
	if err := migrateIntegrationRuntimeSchema(db); err != nil {
		t.Fatalf("second runtime migration: %v", err)
	}
	if !db.Migrator().HasTable(&model.IntegrationExecution{}) || !db.Migrator().HasTable(&model.IntegrationLog{}) {
		t.Fatal("integration runtime tables were not created")
	}

	system, definition := integrationRuntimeMigrationFixtures(t, db)
	execution := validIntegrationExecutionFixture(301, "INT-301", system, definition, "request-301")
	if err := db.Create(&execution).Error; err != nil {
		t.Fatalf("create execution: %v", err)
	}

	duplicateNumber := validIntegrationExecutionFixture(302, execution.ExecutionNo, system, definition, "request-302")
	if err := db.Create(&duplicateNumber).Error; err == nil {
		t.Fatal("expected duplicate execution_no to be rejected")
	}
	duplicateIdempotency := validIntegrationExecutionFixture(303, "INT-303", system, definition, execution.IdempotencyKey)
	if err := db.Create(&duplicateIdempotency).Error; err == nil {
		t.Fatal("expected duplicate interface version and idempotency identity to be rejected")
	}

	startedAt := model.Now()
	firstLog := model.IntegrationLog{
		Basic: model.Basic{Id: 401, State: true}, ExecutionID: execution.Id, AttemptNo: 1,
		Status: model.IntegrationLogStatusRunning, StartedAt: startedAt,
		ResultCertainty: model.IntegrationResultCertaintyUnknown,
	}
	if err := db.Create(&firstLog).Error; err != nil {
		t.Fatalf("create integration log: %v", err)
	}
	firstLog.Id = 402
	if err := db.Create(&firstLog).Error; err == nil {
		t.Fatal("expected duplicate execution and attempt number to be rejected")
	}
}

func TestIntegrationRuntimeSchemaPostgreSQLConstraints(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	schemaName := fmt.Sprintf("integration_runtime_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create PostgreSQL test schema: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error
	})

	db, err := gorm.Open(postgres.Open(postgresDSNWithSearchPath(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
		NowFunc:                                  model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("migrate PostgreSQL integration configuration: %v", err)
	}
	for run := 0; run < 2; run++ {
		if err := migrateIntegrationRuntimeSchema(db); err != nil {
			t.Fatalf("migrate PostgreSQL integration runtime run %d: %v", run+1, err)
		}
	}

	var tableCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = ? AND table_name IN ('integration_execution','integration_log')
	`, schemaName).Scan(&tableCount).Error; err != nil {
		t.Fatalf("count runtime tables: %v", err)
	}
	if tableCount != 2 {
		t.Fatalf("runtime table count = %d, want 2", tableCount)
	}

	system, definition := integrationRuntimeMigrationFixtures(t, db)
	invalid := validIntegrationExecutionFixture(501, "INT-501", system, definition, "request-501")
	invalid.Status = "invalid"
	if err := db.Create(&invalid).Error; err == nil {
		t.Fatal("expected invalid execution status to be rejected")
	}
	invalid = validIntegrationExecutionFixture(502, "INT-502", system, definition, "request-502")
	invalid.ExternalSystemID = 999999
	if err := db.Create(&invalid).Error; err == nil {
		t.Fatal("expected missing external system reference to be rejected")
	}
	invalid = validIntegrationExecutionFixture(503, "INT-503", system, definition, "request-503")
	invalid.InterfaceVersion = 0
	if err := db.Create(&invalid).Error; err == nil {
		t.Fatal("expected invalid interface version to be rejected")
	}
	invalid = validIntegrationExecutionFixture(504, "INT-504", system, definition, "request-504")
	invalid.IdempotencyScope = " "
	if err := db.Create(&invalid).Error; err == nil {
		t.Fatal("expected blank idempotency scope to be rejected")
	}
	invalid = validIntegrationExecutionFixture(505, "INT-505", system, definition, "request-505")
	invalid.ErrorCategory = "unknown_category"
	if err := db.Create(&invalid).Error; err == nil {
		t.Fatal("expected invalid error category to be rejected")
	}
}

func integrationRuntimeMigrationFixtures(
	t *testing.T,
	db *gorm.DB,
) (model.ExternalSystem, model.InterfaceDefinition) {
	t.Helper()
	system := model.ExternalSystem{
		Basic: model.Basic{Id: 101, State: true}, SystemCode: "runtime_hr", Name: "Runtime HR",
		SystemType: model.ExternalSystemTypeHR, BaseURL: "https://hr.example.com",
		OwnerIdentifier: "owner-runtime", OwnerName: "运行负责人",
		Status: model.ExternalSystemStatusEnabled, Revision: 1,
	}
	if err := db.Create(&system).Error; err != nil {
		t.Fatalf("create runtime system fixture: %v", err)
	}
	definition := model.InterfaceDefinition{
		Basic: model.Basic{Id: 201, State: true}, ExternalSystemID: system.Id,
		InterfaceCode: "employee_list", Name: "员工列表", Version: 1,
		Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET,
		RelativePath: "/api/employees", TimeoutSeconds: 30, ResponseLimit: 1024,
		Status: model.InterfaceDefinitionStatusEnabled, Revision: 1,
	}
	if err := db.Create(&definition).Error; err != nil {
		t.Fatalf("create runtime interface fixture: %v", err)
	}
	return system, definition
}

func validIntegrationExecutionFixture(
	id int,
	number string,
	system model.ExternalSystem,
	definition model.InterfaceDefinition,
	idempotencyKey string,
) model.IntegrationExecution {
	return model.IntegrationExecution{
		Basic: model.Basic{Id: id, State: true}, ExecutionNo: number,
		ExternalSystemID: system.Id, ExternalSystemCode: system.SystemCode, ExternalSystemName: system.Name,
		InterfaceDefinitionID: definition.Id, InterfaceCode: definition.InterfaceCode,
		InterfaceName: definition.Name, InterfaceVersion: definition.Version,
		TriggerSource: model.IntegrationTriggerSourceManual, Status: model.IntegrationExecutionStatusCreated,
		IdempotencyScope: "acceptance", IdempotencyKey: idempotencyKey,
		InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Revision: 1,
	}
}
