package main

import (
	"backend/internal/database"
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

func migrateNotificationStandardBaseFields(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return db.AutoMigrate(&model.Notification{}, &model.NotificationRecipient{})
	}
	if !db.Migrator().HasTable("notification") {
		return nil
	}
	if err := db.Exec(`DROP INDEX IF EXISTS idx_notification_created`).Error; err != nil {
		return fmt.Errorf("drop legacy notification time index: %w", err)
	}
	hasCreatedAt := db.Migrator().HasColumn("notification", "created_at")
	hasGmtCreate := db.Migrator().HasColumn("notification", "gmt_create")
	if hasCreatedAt && !hasGmtCreate {
		if err := db.Exec(`ALTER TABLE notification RENAME COLUMN created_at TO gmt_create`).Error; err != nil {
			return fmt.Errorf("rename notification created time: %w", err)
		}
		hasCreatedAt = false
		hasGmtCreate = true
	}
	if !hasGmtCreate {
		if err := db.Exec(`ALTER TABLE notification ADD COLUMN gmt_create timestamptz`).Error; err != nil {
			return fmt.Errorf("add notification create time: %w", err)
		}
	}
	if hasCreatedAt {
		if err := db.Exec(`UPDATE notification SET gmt_create = COALESCE(gmt_create, created_at, CURRENT_TIMESTAMP)`).Error; err != nil {
			return fmt.Errorf("backfill notification create time: %w", err)
		}
		if err := db.Exec(`ALTER TABLE notification DROP COLUMN created_at`).Error; err != nil {
			return fmt.Errorf("drop legacy notification create time: %w", err)
		}
	}

	statements := []string{
		`UPDATE notification SET gmt_create = CURRENT_TIMESTAMP WHERE gmt_create IS NULL`,
		`ALTER TABLE notification ALTER COLUMN gmt_create SET DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE notification ALTER COLUMN gmt_create SET NOT NULL`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS create_user bigint`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS create_name varchar(128)`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS gmt_modify timestamptz`,
		`UPDATE notification SET gmt_modify = gmt_create WHERE gmt_modify IS NULL`,
		`ALTER TABLE notification ALTER COLUMN gmt_modify SET DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE notification ALTER COLUMN gmt_modify SET NOT NULL`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS modify_user bigint`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS modify_name varchar(128)`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS gmt_delete timestamptz`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS delete_user bigint`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS delete_name varchar(128)`,
		`ALTER TABLE notification ADD COLUMN IF NOT EXISTS state boolean`,
		`UPDATE notification SET state = TRUE WHERE state IS NULL`,
		`ALTER TABLE notification ALTER COLUMN state SET DEFAULT TRUE`,
		`ALTER TABLE notification ALTER COLUMN state SET NOT NULL`,
		`DROP INDEX IF EXISTS ux_notification_source_dedup`,
		`CREATE UNIQUE INDEX ux_notification_source_dedup
			ON notification (source_module, dedup_key)
			WHERE dedup_key IS NOT NULL AND dedup_key <> '' AND gmt_delete IS NULL`,
		`CREATE INDEX idx_notification_created
			ON notification (gmt_create DESC, id DESC)
			WHERE gmt_delete IS NULL AND state = TRUE`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("migrate notification standard base fields: %w", err)
		}
	}
	return nil
}

func syncMetadataColumnComments(db *gorm.DB) error {
	_, err := database.SyncMetadataColumnComments(db, "")
	return err
}
