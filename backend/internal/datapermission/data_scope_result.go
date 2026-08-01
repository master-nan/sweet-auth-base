package datapermission

import (
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	myerrors "backend/internal/errors"
)

const (
	DataScopeMaxConditionGroups    = 32
	DataScopeMaxConditionsPerGroup = 8
	DataScopeMaxConditions         = 256
	DataScopeMaxValuesPerCondition = 5000
	DataScopeMaxTotalParameters    = 5000
)

type DataScopeDecision string

const (
	DataScopeDecisionNotApplicable DataScopeDecision = "not_applicable"
	DataScopeDecisionAll           DataScopeDecision = "all"
	DataScopeDecisionNone          DataScopeDecision = "none"
	DataScopeDecisionFiltered      DataScopeDecision = "filtered"
)

type DataScopeOperator string

const (
	DataScopeOperatorEqual DataScopeOperator = "eq"
	DataScopeOperatorIn    DataScopeOperator = "in"
)

type DataScopeValueType string

const (
	DataScopeValueTypeBigint DataScopeValueType = "bigint"
	DataScopeValueTypeString DataScopeValueType = "string"
)

var (
	dataScopeResourceCodePattern  = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._:-][a-z0-9]+)*$`)
	dataScopeOwnershipCodePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	dataScopeOperations           = map[string]struct{}{
		"query":  {},
		"detail": {},
		"create": {},
		"update": {},
		"delete": {},
		"export": {},
		"run":    {},
	}
)

// DataScopeConditionInput is a typed construction boundary. Values are copied,
// normalized, and stored in a private bigint or string slice.
type DataScopeConditionInput struct {
	OwnershipCode string
	DimensionId   int
	Operator      DataScopeOperator
	ValueType     DataScopeValueType
	Values        []any
}

// DataScopeCondition contains only Resolver semantics. Database fields and
// executable expressions are intentionally absent.
type DataScopeCondition struct {
	ownershipCode string
	dimensionId   int
	operator      DataScopeOperator
	valueType     DataScopeValueType
	bigintValues  []int64
	stringValues  []string
}

// DataScopeConditionGroup is one AND branch. Multiple groups in a result are
// OR branches.
type DataScopeConditionGroup struct {
	conditions []DataScopeCondition
}

type DataScopeResultInput struct {
	ResourceCode    string
	Operation       string
	Decision        DataScopeDecision
	ConditionGroups []DataScopeConditionGroup
}

// DataScopeResult is the immutable Resolver output for one resource and
// operation. It never contains SQL, table names, field names, or ORM clauses.
type DataScopeResult struct {
	resourceCode    string
	operation       string
	decision        DataScopeDecision
	conditionGroups []DataScopeConditionGroup
}

func NewDataScopeCondition(input DataScopeConditionInput) (DataScopeCondition, error) {
	condition := DataScopeCondition{
		ownershipCode: strings.TrimSpace(input.OwnershipCode),
		dimensionId:   input.DimensionId,
		operator: DataScopeOperator(
			strings.ToLower(strings.TrimSpace(string(input.Operator))),
		),
		valueType: DataScopeValueType(
			strings.ToLower(strings.TrimSpace(string(input.ValueType))),
		),
	}
	if !dataScopeOwnershipCodePattern.MatchString(condition.ownershipCode) {
		return DataScopeCondition{}, myerrors.ErrDataScopeOwnershipCodeInvalid
	}
	if condition.dimensionId <= 0 {
		return DataScopeCondition{}, myerrors.ErrDataScopeDimensionInvalid
	}
	if condition.operator != DataScopeOperatorEqual && condition.operator != DataScopeOperatorIn {
		return DataScopeCondition{}, myerrors.ErrDataScopeOperatorInvalid
	}
	if condition.valueType != DataScopeValueTypeBigint &&
		condition.valueType != DataScopeValueTypeString {
		return DataScopeCondition{}, myerrors.ErrDataScopeValueTypeInvalid
	}
	if len(input.Values) == 0 {
		return DataScopeCondition{}, myerrors.ErrDataScopeFilterConditionMissing
	}
	if len(input.Values) > DataScopeMaxValuesPerCondition {
		return DataScopeCondition{}, myerrors.ErrDataScopeValueCountExceeded
	}

	var err error
	switch condition.valueType {
	case DataScopeValueTypeBigint:
		condition.bigintValues, err = normalizeBigintValues(input.Values)
	case DataScopeValueTypeString:
		condition.stringValues, err = normalizeStringValues(input.Values)
	}
	if err != nil {
		return DataScopeCondition{}, err
	}
	if condition.valueCount() == 0 {
		return DataScopeCondition{}, myerrors.ErrDataScopeFilterConditionMissing
	}
	if condition.operator == DataScopeOperatorEqual && condition.valueCount() != 1 {
		return DataScopeCondition{}, myerrors.ErrDataScopeResultConditionMismatch
	}
	return condition, nil
}

func NewDataScopeConditionGroup(
	conditions []DataScopeCondition,
) (DataScopeConditionGroup, error) {
	if len(conditions) == 0 {
		return DataScopeConditionGroup{}, myerrors.ErrDataScopeConditionGroupEmpty
	}

	normalized := make([]DataScopeCondition, 0, len(conditions))
	for _, condition := range conditions {
		if err := condition.Validate(); err != nil {
			return DataScopeConditionGroup{}, err
		}
		normalized = append(normalized, condition.clone())
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].canonicalKey() < normalized[j].canonicalKey()
	})
	// Only identical conditions are removed. Different value sets remain
	// separate AND conditions so this base model never guesses intersection or
	// union semantics that belong to the Resolver.
	normalized = deduplicateConditions(normalized)
	if len(normalized) > DataScopeMaxConditionsPerGroup {
		return DataScopeConditionGroup{}, myerrors.ErrDataScopeComplexityExceeded
	}
	return DataScopeConditionGroup{conditions: normalized}, nil
}

func NewDataScopeResult(input DataScopeResultInput) (DataScopeResult, error) {
	resourceCode := strings.TrimSpace(input.ResourceCode)
	operation := strings.ToLower(strings.TrimSpace(input.Operation))
	decision := DataScopeDecision(strings.ToLower(strings.TrimSpace(string(input.Decision))))
	if !dataScopeResourceCodePattern.MatchString(resourceCode) {
		return DataScopeResult{}, myerrors.ErrDataScopeResultIdentityInvalid
	}
	if _, supported := dataScopeOperations[operation]; !supported {
		return DataScopeResult{}, myerrors.ErrDataScopeResultIdentityInvalid
	}
	if !isDataScopeDecision(decision) {
		return DataScopeResult{}, myerrors.ErrDataScopeDecisionInvalid
	}
	if decision != DataScopeDecisionFiltered {
		if len(input.ConditionGroups) > 0 {
			return DataScopeResult{}, myerrors.ErrDataScopeResultConditionMismatch
		}
		return DataScopeResult{
			resourceCode: resourceCode,
			operation:    operation,
			decision:     decision,
		}, nil
	}
	if len(input.ConditionGroups) == 0 {
		return NewNoneResult(resourceCode, operation)
	}

	groups := make([]DataScopeConditionGroup, 0, len(input.ConditionGroups))
	for _, group := range input.ConditionGroups {
		normalized, err := NewDataScopeConditionGroup(group.Conditions())
		if err != nil {
			return DataScopeResult{}, err
		}
		groups = append(groups, normalized)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].canonicalKey() < groups[j].canonicalKey()
	})
	groups = deduplicateConditionGroups(groups)
	if err := validateDataScopeComplexity(groups); err != nil {
		return DataScopeResult{}, err
	}
	return DataScopeResult{
		resourceCode:    resourceCode,
		operation:       operation,
		decision:        decision,
		conditionGroups: groups,
	}, nil
}

func NewNotApplicableResult(resourceCode, operation string) (DataScopeResult, error) {
	return NewDataScopeResult(DataScopeResultInput{
		ResourceCode: resourceCode,
		Operation:    operation,
		Decision:     DataScopeDecisionNotApplicable,
	})
}

func NewAllResult(resourceCode, operation string) (DataScopeResult, error) {
	return NewDataScopeResult(DataScopeResultInput{
		ResourceCode: resourceCode,
		Operation:    operation,
		Decision:     DataScopeDecisionAll,
	})
}

func NewNoneResult(resourceCode, operation string) (DataScopeResult, error) {
	return NewDataScopeResult(DataScopeResultInput{
		ResourceCode: resourceCode,
		Operation:    operation,
		Decision:     DataScopeDecisionNone,
	})
}

func NewFilteredResult(
	resourceCode string,
	operation string,
	groups []DataScopeConditionGroup,
) (DataScopeResult, error) {
	return NewDataScopeResult(DataScopeResultInput{
		ResourceCode:    resourceCode,
		Operation:       operation,
		Decision:        DataScopeDecisionFiltered,
		ConditionGroups: groups,
	})
}

func OrDataScopeResults(left, right DataScopeResult) (DataScopeResult, error) {
	if err := validateMergeInputs(left, right); err != nil {
		return DataScopeResult{}, err
	}
	if left.decision == DataScopeDecisionNotApplicable ||
		right.decision == DataScopeDecisionNotApplicable {
		return DataScopeResult{}, myerrors.ErrDataScopeMergeUnsupported
	}
	if left.decision == DataScopeDecisionAll || right.decision == DataScopeDecisionAll {
		return NewAllResult(left.resourceCode, left.operation)
	}
	if left.decision == DataScopeDecisionNone {
		return right.clone()
	}
	if right.decision == DataScopeDecisionNone {
		return left.clone()
	}
	groups := append(left.ConditionGroups(), right.ConditionGroups()...)
	return NewFilteredResult(left.resourceCode, left.operation, groups)
}

func AndDataScopeResults(left, right DataScopeResult) (DataScopeResult, error) {
	if err := validateMergeInputs(left, right); err != nil {
		return DataScopeResult{}, err
	}
	if left.decision == DataScopeDecisionNotApplicable ||
		right.decision == DataScopeDecisionNotApplicable {
		return DataScopeResult{}, myerrors.ErrDataScopeMergeUnsupported
	}
	if left.decision == DataScopeDecisionNone || right.decision == DataScopeDecisionNone {
		return NewNoneResult(left.resourceCode, left.operation)
	}
	if left.decision == DataScopeDecisionAll {
		return right.clone()
	}
	if right.decision == DataScopeDecisionAll {
		return left.clone()
	}

	// V1 deliberately avoids Boolean distribution. A filtered AND merge is
	// supported only for one branch on each side; callers must leave multi-group
	// intersection to the top-level Resolver.
	if len(left.conditionGroups) != 1 || len(right.conditionGroups) != 1 {
		return DataScopeResult{}, myerrors.ErrDataScopeMergeUnsupported
	}
	conditions := append(
		left.conditionGroups[0].Conditions(),
		right.conditionGroups[0].Conditions()...,
	)
	group, err := NewDataScopeConditionGroup(conditions)
	if err != nil {
		return DataScopeResult{}, err
	}
	return NewFilteredResult(left.resourceCode, left.operation, []DataScopeConditionGroup{group})
}

func (condition DataScopeCondition) Validate() error {
	if !dataScopeOwnershipCodePattern.MatchString(condition.ownershipCode) {
		return myerrors.ErrDataScopeOwnershipCodeInvalid
	}
	if condition.dimensionId <= 0 {
		return myerrors.ErrDataScopeDimensionInvalid
	}
	if condition.operator != DataScopeOperatorEqual && condition.operator != DataScopeOperatorIn {
		return myerrors.ErrDataScopeOperatorInvalid
	}
	if condition.valueType != DataScopeValueTypeBigint &&
		condition.valueType != DataScopeValueTypeString {
		return myerrors.ErrDataScopeValueTypeInvalid
	}
	if condition.valueType == DataScopeValueTypeBigint && len(condition.stringValues) > 0 ||
		condition.valueType == DataScopeValueTypeString && len(condition.bigintValues) > 0 {
		return myerrors.ErrDataScopeValueTypeMismatch
	}
	if condition.valueCount() == 0 {
		return myerrors.ErrDataScopeFilterConditionMissing
	}
	if condition.valueCount() > DataScopeMaxValuesPerCondition {
		return myerrors.ErrDataScopeValueCountExceeded
	}
	if condition.operator == DataScopeOperatorEqual && condition.valueCount() != 1 {
		return myerrors.ErrDataScopeResultConditionMismatch
	}
	return nil
}

func (group DataScopeConditionGroup) Validate() error {
	if len(group.conditions) == 0 {
		return myerrors.ErrDataScopeConditionGroupEmpty
	}
	if len(group.conditions) > DataScopeMaxConditionsPerGroup {
		return myerrors.ErrDataScopeComplexityExceeded
	}
	for _, condition := range group.conditions {
		if err := condition.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (result DataScopeResult) Validate() error {
	if !dataScopeResourceCodePattern.MatchString(result.resourceCode) {
		return myerrors.ErrDataScopeResultIdentityInvalid
	}
	if _, supported := dataScopeOperations[result.operation]; !supported {
		return myerrors.ErrDataScopeResultIdentityInvalid
	}
	if !isDataScopeDecision(result.decision) {
		return myerrors.ErrDataScopeDecisionInvalid
	}
	if result.decision != DataScopeDecisionFiltered {
		if len(result.conditionGroups) > 0 {
			return myerrors.ErrDataScopeResultConditionMismatch
		}
		return nil
	}
	if len(result.conditionGroups) == 0 {
		return myerrors.ErrDataScopeFilterConditionMissing
	}
	for _, group := range result.conditionGroups {
		if err := group.Validate(); err != nil {
			return err
		}
	}
	return validateDataScopeComplexity(result.conditionGroups)
}

func (condition DataScopeCondition) OwnershipCode() string {
	return condition.ownershipCode
}

func (condition DataScopeCondition) DimensionId() int {
	return condition.dimensionId
}

func (condition DataScopeCondition) Operator() DataScopeOperator {
	return condition.operator
}

func (condition DataScopeCondition) ValueType() DataScopeValueType {
	return condition.valueType
}

func (condition DataScopeCondition) BigintValues() []int64 {
	return append([]int64(nil), condition.bigintValues...)
}

func (condition DataScopeCondition) StringValues() []string {
	return append([]string(nil), condition.stringValues...)
}

func (group DataScopeConditionGroup) Conditions() []DataScopeCondition {
	result := make([]DataScopeCondition, len(group.conditions))
	for index, condition := range group.conditions {
		result[index] = condition.clone()
	}
	return result
}

func (result DataScopeResult) ResourceCode() string {
	return result.resourceCode
}

func (result DataScopeResult) Operation() string {
	return result.operation
}

func (result DataScopeResult) Decision() DataScopeDecision {
	return result.decision
}

func (result DataScopeResult) ConditionGroups() []DataScopeConditionGroup {
	groups := make([]DataScopeConditionGroup, len(result.conditionGroups))
	for index, group := range result.conditionGroups {
		groups[index] = group.clone()
	}
	return groups
}

func (condition DataScopeCondition) MarshalJSON() ([]byte, error) {
	if err := condition.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(condition.jsonValue())
}

func (group DataScopeConditionGroup) MarshalJSON() ([]byte, error) {
	if err := group.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Conditions []DataScopeCondition `json:"all_of_conditions"`
	}{Conditions: group.Conditions()})
}

func (result DataScopeResult) MarshalJSON() ([]byte, error) {
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ResourceCode    string                    `json:"resource_code"`
		Operation       string                    `json:"operation"`
		Decision        DataScopeDecision         `json:"decision"`
		ConditionGroups []DataScopeConditionGroup `json:"condition_groups"`
	}{
		ResourceCode:    result.resourceCode,
		Operation:       result.operation,
		Decision:        result.decision,
		ConditionGroups: result.ConditionGroups(),
	})
}

func normalizeBigintValues(values []any) ([]int64, error) {
	normalized := make([]int64, 0, len(values))
	for _, value := range values {
		integer, ok := dataScopeInt64(value)
		if !ok || integer <= 0 {
			return nil, myerrors.ErrDataScopeValueTypeMismatch
		}
		normalized = append(normalized, integer)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return deduplicateInt64s(normalized), nil
}

func normalizeStringValues(values []any) ([]string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		stringValue, ok := value.(string)
		if !ok {
			return nil, myerrors.ErrDataScopeValueTypeMismatch
		}
		stringValue = strings.TrimSpace(stringValue)
		if stringValue == "" {
			return nil, myerrors.ErrDataScopeFilterConditionMissing
		}
		normalized = append(normalized, stringValue)
	}
	sort.Strings(normalized)
	return deduplicateStrings(normalized), nil
}

func dataScopeInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		if uint64(typed) > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case json.Number:
		integer, err := strconv.ParseInt(string(typed), 10, 64)
		return integer, err == nil
	default:
		return 0, false
	}
}

func validateDataScopeComplexity(groups []DataScopeConditionGroup) error {
	if len(groups) > DataScopeMaxConditionGroups {
		return myerrors.ErrDataScopeComplexityExceeded
	}
	conditionCount := 0
	parameterCount := 0
	for _, group := range groups {
		if err := group.Validate(); err != nil {
			return err
		}
		conditionCount += len(group.conditions)
		for _, condition := range group.conditions {
			parameterCount += condition.valueCount()
		}
	}
	if conditionCount > DataScopeMaxConditions || parameterCount > DataScopeMaxTotalParameters {
		return myerrors.ErrDataScopeComplexityExceeded
	}
	return nil
}

func validateMergeInputs(left, right DataScopeResult) error {
	if err := left.Validate(); err != nil {
		return err
	}
	if err := right.Validate(); err != nil {
		return err
	}
	if left.resourceCode != right.resourceCode || left.operation != right.operation {
		return myerrors.ErrDataScopeMergeUnsupported
	}
	return nil
}

func isDataScopeDecision(decision DataScopeDecision) bool {
	switch decision {
	case DataScopeDecisionNotApplicable,
		DataScopeDecisionAll,
		DataScopeDecisionNone,
		DataScopeDecisionFiltered:
		return true
	default:
		return false
	}
}

func (condition DataScopeCondition) valueCount() int {
	if condition.valueType == DataScopeValueTypeBigint {
		return len(condition.bigintValues)
	}
	return len(condition.stringValues)
}

func (condition DataScopeCondition) clone() DataScopeCondition {
	condition.bigintValues = condition.BigintValues()
	condition.stringValues = condition.StringValues()
	return condition
}

func (group DataScopeConditionGroup) clone() DataScopeConditionGroup {
	return DataScopeConditionGroup{conditions: group.Conditions()}
}

func (result DataScopeResult) clone() (DataScopeResult, error) {
	return NewDataScopeResult(DataScopeResultInput{
		ResourceCode:    result.resourceCode,
		Operation:       result.operation,
		Decision:        result.decision,
		ConditionGroups: result.ConditionGroups(),
	})
}

func (condition DataScopeCondition) jsonValue() any {
	if condition.valueType == DataScopeValueTypeBigint {
		return struct {
			OwnershipCode string             `json:"ownership_code"`
			DimensionId   int                `json:"dimension_id"`
			Operator      DataScopeOperator  `json:"operator"`
			ValueType     DataScopeValueType `json:"value_type"`
			Values        []int64            `json:"values"`
		}{
			OwnershipCode: condition.ownershipCode,
			DimensionId:   condition.dimensionId,
			Operator:      condition.operator,
			ValueType:     condition.valueType,
			Values:        condition.BigintValues(),
		}
	}
	return struct {
		OwnershipCode string             `json:"ownership_code"`
		DimensionId   int                `json:"dimension_id"`
		Operator      DataScopeOperator  `json:"operator"`
		ValueType     DataScopeValueType `json:"value_type"`
		Values        []string           `json:"values"`
	}{
		OwnershipCode: condition.ownershipCode,
		DimensionId:   condition.dimensionId,
		Operator:      condition.operator,
		ValueType:     condition.valueType,
		Values:        condition.StringValues(),
	}
}

func (condition DataScopeCondition) canonicalKey() string {
	values := any(condition.StringValues())
	if condition.valueType == DataScopeValueTypeBigint {
		values = condition.BigintValues()
	}
	encoded, _ := json.Marshal(struct {
		DimensionId   int
		OwnershipCode string
		Operator      DataScopeOperator
		ValueType     DataScopeValueType
		Values        any
	}{
		DimensionId:   condition.dimensionId,
		OwnershipCode: condition.ownershipCode,
		Operator:      condition.operator,
		ValueType:     condition.valueType,
		Values:        values,
	})
	return string(encoded)
}

func (group DataScopeConditionGroup) canonicalKey() string {
	keys := make([]string, len(group.conditions))
	for index, condition := range group.conditions {
		keys[index] = condition.canonicalKey()
	}
	encoded, _ := json.Marshal(keys)
	return string(encoded)
}

func deduplicateConditions(conditions []DataScopeCondition) []DataScopeCondition {
	if len(conditions) < 2 {
		return conditions
	}
	result := conditions[:0]
	lastKey := ""
	for index, condition := range conditions {
		key := condition.canonicalKey()
		if index > 0 && key == lastKey {
			continue
		}
		result = append(result, condition)
		lastKey = key
	}
	return result
}

func deduplicateConditionGroups(groups []DataScopeConditionGroup) []DataScopeConditionGroup {
	if len(groups) < 2 {
		return groups
	}
	result := groups[:0]
	lastKey := ""
	for index, group := range groups {
		key := group.canonicalKey()
		if index > 0 && key == lastKey {
			continue
		}
		result = append(result, group)
		lastKey = key
	}
	return result
}

func deduplicateInt64s(values []int64) []int64 {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func deduplicateStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
