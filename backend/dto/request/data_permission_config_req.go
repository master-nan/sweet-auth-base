package request

import (
	"encoding/json"
	"time"
)

const (
	DataPermissionConfigObjectDimension = "dimension"
	DataPermissionConfigObjectResource  = "resource"
	DataPermissionConfigObjectOperation = "resource_operation"
	DataPermissionConfigObjectOwnership = "ownership_field"
	DataPermissionConfigObjectPolicy    = "policy"
	DataPermissionConfigObjectRule      = "policy_rule"
	DataPermissionConfigObjectGrant     = "grant"
)

// DataPermissionConfigQueryReq 开放平台查询协议，但不接受客户端提交的表名、
// 自由过滤条件、删除可见性或数据范围。
type DataPermissionConfigQueryReq struct {
	Page        int               `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num         int               `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order       Order             `form:"order" json:"order"`
	Expressions []ExpressionGroup `form:"expressions" json:"expressions" binding:"omitempty,max=8"`
	QuickQuery  *QuickQuery       `form:"quick_query" json:"quick_query"`
}

func (r DataPermissionConfigQueryReq) ToBasic() Basic {
	return Basic{
		Page:        r.Page,
		Num:         r.Num,
		Order:       r.Order,
		Expressions: r.Expressions,
		QuickQuery:  r.QuickQuery,
		Filters:     make(map[string]any),
	}
}

type DataPermissionConfigIdReq struct {
	Id int `form:"id" json:"id" binding:"required,gt=0"`
}

type DataPermissionConfigStateReq struct {
	State *bool `form:"state" json:"state" binding:"required"`
}

type DataResourcePermissionStateReq struct {
	PermissionEnabled *bool `form:"permission_enabled" json:"permission_enabled" binding:"required"`
}

type DataDimensionDefinitionQueryReq struct {
	DataPermissionConfigQueryReq
	Category  string `form:"category" json:"category" binding:"omitempty,oneof=organization employee user business system"`
	ValueType string `form:"value_type" json:"value_type" binding:"omitempty,oneof=bigint string"`
	State     *bool  `form:"state" json:"state"`
}

func (r DataDimensionDefinitionQueryReq) ToBasic() Basic {
	basic := r.DataPermissionConfigQueryReq.ToBasic()
	setDataPermissionFilter(basic.Filters, "category", r.Category)
	setDataPermissionFilter(basic.Filters, "value_type", r.ValueType)
	setDataPermissionFilter(basic.Filters, "state", r.State)
	return compactDataPermissionFilters(basic)
}

type DataResourceQueryReq struct {
	DataPermissionConfigQueryReq
	ResourceType      string `form:"resource_type" json:"resource_type" binding:"omitempty,oneof=low_code_table business_service report"`
	PermissionEnabled *bool  `form:"permission_enabled" json:"permission_enabled"`
	State             *bool  `form:"state" json:"state"`
}

func (r DataResourceQueryReq) ToBasic() Basic {
	basic := r.DataPermissionConfigQueryReq.ToBasic()
	setDataPermissionFilter(basic.Filters, "resource_type", r.ResourceType)
	setDataPermissionFilter(basic.Filters, "permission_enabled", r.PermissionEnabled)
	setDataPermissionFilter(basic.Filters, "state", r.State)
	return compactDataPermissionFilters(basic)
}

type DataResourceOperationQueryReq struct {
	DataPermissionConfigQueryReq
	ResourceId        *int   `form:"resource_id" json:"resource_id" binding:"omitempty,gt=0"`
	Operation         string `form:"operation" json:"operation" binding:"omitempty,oneof=query detail create update delete export run"`
	PermissionEnabled *bool  `form:"permission_enabled" json:"permission_enabled"`
	State             *bool  `form:"state" json:"state"`
}

func (r DataResourceOperationQueryReq) ToBasic() Basic {
	basic := r.DataPermissionConfigQueryReq.ToBasic()
	setDataPermissionFilter(basic.Filters, "resource_id", r.ResourceId)
	setDataPermissionFilter(basic.Filters, "operation", r.Operation)
	setDataPermissionFilter(basic.Filters, "permission_enabled", r.PermissionEnabled)
	setDataPermissionFilter(basic.Filters, "state", r.State)
	return compactDataPermissionFilters(basic)
}

