package datapermission

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"backend/dto/response"
	myerrors "backend/internal/errors"
)

func TestRegisteredFieldExecutionRegistryRegistrationRules(t *testing.T) {
	executor := &registeredFieldTestExecutor{}
	dimensions := []int{202, 201, 201}
	operations := []string{"export", "query", "query"}
	operators := []DataScopeOperator{DataScopeOperatorIn, DataScopeOperatorEqual, DataScopeOperatorIn}
	registration := registeredFieldTestRegistration(
		"transport_order", "transport_order_filter", "owner_org_id", []int{201, 202}, executor,
	)
	registration.DimensionIds = dimensions
	registration.SupportedOperations = operations
	registration.SupportedOperators = operators

	registry, err := NewRegisteredFieldExecutionRegistry(registration)
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}
	dimensions[0] = 999
	operations[0] = "delete"
	operators[0] = "unsupported"

	entry, err := registry.resolve("transport_order_filter", "owner_org_id")
	if err != nil {
		t.Fatalf("resolve registration: %v", err)
	}
	if !intSlicesEqual(entry.registration.DimensionIds, []int{201, 202}) ||
		!stringSlicesEqual(entry.registration.SupportedOperations, []string{"export", "query"}) ||
		!operatorSlicesEqual(entry.registration.SupportedOperators, []DataScopeOperator{DataScopeOperatorEqual, DataScopeOperatorIn}) {
		t.Fatalf("registration was not normalized and isolated: %+v", entry.registration)
	}

	duplicate := registeredFieldTestRegistration(
		"transport_order", "transport_order_filter", "owner_org_id", []int{202, 201}, &registeredFieldTestExecutor{},
	)
	duplicate.SupportedOperations = []string{"query", "export"}
	if err = registry.Register(duplicate); !isAdminErrorCode(err, myerrors.ErrorCodeRegisteredAdapterRegistrationDuplicate) {
		t.Fatalf("duplicate error = %v", err)
	}

	conflict := duplicate
	conflict.ResourceCode = "inventory"
	if err = registry.Register(conflict); !isAdminErrorCode(err, myerrors.ErrorCodeRegisteredAdapterRegistrationConflict) {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestRegisteredFieldExecutionRegistryLookupIsolationAndConcurrency(t *testing.T) {
	registration := registeredFieldTestRegistration(
		"transport_order", "transport_order_filter", "owner_org_id", []int{201}, &registeredFieldTestExecutor{},
	)
	first, err := NewRegisteredFieldExecutionRegistry(registration)
	if err != nil {
		t.Fatalf("create first registry: %v", err)
	}
	second, err := NewRegisteredFieldExecutionRegistry()
	if err != nil {
		t.Fatalf("create isolated registry: %v", err)
	}
	if _, err = second.resolve("transport_order_filter", "owner_org_id"); !isAdminErrorCode(err, myerrors.ErrorCodeRegisteredAdapterUnregistered) {
		t.Fatalf("isolated registry unexpectedly inherited registration: %v", err)
	}

	if err = first.Register(registeredFieldTestRegistration(
		"transport_order", "transport_order_filter", "legal_entity_id", []int{202}, &registeredFieldTestExecutor{},
	)); err != nil {
		t.Fatalf("register second field: %v", err)
	}
	if _, err = first.resolve("transport_order_filter", "missing_field"); !isAdminErrorCode(err, myerrors.ErrorCodeRegisteredAdapterFieldNotFound) {
		t.Fatalf("missing field error = %v", err)
	}
	if _, err = first.resolve("missing_adapter", "owner_org_id"); !isAdminErrorCode(err, myerrors.ErrorCodeRegisteredAdapterUnregistered) {
		t.Fatalf("missing adapter error = %v", err)
	}

	var wait sync.WaitGroup
	errorsCh := make(chan error, 128)
	for index := 0; index < 128; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			entry, resolveErr := first.resolve("transport_order_filter", "owner_org_id")
			if resolveErr != nil || entry.registration.ResourceCode != "transport_order" {
				errorsCh <- resolveErr
			}
		}()
	}
	wait.Wait()
	close(errorsCh)
	for resolveErr := range errorsCh {
		t.Fatalf("concurrent registry read failed: %v", resolveErr)
	}
}

