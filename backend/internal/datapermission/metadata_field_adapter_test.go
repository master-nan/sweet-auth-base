package datapermission

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
)

func TestMetadataFieldAdapterExecutionModes(t *testing.T) {
	reader := newMetadataAdapterTestReader()
	adapter := mustMetadataAdapter(t, reader)
	resource := adapterTestResource(t)
	tests := []struct {
		name     string
		result   func(string, string) (DataScopeResult, error)
		wantMode AdapterExecutionMode
	}{
		{name: "not applicable", result: NewNotApplicableResult, wantMode: AdapterExecutionModeNotApplicable},
		{name: "allow all", result: NewAllResult, wantMode: AdapterExecutionModeAllowAll},
		{name: "deny all", result: NewNoneResult, wantMode: AdapterExecutionModeDenyAll},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.result(resource.ResourceCode(), resource.Operation())
			if err != nil {
				t.Fatalf("create result: %v", err)
			}
			input := mustMetadataAdapterInput(t, resource, result, nil)
			execution, err := adapter.Apply(context.Background(), input)
			if err != nil {
				t.Fatalf("apply metadata adapter: %v", err)
			}
			if execution.Mode() != test.wantMode || len(execution.ConditionGroups()) != 0 {
				t.Fatalf("mode=%s groups=%d, want %s with no groups", execution.Mode(), len(execution.ConditionGroups()), test.wantMode)
			}
		})
	}
	if reader.tableCallCount() != 0 || reader.fieldCallCount(501) != 0 {
		t.Fatalf("non-filter decisions unexpectedly loaded metadata")
	}
}

func TestMetadataFieldAdapterEmptyFilteredResultStaysDenyAll(t *testing.T) {
	resource := adapterTestResource(t)
	result, err := NewFilteredResult(resource.ResourceCode(), resource.Operation(), nil)
	if err != nil {
		t.Fatalf("normalize empty filtered result: %v", err)
	}
	input := mustMetadataAdapterInput(t, resource, result, nil)
	execution, err := mustMetadataAdapter(t, newMetadataAdapterTestReader()).Apply(context.Background(), input)
	if err != nil {
		t.Fatalf("apply normalized empty filter: %v", err)
	}
	if execution.Mode() != AdapterExecutionModeDenyAll || len(execution.ConditionGroups()) != 0 {
		t.Fatalf("empty filter expanded permission: %+v", execution)
	}
}

