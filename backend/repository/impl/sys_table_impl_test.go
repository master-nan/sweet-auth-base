package impl

import (
	"backend/internal/database"
	myerrors "backend/internal/errors"
	testutil "backend/internal/test"
	"errors"
	"testing"
)

func TestQuoteSQLIdentifierEscapesDoubleQuotes(t *testing.T) {
	got := quoteSQLIdentifier(`idx"name`)
	if got != `"idx""name"` {
		t.Fatalf("unexpected quoted identifier: %s", got)
	}
}

func TestQuoteSQLIdentifierListSkipsEmptyParts(t *testing.T) {
	got := quoteSQLIdentifierList(" name, ,tenant_id ")
	if got != `"name","tenant_id"` {
		t.Fatalf("unexpected quoted identifier list: %s", got)
	}
}

func TestSysTableRepositoryDoesNotReplaceExistingPhysicalObject(t *testing.T) {
	db := testutil.OpenSQLite(t)
	if err := db.Exec(`CREATE TABLE existing_business_data (id integer)`).Error; err != nil {
		t.Fatalf("create existing table: %v", err)
	}
	repository := NewSysTableRepositoryImpl(&database.PrimaryDB{DB: db})
	if !repository.HasPhysicalTable(db, "existing_business_data") {
		t.Fatal("existing physical table was not detected")
	}
	if err := repository.CreateTable(db, "existing_business_data", &struct{ ID int }{}); !errors.Is(err, myerrors.ErrTableExist) {
		t.Fatalf("existing physical table replacement error = %v", err)
	}
	if !db.Migrator().HasColumn("existing_business_data", "id") {
		t.Fatal("existing physical table was altered")
	}
}

func TestSysTableRepositoryRejectsExecutableViewDefinition(t *testing.T) {
	db := testutil.OpenSQLite(t)
	repository := NewSysTableRepositoryImpl(&database.PrimaryDB{DB: db})
	if err := repository.CreateView(db, "unsafe_view", "SELECT 1; DROP TABLE sys_user"); err == nil {
		t.Fatal("executable view definition accepted")
	}
}