func TestRegisteredFieldAdapterExecutionModes(t *testing.T) {
	registry, err := NewRegisteredFieldExecutionRegistry()
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}
	adapter := mustRegisteredFieldAdapter(t, registry)
	resource := registeredFieldTestResource(t, "transport_order", "query", "transport_order_filter")
	tests := []struct {
		name     string
		create   func(string, string) (DataScopeResult, error)
		wantMode AdapterExecutionMode
	}{
		{name: "not applicable", create: NewNotApplicableResult, wantMode: AdapterExecutionModeNotApplicable},
		{name: "allow all", create: NewAllResult, wantMode: AdapterExecutionModeAllowAll},
		{name: "deny all", create: NewNoneResult, wantMode: AdapterExecutionModeDenyAll},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, createErr := test.create(resource.ResourceCode(), resource.Operation())
			if createErr != nil {
				t.Fatalf("create result: %v", createErr)
			}
			input := mustRegisteredFieldInput(t, resource, result, nil)
			execution, applyErr := adapter.Apply(context.Background(), input)
			if applyErr != nil {
				t.Fatalf("apply registered adapter: %v", applyErr)
			}
			if execution.Mode() != test.wantMode || len(execution.ConditionGroups()) != 0 {
				t.Fatalf("execution = %+v, want mode %s without filters", execution, test.wantMode)
			}
		})
	}
}

func TestRegisteredFieldAdapterPreservesAndOrAndMultipleOwnerships(t *testing.T) {
	executor := &registeredFieldTestExecutor{}
	registry := mustRegisteredFieldRegistry(t,
		registeredFieldTestRegistration("transport_order", "transport_order_filter", "owner_org_id", []int{201}, executor),
		registeredFieldTestRegistration("transport_order", "transport_order_filter", "legal_entity_id", []int{202}, executor),
		registeredFieldTestRegistration("transport_order", "transport_order_filter", "owner_employee_id", []int{203}, executor),
	)
	resource := registeredFieldTestResource(t, "transport_order", "query", "transport_order_filter")
	ownerOrg := mustRegisteredFieldCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(2), int64(1), int64(2)})
	legalEntity := mustRegisteredFieldCondition(t, "legal_entity", 202, DataScopeOperatorEqual, DataScopeValueTypeBigint, []any{int64(10)})
	ownerEmployee := mustRegisteredFieldCondition(t, "owner_employee", 203, DataScopeOperatorEqual, DataScopeValueTypeBigint, []any{int64(1001)})
	result := mustRegisteredFieldResult(t, resource, [][]DataScopeCondition{
		{ownerOrg, legalEntity},
		{ownerEmployee},
		{ownerEmployee},
	})
	definitions := []AdapterOwnershipDefinition{
		mustRegisteredFieldOwnership(t, "owner_org", 201, "owner_org_id", DataScopeValueTypeBigint),
		mustRegisteredFieldOwnership(t, "legal_entity", 202, "legal_entity_id", DataScopeValueTypeBigint),
		mustRegisteredFieldOwnership(t, "owner_employee", 203, "owner_employee_id", DataScopeValueTypeBigint),
	}

	execution, err := mustRegisteredFieldAdapter(t, registry).Apply(
		context.Background(),
		mustRegisteredFieldInput(t, resource, result, definitions),
	)
	if err != nil {
		t.Fatalf("apply registered adapter: %v", err)
	}
	groups := execution.ConditionGroups()
	if execution.Mode() != AdapterExecutionModeApplyFilter || len(groups) != 2 {
		t.Fatalf("OR groups were not preserved and deduplicated: %+v", execution)
	}
	sizes := map[int]int{}
	for _, group := range groups {
		sizes[len(group.Conditions())]++
	}
	if sizes[1] != 1 || sizes[2] != 1 {
		t.Fatalf("AND conditions were not preserved: %+v", sizes)
	}
	if executor.callCount() != 3 {
		t.Fatalf("executor calls = %d, want 3 unique conditions", executor.callCount())
	}
	for _, request := range executor.snapshot() {
		if request.ResourceContext().ResourceCode() != "transport_order" ||
			request.Condition().BindingType() != AdapterBindingTypeRegisteredField {
			t.Fatalf("executor received unsafe or incomplete request: %+v", request)
		}
	}
}

