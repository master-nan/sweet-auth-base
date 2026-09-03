package main

import (
	"backend/enum"
	"backend/model"
	"testing"
)

func TestUnifyReportRuntimeComponentOnlyUpdatesPublishedReportMenus(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysMenu{}); err != nil {
		t.Fatalf("migrate sys_menu: %v", err)
	}

	menus := []model.SysMenu{
		{
			Basic:     model.Basic{Id: 1001, State: true},
			Name:      "published_report",
			PageType:  enum.MenuPageTypeReport,
			Component: legacyReportRuntimeComponent,
		},
		{
			Basic:     model.Basic{Id: 1002, State: true},
			Name:      "fixed_page",
			PageType:  enum.MenuPageTypeFixed,
			Component: legacyReportRuntimeComponent,
		},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	if err := unifyReportRuntimeComponent(db); err != nil {
		t.Fatalf("unify report runtime component: %v", err)
	}
	if err := unifyReportRuntimeComponent(db); err != nil {
		t.Fatalf("repeat report runtime component migration: %v", err)
	}

	var reportMenu model.SysMenu
	if err := db.First(&reportMenu, 1001).Error; err != nil {
		t.Fatalf("query report menu: %v", err)
	}
	if reportMenu.Component != currentReportRuntimeComponent {
		t.Fatalf("report component = %q, want %q", reportMenu.Component, currentReportRuntimeComponent)
	}

	var fixedMenu model.SysMenu
	if err := db.First(&fixedMenu, 1002).Error; err != nil {
		t.Fatalf("query fixed menu: %v", err)
	}
	if fixedMenu.Component != legacyReportRuntimeComponent {
		t.Fatalf("fixed component = %q, want unchanged %q", fixedMenu.Component, legacyReportRuntimeComponent)
	}
}

func TestPurgeHistoricalReportRecordsRemovesReportOwnedData(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(
		&model.ReportDefinition{},
		&model.ReportDefinitionVersion{},
		&model.ReportExecutionLog{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.CasbinRule{},
		&model.DataResource{},
		&model.DataResourceOperation{},
	); err != nil {
		t.Fatalf("migrate report cleanup tables: %v", err)
	}

	reportID := 1001
	reportMenuID := 2001
	otherMenuID := 2002
	reportButtonID := 3001
	otherButtonID := 3002
	resourceID := 4001
	report := model.ReportDefinition{
		Basic:            model.Basic{Id: reportID, State: true},
		Code:             "historical_report",
		Name:             "历史报表",
		Status:           "published",
		PermissionMenuId: reportMenuID,
	}
	if err := db.Create(&report).Error; err != nil {
		t.Fatalf("seed report: %v", err)
	}
	if err := db.Create(&model.ReportDefinitionVersion{
		Basic:      model.Basic{Id: 1101, State: true},
		ReportId:   reportID,
		VersionNo:  1,
		ReportCode: report.Code,
		ReportName: report.Name,
		Status:     "published",
	}).Error; err != nil {
		t.Fatalf("seed report version: %v", err)
	}
	if err := db.Create(&model.ReportExecutionLog{
		Basic:      model.Basic{Id: 1201, State: true},
		ReportId:   reportID,
		ReportCode: report.Code,
		Action:     "preview",
	}).Error; err != nil {
		t.Fatalf("seed report execution log: %v", err)
	}
	menus := []model.SysMenu{
		{Basic: model.Basic{Id: reportMenuID, State: true}, Name: "historical_report", PageType: enum.MenuPageTypeReport},
		{Basic: model.Basic{Id: otherMenuID, State: true}, Name: "other", PageType: enum.MenuPageTypeFixed},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	buttons := []model.SysMenuButton{
		{Basic: model.Basic{Id: reportButtonID, State: true}, MenuId: reportMenuID, Code: "historical_report_run", Name: "运行", Path: "/admin/report/:id/run", Method: "POST"},
		{Basic: model.Basic{Id: otherButtonID, State: true}, MenuId: otherMenuID, Code: "other_query", Name: "查询", Path: "/admin/other/query", Method: "POST"},
	}
	if err := db.Create(&buttons).Error; err != nil {
		t.Fatalf("seed menu buttons: %v", err)
	}
	if err := db.Create(&model.SysRoleMenu{RoleId: 1, MenuId: reportMenuID}).Error; err != nil {
		t.Fatalf("seed role menu: %v", err)
	}
	if err := db.Create(&model.SysRoleMenuButton{RoleId: 1, MenuId: reportMenuID, ButtonId: reportButtonID}).Error; err != nil {
		t.Fatalf("seed role menu button: %v", err)
	}
	if err := db.Create(&model.CasbinRule{Id: 5001, PType: "p", V0: "admin", V1: "/admin/report/:id/run", V2: "POST"}).Error; err != nil {
		t.Fatalf("seed report policy: %v", err)
	}
	if err := db.Create(&model.DataResource{
		Basic:              model.Basic{Id: resourceID, State: true},
		ResourceCode:       "report:historical_report",
		Name:               "历史报表",
		ResourceType:       model.DataResourceTypeReport,
		ReportDefinitionId: &reportID,
		AdapterCode:        "report",
	}).Error; err != nil {
		t.Fatalf("seed report resource: %v", err)
	}
	if err := db.Create(&model.DataResourceOperation{
		Basic:      model.Basic{Id: 4101, State: true},
		ResourceId: resourceID,
		Operation:  model.DataPermissionOperationRun,
	}).Error; err != nil {
		t.Fatalf("seed report resource operation: %v", err)
	}

	if err := purgeHistoricalReportRecords(db); err != nil {
		t.Fatalf("purge report records: %v", err)
	}
	if err := purgeHistoricalReportRecords(db); err != nil {
		t.Fatalf("repeat report purge: %v", err)
	}

	for name, value := range map[string]any{
		"report definitions": &model.ReportDefinition{},
		"report versions":    &model.ReportDefinitionVersion{},
		"report logs":        &model.ReportExecutionLog{},
		"report menu":        &model.SysMenu{},
		"report button":      &model.SysMenuButton{},
		"report role menu":   &model.SysRoleMenu{},
		"report role button": &model.SysRoleMenuButton{},
		"report policy":      &model.CasbinRule{},
		"report resource":    &model.DataResource{},
		"resource operation": &model.DataResourceOperation{},
	} {
		query := db.Unscoped().Model(value)
		switch name {
		case "report menu":
			query = query.Where("id = ?", reportMenuID)
		case "report button":
			query = query.Where("id = ?", reportButtonID)
		case "report role menu", "report role button":
			query = query.Where("menu_id = ?", reportMenuID)
		case "report policy":
			query = query.Where("v1 = ?", "/admin/report/:id/run")
		case "report resource":
			query = query.Where("id = ?", resourceID)
		case "resource operation":
			query = query.Where("resource_id = ?", resourceID)
		}
		var count int64
		if err := query.Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", name, count)
		}
	}

	var otherMenuCount int64
	if err := db.Model(&model.SysMenu{}).Where("id = ?", otherMenuID).Count(&otherMenuCount).Error; err != nil {
		t.Fatalf("count unrelated menu: %v", err)
	}
	if otherMenuCount != 1 {
		t.Fatalf("unrelated menu count = %d, want 1", otherMenuCount)
	}
}
