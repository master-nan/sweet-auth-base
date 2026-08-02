package datapermission

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"backend/dto/response"
	myerrors "backend/internal/errors"
)

func TestAdapterContractCreatesInterfaceAndExplicitDecisionModes(t *testing.T) {
	resource := adapterTestResource(t)
	var adapter Adapter = AdapterFunc(func(
		_ context.Context,
		input AdapterInput,
	) (AdapterExecution, error) {
		return BuildAdapterExecution(input)
	})

	tests := []struct {
		name     string
		create   func(string, string) (DataScopeResult, error)
		wantMode AdapterExecutionMode
	}{
		{name: "all", create: NewAllResult, wantMode: AdapterExecutionModeAllowAll},
		{name: "none", create: NewNoneResult, wantMode: AdapterExecutionModeDenyAll},
		{name: "not applicable", create: NewNotApplicableResult, wantMode: AdapterExecutionModeNotApplicable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.create(resource.ResourceCode(), resource.Operation())
			if err != nil {
				t.Fatalf("create DataScopeResult: %v", err)
			}
			input, err := NewAdapterInput(resource, result, nil)
			if err != nil {
				t.Fatalf("create AdapterInput: %v", err)
			}
			execution, err := adapter.Apply(context.Background(), input)
			if err != nil {
				t.Fatalf("apply Adapter: %v", err)
			}
			if execution.Mode() != test.wantMode || len(execution.ConditionGroups()) != 0 {
				t.Fatalf("mode=%s groups=%d, want %s and no groups",
					execution.Mode(), len(execution.ConditionGroups()), test.wantMode)
			}
		})
	}
}

func TestAdapterContractBuildsFilteredMetadataAndRegisteredBindings(t *testing.T) {
	resource := adapterTestResource(t)
	metadata := adapterTestOwnership(
		t,
		"owner_org",
		201,
		AdapterBindingTypeMetadataField,
		501,
		"",
	)
	registered := adapterTestOwnership(
		t,
		"legal_entity",
		202,
		AdapterBindingTypeRegisteredField,
		0,
		"legal_entity_id",
	)
	result := adapterTestFilteredResult(t, resource)
	input, err := NewAdapterInput(resource, result, []AdapterOwnershipDefinition{metadata, registered})
	if err != nil {
		t.Fatalf("create filtered AdapterInput: %v", err)
	}

	execution, err := BuildAdapterExecution(input)
	if err != nil {
		t.Fatalf("build filtered Adapter execution: %v", err)
	}
	if execution.Mode() != AdapterExecutionModeApplyFilter {
		t.Fatalf("mode = %s, want apply_filter", execution.Mode())
	}
	groups := execution.ConditionGroups()
	if len(groups) != 1 || len(groups[0].Conditions()) != 2 {
		t.Fatalf("unexpected filtered groups: %+v", groups)
	}
	bindings := make(map[string]AdapterCondition)
	for _, condition := range groups[0].Conditions() {
		bindings[condition.ScopeCondition().OwnershipCode()] = condition
	}
	if bindings["owner_org"].BindingType() != AdapterBindingTypeMetadataField ||
		bindings["owner_org"].TableFieldId() != 501 ||
		bindings["owner_org"].AdapterFieldCode() != "" {
		t.Fatalf("unexpected metadata binding: %+v", bindings["owner_org"])
	}
	if bindings["legal_entity"].BindingType() != AdapterBindingTypeRegisteredField ||
		bindings["legal_entity"].TableFieldId() != 0 ||
		bindings["legal_entity"].AdapterFieldCode() != "legal_entity_id" {
		t.Fatalf("unexpected registered binding: %+v", bindings["legal_entity"])
	}
}

func TestAdapterContractRejectsUnknownBindingAndMissingOwnership(t *testing.T) {
	_, err := NewAdapterOwnershipDefinition(AdapterOwnershipDefinitionInput{
		OwnershipCode: "owner_org",
		DimensionId:   201,
		BindingType:   "report_source",
		TableFieldId:  501,
		ValueType:     DataScopeValueTypeBigint,
	})
	assertAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterTypeUnsupported)

	resource := adapterTestResource(t)
	result := adapterTestFilteredResult(t, resource)
	input, err := NewAdapterInput(resource, result, nil)
	if err != nil {
		t.Fatalf("create AdapterInput without Ownerships: %v", err)
	}
	execution, err := BuildAdapterExecution(input)
	assertAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterOwnershipMissing)
	assertAdapterExecutionEmpty(t, execution)
}

func TestAdapterContractRejectsIllegalResultAndOwnershipMismatch(t *testing.T) {
	resource := adapterTestResource(t)
	_, err := NewAdapterInput(resource, DataScopeResult{}, nil)
	assertAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterInputInvalid)

	result := adapterTestFilteredResult(t, resource)
	mismatched := adapterTestOwnership(
		t,
		"owner_org",
		999,
		AdapterBindingTypeMetadataField,
		501,
		"",
	)
	legal := adapterTestOwnership(
		t,
		"legal_entity",
		202,
		AdapterBindingTypeRegisteredField,
		0,
		"legal_entity_id",
	)
	input, err := NewAdapterInput(resource, result, []AdapterOwnershipDefinition{mismatched, legal})
	if err != nil {
		t.Fatalf("create mismatched AdapterInput: %v", err)
	}
	execution, err := BuildAdapterExecution(input)
	assertAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterOwnershipMismatch)
	assertAdapterExecutionEmpty(t, execution)
}

