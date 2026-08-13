package impl

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestOrgLegalEntityRepositoryQueryPaginatesAndFilters(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.OrgLegalEntity{})
	repo := NewOrgLegalEntityRepositoryImpl(&database.PrimaryDB{DB: db})
	entities := []model.OrgLegalEntity{
		newLegalEntityFixture("src-1", "LE-001", "Alpha", "legal_company", "enabled"),
		newLegalEntityFixture("src-2", "LE-002", "Beta", "branch", "enabled"),
		newLegalEntityFixture("src-3", "LE-003", "Gamma", "legal_company", "enabled"),
		newLegalEntityFixture("src-4", "LE-004", "Disabled", "legal_company", "disabled"),
	}
	for index := range entities {
		entities[index].Id = index + 1
		testutil.MustCreate(t, db, &entities[index])
	}

	req := &request.OrgLegalEntityQueryReq{
		Basic: request.Basic{
			Page: 1,
			Num:  2,
			Order: request.Order{
				Field: "code",
				IsAsc: true,
			},
		},
		EntityType: "legal_company",
		Status:     "enabled",
	}
	result, err := repo.Query(nil, req, organizationTestTable("org_legal_entity", map[string]enum.SysTableFieldType{
		"code":        enum.VarcharFieldType,
		"entity_type": enum.VarcharFieldType,
		"status":      enum.VarcharFieldType,
	}), repository.OrgLegalEntityReadScope{AsOf: model.Now()})
	if err != nil {
		t.Fatalf("query legal entities: %v", err)
	}
	if result.Total != 2 || len(result.Data) != 2 {
		t.Fatalf("unexpected pagination total=%d rows=%d", result.Total, len(result.Data))
	}
	if result.Data[0].Code != "LE-001" || result.Data[1].Code != "LE-003" {
		t.Fatalf("unexpected filtered order: %+v", result.Data)
	}
	if req.Filters != nil {
		t.Fatalf("repository mutated caller filters: %+v", req.Filters)
	}

	restrictedTable := organizationTestTable("org_legal_entity", map[string]enum.SysTableFieldType{
		"code": enum.VarcharFieldType,
	})
	restrictedTable.TableFields = append(restrictedTable.TableFields, model.SysTableField{
		FieldCode: "source_id",
		FieldType: enum.VarcharFieldType,
	})
	restrictedResult, err := repo.Query(nil, &request.OrgLegalEntityQueryReq{
		Basic: request.Basic{Filters: map[string]any{"source_id": "src-1"}},
	}, restrictedTable, repository.OrgLegalEntityReadScope{AsOf: model.Now()})
	if err != nil {
		t.Fatalf("query with restricted metadata field: %v", err)
	}
	if restrictedResult.Total != 0 || len(restrictedResult.Data) != 0 {
		t.Fatalf("hidden source_id was accepted as a generic query field: %+v", restrictedResult)
	}
}

