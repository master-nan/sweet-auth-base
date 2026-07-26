package main

import (
	"testing"

	testutil "backend/internal/test"
	"backend/model"
)

func TestCoreMigrationRegistersOrganizationDomainModels(t *testing.T) {
	db := testutil.OpenSQLite(t)

	if err := autoMigrateCoreSchema(db); err != nil {
		t.Fatalf("migrate core schema with organization models: %v", err)
	}
	if err := autoMigrateCoreSchema(db); err != nil {
		t.Fatalf("repeat core schema migration: %v", err)
	}

	for _, value := range []interface{}{
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgStructure{},
		&model.OrgStructureNode{},
		&model.OrgPosition{},
		&model.OrgEmployee{},
		&model.OrgAssignment{},
		&model.OrgSyncBatch{},
		&model.OrgSyncRecord{},
	} {
		if !db.Migrator().HasTable(value) {
			t.Errorf("core migration is missing organization table for %T", value)
		}
	}
}