func TestMetadataFieldAdapterResolvesAndValidatesMetadata(t *testing.T) {
	t.Run("normal field", func(t *testing.T) {
		reader := newMetadataAdapterTestReader()
		execution := applyDefaultMetadataFilter(t, reader)
		groups := execution.ConditionGroups()
		if execution.Mode() != AdapterExecutionModeApplyFilter || len(groups) != 1 || len(groups[0].Conditions()) != 1 {
			t.Fatalf("unexpected execution: %+v", execution)
		}
		condition := groups[0].Conditions()[0]
		if condition.TableFieldId() != 501 || condition.BindingType() != AdapterBindingTypeMetadataField {
			t.Fatalf("unexpected metadata binding: %+v", condition)
		}
	})

	tests := []struct {
		name       string
		configure  func(*metadataAdapterTestReader)
		resourceId int
		fieldId    int
		wantCode   int
	}{
		{
			name: "table not found",
			configure: func(reader *metadataAdapterTestReader) {
				delete(reader.tables, 101)
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterTableNotFound,
		},
		{
			name: "table disabled",
			configure: func(reader *metadataAdapterTestReader) {
				table := reader.tables[101]
				table.State = false
				reader.tables[101] = table
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterTableNotFound,
		},
		{
			name: "table deleted",
			configure: func(reader *metadataAdapterTestReader) {
				table := reader.tables[101]
				table.Deleted = true
				reader.tables[101] = table
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterTableNotFound,
		},
		{
			name: "field not found",
			configure: func(reader *metadataAdapterTestReader) {
				delete(reader.fields, 501)
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterFieldNotFound,
		},
		{
			name: "field belongs to another table",
			configure: func(reader *metadataAdapterTestReader) {
				field := reader.fields[501]
				field.TableId = 999
				reader.fields[501] = field
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterFieldResourceMismatch,
		},
		{
			name: "field disabled",
			configure: func(reader *metadataAdapterTestReader) {
				field := reader.fields[501]
				field.State = false
				reader.fields[501] = field
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterFieldInactive,
		},
		{
			name: "field deleted",
			configure: func(reader *metadataAdapterTestReader) {
				field := reader.fields[501]
				field.Deleted = true
				reader.fields[501] = field
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterFieldInactive,
		},
		{
			name: "unsupported field type",
			configure: func(reader *metadataAdapterTestReader) {
				field := reader.fields[501]
				field.FieldType = enum.FloatFieldType
				reader.fields[501] = field
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterFieldTypeUnsupported,
		},
		{
			name: "field type drift",
			configure: func(reader *metadataAdapterTestReader) {
				field := reader.fields[501]
				field.FieldType = enum.VarcharFieldType
				reader.fields[501] = field
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterFieldTypeDrift,
		},
		{
			name: "table lookup failure",
			configure: func(reader *metadataAdapterTestReader) {
				reader.tableErrors[101] = errors.New("storage unavailable")
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterFailed,
		},
		{
			name: "field lookup failure",
			configure: func(reader *metadataAdapterTestReader) {
				reader.fieldErrors[501] = errors.New("storage unavailable")
			},
			wantCode: myerrors.ErrorCodeMetadataAdapterFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := newMetadataAdapterTestReader()
			test.configure(reader)
			adapter := mustMetadataAdapter(t, reader)
			input := defaultMetadataAdapterInput(t)
			execution, err := adapter.Apply(context.Background(), input)
			assertMetadataAdapterError(t, err, test.wantCode)
			assertAdapterExecutionEmpty(t, execution)
		})
	}
}

func TestMetadataFieldAdapterRejectsMissingTableBinding(t *testing.T) {
	resource, err := NewAdapterResourceContext(AdapterResourceContextInput{
		ResourceCode: "transport_order",
		Operation:    "query",
		AdapterCode:  "transport_order_filter",
	})
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}
	condition := mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(11)})
	result := mustMetadataResult(t, resource, [][]DataScopeCondition{{condition}})
	definition := mustMetadataOwnership(t, "owner_org", 201, 501, DataScopeValueTypeBigint)
	input := mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{definition})
	execution, err := mustMetadataAdapter(t, newMetadataAdapterTestReader()).Apply(context.Background(), input)
	assertMetadataAdapterError(t, err, myerrors.ErrorCodeMetadataAdapterResourceTableMissing)
	assertAdapterExecutionEmpty(t, execution)
}

func TestMetadataFieldAdapterRejectsForbiddenFilterFields(t *testing.T) {
	expression := "owner_org_id + 1"
	tests := []struct {
		name      string
		configure func(*MetadataFieldRecord)
	}{
		{name: "advanced query disabled", configure: func(field *MetadataFieldRecord) { field.IsAdvancedSearch = false }},
		{name: "primary key", configure: func(field *MetadataFieldRecord) { field.IsPrimaryKey = true }},
		{name: "calculated field", configure: func(field *MetadataFieldRecord) { field.FieldCategory = enum.CalculatedField }},
		{name: "virtual field", configure: func(field *MetadataFieldRecord) { field.FieldCategory = enum.VirtualField }},
		{name: "field expression", configure: func(field *MetadataFieldRecord) { field.Expression = expression }},
		{name: "file input", configure: func(field *MetadataFieldRecord) { field.InputType = enum.FilePickerInputType }},
		{name: "rich text input", configure: func(field *MetadataFieldRecord) { field.InputType = enum.RichTextInputType }},
		{name: "technical id", configure: func(field *MetadataFieldRecord) { field.FieldCode = "id" }},
		{name: "technical timestamp", configure: func(field *MetadataFieldRecord) { field.FieldCode = "gmt_modify" }},
		{name: "source field", configure: func(field *MetadataFieldRecord) { field.FieldCode = "source_version" }},
		{name: "tree path", configure: func(field *MetadataFieldRecord) { field.FieldCode = "path" }},
		{name: "tree level", configure: func(field *MetadataFieldRecord) { field.FieldCode = "level" }},
		{name: "structure node", configure: func(field *MetadataFieldRecord) { field.FieldCode = "structure_node_id" }},
		{name: "parent node", configure: func(field *MetadataFieldRecord) { field.FieldCode = "parent_node_id" }},
		{name: "display name", configure: func(field *MetadataFieldRecord) { field.FieldCode = "owner_org_name" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := newMetadataAdapterTestReader()
			field := reader.fields[501]
			test.configure(&field)
			reader.fields[501] = field
			execution, err := mustMetadataAdapter(t, reader).Apply(context.Background(), defaultMetadataAdapterInput(t))
			assertMetadataAdapterError(t, err, myerrors.ErrorCodeMetadataAdapterFieldNotFilterable)
			assertAdapterExecutionEmpty(t, execution)
		})
	}
}

func TestMetadataFieldAdapterMapsSupportedFieldTypes(t *testing.T) {
	tests := []struct {
		name      string
		fieldType enum.SysTableFieldType
		valueType DataScopeValueType
		values    []any
	}{
		{name: "bigint", fieldType: enum.BigIntFieldType, valueType: DataScopeValueTypeBigint, values: []any{int64(1)}},
		{name: "int", fieldType: enum.IntFieldType, valueType: DataScopeValueTypeBigint, values: []any{int64(1)}},
		{name: "tinyint", fieldType: enum.TinyintFieldType, valueType: DataScopeValueTypeBigint, values: []any{int64(1)}},
		{name: "varchar", fieldType: enum.VarcharFieldType, valueType: DataScopeValueTypeString, values: []any{"ORG-A"}},
		{name: "text", fieldType: enum.TextFieldType, valueType: DataScopeValueTypeString, values: []any{"ORG-A"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := newMetadataAdapterTestReader()
			field := reader.fields[501]
			field.FieldType = test.fieldType
			reader.fields[501] = field
			resource := adapterTestResource(t)
			condition := mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, test.valueType, test.values)
			result := mustMetadataResult(t, resource, [][]DataScopeCondition{{condition}})
			definition := mustMetadataOwnership(t, "owner_org", 201, 501, test.valueType)
			input := mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{definition})
			if _, err := mustMetadataAdapter(t, reader).Apply(context.Background(), input); err != nil {
				t.Fatalf("apply supported field type: %v", err)
			}
		})
	}
}

func TestMetadataFieldAdapterPreservesAndOrFilterTree(t *testing.T) {
	reader := newMetadataAdapterTestReader()
	reader.fields[502] = MetadataFieldRecord{
		Id: 502, TableId: 101, State: true, FieldCode: "legal_entity_id",
		FieldType: enum.BigIntFieldType, InputType: enum.InputNumberInputType,
		FieldCategory: enum.NormalField, IsAdvancedSearch: true,
	}
	resource := adapterTestResource(t)
	ownerOrg := mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(11), int64(12)})
	legalEntity := mustMetadataCondition(t, "legal_entity", 202, DataScopeOperatorEqual, DataScopeValueTypeBigint, []any{int64(21)})
	otherOrg := mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorEqual, DataScopeValueTypeBigint, []any{int64(99)})
	result := mustMetadataResult(t, resource, [][]DataScopeCondition{{ownerOrg, legalEntity}, {otherOrg}})
	input := mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{
		mustMetadataOwnership(t, "owner_org", 201, 501, DataScopeValueTypeBigint),
		mustMetadataOwnership(t, "legal_entity", 202, 502, DataScopeValueTypeBigint),
	})
	execution, err := mustMetadataAdapter(t, reader).Apply(context.Background(), input)
	if err != nil {
		t.Fatalf("apply grouped filter: %v", err)
	}
	groups := execution.ConditionGroups()
	if len(groups) != 2 {
		t.Fatalf("OR group count=%d, want 2", len(groups))
	}
	conditionCounts := map[int]int{}
	for _, group := range groups {
		conditionCounts[len(group.Conditions())]++
	}
	if conditionCounts[1] != 1 || conditionCounts[2] != 1 {
		t.Fatalf("AND conditions were not preserved: %+v", conditionCounts)
	}
	if reader.tableCallCount() != 1 || reader.fieldCallCount(501) != 1 || reader.fieldCallCount(502) != 1 {
		t.Fatalf("request cache missed: table=%d field501=%d field502=%d", reader.tableCallCount(), reader.fieldCallCount(501), reader.fieldCallCount(502))
	}
}

