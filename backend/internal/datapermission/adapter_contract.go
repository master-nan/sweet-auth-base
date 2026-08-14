package datapermission

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	myerrors "backend/internal/errors"
)

type AdapterBindingType string

const (
	AdapterBindingTypeMetadataField   AdapterBindingType = "metadata_field"
	AdapterBindingTypeRegisteredField AdapterBindingType = "registered_field"
)

// AdapterExecutionMode 使用显式模式，避免消费者根据空 Condition 切片推断授权。
type AdapterExecutionMode string

const (
	AdapterExecutionModeNotApplicable AdapterExecutionMode = "not_applicable"
	AdapterExecutionModeAllowAll      AdapterExecutionMode = "allow_all"
	AdapterExecutionModeDenyAll       AdapterExecutionMode = "deny_all"
	AdapterExecutionModeApplyFilter   AdapterExecutionMode = "apply_filter"
)

type AdapterResourceContextInput struct {
	ResourceCode string
	Operation    string
	AdapterCode  string
	TableId      int
}

// AdapterResourceContext 仅包含服务端解析的 Resource 身份，不包含表名和字段名。
type AdapterResourceContext struct {
	resourceCode string
	operation    string
	adapterCode  string
	tableId      int
}

type AdapterOwnershipDefinitionInput struct {
	OwnershipCode    string
	DimensionId      int
	BindingType      AdapterBindingType
	TableFieldId     int
	AdapterFieldCode string
	ValueType        DataScopeValueType
}

// AdapterOwnershipDefinition 是 Ownership 配置到 Adapter 的安全桥梁。
// Binding 仅包含元数据 ID 或已评审的注册编码，不包含表名、字段表达式或 SQL 片段。
type AdapterOwnershipDefinition struct {
	ownershipCode    string
	dimensionId      int
	bindingType      AdapterBindingType
	tableFieldId     int
	adapterFieldCode string
	valueType        DataScopeValueType
}

type AdapterInput struct {
	resource   AdapterResourceContext
	result     DataScopeResult
	ownerships []AdapterOwnershipDefinition
}

type AdapterCondition struct {
	condition        DataScopeCondition
	bindingType      AdapterBindingType
	tableFieldId     int
	adapterFieldCode string
}

type AdapterConditionGroup struct {
	conditions []AdapterCondition
}

// AdapterExecution 是不可变、受控的执行能力，并保持与 ORM 和数据库无关。
// 具体 Adapter 使用其中的 Binding ID 或注册编码，但不得改变 Resolver 决策。
type AdapterExecution struct {
	resourceCode string
	operation    string
	mode         AdapterExecutionMode
	groups       []AdapterConditionGroup
}

// Adapter 将已校验的 DataScopeResult 应用于目标执行能力。
// 实现可以解析元数据 ID 或已评审的服务端注册项，但不得加载或重新解释 Policy 配置。
type Adapter interface {
	Apply(context.Context, AdapterInput) (AdapterExecution, error)
}

type AdapterFunc func(context.Context, AdapterInput) (AdapterExecution, error)

var _ Adapter = AdapterFunc(nil)

func NewAdapterResourceContext(
	input AdapterResourceContextInput,
) (AdapterResourceContext, error) {
	resource := AdapterResourceContext{
		resourceCode: strings.TrimSpace(input.ResourceCode),
		operation:    strings.ToLower(strings.TrimSpace(input.Operation)),
		adapterCode:  strings.TrimSpace(input.AdapterCode),
		tableId:      input.TableId,
	}
	if err := resource.Validate(); err != nil {
		return AdapterResourceContext{}, err
	}
	return resource, nil
}

func (resource AdapterResourceContext) Validate() error {
	if !dataScopeResourceCodePattern.MatchString(resource.resourceCode) {
		return myerrors.ErrDataPermissionAdapterInputInvalid
	}
	if _, supported := dataScopeOperations[resource.operation]; !supported {
		return myerrors.ErrDataPermissionAdapterInputInvalid
	}
	if !ownershipFieldRegistrationCodePattern.MatchString(resource.adapterCode) ||
		resource.tableId < 0 {
		return myerrors.ErrDataPermissionAdapterInputInvalid
	}
	return nil
}

func (resource AdapterResourceContext) ResourceCode() string {
	return resource.resourceCode
}

func (resource AdapterResourceContext) Operation() string {
	return resource.operation
}

