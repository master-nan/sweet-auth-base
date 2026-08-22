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

// RunInTransaction 是Application Service的统一事务入口；
// 传入已有事务时会有意创建嵌套保存点，调用方必须显式传递请求Context。
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
