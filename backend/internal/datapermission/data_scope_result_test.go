package datapermission_test

import (
	"encoding/json"
	stderrors "errors"
	"reflect"
	"strings"
	"testing"

	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
)

const (
	testScopeResource  = "service:tms.transport_order"
	testScopeOperation = "query"
)

func TestDataScopeResultDecisionConstructors(t *testing.T) {
	tests := []struct {
		name     string
		create   func(string, string) (datapermission.DataScopeResult, error)
		decision datapermission.DataScopeDecision
	}{
		{
			name:     "not applicable",
			create:   datapermission.NewNotApplicableResult,
			decision: datapermission.DataScopeDecisionNotApplicable,
		},
		{
			name:     "all",
			create:   datapermission.NewAllResult,
			decision: datapermission.DataScopeDecisionAll,
		},
		{
			name:     "none",
			create:   datapermission.NewNoneResult,
			decision: datapermission.DataScopeDecisionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.create(testScopeResource, testScopeOperation)
			if err != nil {
				t.Fatalf("create result: %v", err)
			}
			if result.Decision() != tt.decision {
				t.Fatalf("decision = %q, want %q", result.Decision(), tt.decision)
			}
			if result.ResourceCode() != testScopeResource ||
				result.Operation() != testScopeOperation {
				t.Fatalf("identity = %s/%s", result.ResourceCode(), result.Operation())
			}
			if len(result.ConditionGroups()) != 0 {
				t.Fatalf("non-filtered decision contains groups: %+v", result.ConditionGroups())
			}
		})
	}
}

func TestFilteredDataScopeResultConstruction(t *testing.T) {
	condition := newBigintScopeCondition(
		t,
		"owner_org",
		11,
		datapermission.DataScopeOperatorIn,
		int64(2), int64(1), int64(2),
	)
	group := newScopeGroup(t, condition)
	result, err := datapermission.NewFilteredResult(
		testScopeResource,
		testScopeOperation,
		[]datapermission.DataScopeConditionGroup{group},
	)
	if err != nil {
		t.Fatalf("NewFilteredResult(): %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered {
		t.Fatalf("decision = %q, want filtered", result.Decision())
	}
	groups := result.ConditionGroups()
	if len(groups) != 1 || len(groups[0].Conditions()) != 1 {
		t.Fatalf("groups = %+v, want one group with one condition", groups)
	}
	if got := groups[0].Conditions()[0].BigintValues(); !reflect.DeepEqual(
		got,
		[]int64{1, 2},
	) {
		t.Fatalf("normalized values = %v, want [1 2]", got)
	}
}

func TestDataScopeResultRejectsDecisionConditionMismatch(t *testing.T) {
	group := newScopeGroup(t, newBigintScopeCondition(
		t,
		"owner_org",
		11,
		datapermission.DataScopeOperatorIn,
		int64(1),
	))
	_, err := datapermission.NewDataScopeResult(datapermission.DataScopeResultInput{
		ResourceCode:    testScopeResource,
		Operation:       testScopeOperation,
		Decision:        datapermission.DataScopeDecisionAll,
		ConditionGroups: []datapermission.DataScopeConditionGroup{group},
	})
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeResultConditionMismatch)
}

func TestEmptyFilteredResultNormalizesToNone(t *testing.T) {
	result, err := datapermission.NewFilteredResult(
		testScopeResource,
		testScopeOperation,
		nil,
	)
	if err != nil {
		t.Fatalf("NewFilteredResult(): %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionNone {
		t.Fatalf("empty filtered decision = %q, want none", result.Decision())
	}
	if result.Decision() == datapermission.DataScopeDecisionAll {
		t.Fatal("empty filtered result must never expand to all")
	}
}

func TestDataScopeConditionGroupRejectsEmpty(t *testing.T) {
	_, err := datapermission.NewDataScopeConditionGroup(nil)
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeConditionGroupEmpty)
}

