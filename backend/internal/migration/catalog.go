package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const LedgerTable = "schema_migration"

type Definition struct {
	Version  int64
	Key      string
	Contract string
	Checksum string
}

var catalog = []Definition{
	{Version: 1, Key: "auto_migrate_core_schema", Contract: "v1|core-schema|36-core-models|gorm-1.25.12", Checksum: "73d77d615212a5024dd8a190023f1edea7f0fbd883ff779a50de490e7c45629a"},
	{Version: 2, Key: "metadata_value_contract", Contract: "v1|metadata-value-contract|field-types-1-11|decimal-shape|logical-display-list-checks", Checksum: "3d75686932f51ab1f435e686ac7d46ac25529cc2e0546b17d0b90794a00b6c91"},
	{Version: 3, Key: "backfill_sys_table_index_field_sequence", Contract: "v1|sys-table-index-field-sequence|postgres-pg-index-ordinality", Checksum: "d180f4c32b32aeb211b6fccf2f8e7a32d62d1cec15680adfcb48245944c96e57"},
	{Version: 4, Key: "query_scheme_schema", Contract: "v1|query-scheme-schema|tables-checks-partial-indexes-foreign-keys|retire-dictionary-scope", Checksum: "b0c18d2f5aa7aa4017ae6d55dbded5b7c2a9a938c045104de4888bf7a51b6d3a"},
	{Version: 5, Key: "integration_configuration_schema", Contract: "v1|integration-configuration-schema|configuration-models|runtime-limits|checks-indexes-retry-policy-fk", Checksum: "5dd1372012524ae6ad99f4ca6c04b93241c7e3535a9903773fb94625455f042a"},
	{Version: 6, Key: "integration_runtime_schema", Contract: "v1|integration-runtime-schema|execution-log-models|legacy-snapshot-closure|checks-foreign-keys", Checksum: "5a20cafb89e677c4cf1b3875d368136a3f3cdfbc8f78db9aebaf25571fe8a2a2"},
	{Version: 7, Key: "integration_sync_schema", Contract: "v1|integration-sync-schema|task-batch-execution-sync-contract|checks-indexes-foreign-keys", Checksum: "97d63435e863ab0b961f1b851bb8466b343b314be871f1f1a7d48291a81920a6"},
	{Version: 8, Key: "organization_sync_integrity_schema", Contract: "v1|organization-sync-integrity-schema|action-aliases|dictionary-cleanup|checks-indexes-execution-fk", Checksum: "13c61a7e75e5d86852dd6eb92c0d0120cd8a8419f044dfbeb18c7185a65cb78f"},
	{Version: 9, Key: "data_permission_domain_schema", Contract: "v1|data-permission-domain-schema|seven-models|checks|foreign-keys", Checksum: "75604e6eb748bc97810ebcadb2dd25fe0e6cba593fab218979942f41ca4c507a"},
	{Version: 10, Key: "remove_legacy_data_permission_schema", Contract: "v1|remove-legacy-data-permission-schema|tables|buttons|casbin|dictionary|metadata", Checksum: "c153561355f124ff218d501169746d5e060cf66185f575f2989c39c9a3b902c4"},
	{Version: 11, Key: "ensure_sys_menu_option_text", Contract: "v1|sys-menu-option-text|postgres-text", Checksum: "b11968bf852352a09abf63ad71927972dd7ce44aed3734dee0e7f3504cf77a00"},
	{Version: 12, Key: "backfill_sys_menu_page_binding", Contract: "v1|sys-menu-page-binding|page-type|single-table-option", Checksum: "3083840eb78345276beb1e7fe6d250f6809bcad949e01f48f398237643f334be"},
	{Version: 13, Key: "canonical_runtime_contract", Contract: "v1|canonical-runtime-contract|organization-selector-aliases|legacy-low-code-buttons", Checksum: "987f5fe06134ff1ac52f62954efcdaa40829089cb5f17274983d90829cadf313"},
	{Version: 14, Key: "organization_database_comments", Contract: "v1|organization-database-comments|nine-tables|all-model-columns", Checksum: "9bfa1afbc61d20fda1789c4a97d071160f0118cc912c66d00c654073c6e212a7"},
	{Version: 15, Key: "access_log_operational_indexes", Contract: "v1|access-log-operational-indexes|time|action-time|resource-time|success-time", Checksum: "8123677a52299c64b11482c4b74e7625f34ba3b83b257227d093148d1ac840f0"},
	{Version: 16, Key: "product_walkthrough_corrections", Contract: "v1|product-walkthrough-corrections|remove-report-workbench|protect-last-login-metadata", Checksum: "a7ce186463fa53bfe21cbbb8b7e4b6ca4977995124d613486401ebdf05e61e23"},
	{Version: 17, Key: "notification_center_schema", Contract: "v1|notification-center-schema|notifications-recipients|checks-fks-partial-indexes-dedup", Checksum: "6ade5286bb2206c70d21c8d76bd67c69784acba3ade29ec3141f10f78f4aa8c6"},
	{Version: 18, Key: "organization_source_code_indexes", Contract: "v1|organization-source-code-indexes|source-id-identity|non-unique-business-code", Checksum: "d9ae0d2b44a019fd02dcafd1da02dbadb6f363dcd4e6fc9971fa8045ab9f579c"},
	{Version: 19, Key: "canonical_time_id_contract", Contract: "v1|canonical-time-id-contract|timestamptz-asia-shanghai-conversion|snowflake-id-no-sequences", Checksum: "1343a552cf3b7a7dd4e8eeb6df4a92e7d1ce23af8947d674c3567fa11390aa1a"},
	{Version: 20, Key: "integration_reference_integrity", Contract: "v1|integration-reference-integrity|remove-orphan-credentials|configuration-fks", Checksum: "ba55364ff685b82e3ad7549a782c282824b81247080b7ba2c0497c25887d22d1"},
	{Version: 21, Key: "user_session_schema", Contract: "v1|user-session-schema|snowflake-primary-key|hashed-sid|online-heartbeat|revocation", Checksum: "371297457dde1c8fb5fcdd353b26726984a4d4b8724b2e5c336b83da5ec53a98"},
	{Version: 22, Key: "user_session_audit_fields", Contract: "v1|user-session-audit-fields|username-snapshot|closure-operator|deleted-account-history|filtered-csv-export", Checksum: "3627d65700efeb1f4a395212b18c0f92960f5f5ec34eb4fb652e2280e66a5e9e"},
	{Version: 23, Key: "notification_standard_base_fields", Contract: "v1|notification-standard-base-fields|rename-created-at|basic-audit-columns|active-soft-delete-index", Checksum: "b54042618cfb601910ce81eff48bcf157be5600d3dd670d2e4781a82b199f98e"},
	{Version: 24, Key: "metadata_column_comments", Contract: "v1|metadata-column-comments|historical-backfill|seed-bootstrap|low-code-create-update-sync", Checksum: "5cc084f0a7c8c4b6b07300809a11f89a2d2eca995341d518518dbd780110b129"},
	{Version: 25, Key: "unify_report_runtime_component", Contract: "v1|unify-report-runtime-component|report-menu-component|remove-v2-runtime", Checksum: "1aae2d369d5459ea872516d42d25cd134cadec41cc11a1340cd06047fe2340b2"},
}