type DataResourceTargetReq struct {
	ReferenceId   *int    `form:"reference_id" json:"reference_id" binding:"omitempty,gt=0"`
	ReferenceCode *string `form:"reference_code" json:"reference_code" binding:"omitempty,max=128"`
}

type DataResourceCreateReq struct {
	ResourceCode      string                               `form:"resource_code" json:"resource_code" binding:"required,max=128"`
	Name              string                               `form:"name" json:"name" binding:"required,max=128"`
	ResourceType      string                               `form:"resource_type" json:"resource_type" binding:"required,oneof=low_code_table business_service report"`
	Target            DataResourceTargetReq                `form:"target" json:"target"`
	AdapterCode       string                               `form:"adapter_code" json:"adapter_code" binding:"required,max=64"`
	PermissionEnabled *bool                                `form:"permission_enabled" json:"permission_enabled"`
	Description       string                               `form:"description" json:"description" binding:"omitempty,max=256"`
	State             *bool                                `form:"state" json:"state"`
	Operations        []DataResourceOperationCreateItemReq `form:"operations" json:"operations" binding:"omitempty,max=7,dive"`
}

type DataResourceUpdateReq struct {
	Id                int                    `form:"id" json:"id" binding:"required,gt=0"`
	ResourceCode      *string                `form:"resource_code" json:"resource_code" binding:"omitempty,max=128"`
	Name              *string                `form:"name" json:"name" binding:"omitempty,max=128"`
	ResourceType      *string                `form:"resource_type" json:"resource_type" binding:"omitempty,oneof=low_code_table business_service report"`
	Target            *DataResourceTargetReq `form:"target" json:"target"`
	AdapterCode       *string                `form:"adapter_code" json:"adapter_code" binding:"omitempty,max=64"`
	PermissionEnabled *bool                  `form:"permission_enabled" json:"permission_enabled"`
	Description       *string                `form:"description" json:"description" binding:"omitempty,max=256"`
	State             *bool                  `form:"state" json:"state"`
}

type DataResourceOperationCreateItemReq struct {
	Operation         string `form:"operation" json:"operation" binding:"required,oneof=query detail create update delete export run"`
	PermissionEnabled *bool  `form:"permission_enabled" json:"permission_enabled"`
	Description       string `form:"description" json:"description" binding:"omitempty,max=256"`
	State             *bool  `form:"state" json:"state"`
}

type DataResourceOperationBatchReq struct {
	ResourceId int                                  `form:"resource_id" json:"resource_id" binding:"required,gt=0"`
	Items      []DataResourceOperationCreateItemReq `form:"items" json:"items" binding:"required,min=1,max=7,dive"`
}

type DataOwnershipFieldQueryReq struct {
	DataPermissionConfigQueryReq
	ResourceId  *int   `form:"resource_id" json:"resource_id" binding:"omitempty,gt=0"`
	DimensionId *int   `form:"dimension_id" json:"dimension_id" binding:"omitempty,gt=0"`
	BindingType string `form:"binding_type" json:"binding_type" binding:"omitempty,oneof=metadata_field registered_field"`
	ValueType   string `form:"value_type" json:"value_type" binding:"omitempty,oneof=bigint string"`
	State       *bool  `form:"state" json:"state"`
}

func (r DataOwnershipFieldQueryReq) ToBasic() Basic {
	basic := r.DataPermissionConfigQueryReq.ToBasic()
	setDataPermissionFilter(basic.Filters, "resource_id", r.ResourceId)
	setDataPermissionFilter(basic.Filters, "dimension_id", r.DimensionId)
	setDataPermissionFilter(basic.Filters, "binding_type", r.BindingType)
	setDataPermissionFilter(basic.Filters, "value_type", r.ValueType)
	setDataPermissionFilter(basic.Filters, "state", r.State)
	return compactDataPermissionFilters(basic)
}

type DataOwnershipBindingTargetReq struct {
	ReferenceId   *int    `form:"reference_id" json:"reference_id" binding:"omitempty,gt=0"`
	ReferenceCode *string `form:"reference_code" json:"reference_code" binding:"omitempty,max=128"`
}

