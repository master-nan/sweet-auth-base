package migration

import (
	"strings"
	"testing"
	"time"
)

func TestCatalogContractsAndChecksums(t *testing.T) {
	definitions := Catalog()
	if len(definitions) != 16 {
		t.Fatalf("catalog has %d definitions, want 16", len(definitions))
	}
	if err := ValidateCatalog(definitions); err != nil {
		t.Fatalf("validate catalog: %v", err)
	}
}

func TestManagedTablesHasCompleteUniqueCatalog(t *testing.T) {
	tables := ManagedTables()
	if len(tables) != 53 {
		t.Fatalf("managed table catalog has %d tables, want 53", len(tables))
	}
	seen := make(map[string]struct{}, len(tables))
	for _, table := range tables {
		if _, exists := seen[table]; exists {
			t.Fatalf("managed table catalog contains duplicate %q", table)
		}
		seen[table] = struct{}{}
	}
}

func TestValidateLedgerFailsClosed(t *testing.T) {
	definitions := Catalog()
	entries := ledgerEntriesForDefinitions(definitions)

	t.Run("checksum", func(t *testing.T) {
		corrupt := append([]LedgerEntry(nil), entries...)
		corrupt[0].Checksum = strings.Repeat("0", 64)
		if err := ValidateLedger(corrupt, definitions, true); err == nil || !strings.Contains(err.Error(), "checksum") {
			t.Fatalf("expected checksum failure, got %v", err)
		}
	})

	t.Run("order", func(t *testing.T) {
		corrupt := append([]LedgerEntry(nil), entries...)
		corrupt[0], corrupt[1] = corrupt[1], corrupt[0]
		if err := ValidateLedger(corrupt, definitions, true); err == nil || !strings.Contains(err.Error(), "order") {
			t.Fatalf("expected order failure, got %v", err)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		corrupt := append([]LedgerEntry(nil), entries...)
		corrupt[0].Key = "unknown_migration"
		if err := ValidateLedger(corrupt, definitions, true); err == nil || !strings.Contains(err.Error(), "unknown") {
			t.Fatalf("expected unknown migration failure, got %v", err)
		}
	})

	t.Run("incomplete", func(t *testing.T) {
		if err := ValidateLedger(entries[:len(entries)-1], definitions, true); err == nil || !strings.Contains(err.Error(), "incomplete") {
			t.Fatalf("expected incomplete ledger failure, got %v", err)
		}
	})
}

func ledgerEntriesForDefinitions(definitions []Definition) []LedgerEntry {
	entries := make([]LedgerEntry, 0, len(definitions))
	for _, definition := range definitions {
		entries = append(entries, LedgerEntry{
			Version:   definition.Version,
			Key:       definition.Key,
			Checksum:  definition.Checksum,
			AppliedAt: time.Unix(definition.Version, 0),
		})
	}
	return entries
}