func TestDataScopeConditionNormalizesBigintValues(t *testing.T) {
	condition := newBigintScopeCondition(
		t,
		"legal_entity",
		21,
		datapermission.DataScopeOperatorIn,
		int64(30), int(10), uint32(20), int64(10),
	)
	if got := condition.BigintValues(); !reflect.DeepEqual(got, []int64{10, 20, 30}) {
		t.Fatalf("BigintValues() = %v, want [10 20 30]", got)
	}
	if len(condition.StringValues()) != 0 {
		t.Fatalf("bigint condition contains strings: %v", condition.StringValues())
	}
}

func TestDataScopeConditionNormalizesStringValues(t *testing.T) {
	condition := newStringScopeCondition(
		t,
		"warehouse",
		31,
		datapermission.DataScopeOperatorIn,
		"WH-B", " WH-A ", "WH-B",
	)
	if got := condition.StringValues(); !reflect.DeepEqual(got, []string{"WH-A", "WH-B"}) {
		t.Fatalf("StringValues() = %v, want [WH-A WH-B]", got)
	}
	if len(condition.BigintValues()) != 0 {
		t.Fatalf("string condition contains bigints: %v", condition.BigintValues())
	}
}

func TestDataScopeConditionRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		valueType datapermission.DataScopeValueType
		operator  datapermission.DataScopeOperator
		values    []any
		errorCode int
	}{
		{
			name:      "mixed values",
			valueType: datapermission.DataScopeValueTypeBigint,
			operator:  datapermission.DataScopeOperatorIn,
			values:    []any{int64(1), "2"},
			errorCode: myerrors.ErrorCodeDataScopeValueTypeMismatch,
		},
		{
			name:      "null value",
			valueType: datapermission.DataScopeValueTypeString,
			operator:  datapermission.DataScopeOperatorIn,
			values:    []any{nil},
			errorCode: myerrors.ErrorCodeDataScopeValueTypeMismatch,
		},
		{
			name:      "empty string",
			valueType: datapermission.DataScopeValueTypeString,
			operator:  datapermission.DataScopeOperatorIn,
			values:    []any{"  "},
			errorCode: myerrors.ErrorCodeDataScopeFilterConditionMissing,
		},
		{
			name:      "no values",
			valueType: datapermission.DataScopeValueTypeBigint,
			operator:  datapermission.DataScopeOperatorIn,
			values:    nil,
			errorCode: myerrors.ErrorCodeDataScopeFilterConditionMissing,
		},
		{
			name:      "multiple eq values",
			valueType: datapermission.DataScopeValueTypeBigint,
			operator:  datapermission.DataScopeOperatorEqual,
			values:    []any{int64(1), int64(2)},
			errorCode: myerrors.ErrorCodeDataScopeResultConditionMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
				OwnershipCode: "owner_org",
				DimensionId:   11,
				Operator:      tt.operator,
				ValueType:     tt.valueType,
				Values:        tt.values,
			})
			assertDataScopeError(t, err, tt.errorCode)
		})
	}
}

func TestDataScopeConditionRejectsValueCountExceeded(t *testing.T) {
	values := make([]any, datapermission.DataScopeMaxValuesPerCondition+1)
	for index := range values {
		values[index] = int64(index + 1)
	}
	_, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: "owner_org",
		DimensionId:   11,
		Operator:      datapermission.DataScopeOperatorIn,
		ValueType:     datapermission.DataScopeValueTypeBigint,
		Values:        values,
	})
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeValueCountExceeded)
}

