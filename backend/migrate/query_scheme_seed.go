package main

import (
	"backend/enum"
	"backend/internal/queryscheme"
	"backend/internal/utils"
	"backend/model"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func seedQuerySchemeFoundation(db *gorm.DB, sf *utils.Snowflake) error {
	for _, declaration := range queryscheme.FixedScopeDeclarations() {
		var menu model.SysMenu
		if err := db.Where("name = ? AND gmt_delete IS NULL", declaration.MenuName).First(&menu).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return fmt.Errorf("query scope menu %s: %w", declaration.MenuName, err)
		}
		if !queryScopeMenuSupportsTable(menu, declaration.Config.TableCode) {
			return fmt.Errorf("query scope %s table mismatch", declaration.ScopeCode)
		}
		if err := db.Model(&model.SysMenu{}).Where("id = ?", menu.Id).
			Update("query_scope_code", declaration.ScopeCode).Error; err != nil {
			return err
		}
	}

	var systemMenu model.SysMenu
	if err := db.Where("name = ? AND gmt_delete IS NULL", "system").First(&systemMenu).Error; err != nil {
		return fmt.Errorf("query scheme capability container: %w", err)
	}
	role, err := seedRole(db, sf)
	if err != nil {
		return err
	}
	policies := []struct {
		name   string
		code   string
		path   string
		method string
	}{
		{"新建共享查询方案", "query_scheme_shared_manage_create", "/admin/query-schemes/shared", "POST"},
		{"更新共享查询方案", "query_scheme_shared_manage_update", "/admin/query-schemes/shared/:id", "PUT"},
		{"删除共享查询方案", "query_scheme_shared_manage_delete", "/admin/query-schemes/shared/:id", "DELETE"},
		{"启停共享查询方案", "query_scheme_shared_manage_enable", "/admin/query-schemes/shared/:id/enabled", "PUT"},
	}
	for index, policy := range policies {
		button := apiPermissionWithAPI(
			0, systemMenu.Id, policy.name, policy.code, enum.Top,
			queryscheme.SharedManageCapability, "manage_search", "primary", uint8(index+1), policy.path, policy.method,
		)
		button.IsHidden = true
		if err := seedMenuButton(db, sf, role.Id, role.Name, button); err != nil {
			return err
		}
	}
	return nil
}

func queryScopeMenuSupportsTable(menu model.SysMenu, tableCode string) bool {
	if strings.TrimSpace(menu.TableCode) == tableCode {
		return true
	}
	for _, configured := range strings.Split(menu.Option, ",") {
		if strings.TrimSpace(configured) == tableCode {
			return true
		}
	}
	return false
}