var managedTables = []string{
	"sys_configure",
	"sys_table",
	"sys_table_field",
	"sys_table_relation",
	"sys_table_index",
	"sys_table_index_field",
	"sys_dict",
	"sys_dict_item",
	"access_log",
	"login_log",
	"sys_user",
	"sys_user_role",
	"sys_menu",
	"sys_menu_button",
	"sys_menu_button_template",
	"sys_role",
	"sys_role_menu",
	"sys_role_menu_button",
	"org_legal_entity",
	"org_unit",
	"org_structure",
	"org_structure_node",
	"org_position",
	"org_employee",
	"org_assignment",
	"org_sync_batch",
	"org_sync_record",
	"report_definition",
	"report_definition_version",
	"report_execution_log",
	"application",
	"sms_template",
	"sms_log",
	"file",
	"file_chunk",
	"casbin_rule",
	"query_scheme",
	"query_scheme_role",
	"integration_external_system",
	"integration_credential",
	"integration_retry_policy",
	"integration_interface_definition",
	"integration_execution",
	"integration_log",
	"integration_sync_task",
	"integration_sync_batch",
	"sys_data_dimension_definition",
	"sys_data_resource",
	"sys_data_resource_operation",
	"sys_data_ownership_field",
	"sys_data_policy",
	"sys_data_policy_rule",
	"sys_data_grant",
	"notification",
	"notification_recipient",
	"sys_user_session",
}

func Catalog() []Definition {
	return append([]Definition(nil), catalog...)
}

func ManagedTables() []string {
	return append([]string(nil), managedTables...)
}

func ValidateCatalog(definitions []Definition) error {
	seenKeys := make(map[string]struct{}, len(definitions))
	for i, definition := range definitions {
		expectedVersion := int64(i + 1)
		if definition.Version != expectedVersion {
			return fmt.Errorf("migration catalog order invalid at position %d: version %d, expected %d", i, definition.Version, expectedVersion)
		}
		if definition.Key == "" {
			return fmt.Errorf("migration catalog version %d has an empty key", definition.Version)
		}
		if _, exists := seenKeys[definition.Key]; exists {
			return fmt.Errorf("migration catalog has duplicate key %q", definition.Key)
		}
		seenKeys[definition.Key] = struct{}{}
		digest := sha256.Sum256([]byte(definition.Contract))
		if definition.Checksum != hex.EncodeToString(digest[:]) {
			return fmt.Errorf("migration catalog checksum mismatch for version %d (%s)", definition.Version, definition.Key)
		}
	}
	return nil
}
