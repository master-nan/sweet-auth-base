package impl

import (
	"backend/dto/request"
	"backend/enum"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestBasicRepositoryReadOptionsApplyToFind(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:basic_repo_options?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}
	if err := db.Create(&basicRepositoryFindByFieldFixture{
		Basic:   model.Basic{Id: 1, State: true},
		Name:    "alpha",
		ScopeID: 7,
	}).Error; err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{}).
		WithSelect("id", "name")
	got, err := repo.FindByField("name", "alpha")
	if err != nil {
		t.Fatalf("find by field: %v", err)
	}
	if got.Id != 1 || got.Name != "alpha" {
		t.Fatalf("expected selected fields, got %+v", got)
	}
	if got.ScopeID != 0 {
		t.Fatalf("expected unselected scope_id to remain zero, got %+v", got)
	}
}

func TestBasicRepositoryWithOptionsReturnsIndependentClone(t *testing.T) {
	repo := NewBasicRepositoryImpl(&gorm.DB{}, &basicRepositoryFindByFieldFixture{})
	repo.selects = make([]string, 1, 4)
	repo.selects[0] = "id"

	first := repo.WithSelect("name").(*BasicRepositoryImpl[basicRepositoryFindByFieldFixture])
	second := repo.WithSelect("scope_id").(*BasicRepositoryImpl[basicRepositoryFindByFieldFixture])

	if first.selects[1] != "name" {
		t.Fatalf("first clone was mutated by sibling clone: %v", first.selects)
	}
	if second.selects[1] != "scope_id" {
		t.Fatalf("unexpected second clone selects: %v", second.selects)
	}
	if len(repo.selects) != 1 || repo.selects[0] != "id" {
		t.Fatalf("base repository options were mutated: %v", repo.selects)
	}
}

func TestBasicRepositoryWithUnscopedAppliesToPagination(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:basic_repo_unscoped?mode=memory&cache=shared"), &gorm.Config{})
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
		{Basic: model.Basic{Id: 1, State: true}, Name: "active"},
		{Basic: model.Basic{Id: 2, State: true}, Name: "deleted"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed fixture: %v", err)
	}
	if err := db.Delete(&rows[1]).Error; err != nil {
		t.Fatalf("soft delete fixture: %v", err)
	}

	var active []basicRepositoryFindByFieldFixture
	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	activeTotal, err := repo.PaginateAndCountAsync(
		&request.Basic{Page: 1, Num: 10},
		&active,
		model.SysTable{},
	)
	if err != nil {
		t.Fatalf("paginate active rows: %v", err)
	}
	if activeTotal != 1 || len(active) != 1 {
		t.Fatalf("expected one active row, total=%d rows=%+v", activeTotal, active)
	}

	var all []basicRepositoryFindByFieldFixture
	allTotal, err := repo.WithUnscoped().PaginateAndCountAsync(
		&request.Basic{Page: 1, Num: 10},
		&all,
		model.SysTable{},
	)
	if err != nil {
		t.Fatalf("paginate unscoped rows: %v", err)
	}
	if allTotal != 2 || len(all) != 2 {
		t.Fatalf("expected active and deleted rows, total=%d rows=%+v", allTotal, all)
	}
}

func TestBasicRepositoryPropagatesContextCancellation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:basic_repo_context?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}

	requestContext, cancel := context.WithCancel(context.Background())
	cancel()
	ginContext, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine.ContextWithFallback = true
	ginContext.Request = httptest.NewRequest("GET", "/", nil).WithContext(requestContext)

	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{}).WithContext(ginContext)
	_, err = repo.FindByField("name", "alpha")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestBasicRepositoryReturnsStableValidationErrors(t *testing.T) {
	repo := NewBasicRepositoryImpl(&gorm.DB{}, &basicRepositoryFindByFieldFixture{})

	if _, err := repo.FindByField("name; DROP TABLE fixture", "alpha"); !errors.Is(err, repository.ErrInvalidField) {
		t.Fatalf("expected invalid field error, got %v", err)
	}
	if _, err := repo.FindListByFieldIn("id", 1); !errors.Is(err, repository.ErrInvalidFieldValues) {
		t.Fatalf("expected invalid field values error, got %v", err)
	}
}

func TestBasicRepositoryPreservesGormErrors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:basic_repo_errors?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}

	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	_, err = repo.FindByField("name", "missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm record-not-found error, got %v", err)
	}
}
