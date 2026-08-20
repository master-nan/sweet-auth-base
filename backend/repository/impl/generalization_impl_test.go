package impl

import (
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestGeneralizationRepositoryRowExistsIgnoresSoftDeletedRows(t *testing.T) {
	repo, db := newGeneralizationRepositoryForTest(t)
	table := softDeleteSmokeTable()

	if err := db.Exec("INSERT INTO smk_generalization_repo (id, name, gmt_delete) VALUES (1, 'active', NULL), (2, 'deleted', ?)", time.Now()).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	active, err := repo.RowExists(context.Background(), table, 1)
	if err != nil {
		t.Fatalf("row exists active: %v", err)
	}
	if !active {
		t.Fatal("expected active row to exist")
	}

	deleted, err := repo.RowExists(context.Background(), table, 2)
	if err != nil {
		t.Fatalf("row exists deleted: %v", err)
	}
	if deleted {
		t.Fatal("soft-deleted row should not be treated as writable")
	}
}

func TestGeneralizationRepositoryUpdateDoesNotTouchSoftDeletedRows(t *testing.T) {
	repo, db := newGeneralizationRepositoryForTest(t)
	table := softDeleteSmokeTable()

	if err := db.Exec("INSERT INTO smk_generalization_repo (id, name, gmt_delete) VALUES (1, 'deleted', ?)", time.Now()).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}
	if err := repo.Update(context.Background(), table, 1, map[string]interface{}{"name": "updated"}); err != nil {
		t.Fatalf("update soft-deleted row: %v", err)
	}

	var name string
	if err := db.Table("smk_generalization_repo").Select("name").Where("id = ?", 1).Take(&name).Error; err != nil {
		t.Fatalf("read row: %v", err)
	}
	if name != "deleted" {
		t.Fatalf("soft-deleted row was updated, got name=%s", name)
	}
}

func newGeneralizationRepositoryForTest(t *testing.T) (*GeneralizationRepositoryImpl, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(t)
	if err := db.Exec("CREATE TABLE smk_generalization_repo (id INTEGER PRIMARY KEY, name TEXT, gmt_delete DATETIME)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	return NewGeneralizationRepositoryImpl(&database.PrimaryDB{DB: db}), db
}

func softDeleteSmokeTable() model.SysTable {
	return model.SysTable{
		TableCode: "smk_generalization_repo",
		TableFields: []model.SysTableField{
			{FieldCode: "id"},
			{FieldCode: "name"},
			{FieldCode: "gmt_delete"},
		},
	}
}
