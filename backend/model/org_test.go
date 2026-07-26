package model_test

import (
	"reflect"
	"sync"
	"testing"
	"time"

	testutil "backend/internal/test"
	"backend/model"

	"gorm.io/gorm/schema"
)

func TestOrganizationModelsExposeReviewedTableAndFieldMappings(t *testing.T) {
	expected := map[string]struct {
		value   interface{}
		columns []string
	}{
		"org_legal_entity": {
			value: &model.OrgLegalEntity{},
			columns: []string{
				"id", "source_system_code", "source_id", "source_code", "code", "name",
				"entity_type", "parent_id", "status", "source_deleted", "sync_status",
				"local_note", "local_tags", "display_order", "local_handling_status",
			},
		},
		"org_unit": {
			value: &model.OrgUnit{},
			columns: []string{
				"id", "source_system_code", "source_id", "code", "name", "unit_type",
				"primary_legal_entity_id", "status", "source_deleted", "sync_status",
				"local_note", "local_tags", "display_order", "local_handling_status",
			},
		},
		"org_structure": {
			value: &model.OrgStructure{},
			columns: []string{
				"id", "code", "name", "structure_type", "source_system_code", "source_id",
				"status", "is_default", "valid_from", "valid_to", "sync_status",
			},
		},
		"org_structure_node": {
			value: &model.OrgStructureNode{},
			columns: []string{
				"id", "structure_id", "org_unit_id", "parent_node_id", "source_system_code",
				"source_id", "source_parent_id", "path", "level", "sort", "status",
				"source_deleted", "sync_status",
			},
		},
		"org_position": {
			value: &model.OrgPosition{},
			columns: []string{
				"id", "source_system_code", "source_id", "code", "name", "org_unit_id",
				"position_type", "job_level", "is_manager_position", "status",
				"source_deleted", "sync_status", "local_note",
			},
		},
		"org_employee": {
			value: &model.OrgEmployee{},
			columns: []string{
				"id", "source_system_code", "source_id", "employee_no", "name",
				"employment_status", "primary_legal_entity_id", "user_id",
				"source_deleted", "sync_status", "local_note", "local_tags",
			},
		},
		"org_assignment": {
			value: &model.OrgAssignment{},
			columns: []string{
				"id", "source_system_code", "source_id", "employee_id", "legal_entity_id",
				"org_unit_id", "position_id", "assignment_type", "is_primary",
				"valid_from", "valid_to", "status", "source_deleted", "sync_status",
			},
		},
		"org_sync_batch": {
			value: &model.OrgSyncBatch{},
			columns: []string{
				"id", "batch_no", "execution_id", "sync_type", "object_scope",
				"started_at", "completed_at", "total_count", "success_count",
				"failed_count", "skipped_count", "status", "error_summary",
			},
		},
		"org_sync_record": {
			value: &model.OrgSyncRecord{},
			columns: []string{
				"id", "batch_id", "execution_id", "object_type", "source_id",
				"source_code", "local_id", "action", "status", "error_code",
				"dependency_type", "dependency_key", "retry_count", "last_retry_at",
				"local_handling_status",
			},
		},
	}

	for expectedTable, item := range expected {
		parsed, err := schema.Parse(
			item.value,
			&sync.Map{},
			schema.NamingStrategy{SingularTable: true},
		)
		if err != nil {
			t.Fatalf("parse %T schema: %v", item.value, err)
		}
		if parsed.Table != expectedTable {
			t.Fatalf("%T table = %q, want %q", item.value, parsed.Table, expectedTable)
		}
		for _, column := range item.columns {
			if parsed.LookUpField(column) == nil {
				t.Errorf("%s is missing reviewed column %q", expectedTable, column)
			}
		}
		if parsed.LookUpField("organization_id") != nil {
			t.Errorf("%s must not expose ambiguous organization_id", expectedTable)
		}
	}
}

