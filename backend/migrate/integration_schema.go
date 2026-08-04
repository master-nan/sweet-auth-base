package main

import (
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

func migrateIntegrationConfigurationSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.ExternalSystem{}); err != nil {
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
		}
		for _, check := range checks {
			if err := createPostgresCheckConstraint(tx, check); err != nil {
				return err
			}
		}
		return nil
	})
}
