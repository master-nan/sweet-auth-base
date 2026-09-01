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
