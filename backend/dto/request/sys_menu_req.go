/**
 * @Author: Nan
 * @Date: 2024/7/26 下午5:14
 */

package request

import "backend/enum"

type MenuCreateReq struct {
	Pid       *int                 `json:"pid" binding:"required"`
	Name      string               `json:"name" binding:"required"`
	Path      string               `json:"path" binding:"required"`
	Component string               `json:"component" binding:"required"`
	Title     string               `json:"title" binding:"required"`
	IsHidden  *bool                `json:"is_hidden" binding:"required"`
	Sequence  *uint8               `json:"sequence" binding:"required"`
	PageType  enum.SysMenuPageType `json:"page_type"`
	TableCode string               `json:"table_code"`
	Option    string               `json:"option"`
	Icon      *string              `json:"icon"`
	Redirect  *string              `json:"redirect"`
}

type MenuUpdateReq struct {
	Id        int                  `json:"id" binding:"required"`
	Pid       *int                 `json:"pid" binding:"required"`
	Name      string               `json:"name" binding:"required"`
	Path      string               `json:"path" binding:"required"`
	Component string               `json:"component" binding:"required"`
	Title     string               `json:"title" binding:"required"`
	IsHidden  *bool                `json:"is_hidden" binding:"required"`
	Sequence  *uint8               `json:"sequence" binding:"required"`
	PageType  enum.SysMenuPageType `json:"page_type"`
	TableCode string               `json:"table_code"`
	Option    string               `json:"option"`
	Icon      *string              `json:"icon"`
	Redirect  *string              `json:"redirect"`
}

type MenuOrderItemReq struct {
	Id       int   `json:"id" binding:"required"`
	Sequence uint8 `json:"sequence" binding:"required"`
}

type MenuOrderUpdateReq struct {
	Menus []MenuOrderItemReq `json:"menus" binding:"required"`
}