func TestOrganizationCoreRepositoriesConditionQueries(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgEmployee{},
		&model.OrgPosition{},
		&model.OrgAssignment{},
	)
	primaryDB := &database.PrimaryDB{DB: db}

	legal := newLegalEntityFixture("legal-1", "LE-001", "Legal", "legal_company", "enabled")
	legal.Id = 101
	testutil.MustCreate(t, db, &legal)
	unit := model.OrgUnit{
		Basic:                model.Basic{Id: 201},
		SourceSystemCode:     "authority",
		SourceId:             "unit-1",
		SourceCode:           "SRC-U-1",
		Code:                 "U-001",
		Name:                 "Operations",
		UnitType:             "department",
		PrimaryLegalEntityId: &legal.Id,
		Status:               "enabled",
		SyncStatus:           "synced",
	}
	testutil.MustCreate(t, db, &unit)
	userID := 9001
	employee := model.OrgEmployee{
		Basic:                model.Basic{Id: 301},
		SourceSystemCode:     "authority",
		SourceId:             "employee-1",
		SourceCode:           "SRC-E-1",
		EmployeeNo:           "EMP-001",
		Name:                 "Alice",
		Mobile:               "13800138000",
		Email:                "alice@example.com",
		EmploymentStatus:     "active",
		PrimaryLegalEntityId: &legal.Id,
		SourceVersion:        "version-1",
		SyncStatus:           "synced",
		UserId:               &userID,
	}
	testutil.MustCreate(t, db, &employee)
	position := model.OrgPosition{
		Basic:            model.Basic{Id: 401},
		SourceSystemCode: "authority",
		SourceId:         "position-1",
		SourceCode:       "SRC-P-1",
		Code:             "P-001",
		Name:             "Manager",
		OrgUnitId:        unit.Id,
		PositionType:     "management",
		Status:           "enabled",
		SyncStatus:       "synced",
	}
	testutil.MustCreate(t, db, &position)
	assignment := model.OrgAssignment{
		Basic:            model.Basic{Id: 501},
		SourceSystemCode: "authority",
		SourceId:         "assignment-1",
		EmployeeId:       employee.Id,
		LegalEntityId:    legal.Id,
		OrgUnitId:        unit.Id,
		PositionId:       &position.Id,
		AssignmentType:   "primary",
		IsPrimary:        true,
		Status:           "enabled",
		SyncStatus:       "synced",
	}
	testutil.MustCreate(t, db, &assignment)

	unitResult, err := NewOrgUnitRepositoryImpl(primaryDB).Query(nil, &request.OrgUnitQueryReq{
		PrimaryLegalEntityId: &legal.Id,
		Status:               "enabled",
	}, organizationTestTable("org_unit", map[string]enum.SysTableFieldType{
		"primary_legal_entity_id": enum.BigIntFieldType,
		"status":                  enum.VarcharFieldType,
	}))
	if err != nil || unitResult.Total != 1 || unitResult.Data[0].Id != unit.Id {
		t.Fatalf("unexpected unit result=%+v err=%v", unitResult, err)
	}
	employeeResult, err := NewOrgEmployeeRepositoryImpl(primaryDB).Query(nil, &request.OrgEmployeeQueryReq{
		EmploymentStatus: "active",
		BoundUserId:      &userID,
	}, organizationTestTable("org_employee", map[string]enum.SysTableFieldType{
		"employment_status": enum.VarcharFieldType,
		"user_id":           enum.BigIntFieldType,
	}))
	if err != nil || employeeResult.Total != 1 || employeeResult.Data[0].Id != employee.Id {
		t.Fatalf("unexpected employee result=%+v err=%v", employeeResult, err)
	}
	positionResult, err := NewOrgPositionRepositoryImpl(primaryDB).Query(nil, &request.OrgPositionQueryReq{
		OrgUnitId:    &unit.Id,
		PositionType: "management",
	}, organizationTestTable("org_position", map[string]enum.SysTableFieldType{
		"org_unit_id":   enum.BigIntFieldType,
		"position_type": enum.VarcharFieldType,
	}))
	if err != nil || positionResult.Total != 1 || positionResult.Data[0].Id != position.Id {
		t.Fatalf("unexpected position result=%+v err=%v", positionResult, err)
	}
	assignmentResult, err := NewOrgAssignmentRepositoryImpl(primaryDB).Query(nil, &request.OrgAssignmentQueryReq{
		EmployeeId: &employee.Id,
		IsPrimary:  boolPointer(true),
	}, organizationTestTable("org_assignment", map[string]enum.SysTableFieldType{
		"employee_id": enum.BigIntFieldType,
		"is_primary":  enum.BooleanFieldType,
	}))
	if err != nil || assignmentResult.Total != 1 || assignmentResult.Data[0].Id != assignment.Id {
		t.Fatalf("unexpected assignment result=%+v err=%v", assignmentResult, err)
	}
}