func TestDataScopeConditionsAndGroupsUseStableOrder(t *testing.T) {
	first := newBigintScopeCondition(
		t,
		"legal_entity",
		10,
		datapermission.DataScopeOperatorIn,
		int64(1),
	)
	second := newBigintScopeCondition(
		t,
		"owner_org",
		20,
		datapermission.DataScopeOperatorIn,
		int64(2),
	)
	group, err := datapermission.NewDataScopeConditionGroup(
		[]datapermission.DataScopeCondition{second, first, first},
	)
	if err != nil {
		t.Fatalf("NewDataScopeConditionGroup(): %v", err)
	}
	conditions := group.Conditions()
	if len(conditions) != 2 {
		t.Fatalf("condition count = %d, want 2", len(conditions))
	}
	if conditions[0].DimensionId() != 10 || conditions[1].DimensionId() != 20 {
		t.Fatalf("condition order = [%d %d], want [10 20]",
			conditions[0].DimensionId(), conditions[1].DimensionId())
	}

	groupOne := newScopeGroup(t, first)
	groupTwo := newScopeGroup(t, second)
	result, err := datapermission.NewFilteredResult(
		testScopeResource,
		testScopeOperation,
		[]datapermission.DataScopeConditionGroup{groupTwo, groupOne, groupOne},
	)
	if err != nil {
		t.Fatalf("NewFilteredResult(): %v", err)
	}
	groups := result.ConditionGroups()
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	if groups[0].Conditions()[0].DimensionId() != 10 ||
		groups[1].Conditions()[0].DimensionId() != 20 {
		t.Fatalf("group order is not stable: %+v", groups)
	}
}

func TestDataScopeResultDoesNotShareMutableSlices(t *testing.T) {
	values := []any{int64(2), int64(1)}
	condition, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: "owner_org",
		DimensionId:   11,
		Operator:      datapermission.DataScopeOperatorIn,
		ValueType:     datapermission.DataScopeValueTypeBigint,
		Values:        values,
	})
	if err != nil {
		t.Fatalf("NewDataScopeCondition(): %v", err)
	}
	values[0] = int64(999)
	returnedValues := condition.BigintValues()
	returnedValues[0] = 888
	if !reflect.DeepEqual(condition.BigintValues(), []int64{1, 2}) {
		t.Fatalf("condition values were mutated: %v", condition.BigintValues())
	}

	conditions := []datapermission.DataScopeCondition{condition}
	group := newScopeGroup(t, conditions...)
	conditions[0] = datapermission.DataScopeCondition{}
	returnedConditions := group.Conditions()
	returnedConditions[0] = datapermission.DataScopeCondition{}
	if len(group.Conditions()) != 1 || group.Conditions()[0].OwnershipCode() != "owner_org" {
		t.Fatalf("group conditions were mutated: %+v", group.Conditions())
	}

	groups := []datapermission.DataScopeConditionGroup{group}
	result, err := datapermission.NewFilteredResult(
		testScopeResource,
		testScopeOperation,
		groups,
	)
	if err != nil {
		t.Fatalf("NewFilteredResult(): %v", err)
	}
	groups[0] = datapermission.DataScopeConditionGroup{}
	returnedGroups := result.ConditionGroups()
	returnedGroups[0] = datapermission.DataScopeConditionGroup{}
	if len(result.ConditionGroups()) != 1 ||
		result.ConditionGroups()[0].Conditions()[0].OwnershipCode() != "owner_org" {
		t.Fatalf("result groups were mutated: %+v", result.ConditionGroups())
	}
}

func TestDataScopeResultJSONUsesStrictWhitelist(t *testing.T) {
	condition := newStringScopeCondition(
		t,
		"warehouse",
		31,
		datapermission.DataScopeOperatorIn,
		"WH-A", "WH-B",
	)
	result := newFilteredScopeResult(t, newScopeGroup(t, condition))
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(): %v", err)
	}

	var object map[string]any
	if err = json.Unmarshal(payload, &object); err != nil {
		t.Fatalf("json.Unmarshal(): %v", err)
	}
	assertExactMapKeys(t, object, []string{
		"resource_code", "operation", "decision", "condition_groups",
	})
	groups := object["condition_groups"].([]any)
	group := groups[0].(map[string]any)
	assertExactMapKeys(t, group, []string{"all_of_conditions"})
	conditionObject := group["all_of_conditions"].([]any)[0].(map[string]any)
	assertExactMapKeys(t, conditionObject, []string{
		"ownership_code", "dimension_id", "operator", "value_type", "values",
	})

	serialized := string(payload)
	for _, forbidden := range []string{
		"sql", "table", "field", "join", "gorm", "policy_id", "grant_id", "role_name",
	} {
		if strings.Contains(strings.ToLower(serialized), forbidden) {
			t.Fatalf("serialized result leaked forbidden token %q: %s", forbidden, serialized)
		}
	}
}

