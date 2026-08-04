package main

import (
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

func migrateIntegrationConfigurationSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.ExternalSystem{}, &model.InterfaceDefinition{}); err != nil {
			return fmt.Errorf("auto migrate external system: %w", err)
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
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
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_timeout", expression: "timeout_seconds BETWEEN 1 AND 300"},
			{model: &model.InterfaceDefinition{}, name: "chk_integration_interface_response_limit", expression: "response_limit BETWEEN 1024 AND 104857600"},
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
		return nil
	})
}
