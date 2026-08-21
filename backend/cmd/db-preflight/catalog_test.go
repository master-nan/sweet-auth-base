package main

import "testing"

func TestRequiredPrimaryTablesUsesCompleteManagedCatalog(t *testing.T) {
	tables := requiredPrimaryTables()
	if len(tables) != 53 {
		t.Fatalf("required primary tables = %d, want 53", len(tables))
	}
	required := map[string]bool{
		"org_assignment":                   false,
		"report_definition_version":        false,
		"integration_interface_definition": false,
		"query_scheme_role":                false,
		"sys_menu_button_template":         false,
	}
	for _, table := range tables {
		if _, exists := required[table]; exists {
			required[table] = true
		}
	}
	for table, found := range required {
		if !found {
			t.Errorf("required primary tables missing %s", table)
		}
	}
}
