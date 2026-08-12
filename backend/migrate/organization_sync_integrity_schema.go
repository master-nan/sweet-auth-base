package main

import (
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

// migrateOrganizationSyncIntegritySchema 在 IntegrationExecution 建表后补齐 Organization 业务追踪完整性。
func migrateOrganizationSyncIntegritySchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.OrgSyncBatch{}, &model.OrgSyncRecord{}); err != nil {
			return fmt.Errorf("auto migrate organization sync integrity schema: %w", err)
		}

		// 旧开发数据使用过早期动作编码；只迁移语义明确的一一对应值。
		aliases := map[string]string{
			"insert":            model.OrgSyncRecordActionCreate,
			"delete_to_disable": model.OrgSyncRecordActionDisable,
			"skip":              model.OrgSyncRecordActionNoop,
			"no_change":         model.OrgSyncRecordActionNoop,
		}
		for oldValue, newValue := range aliases {
			if err := tx.Model(&model.OrgSyncRecord{}).Where("action = ?", oldValue).Update("action", newValue).Error; err != nil {
				return fmt.Errorf("migrate organization sync action %s: %w", oldValue, err)
			}
		}
		if tx.Migrator().HasTable(&model.SysDictItem{}) {
			if err := tx.Model(&model.SysDictItem{}).Where("item_code = ?", "org_sync_action_insert").Updates(map[string]any{"item_code": "org_sync_action_create", "item_value": model.OrgSyncRecordActionCreate}).Error; err != nil {
				return fmt.Errorf("migrate organization sync create dictionary item: %w", err)
			}
			if err := tx.Model(&model.SysDictItem{}).Where("item_code = ?", "org_sync_action_no_change").Updates(map[string]any{"item_code": "org_sync_action_noop", "item_value": model.OrgSyncRecordActionNoop}).Error; err != nil {
				return fmt.Errorf("migrate organization sync noop dictionary item: %w", err)
			}
			if err := tx.Where("item_code IN ?", []string{"org_sync_action_delete_to_disable", "org_sync_action_skip"}).Delete(&model.SysDictItem{}).Error; err != nil {
				return fmt.Errorf("retire legacy organization sync dictionary items: %w", err)
			}
		}

		indexes := []struct{ name, sql string }{
			{"uni_org_sync_batch_execution", `CREATE UNIQUE INDEX IF NOT EXISTS uni_org_sync_batch_execution ON org_sync_batch (execution_id) WHERE execution_id IS NOT NULL AND gmt_delete IS NULL`},
			{"uni_org_sync_record_business", `CREATE UNIQUE INDEX IF NOT EXISTS uni_org_sync_record_business ON org_sync_record (batch_id, object_type, source_id) WHERE source_id <> '' AND gmt_delete IS NULL`},
		}
		for _, index := range indexes {
			if err := tx.Exec(index.sql).Error; err != nil {
				return fmt.Errorf("create %s: %w", index.name, err)
			}
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
		}

		checks := []postgresCheckConstraint{
			{model: &model.OrgStructure{}, name: "chk_org_structure_type", expression: "structure_type IN ('management','legal')"},
			{model: &model.OrgSyncBatch{}, name: "chk_org_sync_batch_counts", expression: "total_count >= 0 AND success_count >= 0 AND failed_count >= 0 AND skipped_count >= 0"},
			{model: &model.OrgSyncBatch{}, name: "chk_org_sync_batch_status", expression: "status IN ('pending','processing','success','failed','dependency_waiting','ignored')"},
			{model: &model.OrgSyncRecord{}, name: "chk_org_sync_record_action", expression: "action IN ('create','update','disable','close','noop','error','deferred')"},
			{model: &model.OrgSyncRecord{}, name: "chk_org_sync_record_status", expression: "status IN ('pending','processing','success','failed','dependency_waiting','ignored')"},
			{model: &model.OrgSyncRecord{}, name: "chk_org_sync_record_identity", expression: "btrim(object_type) <> '' AND retry_count >= 0 AND ((btrim(source_id) <> '') OR action = 'error')"},
		}
		for _, check := range checks {
			if err := createPostgresCheckConstraint(tx, check); err != nil {
				return err
			}
		}
		return createPostgresForeignKeyConstraint(tx, postgresForeignKeyConstraint{
			model: &model.OrgSyncBatch{}, name: "fk_org_sync_batch_execution", columns: []string{"execution_id"},
			referenceModel: &model.IntegrationExecution{}, referenceFields: []string{"id"},
		})
	})
}
