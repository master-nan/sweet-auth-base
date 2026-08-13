package metadata

import "testing"

func TestValidateReadOnlyQuery(t *testing.T) {
	for _, query := range []string{
		"SELECT id, name FROM org_unit",
		"WITH active AS (SELECT id FROM org_unit WHERE state = true) SELECT * FROM active",
	} {
		if _, err := ValidateReadOnlyQuery(query); err != nil {
			t.Fatalf("valid read-only query rejected: %q: %v", query, err)
		}
	}

	for _, query := range []string{
		"",
		"DELETE FROM org_unit",
		"SELECT id FROM org_unit; DROP TABLE org_unit",
		"CREATE VIEW leaked AS SELECT password FROM sys_user",
		"WITH removed AS (DELETE FROM org_unit RETURNING id) SELECT * FROM removed",
		"EXPLAIN ANALYZE SELECT * FROM org_unit",
	} {
		if _, err := ValidateReadOnlyQuery(query); err == nil {
			t.Fatalf("unsafe query accepted: %q", query)
		}
	}
}