func TestMetadataFieldAdapterExecutionIsReusableAcrossQueryChains(t *testing.T) {
	for _, operation := range []string{"query", "detail", "update", "delete", "export"} {
		t.Run(operation, func(t *testing.T) {
			resource, err := NewAdapterResourceContext(AdapterResourceContextInput{
				ResourceCode: "transport_order",
				Operation:    operation,
				AdapterCode:  "transport_order_filter",
				TableId:      101,
			})
			if err != nil {
				t.Fatalf("create resource context: %v", err)
			}
			condition := mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(11)})
			result := mustMetadataResult(t, resource, [][]DataScopeCondition{{condition}})
			input := mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{
				mustMetadataOwnership(t, "owner_org", 201, 501, DataScopeValueTypeBigint),
			})
			execution, err := mustMetadataAdapter(t, newMetadataAdapterTestReader()).Apply(context.Background(), input)
			if err != nil {
				t.Fatalf("apply %s metadata filter: %v", operation, err)
			}
			if execution.Operation() != operation || execution.Mode() != AdapterExecutionModeApplyFilter {
				t.Fatalf("operation=%s mode=%s, want %s/apply_filter", execution.Operation(), execution.Mode(), operation)
			}
		})
	}
}

func TestMetadataFieldAdapterValidatesOperatorsAndBindings(t *testing.T) {
	t.Run("eq requires one value", func(t *testing.T) {
		condition := DataScopeCondition{operator: DataScopeOperatorEqual, valueType: DataScopeValueTypeBigint, bigintValues: []int64{1, 2}}
		assertMetadataAdapterError(t, validateMetadataOperator(condition), myerrors.ErrorCodeMetadataAdapterValueTypeMismatch)
	})
	t.Run("in requires values", func(t *testing.T) {
		condition := DataScopeCondition{operator: DataScopeOperatorIn, valueType: DataScopeValueTypeBigint}
		assertMetadataAdapterError(t, validateMetadataOperator(condition), myerrors.ErrorCodeMetadataAdapterValueTypeMismatch)
	})
	t.Run("unknown operator", func(t *testing.T) {
		condition := DataScopeCondition{operator: "like", valueType: DataScopeValueTypeString, stringValues: []string{"x"}}
		assertMetadataAdapterError(t, validateMetadataOperator(condition), myerrors.ErrorCodeMetadataAdapterOperatorUnsupported)
	})
	t.Run("ownership missing", func(t *testing.T) {
		input := defaultMetadataAdapterInputWithoutDefinitions(t)
		execution, err := mustMetadataAdapter(t, newMetadataAdapterTestReader()).Apply(context.Background(), input)
		assertMetadataAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterOwnershipMissing)
		assertAdapterExecutionEmpty(t, execution)
	})
	t.Run("dimension mismatch", func(t *testing.T) {
		resource := adapterTestResource(t)
		result := mustMetadataResult(t, resource, [][]DataScopeCondition{{
			mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(1)}),
		}})
		definition := mustMetadataOwnership(t, "owner_org", 999, 501, DataScopeValueTypeBigint)
		input := mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{definition})
		execution, err := mustMetadataAdapter(t, newMetadataAdapterTestReader()).Apply(context.Background(), input)
		assertMetadataAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterOwnershipMismatch)
		assertAdapterExecutionEmpty(t, execution)
	})
	t.Run("value type mismatch", func(t *testing.T) {
		resource := adapterTestResource(t)
		result := mustMetadataResult(t, resource, [][]DataScopeCondition{{
			mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(1)}),
		}})
		definition := mustMetadataOwnership(t, "owner_org", 201, 501, DataScopeValueTypeString)
		input := mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{definition})
		execution, err := mustMetadataAdapter(t, newMetadataAdapterTestReader()).Apply(context.Background(), input)
		assertMetadataAdapterError(t, err, myerrors.ErrorCodeMetadataAdapterValueTypeMismatch)
		assertAdapterExecutionEmpty(t, execution)
	})
	t.Run("registered binding rejected", func(t *testing.T) {
		resource := adapterTestResource(t)
		condition := mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(1)})
		result := mustMetadataResult(t, resource, [][]DataScopeCondition{{condition}})
		definition := adapterTestOwnership(t, "owner_org", 201, AdapterBindingTypeRegisteredField, 0, "owner_org_id")
		input := mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{definition})
		execution, err := mustMetadataAdapter(t, newMetadataAdapterTestReader()).Apply(context.Background(), input)
		assertMetadataAdapterError(t, err, myerrors.ErrorCodeDataPermissionAdapterTypeUnsupported)
		assertAdapterExecutionEmpty(t, execution)
	})
}

