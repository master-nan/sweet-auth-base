package service

import (
	"testing"

	"gorm.io/gorm"
)

func openPostgresTestDB(t *testing.T, dialector gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error) {
	t.Helper()
	db, err := gorm.Open(dialector, opts...)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db, nil
}
