package main

import (
	"backend/internal/integration"
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

func migrateIntegrationSyncSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.IntegrationSyncTask{}, &model.IntegrationSyncBatch{}, &model.IntegrationExecution{}); err != nil {
			return fmt.Errorf("auto migrate integration sync schema: %w", err)
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
		}
		checks := []postgresCheckConstraint{
			{model: &model.IntegrationSyncTask{}, name: "chk_integration_sync_task_status", expression: "status IN ('draft','enabled','disabled')"},
			{model: &model.IntegrationSyncTask{}, name: "chk_integration_sync_task_version", expression: "version > 0"},
			{model: &model.IntegrationSyncTask{}, name: "chk_integration_sync_task_revision", expression: "revision > 0"},
			{model: &model.IntegrationSyncTask{}, name: "chk_integration_sync_task_consumer", expression: "btrim(consumer_code) <> '' AND consumer_version > 0"},
			{model: &model.IntegrationSyncTask{}, name: "chk_integration_sync_task_schedule", expression: "(schedule_type = 'none' AND cron_expression = '' AND next_scheduled_at IS NULL) OR (schedule_type = 'cron' AND btrim(cron_expression) <> '' AND (status <> 'enabled' OR next_scheduled_at IS NOT NULL))"},
			{model: &model.IntegrationSyncTask{}, name: "chk_integration_sync_task_checkpoint", expression: "(checkpoint_mode = 'none' AND initial_checkpoint_at IS NULL AND checkpoint_at IS NULL AND lookback_seconds = 0 AND window_slice_seconds = 0) OR (checkpoint_mode = 'timestamp' AND initial_checkpoint_at IS NOT NULL AND lookback_seconds BETWEEN 0 AND 604800 AND window_slice_seconds BETWEEN 60 AND 604800 AND (status <> 'enabled' OR checkpoint_at IS NOT NULL))"},
			{model: &model.IntegrationSyncTask{}, name: "chk_integration_sync_task_input_plan", expression: fmt.Sprintf("jsonb_typeof(input_plan) = 'object' AND input_plan ? 'version' AND input_plan->>'version' = '%d' AND input_plan ? 'static_input' AND jsonb_typeof(input_plan->'static_input') = 'object'", integration.SyncExecutionInputPlanVersion)},
			{model: &model.IntegrationSyncBatch{}, name: "chk_integration_sync_batch_status", expression: "status IN ('created','running','succeeded','failed')"},
			{model: &model.IntegrationSyncBatch{}, name: "chk_integration_sync_batch_trigger", expression: "(trigger_type = 'manual' AND scheduled_for IS NULL) OR (trigger_type = 'scheduled' AND scheduled_for IS NOT NULL)"},
			{model: &model.IntegrationSyncBatch{}, name: "chk_integration_sync_batch_identity", expression: "btrim(batch_no) <> '' AND btrim(trigger_key) <> '' AND task_version > 0 AND task_revision > 0 AND interface_version > 0 AND consumer_version > 0"},
			{model: &model.IntegrationSyncBatch{}, name: "chk_integration_sync_batch_window", expression: "(window_start IS NULL AND window_end IS NULL) OR (window_start IS NOT NULL AND window_end IS NOT NULL AND window_end >= window_start)"},
			{model: &model.IntegrationSyncBatch{}, name: "chk_integration_sync_batch_checkpoint", expression: "checkpoint_mode IN ('none','timestamp') AND ((checkpoint_mode = 'none' AND checkpoint_before IS NULL AND checkpoint_after IS NULL AND lookback_seconds = 0 AND window_slice_seconds = 0) OR (checkpoint_mode = 'timestamp' AND checkpoint_before IS NOT NULL AND lookback_seconds BETWEEN 0 AND 604800 AND window_slice_seconds BETWEEN 60 AND 604800))"},
			{model: &model.IntegrationSyncBatch{}, name: "chk_integration_sync_batch_counts", expression: "planned_slice_count >= 0 AND current_slice_no >= 0 AND execution_count >= 0 AND technical_success_count >= 0 AND technical_failed_count >= 0 AND business_success_count >= 0 AND business_failed_count >= 0"},
			{model: &model.IntegrationSyncBatch{}, name: "chk_integration_sync_batch_revision", expression: "revision > 0"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_sync_source", expression: "(sync_batch_id IS NULL AND sync_slice_no IS NULL AND sync_window_start IS NULL AND sync_window_end IS NULL AND sync_consumer_code = '' AND sync_consumer_version IS NULL) OR (sync_batch_id IS NOT NULL AND sync_slice_no >= 1 AND btrim(sync_consumer_code) <> '' AND sync_consumer_version > 0 AND ((sync_window_start IS NULL AND sync_window_end IS NULL) OR (sync_window_start IS NOT NULL AND sync_window_end IS NOT NULL AND sync_window_end > sync_window_start)))"},
		}
		for _, check := range checks {
			if err := createPostgresCheckConstraint(tx, check); err != nil {
				return err
			}
		}
		indexes := []struct{ name, sql string }{
			{"uni_integration_sync_task_enabled", `CREATE UNIQUE INDEX IF NOT EXISTS uni_integration_sync_task_enabled ON integration_sync_task (task_code) WHERE status = 'enabled' AND gmt_delete IS NULL`},
			{"idx_integration_sync_task_schedule_ready", `CREATE INDEX IF NOT EXISTS idx_integration_sync_task_schedule_ready ON integration_sync_task (status, next_scheduled_at, id)`},
			{"uni_integration_sync_batch_scheduled", `CREATE UNIQUE INDEX IF NOT EXISTS uni_integration_sync_batch_scheduled ON integration_sync_batch (task_code, scheduled_for) WHERE trigger_type = 'scheduled' AND gmt_delete IS NULL`},
			{"uni_integration_sync_batch_active", `CREATE UNIQUE INDEX IF NOT EXISTS uni_integration_sync_batch_active ON integration_sync_batch (task_code) WHERE status IN ('created','running') AND gmt_delete IS NULL`},
			{"idx_integration_sync_batch_task_created", `CREATE INDEX IF NOT EXISTS idx_integration_sync_batch_task_created ON integration_sync_batch (sync_task_id, gmt_create)`},
			{"idx_integration_sync_batch_status_created", `CREATE INDEX IF NOT EXISTS idx_integration_sync_batch_status_created ON integration_sync_batch (status, gmt_create)`},
			{"uni_integration_execution_sync_slice", `CREATE UNIQUE INDEX IF NOT EXISTS uni_integration_execution_sync_slice ON integration_execution (sync_batch_id, sync_slice_no) WHERE sync_batch_id IS NOT NULL AND gmt_delete IS NULL`},
			{"idx_integration_execution_sync_runtime", `CREATE INDEX IF NOT EXISTS idx_integration_execution_sync_runtime ON integration_execution (sync_batch_id, status, sync_slice_no) WHERE sync_batch_id IS NOT NULL`},
		}
		for _, index := range indexes {
			if err := tx.Exec(index.sql).Error; err != nil {
				return fmt.Errorf("create %s: %w", index.name, err)
			}
		}
		foreignKeys := []postgresForeignKeyConstraint{
			{model: &model.IntegrationSyncTask{}, name: "fk_integration_sync_task_system", columns: []string{"external_system_id"}, referenceModel: &model.ExternalSystem{}, referenceFields: []string{"id"}},
			{model: &model.IntegrationSyncTask{}, name: "fk_integration_sync_task_interface", columns: []string{"interface_definition_id"}, referenceModel: &model.InterfaceDefinition{}, referenceFields: []string{"id"}},
			{model: &model.IntegrationSyncBatch{}, name: "fk_integration_sync_batch_task", columns: []string{"sync_task_id"}, referenceModel: &model.IntegrationSyncTask{}, referenceFields: []string{"id"}},
			{model: &model.IntegrationExecution{}, name: "fk_integration_execution_sync_batch", columns: []string{"sync_batch_id"}, referenceModel: &model.IntegrationSyncBatch{}, referenceFields: []string{"id"}},
		}
		for _, foreignKey := range foreignKeys {
			if err := createPostgresForeignKeyConstraint(tx, foreignKey); err != nil {
				return err
			}
		}
		return nil
	})
}
