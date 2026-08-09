package main

import (
	"backend/internal/integration"
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

func migrateIntegrationConfigurationSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.ExternalSystem{}, &model.Credential{}, &model.RetryPolicy{}, &model.InterfaceDefinition{}); err != nil {
			return fmt.Errorf("auto migrate external system: %w", err)
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
		}
		limits := integration.RuntimeLimits()
		if err := tx.Exec(`
			UPDATE integration_interface_definition AS definition
			SET status = 'disabled', state = FALSE, revision = revision + 1, gmt_modify = CURRENT_TIMESTAMP
			WHERE definition.gmt_delete IS NULL
			  AND definition.status = 'enabled'
			  AND definition.retry_policy_id IS NOT NULL
			  AND NOT EXISTS (
				SELECT 1 FROM integration_retry_policy AS policy
				WHERE policy.id = definition.retry_policy_id
				  AND policy.status = 'enabled'
				  AND policy.gmt_delete IS NULL
			  )
		`).Error; err != nil {
			return fmt.Errorf("disable interface definitions with invalid retry policy: %w", err)
		}
		if err := tx.Exec(`
			UPDATE integration_interface_definition
			SET status = 'disabled', state = FALSE, revision = revision + 1, gmt_modify = CURRENT_TIMESTAMP
			WHERE gmt_delete IS NULL
			  AND status = 'enabled'
			  AND (timeout_seconds > ? OR response_limit > ?)
		`, int(limits.MaxRequestTimeout.Seconds()), limits.MaxResponseBytes).Error; err != nil {
			return fmt.Errorf("disable runtime-incompatible interface definitions: %w", err)
		}
		for _, legacyConstraint := range []string{"chk_integration_interface_timeout", "chk_integration_interface_response_limit"} {
			if err := tx.Exec(fmt.Sprintf(`ALTER TABLE integration_interface_definition DROP CONSTRAINT IF EXISTS %s`, legacyConstraint)).Error; err != nil {
				return fmt.Errorf("drop legacy interface runtime constraint %s: %w", legacyConstraint, err)
			}
		}
		checks := []postgresCheckConstraint{
			{
				model:      &model.ExternalSystem{},
				name:       "chk_integration_external_system_status",
				expression: "status IN ('draft','enabled','disabled')",
			},
			{
				model:      &model.ExternalSystem{},
				name:       "chk_integration_external_system_type",
				expression: "system_type IN ('hr','erp','tms','wms','other')",
			},
			{
				model:      &model.ExternalSystem{},
				name:       "chk_integration_external_system_revision",
				expression: "revision > 0",
			},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_status", expression: "status IN ('draft','enabled','disabled')"},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_protocol", expression: "protocol IN ('http','https')"},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_method", expression: "http_method IN ('GET','POST','PUT','PATCH','DELETE')"},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_version", expression: "version > 0"},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_revision", expression: "revision > 0"},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_enabled_timeout", expression: fmt.Sprintf("timeout_seconds >= 1 AND (status <> 'enabled' OR timeout_seconds <= %d)", int(limits.MaxRequestTimeout.Seconds()))},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_enabled_response_limit", expression: fmt.Sprintf("response_limit >= %d AND (status <> 'enabled' OR response_limit <= %d)", limits.MinResponseBytes, limits.MaxResponseBytes)},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_input_contract", expression: "jsonb_typeof(input_contract) = 'object' AND input_contract ? 'version' AND input_contract->>'version' = '1' AND input_contract ? 'parameters' AND jsonb_typeof(input_contract->'parameters') = 'array'"},
			{model: &model.Credential{}, name: "chk_integration_credential_type", expression: "credential_type IN ('basic','api_key','bearer_token','oauth_client')"},
			{model: &model.Credential{}, name: "chk_integration_credential_status", expression: "status IN ('draft','active','disabled','revoked')"},
			{model: &model.Credential{}, name: "chk_integration_credential_version", expression: "version > 0"},
			{model: &model.Credential{}, name: "chk_integration_credential_revision", expression: "revision > 0"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_status", expression: "status IN ('draft','enabled','disabled')"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_version", expression: "version > 0"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_attempts", expression: "max_attempts BETWEEN 1 AND 10"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_initial_delay", expression: "initial_delay_ms BETWEEN 1000 AND 3600000"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_max_delay", expression: "max_delay_ms >= initial_delay_ms AND max_delay_ms <= 86400000"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_backoff", expression: "(backoff_type = 'fixed' AND backoff_multiplier = 1) OR (backoff_type = 'exponential' AND backoff_multiplier BETWEEN 1.1 AND 4)"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_jitter", expression: "(jitter_type = 'none' AND jitter_ratio = 0) OR (jitter_type = 'full' AND jitter_ratio = 1)"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_window", expression: "retry_window_ms BETWEEN 60000 AND 604800000 AND retry_window_ms >= initial_delay_ms"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_error_categories", expression: "jsonb_typeof(retryable_error_categories) = 'array' AND retryable_error_categories <@ '[\"network\",\"timeout\",\"remote\"]'::jsonb"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_http_statuses", expression: "jsonb_typeof(retryable_http_statuses) = 'array' AND retryable_http_statuses <@ '[429,502,503,504]'::jsonb"},
			{model: &model.RetryPolicy{}, name: "chk_integration_retry_policy_revision", expression: "revision > 0"},
		}
		for _, check := range checks {
			if err := createPostgresCheckConstraint(tx, check); err != nil {
				return err
			}
		}
		if err := tx.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS uni_integration_interface_enabled
			ON integration_interface_definition (external_system_id, interface_code)
			WHERE status = 'enabled' AND gmt_delete IS NULL
		`).Error; err != nil {
			return fmt.Errorf("create enabled interface version index: %w", err)
		}
		if err := tx.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS uni_integration_retry_policy_enabled
			ON integration_retry_policy (policy_code)
			WHERE status = 'enabled' AND gmt_delete IS NULL
		`).Error; err != nil {
			return fmt.Errorf("create enabled retry policy version index: %w", err)
		}
		if err := createPostgresForeignKeyConstraint(tx, postgresForeignKeyConstraint{
			model: &model.InterfaceDefinition{}, name: "fk_integration_interface_retry_policy",
			columns: []string{"retry_policy_id"}, referenceModel: &model.RetryPolicy{}, referenceFields: []string{"id"},
		}); err != nil {
			return err
		}
		return nil
	})
}