func TestOrganizationRepositoriesUseStableUniqueLookups(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgEmployee{},
		&model.OrgPosition{},
		&model.OrgAssignment{},
		&model.OrgSyncBatch{},
	)
	primaryDB := &database.PrimaryDB{DB: db}

	legal := newLegalEntityFixture("legal-source", "LE-001", "Legal", "legal_company", "enabled")
	legal.Id = 101
	testutil.MustCreate(t, db, &legal)
	unit := model.OrgUnit{
		Basic:            model.Basic{Id: 201},
		SourceSystemCode: "authority", SourceId: "unit-source", Code: "U-001",
		Name: "Unit", UnitType: "department", Status: "enabled", SyncStatus: "synced",
	}
	testutil.MustCreate(t, db, &unit)
	employee := model.OrgEmployee{
		Basic:            model.Basic{Id: 301},
		SourceSystemCode: "authority", SourceId: "employee-source", EmployeeNo: "EMP-001",
		Name: "Employee", EmploymentStatus: "active", SyncStatus: "synced",
	}
	testutil.MustCreate(t, db, &employee)
	position := model.OrgPosition{
		Basic:            model.Basic{Id: 401},
		SourceSystemCode: "authority", SourceId: "position-source", Code: "P-001",
		Name: "Position", OrgUnitId: unit.Id, PositionType: "professional",
		Status: "enabled", SyncStatus: "synced",
	}
	testutil.MustCreate(t, db, &position)
	assignment := model.OrgAssignment{
		Basic:            model.Basic{Id: 501},
		SourceSystemCode: "authority", SourceId: "assignment-source",
		EmployeeId: employee.Id, LegalEntityId: legal.Id, OrgUnitId: unit.Id,
		AssignmentType: "primary", IsPrimary: true, Status: "enabled", SyncStatus: "synced",
	}
	testutil.MustCreate(t, db, &assignment)
	batch := model.OrgSyncBatch{
		Basic: model.Basic{Id: 601}, BatchNo: "ORG-BATCH-001",
		SyncType: "full", ObjectScope: "all", Status: "success",
	}
	testutil.MustCreate(t, db, &batch)

	if got, err := NewOrgLegalEntityRepositoryImpl(primaryDB).FindBySourceIdentity(nil, "authority", "legal-source"); err != nil || got.Id != legal.Id {
		t.Fatalf("find legal entity by source identity got=%+v err=%v", got, err)
	}
	if got, err := NewOrgUnitRepositoryImpl(primaryDB).FindByCode(nil, "authority", "U-001"); err != nil || got.Id != unit.Id {
		t.Fatalf("find unit by code got=%+v err=%v", got, err)
	}
	if got, err := NewOrgEmployeeRepositoryImpl(primaryDB).FindByEmployeeNo(nil, "authority", "EMP-001"); err != nil || got.Id != employee.Id {
		t.Fatalf("find employee by number got=%+v err=%v", got, err)
	}
	if got, err := NewOrgPositionRepositoryImpl(primaryDB).FindBySourceIdentity(nil, "authority", "position-source"); err != nil || got.Id != position.Id {
		t.Fatalf("find position by source identity got=%+v err=%v", got, err)
	}
	if got, err := NewOrgAssignmentRepositoryImpl(primaryDB).FindBySourceIdentity(nil, "authority", "assignment-source"); err != nil || got.Id != assignment.Id {
		t.Fatalf("find assignment by source identity got=%+v err=%v", got, err)
	}
	if got, err := NewOrgSyncBatchRepositoryImpl(primaryDB).FindByBatchNo(nil, "ORG-BATCH-001"); err != nil || got.Id != batch.Id {
		t.Fatalf("find sync batch by number got=%+v err=%v", got, err)
	}

	_, err := NewOrgEmployeeRepositoryImpl(primaryDB).FindBySourceIdentity(nil, "authority", "missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm record-not-found propagation, got %v", err)
	}
}

