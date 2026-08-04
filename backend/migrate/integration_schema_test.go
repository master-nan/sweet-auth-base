package main

import (
	"backend/internal/utils"
	"backend/model"
	"testing"
)

func TestIntegrationConfigurationSchemaIsIdempotentAndUnique(t *testing.T) {
	db := migrateTestDB(t)
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("first integration migration: %v", err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("second integration migration: %v", err)
	}
	if !db.Migrator().HasTable(&model.ExternalSystem{}) {
		t.Fatal("external system table was not created")
	}
	if !db.Migrator().HasTable(&model.InterfaceDefinition{}) {
		t.Fatal("interface definition table was not created")
	}
	first := model.ExternalSystem{
		Basic:           model.Basic{Id: 1},
		SystemCode:      "demo_erp",
		Name:            "Demo ERP",
		SystemType:      model.ExternalSystemTypeERP,
		BaseURL:         "https://erp.example.com",
		OwnerIdentifier: "owner-1",
		OwnerName:       "实施负责人",
		Status:          model.ExternalSystemStatusDraft,
		Revision:        1,
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create external system: %v", err)
	}
	first.Id = 2
	if err := db.Create(&first).Error; err == nil {
		t.Fatal("expected duplicate system code to be rejected")
	}
	definition := model.InterfaceDefinition{
		Basic: model.Basic{Id: 11}, ExternalSystemID: 1, InterfaceCode: "order_query", Name: "订单查询", Version: 1,
		Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET, RelativePath: "/api/orders",
		TimeoutSeconds: 30, ResponseLimit: 1024, Status: model.InterfaceDefinitionStatusDraft, Revision: 1,
	}
	if err := db.Create(&definition).Error; err != nil {
		t.Fatalf("create interface definition: %v", err)
	}
	definition.Id = 12
	if err := db.Create(&definition).Error; err == nil {
		t.Fatal("expected duplicate interface version to be rejected")
	}
}

func TestIntegrationConfigurationSeedCreatesMenuButtonsAndCasbin(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(
		&model.ExternalSystem{},
		&model.InterfaceDefinition{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysRole{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.CasbinRule{},
	); err != nil {
		t.Fatalf("migrate integration seed fixtures: %v", err)
	}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	for run := 0; run < 2; run++ {
		if err := seedIntegrationConfigurationFoundation(db, sf); err != nil {
			t.Fatalf("seed integration foundation run %d: %v", run+1, err)
		}
	}

	var menu model.SysMenu
	if err := db.Where("name = ?", "integration_external_system").First(&menu).Error; err != nil {
		t.Fatalf("load external system menu: %v", err)
	}
	if menu.TableCode != externalSystemTableCode || menu.Component != "pages/integration/external-system/Index.vue" {
		t.Fatalf("unexpected external system menu: %+v", menu)
	}
	var buttonCount int64
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ?", menu.Id).Count(&buttonCount).Error; err != nil {
		t.Fatalf("count external system buttons: %v", err)
	}
	if buttonCount != 7 {
		t.Fatalf("button count = %d, want 7", buttonCount)
	}
	var interfaceMenu model.SysMenu
	if err := db.Where("name = ?", "integration_interface_definition").First(&interfaceMenu).Error; err != nil {
		t.Fatalf("load interface definition menu: %v", err)
	}
	if interfaceMenu.TableCode != interfaceDefinitionTableCode || interfaceMenu.Component != "pages/integration/interface-definition/Index.vue" {
		t.Fatalf("unexpected interface definition menu: %+v", interfaceMenu)
	}
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ?", interfaceMenu.Id).Count(&buttonCount).Error; err != nil {
		t.Fatalf("count interface definition buttons: %v", err)
	}
	if buttonCount != 8 {
		t.Fatalf("interface button count = %d, want 8", buttonCount)
	}
	var interfaceCasbinCount int64
	if err := db.Model(&model.CasbinRule{}).
		Where("v1 LIKE ?", "%/admin/integration/interface-definition%").
		Count(&interfaceCasbinCount).Error; err != nil {
		t.Fatalf("count interface definition Casbin policies: %v", err)
	}
	if interfaceCasbinCount != 7 {
		t.Fatalf("interface Casbin policy count = %d, want 7", interfaceCasbinCount)
	}
	var casbinCount int64
	if err := db.Model(&model.CasbinRule{}).
		Where("v1 LIKE ?", "%/admin/integration/external-system%").
		Count(&casbinCount).Error; err != nil {
		t.Fatalf("count integration Casbin policies: %v", err)
	}
	if casbinCount != 6 {
		t.Fatalf("Casbin policy count = %d, want 6", casbinCount)
	}
}
