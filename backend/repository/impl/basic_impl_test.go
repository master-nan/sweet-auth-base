package impl

import (
	"backend/dto/request"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
)

type basicRepositoryFindByFieldFixture struct {
	model.Basic
	Name     string `gorm:"size:64"`
	ScopeID  int
	Revision int
}

func TestBasicRepositoryPaginateAndCountAsyncReturnsDataQueryError(t *testing.T) {
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
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
	_, err := repo.PaginateAndCountAsync(&request.Basic{Page: 1, Num: 10}, &rows, model.SysTable{})
	if err == nil {
		t.Fatal("expected data query error")
	}
}

func TestBasicRepositoryPaginateAndCountAsyncWaitsForBothQueriesBeforeReturning(t *testing.T) {
	db := testutil.OpenSQLite(t, &basicRepositoryFindByFieldFixture{})
	if err := db.Create(&basicRepositoryFindByFieldFixture{
		Basic: model.Basic{Id: 1, State: true}, Name: "alpha",
	}).Error; err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	countErr := errors.New("count query failed")
	countFailed := make(chan struct{})
	dataStarted := make(chan struct{})
	releaseData := make(chan struct{})
	var countOnce, dataOnce sync.Once
	callbackName := "test:wait_for_parallel_page_queries"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		switch tx.Statement.Dest.(type) {
		case *int64:
			tx.AddError(countErr)
			countOnce.Do(func() { close(countFailed) })
		case *[]basicRepositoryFindByFieldFixture:
			dataOnce.Do(func() { close(dataStarted) })
			<-releaseData
		}
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	result := make(chan error, 1)
	go func() {
		var rows []basicRepositoryFindByFieldFixture
		_, err := repo.PaginateAndCountAsync(&request.Basic{Page: 1, Num: 10}, &rows, model.SysTable{})
		result <- err
	}()

	select {
	case <-dataStarted:
	case <-time.After(time.Second):
		t.Fatal("data query did not start")
	}
	select {
	case <-countFailed:
	case <-time.After(time.Second):
		t.Fatal("count query did not fail")
	}
	select {
	case err := <-result:
		t.Fatalf("pagination returned before the data query completed: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(releaseData)
	select {
	case err := <-result:
		if !errors.Is(err, countErr) {
			t.Fatalf("expected count error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("pagination did not return after both queries completed")
	}
}

func TestBasicRepositoryPaginateAndCountAsyncUsesTransactionSequentially(t *testing.T) {
	db := testutil.OpenSQLite(t, &basicRepositoryFindByFieldFixture{})
	for i := 1; i <= 3; i++ {
		if err := db.Create(&basicRepositoryFindByFieldFixture{Basic: model.Basic{Id: i}, Name: fmt.Sprintf("item-%d", i)}).Error; err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		repo := NewBasicRepositoryImpl(tx, &basicRepositoryFindByFieldFixture{})
		var rows []basicRepositoryFindByFieldFixture
		total, err := repo.PaginateAndCountAsync(&request.Basic{Page: 1, Num: 2}, &rows, model.SysTable{})
		if err != nil {
			return err
		}
		if total != 3 || len(rows) != 2 {
			t.Fatalf("transaction page total=%d rows=%d", total, len(rows))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("transaction pagination: %v", err)
	}
}

func TestBasicRepositoryPaginateAndCountQueryPreservesScopedFilter(t *testing.T) {
	db := testutil.OpenSQLite(t, &basicRepositoryFindByFieldFixture{})
	for id, name := range []string{"allowed-1", "denied", "allowed-2"} {
		if err := db.Create(&basicRepositoryFindByFieldFixture{
			Basic: model.Basic{Id: id + 1, State: true}, Name: name,
		}).Error; err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}
	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	query := db.Model(&basicRepositoryFindByFieldFixture{}).
		Where("name LIKE ?", "allowed-%").Order("id ASC").Limit(1).Offset(1)
	var rows []basicRepositoryFindByFieldFixture
	total, err := repo.PaginateAndCountQuery(query, &rows)
	if err != nil || total != 2 || len(rows) != 1 || rows[0].Name != "allowed-2" {
		t.Fatalf("scoped page total=%d rows=%+v err=%v", total, rows, err)
	}
}

func TestBasicRepositoryFindByFieldScansIntoEntity(t *testing.T) {
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
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

func TestBasicRepositoryTransactionReadAndFieldUpdate(t *testing.T) {
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
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
	byID, err := repo.FindByIdWithDB(db, 1)
	if err != nil || byID.Name != "alpha" {
		t.Fatalf("find by ID with DB: value=%+v err=%v", byID, err)
	}
	byName, err := repo.FindByFieldWithDB(db, "name", "alpha")
	if err != nil || byName.Id != 1 {
		t.Fatalf("find by field with DB: value=%+v err=%v", byName, err)
	}
	listByName, err := repo.FindListByFieldWithDB(db, "name", "alpha")
	if err != nil || len(listByName) != 1 || listByName[0].Id != 1 {
		t.Fatalf("find list by field with DB: value=%+v err=%v", listByName, err)
	}
	countByName, err := repo.CountByField(db, "name", "alpha")
	if err != nil || countByName != 1 {
		t.Fatalf("count by field: count=%d err=%v", countByName, err)
	}
	updated, err := repo.UpdateFields(db, 1, map[string]any{"name": "beta"})
	if err != nil || !updated {
		t.Fatalf("update fields: updated=%v err=%v", updated, err)
	}
	if got, err := repo.FindById(1); err != nil || got.Name != "beta" {
		t.Fatalf("find updated fixture: value=%+v err=%v", got, err)
	}
	var nilContext context.Context
	if got, err := repo.WithContext(nilContext).FindById(1); err != nil || got.Name != "beta" {
		t.Fatalf("typed nil context should be ignored: value=%+v err=%v", got, err)
	}
}

func TestBasicRepositoryUpdateOmitsEmbeddedBasicField(t *testing.T) {
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
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

func TestBasicRepositoryReadOptionsApplyToFind(t *testing.T) {
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
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
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
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
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}

	requestContext, cancel := context.WithCancel(context.Background())
	cancel()
	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{}).WithContext(requestContext)
	_, err := repo.FindByField("name", "alpha")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestBasicRepositoryFindByIdForUpdateAndRevisionUpdate(t *testing.T) {
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}
	row := basicRepositoryFindByFieldFixture{
		Basic: model.Basic{Id: 1, State: true},
		Name:  "before", Revision: 1,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	err := db.Transaction(func(tx *gorm.DB) error {
		locked, lockErr := repo.FindByIdForUpdate(tx, row.Id)
		if lockErr != nil {
			return lockErr
		}
		if locked.Name != "before" || locked.Revision != 1 {
			t.Fatalf("unexpected locked row: %+v", locked)
		}
		updated, updateErr := repo.UpdateFieldsByRevision(tx, row.Id, 1, map[string]any{
			"name": "after", "revision": 2,
		})
		if updateErr != nil {
			return updateErr
		}
		if !updated {
			t.Fatal("expected revision update to affect one row")
		}
		stale, staleErr := repo.UpdateFieldsByRevision(tx, row.Id, 1, map[string]any{"name": "stale"})
		if staleErr != nil {
			return staleErr
		}
		if stale {
			t.Fatal("stale revision must not update the row")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
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
	if _, err := repo.CountByField(&gorm.DB{}, "name; DROP TABLE fixture", "alpha"); !errors.Is(err, repository.ErrInvalidField) {
		t.Fatalf("expected count invalid field error, got %v", err)
	}
	if _, err := repo.FindListByFieldWithDB(&gorm.DB{}, "name; DROP TABLE fixture", "alpha"); !errors.Is(err, repository.ErrInvalidField) {
		t.Fatalf("expected list invalid field error, got %v", err)
	}
}

func TestBasicRepositoryPreservesGormErrors(t *testing.T) {
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
	if err := db.AutoMigrate(&basicRepositoryFindByFieldFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}

	repo := NewBasicRepositoryImpl(db, &basicRepositoryFindByFieldFixture{})
	_, err := repo.FindByField("name", "missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm record-not-found error, got %v", err)
	}
}
