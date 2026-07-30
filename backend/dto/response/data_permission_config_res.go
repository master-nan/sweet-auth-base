package response

import (
	"backend/model"
	"encoding/json"
	"time"
)

type DataPermissionConfigBaseRes struct {
	Id        int              `json:"id"`
	GmtCreate model.CustomTime `json:"gmt_create"`
	GmtModify model.CustomTime `json:"gmt_modify"`
	State     bool             `json:"state"`
}

type DataDimensionDefinitionListRes struct {
	DataPermissionConfigBaseRes
	DimensionCode string  `json:"dimension_code"`
	Name          string  `json:"name"`
	Category      string  `json:"category"`
	ValueType     string  `json:"value_type"`
	ProviderCode  string  `json:"provider_code"`
	SelectorType  *string `json:"selector_type,omitempty"`
}

type DataDimensionDefinitionDetailRes struct {
	DataDimensionDefinitionListRes
}

type DataResourceTargetRes struct {
	Type          string  `json:"type"`
	ReferenceId   *int    `json:"reference_id,omitempty"`
	ReferenceCode *string `json:"reference_code,omitempty"`
}

type DataResourceListRes struct {
	DataPermissionConfigBaseRes
	ResourceCode      string `json:"resource_code"`
	Name              string `json:"name"`
	ResourceType      string `json:"resource_type"`
	PermissionEnabled bool   `json:"permission_enabled"`
}

type DataResourceDetailRes struct {
	DataResourceListRes
	AdapterCode string                `json:"adapter_code"`
	Target      DataResourceTargetRes `json:"target"`
}

type DataResourceOperationListRes struct {
	DataPermissionConfigBaseRes
	ResourceId        int    `json:"resource_id"`
	Operation         string `json:"operation"`
	PermissionEnabled bool   `json:"permission_enabled"`
}

type DataResourceOperationDetailRes struct {
	DataResourceOperationListRes
	Resource *DataPermissionReferenceSummaryRes `json:"resource,omitempty"`
}

type DataOwnershipBindingTargetRes struct {
	Type          string  `json:"type"`
	ReferenceId   *int    `json:"reference_id,omitempty"`
	ReferenceCode *string `json:"reference_code,omitempty"`
}

type DataOwnershipFieldListRes struct {
	DataPermissionConfigBaseRes
	ResourceId    int    `json:"resource_id"`
	OwnershipCode string `json:"ownership_code"`
	DimensionId   int    `json:"dimension_id"`
	BindingType   string `json:"binding_type"`
	ValueType     string `json:"value_type"`
}

type DataOwnershipFieldDetailRes struct {
	DataOwnershipFieldListRes
	BindingTarget DataOwnershipBindingTargetRes      `json:"binding_target"`
	Resource      *DataPermissionReferenceSummaryRes `json:"resource,omitempty"`
	Dimension     *DataPermissionReferenceSummaryRes `json:"dimension,omitempty"`
}

type DataPolicyListRes struct {
	DataPermissionConfigBaseRes
	PolicyCode string `json:"policy_code"`
	Name       string `json:"name"`
	PolicyType string `json:"policy_type"`
}

type DataPolicyDetailRes struct {
	DataPolicyListRes
	Rules []DataPolicyRuleDetailRes `json:"rules"`
}

type DataPolicyRuleListRes struct {
	DataPermissionConfigBaseRes
	PolicyId      int    `json:"policy_id"`
	Sequence      int    `json:"sequence"`
	DimensionId   int    `json:"dimension_id"`
	OwnershipCode string `json:"ownership_code"`
	ScopeSource   string `json:"scope_source"`
	Relation      string `json:"relation"`
	Operator      string `json:"operator"`
}

type DataPolicyRuleDetailRes struct {
	DataPolicyRuleListRes
	SpecifiedValues json.RawMessage                    `json:"specified_values,omitempty"`
	StructureCode   *string                            `json:"structure_code,omitempty"`
	Policy          *DataPermissionReferenceSummaryRes `json:"policy,omitempty"`
	Dimension       *DataPermissionReferenceSummaryRes `json:"dimension,omitempty"`
}

type DataGrantListRes struct {
	DataPermissionConfigBaseRes
	SubjectType string     `json:"subject_type"`
	SubjectId   int        `json:"subject_id"`
	ResourceId  int        `json:"resource_id"`
	Operation   string     `json:"operation"`
	PolicyId    int        `json:"policy_id"`
	ValidFrom   *time.Time `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to"`
}

type DataGrantDetailRes struct {
	DataGrantListRes
	Resource *DataPermissionReferenceSummaryRes `json:"resource,omitempty"`
	Policy   *DataPermissionReferenceSummaryRes `json:"policy,omitempty"`
}