func TestRegisteredFieldAdapterValidationFailures(t *testing.T) {
	baseRegistration := func() RegisteredFieldExecutionRegistration {
		return registeredFieldTestRegistration(
			"transport_order", "transport_order_filter", "owner_org_id", []int{201}, &registeredFieldTestExecutor{},
		)
	}
	tests := []struct {
		name         string
		resourceCode string
		adapterCode  string
		operation    string
		fieldCode    string
		dimensionId  int
		valueType    DataScopeValueType
		operator     DataScopeOperator
		values       []any
		configure    func(*RegisteredFieldExecutionRegistration)
		wantCode     int
	}{
		{
			name: "adapter not registered", adapterCode: "unknown_filter",
			wantCode: myerrors.ErrorCodeRegisteredAdapterUnregistered,
		},
		{
			name: "registered field missing", fieldCode: "missing_field",
			wantCode: myerrors.ErrorCodeRegisteredAdapterFieldNotFound,
		},
		{
			name: "resource mismatch", resourceCode: "inventory",
			wantCode: myerrors.ErrorCodeRegisteredAdapterResourceMismatch,
		},
		{
			name: "dimension unsupported", configure: func(registration *RegisteredFieldExecutionRegistration) {
				registration.DimensionIds = []int{999}
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterDimensionUnsupported,
		},
		{
			name: "value type unsupported", valueType: DataScopeValueTypeString, values: []any{"ORG-A"},
			configure: func(registration *RegisteredFieldExecutionRegistration) {
				registration.ValueType = DataScopeValueTypeBigint
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterValueTypeUnsupported,
		},
		{
			name: "operation unsupported", operation: "detail",
			configure: func(registration *RegisteredFieldExecutionRegistration) {
				registration.SupportedOperations = []string{"query"}
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterOperationUnsupported,
		},
		{
			name: "create unsupported", operation: "create",
			configure: func(registration *RegisteredFieldExecutionRegistration) {
				registration.SupportedOperations = []string{"query", "detail", "update", "delete", "export"}
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterOperationUnsupported,
		},
		{
			name: "run unsupported", operation: "run",
			configure: func(registration *RegisteredFieldExecutionRegistration) {
				registration.SupportedOperations = []string{"query", "detail", "update", "delete", "export"}
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterOperationUnsupported,
		},
		{
			name: "operator unsupported", operator: DataScopeOperatorIn,
			configure: func(registration *RegisteredFieldExecutionRegistration) {
				registration.SupportedOperators = []DataScopeOperator{DataScopeOperatorEqual}
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterOperatorUnsupported,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registration := baseRegistration()
			if test.configure != nil {
				test.configure(&registration)
			}
			registry := mustRegisteredFieldRegistry(t, registration)
			resourceCode := defaultString(test.resourceCode, "transport_order")
			adapterCode := defaultString(test.adapterCode, "transport_order_filter")
			operation := defaultString(test.operation, "query")
			fieldCode := defaultString(test.fieldCode, "owner_org_id")
			dimensionId := test.dimensionId
			if dimensionId == 0 {
				dimensionId = 201
			}
			valueType := test.valueType
			if valueType == "" {
				valueType = DataScopeValueTypeBigint
			}
			operator := test.operator
			if operator == "" {
				operator = DataScopeOperatorEqual
			}
			values := test.values
			if values == nil {
				values = []any{int64(1)}
			}
			resource := registeredFieldTestResource(t, resourceCode, operation, adapterCode)
			condition := mustRegisteredFieldCondition(t, "owner_org", dimensionId, operator, valueType, values)
			result := mustRegisteredFieldResult(t, resource, [][]DataScopeCondition{{condition}})
			definition := mustRegisteredFieldOwnership(t, "owner_org", dimensionId, fieldCode, valueType)
			execution, err := mustRegisteredFieldAdapter(t, registry).Apply(
				context.Background(),
				mustRegisteredFieldInput(t, resource, result, []AdapterOwnershipDefinition{definition}),
			)
			assertRegisteredAdapterError(t, err, test.wantCode)
			assertAdapterExecutionEmpty(t, execution)
		})
	}
}

func TestRegisteredFieldAdapterRejectsOwnershipAndExecutorFailures(t *testing.T) {
	resource := registeredFieldTestResource(t, "transport_order", "query", "transport_order_filter")
	condition := mustRegisteredFieldCondition(t, "owner_org", 201, DataScopeOperatorEqual, DataScopeValueTypeBigint, []any{int64(1)})
	result := mustRegisteredFieldResult(t, resource, [][]DataScopeCondition{{condition}})

	t.Run("missing ownership", func(t *testing.T) {
		registry := mustRegisteredFieldRegistry(t, registeredFieldTestRegistration(
			"transport_order", "transport_order_filter", "owner_org_id", []int{201}, &registeredFieldTestExecutor{},
		))
		execution, err := mustRegisteredFieldAdapter(t, registry).Apply(
			context.Background(), mustRegisteredFieldInput(t, resource, result, nil),
		)
		assertRegisteredAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterOwnershipMissing)
		assertAdapterExecutionEmpty(t, execution)
	})

	t.Run("dimension mismatch", func(t *testing.T) {
		definition := mustRegisteredFieldOwnership(t, "owner_org", 999, "owner_org_id", DataScopeValueTypeBigint)
		input := mustRegisteredFieldInput(t, resource, result, []AdapterOwnershipDefinition{definition})
		registry := mustRegisteredFieldRegistry(t, registeredFieldTestRegistration(
			"transport_order", "transport_order_filter", "owner_org_id", []int{201}, &registeredFieldTestExecutor{},
		))
		execution, err := mustRegisteredFieldAdapter(t, registry).Apply(context.Background(), input)
		assertRegisteredAdapterError(t, err, myerrors.ErrorCodeRegisteredAdapterDimensionUnsupported)
		assertAdapterExecutionEmpty(t, execution)
	})

	t.Run("metadata binding", func(t *testing.T) {
		definition := mustMetadataOwnership(t, "owner_org", 201, 501, DataScopeValueTypeBigint)
		input := mustRegisteredFieldInput(t, resource, result, []AdapterOwnershipDefinition{definition})
		registry := mustRegisteredFieldRegistry(t)
		execution, err := mustRegisteredFieldAdapter(t, registry).Apply(context.Background(), input)
		assertRegisteredAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterTypeUnsupported)
		assertAdapterExecutionEmpty(t, execution)
	})

	t.Run("executor failure is atomic", func(t *testing.T) {
		executor := &registeredFieldTestExecutor{err: errors.New("business repository detail")}
		registry := mustRegisteredFieldRegistry(t, registeredFieldTestRegistration(
			"transport_order", "transport_order_filter", "owner_org_id", []int{201}, executor,
		))
		definition := mustRegisteredFieldOwnership(t, "owner_org", 201, "owner_org_id", DataScopeValueTypeBigint)
		execution, err := mustRegisteredFieldAdapter(t, registry).Apply(
			context.Background(), mustRegisteredFieldInput(t, resource, result, []AdapterOwnershipDefinition{definition}),
		)
		assertRegisteredAdapterError(t, err, myerrors.ErrorCodeRegisteredAdapterPartialConversion)
		assertAdapterExecutionEmpty(t, execution)
	})

	t.Run("later condition failure returns no partial execution", func(t *testing.T) {
		accepted := &registeredFieldTestExecutor{}
		rejected := &registeredFieldTestExecutor{err: errors.New("rejected by module")}
		registry := mustRegisteredFieldRegistry(t,
			registeredFieldTestRegistration(
				"transport_order", "transport_order_filter", "a_owner_id", []int{201}, accepted,
			),
			registeredFieldTestRegistration(
				"transport_order", "transport_order_filter", "z_owner_id", []int{202}, rejected,
			),
		)
		first := mustRegisteredFieldCondition(t, "a_owner", 201, DataScopeOperatorEqual, DataScopeValueTypeBigint, []any{int64(1)})
		second := mustRegisteredFieldCondition(t, "z_owner", 202, DataScopeOperatorEqual, DataScopeValueTypeBigint, []any{int64(2)})
		multiResult := mustRegisteredFieldResult(t, resource, [][]DataScopeCondition{{first, second}})
		definitions := []AdapterOwnershipDefinition{
			mustRegisteredFieldOwnership(t, "a_owner", 201, "a_owner_id", DataScopeValueTypeBigint),
			mustRegisteredFieldOwnership(t, "z_owner", 202, "z_owner_id", DataScopeValueTypeBigint),
		}
		execution, err := mustRegisteredFieldAdapter(t, registry).Apply(
			context.Background(), mustRegisteredFieldInput(t, resource, multiResult, definitions),
		)
		assertRegisteredAdapterError(t, err, myerrors.ErrorCodeRegisteredAdapterPartialConversion)
		assertAdapterExecutionEmpty(t, execution)
		if accepted.callCount() != 1 || rejected.callCount() != 1 {
			t.Fatalf("conditions were not prepared in a complete pass: accepted=%d rejected=%d",
				accepted.callCount(), rejected.callCount())
		}
	})
}

func TestRegisteredFieldAdapterSupportsStrictStringValues(t *testing.T) {
	executor := &registeredFieldTestExecutor{}
	registration := registeredFieldTestRegistration(
		"customer", "customer_filter", "territory_code", []int{301}, executor,
	)
	registration.ValueType = DataScopeValueTypeString
	registry := mustRegisteredFieldRegistry(t, registration)
	resource := registeredFieldTestResource(t, "customer", "export", "customer_filter")
	condition := mustRegisteredFieldCondition(
		t, "territory", 301, DataScopeOperatorIn, DataScopeValueTypeString, []any{"CN-EAST", "CN-NORTH"},
	)
	result := mustRegisteredFieldResult(t, resource, [][]DataScopeCondition{{condition}})
	definition := mustRegisteredFieldOwnership(t, "territory", 301, "territory_code", DataScopeValueTypeString)

	execution, err := mustRegisteredFieldAdapter(t, registry).Apply(
		context.Background(), mustRegisteredFieldInput(t, resource, result, []AdapterOwnershipDefinition{definition}),
	)
	if err != nil {
		t.Fatalf("apply string registration: %v", err)
	}
	values := execution.ConditionGroups()[0].Conditions()[0].ScopeCondition().StringValues()
	if len(values) != 2 || values[0] != "CN-EAST" || values[1] != "CN-NORTH" {
		t.Fatalf("string values changed: %v", values)
	}
}

func TestRegisteredFieldAdapterOperatorCardinalityGuards(t *testing.T) {
	registry := mustRegisteredFieldRegistry(t, registeredFieldTestRegistration(
		"transport_order", "transport_order_filter", "owner_org_id", []int{201}, &registeredFieldTestExecutor{},
	))
	entry, err := registry.resolve("transport_order_filter", "owner_org_id")
	if err != nil {
		t.Fatalf("resolve registration: %v", err)
	}
	resource := registeredFieldTestResource(t, "transport_order", "query", "transport_order_filter")
	tests := []struct {
		name      string
		condition DataScopeCondition
		wantCode  int
	}{
		{
			name: "eq multiple values",
			condition: DataScopeCondition{
				ownershipCode: "owner_org", dimensionId: 201, operator: DataScopeOperatorEqual,
				valueType: DataScopeValueTypeBigint, bigintValues: []int64{1, 2},
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterExecutionInvalid,
		},
		{
			name: "in empty values",
			condition: DataScopeCondition{
				ownershipCode: "owner_org", dimensionId: 201, operator: DataScopeOperatorIn,
				valueType: DataScopeValueTypeBigint,
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterExecutionInvalid,
		},
		{
			name: "unknown operator",
			condition: DataScopeCondition{
				ownershipCode: "owner_org", dimensionId: 201, operator: "contains",
				valueType: DataScopeValueTypeBigint, bigintValues: []int64{1},
			},
			wantCode: myerrors.ErrorCodeRegisteredAdapterOperatorUnsupported,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			condition := AdapterCondition{
				condition: test.condition, bindingType: AdapterBindingTypeRegisteredField,
				adapterFieldCode: "owner_org_id",
			}
			err := validateRegisteredExecutionCondition(resource, condition, entry)
			assertRegisteredAdapterError(t, err, test.wantCode)
		})
	}
}

func TestRegisteredFieldAdapterOutputSafetyAndIsolation(t *testing.T) {
	executor := &registeredFieldTestExecutor{}
	registration := registeredFieldTestRegistration(
		"transport_order", "transport_order_filter", "owner_org_id", []int{201}, executor,
	)
	registry := mustRegisteredFieldRegistry(t, registration)
	resource := registeredFieldTestResource(t, "transport_order", "query", "transport_order_filter")
	values := []any{int64(3), int64(1)}
	condition := mustRegisteredFieldCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, values)
	values[0] = int64(999)
	result := mustRegisteredFieldResult(t, resource, [][]DataScopeCondition{{condition}})
	definition := mustRegisteredFieldOwnership(t, "owner_org", 201, "owner_org_id", DataScopeValueTypeBigint)
	execution, err := mustRegisteredFieldAdapter(t, registry).Apply(
		context.Background(), mustRegisteredFieldInput(t, resource, result, []AdapterOwnershipDefinition{definition}),
	)
	if err != nil {
		t.Fatalf("apply adapter: %v", err)
	}

	groups := execution.ConditionGroups()
	readValues := groups[0].Conditions()[0].ScopeCondition().BigintValues()
	readValues[0] = 999
	again := execution.ConditionGroups()[0].Conditions()[0].ScopeCondition().BigintValues()
	if len(again) != 2 || again[0] != 1 || again[1] != 3 {
		t.Fatalf("execution values are mutable: %v", again)
	}

	payload, err := json.Marshal(execution)
	if err != nil {
		t.Fatalf("marshal execution: %v", err)
	}
	encoded := strings.ToLower(string(payload))
	for _, forbidden := range []string{"\"sql\"", "table_name", "field_name", "column_name", "expression", "join", "gorm", "select "} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("registered execution contains forbidden content %q: %s", forbidden, encoded)
		}
	}
}

func TestRegisteredFieldAdapterConcurrentApply(t *testing.T) {
	executor := &registeredFieldTestExecutor{}
	registry := mustRegisteredFieldRegistry(t, registeredFieldTestRegistration(
		"transport_order", "transport_order_filter", "owner_org_id", []int{201}, executor,
	))
	adapter := mustRegisteredFieldAdapter(t, registry)
	resource := registeredFieldTestResource(t, "transport_order", "query", "transport_order_filter")
	condition := mustRegisteredFieldCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(1), int64(2)})
	result := mustRegisteredFieldResult(t, resource, [][]DataScopeCondition{{condition}})
	definition := mustRegisteredFieldOwnership(t, "owner_org", 201, "owner_org_id", DataScopeValueTypeBigint)
	input := mustRegisteredFieldInput(t, resource, result, []AdapterOwnershipDefinition{definition})

	var wait sync.WaitGroup
	errorsCh := make(chan error, 100)
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			execution, err := adapter.Apply(context.Background(), input)
			if err != nil || execution.Mode() != AdapterExecutionModeApplyFilter {
				errorsCh <- err
			}
		}()
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("concurrent adapter apply failed: %v", err)
	}
	if executor.callCount() != 100 {
		t.Fatalf("executor calls = %d, want 100", executor.callCount())
	}
}