func TestMetadataFieldAdapterFailsAtomicallyAndKeepsOutputIsolated(t *testing.T) {
	reader := newMetadataAdapterTestReader()
	reader.fields[502] = MetadataFieldRecord{
		Id: 502, TableId: 999, State: true, FieldCode: "legal_entity_id",
		FieldType: enum.BigIntFieldType, InputType: enum.InputNumberInputType,
		FieldCategory: enum.NormalField, IsAdvancedSearch: true,
	}
	resource := adapterTestResource(t)
	result := mustMetadataResult(t, resource, [][]DataScopeCondition{{
		mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(11)}),
		mustMetadataCondition(t, "legal_entity", 202, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(21)}),
	}})
	input := mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{
		mustMetadataOwnership(t, "owner_org", 201, 501, DataScopeValueTypeBigint),
		mustMetadataOwnership(t, "legal_entity", 202, 502, DataScopeValueTypeBigint),
	})
	execution, err := mustMetadataAdapter(t, reader).Apply(context.Background(), input)
	assertMetadataAdapterError(t, err, myerrors.ErrorCodeMetadataAdapterFieldResourceMismatch)
	assertAdapterExecutionEmpty(t, execution)

	validExecution := applyDefaultMetadataFilter(t, newMetadataAdapterTestReader())
	groups := validExecution.ConditionGroups()
	groups[0].conditions = nil
	if len(validExecution.ConditionGroups()[0].Conditions()) != 1 {
		t.Fatalf("output group mutation changed immutable execution")
	}
	condition := validExecution.ConditionGroups()[0].Conditions()[0].ScopeCondition()
	values := condition.BigintValues()
	values[0] = 999
	if validExecution.ConditionGroups()[0].Conditions()[0].ScopeCondition().BigintValues()[0] == 999 {
		t.Fatalf("output value mutation changed immutable execution")
	}
}