func TestOrganizationModelsMigrateAndCreateReviewedIndexes(t *testing.T) {
	db := testutil.OpenSQLite(t, organizationModels()...)

	for _, value := range organizationModels() {
		if !db.Migrator().HasTable(value) {
			t.Errorf("missing migrated table for %T", value)
		}
	}

	for _, item := range []struct {
		value interface{}
		index string
	}{
		{&model.OrgLegalEntity{}, "uni_org_legal_entity_source"},
		{&model.OrgUnit{}, "uni_org_unit_source"},
		{&model.OrgStructure{}, "uni_org_structure_code"},
		{&model.OrgStructureNode{}, "uni_org_structure_node_current"},
		{&model.OrgPosition{}, "uni_org_position_unit_code"},
		{&model.OrgEmployee{}, "uni_org_employee_user"},
		{&model.OrgAssignment{}, "uni_org_assignment_current_primary"},
		{&model.OrgSyncBatch{}, "uni_org_sync_batch_no"},
		{&model.OrgSyncRecord{}, "idx_org_sync_record_batch"},
	} {
		if !db.Migrator().HasIndex(item.value, item.index) {
			t.Errorf("%T is missing index %q", item.value, item.index)
		}
	}
}

func TestOrganizationSourceIdentityUsesStableBusinessKey(t *testing.T) {
	tests := []struct {
		name      string
		first     func() interface{}
		duplicate func() interface{}
	}{
		{
			name: "legal entity",
			first: func() interface{} {
				return &model.OrgLegalEntity{
					SourceSystemCode: "master",
					SourceId:         "legal-1",
					Code:             "LE-1",
					Name:             "Legal Entity 1",
				}
			},
			duplicate: func() interface{} {
				return &model.OrgLegalEntity{
					SourceSystemCode: "master",
					SourceId:         "legal-1",
					Code:             "LE-2",
					Name:             "Legal Entity 2",
				}
			},
		},
		{
			name: "unit",
			first: func() interface{} {
				return &model.OrgUnit{
					SourceSystemCode: "master",
					SourceId:         "unit-1",
					Code:             "OU-1",
					Name:             "Unit 1",
				}
			},
			duplicate: func() interface{} {
				return &model.OrgUnit{
					SourceSystemCode: "master",
					SourceId:         "unit-1",
					Code:             "OU-2",
					Name:             "Unit 2",
				}
			},
		},
		{
			name: "structure",
			first: func() interface{} {
				return &model.OrgStructure{
					Code:             "management-1",
					Name:             "Management 1",
					SourceSystemCode: "master",
					SourceId:         "structure-1",
				}
			},
			duplicate: func() interface{} {
				return &model.OrgStructure{
					Code:             "management-2",
					Name:             "Management 2",
					SourceSystemCode: "master",
					SourceId:         "structure-1",
				}
			},
		},
		{
			name: "structure node",
			first: func() interface{} {
				return &model.OrgStructureNode{
					StructureId:      1,
					OrgUnitId:        1,
					SourceSystemCode: "master",
					SourceId:         "node-1",
					Path:             "/1/",
				}
			},
			duplicate: func() interface{} {
				return &model.OrgStructureNode{
					StructureId:      2,
					OrgUnitId:        2,
					SourceSystemCode: "master",
					SourceId:         "node-1",
					Path:             "/2/",
				}
			},
		},
		{
			name: "position",
			first: func() interface{} {
				return &model.OrgPosition{
					SourceSystemCode: "master",
					SourceId:         "position-1",
					Code:             "POS-1",
					Name:             "Position 1",
					OrgUnitId:        1,
				}
			},
			duplicate: func() interface{} {
				return &model.OrgPosition{
					SourceSystemCode: "master",
					SourceId:         "position-1",
					Code:             "POS-2",
					Name:             "Position 2",
					OrgUnitId:        2,
				}
			},
		},
		{
			name: "employee",
			first: func() interface{} {
				return &model.OrgEmployee{
					SourceSystemCode: "master",
					SourceId:         "employee-1",
					EmployeeNo:       "EMP-1",
					Name:             "Employee 1",
				}
			},
			duplicate: func() interface{} {
				return &model.OrgEmployee{
					SourceSystemCode: "master",
					SourceId:         "employee-1",
					EmployeeNo:       "EMP-2",
					Name:             "Employee 2",
				}
			},
		},
		{
			name: "assignment",
			first: func() interface{} {
				return &model.OrgAssignment{
					SourceSystemCode: "master",
					SourceId:         "assignment-1",
					EmployeeId:       1,
					LegalEntityId:    1,
					OrgUnitId:        1,
				}
			},
			duplicate: func() interface{} {
				return &model.OrgAssignment{
					SourceSystemCode: "master",
					SourceId:         "assignment-1",
					EmployeeId:       2,
					LegalEntityId:    2,
					OrgUnitId:        2,
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := testutil.OpenSQLite(t, organizationModels()...)
			testutil.MustCreate(t, db, test.first())
			if err := db.Create(test.duplicate()).Error; err == nil {
				t.Fatal("expected duplicate source_system_code + source_id to fail")
			}
		})
	}
}

func TestOrgEmployeeUserBindingAllowsNullAndRejectsDuplicateAccount(t *testing.T) {
	db := testutil.OpenSQLite(t, organizationModels()...)

	testutil.MustCreate(t, db, &model.OrgEmployee{
		SourceSystemCode: "master",
		SourceId:         "employee-unbound-1",
		EmployeeNo:       "EMP-U1",
		Name:             "Unbound 1",
	})
	testutil.MustCreate(t, db, &model.OrgEmployee{
		SourceSystemCode: "master",
		SourceId:         "employee-unbound-2",
		EmployeeNo:       "EMP-U2",
		Name:             "Unbound 2",
	})

	userId := 101
	testutil.MustCreate(t, db, &model.OrgEmployee{
		SourceSystemCode: "master",
		SourceId:         "employee-bound-1",
		EmployeeNo:       "EMP-B1",
		Name:             "Bound 1",
		UserId:           &userId,
	})
	err := db.Create(&model.OrgEmployee{
		SourceSystemCode: "master",
		SourceId:         "employee-bound-2",
		EmployeeNo:       "EMP-B2",
		Name:             "Bound 2",
		UserId:           &userId,
	}).Error
	if err == nil {
		t.Fatal("expected non-null user_id to be unique")
	}
}

func TestOrganizationCurrentRecordConstraints(t *testing.T) {
	t.Run("structure node", func(t *testing.T) {
		db := testutil.OpenSQLite(t, organizationModels()...)
		testutil.MustCreate(t, db, &model.OrgStructureNode{
			StructureId:      1,
			OrgUnitId:        1,
			SourceSystemCode: "master",
			SourceId:         "node-current-1",
			Path:             "/1/",
		})
		if err := db.Create(&model.OrgStructureNode{
			StructureId:      1,
			OrgUnitId:        1,
			SourceSystemCode: "master",
			SourceId:         "node-current-2",
			Path:             "/1/",
		}).Error; err == nil {
			t.Fatal("expected one current node per structure and org unit")
		}

		expiredAt := time.Now().Add(-time.Hour)
		testutil.MustCreate(t, db, &model.OrgStructureNode{
			StructureId:      1,
			OrgUnitId:        1,
			SourceSystemCode: "master",
			SourceId:         "node-history-1",
			Path:             "/1/",
			ValidTo:          &expiredAt,
		})
	})

	t.Run("primary assignment", func(t *testing.T) {
		db := testutil.OpenSQLite(t, organizationModels()...)
		testutil.MustCreate(t, db, &model.OrgAssignment{
			SourceSystemCode: "master",
			SourceId:         "assignment-primary-1",
			EmployeeId:       1,
			LegalEntityId:    1,
			OrgUnitId:        1,
			IsPrimary:        true,
		})
		if err := db.Create(&model.OrgAssignment{
			SourceSystemCode: "master",
			SourceId:         "assignment-primary-2",
			EmployeeId:       1,
			LegalEntityId:    1,
			OrgUnitId:        2,
			IsPrimary:        true,
		}).Error; err == nil {
			t.Fatal("expected one open-ended current primary assignment per employee")
		}

		expiredAt := time.Now().Add(-time.Hour)
		testutil.MustCreate(t, db, &model.OrgAssignment{
			SourceSystemCode: "master",
			SourceId:         "assignment-primary-history",
			EmployeeId:       1,
			LegalEntityId:    1,
			OrgUnitId:        2,
			IsPrimary:        true,
			ValidTo:          &expiredAt,
		})
	})
}

func TestOrganizationModelsPersistBasicAssociationsAndDefaults(t *testing.T) {
	db := testutil.OpenSQLite(t, organizationModels()...)

	legalEntity := &model.OrgLegalEntity{
		Basic:            model.Basic{Id: 101},
		SourceSystemCode: "master",
		SourceId:         "legal-association",
		Code:             "LE-ASSOC",
		Name:             "Association Legal Entity",
	}
	testutil.MustCreate(t, db, legalEntity)

	unit := &model.OrgUnit{
		Basic:                model.Basic{Id: 201},
		SourceSystemCode:     "master",
		SourceId:             "unit-association",
		Code:                 "OU-ASSOC",
		Name:                 "Association Unit",
		PrimaryLegalEntityId: &legalEntity.Id,
	}
	testutil.MustCreate(t, db, unit)

	structure := &model.OrgStructure{
		Basic:            model.Basic{Id: 301},
		Code:             "management-association",
		Name:             "Association Management Structure",
		SourceSystemCode: "master",
		SourceId:         "structure-association",
	}
	testutil.MustCreate(t, db, structure)

	node := &model.OrgStructureNode{
		Basic:            model.Basic{Id: 401},
		StructureId:      structure.Id,
		OrgUnitId:        unit.Id,
		SourceSystemCode: "master",
		SourceId:         "node-association",
		Path:             "/1/",
	}
	testutil.MustCreate(t, db, node)

	position := &model.OrgPosition{
		Basic:            model.Basic{Id: 501},
		SourceSystemCode: "master",
		SourceId:         "position-association",
		Code:             "POS-ASSOC",
		Name:             "Association Position",
		OrgUnitId:        unit.Id,
	}
	testutil.MustCreate(t, db, position)

	employee := &model.OrgEmployee{
		Basic:                model.Basic{Id: 601},
		SourceSystemCode:     "master",
		SourceId:             "employee-association",
		EmployeeNo:           "EMP-ASSOC",
		Name:                 "Association Employee",
		PrimaryLegalEntityId: &legalEntity.Id,
	}
	testutil.MustCreate(t, db, employee)

	assignment := &model.OrgAssignment{
		Basic:            model.Basic{Id: 701},
		SourceSystemCode: "master",
		SourceId:         "assignment-association",
		EmployeeId:       employee.Id,
		LegalEntityId:    legalEntity.Id,
		OrgUnitId:        unit.Id,
		PositionId:       &position.Id,
		IsPrimary:        true,
	}
	testutil.MustCreate(t, db, assignment)

	batch := &model.OrgSyncBatch{
		Basic:   model.Basic{Id: 801},
		BatchNo: "ORG-BATCH-ASSOC",
	}
	testutil.MustCreate(t, db, batch)
	record := &model.OrgSyncRecord{
		Basic:      model.Basic{Id: 901},
		BatchId:    batch.Id,
		ObjectType: "employee",
		SourceId:   employee.SourceId,
		LocalId:    &employee.Id,
		Action:     "insert",
	}
	testutil.MustCreate(t, db, record)

	var storedAssignment model.OrgAssignment
	if err := db.
		Preload("Employee").
		Preload("LegalEntity").
		Preload("OrgUnit").
		Preload("Position").
		First(&storedAssignment, assignment.Id).Error; err != nil {
		t.Fatalf("load assignment associations: %v", err)
	}
	if storedAssignment.Employee == nil || storedAssignment.Employee.Id != employee.Id {
		t.Fatal("assignment employee association was not persisted")
	}
	if storedAssignment.LegalEntity == nil || storedAssignment.LegalEntity.Id != legalEntity.Id {
		t.Fatal("assignment legal entity association was not persisted")
	}
	if storedAssignment.OrgUnit == nil || storedAssignment.OrgUnit.Id != unit.Id {
		t.Fatal("assignment org unit association was not persisted")
	}
	if storedAssignment.Position == nil || storedAssignment.Position.Id != position.Id {
		t.Fatal("assignment position association was not persisted")
	}

	var storedNode model.OrgStructureNode
	if err := db.Preload("Structure").Preload("OrgUnit").First(&storedNode, node.Id).Error; err != nil {
		t.Fatalf("load structure node associations: %v", err)
	}
	if storedNode.Structure == nil || storedNode.OrgUnit == nil {
		t.Fatal("structure node associations were not persisted")
	}

	var storedRecord model.OrgSyncRecord
	if err := db.Preload("Batch").First(&storedRecord, record.Id).Error; err != nil {
		t.Fatalf("load sync record association: %v", err)
	}
	if storedRecord.Batch == nil || storedRecord.Batch.Id != batch.Id {
		t.Fatal("sync record batch association was not persisted")
	}

	if legalEntity.Status != "enabled" || legalEntity.EntityType != "legal_company" || legalEntity.SyncStatus != "pending" {
		t.Fatalf(
			"unexpected legal entity defaults: status=%q type=%q sync=%q",
			legalEntity.Status,
			legalEntity.EntityType,
			legalEntity.SyncStatus,
		)
	}
	if assignment.AssignmentType != "primary" || assignment.Status != "enabled" || assignment.SyncStatus != "pending" {
		t.Fatalf(
			"unexpected assignment defaults: type=%q status=%q sync=%q",
			assignment.AssignmentType,
			assignment.Status,
			assignment.SyncStatus,
		)
	}
	if batch.SyncType != "incremental" || batch.ObjectScope != "all" || batch.Status != "pending" {
		t.Fatalf(
			"unexpected sync batch defaults: type=%q scope=%q status=%q",
			batch.SyncType,
			batch.ObjectScope,
			batch.Status,
		)
	}
}

func organizationModels() []interface{} {
	return []interface{}{
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgStructure{},
		&model.OrgStructureNode{},
		&model.OrgPosition{},
		&model.OrgEmployee{},
		&model.OrgAssignment{},
		&model.OrgSyncBatch{},
		&model.OrgSyncRecord{},
	}
}

func TestOrganizationModelSourceAndPlatformFieldsRemainDistinct(t *testing.T) {
	sourceFields := []string{
		"SourceSystemCode",
		"SourceId",
		"EmployeeNo",
		"Name",
		"EmploymentStatus",
		"PrimaryLegalEntityId",
		"SourceDeleted",
		"SyncStatus",
	}
	platformFields := []string{"UserId", "LocalNote", "LocalTags"}

	employeeType := reflect.TypeOf(model.OrgEmployee{})
	for _, name := range sourceFields {
		if _, ok := employeeType.FieldByName(name); !ok {
			t.Errorf("org_employee source field %s is missing", name)
		}
	}
	for _, name := range platformFields {
		if _, ok := employeeType.FieldByName(name); !ok {
			t.Errorf("org_employee platform field %s is missing", name)
		}
	}

	userIdField, ok := employeeType.FieldByName("UserId")
	if !ok {
		t.Fatal("org_employee UserId field is missing")
	}
	if userIdField.Type.Kind() != reflect.Ptr || userIdField.Type.Elem().Kind() != reflect.Int {
		t.Fatalf("org_employee UserId type = %s, want nullable *int account binding", userIdField.Type)
	}
}