type DataOwnershipFieldCreateReq struct {
	ResourceId    int                           `form:"resource_id" json:"resource_id" binding:"required,gt=0"`
	OwnershipCode string                        `form:"ownership_code" json:"ownership_code" binding:"required,max=64"`
	DimensionId   int                           `form:"dimension_id" json:"dimension_id" binding:"required,gt=0"`
	BindingType   string                        `form:"binding_type" json:"binding_type" binding:"required,oneof=metadata_field registered_field"`
	BindingTarget DataOwnershipBindingTargetReq `form:"binding_target" json:"binding_target"`
	ValueType     string                        `form:"value_type" json:"value_type" binding:"required,oneof=bigint string"`
	State         *bool                         `form:"state" json:"state"`
}

// DataOwnershipFieldUpdateReq 包含不可变身份字段，便于 Service 明确拒绝修改而不是静默忽略。
type DataOwnershipFieldUpdateReq struct {
	Id            int                            `form:"id" json:"id" binding:"required,gt=0"`
	ResourceId    *int                           `form:"resource_id" json:"resource_id" binding:"omitempty,gt=0"`
	OwnershipCode *string                        `form:"ownership_code" json:"ownership_code" binding:"omitempty,max=64"`
	DimensionId   *int                           `form:"dimension_id" json:"dimension_id" binding:"omitempty,gt=0"`
	BindingType   *string                        `form:"binding_type" json:"binding_type" binding:"omitempty,oneof=metadata_field registered_field"`
	BindingTarget *DataOwnershipBindingTargetReq `form:"binding_target" json:"binding_target"`
	ValueType     *string                        `form:"value_type" json:"value_type" binding:"omitempty,oneof=bigint string"`
	State         *bool                          `form:"state" json:"state"`
}

type DataPolicyQueryReq struct {
	DataPermissionConfigQueryReq
	PolicyType string `form:"policy_type" json:"policy_type" binding:"omitempty,oneof=all none rule_set"`
	State      *bool  `form:"state" json:"state"`
}

func (r DataPolicyQueryReq) ToBasic() Basic {
	basic := r.DataPermissionConfigQueryReq.ToBasic()
	setDataPermissionFilter(basic.Filters, "policy_type", r.PolicyType)
	setDataPermissionFilter(basic.Filters, "state", r.State)
	return compactDataPermissionFilters(basic)
}

type DataPolicyCreateReq struct {
	PolicyCode  string                        `form:"policy_code" json:"policy_code" binding:"required,max=128"`
	Name        string                        `form:"name" json:"name" binding:"required,max=128"`
	Description string                        `form:"description" json:"description" binding:"omitempty,max=512"`
	State       *bool                         `form:"state" json:"state"`
	Rules       []DataPolicyRuleCreateItemReq `form:"rules" json:"rules" binding:"omitempty,max=8,dive"`
}

// DataPolicyUpdateReq 包含 policy_code，便于明确拒绝不可变身份修改而不是静默忽略。
type DataPolicyUpdateReq struct {
	Id          int     `form:"id" json:"id" binding:"required,gt=0"`
	PolicyCode  *string `form:"policy_code" json:"policy_code" binding:"omitempty,max=128"`
	Name        *string `form:"name" json:"name" binding:"omitempty,max=128"`
	Description *string `form:"description" json:"description" binding:"omitempty,max=512"`
	State       *bool   `form:"state" json:"state"`
}

type DataPolicyRuleQueryReq struct {
	DataPermissionConfigQueryReq
	PolicyId      *int   `form:"policy_id" json:"policy_id" binding:"omitempty,gt=0"`
	DimensionId   *int   `form:"dimension_id" json:"dimension_id" binding:"omitempty,gt=0"`
	OwnershipCode string `form:"ownership_code" json:"ownership_code" binding:"omitempty,max=64"`
	ScopeSource   string `form:"scope_source" json:"scope_source" binding:"omitempty,oneof=current_user current_employee effective_legal_entities effective_org_units specified_values provider_subject_scope"`
	Relation      string `form:"relation" json:"relation" binding:"omitempty,oneof=exact self_and_descendants"`
	Operator      string `form:"operator" json:"operator" binding:"omitempty,oneof=eq in"`
	State         *bool  `form:"state" json:"state"`
}