func TestMetadataFieldAdapterOutputHasNoExecutableMetadata(t *testing.T) {
	execution := applyDefaultMetadataFilter(t, newMetadataAdapterTestReader())
	payload, err := json.Marshal(execution)
	if err != nil {
		t.Fatalf("marshal execution: %v", err)
	}
	encoded := strings.ToLower(string(payload))
	for _, forbidden := range []string{"sql", "table_name", "table_code", "field_name", "field_code", "column", "join", "gorm", "expression"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("metadata adapter output leaked %q: %s", forbidden, encoded)
		}
	}
	if !strings.Contains(encoded, "table_field_id") {
		t.Fatalf("metadata adapter output omitted stable table_field_id: %s", encoded)
	}
}

func TestMetadataFieldAdapterComplexityGuard(t *testing.T) {
	execution := AdapterExecution{
		resourceCode: "transport_order",
		operation:    "query",
		mode:         AdapterExecutionModeApplyFilter,
		groups:       make([]AdapterConditionGroup, DataScopeMaxConditionGroups+1),
	}
	assertMetadataAdapterError(t, validateMetadataExecutionComplexity(execution), myerrors.ErrorCodeMetadataAdapterComplexityExceeded)
}

func TestMetadataFieldAdapterConcurrentReads(t *testing.T) {
	reader := newMetadataAdapterTestReader()
	adapter := mustMetadataAdapter(t, reader)
	input := defaultMetadataAdapterInput(t)
	const workers = 64
	var wait sync.WaitGroup
	errorsChannel := make(chan error, workers)
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			execution, err := adapter.Apply(context.Background(), input)
			if err == nil && execution.Mode() != AdapterExecutionModeApplyFilter {
				err = errors.New("unexpected execution mode")
			}
			errorsChannel <- err
		}()
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent Apply failed: %v", err)
		}
	}
}

