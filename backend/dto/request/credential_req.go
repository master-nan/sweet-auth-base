package request

import "time"

type CredentialQueryReq struct {
	Page             int               `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num              int               `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order            Order             `form:"order" json:"order"`
	Expressions      []ExpressionGroup `form:"expressions" json:"expressions" binding:"omitempty,max=8"`
	QuickQuery       *QuickQuery       `form:"quick_query" json:"quick_query"`
	ExternalSystemID int               `form:"external_system_id" json:"external_system_id" binding:"omitempty,gt=0"`
	CredentialType   string            `form:"credential_type" json:"credential_type" binding:"omitempty,oneof=basic api_key bearer_token oauth_client"`
	Status           string            `form:"status" json:"status" binding:"omitempty,oneof=draft active disabled revoked expired"`
}

func (r CredentialQueryReq) ToBasic() Basic {
	return Basic{Page: r.Page, Num: r.Num, Order: r.Order, Expressions: r.Expressions, QuickQuery: r.QuickQuery}
}

type CredentialCreateReq struct {
	ExternalSystemID int               `form:"external_system_id" json:"external_system_id" binding:"required,gt=0"`
	CredentialCode   string            `form:"credential_code" json:"credential_code" binding:"required,max=64"`
	Name             string            `form:"name" json:"name" binding:"required,max=128"`
	CredentialType   string            `form:"credential_type" json:"credential_type" binding:"required,oneof=basic api_key bearer_token oauth_client"`
	Secret           map[string]string `form:"secret" json:"secret" binding:"required"`
	ExpiresAt        *time.Time        `form:"expires_at" json:"expires_at"`
	Description      string            `form:"description" json:"description" binding:"omitempty,max=512"`
}

type CredentialUpdateReq struct {
	ExternalSystemID *int       `form:"external_system_id" json:"external_system_id" binding:"omitempty,gt=0"`
	CredentialCode   *string    `form:"credential_code" json:"credential_code" binding:"omitempty,max=64"`
	CredentialType   *string    `form:"credential_type" json:"credential_type" binding:"omitempty,oneof=basic api_key bearer_token oauth_client"`
	Name             *string    `form:"name" json:"name" binding:"omitempty,max=128"`
	ExpiresAt        *time.Time `form:"expires_at" json:"expires_at"`
	ClearExpiresAt   bool       `form:"clear_expires_at" json:"clear_expires_at"`
	Description      *string    `form:"description" json:"description" binding:"omitempty,max=512"`
	Revision         int        `form:"revision" json:"revision" binding:"required,gt=0"`
}

type CredentialRotateReq struct {
	Secret   map[string]string `form:"secret" json:"secret" binding:"required"`
	Revision int               `form:"revision" json:"revision" binding:"required,gt=0"`
}

type CredentialStateReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,gt=0"`
}
