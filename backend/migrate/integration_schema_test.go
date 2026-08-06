package main

import (
	"backend/enum"
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
	var role model.SysRole
	if err := db.Where("name = ?", "super_admin").First(&role).Error; err != nil {
		t.Fatalf("load integration seed role: %v", err)
	}
	legacyButtons := []model.SysMenuButton{
		apiPermissionWithAPI(12304, rootMenu.Id, "启动执行", "integration_execution_start", enum.Top, "start", "play_arrow", "primary", 121, "/admin/integration/execution/:id/start", "PUT"),
		apiPermissionWithAPI(12305, rootMenu.Id, "完成执行", "integration_execution_complete", enum.Top, "complete", "done", "positive", 122, "/admin/integration/execution/:id/complete", "PUT"),
		apiPermissionWithAPI(12306, rootMenu.Id, "执行失败", "integration_execution_fail", enum.Top, "fail", "error", "negative", 123, "/admin/integration/execution/:id/fail", "PUT"),
	}
	if err := seedMenuButtons(db, sf, role.Id, role.Name, legacyButtons); err != nil {
		t.Fatalf("prepare legacy execution permissions: %v", err)
	}
	crossMenuButton := apiPermissionWithAPI(
		22304, menu.Id, "同编码非集成命令", "integration_execution_start", enum.Top,
		"custom", "play_arrow", "primary", 200, "/admin/custom/integration-start", "POST",
	)
	if err := seedMenuButtons(db, sf, role.Id, role.Name, []model.SysMenuButton{crossMenuButton}); err != nil {
		t.Fatalf("prepare cross-menu permission fixture: %v", err)
	}
	if err := seedIntegrationConfigurationFoundation(db, sf); err != nil {
		t.Fatalf("clean legacy execution permissions: %v", err)
	}
	if err := seedIntegrationConfigurationFoundation(db, sf); err != nil {
		t.Fatalf("repeat legacy execution permission cleanup: %v", err)
	}
	var executionMenu model.SysMenu
	if err := db.Where("name = ?", "integration_execution").First(&executionMenu).Error; err != nil {
		t.Fatalf("load execution menu: %v", err)
	}
	if executionMenu.TableCode != integrationExecutionTableCode || executionMenu.Title != "router.integration.execution" {
		t.Fatalf("unexpected execution menu: %+v", executionMenu)
	}
	var executionButtonCount int64
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ?", executionMenu.Id).Count(&executionButtonCount).Error; err != nil {
		t.Fatalf("count execution permissions: %v", err)
	}
	if executionButtonCount != 3 {
		t.Fatalf("execution permission count = %d, want 3", executionButtonCount)
	}
	var executionCasbinCount int64
	if err := db.Model(&model.CasbinRule{}).
		Where("v1 LIKE ?", "%/admin/integration/execution%").
		Count(&executionCasbinCount).Error; err != nil {
		t.Fatalf("count execution Casbin policies: %v", err)
	}
	if executionCasbinCount != 4 {
		t.Fatalf("execution Casbin policy count = %d, want 4", executionCasbinCount)
	}
	var deprecatedButtonCount int64
	deprecatedCodes := []string{"integration_execution_start", "integration_execution_complete", "integration_execution_fail"}
	if err := db.Unscoped().Model(&model.SysMenuButton{}).
		Where("menu_id = ? AND code IN ?", rootMenu.Id, deprecatedCodes).
		Count(&deprecatedButtonCount).Error; err != nil {
		t.Fatalf("count deprecated execution permissions: %v", err)
	}
	if deprecatedButtonCount != 0 {
		t.Fatalf("deprecated execution permission count = %d, want 0", deprecatedButtonCount)
	}
	var preservedCrossMenuButtonCount int64
	if err := db.Model(&model.SysMenuButton{}).
		Where("menu_id = ? AND code = ? AND path = ?", menu.Id, crossMenuButton.Code, crossMenuButton.Path).
		Count(&preservedCrossMenuButtonCount).Error; err != nil {
		t.Fatalf("count preserved cross-menu permission: %v", err)
	}
	if preservedCrossMenuButtonCount != 1 {
		t.Fatalf("preserved cross-menu permission count = %d, want 1", preservedCrossMenuButtonCount)
	}
	var deprecatedRoleButtonCount int64
	if err := db.Model(&model.SysRoleMenuButton{}).
		Where("button_id IN ?", []int{12304, 12305, 12306}).
		Count(&deprecatedRoleButtonCount).Error; err != nil {
		t.Fatalf("count deprecated role execution permissions: %v", err)
	}
	if deprecatedRoleButtonCount != 0 {
		t.Fatalf("deprecated role execution permission count = %d, want 0", deprecatedRoleButtonCount)
	}
	var deprecatedCasbinCount int64
	deprecatedPaths := []string{
		"/admin/integration/execution/:id/start",
		"/admin/integration/execution/:id/complete",
		"/admin/integration/execution/:id/fail",
	}
	if err := db.Model(&model.CasbinRule{}).Where("v1 IN ?", deprecatedPaths).Count(&deprecatedCasbinCount).Error; err != nil {
		t.Fatalf("count deprecated execution Casbin policies: %v", err)
	}
	if deprecatedCasbinCount != 0 {
		t.Fatalf("deprecated execution Casbin policy count = %d, want 0", deprecatedCasbinCount)
	}
	var logMenu model.SysMenu
	if err := db.Where("name = ?", "integration_log").First(&logMenu).Error; err != nil {
		t.Fatalf("load log menu: %v", err)
	}
	if logMenu.TableCode != integrationLogTableCode || logMenu.Title != "router.integration.log" {
		t.Fatalf("unexpected log menu: %+v", logMenu)
	}
	var logButtonCount int64
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ?", logMenu.Id).Count(&logButtonCount).Error; err != nil {
		t.Fatalf("count log permissions: %v", err)
	}
	if logButtonCount != 2 {
		t.Fatalf("log permission count = %d, want 2", logButtonCount)
	}
	var logCasbinCount int64
	if err := db.Model(&model.CasbinRule{}).Where("v1 LIKE ?", "%/admin/integration/log%").Count(&logCasbinCount).Error; err != nil {
		t.Fatalf("count log Casbin policies: %v", err)
	}
	if logCasbinCount != 2 {
		t.Fatalf("log Casbin policy count = %d, want 2", logCasbinCount)
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
