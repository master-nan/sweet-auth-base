package datapermission

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"backend/dto/response"
	myerrors "backend/internal/errors"
)

// RegisteredFieldExecutionRequest contains only immutable, structured
// permission semantics. A business module can inspect it without receiving a
// table name, column name, SQL fragment, or ORM clause.
type RegisteredFieldExecutionRequest struct {
	resource  AdapterResourceContext
	condition AdapterCondition
}

func (request RegisteredFieldExecutionRequest) ResourceContext() AdapterResourceContext {
	return request.resource
}

func (request RegisteredFieldExecutionRequest) Condition() AdapterCondition {
	return request.condition.clone()
}

// RegisteredFieldExecutor is implemented by reviewed server code owned by a
// fixed business module. Prepare validates that the module can consume the
// structured condition; the returned AdapterExecution remains the safe,
// ORM-independent description consumed by a later repository integration.
type RegisteredFieldExecutor interface {
	Prepare(context.Context, RegisteredFieldExecutionRequest) error
}

type RegisteredFieldExecutorFunc func(context.Context, RegisteredFieldExecutionRequest) error

func (executor RegisteredFieldExecutorFunc) Prepare(
	ctx context.Context,
	request RegisteredFieldExecutionRequest,
) error {
	if executor == nil {
		return myerrors.ErrRegisteredAdapterExecutionInvalid
	}
	return executor(ctx, request)
}

// RegisteredFieldExecutionRegistration is a server-owned declaration. It is
// constructed by application modules, never from an administrative request.
type RegisteredFieldExecutionRegistration struct {
	ResourceCode        string
	AdapterCode         string
	AdapterFieldCode    string
	DimensionIds        []int
	ValueType           DataScopeValueType
	SupportedOperations []string
	SupportedOperators  []DataScopeOperator
	Executor            RegisteredFieldExecutor
}

type registeredFieldExecutionEntry struct {
	registration RegisteredFieldExecutionRegistration
	dimensions   map[int]struct{}
	operations   map[string]struct{}
	operators    map[DataScopeOperator]struct{}
}

// RegisteredFieldExecutionRegistry is process-local and safe for concurrent
// reads. Each application instance builds its own registry during startup;
// this type is not exposed through a management API and has no global state.
type RegisteredFieldExecutionRegistry struct {
	mu           sync.RWMutex
	entries      map[string]registeredFieldExecutionEntry
	adapterCodes map[string]struct{}
}

