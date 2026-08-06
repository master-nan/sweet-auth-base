package request

import "encoding/json"

type InterfaceDefinitionQueryReq struct {
	Page             int               `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num              int               `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order            Order             `form:"order" json:"order"`
	Expressions      []ExpressionGroup `form:"expressions" json:"expressions" binding:"omitempty,max=8"`
	QuickQuery       *QuickQuery       `form:"quick_query" json:"quick_query"`
	ExternalSystemID int               `form:"external_system_id" json:"external_system_id" binding:"omitempty,gt=0"`
	HTTPMethod       string            `form:"http_method" json:"http_method" binding:"omitempty,oneof=GET POST PUT PATCH DELETE"`
	Status           string            `form:"status" json:"status" binding:"omitempty,oneof=draft enabled disabled"`
}

func (r InterfaceDefinitionQueryReq) ToBasic() Basic {
	basic := Basic{Page: r.Page, Num: r.Num, Order: r.Order, Expressions: r.Expressions, QuickQuery: r.QuickQuery}
	filters := make(map[string]any, 3)
	if r.ExternalSystemID > 0 {
		filters["external_system_id"] = r.ExternalSystemID
	}
	if r.HTTPMethod != "" {
		filters["http_method"] = r.HTTPMethod
	}
	if r.Status != "" {
		filters["status"] = r.Status
	}
	if len(filters) > 0 {
		basic.Filters = filters
	}
	return basic
}

type InterfaceDefinitionCreateReq struct {
	ExternalSystemID int             `form:"external_system_id" json:"external_system_id" binding:"required,gt=0"`
	InterfaceCode    string          `form:"interface_code" json:"interface_code" binding:"required,max=64"`
	Name             string          `form:"name" json:"name" binding:"required,max=128"`
	Protocol         string          `form:"protocol" json:"protocol" binding:"required,oneof=http https"`
	HTTPMethod       string          `form:"http_method" json:"http_method" binding:"required,oneof=GET POST PUT PATCH DELETE"`
	RelativePath     string          `form:"relative_path" json:"relative_path" binding:"required,max=512"`
	InputContract    json.RawMessage `form:"input_contract" json:"input_contract,omitempty"`
	CredentialID     *int            `form:"credential_id" json:"credential_id" binding:"omitempty,gt=0"`
	TimeoutSeconds   int             `form:"timeout_seconds" json:"timeout_seconds" binding:"required,gte=1"`
	ResponseLimit    int64           `form:"response_limit" json:"response_limit" binding:"required,gte=1024"`
	RetryPolicyID    *int            `form:"retry_policy_id" json:"retry_policy_id" binding:"omitempty,gt=0"`
	Description      string          `form:"description" json:"description" binding:"omitempty,max=512"`
}

type InterfaceDefinitionUpdateReq struct {
	ExternalSystemID *int            `form:"external_system_id" json:"external_system_id" binding:"omitempty,gt=0"`
	InterfaceCode    *string         `form:"interface_code" json:"interface_code" binding:"omitempty,max=64"`
	Version          *int            `form:"version" json:"version" binding:"omitempty,gt=0"`
	Name             *string         `form:"name" json:"name" binding:"omitempty,max=128"`
	Protocol         *string         `form:"protocol" json:"protocol" binding:"omitempty,oneof=http https"`
	HTTPMethod       *string         `form:"http_method" json:"http_method" binding:"omitempty,oneof=GET POST PUT PATCH DELETE"`
	RelativePath     *string         `form:"relative_path" json:"relative_path" binding:"omitempty,max=512"`
	InputContract    json.RawMessage `form:"input_contract" json:"input_contract,omitempty"`
	CredentialID     *int            `form:"credential_id" json:"credential_id" binding:"omitempty,gt=0"`
	ClearCredential  bool            `form:"clear_credential" json:"clear_credential"`
	TimeoutSeconds   *int            `form:"timeout_seconds" json:"timeout_seconds" binding:"omitempty,gte=1"`
	ResponseLimit    *int64          `form:"response_limit" json:"response_limit" binding:"omitempty,gte=1024"`
	RetryPolicyID    *int            `form:"retry_policy_id" json:"retry_policy_id" binding:"omitempty,gt=0"`
	ClearRetryPolicy bool            `form:"clear_retry_policy" json:"clear_retry_policy"`
	Description      *string         `form:"description" json:"description" binding:"omitempty,max=512"`
	Revision         int             `form:"revision" json:"revision" binding:"required,gt=0"`
}

type InterfaceDefinitionVersionReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,gt=0"`
}

type InterfaceDefinitionStateReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,gt=0"`
}