func (resource AdapterResourceContext) AdapterCode() string {
	return resource.adapterCode
}

func (resource AdapterResourceContext) TableId() int {
	return resource.tableId
}

func NewAdapterOwnershipDefinition(
	input AdapterOwnershipDefinitionInput,
) (AdapterOwnershipDefinition, error) {
	definition := AdapterOwnershipDefinition{
		ownershipCode:    strings.TrimSpace(input.OwnershipCode),
		dimensionId:      input.DimensionId,
		bindingType:      AdapterBindingType(strings.ToLower(strings.TrimSpace(string(input.BindingType)))),
		tableFieldId:     input.TableFieldId,
		adapterFieldCode: strings.TrimSpace(input.AdapterFieldCode),
		valueType:        DataScopeValueType(strings.ToLower(strings.TrimSpace(string(input.ValueType)))),
	}
	if err := definition.Validate(); err != nil {
		return AdapterOwnershipDefinition{}, err
	}
	return definition, nil
}

func (definition AdapterOwnershipDefinition) Validate() error {
	if !dataScopeOwnershipCodePattern.MatchString(definition.ownershipCode) ||
		definition.dimensionId <= 0 ||
		definition.valueType != DataScopeValueTypeBigint &&
			definition.valueType != DataScopeValueTypeString {
		return myerrors.ErrDataPermissionAdapterOwnershipMismatch
	}
	switch definition.bindingType {
	case AdapterBindingTypeMetadataField:
		if definition.tableFieldId <= 0 || definition.adapterFieldCode != "" {
			return myerrors.ErrDataPermissionAdapterOwnershipMismatch
		}
	case AdapterBindingTypeRegisteredField:
		if definition.tableFieldId != 0 ||
			!ownershipFieldRegistrationCodePattern.MatchString(definition.adapterFieldCode) {
			return myerrors.ErrDataPermissionAdapterOwnershipMismatch
		}
	default:
		return myerrors.ErrDataPermissionAdapterTypeUnsupported
	}
	return nil
}

func (definition AdapterOwnershipDefinition) OwnershipCode() string {
	return definition.ownershipCode
}

func (definition AdapterOwnershipDefinition) DimensionId() int {
	return definition.dimensionId
}

func (definition AdapterOwnershipDefinition) BindingType() AdapterBindingType {
	return definition.bindingType
}

func (definition AdapterOwnershipDefinition) TableFieldId() int {
	return definition.tableFieldId
}

func (definition AdapterOwnershipDefinition) AdapterFieldCode() string {
	return definition.adapterFieldCode
}

func (definition AdapterOwnershipDefinition) ValueType() DataScopeValueType {
	return definition.valueType
}

func NewAdapterInput(
	resource AdapterResourceContext,
	result DataScopeResult,
	ownerships []AdapterOwnershipDefinition,
) (AdapterInput, error) {
	input := AdapterInput{
		resource:   resource,
		result:     result,
		ownerships: append([]AdapterOwnershipDefinition(nil), ownerships...),
	}
	if err := input.Validate(); err != nil {
		return AdapterInput{}, err
	}
	return input, nil
}

func (input AdapterInput) Validate() error {
	if err := input.resource.Validate(); err != nil {
		return err
	}
	if err := input.result.Validate(); err != nil {
		return myerrors.ErrDataPermissionAdapterInputInvalid
	}
	if input.result.ResourceCode() != input.resource.resourceCode ||
		input.result.Operation() != input.resource.operation {
		return myerrors.ErrDataPermissionAdapterInputInvalid
	}
	seen := make(map[string]struct{}, len(input.ownerships))
	for _, definition := range input.ownerships {
		if err := definition.Validate(); err != nil {
			return err
		}
		if _, exists := seen[definition.ownershipCode]; exists {
			return myerrors.ErrDataPermissionAdapterOwnershipMismatch
		}
		seen[definition.ownershipCode] = struct{}{}
	}
	return nil
}

func (input AdapterInput) ResourceContext() AdapterResourceContext {
	return input.resource
}

func (input AdapterInput) DataScopeResult() DataScopeResult {
	cloned, _ := input.result.clone()
	return cloned
}

func (input AdapterInput) OwnershipDefinitions() []AdapterOwnershipDefinition {
	return append([]AdapterOwnershipDefinition(nil), input.ownerships...)
}

