package main

import (
	"backend/enum"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
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
		TimeoutSeconds: 30, ResponseLimit: 1024, IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod,
		Status: model.InterfaceDefinitionStatusDraft, Revision: 1,
	}
	if err := db.Create(&definition).Error; err != nil {
		t.Fatalf("create interface definition: %v", err)
	}
	definition.Id = 12
	if err := db.Create(&definition).Error; err == nil {
		t.Fatal("expected duplicate interface version to be rejected")
	}
}

func TestIntegrationRuntimeContractPostgresMigration(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("integration_runtime_contract_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error })
	db, err := gorm.Open(postgres.Open(postgresDSNWithSearchPath(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("initial migration: %v", err)
	}
	for _, name := range []string{"chk_integration_interface_enabled_timeout", "chk_integration_interface_enabled_response_limit"} {
		if err := db.Exec(fmt.Sprintf(`ALTER TABLE integration_interface_definition DROP CONSTRAINT IF EXISTS %s`, name)).Error; err != nil {
			t.Fatalf("drop runtime constraint %s: %v", name, err)
		}
	}
	system := model.ExternalSystem{
		Basic: model.Basic{Id: 7001, State: true}, SystemCode: "runtime_contract", Name: "Runtime Contract",
		SystemType: model.ExternalSystemTypeERP, BaseURL: "https://runtime.example.com", OwnerIdentifier: "owner",
		OwnerName: "owner", Status: model.ExternalSystemStatusEnabled, Revision: 1,
	}
	if err := db.Create(&system).Error; err != nil {
		t.Fatalf("create system: %v", err)
	}
	definition := model.InterfaceDefinition{
		Basic: model.Basic{Id: 7002, State: true}, ExternalSystemID: system.Id, InterfaceCode: "legacy_limit",
		Name: "Legacy Limit", Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET,
		RelativePath: "/api/legacy", InputContract: datatypes.JSON([]byte(`{"version":1,"parameters":[]}`)),
		TimeoutSeconds: 200, ResponseLimit: 80 * 1024 * 1024, IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod,
		Status: model.InterfaceDefinitionStatusEnabled, Revision: 1,
	}
	if err := db.Create(&definition).Error; err != nil {
		t.Fatalf("create legacy enabled interface: %v", err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("contract migration: %v", err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("repeat contract migration: %v", err)
	}
	var migrated model.InterfaceDefinition
	if err := db.First(&migrated, definition.Id).Error; err != nil {
		t.Fatalf("load migrated interface: %v", err)
	}
	if migrated.Status != model.InterfaceDefinitionStatusDisabled || migrated.State || migrated.TimeoutSeconds != 200 || migrated.ResponseLimit != 80*1024*1024 {
		t.Fatalf("legacy enabled interface was not safely disabled: %+v", migrated)
	}
	if err := db.Model(&model.InterfaceDefinition{}).Where("id = ?", definition.Id).Updates(map[string]any{"status": model.InterfaceDefinitionStatusEnabled, "state": true}).Error; err == nil {
		t.Fatal("expected PostgreSQL runtime contract CHECK to reject incompatible enable")
	}
}

func TestRetryPolicyPostgresConstraintsAndExecutionSnapshot(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("integration_retry_policy_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error })
	db, err := gorm.Open(postgres.Open(postgresDSNWithSearchPath(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("initial configuration migration: %v", err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatalf("repeat configuration migration: %v", err)
	}

	policy := postgresRetryPolicyFixture(9101, 1, model.RetryPolicyStatusEnabled)
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("create enabled retry policy: %v", err)
	}
	conflict := postgresRetryPolicyFixture(9102, 2, model.RetryPolicyStatusEnabled)
	if err := db.Create(&conflict).Error; err == nil {
		t.Fatal("expected partial unique index to reject two enabled versions")
	}
	invalidAttempts := postgresRetryPolicyFixture(9103, 2, model.RetryPolicyStatusDraft)
	invalidAttempts.MaxAttempts = 11
	if err := db.Create(&invalidAttempts).Error; err == nil {
		t.Fatal("expected max_attempts CHECK to reject invalid value")
	}
	invalidHTTP := postgresRetryPolicyFixture(9104, 2, model.RetryPolicyStatusDraft)
	invalidHTTP.RetryableHTTPStatuses = datatypes.JSON([]byte(`[500]`))
	if err := db.Create(&invalidHTTP).Error; err == nil {
		t.Fatal("expected HTTP status whitelist CHECK to reject invalid value")
	}
	var indexCount int64
	if err := db.Raw(`SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'uni_integration_retry_policy_enabled'`).Scan(&indexCount).Error; err != nil || indexCount != 1 {
		t.Fatalf("partial index count=%d err=%v", indexCount, err)
	}

	system := model.ExternalSystem{Basic: model.Basic{Id: 9201, State: true}, SystemCode: "retry_pg", Name: "Retry PG", SystemType: model.ExternalSystemTypeERP, BaseURL: "https://example.com", OwnerIdentifier: "owner", OwnerName: "owner", Status: model.ExternalSystemStatusEnabled, Revision: 1}
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: 9202, State: true}, ExternalSystemID: system.Id, InterfaceCode: "retry_call", Name: "Retry Call", Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET, RelativePath: "/retry", InputContract: datatypes.JSON([]byte(`{"version":1,"parameters":[]}`)), TimeoutSeconds: 30, ResponseLimit: 1024, RetryPolicyID: &policy.Id, IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	if err := db.Create(&system).Error; err != nil {
		t.Fatalf("create system: %v", err)
	}
	if err := db.Create(&definition).Error; err != nil {
		t.Fatalf("create interface: %v", err)
	}
	if err := migrateIntegrationRuntimeSchema(db); err != nil {
		t.Fatalf("runtime migration: %v", err)
	}
	if err := migrateIntegrationRuntimeSchema(db); err != nil {
		t.Fatalf("repeat runtime migration: %v", err)
	}
	execution := model.IntegrationExecution{
		Basic: model.Basic{Id: 9301}, ExecutionNo: "EXEC-RETRY-PG", ExternalSystemID: system.Id,
		ExternalSystemCode: system.SystemCode, ExternalSystemName: system.Name, InterfaceDefinitionID: definition.Id,
		InterfaceCode: definition.InterfaceCode, InterfaceName: definition.Name, InterfaceVersion: definition.Version,
		TriggerSource: model.IntegrationTriggerSourceManual, Status: model.IntegrationExecutionStatusSucceeded,
		IdempotencyScope: "retry-pg", IdempotencyKey: "snapshot", InputHash: strings.Repeat("a", 64),
		InputSnapshot: datatypes.JSON([]byte(`{"version":1,"path_params":{},"query_params":{},"headers":{}}`)), InputSnapshotVersion: 1, InputSnapshotSize: 68,
		RetryPolicyID:              &policy.Id,
		RetryPolicySnapshot:        datatypes.JSON([]byte(`{"version":1,"policy_code":"pg_retry","policy_version":1,"max_attempts":3,"initial_delay_ms":5000,"max_delay_ms":300000,"backoff_type":"exponential","backoff_multiplier":"2","jitter_type":"full","jitter_ratio":"1","retry_window_ms":86400000,"retryable_error_categories":["network","remote","timeout"],"retryable_http_statuses":[429,502,503,504],"respect_retry_after":true,"idempotency_mode":"safe_method","remote_idempotency_header":""}`)),
		RetryPolicySnapshotVersion: 1, Revision: 1,
		RemoteIdempotencyMode: model.InterfaceIdempotencyModeSafeMethod,
	}
	if err := db.Create(&execution).Error; err != nil {
		t.Fatalf("persist execution retry snapshot: %v", err)
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, execution.Id).Error; err != nil {
		t.Fatalf("load execution retry snapshot: %v", err)
	}
	if stored.RetryPolicyID == nil || *stored.RetryPolicyID != policy.Id || stored.RetryPolicySnapshotVersion != 1 || !strings.Contains(string(stored.RetryPolicySnapshot), `"policy_code": "pg_retry"`) {
		t.Fatalf("unexpected PostgreSQL retry snapshot: id=%v version=%d snapshot=%s", stored.RetryPolicyID, stored.RetryPolicySnapshotVersion, stored.RetryPolicySnapshot)
	}
}

func postgresRetryPolicyFixture(id, version int, status string) model.RetryPolicy {
	return model.RetryPolicy{
		Basic: model.Basic{Id: id, State: status == model.RetryPolicyStatusEnabled}, PolicyCode: "pg_retry", PolicyName: "PG Retry",
		Version: version, Status: status, MaxAttempts: 3, InitialDelayMs: 5000, MaxDelayMs: 300000,
		BackoffType: model.RetryBackoffTypeExponential, BackoffMultiplier: 2,
		JitterType: model.RetryJitterTypeFull, JitterRatio: 1, RetryWindowMs: 86400000,
		RetryableErrorCategories: datatypes.JSON([]byte(`["network","timeout","remote"]`)),
		RetryableHTTPStatuses:    datatypes.JSON([]byte(`[429,502,503,504]`)), RespectRetryAfter: true, Revision: 1,
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
	var retryPolicyMenu model.SysMenu
	if err := db.Where("name = ?", "integration_retry_policy").First(&retryPolicyMenu).Error; err != nil {
		t.Fatalf("load retry policy menu: %v", err)
	}
	if retryPolicyMenu.TableCode != retryPolicyTableCode || retryPolicyMenu.Title != "router.integration.retryPolicy" || retryPolicyMenu.Sequence != 4 {
		t.Fatalf("unexpected retry policy menu: %+v", retryPolicyMenu)
	}
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ?", retryPolicyMenu.Id).Count(&buttonCount).Error; err != nil {
		t.Fatalf("count retry policy buttons: %v", err)
	}
	if buttonCount != 8 {
		t.Fatalf("retry policy button count = %d, want 8", buttonCount)
	}
	var forbiddenRetryButtons int64
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ? AND event_action IN ?", retryPolicyMenu.Id, []string{"delete", "retry_now", "replay"}).Count(&forbiddenRetryButtons).Error; err != nil || forbiddenRetryButtons != 0 {
		t.Fatalf("forbidden retry buttons=%d err=%v", forbiddenRetryButtons, err)
	}
	var retryPolicyCasbinCount int64
	if err := db.Model(&model.CasbinRule{}).Where("v1 LIKE ?", "%/admin/integration/retry-policy%").Count(&retryPolicyCasbinCount).Error; err != nil {
		t.Fatalf("count retry policy Casbin policies: %v", err)
	}
	if retryPolicyCasbinCount != 7 {
		t.Fatalf("retry policy Casbin policy count = %d, want 7", retryPolicyCasbinCount)
	}
	var syncTaskMenu model.SysMenu
	if err := db.Where("name = ?", "integration_sync_task").First(&syncTaskMenu).Error; err != nil {
		t.Fatalf("load sync task menu: %v", err)
	}
	if syncTaskMenu.TableCode != integrationSyncTaskTableCode || syncTaskMenu.Title != "router.integration.syncTask" || syncTaskMenu.Sequence != 5 {
		t.Fatalf("unexpected sync task menu: %+v", syncTaskMenu)
	}
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ?", syncTaskMenu.Id).Count(&buttonCount).Error; err != nil || buttonCount != 11 {
		t.Fatalf("sync task button count=%d err=%v", buttonCount, err)
	}
	var forbiddenSyncTaskButtons int64
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ? AND event_action IN ?", syncTaskMenu.Id, []string{"cancel", "delete", "checkpoint"}).Count(&forbiddenSyncTaskButtons).Error; err != nil || forbiddenSyncTaskButtons != 0 {
		t.Fatalf("forbidden sync task buttons=%d err=%v", forbiddenSyncTaskButtons, err)
	}
	var syncTaskCasbinCount int64
	if err := db.Model(&model.CasbinRule{}).Where("v1 LIKE ?", "%/admin/integration/sync-task%").Count(&syncTaskCasbinCount).Error; err != nil || syncTaskCasbinCount != 10 {
		t.Fatalf("sync task Casbin policies=%d err=%v", syncTaskCasbinCount, err)
	}
	var syncBatchMenu model.SysMenu
	if err := db.Where("name = ?", "integration_sync_batch").First(&syncBatchMenu).Error; err != nil {
		t.Fatalf("load sync batch menu: %v", err)
	}
	if syncBatchMenu.TableCode != integrationSyncBatchTableCode || syncBatchMenu.Title != "router.integration.syncBatch" || syncBatchMenu.Sequence != 6 {
		t.Fatalf("unexpected sync batch menu: %+v", syncBatchMenu)
	}
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ?", syncBatchMenu.Id).Count(&buttonCount).Error; err != nil || buttonCount != 3 {
		t.Fatalf("sync batch button count=%d err=%v", buttonCount, err)
	}
	var forbiddenSyncBatchButtons int64
	if err := db.Model(&model.SysMenuButton{}).Where("menu_id = ? AND event_action IN ?", syncBatchMenu.Id, []string{"run", "cancel", "delete"}).Count(&forbiddenSyncBatchButtons).Error; err != nil || forbiddenSyncBatchButtons != 0 {
		t.Fatalf("forbidden sync batch buttons=%d err=%v", forbiddenSyncBatchButtons, err)
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
