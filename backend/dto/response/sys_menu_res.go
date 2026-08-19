package response

import "backend/enum"

type SysMenuButtonRes struct {
	BasicRes
	MenuId       int                           `json:"menu_id"`
	Name         string                        `json:"name"`
	Code         string                        `json:"code"`
	Memo         string                        `json:"memo"`
	Position     enum.SysMenuButtonPosition    `json:"position"`
	EventType    string                        `json:"event_type"`
	EventAction  string                        `json:"event_action"`
	Icon         string                        `json:"icon"`
	Color        string                        `json:"color"`
	DisplayMode  enum.SysMenuButtonDisplayMode `json:"display_mode"`
	Sequence     uint8                         `json:"sequence"`
	ApiPath      string                        `json:"api_path"`
	HttpMethod   string                        `json:"http_method"`
	ParamsSchema string                        `json:"params_schema"`
	ConfirmText  string                        `json:"confirm_text"`
	DisableWhen  string                        `json:"disable_when"`
	IsButton     bool                          `json:"is_button"`
	IsHidden     bool                          `json:"is_hidden"`
	IsDisabled   bool                          `json:"is_disabled"`
	BeforeHooks  string                        `json:"before_hooks"`
	AfterHooks   string                        `json:"after_hooks"`
}

// SysMenuListRes 是菜单树、当前用户菜单和角色菜单共用的白名单节点。
type SysMenuListRes struct {
	BasicRes
	Pid            int                    `json:"pid"`
	Name           string                 `json:"name"`
	Path           string                 `json:"path"`
	Component      string                 `json:"component"`
	Title          string                 `json:"title"`
	IsHidden       bool                   `json:"is_hidden"`
	Sequence       uint8                  `json:"sequence"`
	PageType       enum.SysMenuPageType   `json:"page_type"`
	TableCode      string                 `json:"table_code"`
	QueryScopeCode *string                `json:"query_scope_code,omitempty"`
	Option         string                 `json:"option"`
	Icon           *string                `json:"icon"`
	Redirect       *string                `json:"redirect"`
	IsUnfold       bool                   `json:"is_unfold"`
	DetailOpenMode enum.SysDetailOpenMode `json:"detail_open_mode,omitempty"`
	MenuButtons    []SysMenuButtonRes     `json:"menu_buttons"`
	Children       []SysMenuListRes       `json:"children"`
}

type SysMenuDetailRes struct {
	SysMenuListRes
}