type metadataAdapterTestReader struct {
	mutex       sync.Mutex
	tables      map[int]MetadataTableRecord
	fields      map[int]MetadataFieldRecord
	tableErrors map[int]error
	fieldErrors map[int]error
	tableCalls  map[int]int
	fieldCalls  map[int]int
}

func newMetadataAdapterTestReader() *metadataAdapterTestReader {
	return &metadataAdapterTestReader{
		tables: map[int]MetadataTableRecord{
			101: {Id: 101, State: true},
		},
		fields: map[int]MetadataFieldRecord{
			501: {
				Id: 501, TableId: 101, State: true, FieldCode: "owner_org_id",
				FieldType: enum.BigIntFieldType, InputType: enum.InputNumberInputType,
				FieldCategory: enum.NormalField, IsAdvancedSearch: true,
			},
		},
		tableErrors: make(map[int]error),
		fieldErrors: make(map[int]error),
		tableCalls:  make(map[int]int),
		fieldCalls:  make(map[int]int),
	}
}

func (reader *metadataAdapterTestReader) FindMetadataTable(
	_ context.Context,
	tableId int,
) (MetadataTableRecord, error) {
	reader.mutex.Lock()
	defer reader.mutex.Unlock()
	reader.tableCalls[tableId]++
	if err := reader.tableErrors[tableId]; err != nil {
		return MetadataTableRecord{}, err
	}
	table, exists := reader.tables[tableId]
	if !exists {
		return MetadataTableRecord{}, ErrMetadataTableRecordNotFound
	}
	return table, nil
}

func (reader *metadataAdapterTestReader) FindMetadataField(
	_ context.Context,
	fieldId int,
) (MetadataFieldRecord, error) {
	reader.mutex.Lock()
	defer reader.mutex.Unlock()
	reader.fieldCalls[fieldId]++
	if err := reader.fieldErrors[fieldId]; err != nil {
		return MetadataFieldRecord{}, err
	}
	field, exists := reader.fields[fieldId]
	if !exists {
		return MetadataFieldRecord{}, ErrMetadataFieldRecordNotFound
	}
	return field, nil
}

func (reader *metadataAdapterTestReader) tableCallCount() int {
	reader.mutex.Lock()
	defer reader.mutex.Unlock()
	total := 0
	for _, count := range reader.tableCalls {
		total += count
	}
	return total
}

func (reader *metadataAdapterTestReader) fieldCallCount(fieldId int) int {
	reader.mutex.Lock()
	defer reader.mutex.Unlock()
	return reader.fieldCalls[fieldId]
}

