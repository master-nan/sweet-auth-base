package main

import (
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

var legacyDataPermissionTables = []string{
	"sys_user_dimension_value",
	"sys_user_data_scope_override",
	"sys_role_data_scope",
	"sys_data_scope_binding",
	"sys_data_dimension",
}

var legacyDataPermissionButtonCodes = []string{
	"system_role_data_permission_query",
	"system_role_data_permission_save",
	"system_user_data_permission",
	"system_user_data_permission_query",
	"system_user_dimension_value_query",
	"system_user_dimension_value_save",
	"system_data_permission_dimension_query",
	"system_data_permission_dimension_detail",
	"system_data_permission_dimension_create",
	"system_data_permission_dimension_update",
	"system_data_permission_dimension_delete",
	"system_data_permission_dimension_options",
	"system_data_permission_binding_query",
	"system_data_permission_binding_save",
	"system_data_permission_role_query",
	"system_data_permission_role_save",
	"system_data_permission_user_query",
	"system_data_permission_user_save",
	"system_data_permission_user_dimension_query",
	"system_data_permission_user_dimension_save",
	"system_data_permission_debug",
}

var legacyDataPermissionPaths = []string{
	"/admin/data-permission/dimension/query",
	"/admin/data-permission/dimension/:id",
	"/admin/data-permission/dimension",
	"/admin/data-permission/dimension-options/:code",
	"/admin/data-permission/bindings/menu/:menuId",
	"/admin/data-permission/debug",
	"/admin/role/:id/data-permissions",
	"/admin/user/:id/data-permissions",
	"/admin/user/:id/dimension-values",
}

func removeLegacyDataPermissionSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := removeLegacyDataPermissionConfiguration(tx); err != nil {
			return err
		}
		for _, table := range legacyDataPermissionTables {
			if !tx.Migrator().HasTable(table) {
				continue
			}
			if err := tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %q`, table)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func removeLegacyDataPermissionConfiguration(tx *gorm.DB) error {
	if tx.Migrator().HasTable(&model.SysMenuButton{}) {
		var buttonIDs []int
		if err := tx.Unscoped().Model(&model.SysMenuButton{}).
			Where("code IN ?", legacyDataPermissionButtonCodes).
			Pluck("id", &buttonIDs).Error; err != nil {
			return err
		}
		if len(buttonIDs) > 0 && tx.Migrator().HasTable(&model.SysRoleMenuButton{}) {
			if err := tx.Where("button_id IN ?", buttonIDs).Delete(&model.SysRoleMenuButton{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Unscoped().Where("code IN ?", legacyDataPermissionButtonCodes).
			Delete(&model.SysMenuButton{}).Error; err != nil {
			return err
		}
	}

	if tx.Migrator().HasTable(&model.CasbinRule{}) {
		if err := tx.Where("v1 IN ?", legacyDataPermissionPaths).Delete(&model.CasbinRule{}).Error; err != nil {
			return err
		}
	}

	if tx.Migrator().HasTable(&model.SysDictItem{}) {
		if err := tx.Unscoped().Where("item_code IN ?", []string{
			"sys_menu_button_event_action_assign_data_permission",
			"sys_menu_button_event_action_query_data_permission",
		}).Delete(&model.SysDictItem{}).Error; err != nil {
			return err
		}
	}

	return removeLegacyDataPermissionMetadata(tx)
}

func removeLegacyDataPermissionMetadata(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.SysTable{}) {
		return nil
	}
	var tableIDs []int
	if err := tx.Unscoped().Model(&model.SysTable{}).
		Where("table_code IN ?", legacyDataPermissionTables).
		Pluck("id", &tableIDs).Error; err != nil {
		return err
	}
	if len(tableIDs) == 0 {
		return nil
	}
	if tx.Migrator().HasTable(&model.SysTableRelation{}) {
		if err := tx.Unscoped().Where("table_id IN ? OR related_table_id IN ?", tableIDs, tableIDs).
			Delete(&model.SysTableRelation{}).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable(&model.SysTableIndexField{}) && tx.Migrator().HasTable(&model.SysTableIndex{}) {
		indexIDs := tx.Model(&model.SysTableIndex{}).
			Select("id").
			Where("table_id IN ?", tableIDs)
		if err := tx.Where("index_id IN (?)", indexIDs).
			Delete(&model.SysTableIndexField{}).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable(&model.SysTableIndex{}) {
		if err := tx.Unscoped().Where("table_id IN ?", tableIDs).Delete(&model.SysTableIndex{}).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable(&model.SysTableField{}) {
		if err := tx.Unscoped().Where("table_id IN ?", tableIDs).Delete(&model.SysTableField{}).Error; err != nil {
			return err
		}
	}
	return tx.Unscoped().Where("id IN ?", tableIDs).Delete(&model.SysTable{}).Error
}