type registeredFieldTestExecutor struct {
	mu       sync.Mutex
	requests []RegisteredFieldExecutionRequest
	err      error
}

func (executor *registeredFieldTestExecutor) Prepare(
	_ context.Context,
	request RegisteredFieldExecutionRequest,
) error {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.requests = append(executor.requests, RegisteredFieldExecutionRequest{
		resource:  request.ResourceContext(),
		condition: request.Condition(),
	})
	return executor.err
}

func (executor *registeredFieldTestExecutor) callCount() int {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return len(executor.requests)
}

func (executor *registeredFieldTestExecutor) snapshot() []RegisteredFieldExecutionRequest {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	result := make([]RegisteredFieldExecutionRequest, len(executor.requests))
	for index, request := range executor.requests {
		result[index] = RegisteredFieldExecutionRequest{
			resource:  request.ResourceContext(),
			condition: request.Condition(),
		}
	}
	return result
}

func registeredFieldTestRegistration(
	resourceCode string,
	adapterCode string,
	adapterFieldCode string,
	dimensionIds []int,
	executor RegisteredFieldExecutor,
) RegisteredFieldExecutionRegistration {
	return RegisteredFieldExecutionRegistration{
		ResourceCode:        resourceCode,
		AdapterCode:         adapterCode,
		AdapterFieldCode:    adapterFieldCode,
		DimensionIds:        append([]int(nil), dimensionIds...),
		ValueType:           DataScopeValueTypeBigint,
		SupportedOperations: []string{"query", "detail", "update", "delete", "export"},
		SupportedOperators:  []DataScopeOperator{DataScopeOperatorEqual, DataScopeOperatorIn},
		Executor:            executor,
	}
}