func TestOrDataScopeResults(t *testing.T) {
	filteredOne := newFilteredScopeResult(t, newScopeGroup(t, newBigintScopeCondition(
		t,
		"owner_org",
		11,
		datapermission.DataScopeOperatorIn,
		int64(1), int64(2),
	)))
	filteredTwo := newFilteredScopeResult(t, newScopeGroup(t, newBigintScopeCondition(
		t,
		"legal_entity",
		21,
		datapermission.DataScopeOperatorIn,
		int64(10),
	)))
	none := newSimpleScopeResult(t, datapermission.NewNoneResult)
	all := newSimpleScopeResult(t, datapermission.NewAllResult)

	t.Run("none OR filtered", func(t *testing.T) {
		result, err := datapermission.OrDataScopeResults(none, filteredOne)
		if err != nil || result.Decision() != datapermission.DataScopeDecisionFiltered {
			t.Fatalf("result = %+v, error = %v", result, err)
		}
	})
	t.Run("all OR filtered", func(t *testing.T) {
		result, err := datapermission.OrDataScopeResults(all, filteredOne)
		if err != nil || result.Decision() != datapermission.DataScopeDecisionAll {
			t.Fatalf("result = %+v, error = %v", result, err)
		}
	})
	t.Run("filtered OR filtered", func(t *testing.T) {
		result, err := datapermission.OrDataScopeResults(filteredTwo, filteredOne)
		if err != nil {
			t.Fatalf("OrDataScopeResults(): %v", err)
		}
		if len(result.ConditionGroups()) != 2 {
			t.Fatalf("group count = %d, want 2", len(result.ConditionGroups()))
		}
	})
	t.Run("identical filtered branches deduplicate", func(t *testing.T) {
		result, err := datapermission.OrDataScopeResults(filteredOne, filteredOne)
		if err != nil {
			t.Fatalf("OrDataScopeResults(): %v", err)
		}
		if len(result.ConditionGroups()) != 1 {
			t.Fatalf("group count = %d, want 1", len(result.ConditionGroups()))
		}
	})
}

func TestAndDataScopeResults(t *testing.T) {
	orgCondition := newBigintScopeCondition(
		t,
		"owner_org",
		11,
		datapermission.DataScopeOperatorIn,
		int64(1), int64(2),
	)
	legalEntityCondition := newBigintScopeCondition(
		t,
		"legal_entity",
		21,
		datapermission.DataScopeOperatorEqual,
		int64(10),
	)
	filteredOrg := newFilteredScopeResult(t, newScopeGroup(t, orgCondition))
	filteredLegalEntity := newFilteredScopeResult(t, newScopeGroup(t, legalEntityCondition))
	all := newSimpleScopeResult(t, datapermission.NewAllResult)
	none := newSimpleScopeResult(t, datapermission.NewNoneResult)

	t.Run("all AND filtered", func(t *testing.T) {
		result, err := datapermission.AndDataScopeResults(all, filteredOrg)
		if err != nil || result.Decision() != datapermission.DataScopeDecisionFiltered {
			t.Fatalf("result = %+v, error = %v", result, err)
		}
	})
	t.Run("none AND filtered", func(t *testing.T) {
		result, err := datapermission.AndDataScopeResults(none, filteredOrg)
		if err != nil || result.Decision() != datapermission.DataScopeDecisionNone {
			t.Fatalf("result = %+v, error = %v", result, err)
		}
	})
	t.Run("filtered AND filtered", func(t *testing.T) {
		result, err := datapermission.AndDataScopeResults(filteredOrg, filteredLegalEntity)
		if err != nil {
			t.Fatalf("AndDataScopeResults(): %v", err)
		}
		groups := result.ConditionGroups()
		if len(groups) != 1 || len(groups[0].Conditions()) != 2 {
			t.Fatalf("merged groups = %+v, want one group with two conditions", groups)
		}
	})
	t.Run("identical filtered conditions deduplicate", func(t *testing.T) {
		result, err := datapermission.AndDataScopeResults(filteredOrg, filteredOrg)
		if err != nil {
			t.Fatalf("AndDataScopeResults(): %v", err)
		}
		if len(result.ConditionGroups()[0].Conditions()) != 1 {
			t.Fatalf("condition count = %d, want 1", len(result.ConditionGroups()[0].Conditions()))
		}
	})
}

