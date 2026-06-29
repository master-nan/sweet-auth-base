package request

type DataPermissionDimensionCreateReq struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	ValueType   string `json:"value_type"`
	SourceType  string `json:"source_type"`
	SourceCode  string `json:"source_code"`
	LabelField  string `json:"label_field"`
	ValueField  string `json:"value_field"`
	ParentField string `json:"parent_field"`
	Memo        string `json:"memo"`
	State       *bool  `json:"state"`
}

type DataPermissionDimensionUpdateReq struct {
	Id          int    `json:"id" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	ValueType   string `json:"value_type"`
	SourceType  string `json:"source_type"`
	SourceCode  string `json:"source_code"`
	LabelField  string `json:"label_field"`
	ValueField  string `json:"value_field"`
	ParentField string `json:"parent_field"`
	Memo        string `json:"memo"`
	State       *bool  `json:"state"`
}

type DataPermissionBindingItemReq struct {
	Id            int      `json:"id"`
	DimensionCode string   `json:"dimension_code" binding:"required"`
	FieldCode     string   `json:"field_code" binding:"required"`
	MatchType     string   `json:"match_type"`
	Required      *bool    `json:"required"`
	Actions       []string `json:"actions"`
	State         *bool    `json:"state"`
}

type DataPermissionBindingSaveReq struct {
	MenuId   int                            `json:"menu_id" binding:"required"`
	Bindings []DataPermissionBindingItemReq `json:"bindings"`
}

type RoleDataPermissionItemReq struct {
	MenuId        int      `json:"menu_id" binding:"required"`
	TableCode     string   `json:"table_code"`
	DimensionCode string   `json:"dimension_code" binding:"required"`
	Strategy      string   `json:"strategy" binding:"required"`
	ScopeValues   []string `json:"scope_values"`
	State         *bool    `json:"state"`
}

type RoleDataPermissionSaveReq struct {
	RoleId      int                         `json:"role_id" binding:"required"`
	Permissions []RoleDataPermissionItemReq `json:"permissions"`
}

type UserDataPermissionOverrideItemReq struct {
	MenuId        int      `json:"menu_id" binding:"required"`
	TableCode     string   `json:"table_code"`
	DimensionCode string   `json:"dimension_code" binding:"required"`
	Strategy      string   `json:"strategy" binding:"required"`
	ScopeValues   []string `json:"scope_values"`
	OverrideMode  string   `json:"override_mode"`
	ExpireAt      string   `json:"expire_at"`
	State         *bool    `json:"state"`
}

type UserDataPermissionOverrideSaveReq struct {
	UserId    int                                 `json:"user_id" binding:"required"`
	Overrides []UserDataPermissionOverrideItemReq `json:"overrides"`
}

type UserDimensionValueItemReq struct {
	DimensionCode string   `json:"dimension_code" binding:"required"`
	ScopeValues   []string `json:"scope_values"`
	State         *bool    `json:"state"`
}

type UserDimensionValueSaveReq struct {
	UserId int                         `json:"user_id" binding:"required"`
	Items  []UserDimensionValueItemReq `json:"items"`
}
