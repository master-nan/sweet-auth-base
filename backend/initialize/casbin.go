/**
 * @Author: Nan
 * @Date: 2024/10/11 11:49
 */

package initialize

import (
	"backend/internal/database"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

func InitCasbin(PrimaryDB *database.PrimaryDB) (*casbin.SyncedEnforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(PrimaryDB.DB) // 使用GORM适配器
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
