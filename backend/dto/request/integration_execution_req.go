package request

type IntegrationExecutionQueryReq struct {
	Page                  int               `form:"page" json:"page" binding:"omitempty,gte=1"`
	Num                   int               `form:"num" json:"num" binding:"omitempty,gte=1,lte=500"`
	Order                 Order             `form:"order" json:"order"`
	Expressions           []ExpressionGroup `form:"expressions" json:"expressions" binding:"omitempty,max=8"`
	QuickQuery            *QuickQuery       `form:"quick_query" json:"quick_query"`
	ExternalSystemID      int               `form:"external_system_id" json:"external_system_id" binding:"omitempty,gt=0"`
	InterfaceDefinitionID int               `form:"interface_definition_id" json:"interface_definition_id" binding:"omitempty,gt=0"`
	TriggerSource         string            `form:"trigger_source" json:"trigger_source" binding:"omitempty,oneof=manual system_event scheduled"`
	Status                string            `form:"status" json:"status" binding:"omitempty,oneof=created running retry_waiting succeeded failed cancelled"`
}

func (r IntegrationExecutionQueryReq) ToBasic() Basic {
	basic := Basic{Page: r.Page, Num: r.Num, Order: r.Order, Expressions: r.Expressions, QuickQuery: r.QuickQuery}
	filters := make(map[string]any, 4)
	if r.ExternalSystemID > 0 {
		filters["external_system_id"] = r.ExternalSystemID
	}
	if r.InterfaceDefinitionID > 0 {
		filters["interface_definition_id"] = r.InterfaceDefinitionID
	}
	if r.TriggerSource != "" {
		filters["trigger_source"] = r.TriggerSource
	}
	if r.Status != "" {
		filters["status"] = r.Status
	}
	if len(filters) > 0 {
		basic.Filters = filters
	}
	return basic
}

type IntegrationExecutionCreateReq struct {
	ExternalSystemID      int    `form:"external_system_id" json:"external_system_id" binding:"required,gt=0"`
	InterfaceDefinitionID int    `form:"interface_definition_id" json:"interface_definition_id" binding:"required,gt=0"`
	TriggerSource         string `form:"trigger_source" json:"trigger_source" binding:"required,oneof=manual system_event scheduled"`
	IdempotencyScope      string `form:"idempotency_scope" json:"idempotency_scope" binding:"required,max=64"`
	IdempotencyKey        string `form:"idempotency_key" json:"idempotency_key" binding:"required,max=128"`
	InputHash             string `form:"input_hash" json:"input_hash" binding:"required,len=64,hexadecimal"`
}

type IntegrationExecutionCancelReq struct {
	Revision int `form:"revision" json:"revision" binding:"required,gt=0"`
}
