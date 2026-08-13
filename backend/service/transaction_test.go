package service

import (
	testutil "backend/internal/test"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"
)

type transactionFixture struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:64;uniqueIndex"`
}

func TestRunInTransactionCommitsOnSuccess(t *testing.T) {
	db := testutil.OpenSQLite(t, &transactionFixture{})

	err := RunInTransaction(context.Background(), db, func(tx *gorm.DB) error {
		return tx.Create(&transactionFixture{Name: "committed"}).Error
	})
	if err != nil {
		t.Fatalf("run transaction: %v", err)
	}

	assertTransactionFixtureNames(t, db, "committed")
}

func TestRunInTransactionRollsBackAndPropagatesError(t *testing.T) {
	db := testutil.OpenSQLite(t, &transactionFixture{})
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

func TestRunInTransactionRollsBackAndPropagatesPanic(t *testing.T) {
	db := testutil.OpenSQLite(t, &transactionFixture{})
	const panicValue = "transaction panic"

	func() {
		defer func() {
			if recovered := recover(); recovered != panicValue {
				t.Fatalf("expected panic %q, got %#v", panicValue, recovered)
			}
		}()
		_ = RunInTransaction(context.Background(), db, func(tx *gorm.DB) error {
			if err := tx.Create(&transactionFixture{Name: "rolled-back-panic"}).Error; err != nil {
				return err
			}
			panic(panicValue)
		})
	}()

	assertTransactionFixtureNames(t, db)
}

func TestRunInTransactionNestedSavepoint(t *testing.T) {
	db := testutil.OpenSQLite(t, &transactionFixture{})
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
	db := testutil.OpenSQLite(t, &transactionFixture{})
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
	db := testutil.OpenSQLite(t, &transactionFixture{})
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