func TestAdapterContractDoesNotAllowDecisionExpansion(t *testing.T) {
	resource := adapterTestResource(t)
	result, err := NewNoneResult(resource.ResourceCode(), resource.Operation())
	if err != nil {
		t.Fatalf("create none result: %v", err)
	}
	input, err := NewAdapterInput(resource, result, nil)
	if err != nil {
		t.Fatalf("create AdapterInput: %v", err)
	}
	adapter := AdapterFunc(func(
		_ context.Context,
		input AdapterInput,
	) (AdapterExecution, error) {
		return newAdapterExecution(input.ResourceContext(), AdapterExecutionModeAllowAll, nil)
	})

	execution, err := adapter.Apply(context.Background(), input)
	assertAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterExecutionInvalid)
	assertAdapterExecutionEmpty(t, execution)
}

func TestAdapterContractOutputContainsNoExecutableDatabaseContent(t *testing.T) {
	resource := adapterTestResource(t)
	result := adapterTestFilteredResult(t, resource)
	input, err := NewAdapterInput(resource, result, []AdapterOwnershipDefinition{
		adapterTestOwnership(t, "owner_org", 201, AdapterBindingTypeMetadataField, 501, ""),
		adapterTestOwnership(t, "legal_entity", 202, AdapterBindingTypeRegisteredField, 0, "legal_entity_id"),
	})
	if err != nil {
		t.Fatalf("create AdapterInput: %v", err)
	}
	execution, err := BuildAdapterExecution(input)
	if err != nil {
		t.Fatalf("build Adapter execution: %v", err)
	}
	payload, err := json.Marshal(execution)
	if err != nil {
		t.Fatalf("marshal Adapter execution: %v", err)
	}
	encoded := strings.ToLower(string(payload))
	for _, forbidden := range []string{"\"sql\"", "table_name", "field_name", "expression", "join", "gorm"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("Adapter execution contains forbidden content %q: %s", forbidden, encoded)
		}
	}
}

func adapterTestResource(t *testing.T) AdapterResourceContext {
	t.Helper()
	resource, err := NewAdapterResourceContext(AdapterResourceContextInput{
		ResourceCode: "transport_order",
		Operation:    "query",
		AdapterCode:  "transport_order_filter",
		TableId:      101,
	})
	if err != nil {
		t.Fatalf("create AdapterResourceContext: %v", err)
	}
	return resource
}

func adapterTestOwnership(
	t *testing.T,
	ownershipCode string,
	dimensionId int,
	bindingType AdapterBindingType,
	tableFieldId int,
	adapterFieldCode string,
) AdapterOwnershipDefinition {
	t.Helper()
	definition, err := NewAdapterOwnershipDefinition(AdapterOwnershipDefinitionInput{
		OwnershipCode:    ownershipCode,
		DimensionId:      dimensionId,
		BindingType:      bindingType,
		TableFieldId:     tableFieldId,
		AdapterFieldCode: adapterFieldCode,
		ValueType:        DataScopeValueTypeBigint,
	})
	if err != nil {
		t.Fatalf("create AdapterOwnershipDefinition: %v", err)
	}
	return definition
}

func adapterTestFilteredResult(
	t *testing.T,
	resource AdapterResourceContext,
) DataScopeResult {
	t.Helper()
	ownerOrg, err := NewDataScopeCondition(DataScopeConditionInput{
		OwnershipCode: "owner_org",
		DimensionId:   201,
		Operator:      DataScopeOperatorIn,
		ValueType:     DataScopeValueTypeBigint,
		Values:        []any{int64(11), int64(12)},
	})
	if err != nil {
		t.Fatalf("create owner_org condition: %v", err)
	}
	legalEntity, err := NewDataScopeCondition(DataScopeConditionInput{
		OwnershipCode: "legal_entity",
		DimensionId:   202,
		Operator:      DataScopeOperatorEqual,
		ValueType:     DataScopeValueTypeBigint,
		Values:        []any{int64(21)},
	})
	if err != nil {
		t.Fatalf("create legal_entity condition: %v", err)
	}
	group, err := NewDataScopeConditionGroup([]DataScopeCondition{ownerOrg, legalEntity})
	if err != nil {
		t.Fatalf("create Adapter test group: %v", err)
	}
	result, err := NewFilteredResult(
		resource.ResourceCode(),
		resource.Operation(),
		[]DataScopeConditionGroup{group},
	)
	if err != nil {
		t.Fatalf("create Adapter test result: %v", err)
	}
	return result
}

func assertAdapterError(t *testing.T, err error, code int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected Adapter error code %d", code)
	}
	var adminError *response.AdminError
	if !errors.As(err, &adminError) {
		t.Fatalf("error is not AdminError: %T %v", err, err)
	}
	if adminError.ErrorCode != code {
		t.Fatalf("error code = %d, want %d", adminError.ErrorCode, code)
	}
}

func assertAdapterExecutionEmpty(t *testing.T, execution AdapterExecution) {
	t.Helper()
	if execution.ResourceCode() != "" || execution.Operation() != "" ||
		execution.Mode() != "" || len(execution.ConditionGroups()) != 0 {
		t.Fatalf("Adapter failure returned executable output: %+v", execution)
	}
}