func NewRegisteredFieldExecutionRegistry(
	registrations ...RegisteredFieldExecutionRegistration,
) (*RegisteredFieldExecutionRegistry, error) {
	registry := &RegisteredFieldExecutionRegistry{
		entries:      make(map[string]registeredFieldExecutionEntry, len(registrations)),
		adapterCodes: make(map[string]struct{}),
	}
	for _, registration := range registrations {
		if err := registry.Register(registration); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// Register is intended for application construction and isolated tests. No
// API or persisted configuration is allowed to call this method.
func (registry *RegisteredFieldExecutionRegistry) Register(
	registration RegisteredFieldExecutionRegistration,
) error {
	if registry == nil {
		return myerrors.ErrRegisteredAdapterExecutionInvalid
	}
	normalized, err := normalizeRegisteredFieldExecutionRegistration(registration)
	if err != nil {
		return err
	}
	entry := registeredFieldExecutionEntry{
		registration: normalized,
		dimensions:   intSet(normalized.DimensionIds),
		operations:   stringSet(normalized.SupportedOperations),
		operators:    operatorSet(normalized.SupportedOperators),
	}
	key := registeredFieldExecutionKey(normalized.AdapterCode, normalized.AdapterFieldCode)

	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.entries == nil {
		registry.entries = make(map[string]registeredFieldExecutionEntry)
	}
	if registry.adapterCodes == nil {
		registry.adapterCodes = make(map[string]struct{})
	}
	if existing, exists := registry.entries[key]; exists {
		if registeredFieldDescriptorsEqual(existing.registration, normalized) {
			return myerrors.ErrRegisteredAdapterRegistrationDuplicate
		}
		return myerrors.ErrRegisteredAdapterRegistrationConflict
	}
	registry.entries[key] = entry
	registry.adapterCodes[normalized.AdapterCode] = struct{}{}
	return nil
}

func (registry *RegisteredFieldExecutionRegistry) resolve(
	adapterCode string,
	adapterFieldCode string,
) (registeredFieldExecutionEntry, error) {
	if registry == nil {
		return registeredFieldExecutionEntry{}, myerrors.ErrRegisteredAdapterUnregistered
	}
	adapterCode = strings.TrimSpace(adapterCode)
	adapterFieldCode = strings.TrimSpace(adapterFieldCode)

	registry.mu.RLock()
	defer registry.mu.RUnlock()
	entry, exists := registry.entries[registeredFieldExecutionKey(adapterCode, adapterFieldCode)]
	if exists {
		return entry.clone(), nil
	}
	if _, exists = registry.adapterCodes[adapterCode]; !exists {
		return registeredFieldExecutionEntry{}, myerrors.ErrRegisteredAdapterUnregistered
	}
	return registeredFieldExecutionEntry{}, myerrors.ErrRegisteredAdapterFieldNotFound
}

type RegisteredFieldAdapter struct {
	registry *RegisteredFieldExecutionRegistry
}

var _ Adapter = (*RegisteredFieldAdapter)(nil)

func NewRegisteredFieldAdapter(
	registry *RegisteredFieldExecutionRegistry,
) (*RegisteredFieldAdapter, error) {
	if registry == nil {
		return nil, myerrors.ErrDataPermissionAdapterInputInvalid
	}
	return &RegisteredFieldAdapter{registry: registry}, nil
}

func (adapter *RegisteredFieldAdapter) Apply(
	ctx context.Context,
	input AdapterInput,
) (AdapterExecution, error) {
	if adapter == nil || adapter.registry == nil {
		return AdapterExecution{}, myerrors.ErrRegisteredAdapterUnregistered
	}
	if err := input.Validate(); err != nil {
		return AdapterExecution{}, err
	}
	if err := validateRegisteredOwnershipDefinitions(input); err != nil {
		return AdapterExecution{}, err
	}

	execution, err := BuildAdapterExecution(input)
	if err != nil {
		return AdapterExecution{}, mapRegisteredAdapterContractError(err)
	}
	if execution.Mode() != AdapterExecutionModeApplyFilter {
		return execution.clone(), nil
	}
	if err = validateRegisteredExecutionComplexity(execution); err != nil {
		return AdapterExecution{}, err
	}

	resource := input.ResourceContext()
	for _, group := range execution.ConditionGroups() {
		for _, condition := range group.Conditions() {
			if err = adapter.prepareCondition(ctx, resource, condition); err != nil {
				return AdapterExecution{}, err
			}
		}
	}
	return execution.clone(), nil
}

func (adapter *RegisteredFieldAdapter) prepareCondition(
	ctx context.Context,
	resource AdapterResourceContext,
	condition AdapterCondition,
) error {
	if condition.BindingType() != AdapterBindingTypeRegisteredField {
		return myerrors.ErrDataPermissionAdapterTypeUnsupported
	}
	entry, err := adapter.registry.resolve(resource.AdapterCode(), condition.AdapterFieldCode())
	if err != nil {
		return err
	}
	if err = validateRegisteredExecutionCondition(resource, condition, entry); err != nil {
		return err
	}
	request := RegisteredFieldExecutionRequest{
		resource:  resource,
		condition: condition.clone(),
	}
	if err = entry.registration.Executor.Prepare(ctx, request); err != nil {
		var adminError *response.AdminError
		if errors.As(err, &adminError) {
			return err
		}
		return myerrors.ErrRegisteredAdapterPartialConversion
	}
	return nil
}

func validateRegisteredOwnershipDefinitions(input AdapterInput) error {
	definitions := make(map[string]AdapterOwnershipDefinition, len(input.ownerships))
	for _, definition := range input.ownerships {
		definitions[definition.OwnershipCode()] = definition
	}
	for _, group := range input.result.ConditionGroups() {
		for _, condition := range group.Conditions() {
			definition, exists := definitions[condition.OwnershipCode()]
			if !exists {
				return myerrors.ErrDataPermissionAdapterOwnershipMissing
			}
			if definition.BindingType() != AdapterBindingTypeRegisteredField {
				return myerrors.ErrDataPermissionAdapterTypeUnsupported
			}
			if definition.DimensionId() != condition.DimensionId() {
				return myerrors.ErrRegisteredAdapterDimensionUnsupported
			}
			if definition.ValueType() != condition.ValueType() {
				return myerrors.ErrRegisteredAdapterValueTypeUnsupported
			}
		}
	}
	return nil
}

func validateRegisteredExecutionCondition(
	resource AdapterResourceContext,
	condition AdapterCondition,
	entry registeredFieldExecutionEntry,
) error {
	registration := entry.registration
	if registration.ResourceCode != resource.ResourceCode() {
		return myerrors.ErrRegisteredAdapterResourceMismatch
	}
	scopeCondition := condition.ScopeCondition()
	if _, supported := entry.dimensions[scopeCondition.DimensionId()]; !supported {
		return myerrors.ErrRegisteredAdapterDimensionUnsupported
	}
	if registration.ValueType != scopeCondition.ValueType() {
		return myerrors.ErrRegisteredAdapterValueTypeUnsupported
	}
	if _, supported := entry.operations[resource.Operation()]; !supported {
		return myerrors.ErrRegisteredAdapterOperationUnsupported
	}
	if _, supported := entry.operators[scopeCondition.Operator()]; !supported {
		return myerrors.ErrRegisteredAdapterOperatorUnsupported
	}
	valueCount := len(scopeCondition.BigintValues()) + len(scopeCondition.StringValues())
	switch scopeCondition.Operator() {
	case DataScopeOperatorEqual:
		if valueCount != 1 {
			return myerrors.ErrRegisteredAdapterExecutionInvalid
		}
	case DataScopeOperatorIn:
		if valueCount == 0 {
			return myerrors.ErrRegisteredAdapterExecutionInvalid
		}
	default:
		return myerrors.ErrRegisteredAdapterOperatorUnsupported
	}
	return nil
}

func validateRegisteredExecutionComplexity(execution AdapterExecution) error {
	groups := execution.ConditionGroups()
	if len(groups) == 0 || len(groups) > DataScopeMaxConditionGroups {
		return myerrors.ErrRegisteredAdapterExecutionInvalid
	}
	conditionCount := 0
	parameterCount := 0
	for _, group := range groups {
		conditions := group.Conditions()
		if len(conditions) == 0 || len(conditions) > DataScopeMaxConditionsPerGroup {
			return myerrors.ErrRegisteredAdapterExecutionInvalid
		}
		conditionCount += len(conditions)
		for _, condition := range conditions {
			scopeCondition := condition.ScopeCondition()
			parameterCount += len(scopeCondition.BigintValues()) + len(scopeCondition.StringValues())
		}
	}
	if conditionCount > DataScopeMaxConditions || parameterCount > DataScopeMaxTotalParameters {
		return myerrors.ErrRegisteredAdapterExecutionInvalid
	}
	return nil
}

func normalizeRegisteredFieldExecutionRegistration(
	registration RegisteredFieldExecutionRegistration,
) (RegisteredFieldExecutionRegistration, error) {
	registration.ResourceCode = strings.TrimSpace(registration.ResourceCode)
	registration.AdapterCode = strings.TrimSpace(registration.AdapterCode)
	registration.AdapterFieldCode = strings.TrimSpace(registration.AdapterFieldCode)
	registration.ValueType = DataScopeValueType(strings.ToLower(strings.TrimSpace(string(registration.ValueType))))
	registration.DimensionIds = normalizedDimensionIds(registration.DimensionIds)
	registration.SupportedOperations = normalizedUniqueValues(registration.SupportedOperations)
	registration.SupportedOperators = normalizedOperators(registration.SupportedOperators)

	if !dataScopeResourceCodePattern.MatchString(registration.ResourceCode) ||
		!ownershipFieldRegistrationCodePattern.MatchString(registration.AdapterCode) ||
		!ownershipFieldRegistrationCodePattern.MatchString(registration.AdapterFieldCode) ||
		registration.Executor == nil || len(registration.DimensionIds) == 0 ||
		len(registration.SupportedOperations) == 0 || len(registration.SupportedOperators) == 0 {
		return RegisteredFieldExecutionRegistration{}, myerrors.ErrRegisteredAdapterExecutionInvalid
	}
	if registration.ValueType != DataScopeValueTypeBigint &&
		registration.ValueType != DataScopeValueTypeString {
		return RegisteredFieldExecutionRegistration{}, myerrors.ErrRegisteredAdapterValueTypeUnsupported
	}
	for _, dimensionId := range registration.DimensionIds {
		if dimensionId <= 0 {
			return RegisteredFieldExecutionRegistration{}, myerrors.ErrRegisteredAdapterDimensionUnsupported
		}
	}
	for _, operation := range registration.SupportedOperations {
		if _, supported := dataScopeOperations[operation]; !supported {
			return RegisteredFieldExecutionRegistration{}, myerrors.ErrRegisteredAdapterOperationUnsupported
		}
	}
	for _, operator := range registration.SupportedOperators {
		if operator != DataScopeOperatorEqual && operator != DataScopeOperatorIn {
			return RegisteredFieldExecutionRegistration{}, myerrors.ErrRegisteredAdapterOperatorUnsupported
		}
	}
	return registration, nil
}

func registeredFieldDescriptorsEqual(
	left RegisteredFieldExecutionRegistration,
	right RegisteredFieldExecutionRegistration,
) bool {
	return left.ResourceCode == right.ResourceCode &&
		left.AdapterCode == right.AdapterCode &&
		left.AdapterFieldCode == right.AdapterFieldCode &&
		left.ValueType == right.ValueType &&
		intSlicesEqual(left.DimensionIds, right.DimensionIds) &&
		stringSlicesEqual(left.SupportedOperations, right.SupportedOperations) &&
		operatorSlicesEqual(left.SupportedOperators, right.SupportedOperators)
}

func (entry registeredFieldExecutionEntry) clone() registeredFieldExecutionEntry {
	entry.registration.DimensionIds = append([]int(nil), entry.registration.DimensionIds...)
	entry.registration.SupportedOperations = append([]string(nil), entry.registration.SupportedOperations...)
	entry.registration.SupportedOperators = append([]DataScopeOperator(nil), entry.registration.SupportedOperators...)
	entry.dimensions = intSet(entry.registration.DimensionIds)
	entry.operations = stringSet(entry.registration.SupportedOperations)
	entry.operators = operatorSet(entry.registration.SupportedOperators)
	return entry
}

func registeredFieldExecutionKey(adapterCode, adapterFieldCode string) string {
	return adapterCode + "\x00" + adapterFieldCode
}

func normalizedDimensionIds(values []int) []int {
	result := append([]int(nil), values...)
	sort.Ints(result)
	write := 0
	for _, value := range result {
		if write > 0 && result[write-1] == value {
			continue
		}
		result[write] = value
		write++
	}
	return result[:write]
}

func normalizedOperators(values []DataScopeOperator) []DataScopeOperator {
	result := make([]DataScopeOperator, 0, len(values))
	seen := make(map[DataScopeOperator]struct{}, len(values))
	for _, value := range values {
		value = DataScopeOperator(strings.ToLower(strings.TrimSpace(string(value))))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func intSet(values []int) map[int]struct{} {
	result := make(map[int]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func operatorSet(values []DataScopeOperator) map[DataScopeOperator]struct{} {
	result := make(map[DataScopeOperator]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func intSlicesEqual(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func operatorSlicesEqual(left, right []DataScopeOperator) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func mapRegisteredAdapterContractError(err error) error {
	switch {
	case errors.Is(err, myerrors.ErrDataScopeOperatorInvalid):
		return myerrors.ErrRegisteredAdapterOperatorUnsupported
	case errors.Is(err, myerrors.ErrDataScopeValueTypeMismatch),
		errors.Is(err, myerrors.ErrDataScopeValueTypeInvalid):
		return myerrors.ErrRegisteredAdapterValueTypeUnsupported
	case errors.Is(err, myerrors.ErrDataScopeComplexityExceeded):
		return myerrors.ErrRegisteredAdapterExecutionInvalid
	default:
		return err
	}
}
