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

func TestOrganizationSyncIntegritySchemaSQLiteIsIdempotent(t *testing.T) {
	db := migrateTestDB(t)
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateIntegrationRuntimeSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateIntegrationSyncSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OrgStructure{}); err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 2; run++ {
		if err := migrateOrganizationSyncIntegritySchema(db); err != nil {
			t.Fatalf("organization sync migration run %d: %v", run+1, err)
		}
	}
}

func TestOrganizationSourceCodeIndexesAllowDuplicateBusinessCodes(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.OrgUnit{}, &model.OrgPosition{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX uni_org_unit_source_code ON org_unit (source_system_code, source_code) WHERE source_code IS NOT NULL AND source_code <> ''`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX uni_org_position_source_code ON org_position (source_system_code, source_code) WHERE source_code IS NOT NULL AND source_code <> ''`).Error; err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 2; run++ {
		if err := migrateOrganizationSourceCodeIndexes(db); err != nil {
			t.Fatalf("organization source code index migration run %d: %v", run+1, err)
		}
	}
	if db.Migrator().HasIndex(&model.OrgUnit{}, "uni_org_unit_source_code") || db.Migrator().HasIndex(&model.OrgPosition{}, "uni_org_position_source_code") {
		t.Fatal("legacy unique source code index still exists")
	}
	units := []model.OrgUnit{
		{Basic: model.Basic{Id: 88001}, SourceSystemCode: "hr_source", SourceId: "unit-1", SourceCode: "SHARED", Code: "unit-1", Name: "Unit 1"},
		{Basic: model.Basic{Id: 88002}, SourceSystemCode: "hr_source", SourceId: "unit-2", SourceCode: "SHARED", Code: "unit-2", Name: "Unit 2"},
	}
	if err := db.Create(&units).Error; err != nil {
		t.Fatalf("duplicate organization business code rejected: %v", err)
	}
	positions := []model.OrgPosition{
		{Basic: model.Basic{Id: 88101}, SourceSystemCode: "hr_source", SourceId: "position-1", SourceCode: "SHARED", Code: "position-1", Name: "Position 1", OrgUnitId: units[0].Id},
		{Basic: model.Basic{Id: 88102}, SourceSystemCode: "hr_source", SourceId: "position-2", SourceCode: "SHARED", Code: "position-2", Name: "Position 2", OrgUnitId: units[1].Id},
	}
	if err := db.Create(&positions).Error; err != nil {
		t.Fatalf("duplicate position business code rejected: %v", err)
	}
}

func TestOrganizationSyncIntegritySchemaPostgreSQLConstraints(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := testutil.OpenPostgres(t, postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("organization_sync_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error })
	db, err := testutil.OpenPostgres(t, postgres.Open(postgresDSNWithSearchPath(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateIntegrationRuntimeSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateIntegrationSyncSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OrgStructure{}); err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 2; run++ {
		if err := migrateOrganizationSyncIntegritySchema(db); err != nil {
			t.Fatalf("organization sync migration run %d: %v", run+1, err)
		}
	}

	system, definition := integrationRuntimeMigrationFixtures(t, db)
	execution := validIntegrationExecutionFixture(99601, "INT-ORG-99601", system, definition, "org-sync-99601")
	if err := db.Create(&execution).Error; err != nil {
		t.Fatal(err)
	}
	batch := model.OrgSyncBatch{Basic: model.Basic{Id: 99602, State: true}, BatchNo: "ORG-99602", ExecutionId: &execution.Id, SyncType: "incremental", ObjectScope: "legal_entity", Status: "success"}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	duplicate := batch
	duplicate.Id, duplicate.BatchNo = 99603, "ORG-99603"
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("expected one Organization batch per Integration execution")
	}
	missingExecution := 999999
	badFK := model.OrgSyncBatch{Basic: model.Basic{Id: 99604, State: true}, BatchNo: "ORG-99604", ExecutionId: &missingExecution, SyncType: "incremental", ObjectScope: "legal_entity", Status: "failed"}
	if err := db.Create(&badFK).Error; err == nil {
		t.Fatal("expected IntegrationExecution foreign key violation")
	}

	record := model.OrgSyncRecord{Basic: model.Basic{Id: 99605, State: true}, BatchId: batch.Id, ObjectType: "legal_entity", SourceId: "source-1", Action: model.OrgSyncRecordActionCreate, Status: "success"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	duplicateRecord := record
	duplicateRecord.Id = 99606
	if err := db.Create(&duplicateRecord).Error; err == nil {
		t.Fatal("expected Organization business record idempotency violation")
	}
	invalidAction := record
	invalidAction.Id, invalidAction.SourceId, invalidAction.Action = 99607, "source-2", "insert"
	if err := db.Create(&invalidAction).Error; err == nil {
		t.Fatal("expected controlled action check violation")
	}
	missingIDError := model.OrgSyncRecord{Basic: model.Basic{Id: 99608, State: true}, BatchId: batch.Id, ObjectType: "legal_entity", Action: model.OrgSyncRecordActionError, Status: "failed", ErrorCode: "org_sync_source_id_missing"}
	if err := db.Create(&missingIDError).Error; err != nil {
		t.Fatalf("missing source ID diagnostic must remain representable: %v", err)
	}

	for index, structureType := range []string{model.OrgStructureTypeManagement, model.OrgStructureTypeLegal} {
		structure := model.OrgStructure{Basic: model.Basic{Id: 99700 + index, State: true}, Code: fmt.Sprintf("structure_%d", index), Name: "Structure", StructureType: structureType, SourceSystemCode: "hr_source", Status: "enabled", SyncStatus: "success"}
		if err := db.Create(&structure).Error; err != nil {
			t.Fatalf("create %s structure: %v", structureType, err)
		}
	}
	unknown := model.OrgStructure{Basic: model.Basic{Id: 99703, State: true}, Code: "unknown", Name: "Unknown", StructureType: "matrix", SourceSystemCode: "hr_source", Status: "enabled", SyncStatus: "success"}
	if err := db.Create(&unknown).Error; err == nil {
		t.Fatal("expected unknown structure type rejection")
	}
}