func TestOrgEmployeeRepositoryQueryDoesNotLoadSensitiveOrInternalColumns(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.OrgEmployee{})
	repo := NewOrgEmployeeRepositoryImpl(&database.PrimaryDB{DB: db})
	employee := model.OrgEmployee{
		Basic:            model.Basic{Id: 1},
		SourceSystemCode: "authority",
		SourceId:         "employee-source",
		SourceCode:       "source-code",
		EmployeeNo:       "EMP-001",
		Name:             "Alice",
		Mobile:           "13800138000",
		Email:            "alice@example.com",
		EmploymentStatus: "active",
		SourceVersion:    "version-1",
		SyncStatus:       "synced",
	}
	testutil.MustCreate(t, db, &employee)

	result, err := repo.Query(nil, &request.OrgEmployeeQueryReq{}, model.SysTable{TableCode: "org_employee"})
	if err != nil {
		t.Fatalf("query employees: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected one employee, got %d", len(result.Data))
	}
	got := result.Data[0]
	if got.Mobile != "" || got.Email != "" || got.SourceId != "" || got.SourceVersion != "" || got.SyncStatus != "" {
		t.Fatalf("list query loaded restricted columns: %+v", got)
	}
	if got.EmployeeNo != employee.EmployeeNo || got.Name != employee.Name {
		t.Fatalf("list query omitted safe display fields: %+v", got)
	}
}

func TestOrgEmployeeRepositoryQueryUsersForBindingReturnsSafeStableOptions(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysUser{}, &model.OrgEmployee{})
	repo := NewOrgEmployeeRepositoryImpl(&database.PrimaryDB{DB: db})
	users := []model.SysUser{
		{Basic: model.Basic{Id: 1, State: true}, UserName: "alpha"},
		{Basic: model.Basic{Id: 2, State: true}, UserName: "beta", Password: "secret"},
		{Basic: model.Basic{Id: 3, State: false}, UserName: "disabled"},
		{Basic: model.Basic{Id: 4, State: true}, UserName: "gamma"},
	}
	for index := range users {
		testutil.MustCreate(t, db, &users[index])
	}
	if err := db.Model(&model.SysUser{}).Where("id = ?", 3).Update("state", false).Error; err != nil {
		t.Fatalf("disable user fixture: %v", err)
	}
	boundUserID := 2
	testutil.MustCreate(t, db, &model.OrgEmployee{
		Basic:            model.Basic{Id: 10, State: true},
		SourceSystemCode: "authority",
		SourceId:         "employee-10",
		EmployeeNo:       "EMP-010",
		Name:             "Bound Employee",
		EmploymentStatus: "active",
		UserId:           &boundUserID,
	})

	result, err := repo.QueryUsersForBinding(nil, "", 1, 20)
	if err != nil {
		t.Fatalf("query binding users: %v", err)
	}
	if result.Total != 2 || len(result.Data) != 2 {
		t.Fatalf("binding user options total=%d rows=%d", result.Total, len(result.Data))
	}
	if result.Data[0].UserName != "alpha" || result.Data[0].Disabled {
		t.Fatalf("unexpected first binding option: %+v", result.Data[0])
	}
	if result.Data[1].UserName != "gamma" || result.Data[1].Disabled {
		t.Fatalf("unexpected second binding option: %+v", result.Data[1])
	}

	secondPage, err := repo.QueryUsersForBinding(nil, "", 2, 1)
	if err != nil {
		t.Fatalf("query second binding user page: %v", err)
	}
	if secondPage.Total != 2 || len(secondPage.Data) != 1 || secondPage.Data[0].UserId != 4 {
		t.Fatalf("unexpected second binding user page: %+v", secondPage)
	}

	filtered, err := repo.QueryUsersForBinding(nil, "ALP", 1, 20)
	if err != nil {
		t.Fatalf("query binding users by keyword: %v", err)
	}
	if filtered.Total != 1 || len(filtered.Data) != 1 || filtered.Data[0].UserId != 1 {
		t.Fatalf("unexpected keyword result: %+v", filtered)
	}
}

