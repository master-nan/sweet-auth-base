package impl

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"backend/enum"
	"backend/internal/database"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGeneralizationPermissionRepositoryAppliesIdAndPermissionAtomically(t *testing.T) {
	db := generalizationPermissionRepositoryDB(t)
	table := generalizationPermissionRepositoryTable()
	repo := NewGeneralizationRepositoryImpl(&database.PrimaryDB{DB: db})

	detailPermission := generalizationRepositoryPermission(t, model.DataPermissionOperationDetail, []int64{11})
	allowed, err := repo.GetByIdWithPermission(table, 1, detailPermission)
	if err != nil || allowed["name"] != "allowed" {
		t.Fatalf("allowed detail = %+v, err=%v", allowed, err)
	}
	_, deniedErr := repo.GetByIdWithPermission(table, 2, detailPermission)
	_, missingErr := repo.GetByIdWithPermission(table, 999, detailPermission)
	if !errors.Is(deniedErr, myerrors.ErrDataNotFound) || !errors.Is(missingErr, myerrors.ErrDataNotFound) {
		t.Fatalf("denied and missing details must share not-found semantics: denied=%v missing=%v", deniedErr, missingErr)
	}

	updatePermission := generalizationRepositoryPermission(t, model.DataPermissionOperationUpdate, []int64{11})
	updated, err := repo.UpdateWithPermission(table, 1, map[string]interface{}{"name": "changed"}, updatePermission)
	if err != nil || !updated {
		t.Fatalf("allowed update failed: updated=%v err=%v", updated, err)
	}
	updated, err = repo.UpdateWithPermission(table, 2, map[string]interface{}{"name": "leaked"}, updatePermission)
	if err != nil || updated {
		t.Fatalf("denied update executed: updated=%v err=%v", updated, err)
	}
	assertGeneralizationPermissionRow(t, db, 1, "changed", true)
	assertGeneralizationPermissionRow(t, db, 2, "denied", true)

	deletePermission := generalizationRepositoryPermission(t, model.DataPermissionOperationDelete, []int64{11})
	deleted, err := repo.SoftDeleteWithPermission(
		table,
		1,
		map[string]interface{}{"gmt_delete": model.Now()},
		deletePermission,
	)
	if err != nil || !deleted {
		t.Fatalf("allowed delete failed: deleted=%v err=%v", deleted, err)
	}
	deleted, err = repo.SoftDeleteWithPermission(
		table,
		2,
		map[string]interface{}{"gmt_delete": model.Now()},
		deletePermission,
	)
	if err != nil || deleted {
		t.Fatalf("denied delete executed: deleted=%v err=%v", deleted, err)
	}
	assertGeneralizationPermissionRow(t, db, 1, "changed", false)
	assertGeneralizationPermissionRow(t, db, 2, "denied", true)
}

func TestGeneralizationPermissionRepositoryDoesNotExecuteInvalidOrDenyAllFilters(t *testing.T) {
	db := generalizationPermissionRepositoryDB(t)
	table := generalizationPermissionRepositoryTable()
	repo := NewGeneralizationRepositoryImpl(&database.PrimaryDB{DB: db})

	denyExecution := generalizationRepositoryExecution(t, model.DataPermissionOperationUpdate, datapermission.DataScopeDecisionNone, nil)
	updated, err := repo.UpdateWithPermission(
		table,
		1,
		map[string]interface{}{"name": "leaked"},
		repository.GeneralizationPermission{
			AdapterExecution: &denyExecution,
		},
	)
	if err != nil || updated {
		t.Fatalf("deny_all update = %v, err=%v", updated, err)
	}
	assertGeneralizationPermissionRow(t, db, 1, "allowed", true)

	notApplicable := generalizationRepositoryExecution(t, model.DataPermissionOperationDetail, datapermission.DataScopeDecisionNotApplicable, nil)
	if _, err = repo.GetByIdWithPermission(
		table,
		1,
		repository.GeneralizationPermission{
			AdapterExecution: &notApplicable,
		},
	); err != nil {
		t.Fatalf("not_applicable detail query: %v", err)
	}
}

func TestGeneralizationPermissionRepositoryBatchDeleteRollsBackPartialAuthorization(t *testing.T) {
	db := generalizationPermissionRepositoryDB(t)
	table := generalizationPermissionRepositoryTable()
	repo := NewGeneralizationRepositoryImpl(&database.PrimaryDB{DB: db})
	permission := generalizationRepositoryPermission(t, model.DataPermissionOperationDelete, []int64{11})

	deleted, err := repo.BatchSoftDeleteWithPermission(
		table,
		[]int{1, 2},
		map[string]interface{}{"gmt_delete": model.Now()},
		permission,
	)
	if err != nil || deleted {
		t.Fatalf("partially authorized batch delete = %v, err=%v", deleted, err)
	}
	assertGeneralizationPermissionRow(t, db, 1, "allowed", true)
	assertGeneralizationPermissionRow(t, db, 2, "denied", true)

	deleted, err = repo.BatchSoftDeleteWithPermission(
		table,
		[]int{1},
		map[string]interface{}{"gmt_delete": model.Now()},
		permission,
	)
	if err != nil || !deleted {
		t.Fatalf("fully authorized batch delete = %v, err=%v", deleted, err)
	}
	assertGeneralizationPermissionRow(t, db, 1, "allowed", false)
}

func generalizationPermissionRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err = db.Exec(`CREATE TABLE permission_record (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		owner_org_id INTEGER NOT NULL,
		gmt_delete DATETIME NULL
	)`).Error; err != nil {
		t.Fatalf("create permission record table: %v", err)
	}
	if err = db.Exec(`INSERT INTO permission_record (id, name, owner_org_id) VALUES
		(1, 'allowed', 11),
		(2, 'denied', 12)`).Error; err != nil {
		t.Fatalf("seed permission records: %v", err)
	}
	return db
}

func generalizationPermissionRepositoryTable() model.SysTable {
	return model.SysTable{
		Basic:     model.Basic{Id: 9101, State: true},
		TableCode: "permission_record",
		TableFields: []model.SysTableField{
			{Basic: model.Basic{Id: 1, State: true}, TableId: 9101, FieldCode: "id", FieldType: enum.BigIntFieldType, IsListShow: true, IsPrimaryKey: true},
			{Basic: model.Basic{Id: 2, State: true}, TableId: 9101, FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true, IsUpdateShow: true},
			{Basic: model.Basic{Id: 501, State: true}, TableId: 9101, FieldCode: "owner_org_id", FieldType: enum.BigIntFieldType, IsListShow: true, IsAdvancedSearch: true},
			{Basic: model.Basic{Id: 3, State: true}, TableId: 9101, FieldCode: "gmt_delete", FieldType: enum.DatetimeFieldType},
		},
	}
}

func generalizationRepositoryPermission(
	t *testing.T,
	operation string,
	orgIds []int64,
) repository.GeneralizationPermission {
	t.Helper()
	values := make([]any, len(orgIds))
	for index, orgId := range orgIds {
		values[index] = orgId
	}
	condition, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: "owner_org",
		DimensionId:   201,
		Operator:      datapermission.DataScopeOperatorIn,
		ValueType:     datapermission.DataScopeValueTypeBigint,
		Values:        values,
	})
	if err != nil {
		t.Fatalf("create permission condition: %v", err)
	}
	execution := generalizationRepositoryExecution(
		t,
		operation,
		datapermission.DataScopeDecisionFiltered,
		[]datapermission.DataScopeCondition{condition},
	)
	return repository.GeneralizationPermission{
		AdapterExecution: &execution,
	}
}

func generalizationRepositoryExecution(
	t *testing.T,
	operation string,
	decision datapermission.DataScopeDecision,
	conditions []datapermission.DataScopeCondition,
) datapermission.AdapterExecution {
	t.Helper()
	resource, err := datapermission.NewAdapterResourceContext(datapermission.AdapterResourceContextInput{
		ResourceCode: "permission_record",
		Operation:    operation,
		AdapterCode:  "metadata_filter",
		TableId:      9101,
	})
	if err != nil {
		t.Fatalf("create adapter resource: %v", err)
	}
	groups := make([]datapermission.DataScopeConditionGroup, 0, 1)
	if len(conditions) > 0 {
		group, groupErr := datapermission.NewDataScopeConditionGroup(conditions)
		if groupErr != nil {
			t.Fatalf("create condition group: %v", groupErr)
		}
		groups = append(groups, group)
	}
	result, err := datapermission.NewDataScopeResult(datapermission.DataScopeResultInput{
		ResourceCode:    resource.ResourceCode(),
		Operation:       operation,
		Decision:        decision,
		ConditionGroups: groups,
	})
	if err != nil {
		t.Fatalf("create scope result: %v", err)
	}
	definition, err := datapermission.NewAdapterOwnershipDefinition(datapermission.AdapterOwnershipDefinitionInput{
		OwnershipCode: "owner_org",
		DimensionId:   201,
		BindingType:   datapermission.AdapterBindingTypeMetadataField,
		TableFieldId:  501,
		ValueType:     datapermission.DataScopeValueTypeBigint,
	})
	if err != nil {
		t.Fatalf("create ownership definition: %v", err)
	}
	input, err := datapermission.NewAdapterInput(resource, result, []datapermission.AdapterOwnershipDefinition{definition})
	if err != nil {
		t.Fatalf("create adapter input: %v", err)
	}
	execution, err := datapermission.BuildAdapterExecution(input)
	if err != nil {
		t.Fatalf("build adapter execution: %v", err)
	}
	return execution
}

func assertGeneralizationPermissionRow(
	t *testing.T,
	db *gorm.DB,
	id int,
	wantName string,
	wantActive bool,
) {
	t.Helper()
	var row struct {
		Name      string
		GmtDelete sql.NullTime `gorm:"column:gmt_delete"`
	}
	if err := db.Table("permission_record").Select("name, gmt_delete").Where("id = ?", id).Take(&row).Error; err != nil {
		t.Fatalf("read row %d: %v", id, err)
	}
	if row.Name != wantName || !row.GmtDelete.Valid != wantActive {
		t.Fatalf("row %d = name:%q delete:%v, want name:%q active:%v", id, row.Name, row.GmtDelete, wantName, wantActive)
	}
}
