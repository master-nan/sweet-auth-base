package main

import (
	migrationstate "backend/internal/migration"
	"testing"
)

func TestMigrationStepsMatchCatalogContracts(t *testing.T) {
	steps := migrationSteps()
	definitions := migrationstate.Catalog()
	if len(steps) != len(definitions) {
		t.Fatalf("migration steps = %d, catalog definitions = %d", len(steps), len(definitions))
	}
	for i, step := range steps {
		definition := definitions[i]
		if step.version != definition.Version || step.name != definition.Key || step.contract != definition.Contract || step.checksum != definition.Checksum {
			t.Fatalf("migration step %d does not match catalog: %#v vs %#v", i, step, definition)
		}
	}
}

func TestMigrationCommandAcceptsAdopt(t *testing.T) {
	if command := migrationCommand([]string{"migrate", "adopt"}); command != "adopt" {
		t.Fatalf("migration command = %q, want adopt", command)
	}
}
