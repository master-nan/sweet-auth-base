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
		migrateProductWalkthroughCorrections,
		migrateNotificationCenterSchema,
		migrateOrganizationSourceCodeIndexes,
		migrateCanonicalTimeAndIDContract,
		migrateIntegrationReferenceIntegritySchema,
		migrateUserSessionSchema,
		migrateUserSessionAuditFields,
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

func migrateUserSessionSchema(db *gorm.DB) error {
	return db.AutoMigrate(&model.SysUserSession{})
}

func migrateUserSessionAuditFields(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.SysUserSession{}); err != nil {
		return err
	}
	sessionTable := db.NamingStrategy.TableName("SysUserSession")
	userTable := db.NamingStrategy.TableName("SysUser")
	statement := fmt.Sprintf(`UPDATE "%s"
		SET "user_name_snapshot" = COALESCE((
			SELECT "user_name" FROM "%s"
			WHERE "%s"."id" = "%s"."user_id"
		), '')
		WHERE "user_name_snapshot" = ''`, sessionTable, userTable, userTable, sessionTable)
	return db.Exec(statement).Error
}

func migrateNotificationCenterSchema(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return db.AutoMigrate(&model.Notification{}, &model.NotificationRecipient{})
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS notification (
			id bigint PRIMARY KEY,
			category varchar(24) NOT NULL,
			level varchar(16) NOT NULL,
			title varchar(160) NOT NULL,
			content text NOT NULL,
			source_module varchar(64) NOT NULL,
			source_type varchar(64) NOT NULL,
			source_id varchar(128) NOT NULL DEFAULT '',
			action_menu_name varchar(32) NOT NULL DEFAULT '',
			action_path varchar(512) NOT NULL DEFAULT '',
			dedup_key varchar(128),
			created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT chk_notification_category CHECK (category IN ('SYSTEM','BUSINESS','TASK','REMINDER','SECURITY','INTEGRATION')),
			CONSTRAINT chk_notification_level CHECK (level IN ('INFO','SUCCESS','WARNING','ERROR')),
			CONSTRAINT chk_notification_title CHECK (char_length(btrim(title)) BETWEEN 1 AND 160),
			CONSTRAINT chk_notification_content CHECK (
				char_length(content) BETWEEN 1 AND 4000 AND octet_length(content) <= 16384
			),
			CONSTRAINT chk_notification_source CHECK (
				char_length(btrim(source_module)) BETWEEN 1 AND 64
				AND char_length(btrim(source_type)) BETWEEN 1 AND 64
			),
			CONSTRAINT chk_notification_action CHECK (
				action_path = '' OR action_menu_name <> ''
			)
		)`,
		`CREATE TABLE IF NOT EXISTS notification_recipient (
			notification_id bigint NOT NULL,
			user_id bigint NOT NULL,
			read_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (notification_id, user_id),
			CONSTRAINT fk_notification_recipient_notification
				FOREIGN KEY (notification_id) REFERENCES notification(id) ON DELETE RESTRICT,
			CONSTRAINT fk_notification_recipient_user
				FOREIGN KEY (user_id) REFERENCES sys_user(id) ON DELETE RESTRICT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_notification_source_dedup
			ON notification (source_module, dedup_key)
			WHERE dedup_key IS NOT NULL AND dedup_key <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_notification_created
			ON notification (created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_recipient_user_created
			ON notification_recipient (user_id, created_at DESC, notification_id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_recipient_user_unread
			ON notification_recipient (user_id, created_at DESC, notification_id DESC)
			WHERE read_at IS NULL`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("migrate notification center schema: %w", err)
		}
	}
	return nil
}

func migrateProductWalkthroughCorrections(db *gorm.DB) error {
	var menuIDs []int
	if err := db.Model(&model.SysMenu{}).
		Where("name = ?", "report_v2_workbench").
		Pluck("id", &menuIDs).Error; err != nil {
		return fmt.Errorf("query obsolete report workbench menu: %w", err)
	}
	if len(menuIDs) > 0 {
		var buttonIDs []int
		if err := db.Model(&model.SysMenuButton{}).
			Where("menu_id IN ?", menuIDs).
			Pluck("id", &buttonIDs).Error; err != nil {
			return fmt.Errorf("query obsolete report workbench buttons: %w", err)
		}
		if len(buttonIDs) > 0 {
			if err := db.Where("button_id IN ?", buttonIDs).Delete(&model.SysRoleMenuButton{}).Error; err != nil {
				return fmt.Errorf("delete obsolete report workbench button grants: %w", err)
			}
		}
		if err := db.Where("menu_id IN ?", menuIDs).Delete(&model.SysRoleMenu{}).Error; err != nil {
			return fmt.Errorf("delete obsolete report workbench menu grants: %w", err)
		}
		if err := db.Where("menu_id IN ?", menuIDs).Delete(&model.SysMenuButton{}).Error; err != nil {
			return fmt.Errorf("delete obsolete report workbench buttons: %w", err)
		}
		if err := db.Where("id IN ?", menuIDs).Delete(&model.SysMenu{}).Error; err != nil {
			return fmt.Errorf("delete obsolete report workbench menu: %w", err)
		}
		if err := rebuildFunctionalPermissionPolicies(db); err != nil {
			return fmt.Errorf("rebuild permissions after report workbench cleanup: %w", err)
		}
	}
	userTables := db.Model(&model.SysTable{}).Select("id").Where("table_code = ?", "sys_user")
	if err := db.Model(&model.SysTableField{}).
		Where("table_id IN (?) AND field_code = ?", userTables, "gmt_last_login").
		Where("is_insert_show = ? OR is_update_show = ?", true, true).
		Updates(map[string]interface{}{
			"is_insert_show": false,
			"is_update_show": false,
			"gmt_modify":     model.Now(),
		}).Error; err != nil {
		return fmt.Errorf("hide last login from user mutations: %w", err)
	}
	return nil
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
		// 每一步Schema变更和Ledger写入处于同一事务；失败时两者一起回滚。
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
