/**
 * @Author: Nan
 * @Date: 2024/8/10 下午3:15
 */

package request

import "backend/enum"

type MenuButtonCreateReq struct {
	MenuId       int                           `json:"menu_id" binding:"required"`
	Name         string                        `json:"name" binding:"required"`
	Code         string                        `json:"code" binding:"required"`
	Icon         string                        `json:"icon"`
	Color        string                        `json:"color"`
	Sequence     uint8                         `json:"sequence"`
	Memo         string                        `json:"memo"`
	Position     enum.SysMenuButtonPosition    `json:"position" binding:"required"`
	EventType    string                        `json:"event_type"`
	EventAction  string                        `json:"event_action"`
	ApiPath      string                        `json:"api_path"`
	HttpMethod   string                        `json:"http_method"`
	DisplayMode  enum.SysMenuButtonDisplayMode `json:"display_mode"`
	ParamsSchema string                        `json:"params_schema"`
	ConfirmText  string                        `json:"confirm_text"`
	DisableWhen  string                        `json:"disable_when"`
	IsButton     *bool                         `json:"is_button"`
	IsHidden     bool                          `json:"is_hidden"`
	IsDisabled   bool                          `json:"is_disabled"`
	BeforeHooks  string                        `json:"before_hooks"`
	AfterHooks   string                        `json:"after_hooks"`
}

type MenuButtonUpdateReq struct {
	Id           int                           `json:"id" binding:"required"`
	MenuId       int                           `json:"menu_id" binding:"required"`
	Name         string                        `json:"name" binding:"required"`
	Code         string                        `json:"code" binding:"required"`
	Icon         string                        `json:"icon"`
	Color        string                        `json:"color"`
	Sequence     uint8                         `json:"sequence"`
	Memo         string                        `json:"memo"`
	Position     enum.SysMenuButtonPosition    `json:"position" binding:"required"`
	EventType    string                        `json:"event_type"`
	EventAction  string                        `json:"event_action"`
	ApiPath      string                        `json:"api_path"`
	HttpMethod   string                        `json:"http_method"`
	DisplayMode  enum.SysMenuButtonDisplayMode `json:"display_mode"`
	ParamsSchema string                        `json:"params_schema"`
	ConfirmText  string                        `json:"confirm_text"`
	DisableWhen  string                        `json:"disable_when"`
	IsButton     *bool                         `json:"is_button"`
	IsHidden     bool                          `json:"is_hidden"`
	IsDisabled   bool                          `json:"is_disabled"`
	BeforeHooks  string                        `json:"before_hooks"`
	AfterHooks   string                        `json:"after_hooks"`
}
