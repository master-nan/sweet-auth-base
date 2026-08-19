package queryscheme

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/metadata"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type metadataReaderStub struct {
	table metadata.TableMetadata
}

func (stub metadataReaderStub) GetTable(context.Context, string) (metadata.TableMetadata, error) {
	return stub.table, nil
}
func (stub metadataReaderStub) GetTableByID(context.Context, int) (metadata.TableMetadata, error) {
	return stub.table, nil
}
func (stub metadataReaderStub) GetField(context.Context, int) (metadata.FieldMetadata, error) {
	return stub.table.Fields[0], nil
}
func (stub metadataReaderStub) GetFields(context.Context, int) ([]metadata.FieldMetadata, error) {
	return stub.table.Fields, nil
}
func (stub metadataReaderStub) ListTables(context.Context) ([]metadata.TableMetadata, error) {
	return []metadata.TableMetadata{stub.table}, nil
}

type fixedClock struct{ value time.Time }

func (clock fixedClock) Now() time.Time { return clock.value }

func TestValidateSchemaLimitsDepthRulesAndMultiValues(t *testing.T) {
	valid := testPayload()
	if result := ValidateSchema(valid); result.Status != ValidationValid {
		t.Fatalf("valid payload: %+v", result)
	}

	depthFour := valid
	depthFour.Expressions[0].Nested = []request.ExpressionGroup{{Logic: enum.And, Nested: []request.ExpressionGroup{{Logic: enum.And, Nested: []request.ExpressionGroup{{Logic: enum.And}}}}}}
	if result := ValidateSchema(depthFour); result.Status != ValidationInvalid {
		t.Fatalf("depth four status = %s", result.Status)
	}

	tooManyRules := valid
	tooManyRules.Expressions[0].Rules = make([]request.QueryRule, MaxRules+1)
	for index := range tooManyRules.Expressions[0].Rules {
		tooManyRules.Expressions[0].Rules[index] = valid.Expressions[0].Rules[0]
	}
	if result := ValidateSchema(tooManyRules); result.Status != ValidationInvalid {
		t.Fatalf("51 rule status = %s", result.Status)
	}

	tooManyValues := valid
	values := make([]any, MaxMultiValues+1)
	tooManyValues.Expressions[0].Rules[0].ExpressionType = enum.In
	tooManyValues.Expressions[0].Rules[0].Value = values
	if result := ValidateSchema(tooManyValues); result.Status != ValidationInvalid {
		t.Fatalf("101 values status = %s", result.Status)
	}
}

func TestValidatorReturnsDegradedWithoutDroppingRules(t *testing.T) {
	payload := testPayload()
	reader := metadataReaderStub{table: metadata.TableMetadata{Code: "orders", Fields: []metadata.FieldMetadata{{
		Code: "status", StorageType: enum.VarcharFieldType, AdvancedQuery: false, Sortable: true,
	}}}}
	result, err := NewValidator(reader).ValidateMetadata(context.Background(), ScopeConfig{TableCode: "orders"}, payload)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != ValidationDegraded || len(result.Issues) != 1 || result.Issues[0].FieldCode != "status" {
		t.Fatalf("unexpected degraded result: %+v", result)
	}
	if len(payload.Expressions[0].Rules) != 1 {
		t.Fatal("validation changed the source query")
	}
}

func TestValidatorRejectsIncompatibleOperatorAndSort(t *testing.T) {
	payload := testPayload()
	payload.Expressions[0].Rules[0].ExpressionType = enum.Gt
	payload.Order.Field = "not_sortable"
	reader := metadataReaderStub{table: metadata.TableMetadata{Code: "orders", Fields: []metadata.FieldMetadata{
		{Code: "status", StorageType: enum.VarcharFieldType, AdvancedQuery: true, Sortable: true, DictionaryCode: stringPointer("order_status")},
		{Code: "not_sortable", StorageType: enum.VarcharFieldType, AdvancedQuery: true, Sortable: false},
	}}}
	result, err := NewValidator(reader).ValidateMetadata(context.Background(), ScopeConfig{TableCode: "orders"}, payload)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[IssueCode]bool{}
	for _, issue := range result.Issues {
		codes[issue.Code] = true
	}
	if result.Status != ValidationDegraded || !codes[IssueOperatorIncompatible] || !codes[IssueSortUnavailable] {
		t.Fatalf("unexpected issues: %+v", result)
	}
}

