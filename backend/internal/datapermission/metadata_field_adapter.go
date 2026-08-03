package datapermission

import (
	"context"
	"errors"
	"strings"

	"backend/enum"
	myerrors "backend/internal/errors"
)

var (
	ErrMetadataTableRecordNotFound = errors.New("metadata table record not found")
	ErrMetadataFieldRecordNotFound = errors.New("metadata field record not found")
)

// MetadataTableRecord 是 Metadata Adapter 所需的最小服务端投影。
// 它不包含表名和 SQL 元数据。
type MetadataTableRecord struct {
	Id      int
	State   bool
	Deleted bool
}

// MetadataFieldRecord 是服务端解析后的字段投影。
// FieldCode 仅用于安全分类，不会复制到 AdapterExecution。
type MetadataFieldRecord struct {
	Id               int
	TableId          int
	State            bool
	Deleted          bool
	FieldCode        string
	FieldType        enum.SysTableFieldType
	InputType        enum.SysTableFieldInputType
	FieldCategory    enum.SysTableFieldCategory
	Expression       string
	IsPrimaryKey     bool
	IsAdvancedSearch bool
}

// MetadataFieldReader 按数字身份加载已评审的元数据。
// 具体 Reader 必须使用参数化查询，不得接受 AdapterInput 中的表名或字段名。
type MetadataFieldReader interface {
	FindMetadataTable(context.Context, int) (MetadataTableRecord, error)
	FindMetadataField(context.Context, int) (MetadataFieldRecord, error)
}

// MetadataFieldAdapter 校验元数据绑定并返回已冻结的结构化 AdapterExecution 树。
// 它不生成 SQL 或 ORM Clause。
type MetadataFieldAdapter struct {
	reader MetadataFieldReader
}

var _ Adapter = (*MetadataFieldAdapter)(nil)

func NewMetadataFieldAdapter(reader MetadataFieldReader) (*MetadataFieldAdapter, error) {
	if reader == nil {
		return nil, myerrors.ErrDataPermissionAdapterInputInvalid
	}
	return &MetadataFieldAdapter{reader: reader}, nil
}

func (adapter *MetadataFieldAdapter) Apply(
	ctx context.Context,
	input AdapterInput,
) (AdapterExecution, error) {
	if adapter == nil || adapter.reader == nil {
		return AdapterExecution{}, myerrors.ErrMetadataAdapterFailed
	}
	if err := input.Validate(); err != nil {
		return AdapterExecution{}, err
	}
	if err := validateMetadataOwnershipDefinitions(input); err != nil {
		return AdapterExecution{}, err
	}

	execution, err := BuildAdapterExecution(input)
	if err != nil {
		return AdapterExecution{}, mapMetadataAdapterContractError(err)
	}
	if execution.Mode() != AdapterExecutionModeApplyFilter {
		return execution.clone(), nil
	}

	resource := input.ResourceContext()
	if resource.TableId() <= 0 {
		return AdapterExecution{}, myerrors.ErrMetadataAdapterResourceTableMissing
	}
	if err = validateMetadataExecutionComplexity(execution); err != nil {
		return AdapterExecution{}, err
	}

	session := metadataAdapterSession{
		reader: adapter.reader,
		fields: make(map[int]MetadataFieldRecord),
	}
	if _, err = session.loadTable(ctx, resource.TableId()); err != nil {
		return AdapterExecution{}, err
	}
	for _, group := range execution.ConditionGroups() {
		for _, condition := range group.Conditions() {
			if err = session.validateCondition(ctx, resource.TableId(), condition); err != nil {
				return AdapterExecution{}, err
			}
		}
	}
	return execution.clone(), nil
}

func validateMetadataOwnershipDefinitions(input AdapterInput) error {
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
			if definition.BindingType() != AdapterBindingTypeMetadataField {
				return myerrors.ErrDataPermissionAdapterTypeUnsupported
			}
			if definition.DimensionId() != condition.DimensionId() {
				return myerrors.ErrDataPermissionAdapterOwnershipMismatch
			}
			if definition.ValueType() != condition.ValueType() {
				return myerrors.ErrMetadataAdapterValueTypeMismatch
			}
		}
	}
	return nil
}

type metadataAdapterSession struct {
	reader      MetadataFieldReader
	tableLoaded bool
	table       MetadataTableRecord
	fields      map[int]MetadataFieldRecord
}

func (session *metadataAdapterSession) loadTable(
	ctx context.Context,
	tableId int,
) (MetadataTableRecord, error) {
	if session.tableLoaded {
		return session.table, nil
	}
	table, err := session.reader.FindMetadataTable(ctx, tableId)
	if errors.Is(err, ErrMetadataTableRecordNotFound) {
		return MetadataTableRecord{}, myerrors.ErrMetadataAdapterTableNotFound
	}
	if err != nil {
		return MetadataTableRecord{}, myerrors.ErrMetadataAdapterFailed
	}
	if table.Id != tableId || !table.State || table.Deleted {
		return MetadataTableRecord{}, myerrors.ErrMetadataAdapterTableNotFound
	}
	session.table = table
	session.tableLoaded = true
	return table, nil
}

