package initialize

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// CloseRuntimeResources 只在HTTP和后台任务停止后关闭外部客户端，
// 重复引用的SQL连接池只关闭一次。
func CloseRuntimeResources(app *App) error {
	if app == nil {
		return nil
	}
	var closeErrors []error
	if app.Redis != nil {
		if err := app.Redis.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("close Redis: %w", err))
		}
	}
	seen := make(map[*gorm.DB]struct{}, len(app.DBs))
	for name, db := range app.DBs {
		if db == nil {
			continue
		}
		if _, exists := seen[db]; exists {
			continue
		}
		seen[db] = struct{}{}
		sqlDB, err := db.DB()
		if err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("resolve database %s pool: %w", name, err))
			continue
		}
		if err := sqlDB.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("close database %s: %w", name, err))
		}
	}
	return errors.Join(closeErrors...)
}
