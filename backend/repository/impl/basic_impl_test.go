package impl

import (
	"backend/dto/request"
	"backend/enum"
	"backend/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type basicRepositoryFindByFieldFixture struct {
	model.Basic
	Name    string `gorm:"size:64"`
	ScopeID int
}

func TestBasicRepositoryPaginateAndCountAsyncReturnsDataQueryError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:basic_repo_error?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}
	if err := db.Create(&basicRepositoryFindByFieldFixture{
		Basic: model.Basic{Id: 1, State: true},
		Name:  "alpha",
	}).Error; err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	var rows []basicRepositoryFindByFieldFixture
	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{}).WithSelect("missing_column")
	_, err = repo.PaginateAndCountAsync(&request.Basic{Page: 1, Num: 10}, &rows, model.SysTable{})
	if err == nil {
		t.Fatal("expected data query error")
	}
}

func TestBasicRepositoryFindByFieldScansIntoEntity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:basic_repo_find?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}
	if err := db.Create(&basicRepositoryFindByFieldFixture{
		Basic: model.Basic{Id: 1, State: true},
		Name:  "alpha",
	}).Error; err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	got, err := repo.FindByField("name", "alpha")
	if err != nil {
		t.Fatalf("find by field: %v", err)
	}
	if got.Id != 1 || got.Name != "alpha" {
		t.Fatalf("unexpected entity: %+v", got)
	}
}

func TestBasicRepositoryUpdateOmitsEmbeddedBasicField(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:basic_repo_update?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}
	if err := db.Create(&basicRepositoryFindByFieldFixture{
		Basic: model.Basic{Id: 1, State: true},
		Name:  "alpha",
	}).Error; err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	if err := repo.Update(db, &basicRepositoryFindByFieldFixture{
		Basic: model.Basic{Id: 1, State: true},
		Name:  "beta",
	}, 1); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	got, err := repo.FindByField("id", 1)
	if err != nil {
		t.Fatalf("find updated fixture: %v", err)
	}
	if got.Name != "beta" {
		t.Fatalf("expected updated name, got %+v", got)
	}
}

func TestBasicRepositoryPaginateAndCountAsyncAppliesDataScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:basic_repo_scope?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sqlite db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}
	rows := []basicRepositoryFindByFieldFixture{
		{Basic: model.Basic{Id: 1, State: true}, Name: "alpha", ScopeID: 1},
		{Basic: model.Basic{Id: 2, State: true}, Name: "beta", ScopeID: 2},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	var got []basicRepositoryFindByFieldFixture
	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	total, err := repo.PaginateAndCountAsync(&request.Basic{
		Page: 1,
		Num:  10,
		DataScope: &request.DataScope{Conditions: []request.DataScopeCondition{
			{Field: "scope_id", MatchType: "in", Values: []string{"1"}},
		}},
	}, &got, model.SysTable{TableFields: []model.SysTableField{
		{FieldCode: "scope_id", FieldType: enum.IntFieldType},
	}})
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].Name != "alpha" {
		t.Fatalf("unexpected scoped result total=%d rows=%+v", total, got)
	}
}
