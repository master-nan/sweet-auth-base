package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type transactionFixture struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:64;uniqueIndex"`
}

func TestRunInTransactionCommitsOnSuccess(t *testing.T) {
	db := newTransactionTestDB(t)

	err := RunInTransaction(context.Background(), db, func(tx *gorm.DB) error {
		return tx.Create(&transactionFixture{Name: "committed"}).Error
	})
	if err != nil {
		t.Fatalf("run transaction: %v", err)
	}

	assertTransactionFixtureNames(t, db, "committed")
}

func TestRunInTransactionRollsBackAndPropagatesError(t *testing.T) {
	db := newTransactionTestDB(t)
	expectedErr := errors.New("write rejected")

	err := RunInTransaction(context.Background(), db, func(tx *gorm.DB) error {
		if err := tx.Create(&transactionFixture{Name: "rolled-back"}).Error; err != nil {
			return err
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected original error, got %v", err)
	}

	assertTransactionFixtureNames(t, db)
}

func TestRunInTransactionNestedSavepoint(t *testing.T) {
	db := newTransactionTestDB(t)
	nestedErr := errors.New("nested write rejected")

	err := RunInTransaction(context.Background(), db, func(tx *gorm.DB) error {
		if err := tx.Create(&transactionFixture{Name: "outer-before"}).Error; err != nil {
			return err
		}

		err := RunInTransaction(context.Background(), tx, func(nestedTx *gorm.DB) error {
			if err := nestedTx.Create(&transactionFixture{Name: "inner"}).Error; err != nil {
				return err
			}
			return nestedErr
		})
		if !errors.Is(err, nestedErr) {
			return fmt.Errorf("expected nested error: %w", err)
		}

		return tx.Create(&transactionFixture{Name: "outer-after"}).Error
	})
	if err != nil {
		t.Fatalf("run outer transaction: %v", err)
	}

	assertTransactionFixtureNames(t, db, "outer-after", "outer-before")
}

func TestRunInTransactionNestedErrorRollsBackOuterWhenPropagated(t *testing.T) {
	db := newTransactionTestDB(t)
	expectedErr := errors.New("nested failure")

	err := RunInTransaction(context.Background(), db, func(tx *gorm.DB) error {
		if err := tx.Create(&transactionFixture{Name: "outer"}).Error; err != nil {
			return err
		}
		return RunInTransaction(context.Background(), tx, func(nestedTx *gorm.DB) error {
			if err := nestedTx.Create(&transactionFixture{Name: "inner"}).Error; err != nil {
				return err
			}
			return expectedErr
		})
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected nested error to propagate, got %v", err)
	}

	assertTransactionFixtureNames(t, db)
}

func TestRunInTransactionValidatesRequiredInputs(t *testing.T) {
	db := newTransactionTestDB(t)
	callback := func(*gorm.DB) error { return nil }

	tests := []struct {
		name string
		ctx  context.Context
		db   *gorm.DB
		fn   func(*gorm.DB) error
		want error
	}{
		{
			name: "context",
			db:   db,
			fn:   callback,
			want: ErrTransactionContextRequired,
		},
		{
			name: "database",
			ctx:  context.Background(),
			fn:   callback,
			want: ErrTransactionDatabaseRequired,
		},
		{
			name: "callback",
			ctx:  context.Background(),
			db:   db,
			want: ErrTransactionCallbackRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RunInTransaction(tt.ctx, tt.db, tt.fn); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func newTransactionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(strings.ToLower(t.Name()))
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", databaseName)),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sqlite handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(&transactionFixture{}); err != nil {
		t.Fatalf("migrate transaction fixture: %v", err)
	}
	return db
}

func assertTransactionFixtureNames(t *testing.T, db *gorm.DB, want ...string) {
	t.Helper()

	var got []transactionFixture
	if err := db.Order("name ASC").Find(&got).Error; err != nil {
		t.Fatalf("query transaction fixtures: %v", err)
	}
	gotNames := make([]string, 0, len(got))
	for _, item := range got {
		gotNames = append(gotNames, item.Name)
	}
	if strings.Join(gotNames, ",") != strings.Join(want, ",") {
		t.Fatalf("expected rows %v, got %v", want, gotNames)
	}
}