func TestBindingResolverUsesWhitelistSubjectAndApplicationTimezone(t *testing.T) {
	payload := testPayload()
	payload.Expressions[0].Rules = append(payload.Expressions[0].Rules,
		request.QueryRule{Field: "created_at", ExpressionType: enum.Between, Value: []any{"", ""}, Type: enum.DateFieldType},
		request.QueryRule{Field: "owner_id", ExpressionType: enum.Eq, Value: 0, Type: enum.BigIntFieldType},
	)
	payload.Bindings = []Binding{
		{Pointer: "/expressions/0/rules/1/value/0", Kind: BindingStartOfMonth},
		{Pointer: "/expressions/0/rules/1/value/1", Kind: BindingEndOfMonth},
		{Pointer: "/expressions/0/rules/2/value", Kind: BindingCurrentUser},
	}
	location, _ := time.LoadLocation("Asia/Shanghai")
	resolver := NewBindingResolver(fixedClock{value: time.Date(2027, 1, 1, 0, 30, 0, 0, location)}, location)
	resolved, err := resolver.Resolve(context.Background(), payload, ScopeConfig{AllowedDynamicBindings: BindingKinds()}, Subject{UserID: 42})
	if err != nil {
		t.Fatal(err)
	}
	rangeValues := resolved.Expressions[0].Rules[1].Value.([]any)
	if rangeValues[0] != "2027-01-01" || rangeValues[1] != "2027-01-31" {
		t.Fatalf("resolved month range = %#v", rangeValues)
	}
	if resolved.Expressions[0].Rules[2].Value != float64(42) {
		t.Fatalf("resolved user = %#v", resolved.Expressions[0].Rules[2].Value)
	}
}

func TestBindingResolverRejectsClientControlledAndUnknownBindings(t *testing.T) {
	payload := testPayload()
	payload.Bindings = []Binding{{
		Pointer: "/expressions/0/rules/0/value", Kind: BindingCurrentUser,
		Params: BindingParams{DayOffset: intPointer(1)},
	}}
	resolver := NewBindingResolver(fixedClock{value: time.Now()}, time.UTC)
	if _, err := resolver.Resolve(context.Background(), payload, ScopeConfig{AllowedDynamicBindings: BindingKinds()}, Subject{UserID: 99}); err == nil {
		t.Fatal("expected identity binding parameters to be rejected")
	}
	payload.Bindings[0] = Binding{Pointer: "/expressions/0/rules/0/value", Kind: BindingKind("CUSTOM_SCRIPT")}
	if result := ValidateSchema(payload); result.Status != ValidationInvalid {
		t.Fatalf("unknown binding status = %s", result.Status)
	}
}

func TestOrganizationScopesAllowTheirExistingSystemSort(t *testing.T) {
	registry := NewRegistry()
	config, ok := registry.Get(context.Background(), "organization.position.list")
	if !ok || !config.AllowsSort("gmt_modify") {
		t.Fatalf("organization position scope must allow its existing gmt_modify sort: %+v", config)
	}
	if config.AllowsSort("unregistered_field") {
		t.Fatal("scope accepted an unregistered sort field")
	}
}

