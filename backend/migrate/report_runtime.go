package main

import (
	"backend/enum"
	"backend/model"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	legacyReportRuntimeComponent  = "pages/report-v2/runtime/ReportRuntimePage.vue"
	currentReportRuntimeComponent = "pages/report/runtime/ReportRuntimePage.vue"
)

func unifyReportRuntimeComponent(db *gorm.DB) error {
	if err := db.Model(&model.SysMenu{}).
		Where("page_type = ? AND component = ?", enum.MenuPageTypeReport, legacyReportRuntimeComponent).
		Update("component", currentReportRuntimeComponent).Error; err != nil {
		return fmt.Errorf("update published report runtime component: %w", err)
	}
	return nil
}

type reportPolicyEndpoint struct {
	path   string
	method string
}

func purgeHistoricalReportRecords(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		menuIDs, err := historicalReportMenuIDs(tx)
		if err != nil {
			return err
		}
		if err := purgeHistoricalReportMenus(tx, menuIDs); err != nil {
			return err
		}
		if err := purgeHistoricalReportResources(tx); err != nil {
			return err
		}
		for _, value := range []any{
			&model.ReportExecutionLog{},
			&model.ReportDefinitionVersion{},
			&model.ReportDefinition{},
		} {
			if tx.Migrator().HasTable(value) {
				if err := tx.Unscoped().Where("1 = 1").Delete(value).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func historicalReportMenuIDs(tx *gorm.DB) ([]int, error) {
	ids := make([]int, 0)
	if tx.Migrator().HasTable(&model.SysMenu{}) {
		if err := tx.Unscoped().Model(&model.SysMenu{}).
			Where("page_type = ?", enum.MenuPageTypeReport).
			Pluck("id", &ids).Error; err != nil {
			return nil, err
		}
	}
	if tx.Migrator().HasTable(&model.ReportDefinition{}) {
		var permissionMenuIDs []int
		if err := tx.Unscoped().Model(&model.ReportDefinition{}).
			Where("permission_menu_id > 0").
			Pluck("permission_menu_id", &permissionMenuIDs).Error; err != nil {
			return nil, err
		}
		ids = append(ids, permissionMenuIDs...)
	}
	seen := make(map[int]struct{}, len(ids))
	unique := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique, nil
}

func purgeHistoricalReportMenus(tx *gorm.DB, menuIDs []int) error {
	if len(menuIDs) == 0 {
		return nil
	}
	var buttonIDs []int
	var endpoints []reportPolicyEndpoint
	if tx.Migrator().HasTable(&model.SysMenuButton{}) {
		var buttons []model.SysMenuButton
		if err := tx.Unscoped().Where("menu_id IN ?", menuIDs).Find(&buttons).Error; err != nil {
			return err
		}
		buttonIDs = make([]int, 0, len(buttons))
		endpoints = make([]reportPolicyEndpoint, 0, len(buttons))
		for _, button := range buttons {
			buttonIDs = append(buttonIDs, button.Id)
			path := strings.TrimSpace(button.Path)
			method := strings.ToUpper(strings.TrimSpace(button.Method))
			if path != "" && method != "" {
				endpoints = append(endpoints, reportPolicyEndpoint{path: path, method: method})
			}
		}
	}
	if tx.Migrator().HasTable(&model.SysRoleMenuButton{}) {
		query := tx.Where("menu_id IN ?", menuIDs)
		if len(buttonIDs) > 0 {
			query = query.Or("button_id IN ?", buttonIDs)
		}
		if err := query.Delete(&model.SysRoleMenuButton{}).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable(&model.SysRoleMenu{}) {
		if err := tx.Where("menu_id IN ?", menuIDs).Delete(&model.SysRoleMenu{}).Error; err != nil {
			return err
		}
	}
	if len(buttonIDs) > 0 {
		if err := tx.Unscoped().Where("id IN ?", buttonIDs).Delete(&model.SysMenuButton{}).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable(&model.SysMenu{}) {
		if err := tx.Unscoped().Where("id IN ?", menuIDs).Delete(&model.SysMenu{}).Error; err != nil {
			return err
		}
	}
	return purgeOrphanReportPolicies(tx, endpoints)
}

func purgeOrphanReportPolicies(tx *gorm.DB, endpoints []reportPolicyEndpoint) error {
	if len(endpoints) == 0 || !tx.Migrator().HasTable(&model.CasbinRule{}) || !tx.Migrator().HasTable(&model.SysMenuButton{}) {
		return nil
	}
	seen := make(map[reportPolicyEndpoint]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		if _, exists := seen[endpoint]; exists {
			continue
		}
		seen[endpoint] = struct{}{}
		var count int64
		if err := tx.Model(&model.SysMenuButton{}).
			Where("path = ? AND UPPER(method) = ?", endpoint.path, endpoint.method).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Where("ptype = ? AND v1 = ? AND UPPER(v2) = ?", "p", endpoint.path, endpoint.method).
				Delete(&model.CasbinRule{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func purgeHistoricalReportResources(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.DataResource{}) {
		return nil
	}
	var resourceIDs []int
	if err := tx.Unscoped().Model(&model.DataResource{}).
		Where("report_definition_id IS NOT NULL").
		Pluck("id", &resourceIDs).Error; err != nil {
		return err
	}
	if len(resourceIDs) == 0 {
		return nil
	}
	for _, value := range []any{
		&model.DataGrant{},
		&model.DataOwnershipField{},
		&model.DataResourceOperation{},
	} {
		if tx.Migrator().HasTable(value) {
			if err := tx.Unscoped().Where("resource_id IN ?", resourceIDs).Delete(value).Error; err != nil {
				return err
			}
		}
	}
	return tx.Unscoped().Where("id IN ?", resourceIDs).Delete(&model.DataResource{}).Error
}
