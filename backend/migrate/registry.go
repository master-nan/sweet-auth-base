package main

import (
	"backend/config"
	"backend/internal/utils"
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

type migrationStep struct {
	name string
	run  func(*gorm.DB) error
}

type seedStep struct {
	name string
	run  func(*gorm.DB, *config.Server, *utils.Snowflake) error
}

// migrationSteps 是 schema 变更和 migrate 阶段幂等回填的唯一注册点。
func migrationSteps() []migrationStep {
	return []migrationStep{
		{name: "auto_migrate_core_schema", run: autoMigrateCoreSchema},
		{name: "metadata_value_contract", run: migrateMetadataValueContract},
		{name: "backfill_sys_table_index_field_sequence", run: backfillSysTableIndexFieldSequence},
		{name: "query_scheme_schema", run: migrateQuerySchemeSchema},
		{name: "integration_configuration_schema", run: migrateIntegrationConfigurationSchema},
		{name: "integration_runtime_schema", run: migrateIntegrationRuntimeSchema},
		{name: "integration_sync_schema", run: migrateIntegrationSyncSchema},
		{name: "organization_sync_integrity_schema", run: migrateOrganizationSyncIntegritySchema},
		{name: "data_permission_domain_schema", run: migrateDataPermissionSchema},
		{name: "remove_legacy_data_permission_schema", run: removeLegacyDataPermissionSchema},
		{name: "ensure_sys_menu_option_text", run: ensureSysMenuOptionText},
		{name: "backfill_sys_menu_page_binding", run: backfillSysMenuPageBinding},
		{name: "canonical_runtime_contract", run: migrateCanonicalRuntimeContract},
		{name: "organization_database_comments", run: applyOrganizationDatabaseComments},
	}
}

func runMigrationSteps(db *gorm.DB, steps []migrationStep) error {
	for _, step := range steps {
		if err := step.run(db); err != nil {
			return fmt.Errorf("migration step %s: %w", step.name, err)
		}
	}
	return nil
}

func autoMigrateCoreSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.SysConfigure{},
		&model.SysTable{},
		&model.SysTableField{},
		&model.SysTableRelation{},
		&model.SysTableIndex{},
		&model.SysTableIndexField{},
		&model.SysDict{},
		&model.SysDictItem{},
		&model.AccessLog{},
		&model.LoginLog{},
		&model.SysUser{},
		&model.SysUserRole{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysMenuButtonTemplate{},
		&model.SysRole{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgStructure{},
		&model.OrgStructureNode{},
		&model.OrgPosition{},
		&model.OrgEmployee{},
		&model.OrgAssignment{},
		&model.OrgSyncBatch{},
		&model.OrgSyncRecord{},
		&model.ReportDefinition{},
		&model.ReportDefinitionVersion{},
		&model.ReportExecutionLog{},
		&model.Application{},
		&model.SmsTemplate{},
		&model.SmsLog{},
		&model.File{},
		&model.FileChunk{},
		&model.CasbinRule{},
	)
}

func seedAllData(db *gorm.DB, cfg *config.Server, sf *utils.Snowflake) error {
	return runSeedSteps(db, cfg, sf, platformSeedSteps())
}

func baseSeedSteps() []seedStep {
	return []seedStep{
		{name: "sys_configure", run: func(db *gorm.DB, _ *config.Server, _ *utils.Snowflake) error {
			return seedConfigure(db)
		}},
		{name: "sys_dict", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			return seedDicts(db, sf)
		}},
		{name: "application", run: func(db *gorm.DB, _ *config.Server, _ *utils.Snowflake) error {
			return seedApplication(db)
		}},
		{name: "admin_user", run: func(db *gorm.DB, cfg *config.Server, _ *utils.Snowflake) error {
			return seedAdminUser(db, cfg)
		}},
		{name: "sys_menu_role_button", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			return seedMenusAndRole(db, sf)
		}},
		{name: "lowcode_button_templates", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			return seedLowCodeMenuButtonTemplates(db, sf)
		}},
		{name: "menu_button_defaults_repair", run: func(db *gorm.DB, _ *config.Server, _ *utils.Snowflake) error {
			return repairSeededMenuButtonDefaults(db)
		}},
	}
}

// platformSeedSteps 是平台 Seed 数据的唯一注册点。
// sys_dict 必须位于菜单和按钮 Seed 之前；sys_table/sys_table_field 必须位于 schema 和菜单 Seed 之后，
// 以便元数据发现全部已迁移表。
func platformSeedSteps() []seedStep {
	steps := append([]seedStep{}, baseSeedSteps()...)
	steps = append(steps,
		seedStep{name: "sys_table_and_field_metadata", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			if err := seedSystemTableMetadata(db, sf); err != nil {
				return err
			}
			return seedOrganizationFoundation(db, sf)
		}},
		seedStep{name: "integration_configuration_foundation", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			return seedIntegrationConfigurationFoundation(db, sf)
		}},
		seedStep{name: "query_scheme_foundation", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			return seedQuerySchemeFoundation(db, sf)
		}},
		seedStep{name: "data_permission_dictionary_and_metadata", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			return seedDataPermissionFoundation(db, sf)
		}},
		seedStep{name: "sys_table_relation_metadata", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			return seedSystemTableRelations(db, sf)
		}},
		seedStep{name: "functional_permission_projection", run: func(db *gorm.DB, _ *config.Server, sf *utils.Snowflake) error {
			return seedFunctionalPermissionProjection(db, sf)
		}},
		seedStep{name: "rebuildable_cache_flush", run: func(_ *gorm.DB, cfg *config.Server, _ *utils.Snowflake) error {
			return flushMigrationCaches(cfg)
		}},
	)
	return steps
}

func runSeedSteps(db *gorm.DB, cfg *config.Server, sf *utils.Snowflake, steps []seedStep) error {
	for _, step := range steps {
		if err := step.run(db, cfg, sf); err != nil {
			return fmt.Errorf("seed step %s: %w", step.name, err)
		}
	}
	return nil
}