func (apply AdapterFunc) Apply(
	ctx context.Context,
	input AdapterInput,
) (AdapterExecution, error) {
	if err := input.Validate(); err != nil {
		return AdapterExecution{}, err
	}
	if apply == nil {
		return AdapterExecution{}, myerrors.ErrDataPermissionAdapterFailed
	}
	execution, err := apply(ctx, input)
	if err != nil {
		return AdapterExecution{}, normalizeAdapterError(err)
	}
	if err = execution.Validate(); err != nil {
		return AdapterExecution{}, myerrors.ErrDataPermissionAdapterExecutionInvalid
	}
	if execution.resourceCode != input.resource.resourceCode ||
		execution.operation != input.resource.operation ||
		execution.mode != adapterModeForDecision(input.result.Decision()) {
		return AdapterExecution{}, myerrors.ErrDataPermissionAdapterExecutionInvalid
	}
	return execution.clone(), nil
}

// BuildAdapterExecution 仅执行安全契约转换。
// 它不解析元数据字段、不加载注册项、不生成 SQL，也不操作查询。
func BuildAdapterExecution(input AdapterInput) (AdapterExecution, error) {
	if err := input.Validate(); err != nil {
		return AdapterExecution{}, err
	}
	mode := adapterModeForDecision(input.result.Decision())
	if mode != AdapterExecutionModeApplyFilter {
		return newAdapterExecution(input.resource, mode, nil)
	}

	definitions := make(map[string]AdapterOwnershipDefinition, len(input.ownerships))
	for _, definition := range input.ownerships {
		definitions[definition.ownershipCode] = definition
	}
	groups := make([]AdapterConditionGroup, 0, len(input.result.ConditionGroups()))
	for _, scopeGroup := range input.result.ConditionGroups() {
		conditions := make([]AdapterCondition, 0, len(scopeGroup.Conditions()))
		for _, scopeCondition := range scopeGroup.Conditions() {
			definition, exists := definitions[scopeCondition.OwnershipCode()]
			if !exists {
				return AdapterExecution{}, myerrors.ErrDataPermissionAdapterOwnershipMissing
			}
			condition, err := newAdapterCondition(scopeCondition, definition)
			if err != nil {
				return AdapterExecution{}, err
			}
			conditions = append(conditions, condition)
		}
		groups = append(groups, AdapterConditionGroup{conditions: conditions})
	}
	return newAdapterExecution(input.resource, mode, groups)
}

func newAdapterCondition(
	condition DataScopeCondition,
	definition AdapterOwnershipDefinition,
) (AdapterCondition, error) {
	if condition.OwnershipCode() != definition.ownershipCode ||
		condition.DimensionId() != definition.dimensionId ||
		condition.ValueType() != definition.valueType {
		return AdapterCondition{}, myerrors.ErrDataPermissionAdapterOwnershipMismatch
	}
	result := AdapterCondition{
		condition:        condition.clone(),
		bindingType:      definition.bindingType,
		tableFieldId:     definition.tableFieldId,
		adapterFieldCode: definition.adapterFieldCode,
	}
	if err := result.Validate(); err != nil {
		return AdapterCondition{}, err
	}
	return result, nil
}

func (condition AdapterCondition) Validate() error {
	if err := condition.condition.Validate(); err != nil {
		return myerrors.ErrDataPermissionAdapterExecutionInvalid
	}
	switch condition.bindingType {
	case AdapterBindingTypeMetadataField:
		if condition.tableFieldId <= 0 || condition.adapterFieldCode != "" {
			return myerrors.ErrDataPermissionAdapterExecutionInvalid
		}
	case AdapterBindingTypeRegisteredField:
		if condition.tableFieldId != 0 ||
			!ownershipFieldRegistrationCodePattern.MatchString(condition.adapterFieldCode) {
			return myerrors.ErrDataPermissionAdapterExecutionInvalid
		}
	default:
		return myerrors.ErrDataPermissionAdapterTypeUnsupported
	}
	return nil
}

func (condition AdapterCondition) ScopeCondition() DataScopeCondition {
	return condition.condition.clone()
}

func (condition AdapterCondition) BindingType() AdapterBindingType {
	return condition.bindingType
}

func (condition AdapterCondition) TableFieldId() int {
	return condition.tableFieldId
}

func (condition AdapterCondition) AdapterFieldCode() string {
	return condition.adapterFieldCode
}

