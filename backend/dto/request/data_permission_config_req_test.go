package request

import (
	"encoding/json"
	"reflect"
	"testing"

	"backend/internal/utils"

	"github.com/go-playground/validator/v10"
)

func TestDataPermissionConfigQueryDTOValidation(t *testing.T) {
	validate := validator.New()
	if _, err := utils.InitializeValidator(validate); err != nil {
		t.Fatalf("initialize validator: %v", err)
	}

	valid := []any{
		DataPermissionConfigIdReq{Id: 1},
		DataDimensionDefinitionQueryReq{Category: "organization", ValueType: "bigint"},
		DataResourceQueryReq{ResourceType: "low_code_table"},
		DataResourceOperationQueryReq{Operation: "query"},
		DataOwnershipFieldQueryReq{BindingType: "metadata_field", ValueType: "bigint"},
		DataPolicyQueryReq{PolicyType: "rule_set"},
		DataPolicyRuleQueryReq{
			ScopeSource: "effective_org_units",
			Relation:    "self_and_descendants",
			Operator:    "in",
		},
		DataGrantQueryReq{SubjectType: "role", Operation: "detail"},
	}
	for _, value := range valid {
		if err := validate.Struct(value); err != nil {
			t.Fatalf("expected valid %T: %v", value, err)
		}
	}

	invalid := []any{
		DataPermissionConfigIdReq{},
		DataPermissionConfigQueryReq{Page: -1},
		DataPermissionConfigQueryReq{Num: 501},
		DataDimensionDefinitionQueryReq{Category: "sql"},
		DataResourceQueryReq{ResourceType: "database"},
		DataResourceOperationQueryReq{Operation: "approve"},
		DataOwnershipFieldQueryReq{BindingType: "report_source"},
		DataPolicyQueryReq{PolicyType: "deny"},
		DataPolicyRuleQueryReq{ScopeSource: "client_provider"},
		DataGrantQueryReq{SubjectType: "position"},
	}
	for _, value := range invalid {
		if err := validate.Struct(value); err == nil {
			t.Fatalf("expected invalid %T to fail: %+v", value, value)
		}
	}
}

func TestDataPermissionConfigQueryDTOExcludesUnsafeClientFields(t *testing.T) {
	queryType := reflect.TypeOf(DataPermissionConfigQueryReq{})
	for _, forbidden := range []string{
		"table_code",
		"table_name",
		"sql",
		"join",
		"filters",
		"include_deleted",
		"menu_id",
		"data_scope",
		"provider",
	} {
		if hasJSONField(queryType, forbidden) {
			t.Fatalf("query DTO exposes forbidden client field %q", forbidden)
		}
	}

	var req DataResourceQueryReq
	if err := json.Unmarshal([]byte(`{
		"page": 2,
		"num": 20,
		"table_code": "sys_user",
		"table_name": "sys_user",
		"sql": "select * from sys_user",
		"join": "sys_role",
		"filters": {"state": true},
		"provider": "client_provider"
	}`), &req); err != nil {
		t.Fatalf("unmarshal restricted payload: %v", err)
	}
	basic := req.ToBasic()
	if basic.Page != 2 || basic.Num != 20 {
		t.Fatalf("safe paging fields were not retained: %+v", basic)
	}
	if basic.TableCode != "" ||
		basic.Filters != nil ||
		basic.IncludeDeleted ||
		basic.MenuId != 0 ||
		basic.DataScope != nil {
		t.Fatalf("restricted fields reached Basic request: %+v", basic)
	}
}

func TestDataPermissionConfigFieldBoundaries(t *testing.T) {
	tests := []struct {
		object       string
		immutable    []string
		mutable      []string
		neverExposed []string
	}{
		{
			object:       DataPermissionConfigObjectDimension,
			immutable:    []string{"dimension_code"},
			mutable:      []string{"name", "state"},
			neverExposed: []string{"sql", "table_name", "organization_id"},
		},
		{
			object:       DataPermissionConfigObjectResource,
			immutable:    []string{"resource_code"},
			mutable:      []string{"name", "permission_enabled"},
			neverExposed: []string{"menu_id", "sql", "join"},
		},
		{
			object:       DataPermissionConfigObjectOperation,
			immutable:    []string{"resource_id", "operation"},
			mutable:      []string{"permission_enabled"},
			neverExposed: []string{"sql"},
		},
		{
			object:       DataPermissionConfigObjectOwnership,
			immutable:    []string{"ownership_code", "dimension_id"},
			mutable:      []string{"state"},
			neverExposed: []string{"report_source", "sql"},
		},
		{
			object:       DataPermissionConfigObjectPolicy,
			immutable:    []string{"policy_code"},
			mutable:      []string{"name", "state"},
			neverExposed: []string{"sql"},
		},
		{
			object:       DataPermissionConfigObjectRule,
			immutable:    []string{"dimension_id", "ownership_code"},
			mutable:      []string{"scope_source", "state"},
			neverExposed: []string{"sql", "expression"},
		},
		{
			object:       DataPermissionConfigObjectGrant,
			immutable:    []string{"subject_type", "subject_id"},
			mutable:      []string{"valid_from", "valid_to", "state"},
			neverExposed: []string{"role_name", "provider"},
		},
	}

	for _, test := range tests {
		t.Run(test.object, func(t *testing.T) {
			boundary, ok := GetDataPermissionFieldBoundary(test.object)
			if !ok {
				t.Fatalf("missing field boundary for %s", test.object)
			}
			for _, field := range test.immutable {
				assertStringInSlice(t, boundary.ImmutableAfterUse, field)
				assertStringNotInSlice(t, boundary.Update, field)
			}
			for _, field := range test.mutable {
				assertStringInSlice(t, boundary.Update, field)
			}
			for _, field := range test.neverExposed {
				assertStringNotInSlice(t, boundary.Create, field)
				assertStringNotInSlice(t, boundary.Update, field)
			}

			boundary.Update[0] = "mutated_by_caller"
			fresh, _ := GetDataPermissionFieldBoundary(test.object)
			assertStringNotInSlice(t, fresh.Update, "mutated_by_caller")
		})
	}

	if _, ok := GetDataPermissionFieldBoundary("unknown"); ok {
		t.Fatal("unknown object unexpectedly returned a field boundary")
	}
}

func hasJSONField(value reflect.Type, target string) bool {
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		if field.Anonymous {
			if hasJSONField(field.Type, target) {
				return true
			}
			continue
		}
		tag := field.Tag.Get("json")
		if tag == target || len(tag) > len(target) && tag[:len(target)+1] == target+"," {
			return true
		}
	}
	return false
}

func assertStringInSlice(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, value := range values {
		if value == expected {
			return
		}
	}
	t.Fatalf("%q is missing from %v", expected, values)
}

func assertStringNotInSlice(t *testing.T, values []string, forbidden string) {
	t.Helper()
	for _, value := range values {
		if value == forbidden {
			t.Fatalf("%q unexpectedly exists in %v", forbidden, values)
		}
	}
}
