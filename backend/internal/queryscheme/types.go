package queryscheme

import (
	"backend/dto/request"
	"backend/model"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const (
	SchemaVersion          int16 = 1
	MaxPayloadBytes              = 32 * 1024
	MaxTopLevelGroups            = 8
	MaxRules                     = 50
	MaxSchemaDepth               = 3
	MaxMultiValues               = 100
	MaxKeywordLength             = 256
	MaxNameLength                = 64
	MaxRoleCount                 = 32
	SharedManageCapability       = "query_scheme_shared_manage"
)

type ValidationStatus string

const (
	ValidationValid    ValidationStatus = "VALID"
	ValidationDegraded ValidationStatus = "DEGRADED"
	ValidationInvalid  ValidationStatus = "INVALID"
)

type IssueCode string

const (
	IssueFieldUnavailable     IssueCode = "field_unavailable"
	IssueFieldNotQueryable    IssueCode = "field_not_queryable"
	IssueOperatorIncompatible IssueCode = "operator_incompatible"
	IssueValueInvalid         IssueCode = "value_invalid"
	IssueSortUnavailable      IssueCode = "sort_unavailable"
	IssueBindingUnavailable   IssueCode = "binding_unavailable"
	IssueBindingUnresolvable  IssueCode = "binding_unresolvable"
	IssuePayloadInvalid       IssueCode = "payload_invalid"
	IssueScopeUnavailable     IssueCode = "scope_unavailable"
)

type ValidationIssue struct {
	Code      IssueCode `json:"code"`
	FieldCode string    `json:"field_code,omitempty"`
	Path      string    `json:"path,omitempty"`
	Message   string    `json:"message"`
}

type ValidationResult struct {
	Status ValidationStatus  `json:"status"`
	Issues []ValidationIssue `json:"issues"`
}

type QuerySchemePayloadV1 struct {
	Expressions []request.ExpressionGroup `json:"expressions"`
	QuickQuery  request.QuickQuery        `json:"quick_query"`
	Order       request.Order             `json:"order"`
	Bindings    []Binding                 `json:"bindings"`
}

type Binding struct {
	Pointer string        `json:"pointer"`
	Kind    BindingKind   `json:"kind"`
	Params  BindingParams `json:"params,omitempty"`
}

type BindingParams struct {
	DayOffset   *int `json:"day_offset,omitempty"`
	WeekOffset  *int `json:"week_offset,omitempty"`
	MonthOffset *int `json:"month_offset,omitempty"`
}

type ResolvedQuery struct {
	Expressions []request.ExpressionGroup `json:"expressions"`
	QuickQuery  request.QuickQuery        `json:"quick_query"`
	Order       request.Order             `json:"order"`
}

type SchemeRecord struct {
	Model   model.QueryScheme
	Payload QuerySchemePayloadV1
}

func DecodePayload(raw []byte) (QuerySchemePayloadV1, error) {
	if len(raw) == 0 || len(raw) > MaxPayloadBytes {
		return QuerySchemePayloadV1{}, fmt.Errorf("query scheme payload size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var payload QuerySchemePayloadV1
	if err := decoder.Decode(&payload); err != nil {
		return QuerySchemePayloadV1{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return QuerySchemePayloadV1{}, fmt.Errorf("query scheme payload contains trailing content")
	}
	return payload, nil
}