func TestDataScopeResultMergeRejectsNotApplicable(t *testing.T) {
	notApplicable := newSimpleScopeResult(t, datapermission.NewNotApplicableResult)
	filtered := newFilteredScopeResult(t, newScopeGroup(t, newBigintScopeCondition(
		t,
		"owner_org",
		11,
		datapermission.DataScopeOperatorIn,
		int64(1),
	)))

	_, err := datapermission.OrDataScopeResults(notApplicable, filtered)
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeMergeUnsupported)
	_, err = datapermission.AndDataScopeResults(notApplicable, filtered)
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeMergeUnsupported)
}

func TestDataScopeResultAndRejectsBooleanDistribution(t *testing.T) {
	first := newScopeGroup(t, newBigintScopeCondition(
		t,
		"owner_org",
		11,
		datapermission.DataScopeOperatorIn,
		int64(1),
	))
	second := newScopeGroup(t, newBigintScopeCondition(
		t,
		"legal_entity",
		21,
		datapermission.DataScopeOperatorIn,
		int64(10),
	))
	multipleGroups := newFilteredScopeResult(t, first, second)
	singleGroup := newFilteredScopeResult(t, first)

	_, err := datapermission.AndDataScopeResults(multipleGroups, singleGroup)
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeMergeUnsupported)
}

func TestDataScopeResultRejectsTotalParameterOverflow(t *testing.T) {
	firstValues := make([]any, 3000)
	secondValues := make([]any, 3000)
	for index := 0; index < 3000; index++ {
		firstValues[index] = int64(index + 1)
		secondValues[index] = int64(index + 3001)
	}
	first, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: "owner_org",
		DimensionId:   11,
		Operator:      datapermission.DataScopeOperatorIn,
		ValueType:     datapermission.DataScopeValueTypeBigint,
		Values:        firstValues,
	})
	if err != nil {
		t.Fatalf("first condition: %v", err)
	}
	second, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: "legal_entity",
		DimensionId:   21,
		Operator:      datapermission.DataScopeOperatorIn,
		ValueType:     datapermission.DataScopeValueTypeBigint,
		Values:        secondValues,
	})
	if err != nil {
		t.Fatalf("second condition: %v", err)
	}
	_, err = datapermission.NewFilteredResult(
		testScopeResource,
		testScopeOperation,
		[]datapermission.DataScopeConditionGroup{
			newScopeGroup(t, first),
			newScopeGroup(t, second),
		},
	)
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeComplexityExceeded)
}

func TestDataScopeResultStableValidationErrors(t *testing.T) {
	validValues := []any{int64(1)}
	tests := []struct {
		name      string
		input     datapermission.DataScopeConditionInput
		errorCode int
	}{
		{
			name: "ownership invalid",
			input: datapermission.DataScopeConditionInput{
				OwnershipCode: "Owner Org",
				DimensionId:   11,
				Operator:      datapermission.DataScopeOperatorIn,
				ValueType:     datapermission.DataScopeValueTypeBigint,
				Values:        validValues,
			},
			errorCode: myerrors.ErrorCodeDataScopeOwnershipCodeInvalid,
		},
		{
			name: "dimension invalid",
			input: datapermission.DataScopeConditionInput{
				OwnershipCode: "owner_org",
				DimensionId:   0,
				Operator:      datapermission.DataScopeOperatorIn,
				ValueType:     datapermission.DataScopeValueTypeBigint,
				Values:        validValues,
			},
			errorCode: myerrors.ErrorCodeDataScopeDimensionInvalid,
		},
		{
			name: "operator invalid",
			input: datapermission.DataScopeConditionInput{
				OwnershipCode: "owner_org",
				DimensionId:   11,
				Operator:      "contains",
				ValueType:     datapermission.DataScopeValueTypeBigint,
				Values:        validValues,
			},
			errorCode: myerrors.ErrorCodeDataScopeOperatorInvalid,
		},
		{
			name: "value type invalid",
			input: datapermission.DataScopeConditionInput{
				OwnershipCode: "owner_org",
				DimensionId:   11,
				Operator:      datapermission.DataScopeOperatorIn,
				ValueType:     "number",
				Values:        validValues,
			},
			errorCode: myerrors.ErrorCodeDataScopeValueTypeInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := datapermission.NewDataScopeCondition(tt.input)
			assertDataScopeError(t, err, tt.errorCode)
		})
	}

	_, err := datapermission.NewDataScopeResult(datapermission.DataScopeResultInput{
		ResourceCode: testScopeResource,
		Operation:    testScopeOperation,
		Decision:     "unknown",
	})
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeDecisionInvalid)

	_, err = datapermission.NewAllResult("invalid resource", testScopeOperation)
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeResultIdentityInvalid)
}

