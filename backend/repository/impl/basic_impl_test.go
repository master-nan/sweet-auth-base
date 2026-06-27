package impl

import (
	"backend/dto/request"
	"backend/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type basicRepositoryFindByFieldFixture struct {
	model.Basic
	Name string `gorm:"size:64"`
}

func TestBasicRepositoryPaginateAndCountAsyncReturnsDataQueryError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
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
