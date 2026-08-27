package model_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	testutil "backend/internal/test"
	"backend/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestOrganizationPostgreSQLPartialUniqueConstraints(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)

	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}

	schemaName := fmt.Sprintf("org_model_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create PostgreSQL test schema: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error
	})

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   schemaName + ".",
			SingularTable: true,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
		NowFunc:                                  model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL organization schema: %v", err)
	}
	if err := db.AutoMigrate(organizationModels()...); err != nil {
		t.Fatalf("migrate PostgreSQL organization models: %v", err)
	}

	for _, indexName := range []string{
		"uni_org_legal_entity_source_code",
		"uni_org_legal_entity_credit",
		"uni_org_structure_source",
		"uni_org_structure_node_current",
		"uni_org_position_unit_code",
		"uni_org_employee_source_code",
		"uni_org_employee_user",
		"uni_org_assignment_current_primary",
	} {
		assertPostgreSQLPartialIndex(t, db, schemaName, indexName)
	}

	units := []model.OrgUnit{
		{Basic: model.Basic{Id: 6001}, SourceSystemCode: "master", SourceId: "unit-pg-1", SourceCode: "SHARED", Code: "unit-pg-1", Name: "Unit 1"},
		{Basic: model.Basic{Id: 6002}, SourceSystemCode: "master", SourceId: "unit-pg-2", SourceCode: "SHARED", Code: "unit-pg-2", Name: "Unit 2"},
	}
	if err := db.Create(&units).Error; err != nil {
		t.Fatalf("duplicate organization source code should be allowed: %v", err)
	}
	positions := []model.OrgPosition{
		{Basic: model.Basic{Id: 6101}, SourceSystemCode: "master", SourceId: "position-pg-1", SourceCode: "SHARED", Code: "position-pg-1", Name: "Position 1", OrgUnitId: 6001},
		{Basic: model.Basic{Id: 6102}, SourceSystemCode: "master", SourceId: "position-pg-2", SourceCode: "SHARED", Code: "position-pg-2", Name: "Position 2", OrgUnitId: 6002},
	}
	if err := db.Create(&positions).Error; err != nil {
		t.Fatalf("duplicate position source code should be allowed: %v", err)
	}

	userId := 9001
	if err := db.Create(&model.OrgEmployee{
		SourceSystemCode: "master",
		SourceId:         "employee-pg-1",
		EmployeeNo:       "EMP-PG-1",
		Name:             "PostgreSQL Employee 1",
		UserId:           &userId,
	}).Error; err != nil {
		t.Fatalf("create first PostgreSQL employee binding: %v", err)
	}
	if err := db.Create(&model.OrgEmployee{
		SourceSystemCode: "master",
		SourceId:         "employee-pg-2",
		EmployeeNo:       "EMP-PG-2",
		Name:             "PostgreSQL Employee 2",
		UserId:           &userId,
	}).Error; err == nil {
		t.Fatal("expected PostgreSQL to reject duplicate non-null user_id")
	}

	if err := db.Create(&model.OrgStructureNode{
		StructureId:      7001,
		OrgUnitId:        7001,
		SourceSystemCode: "master",
		SourceId:         "node-pg-1",
		Path:             "/7001/",
	}).Error; err != nil {
		t.Fatalf("create first PostgreSQL current structure node: %v", err)
	}
	if err := db.Create(&model.OrgStructureNode{
		StructureId:      7001,
		OrgUnitId:        7001,
		SourceSystemCode: "master",
		SourceId:         "node-pg-2",
		Path:             "/7001/",
	}).Error; err == nil {
		t.Fatal("expected PostgreSQL to reject duplicate current structure node")
	}

	if err := db.Create(&model.OrgAssignment{
		SourceSystemCode: "master",
		SourceId:         "assignment-pg-1",
		EmployeeId:       8001,
		LegalEntityId:    8001,
		OrgUnitId:        8001,
		IsPrimary:        true,
	}).Error; err != nil {
		t.Fatalf("create first PostgreSQL current primary assignment: %v", err)
	}
	if err := db.Create(&model.OrgAssignment{
		SourceSystemCode: "master",
		SourceId:         "assignment-pg-2",
		EmployeeId:       8001,
		LegalEntityId:    8001,
		OrgUnitId:        8002,
		IsPrimary:        true,
	}).Error; err == nil {
		t.Fatal("expected PostgreSQL to reject duplicate current primary assignment")
	}
}

func assertPostgreSQLPartialIndex(
	t *testing.T,
	db *gorm.DB,
	schemaName string,
	indexName string,
) {
	t.Helper()

	var definition string
	if err := db.Raw(
		`SELECT indexdef FROM pg_indexes WHERE schemaname = ? AND indexname = ?`,
		schemaName,
		indexName,
	).Scan(&definition).Error; err != nil {
		t.Fatalf("query PostgreSQL index %s: %v", indexName, err)
	}
	if definition == "" {
		t.Fatalf("PostgreSQL index %s was not created", indexName)
	}
	if !strings.Contains(strings.ToUpper(definition), " WHERE ") {
		t.Fatalf("PostgreSQL index %s is not partial: %s", indexName, definition)
	}
}
