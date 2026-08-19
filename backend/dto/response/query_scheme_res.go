package response

import (
	"backend/internal/queryscheme"
	"backend/model"
)

type QueryScopeConfigRes struct {
	ScopeCode           string                    `json:"scope_code"`
	ScopeLabel          string                    `json:"scope_label"`
	TableCode           string                    `json:"table_code"`
	QuickDateField      string                    `json:"quick_date_field,omitempty"`
	QuickPresets        []queryscheme.QuickPreset `json:"quick_presets"`
	VirtualSortFields   []string                  `json:"virtual_sort_fields"`
	DynamicBindingKinds []queryscheme.BindingKind `json:"dynamic_binding_kinds"`
}

type QuerySchemeSummaryRes struct {
	ID        int                          `json:"id"`
	Name      string                       `json:"name"`
	Type      model.QuerySchemeType        `json:"type"`
	IsDefault bool                         `json:"is_default"`
	Status    queryscheme.ValidationStatus `json:"status"`
}

type QuerySchemeListRes struct {
	QuerySchemeSummaryRes
	ScopeCode  string           `json:"scope_code"`
	ScopeLabel string           `json:"scope_label"`
	Enabled    bool             `json:"enabled"`
	Creator    string           `json:"creator_display_name,omitempty"`
	RoleIDs    []int            `json:"role_ids,omitempty"`
	Revision   int              `json:"revision"`
	UpdatedAt  model.CustomTime `json:"updated_at"`
}

type QuerySchemeDetailRes struct {
	QuerySchemeListRes
	Payload queryscheme.QuerySchemePayloadV1 `json:"query_payload"`
	Issues  []queryscheme.ValidationIssue    `json:"issues"`
}

type QuerySchemeResolveSourceRes struct {
	ID        int                   `json:"id"`
	Name      string                `json:"name"`
	Type      model.QuerySchemeType `json:"type"`
	Revision  int                   `json:"revision"`
	IsDefault bool                  `json:"is_default"`
}

type QuerySchemeResolveRes struct {
	Scheme           QuerySchemeResolveSourceRes   `json:"scheme"`
	ValidationStatus queryscheme.ValidationStatus  `json:"validation_status"`
	Issues           []queryscheme.ValidationIssue `json:"issues"`
	ResolvedQuery    *queryscheme.ResolvedQuery    `json:"resolved_query,omitempty"`
	BindingKinds     []queryscheme.BindingKind     `json:"binding_kinds"`
}