func (group AdapterConditionGroup) Conditions() []AdapterCondition {
	result := make([]AdapterCondition, len(group.conditions))
	for index, condition := range group.conditions {
		result[index] = condition.clone()
	}
	return result
}

func newAdapterExecution(
	resource AdapterResourceContext,
	mode AdapterExecutionMode,
	groups []AdapterConditionGroup,
) (AdapterExecution, error) {
	execution := AdapterExecution{
		resourceCode: resource.resourceCode,
		operation:    resource.operation,
		mode:         mode,
		groups:       cloneAdapterGroups(groups),
	}
	if err := execution.Validate(); err != nil {
		return AdapterExecution{}, err
	}
	return execution, nil
}

func (execution AdapterExecution) Validate() error {
	if !dataScopeResourceCodePattern.MatchString(execution.resourceCode) {
		return myerrors.ErrDataPermissionAdapterExecutionInvalid
	}
	if _, supported := dataScopeOperations[execution.operation]; !supported {
		return myerrors.ErrDataPermissionAdapterExecutionInvalid
	}
	switch execution.mode {
	case AdapterExecutionModeNotApplicable,
		AdapterExecutionModeAllowAll,
		AdapterExecutionModeDenyAll:
		if len(execution.groups) != 0 {
			return myerrors.ErrDataPermissionAdapterExecutionInvalid
		}
		return nil
	case AdapterExecutionModeApplyFilter:
		if len(execution.groups) == 0 {
			return myerrors.ErrDataPermissionAdapterExecutionInvalid
		}
	default:
		return myerrors.ErrDataPermissionAdapterExecutionInvalid
	}
	for _, group := range execution.groups {
		if len(group.conditions) == 0 {
			return myerrors.ErrDataPermissionAdapterExecutionInvalid
		}
		for _, condition := range group.conditions {
			if err := condition.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (execution AdapterExecution) ResourceCode() string {
	return execution.resourceCode
}

func (execution AdapterExecution) Operation() string {
	return execution.operation
}

func (execution AdapterExecution) Mode() AdapterExecutionMode {
	return execution.mode
}

func (execution AdapterExecution) ConditionGroups() []AdapterConditionGroup {
	return cloneAdapterGroups(execution.groups)
}

func (execution AdapterExecution) MarshalJSON() ([]byte, error) {
	if err := execution.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ResourceCode string                  `json:"resource_code"`
		Operation    string                  `json:"operation"`
		Mode         AdapterExecutionMode    `json:"mode"`
		Groups       []AdapterConditionGroup `json:"condition_groups"`
	}{execution.resourceCode, execution.operation, execution.mode, execution.ConditionGroups()})
}

func (condition AdapterCondition) MarshalJSON() ([]byte, error) {
	if err := condition.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Condition        DataScopeCondition `json:"condition"`
		BindingType      AdapterBindingType `json:"binding_type"`
		TableFieldId     int                `json:"table_field_id,omitempty"`
		AdapterFieldCode string             `json:"adapter_field_code,omitempty"`
	}{condition.ScopeCondition(), condition.bindingType, condition.tableFieldId, condition.adapterFieldCode})
}

func (group AdapterConditionGroup) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Conditions []AdapterCondition `json:"all_of_conditions"`
	}{group.Conditions()})
}

func (condition AdapterCondition) clone() AdapterCondition {
	condition.condition = condition.condition.clone()
	return condition
}

func (execution AdapterExecution) clone() AdapterExecution {
	execution.groups = execution.ConditionGroups()
	return execution
}

func cloneAdapterGroups(groups []AdapterConditionGroup) []AdapterConditionGroup {
	result := make([]AdapterConditionGroup, len(groups))
	for index, group := range groups {
		result[index] = AdapterConditionGroup{conditions: group.Conditions()}
	}
	return result
}

func adapterModeForDecision(decision DataScopeDecision) AdapterExecutionMode {
	switch decision {
	case DataScopeDecisionNotApplicable:
		return AdapterExecutionModeNotApplicable
	case DataScopeDecisionAll:
		return AdapterExecutionModeAllowAll
	case DataScopeDecisionNone:
		return AdapterExecutionModeDenyAll
	case DataScopeDecisionFiltered:
		return AdapterExecutionModeApplyFilter
	default:
		return ""
	}
}

func normalizeAdapterError(err error) error {
	var applicationError *myerrors.ApplicationError
	if errors.As(err, &applicationError) {
		return err
	}
	return myerrors.ErrDataPermissionAdapterFailed
}