type DataPermissionReferenceSummaryRes struct {
	Id   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

func NewDataDimensionDefinitionListRes(value model.DataDimensionDefinition) DataDimensionDefinitionListRes {
	return DataDimensionDefinitionListRes{
		DataPermissionConfigBaseRes: newDataPermissionConfigBaseRes(value.Basic),
		DimensionCode:               value.Code,
		Name:                        value.Name,
		Category:                    value.Category,
		ValueType:                   value.ValueType,
		ProviderCode:                value.ProviderCode,
		SelectorType:                value.SelectorType,
	}
}

func NewDataResourceListRes(value model.DataResource) DataResourceListRes {
	return DataResourceListRes{
		DataPermissionConfigBaseRes: newDataPermissionConfigBaseRes(value.Basic),
		ResourceCode:                value.ResourceCode,
		Name:                        value.Name,
		ResourceType:                value.ResourceType,
		PermissionEnabled:           value.PermissionEnabled,
	}
}

func NewDataResourceDetailRes(value model.DataResource) DataResourceDetailRes {
	target := DataResourceTargetRes{Type: value.ResourceType}
	switch value.ResourceType {
	case model.DataResourceTypeLowCodeTable:
		target.ReferenceId = value.TableId
	case model.DataResourceTypeBusinessService:
		target.ReferenceCode = value.ServiceCode
	case model.DataResourceTypeReport:
		target.ReferenceId = value.ReportDefinitionId
	}
	return DataResourceDetailRes{
		DataResourceListRes: NewDataResourceListRes(value),
		AdapterCode:         value.AdapterCode,
		Target:              target,
	}
}

func NewDataResourceOperationListRes(value model.DataResourceOperation) DataResourceOperationListRes {
	return DataResourceOperationListRes{
		DataPermissionConfigBaseRes: newDataPermissionConfigBaseRes(value.Basic),
		ResourceId:                  value.ResourceId,
		Operation:                   value.Operation,
		PermissionEnabled:           value.PermissionEnabled,
	}
}

func NewDataOwnershipFieldListRes(value model.DataOwnershipField) DataOwnershipFieldListRes {
	return DataOwnershipFieldListRes{
		DataPermissionConfigBaseRes: newDataPermissionConfigBaseRes(value.Basic),
		ResourceId:                  value.ResourceId,
		OwnershipCode:               value.OwnershipCode,
		DimensionId:                 value.DimensionId,
		BindingType:                 value.BindingType,
		ValueType:                   value.ValueType,
	}
}

func NewDataOwnershipFieldDetailRes(value model.DataOwnershipField) DataOwnershipFieldDetailRes {
	target := DataOwnershipBindingTargetRes{Type: value.BindingType}
	if value.BindingType == model.DataOwnershipBindingTypeMetadataField {
		target.ReferenceId = value.TableFieldId
	} else {
		target.ReferenceCode = value.AdapterFieldCode
	}
	return DataOwnershipFieldDetailRes{
		DataOwnershipFieldListRes: NewDataOwnershipFieldListRes(value),
		BindingTarget:             target,
	}
}

func NewDataPolicyListRes(value model.DataPolicy) DataPolicyListRes {
	return DataPolicyListRes{
		DataPermissionConfigBaseRes: newDataPermissionConfigBaseRes(value.Basic),
		PolicyCode:                  value.Code,
		Name:                        value.Name,
		PolicyType:                  value.PolicyType,
	}
}

func NewDataPolicyDetailRes(value model.DataPolicy) DataPolicyDetailRes {
	return DataPolicyDetailRes{
		DataPolicyListRes: NewDataPolicyListRes(value),
		Rules:             []DataPolicyRuleDetailRes{},
	}
}

func NewDataPolicyRuleListRes(value model.DataPolicyRule) DataPolicyRuleListRes {
	return DataPolicyRuleListRes{
		DataPermissionConfigBaseRes: newDataPermissionConfigBaseRes(value.Basic),
		PolicyId:                    value.PolicyId,
		Sequence:                    value.Sequence,
		DimensionId:                 value.DimensionId,
		OwnershipCode:               value.OwnershipCode,
		ScopeSource:                 value.ScopeSource,
		Relation:                    value.Relation,
		Operator:                    value.Operator,
	}
}

func NewDataPolicyRuleDetailRes(value model.DataPolicyRule) DataPolicyRuleDetailRes {
	specifiedValues := append(json.RawMessage(nil), value.SpecifiedValues...)
	return DataPolicyRuleDetailRes{
		DataPolicyRuleListRes: NewDataPolicyRuleListRes(value),
		SpecifiedValues:       specifiedValues,
		StructureCode:         value.StructureCode,
	}
}

func NewDataGrantListRes(value model.DataGrant) DataGrantListRes {
	return DataGrantListRes{
		DataPermissionConfigBaseRes: newDataPermissionConfigBaseRes(value.Basic),
		SubjectType:                 value.SubjectType,
		SubjectId:                   value.SubjectId,
		ResourceId:                  value.ResourceId,
		Operation:                   value.Operation,
		PolicyId:                    value.PolicyId,
		ValidFrom:                   value.ValidFrom,
		ValidTo:                     value.ValidTo,
	}
}

func newDataPermissionConfigBaseRes(value model.Basic) DataPermissionConfigBaseRes {
	return DataPermissionConfigBaseRes{
		Id:        value.Id,
		GmtCreate: value.GmtCreate,
		GmtModify: value.GmtModify,
		State:     value.State,
	}
}