func TestBindingResolverCurrentEmployeeAndWeekBoundary(t *testing.T) {
	payload := testPayload()
	payload.Expressions[0].Rules = []request.QueryRule{
		{Field: "week", ExpressionType: enum.Between, Value: []any{"", ""}, Type: enum.DateFieldType},
		{Field: "employee_id", ExpressionType: enum.Eq, Value: 0, Type: enum.BigIntFieldType},
	}
	payload.Bindings = []Binding{
		{Pointer: "/expressions/0/rules/0/value/0", Kind: BindingStartOfWeek},
		{Pointer: "/expressions/0/rules/0/value/1", Kind: BindingEndOfWeek},
		{Pointer: "/expressions/0/rules/1/value", Kind: BindingCurrentEmployee},
	}
	employeeID := 808
	resolver := NewBindingResolver(fixedClock{value: time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC)}, time.UTC)
	resolved, err := resolver.Resolve(context.Background(), payload, ScopeConfig{AllowedDynamicBindings: BindingKinds()}, Subject{UserID: 1, EmployeeID: &employeeID})
	if err != nil {
		t.Fatal(err)
	}
	week := resolved.Expressions[0].Rules[0].Value.([]any)
	if week[0] != "2026-12-28" || week[1] != "2027-01-03" || resolved.Expressions[0].Rules[1].Value != float64(employeeID) {
		t.Fatalf("week/employee binding mismatch: week=%v employee=%v", week, resolved.Expressions[0].Rules[1].Value)
	}
}

func TestIdentityBindingTargetsAControlledMultiValueElement(t *testing.T) {
	payload := testPayload()
	payload.Expressions[0].Rules[0] = request.QueryRule{
		Field: "owner_id", ExpressionType: enum.In, Value: []any{10, 20, 30}, Type: enum.BigIntFieldType,
	}
	payload.Order = request.Order{}
	payload.Bindings = []Binding{{Pointer: "/expressions/0/rules/0/value/2", Kind: BindingCurrentUser}}
	reader := metadataReaderStub{table: metadata.TableMetadata{Code: "orders", Fields: []metadata.FieldMetadata{{
		Code: "owner_id", StorageType: enum.BigIntFieldType, AdvancedQuery: true,
	}}}}
	config := ScopeConfig{TableCode: "orders", AllowedDynamicBindings: []BindingKind{BindingCurrentUser}}
	result, err := NewValidator(reader).ValidateMetadata(context.Background(), config, payload)
	if err != nil || result.Status != ValidationValid {
		t.Fatalf("multi-value binding validation: result=%+v err=%v", result, err)
	}
	resolved, err := NewBindingResolver(fixedClock{value: time.Now()}, time.UTC).
		Resolve(context.Background(), payload, config, Subject{UserID: 99})
	if err != nil {
		t.Fatal(err)
	}
	values := resolved.Expressions[0].Rules[0].Value.([]any)
	if values[2] != float64(99) {
		t.Fatalf("resolved values = %#v", values)
	}

	payload.Bindings[0].Pointer = "/expressions/0/rules/0/value"
	result, err = NewValidator(reader).ValidateMetadata(context.Background(), config, payload)
	if err != nil || result.Status != ValidationDegraded {
		t.Fatalf("whole multi-value replacement must be rejected: result=%+v err=%v", result, err)
	}
}

func TestPayloadJSONWhitelistAndSizeLimit(t *testing.T) {
	payload := testPayload()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"page_size", "pageSize", "filters", "table_code", "menu_id", "data_scope", "columns"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("payload leaked %s: %s", forbidden, raw)
		}
	}
	payload.QuickQuery.Keyword = strings.Repeat("x", MaxPayloadBytes)
	if result := ValidateSchema(payload); result.Status != ValidationInvalid {
		t.Fatalf("oversize payload status=%s", result.Status)
	}
	if _, err := DecodePayload([]byte(`{"expressions":[],"quick_query":{},"order":{},"bindings":[],"script":"alert(1)"}`)); err == nil {
		t.Fatal("strict decoder accepted an unknown payload field")
	}
}

func testPayload() QuerySchemePayloadV1 {
	return QuerySchemePayloadV1{
		Expressions: []request.ExpressionGroup{{
			Logic: enum.And,
			Rules: []request.QueryRule{{Field: "status", ExpressionType: enum.Eq, Value: "active", Type: enum.VarcharFieldType}},
		}},
		QuickQuery: request.QuickQuery{Keyword: "demo"},
		Order:      request.Order{Field: "status", IsAsc: true},
	}
}

func intPointer(value int) *int { return &value }

func stringPointer(value string) *string { return &value }
