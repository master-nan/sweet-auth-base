package main

import (
	"backend/config"
	migrationstate "backend/internal/migration"
	"backend/internal/utils"
	"backend/model"
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type migrationStep struct {
	version  int64
	name     string
	contract string
	checksum string
	run      func(*gorm.DB) error
}

type seedStep struct {
	name string
	run  func(*gorm.DB, *config.Server, *utils.Snowflake) error
}

// migrationSteps 是 schema 变更和 migrate 阶段幂等回填的唯一注册点。
func migrationSteps() []migrationStep {
	runners := []func(*gorm.DB) error{
		autoMigrateCoreSchema,
		migrateMetadataValueContract,
		backfillSysTableIndexFieldSequence,
		migrateQuerySchemeSchema,
		migrateIntegrationConfigurationSchema,
		migrateIntegrationRuntimeSchema,
		migrateIntegrationSyncSchema,
		migrateOrganizationSyncIntegritySchema,
		migrateDataPermissionSchema,
		removeLegacyDataPermissionSchema,
		ensureSysMenuOptionText,
		backfillSysMenuPageBinding,
		migrateCanonicalRuntimeContract,
		applyOrganizationDatabaseComments,
		ensureAccessLogOperationalIndexes,
	}
	definitions := migrationstate.Catalog()
	if len(definitions) != len(runners) {
		panic(fmt.Sprintf("migration catalog has %d definitions for %d runners", len(definitions), len(runners)))
	}
	steps := make([]migrationStep, 0, len(definitions))
	for i, definition := range definitions {
		steps = append(steps, migrationStep{
			version:  definition.Version,
			name:     definition.Key,
			contract: definition.Contract,
			checksum: definition.Checksum,
			run:      runners[i],
		})
	}
	return steps
}

func ensureAccessLogOperationalIndexes(db *gorm.DB) error {
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS idx_access_log_time ON access_log (gmt_create DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_access_log_action_time ON access_log (action, gmt_create DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_access_log_resource_time ON access_log (resource_type, resource_code, resource_id, gmt_create DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_access_log_success_time ON access_log (success, gmt_create DESC)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("ensure access log operational index: %w", err)
		}
	}
	return nil
}

func runMigrationSteps(db *gorm.DB, steps []migrationStep) error {
	return runMigrationStepsWithMode(db, steps, migrationstate.ManagedTables(), false)
}

func runMigrationStepsWithMode(db *gorm.DB, steps []migrationStep, managedTables []string, adopt bool) error {
	if db.Dialector.Name() != "postgres" {
		return runUntrackedMigrationSteps(db, steps)
	}
	definitions := definitionsForSteps(steps)
	if err := migrationstate.ValidateCatalog(definitions); err != nil {
		return err
	}
	ctx := context.Background()
	return migrationstate.WithAdvisoryLock(ctx, db, func(lockedDB *gorm.DB) error {
		conn := lockedDB.Statement.ConnPool
		existingTables, err := migrationstate.ExistingManagedTables(ctx, conn, managedTables)
		if err != nil {
			return err
		}
		entries, ledgerExists, err := migrationstate.LoadLedger(ctx, conn)
		if err != nil {
			return err
		}
		if len(entries) == 0 && len(existingTables) > 0 && !adopt {
			return fmt.Errorf(
				"database contains managed tables but has no migration history; inspect it, then run explicit `migrate adopt` (found: %s)",
				strings.Join(existingTables, ", "),
			)
		}
		if !ledgerExists {
			if err := migrationstate.EnsureLedger(ctx, conn); err != nil {
				return err
			}
		}

		if err := migrationstate.ValidateLedger(entries, definitions, false); err != nil {
			return err
		}
		for _, step := range steps[len(entries):] {
			transactionDB := lockedDB.Session(&gorm.Session{NewDB: true, DisableNestedTransaction: true})
			if err := transactionDB.Transaction(func(tx *gorm.DB) error {
				if err := step.run(tx); err != nil {
					return err
				}
				if err := tx.Exec(`
INSERT INTO schema_migration (version, "key", checksum, applied_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP)
`, step.version, step.name, step.checksum).Error; err != nil {
					return fmt.Errorf("record migration ledger: %w", err)
				}
				return nil
			}); err != nil {
				return fmt.Errorf("migration step %d/%s: %w", step.version, step.name, err)
			}
		}
		return nil
	})
}

func definitionsForSteps(steps []migrationStep) []migrationstate.Definition {
	definitions := make([]migrationstate.Definition, 0, len(steps))
	for _, step := range steps {
		definitions = append(definitions, migrationstate.Definition{
			Version:  step.version,
			Key:      step.name,
			Contract: step.contract,
			Checksum: step.checksum,
		})
	}
	return definitions
}

func runUntrackedMigrationSteps(db *gorm.DB, steps []migrationStep) error {
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
	return migrationstate.WithAdvisoryLock(context.Background(), db, func(lockedDB *gorm.DB) error {
		return runSeedSteps(lockedDB, cfg, sf, platformSeedSteps())
	})
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