func mustRegisteredFieldRegistry(
	t *testing.T,
	registrations ...RegisteredFieldExecutionRegistration,
) *RegisteredFieldExecutionRegistry {
	t.Helper()
	registry, err := NewRegisteredFieldExecutionRegistry(registrations...)
	if err != nil {
		t.Fatalf("create registered field registry: %v", err)
	}
	return registry
}

func mustRegisteredFieldAdapter(
	t *testing.T,
	registry *RegisteredFieldExecutionRegistry,
) *RegisteredFieldAdapter {
	t.Helper()
	adapter, err := NewRegisteredFieldAdapter(registry)
	if err != nil {
		t.Fatalf("create registered field adapter: %v", err)
	}
	return adapter
}

func registeredFieldTestResource(
	t *testing.T,
	resourceCode string,
	operation string,
	adapterCode string,
) AdapterResourceContext {
	t.Helper()
	resource, err := NewAdapterResourceContext(AdapterResourceContextInput{
		ResourceCode: resourceCode,
		Operation:    operation,
		AdapterCode:  adapterCode,
	})
	if err != nil {
		t.Fatalf("create registered field resource: %v", err)
	}
	return resource
}

func mustRegisteredFieldOwnership(
	t *testing.T,
	ownershipCode string,
	dimensionId int,
	adapterFieldCode string,
	valueType DataScopeValueType,
) AdapterOwnershipDefinition {
	t.Helper()
	definition, err := NewAdapterOwnershipDefinition(AdapterOwnershipDefinitionInput{
		OwnershipCode:    ownershipCode,
		DimensionId:      dimensionId,
		BindingType:      AdapterBindingTypeRegisteredField,
		AdapterFieldCode: adapterFieldCode,
		ValueType:        valueType,
	})
	if err != nil {
		t.Fatalf("create registered ownership: %v", err)
	}
	return definition
}

