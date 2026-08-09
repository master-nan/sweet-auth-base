package main

import (
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

// migrateIntegrationRuntimeSchema 创建 Integration Runtime 基础持久化对象。
func migrateIntegrationRuntimeSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.IntegrationExecution{}, &model.IntegrationLog{}); err != nil {
			return fmt.Errorf("auto migrate integration runtime schema: %w", err)
		}
		// 历史 created/retry_waiting 记录没有可重建输入，不能伪造空快照后继续执行。
		if err := tx.Model(&model.IntegrationExecution{}).
			Where("input_snapshot_version = 0 AND status IN ?", []string{
				model.IntegrationExecutionStatusCreated,
				model.IntegrationExecutionStatusRetryWaiting,
			}).
			Updates(map[string]any{
				"status":           model.IntegrationExecutionStatusFailed,
				"error_category":   model.IntegrationErrorCategoryConfiguration,
				"result_summary":   "执行输入快照缺失，已停止历史待执行记录",
				"completed_at":     model.Now(),
				"lease_owner":      "",
				"lease_expires_at": nil,
				"revision":         gorm.Expr("revision + 1"),
			}).Error; err != nil {
			return fmt.Errorf("close legacy executions without input snapshot: %w", err)
		}
		if err := tx.Model(&model.IntegrationExecution{}).
			Where("retry_policy_snapshot_version = 0 AND status = ?", model.IntegrationExecutionStatusRetryWaiting).
			Updates(map[string]any{
				"status":           model.IntegrationExecutionStatusFailed,
				"error_category":   model.IntegrationErrorCategoryConfiguration,
				"result_summary":   "历史等待重试记录缺少冻结策略，已安全停止",
				"completed_at":     model.Now(),
				"next_run_at":      nil,
				"lease_owner":      "",
				"lease_expires_at": nil,
				"revision":         gorm.Expr("revision + 1"),
			}).Error; err != nil {
			return fmt.Errorf("close legacy retry executions without policy snapshot: %w", err)
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
		}
		if err := tx.Exec(`
			UPDATE integration_execution AS execution
			SET remote_idempotency_mode = definition.idempotency_mode,
			    remote_idempotency_header = definition.remote_idempotency_header,
			    remote_idempotency_key = CASE WHEN definition.idempotency_mode = 'remote_key_header' THEN execution.execution_no ELSE '' END
			FROM integration_interface_definition AS definition
			WHERE execution.interface_definition_id = definition.id
			  AND (btrim(COALESCE(execution.remote_idempotency_mode, '')) = ''
			       OR (execution.remote_idempotency_mode = 'none' AND definition.idempotency_mode <> 'none')
			       OR (execution.remote_idempotency_mode = 'remote_key_header' AND btrim(COALESCE(execution.remote_idempotency_key, '')) = ''))
		`).Error; err != nil {
			return fmt.Errorf("backfill execution idempotency contract: %w", err)
		}
		if err := tx.Exec(`
			UPDATE integration_execution AS execution
			SET retry_policy_snapshot = jsonb_set(
				jsonb_set(execution.retry_policy_snapshot, '{idempotency_mode}', to_jsonb(definition.idempotency_mode), true),
				'{remote_idempotency_header}', to_jsonb(definition.remote_idempotency_header), true
			)
			FROM integration_interface_definition AS definition
			WHERE execution.interface_definition_id = definition.id
			  AND execution.retry_policy_snapshot_version = 1
			  AND (NOT execution.retry_policy_snapshot ? 'idempotency_mode' OR NOT execution.retry_policy_snapshot ? 'remote_idempotency_header')
		`).Error; err != nil {
			return fmt.Errorf("backfill retry snapshot idempotency contract: %w", err)
		}
		if err := tx.Exec(`ALTER TABLE integration_execution DROP CONSTRAINT IF EXISTS chk_integration_execution_retry_snapshot_json`).Error; err != nil {
			return fmt.Errorf("refresh retry snapshot constraint: %w", err)
		}

		checks := []postgresCheckConstraint{
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_status", expression: "status IN ('created','running','retry_waiting','succeeded','failed','cancelled')"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_trigger", expression: "trigger_source IN ('manual','system_event','scheduled')"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_interface_version", expression: "interface_version > 0"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_idempotency_scope", expression: "btrim(idempotency_scope) <> ''"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_idempotency_key", expression: "btrim(idempotency_key) <> ''"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_revision", expression: "revision > 0"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_attempt", expression: "current_attempt >= 0"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_input_hash", expression: "input_hash ~ '^[0-9a-f]{64}$'"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_input_snapshot_json", expression: "jsonb_typeof(input_snapshot) = 'object' AND (input_snapshot_version = 0 OR (input_snapshot ? 'version' AND input_snapshot->>'version' = '1' AND input_snapshot ? 'path_params' AND jsonb_typeof(input_snapshot->'path_params') = 'object' AND input_snapshot ? 'query_params' AND jsonb_typeof(input_snapshot->'query_params') = 'object' AND input_snapshot ? 'headers' AND jsonb_typeof(input_snapshot->'headers') = 'object'))"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_input_snapshot_version", expression: "input_snapshot_version IN (0, 1)"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_input_snapshot_size", expression: "input_snapshot_size BETWEEN 0 AND 393216 AND (input_snapshot_version = 0 OR input_snapshot_size > 0)"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_retry_snapshot_version", expression: "retry_policy_snapshot_version IN (0, 1)"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_remote_idempotency", expression: "(remote_idempotency_mode IN ('none','safe_method','idempotent_method') AND remote_idempotency_header = '' AND remote_idempotency_key = '') OR (remote_idempotency_mode = 'remote_key_header' AND remote_idempotency_header = 'Idempotency-Key' AND btrim(remote_idempotency_key) <> '')"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_retry_snapshot_json", expression: "jsonb_typeof(retry_policy_snapshot) = 'object' AND ((retry_policy_snapshot_version = 0 AND retry_policy_snapshot = '{}'::jsonb AND retry_policy_id IS NULL) OR (retry_policy_snapshot_version = 1 AND retry_policy_id IS NOT NULL AND retry_policy_snapshot ? 'version' AND retry_policy_snapshot->>'version' = '1' AND retry_policy_snapshot ? 'policy_code' AND retry_policy_snapshot ? 'policy_version' AND retry_policy_snapshot ? 'idempotency_mode' AND retry_policy_snapshot ? 'remote_idempotency_header'))"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_result_size", expression: "result_size_bytes >= 0"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_http_status", expression: "result_http_status IS NULL OR result_http_status BETWEEN 100 AND 599"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_error_category", expression: "error_category = '' OR error_category IN ('configuration','credential','network','timeout','remote','response','business','concurrency','system')"},
			{model: &model.IntegrationExecution{}, name: "chk_integration_execution_lease", expression: "lease_expires_at IS NULL OR btrim(lease_owner) <> ''"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_attempt", expression: "attempt_no > 0"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_status", expression: "status IN ('running','succeeded','failed','cancelled')"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_duration", expression: "duration_ms >= 0"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_result_size", expression: "result_size_bytes >= 0"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_http_status", expression: "http_status IS NULL OR http_status BETWEEN 100 AND 599"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_error_category", expression: "error_category = '' OR error_category IN ('configuration','credential','network','timeout','remote','response','business','concurrency','system')"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_certainty", expression: "result_certainty IN ('confirmed','unknown')"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_time_range", expression: "ended_at IS NULL OR ended_at >= started_at"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_retry_delay", expression: "retry_delay_ms >= 0"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_retry_source", expression: "retry_after_source IN ('none','local','http_delta','http_date','ignored','invalid_fallback')"},
			{model: &model.IntegrationLog{}, name: "chk_integration_log_retry_schedule", expression: "(retryable = FALSE AND retry_scheduled_at IS NULL) OR (retryable = TRUE AND retry_scheduled_at IS NOT NULL AND retry_delay_ms > 0)"},
		}
		for _, check := range checks {
			if err := createPostgresCheckConstraint(tx, check); err != nil {
				return err
			}
		}

		foreignKeys := []postgresForeignKeyConstraint{
			{model: &model.IntegrationExecution{}, name: "fk_integration_execution_system", columns: []string{"external_system_id"}, referenceModel: &model.ExternalSystem{}, referenceFields: []string{"id"}},
			{model: &model.IntegrationExecution{}, name: "fk_integration_execution_interface", columns: []string{"interface_definition_id"}, referenceModel: &model.InterfaceDefinition{}, referenceFields: []string{"id"}},
			{model: &model.IntegrationExecution{}, name: "fk_integration_execution_retry_policy", columns: []string{"retry_policy_id"}, referenceModel: &model.RetryPolicy{}, referenceFields: []string{"id"}},
			{model: &model.IntegrationLog{}, name: "fk_integration_log_execution", columns: []string{"execution_id"}, referenceModel: &model.IntegrationExecution{}, referenceFields: []string{"id"}},
		}
		for _, foreignKey := range foreignKeys {
			if err := createPostgresForeignKeyConstraint(tx, foreignKey); err != nil {
				return err
			}
		}
		return nil
	})
}