func mustMetadataAdapter(t *testing.T, reader MetadataFieldReader) *MetadataFieldAdapter {
	t.Helper()
	adapter, err := NewMetadataFieldAdapter(reader)
	if err != nil {
		t.Fatalf("create metadata adapter: %v", err)
	}
	return adapter
}

func defaultMetadataAdapterInput(t *testing.T) AdapterInput {
	t.Helper()
	resource := adapterTestResource(t)
	condition := mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(11), int64(12)})
	result := mustMetadataResult(t, resource, [][]DataScopeCondition{{condition}})
	definition := mustMetadataOwnership(t, "owner_org", 201, 501, DataScopeValueTypeBigint)
	return mustMetadataAdapterInput(t, resource, result, []AdapterOwnershipDefinition{definition})
}

func defaultMetadataAdapterInputWithoutDefinitions(t *testing.T) AdapterInput {
	t.Helper()
	resource := adapterTestResource(t)
	condition := mustMetadataCondition(t, "owner_org", 201, DataScopeOperatorIn, DataScopeValueTypeBigint, []any{int64(11)})
	result := mustMetadataResult(t, resource, [][]DataScopeCondition{{condition}})
	return mustMetadataAdapterInput(t, resource, result, nil)
}

func applyDefaultMetadataFilter(
	t *testing.T,
	reader MetadataFieldReader,
) AdapterExecution {
	t.Helper()
	execution, err := mustMetadataAdapter(t, reader).Apply(context.Background(), defaultMetadataAdapterInput(t))
	if err != nil {
		t.Fatalf("apply default metadata filter: %v", err)
	}
	return execution
}

func mustMetadataAdapterInput(
	t *testing.T,
	resource AdapterResourceContext,
	result DataScopeResult,
	definitions []AdapterOwnershipDefinition,
) AdapterInput {
	t.Helper()
	input, err := NewAdapterInput(resource, result, definitions)
	if err != nil {
		t.Fatalf("create metadata AdapterInput: %v", err)
	}
	return input
}

func mustMetadataOwnership(
	t *testing.T,
	ownershipCode string,
	dimensionId int,
	fieldId int,
	valueType DataScopeValueType,
) AdapterOwnershipDefinition {
	t.Helper()
	definition, err := NewAdapterOwnershipDefinition(AdapterOwnershipDefinitionInput{
		OwnershipCode: ownershipCode,
		DimensionId:   dimensionId,
		BindingType:   AdapterBindingTypeMetadataField,
		TableFieldId:  fieldId,
		ValueType:     valueType,
	})
	if err != nil {
		t.Fatalf("create metadata Ownership definition: %v", err)
	}
	return definition
}

func mustMetadataCondition(
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
		t.Fatalf("create metadata condition: %v", err)
	}
	return condition
}

func mustMetadataResult(
	t *testing.T,
	resource AdapterResourceContext,
	conditionGroups [][]DataScopeCondition,
) DataScopeResult {
	t.Helper()
	groups := make([]DataScopeConditionGroup, 0, len(conditionGroups))
	for _, conditions := range conditionGroups {
		group, err := NewDataScopeConditionGroup(conditions)
		if err != nil {
			t.Fatalf("create metadata condition group: %v", err)
		}
		groups = append(groups, group)
	}
	result, err := NewFilteredResult(resource.ResourceCode(), resource.Operation(), groups)
	if err != nil {
		t.Fatalf("create metadata result: %v", err)
	}
	return result
}

func assertMetadataAdapterError(t *testing.T, err error, code int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected metadata adapter error code %d", code)
	}
	var adminError *response.AdminError
	if !errors.As(err, &adminError) {
		t.Fatalf("error is not AdminError: %T %v", err, err)
	}
	if adminError.ErrorCode != code {
		t.Fatalf("error code=%d, want %d: %v", adminError.ErrorCode, code, err)
	}
}