func TestOrgEmployeeRepositorySeparatesSourceAndPlatformUpdates(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.OrgEmployee{})
	repo := NewOrgEmployeeRepositoryImpl(&database.PrimaryDB{DB: db})
	boundUserID := 101
	employee := model.OrgEmployee{
		Basic:            model.Basic{Id: 1},
		SourceSystemCode: "authority",
		SourceId:         "employee-source",
		EmployeeNo:       "EMP-001",
		Name:             "Before",
		Mobile:           "13800138000",
		EmploymentStatus: "active",
		SourceVersion:    "version-1",
		SyncStatus:       "synced",
		UserId:           &boundUserID,
		LocalNote:        "platform-note",
		LocalTags:        datatypes.JSON([]byte(`["platform-tag"]`)),
	}
	testutil.MustCreate(t, db, &employee)
	if employee.Id == 0 {
		t.Fatal("employee fixture did not receive an id")
	}

	if err := repo.UpdateSourceFields(db, employee.Id, map[string]any{
		"name":           "After source sync",
		"mobile":         "13900139000",
		"source_version": "version-2",
	}); err != nil {
		t.Fatalf("update source fields: %v", err)
	}
	var afterSource model.OrgEmployee
	if err := db.First(&afterSource, employee.Id).Error; err != nil {
		t.Fatalf("reload source update: %v", err)
	}
	if afterSource.UserId == nil || *afterSource.UserId != boundUserID ||
		afterSource.LocalNote != "platform-note" ||
		string(afterSource.LocalTags) != `["platform-tag"]` {
		t.Fatalf("source update overwrote platform fields: %+v", afterSource)
	}

	newUserID := 202
	if err := repo.UpdatePlatformFields(db, employee.Id, map[string]any{
		"user_id":    newUserID,
		"local_note": "updated platform note",
	}); err != nil {
		t.Fatalf("update platform fields: %v", err)
	}
	var afterPlatform model.OrgEmployee
	if err := db.First(&afterPlatform, employee.Id).Error; err != nil {
		t.Fatalf("reload platform update: %v", err)
	}
	if afterPlatform.Name != "After source sync" ||
		afterPlatform.Mobile != "13900139000" ||
		afterPlatform.SourceVersion != "version-2" {
		t.Fatalf("platform update overwrote source fields: %+v", afterPlatform)
	}
	if afterPlatform.UserId == nil || *afterPlatform.UserId != newUserID ||
		afterPlatform.LocalNote != "updated platform note" {
		t.Fatalf("platform fields were not updated: %+v", afterPlatform)
	}

	if err := repo.UpdateSourceFields(db, employee.Id, map[string]any{"user_id": 303}); !errors.Is(err, repository.ErrOrganizationFieldBoundary) {
		t.Fatalf("expected source boundary error, got %v", err)
	}
	if err := repo.UpdatePlatformFields(db, employee.Id, map[string]any{"name": "forbidden"}); !errors.Is(err, repository.ErrOrganizationFieldBoundary) {
		t.Fatalf("expected platform boundary error, got %v", err)
	}
}

func TestOrgRepositoryPropagatesCancelledContext(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.OrgLegalEntity{})
	repo := NewOrgLegalEntityRepositoryImpl(&database.PrimaryDB{DB: db})

	requestContext, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.Query(
		requestContext,
		&request.OrgLegalEntityQueryReq{},
		model.SysTable{TableCode: "org_legal_entity"},
		repository.OrgLegalEntityReadScope{AsOf: model.Now()},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation propagation, got %v", err)
	}
}

func newLegalEntityFixture(sourceID, code, name, entityType, status string) model.OrgLegalEntity {
	return model.OrgLegalEntity{
		SourceSystemCode: "authority",
		SourceId:         sourceID,
		SourceCode:       "source-" + code,
		Code:             code,
		Name:             name,
		EntityType:       entityType,
		Status:           status,
		SyncStatus:       "synced",
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func organizationTestTable(tableCode string, fieldTypes map[string]enum.SysTableFieldType) model.SysTable {
	fields := make([]model.SysTableField, 0, len(fieldTypes))
	for code, fieldType := range fieldTypes {
		fields = append(fields, model.SysTableField{
			FieldCode:        code,
			FieldType:        fieldType,
			IsAdvancedSearch: true,
			IsSort:           true,
		})
	}
	return model.SysTable{TableCode: tableCode, TableFields: fields}
}
