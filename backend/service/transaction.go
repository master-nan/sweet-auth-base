package service

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var (
	ErrTransactionContextRequired  = errors.New("service transaction context is required")
	ErrTransactionDatabaseRequired = errors.New("service transaction database is required")
	ErrTransactionCallbackRequired = errors.New("service transaction callback is required")
)

// RunInTransaction is the Service-layer transaction entry for new code.
// Passing an existing transaction intentionally creates a nested savepoint.
func RunInTransaction(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if ctx == nil {
		return ErrTransactionContextRequired
	}
	if db == nil {
		return ErrTransactionDatabaseRequired
	}
	if fn == nil {
		return ErrTransactionCallbackRequired
	}

	return db.WithContext(ctx).Transaction(fn)
}