func mustRegisteredFieldCondition(
	t *testing.T,
	ownershipCode string,
	dimensionId int,
	operator DataScopeOperator,
	valueType DataScopeValueType,
	values []any,
) DataScopeCondition {
	t.Helper()
	condition, err := NewDataScopeCondition(DataScopeConditionInput{
		OwnershipCode: ownershipCode,
		DimensionId:   dimensionId,
		Operator:      operator,
		ValueType:     valueType,
		Values:        values,
	})
	if err != nil {
		t.Fatalf("create registered condition: %v", err)
	}
	return condition
}

func mustRegisteredFieldResult(
	t *testing.T,
	resource AdapterResourceContext,
	conditionGroups [][]DataScopeCondition,
) DataScopeResult {
	t.Helper()
	groups := make([]DataScopeConditionGroup, 0, len(conditionGroups))
	for _, conditions := range conditionGroups {
		group, err := NewDataScopeConditionGroup(conditions)
		if err != nil {
			t.Fatalf("create registered condition group: %v", err)
		}
		groups = append(groups, group)
	}
	result, err := NewFilteredResult(resource.ResourceCode(), resource.Operation(), groups)
	if err != nil {
		t.Fatalf("create registered result: %v", err)
	}
	return result
}

func mustRegisteredFieldInput(
	t *testing.T,
	resource AdapterResourceContext,
	result DataScopeResult,
	definitions []AdapterOwnershipDefinition,
) AdapterInput {
	t.Helper()
	input, err := NewAdapterInput(resource, result, definitions)
	if err != nil {
		t.Fatalf("create registered AdapterInput: %v", err)
	}
	return input
}

func assertRegisteredAdapterError(t *testing.T, err error, code int) {
	t.Helper()
	if !isAdminErrorCode(err, code) {
		t.Fatalf("error = %v, want code %d", err, code)
	}
}

func isAdminErrorCode(err error, code int) bool {
	var adminError *response.AdminError
	return errors.As(err, &adminError) && adminError.ErrorCode == code
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
