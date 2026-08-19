package main

import (
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

func migrateQuerySchemeSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.SysMenu{}, &model.QueryScheme{}, &model.QuerySchemeRole{}); err != nil {
			return fmt.Errorf("auto migrate query scheme: %w", err)
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
		}
		checks := []postgresCheckConstraint{
			{model: &model.SysMenu{}, name: "chk_sys_menu_query_scope_code", expression: "query_scope_code IS NULL OR query_scope_code ~ '^[a-z][a-z0-9_.-]{0,127}$'"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_type", expression: "scheme_type IN ('PERSONAL','PUBLIC','ROLE','PAGE_DEFAULT')"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_owner", expression: "(scheme_type = 'PERSONAL' AND owner_user_id IS NOT NULL) OR (scheme_type <> 'PERSONAL' AND owner_user_id IS NULL)"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_default_type", expression: "scheme_type IN ('PERSONAL','PAGE_DEFAULT') OR is_default = FALSE"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_personal_enabled", expression: "scheme_type <> 'PERSONAL' OR enabled = TRUE"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_revision", expression: "revision >= 1"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_schema_version", expression: "query_schema_version = 1"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_payload_object", expression: "jsonb_typeof(query_payload) = 'object'"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_payload_size", expression: "octet_length(query_payload::text) <= 32768"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_scope_code", expression: "scope_code ~ '^[a-z][a-z0-9_.-]{0,127}$'"},
			{model: &model.QueryScheme{}, name: "chk_query_scheme_name", expression: "char_length(btrim(name)) BETWEEN 1 AND 64"},
		}
		for _, check := range checks {
			if err := createPostgresCheckConstraint(tx, check); err != nil {
				return err
			}
		}
		indexes := []struct {
			name string
			sql  string
		}{
			{"uni_sys_menu_query_scope_active", `CREATE UNIQUE INDEX IF NOT EXISTS uni_sys_menu_query_scope_active ON sys_menu (query_scope_code) WHERE query_scope_code IS NOT NULL AND state = TRUE AND gmt_delete IS NULL`},
			{"uni_query_scheme_personal_name", `CREATE UNIQUE INDEX IF NOT EXISTS uni_query_scheme_personal_name ON query_scheme (owner_user_id, scope_code, lower(name)) WHERE scheme_type = 'PERSONAL' AND gmt_delete IS NULL`},
			{"uni_query_scheme_shared_name", `CREATE UNIQUE INDEX IF NOT EXISTS uni_query_scheme_shared_name ON query_scheme (scope_code, scheme_type, lower(name)) WHERE scheme_type <> 'PERSONAL' AND gmt_delete IS NULL`},
			{"uni_query_scheme_personal_default", `CREATE UNIQUE INDEX IF NOT EXISTS uni_query_scheme_personal_default ON query_scheme (owner_user_id, scope_code) WHERE scheme_type = 'PERSONAL' AND is_default = TRUE AND enabled = TRUE AND gmt_delete IS NULL`},
			{"uni_query_scheme_page_default", `CREATE UNIQUE INDEX IF NOT EXISTS uni_query_scheme_page_default ON query_scheme (scope_code) WHERE scheme_type = 'PAGE_DEFAULT' AND is_default = TRUE AND enabled = TRUE AND gmt_delete IS NULL`},
		}
		for _, index := range indexes {
			if err := tx.Exec(index.sql).Error; err != nil {
				return fmt.Errorf("create query scheme index %s: %w", index.name, err)
			}
		}
		foreignKeys := []postgresForeignKeyConstraint{
			{model: &model.QueryScheme{}, name: "fk_query_scheme_owner", columns: []string{"owner_user_id"}, referenceModel: &model.SysUser{}, referenceFields: []string{"id"}},
			{model: &model.QuerySchemeRole{}, name: "fk_query_scheme_role_scheme", columns: []string{"scheme_id"}, referenceModel: &model.QueryScheme{}, referenceFields: []string{"id"}},
			{model: &model.QuerySchemeRole{}, name: "fk_query_scheme_role_role", columns: []string{"role_id"}, referenceModel: &model.SysRole{}, referenceFields: []string{"id"}},
		}
		for _, foreignKey := range foreignKeys {
			if err := createPostgresForeignKeyConstraint(tx, foreignKey); err != nil {
				return err
			}
		}
		return nil
	})
}