func (session *metadataAdapterSession) loadField(
	ctx context.Context,
	fieldId int,
) (MetadataFieldRecord, error) {
	if field, exists := session.fields[fieldId]; exists {
		return field, nil
	}
	field, err := session.reader.FindMetadataField(ctx, fieldId)
	if errors.Is(err, ErrMetadataFieldRecordNotFound) {
		return MetadataFieldRecord{}, myerrors.ErrMetadataAdapterFieldNotFound
	}
	if err != nil {
		return MetadataFieldRecord{}, myerrors.ErrMetadataAdapterFailed
	}
	if field.Id != fieldId {
		return MetadataFieldRecord{}, myerrors.ErrMetadataAdapterFieldNotFound
	}
	session.fields[fieldId] = field
	return field, nil
}

func (session *metadataAdapterSession) validateCondition(
	ctx context.Context,
	tableId int,
	condition AdapterCondition,
) error {
	if condition.BindingType() != AdapterBindingTypeMetadataField || condition.TableFieldId() <= 0 {
		return myerrors.ErrDataPermissionAdapterTypeUnsupported
	}
	scopeCondition := condition.ScopeCondition()
	if err := validateMetadataOperator(scopeCondition); err != nil {
		return err
	}

	field, err := session.loadField(ctx, condition.TableFieldId())
	if err != nil {
		return err
	}
	if field.Deleted || !field.State {
		return myerrors.ErrMetadataAdapterFieldInactive
	}
	if field.TableId != tableId {
		return myerrors.ErrMetadataAdapterFieldResourceMismatch
	}
	if !isMetadataFilterField(field) {
		return myerrors.ErrMetadataAdapterFieldNotFilterable
	}
	valueType, supported := metadataFieldValueType(field.FieldType)
	if !supported {
		return myerrors.ErrMetadataAdapterFieldTypeUnsupported
	}
	if valueType != scopeCondition.ValueType() {
		return myerrors.ErrMetadataAdapterFieldTypeDrift
	}
	return nil
}

func validateMetadataOperator(condition DataScopeCondition) error {
	valueCount := len(condition.BigintValues()) + len(condition.StringValues())
	switch condition.Operator() {
	case DataScopeOperatorEqual:
		if valueCount != 1 {
			return myerrors.ErrMetadataAdapterValueTypeMismatch
		}
	case DataScopeOperatorIn:
		if valueCount == 0 {
			return myerrors.ErrMetadataAdapterValueTypeMismatch
		}
	default:
		return myerrors.ErrMetadataAdapterOperatorUnsupported
	}
	return nil
}

func validateMetadataExecutionComplexity(execution AdapterExecution) error {
	groups := execution.ConditionGroups()
	if len(groups) == 0 || len(groups) > DataScopeMaxConditionGroups {
		return myerrors.ErrMetadataAdapterComplexityExceeded
	}
	conditionCount := 0
	parameterCount := 0
	for _, group := range groups {
		conditions := group.Conditions()
		if len(conditions) == 0 || len(conditions) > DataScopeMaxConditionsPerGroup {
			return myerrors.ErrMetadataAdapterComplexityExceeded
		}
		conditionCount += len(conditions)
		for _, condition := range conditions {
			scopeCondition := condition.ScopeCondition()
			parameterCount += len(scopeCondition.BigintValues()) + len(scopeCondition.StringValues())
		}
	}
	if conditionCount > DataScopeMaxConditions || parameterCount > DataScopeMaxTotalParameters {
		return myerrors.ErrMetadataAdapterComplexityExceeded
	}
	return nil
}

func metadataFieldValueType(fieldType enum.SysTableFieldType) (DataScopeValueType, bool) {
	switch fieldType {
	case enum.BigIntFieldType, enum.IntFieldType, enum.TinyintFieldType:
		return DataScopeValueTypeBigint, true
	case enum.VarcharFieldType, enum.TextFieldType:
		return DataScopeValueTypeString, true
	default:
		return "", false
	}
}

func isMetadataFilterField(field MetadataFieldRecord) bool {
	if !field.IsAdvancedSearch || field.IsPrimaryKey || strings.TrimSpace(field.Expression) != "" {
		return false
	}
	if field.FieldCategory != "" && field.FieldCategory != enum.NormalField {
		return false
	}
	if field.InputType == enum.FilePickerInputType || field.InputType == enum.RichTextInputType {
		return false
	}
	return !isForbiddenMetadataFilterFieldCode(field.FieldCode)
}

func isForbiddenMetadataFilterFieldCode(fieldCode string) bool {
	fieldCode = strings.ToLower(strings.TrimSpace(fieldCode))
	if fieldCode == "" {
		return true
	}
	for _, prefix := range []string{"gmt_", "source_", "create_", "modify_", "delete_"} {
		if strings.HasPrefix(fieldCode, prefix) {
			return true
		}
	}
	switch fieldCode {
	case "id", "path", "level", "parent_id", "parent_node_id", "structure_node_id",
		"tree_path", "node_path", "name", "display_name", "display_value", "label":
		return true
	}
	return strings.HasSuffix(fieldCode, "_name") ||
		strings.HasSuffix(fieldCode, "_label") ||
		strings.HasSuffix(fieldCode, "_display")
}

func mapMetadataAdapterContractError(err error) error {
	switch {
	case errors.Is(err, myerrors.ErrDataScopeComplexityExceeded):
		return myerrors.ErrMetadataAdapterComplexityExceeded
	case errors.Is(err, myerrors.ErrDataScopeOperatorInvalid):
		return myerrors.ErrMetadataAdapterOperatorUnsupported
	case errors.Is(err, myerrors.ErrDataScopeValueTypeMismatch),
		errors.Is(err, myerrors.ErrDataScopeValueTypeInvalid):
		return myerrors.ErrMetadataAdapterValueTypeMismatch
	default:
		return err
	}
}