func TestZeroDataScopeResultCannotBeSerialized(t *testing.T) {
	_, err := json.Marshal(datapermission.DataScopeResult{})
	assertDataScopeError(t, err, myerrors.ErrorCodeDataScopeResultIdentityInvalid)
}

func newBigintScopeCondition(
	t *testing.T,
	ownershipCode string,
	dimensionId int,
	operator datapermission.DataScopeOperator,
	values ...any,
) datapermission.DataScopeCondition {
	t.Helper()
	condition, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: ownershipCode,
		DimensionId:   dimensionId,
		Operator:      operator,
		ValueType:     datapermission.DataScopeValueTypeBigint,
		Values:        values,
	})
	if err != nil {
		t.Fatalf("NewDataScopeCondition(): %v", err)
	}
	return condition
}

func newStringScopeCondition(
	t *testing.T,
	ownershipCode string,
	dimensionId int,
	operator datapermission.DataScopeOperator,
	values ...string,
) datapermission.DataScopeCondition {
	t.Helper()
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	condition, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: ownershipCode,
		DimensionId:   dimensionId,
		Operator:      operator,
		ValueType:     datapermission.DataScopeValueTypeString,
		Values:        items,
	})
	if err != nil {
		t.Fatalf("NewDataScopeCondition(): %v", err)
	}
	return condition
}

func newScopeGroup(
	t *testing.T,
	conditions ...datapermission.DataScopeCondition,
) datapermission.DataScopeConditionGroup {
	t.Helper()
	group, err := datapermission.NewDataScopeConditionGroup(conditions)
	if err != nil {
		t.Fatalf("NewDataScopeConditionGroup(): %v", err)
	}
	return group
}

func newFilteredScopeResult(
	t *testing.T,
	groups ...datapermission.DataScopeConditionGroup,
) datapermission.DataScopeResult {
	t.Helper()
	result, err := datapermission.NewFilteredResult(
		testScopeResource,
		testScopeOperation,
		groups,
	)
	if err != nil {
		t.Fatalf("NewFilteredResult(): %v", err)
	}
	return result
}

func newSimpleScopeResult(
	t *testing.T,
	create func(string, string) (datapermission.DataScopeResult, error),
) datapermission.DataScopeResult {
	t.Helper()
	result, err := create(testScopeResource, testScopeOperation)
	if err != nil {
		t.Fatalf("create scope result: %v", err)
	}
	return result
}

func assertDataScopeError(t *testing.T, err error, errorCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d", errorCode)
	}
	var adminError *myerrors.ApplicationError
	if !stderrors.As(err, &adminError) {
		t.Fatalf("error = %T, want *myerrors.ApplicationError", err)
	}
	if adminError.Code != errorCode {
		t.Fatalf("error code = %d, want %d", adminError.Code, errorCode)
	}
}

func assertExactMapKeys(t *testing.T, object map[string]any, expected []string) {
	t.Helper()
	if len(object) != len(expected) {
		t.Fatalf("keys = %v, want %v", mapKeys(object), expected)
	}
	for _, key := range expected {
		if _, exists := object[key]; !exists {
			t.Fatalf("missing key %q in %v", key, mapKeys(object))
		}
	}
}

func mapKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	return keys
}
