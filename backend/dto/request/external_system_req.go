package request

import (
	"backend/enum"
	"strings"
)

type ExternalSystemQueryReq struct {
	Page        int               `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num         int               `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order       Order             `form:"order" json:"order"`
	Expressions []ExpressionGroup `form:"expressions" json:"expressions" binding:"omitempty,max=8"`
	QuickQuery  *QuickQuery       `form:"quick_query" json:"quick_query"`
	SystemType  string            `form:"system_type" json:"system_type" binding:"omitempty,oneof=hr erp tms wms other"`
	Status      string            `form:"status" json:"status" binding:"omitempty,oneof=draft enabled disabled"`
	Owner       string            `form:"owner" json:"owner" binding:"omitempty,max=128"`
}

func (r ExternalSystemQueryReq) ToBasic() Basic {
	basic := Basic{
		Page:        r.Page,
		Num:         r.Num,
		Order:       r.Order,
		Expressions: r.Expressions,
		QuickQuery:  r.QuickQuery,
	}
	filters := make(map[string]any, 2)
	if r.SystemType != "" {
		filters["system_type"] = r.SystemType
	}
	if r.Status != "" {
		filters["status"] = r.Status
	}
	if len(filters) > 0 {
		basic.Filters = filters
	}
	if owner := strings.TrimSpace(r.Owner); owner != "" {
		basic.Expressions = append(basic.Expressions, ExpressionGroup{
			Logic: enum.Or,
			Rules: []QueryRule{
				{Field: "owner_identifier", ExpressionType: enum.Like, Value: owner, Type: enum.VarcharFieldType},
				{Field: "owner_name", ExpressionType: enum.Like, Value: owner, Type: enum.VarcharFieldType},
			},
		})
	}
	return basic
}

type ExternalSystemCreateReq struct {
	SystemCode      string `form:"system_code" json:"system_code" binding:"required,max=64"`
	Name            string `form:"name" json:"name" binding:"required,max=128"`
	SystemType      string `form:"system_type" json:"system_type" binding:"required,oneof=hr erp tms wms other"`
	BaseURL         string `form:"base_url" json:"base_url" binding:"required,max=512"`
	OwnerIdentifier string `form:"owner_identifier" json:"owner_identifier" binding:"required,max=128"`
	OwnerName       string `form:"owner_name" json:"owner_name" binding:"required,max=128"`
	Description     string `form:"description" json:"description" binding:"omitempty,max=512"`
}

type ExternalSystemUpdateReq struct {
	SystemCode      *string `form:"system_code" json:"system_code" binding:"omitempty,max=64"`
	Name            *string `form:"name" json:"name" binding:"omitempty,max=128"`
	SystemType      *string `form:"system_type" json:"system_type" binding:"omitempty,oneof=hr erp tms wms other"`
	BaseURL         *string `form:"base_url" json:"base_url" binding:"omitempty,max=512"`
	OwnerIdentifier *string `form:"owner_identifier" json:"owner_identifier" binding:"omitempty,max=128"`
	OwnerName       *string `form:"owner_name" json:"owner_name" binding:"omitempty,max=128"`
	Description     *string `form:"description" json:"description" binding:"omitempty,max=512"`
	Revision        int     `form:"revision" json:"revision" binding:"required,gt=0"`
}

type ExternalSystemStateReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,gt=0"`
}
