/**
 * @Author: Nan
 * @Date: 2024/10/11 11:49
 */

package initialize

import (
	"backend/internal/database"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

func InitCasbin(PrimaryDB *database.PrimaryDB) (*casbin.SyncedEnforcer, error) {
	db := PrimaryDB.DB.Session(&gorm.Session{})
	// 正式表结构只由 Migration 管理，避免适配器在启动时重新创建自增序列。
	gormadapter.TurnOffAutoMigrate(db)
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewSyncedEnforcer("casbin_model.conf", adapter)
	if err != nil {
		return nil, err
	}
	err = enforcer.LoadPolicy()
	if err != nil {
		return nil, err
	}
	return enforcer, nil
}
