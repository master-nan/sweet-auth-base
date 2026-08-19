package request

import (
	"backend/model"
	"encoding/json"
)

type QueryScopeReq struct {
	ScopeCode string `form:"scope_code" json:"scope_code" binding:"required,max=128"`
}

type QuerySchemeResolveReq struct {
	ScopeCode        string `json:"scope_code" binding:"required,max=128"`
	ExpectedRevision *int   `json:"expected_revision" binding:"omitempty,min=1"`
}

type QuerySchemeManagementQueryReq struct {
	Page       int                   `json:"page" binding:"omitempty,min=1"`
	Num        int                   `json:"num" binding:"omitempty,min=1,max=100"`
	Name       string                `json:"name" binding:"omitempty,max=64"`
	ScopeCode  string                `json:"scope_code" binding:"omitempty,max=128"`
	SchemeType model.QuerySchemeType `json:"scheme_type"`
	Enabled    *bool                 `json:"enabled"`
}

type QuerySchemePersonalCreateReq struct {
	Name      string          `json:"name" binding:"required,max=64"`
	ScopeCode string          `json:"scope_code" binding:"required,max=128"`
	Payload   json.RawMessage `json:"query_payload" binding:"required"`
	IsDefault bool            `json:"is_default"`
}

type QuerySchemePersonalUpdateReq struct {
	Name      string          `json:"name" binding:"required,max=64"`
	Payload   json.RawMessage `json:"query_payload" binding:"required"`
	IsDefault bool            `json:"is_default"`
	Revision  int             `json:"revision" binding:"required,min=1"`
}

type QuerySchemeRevisionReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,min=1"`
}

type QuerySchemeDefaultReq struct {
	IsDefault bool `json:"is_default"`
	Revision  int  `json:"revision" binding:"required,min=1"`
}

type QuerySchemeSharedCreateReq struct {
	Name       string                `json:"name" binding:"required,max=64"`
	ScopeCode  string                `json:"scope_code" binding:"required,max=128"`
	SchemeType model.QuerySchemeType `json:"scheme_type" binding:"required"`
	Payload    json.RawMessage       `json:"query_payload" binding:"required"`
	Enabled    bool                  `json:"enabled"`
	IsDefault  bool                  `json:"is_default"`
	RoleIDs    []int                 `json:"role_ids" binding:"omitempty,max=32,dive,min=1"`
}

type QuerySchemeSharedUpdateReq struct {
	Name      string          `json:"name" binding:"required,max=64"`
	Payload   json.RawMessage `json:"query_payload" binding:"required"`
	IsDefault bool            `json:"is_default"`
	RoleIDs   []int           `json:"role_ids" binding:"omitempty,max=32,dive,min=1"`
	Revision  int             `json:"revision" binding:"required,min=1"`
}

type QuerySchemeEnabledReq struct {
	Enabled  bool `json:"enabled"`
	Revision int  `json:"revision" binding:"required,min=1"`
}

type QuerySchemeCopyReq struct {
	ScopeCode string `json:"scope_code" binding:"required,max=128"`
	Name      string `json:"name" binding:"required,max=64"`
	IsDefault bool   `json:"is_default"`
}
