package datapermission

import (
	"errors"
	"sync"
	"testing"
)

func TestOwnershipFieldRegistryRegistrationAndValidation(t *testing.T) {
	registry, err := NewOwnershipFieldRegistry(registeredOwnerOrgField())
	if err != nil {
		t.Fatalf("initialize registry: %v", err)
	}
	if err = registry.ValidateBinding(OwnershipFieldBindingValidation{
		ResourceCode:     "tms.transport_order",
		OwnershipCode:    "owner_org",
		AdapterFieldCode: "owner_org_id",
		ValueType:        "bigint",
		DimensionCode:    "management_org",
	}); err != nil {
		t.Fatalf("validate registered binding: %v", err)
	}
	if err = registry.ValidateOperation(OwnershipFieldOperationValidation{
		ResourceCode:  "tms.transport_order",
		OwnershipCode: "owner_org",
		Operation:     "query",
	}); err != nil {
		t.Fatalf("validate registered operation: %v", err)
	}
}

func TestOwnershipFieldRegistryRejectsInvalidBindings(t *testing.T) {
	registry, err := NewOwnershipFieldRegistry(registeredOwnerOrgField())
	if err != nil {
		t.Fatalf("initialize registry: %v", err)
	}
	tests := []struct {
		name       string
		validation OwnershipFieldBindingValidation
		want       error
	}{
		{
			name: "unregistered field",
			validation: OwnershipFieldBindingValidation{
				ResourceCode:     "tms.transport_order",
				OwnershipCode:    "legal_entity",
				AdapterFieldCode: "legal_entity_id",
				ValueType:        "bigint",
				DimensionCode:    "legal_entity",
			},
			want: ErrOwnershipFieldRegistrationNotFound,
		},
		{
			name: "resource mismatch",
			validation: OwnershipFieldBindingValidation{
				ResourceCode:     "wms.inventory",
				OwnershipCode:    "owner_org",
				AdapterFieldCode: "owner_org_id",
				ValueType:        "bigint",
				DimensionCode:    "management_org",
			},
			want: ErrOwnershipFieldResourceMismatch,
		},
		{
			name: "value type mismatch",
			validation: OwnershipFieldBindingValidation{
				ResourceCode:     "tms.transport_order",
				OwnershipCode:    "owner_org",
				AdapterFieldCode: "owner_org_id",
				ValueType:        "string",
				DimensionCode:    "management_org",
			},
			want: ErrOwnershipFieldValueTypeMismatch,
		},
		{
			name: "dimension unsupported",
			validation: OwnershipFieldBindingValidation{
				ResourceCode:     "tms.transport_order",
				OwnershipCode:    "owner_org",
				AdapterFieldCode: "owner_org_id",
				ValueType:        "bigint",
				DimensionCode:    "employee",
			},
			want: ErrOwnershipFieldDimensionUnsupported,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := registry.ValidateBinding(tt.validation); !errors.Is(err, tt.want) {
				t.Fatalf("ValidateBinding() error = %v, want %v", err, tt.want)
			}
		})
	}

	if err := registry.ValidateOperation(OwnershipFieldOperationValidation{
		ResourceCode:  "tms.transport_order",
		OwnershipCode: "owner_org",
		Operation:     "delete",
	}); !errors.Is(err, ErrOwnershipFieldOperationUnsupported) {
		t.Fatalf("ValidateOperation() error = %v, want operation unsupported", err)
	}
}

func TestOwnershipFieldRegistryRejectsDuplicateRegistration(t *testing.T) {
	registry, err := NewOwnershipFieldRegistry(registeredOwnerOrgField())
	if err != nil {
		t.Fatalf("initialize registry: %v", err)
	}
	if err = registry.Register(registeredOwnerOrgField()); !errors.Is(
		err,
		ErrOwnershipFieldRegistrationDuplicate,
	) {
		t.Fatalf("duplicate registration error = %v", err)
	}
	conflicting := registeredOwnerOrgField()
	conflicting.AdapterFieldCode = "alternate_org_id"
	if err = registry.Register(conflicting); !errors.Is(
		err,
		ErrOwnershipFieldRegistrationDuplicate,
	) {
		t.Fatalf("conflicting ownership registration error = %v", err)
	}
}

func TestOwnershipFieldRegistryConcurrentReads(t *testing.T) {
	registry, err := NewOwnershipFieldRegistry(registeredOwnerOrgField())
	if err != nil {
		t.Fatalf("initialize registry: %v", err)
	}
	validation := OwnershipFieldBindingValidation{
		ResourceCode:     "tms.transport_order",
		OwnershipCode:    "owner_org",
		AdapterFieldCode: "owner_org_id",
		ValueType:        "bigint",
		DimensionCode:    "management_org",
	}

	const readers = 64
	var wg sync.WaitGroup
	errs := make(chan error, readers)
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- registry.ValidateBinding(validation)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent read: %v", err)
		}
	}
}

func TestOwnershipFieldRegistryInstancesAreIsolated(t *testing.T) {
	first, err := NewOwnershipFieldRegistry(registeredOwnerOrgField())
	if err != nil {
		t.Fatalf("initialize first registry: %v", err)
	}
	second, err := NewOwnershipFieldRegistry()
	if err != nil {
		t.Fatalf("initialize second registry: %v", err)
	}
	validation := OwnershipFieldBindingValidation{
		ResourceCode:     "tms.transport_order",
		OwnershipCode:    "owner_org",
		AdapterFieldCode: "owner_org_id",
		ValueType:        "bigint",
		DimensionCode:    "management_org",
	}
	if err = first.ValidateBinding(validation); err != nil {
		t.Fatalf("first registry validation: %v", err)
	}
	if err = second.ValidateBinding(validation); !errors.Is(
		err,
		ErrOwnershipFieldRegistrationNotFound,
	) {
		t.Fatalf("second registry leaked first registration: %v", err)
	}
}

func registeredOwnerOrgField() OwnershipFieldRegistration {
	return OwnershipFieldRegistration{
		ResourceCode:        "tms.transport_order",
		OwnershipCode:       "owner_org",
		AdapterFieldCode:    "owner_org_id",
		ValueType:           "bigint",
		SupportedDimensions: []string{"management_org"},
		SupportedOperations: []string{"query", "detail", "export"},
	}
}
