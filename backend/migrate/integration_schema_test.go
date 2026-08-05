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
	if !db.Migrator().HasTable(&model.Credential{}) {
		t.Fatal("credential table was not created")
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
	credential := model.Credential{Basic: model.Basic{Id: 5}, ExternalSystemID: 1, CredentialCode: "erp_key", Name: "ERP Key", CredentialType: model.CredentialTypeAPIKey, Status: model.CredentialStatusDraft, SecretStorageRef: "ref-1", SecretCiphertext: "cipher", SecretNonce: "nonce", SecretFingerprint: "fingerprint", Version: 1, Revision: 1}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("create credential: %v", err)
	}
	credential.Id = 6
	credential.SecretStorageRef = "ref-2"
	if err := db.Create(&credential).Error; err == nil {
		t.Fatal("expected duplicate credential code to be rejected")
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
		&model.Credential{},
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
	if menu.Title != "router.integration.externalSystem" {
		t.Fatalf("external system menu title = %q", menu.Title)
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
	if interfaceMenu.Title != "router.integration.interfaceDefinition" {
		t.Fatalf("interface definition menu title = %q", interfaceMenu.Title)
	}
	var rootMenu model.SysMenu
	if err := db.Where("name = ?", "integration").First(&rootMenu).Error; err != nil {
		t.Fatalf("load integration root menu: %v", err)
	}
	if rootMenu.Title != "router.integration.default" {
		t.Fatalf("integration root menu title = %q", rootMenu.Title)
	}
	var executionButtonCount int64
	if err := db.Model(&model.SysMenuButton{}).
		Where("menu_id = ? AND code LIKE ?", rootMenu.Id, "integration_execution_%").
		Count(&executionButtonCount).Error; err != nil {
		t.Fatalf("count execution permissions: %v", err)
	}
	if executionButtonCount != 7 {
		t.Fatalf("execution permission count = %d, want 7", executionButtonCount)
	}
	var executionCasbinCount int64
	if err := db.Model(&model.CasbinRule{}).
		Where("v1 LIKE ?", "%/admin/integration/execution%").
		Count(&executionCasbinCount).Error; err != nil {
		t.Fatalf("count execution Casbin policies: %v", err)
	}
	if executionCasbinCount != 7 {
		t.Fatalf("execution Casbin policy count = %d, want 7", executionCasbinCount)
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
	var credentialMenu model.SysMenu
	if err := db.Where("name = ?", "integration_credential").First(&credentialMenu).Error; err != nil {
		t.Fatalf("load credential menu: %v", err)
	}
	if credentialMenu.TableCode != credentialTableCode || credentialMenu.Title != "router.integration.credential" {
		t.Fatalf("unexpected credential menu: %+v", credentialMenu)
	}
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ?", credentialMenu.Id).Count(&buttonCount).Error; err != nil {
		t.Fatalf("count credential buttons: %v", err)
	}
	if buttonCount != 9 {
		t.Fatalf("credential button count = %d, want 9", buttonCount)
	}
	var credentialCasbinCount int64
	if err := db.Model(&model.CasbinRule{}).Where("v1 LIKE ?", "%/admin/integration/credential%").Count(&credentialCasbinCount).Error; err != nil {
		t.Fatalf("count credential Casbin policies: %v", err)
	}
	if credentialCasbinCount != 8 {
		t.Fatalf("credential Casbin policy count = %d, want 8", credentialCasbinCount)
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

func TestIntegrationExecutionMetadataUsesControlledQueryFields(t *testing.T) {
	status := model.SysTableField{FieldCode: "status"}
	applyIntegrationExecutionFieldDefaults(integrationExecutionTableCode, &status)
	if !status.IsListShow || !status.IsAdvancedSearch || status.DictCode == nil || *status.DictCode != "integration_execution_status" {
		t.Fatalf("status metadata = %+v", status)
	}
	secretFields := []string{"idempotency_key", "input_hash", "result_hash", "result_summary"}
	for _, code := range secretFields {
		field := model.SysTableField{FieldCode: code, IsListShow: true, IsQuickSearch: true, IsAdvancedSearch: true, IsSort: true}
		applyIntegrationExecutionFieldDefaults(integrationExecutionTableCode, &field)
		if field.IsListShow || field.IsQuickSearch || field.IsAdvancedSearch || field.IsSort {
			t.Fatalf("sensitive metadata %s = %+v", code, field)
		}
	}
}
