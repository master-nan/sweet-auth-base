package querycapability

import (
	"backend/enum"
	"backend/internal/metadata"
	"testing"
)

func TestAllowedMetadataOperatorsUsesStorageAndOptionFacts(t *testing.T) {
	decimal := metadata.FieldMetadata{StorageType: enum.DecimalFieldType}
	if !SupportsMetadata(decimal, enum.Between) || SupportsMetadata(decimal, enum.Like) {
		t.Fatal("Decimal operator capability must be ordered without text matching")
	}
	relation := metadata.FieldMetadata{StorageType: enum.BigIntFieldType, Relation: &metadata.RelationDisplayMetadata{TargetTableCode: "org_employee"}}
	if SupportsMetadata(relation, enum.Gt) || !SupportsMetadata(relation, enum.In) {
		t.Fatal("relation operator capability must use controlled equality operators")
	}
}