func (r DataPolicyRuleQueryReq) ToBasic() Basic {
	basic := r.DataPermissionConfigQueryReq.ToBasic()
	setDataPermissionFilter(basic.Filters, "policy_id", r.PolicyId)
	setDataPermissionFilter(basic.Filters, "dimension_id", r.DimensionId)
	setDataPermissionFilter(basic.Filters, "ownership_code", r.OwnershipCode)
	setDataPermissionFilter(basic.Filters, "scope_source", r.ScopeSource)
	setDataPermissionFilter(basic.Filters, "relation", r.Relation)
	setDataPermissionFilter(basic.Filters, "operator", r.Operator)
	setDataPermissionFilter(basic.Filters, "state", r.State)
	return compactDataPermissionFilters(basic)
}

type DataPolicyRuleCreateItemReq struct {
	Sequence        int             `form:"sequence" json:"sequence" binding:"required,gt=0"`
	DimensionId     int             `form:"dimension_id" json:"dimension_id" binding:"required,gt=0"`
	OwnershipCode   string          `form:"ownership_code" json:"ownership_code" binding:"required,max=64"`
	ScopeSource     string          `form:"scope_source" json:"scope_source" binding:"required,oneof=effective_legal_entities effective_org_units current_employee specified_values"`
	Relation        string          `form:"relation" json:"relation" binding:"required,oneof=exact self_and_descendants"`
	Operator        string          `form:"operator" json:"operator" binding:"required,oneof=eq in"`
	SpecifiedValues json.RawMessage `form:"specified_values" json:"specified_values"`
	StructureCode   *string         `form:"structure_code" json:"structure_code" binding:"omitempty,max=64"`
	Description     string          `form:"description" json:"description" binding:"omitempty,max=256"`
	State           *bool           `form:"state" json:"state"`
}

type DataPolicyRuleCreateReq struct {
	PolicyId int `form:"policy_id" json:"policy_id" binding:"required,gt=0"`
	DataPolicyRuleCreateItemReq
}

type DataPolicyRuleBatchReq struct {
	PolicyId int                           `form:"policy_id" json:"policy_id" binding:"required,gt=0"`
	Items    []DataPolicyRuleCreateItemReq `form:"items" json:"items" binding:"required,min=1,max=8,dive"`
}

type DataGrantQueryReq struct {
	DataPermissionConfigQueryReq
	SubjectType string `form:"subject_type" json:"subject_type" binding:"omitempty,oneof=role user"`
	SubjectId   *int   `form:"subject_id" json:"subject_id" binding:"omitempty,gt=0"`
	ResourceId  *int   `form:"resource_id" json:"resource_id" binding:"omitempty,gt=0"`
	Operation   string `form:"operation" json:"operation" binding:"omitempty,oneof=query detail create update delete export run"`
	PolicyId    *int   `form:"policy_id" json:"policy_id" binding:"omitempty,gt=0"`
	State       *bool  `form:"state" json:"state"`
}

func (r DataGrantQueryReq) ToBasic() Basic {
	basic := r.DataPermissionConfigQueryReq.ToBasic()
	setDataPermissionFilter(basic.Filters, "subject_type", r.SubjectType)
	setDataPermissionFilter(basic.Filters, "subject_id", r.SubjectId)
	setDataPermissionFilter(basic.Filters, "resource_id", r.ResourceId)
	setDataPermissionFilter(basic.Filters, "operation", r.Operation)
	setDataPermissionFilter(basic.Filters, "policy_id", r.PolicyId)
	setDataPermissionFilter(basic.Filters, "state", r.State)
	return compactDataPermissionFilters(basic)
}

func compactDataPermissionFilters(basic Basic) Basic {
	if len(basic.Filters) == 0 {
		basic.Filters = nil
	}
	return basic
}

func setDataPermissionFilter(filters map[string]any, field string, value any) {
	switch typed := value.(type) {
	case string:
		if typed != "" {
			filters[field] = typed
		}
	case *int:
		if typed != nil {
			filters[field] = *typed
		}
	case *bool:
		if typed != nil {
			filters[field] = *typed
		}
	}
}

