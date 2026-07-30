package response

import (
	"encoding/json"
	"strings"
	"testing"

	"backend/model"

	"gorm.io/datatypes"
)

func TestDataPermissionConfigResponsesUseSafeWhitelist(t *testing.T) {
	internalDescription := "internal-only-description"
	tableId := 91
	tableFieldId := 92
	values := []any{
		NewDataDimensionDefinitionListRes(model.DataDimensionDefinition{
			Basic:        model.Basic{Id: 1, State: true},
			Code:         "org_unit",
			Name:         "组织",
			Category:     model.DataDimensionCategoryOrganization,
			ValueType:    model.DataDimensionValueTypeBigint,
			ProviderCode: "organization",
			Description:  internalDescription,
		}),
		NewDataResourceDetailRes(model.DataResource{
			Basic:             model.Basic{Id: 2, State: true},
			ResourceCode:      "order",
			Name:              "订单",
			ResourceType:      model.DataResourceTypeLowCodeTable,
			TableId:           &tableId,
			AdapterCode:       "metadata",
			PermissionEnabled: true,
			Description:       internalDescription,
		}),
		NewDataResourceOperationListRes(model.DataResourceOperation{
			Basic:             model.Basic{Id: 3, State: true},
			ResourceId:        2,
			Operation:         model.DataPermissionOperationQuery,
			PermissionEnabled: true,
			Description:       internalDescription,
		}),
		NewDataOwnershipFieldDetailRes(model.DataOwnershipField{
			Basic:         model.Basic{Id: 4, State: true},
			ResourceId:    2,
			OwnershipCode: "org_unit",
			DimensionId:   1,
			BindingType:   model.DataOwnershipBindingTypeMetadataField,
			TableFieldId:  &tableFieldId,
			ValueType:     model.DataDimensionValueTypeBigint,
			Description:   internalDescription,
		}),
		NewDataPolicyListRes(model.DataPolicy{
			Basic:       model.Basic{Id: 5, State: true},
			Code:        "own_org",
			Name:        "本组织",
			PolicyType:  model.DataPolicyTypeRuleSet,
			Description: internalDescription,
		}),
		NewDataPolicyRuleDetailRes(model.DataPolicyRule{
			Basic:           model.Basic{Id: 6, State: true},
			PolicyId:        5,
			Sequence:        1,
			DimensionId:     1,
			OwnershipCode:   "org_unit",
			ScopeSource:     model.DataPolicyScopeSourceEffectiveOrgUnits,
			Relation:        model.DataPolicyRelationExact,
			Operator:        model.DataPolicyOperatorIn,
			SpecifiedValues: datatypes.JSON([]byte(`["1"]`)),
			Description:     internalDescription,
		}),
		NewDataGrantListRes(model.DataGrant{
			Basic:       model.Basic{Id: 7, State: true},
			SubjectType: model.DataGrantSubjectTypeRole,
			SubjectId:   8,
			ResourceId:  2,
			Operation:   model.DataPermissionOperationQuery,
			PolicyId:    5,
			Description: internalDescription,
		}),
	}

	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %T: %v", value, err)
		}
		serialized := string(data)
		for _, forbidden := range []string{
			"description",
			"gmt_delete",
			"create_user",
			"create_name",
			"modify_user",
			"modify_name",
			"delete_user",
			"delete_name",
			"legacy",
			"old_data_scope",
			internalDescription,
		} {
			if strings.Contains(serialized, forbidden) {
				t.Fatalf("%T leaked %q: %s", value, forbidden, serialized)
			}
		}
	}
}

func TestDataPermissionConfigResponsesNormalizeInternalTargets(t *testing.T) {
	tableId := 100
	resource := NewDataResourceDetailRes(model.DataResource{
		Basic:        model.Basic{Id: 1, State: true},
		ResourceCode: "shipment",
		Name:         "运输单",
		ResourceType: model.DataResourceTypeLowCodeTable,
		TableId:      &tableId,
		AdapterCode:  "metadata",
	})
	resourceObject := marshalDataPermissionObject(t, resource)
	assertDataPermissionKeysAbsent(
		t,
		resourceObject,
		"table_id",
		"service_code",
		"report_definition_id",
	)
	target := resourceObject["target"].(map[string]any)
	if int(target["reference_id"].(float64)) != tableId ||
		target["type"] != model.DataResourceTypeLowCodeTable {
		t.Fatalf("unexpected normalized resource target: %+v", target)
	}

	fieldId := 200
	ownership := NewDataOwnershipFieldDetailRes(model.DataOwnershipField{
		Basic:         model.Basic{Id: 2, State: true},
		ResourceId:    1,
		OwnershipCode: "org_unit",
		DimensionId:   3,
		BindingType:   model.DataOwnershipBindingTypeMetadataField,
		TableFieldId:  &fieldId,
		ValueType:     model.DataDimensionValueTypeBigint,
	})
	ownershipObject := marshalDataPermissionObject(t, ownership)
	assertDataPermissionKeysAbsent(t, ownershipObject, "table_field_id", "adapter_field_code")
	bindingTarget := ownershipObject["binding_target"].(map[string]any)
	if int(bindingTarget["reference_id"].(float64)) != fieldId {
		t.Fatalf("unexpected normalized ownership target: %+v", bindingTarget)
	}
}

func marshalDataPermissionObject(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %T: %v", value, err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatalf("unmarshal %T: %v", value, err)
	}
	return object
}

func assertDataPermissionKeysAbsent(t *testing.T, object map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, exists := object[key]; exists {
			t.Fatalf("response unexpectedly contains %q: %+v", key, object)
		}
	}
}