type DataGrantCreateReq struct {
	SubjectType string     `form:"subject_type" json:"subject_type" binding:"required,oneof=role user"`
	SubjectId   int        `form:"subject_id" json:"subject_id" binding:"required,gt=0"`
	ResourceId  int        `form:"resource_id" json:"resource_id" binding:"required,gt=0"`
	Operation   string     `form:"operation" json:"operation" binding:"required,oneof=query detail create update delete export run"`
	PolicyId    int        `form:"policy_id" json:"policy_id" binding:"required,gt=0"`
	ValidFrom   *time.Time `form:"valid_from" json:"valid_from"`
	ValidTo     *time.Time `form:"valid_to" json:"valid_to"`
	Description string     `form:"description" json:"description" binding:"omitempty,max=256"`
	State       *bool      `form:"state" json:"state"`
}

type DataGrantBatchCreateReq struct {
	Items []DataGrantCreateReq `form:"items" json:"items" binding:"required,min=1,max=100,dive"`
}

type DataGrantStateReq struct {
	Id    int   `form:"id" json:"id" binding:"required,gt=0"`
	State *bool `form:"state" json:"state" binding:"required"`
}

// DataPermissionFieldBoundary 声明配置 Service 的 DTO 字段归属。
// 返回的切片均为副本，调用方可以安全修改。
type DataPermissionFieldBoundary struct {
	Create            []string
	Update            []string
	ImmutableAfterUse []string
}

var dataPermissionFieldBoundaries = map[string]DataPermissionFieldBoundary{
	DataPermissionConfigObjectDimension: {
		Create:            []string{"dimension_code", "name", "category", "value_type", "provider_code", "selector_type", "state"},
		Update:            []string{"name", "selector_type", "state"},
		ImmutableAfterUse: []string{"dimension_code", "category", "value_type", "provider_code"},
	},
	DataPermissionConfigObjectResource: {
		Create:            []string{"resource_code", "name", "resource_type", "target", "adapter_code", "permission_enabled", "description", "state"},
		Update:            []string{"name", "resource_type", "target", "adapter_code", "permission_enabled", "description", "state"},
		ImmutableAfterUse: []string{"resource_code", "resource_type", "target", "adapter_code"},
	},
	DataPermissionConfigObjectOperation: {
		Create:            []string{"resource_id", "operation", "permission_enabled", "description", "state"},
		Update:            []string{"permission_enabled", "description", "state"},
		ImmutableAfterUse: []string{"resource_id", "operation"},
	},
	DataPermissionConfigObjectOwnership: {
		Create:            []string{"resource_id", "ownership_code", "dimension_id", "binding_type", "binding_target", "value_type", "state"},
		Update:            []string{"state"},
		ImmutableAfterUse: []string{"resource_id", "ownership_code", "dimension_id", "binding_type", "binding_target", "value_type"},
	},
	DataPermissionConfigObjectPolicy: {
		Create:            []string{"policy_code", "name", "description", "state"},
		Update:            []string{"name", "description", "state"},
		ImmutableAfterUse: []string{"policy_code", "policy_type"},
	},
	DataPermissionConfigObjectRule: {
		Create:            []string{"policy_id", "sequence", "dimension_id", "ownership_code", "scope_source", "relation", "operator", "specified_values", "structure_code", "description", "state"},
		Update:            []string{"state"},
		ImmutableAfterUse: []string{"policy_id", "sequence", "dimension_id", "ownership_code", "scope_source", "relation", "operator", "specified_values", "structure_code"},
	},
	DataPermissionConfigObjectGrant: {
		Create:            []string{"subject_type", "subject_id", "resource_id", "operation", "policy_id", "valid_from", "valid_to", "state"},
		Update:            []string{"valid_from", "valid_to", "state"},
		ImmutableAfterUse: []string{"subject_type", "subject_id", "resource_id", "operation", "policy_id"},
	},
}

func GetDataPermissionFieldBoundary(object string) (DataPermissionFieldBoundary, bool) {
	boundary, ok := dataPermissionFieldBoundaries[object]
	if !ok {
		return DataPermissionFieldBoundary{}, false
	}
	return DataPermissionFieldBoundary{
		Create:            append([]string(nil), boundary.Create...),
		Update:            append([]string(nil), boundary.Update...),
		ImmutableAfterUse: append([]string(nil), boundary.ImmutableAfterUse...),
	}, true
}
