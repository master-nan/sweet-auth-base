package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
)

func menuButtonResponse(data model.SysMenuButton) response.SysMenuButtonRes {
	return response.SysMenuButtonRes{
		BasicRes:     basicResponse(data.Basic),
		MenuId:       data.MenuId,
		Name:         data.Name,
		Code:         data.Code,
		Memo:         data.Memo,
		Position:     data.Position,
		EventType:    data.EventType,
		EventAction:  data.EventAction,
		Icon:         data.Icon,
		Color:        data.Color,
		DisplayMode:  data.DisplayMode,
		Sequence:     data.Sequence,
		ApiPath:      data.Path,
		HttpMethod:   data.Method,
		ParamsSchema: data.ParamsSchema,
		ConfirmText:  data.ConfirmText,
		DisableWhen:  data.DisableWhen,
		IsButton:     data.IsButton,
		IsHidden:     data.IsHidden,
		IsDisabled:   data.IsDisabled,
		BeforeHooks:  data.BeforeHooks,
		AfterHooks:   data.AfterHooks,
	}
}

func menuButtonResponses(items []model.SysMenuButton) []response.SysMenuButtonRes {
	result := make([]response.SysMenuButtonRes, 0, len(items))
	for _, item := range items {
		result = append(result, menuButtonResponse(item))
	}
	return result
}

func menuListResponse(data model.SysMenu) response.SysMenuListRes {
	children := make([]response.SysMenuListRes, 0, len(data.Children))
	for _, child := range data.Children {
		children = append(children, menuListResponse(child))
	}
	return response.SysMenuListRes{
		BasicRes:       basicResponse(data.Basic),
		Pid:            data.Pid,
		Name:           data.Name,
		Path:           data.Path,
		Component:      data.Component,
		Title:          data.Title,
		IsHidden:       data.IsHidden,
		Sequence:       data.Sequence,
		PageType:       data.PageType,
		TableCode:      data.TableCode,
		QueryScopeCode: data.QueryScopeCode,
		Option:         data.Option,
		Icon:           data.Icon,
		Redirect:       data.Redirect,
		IsUnfold:       data.IsUnfold,
		DetailOpenMode: data.DetailOpenMode,
		MenuButtons:    menuButtonResponses(data.MenuButtons),
		Children:       children,
	}
}

func menuListResponses(items []model.SysMenu) []response.SysMenuListRes {
	result := make([]response.SysMenuListRes, 0, len(items))
	for _, item := range items {
		result = append(result, menuListResponse(item))
	}
	return result
}

func (s *SysMenuService) GetMenuByIdResponse(id int) (response.SysMenuDetailRes, error) {
	data, err := s.GetMenuById(id)
	if err != nil {
		return response.SysMenuDetailRes{}, err
	}
	return response.SysMenuDetailRes{SysMenuListRes: menuListResponse(data)}, nil
}

func (s *SysMenuService) GetMenuTreeResponse() ([]response.SysMenuListRes, error) {
	data, err := s.GetMenuTree()
	if err != nil {
		return nil, err
	}
	return menuListResponses(data), nil
}

func (s *SysMenuService) GetUserMenusResponse(userId int) ([]response.SysMenuListRes, error) {
	data, err := s.GetUserMenus(userId)
	if err != nil {
		return nil, err
	}
	return menuListResponses(data), nil
}

func (s *SysMenuService) GetMenuButtonsByMenuIdResponse(menuId int) ([]response.SysMenuButtonRes, error) {
	data, err := s.GetMenuButtonsByMenuId(menuId)
	if err != nil {
		return nil, err
	}
	return menuButtonResponses(data), nil
}

func roleListResponse(data model.SysRole) response.SysRoleListRes {
	return response.SysRoleListRes{
		BasicRes: basicResponse(data.Basic),
		Name:     data.Name,
		Memo:     data.Memo,
	}
}

func (s *SysRoleService) GetRoleByIdResponse(id int) (response.SysRoleDetailRes, error) {
	data, err := s.GetRoleById(id)
	if err != nil {
		return response.SysRoleDetailRes{}, err
	}
	return response.SysRoleDetailRes{
		SysRoleListRes: roleListResponse(data),
		Menus:          menuListResponses(data.Menus),
		Buttons:        menuButtonResponses(data.Buttons),
	}, nil
}

func (s *SysRoleService) GetRoleListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.SysRoleListRes], error) {
	result, err := s.GetRoleList(basic, table)
	if err != nil {
		return response.ListResult[response.SysRoleListRes]{}, err
	}
	items := make([]response.SysRoleListRes, 0, len(result.Data))
	for _, item := range result.Data {
		items = append(items, roleListResponse(item))
	}
	return response.ListResult[response.SysRoleListRes]{Data: items, Total: result.Total}, nil
}

func (s *SysRoleService) GetRoleMenusResponse(roleId int) ([]response.SysMenuListRes, error) {
	data, err := s.GetRoleMenus(roleId)
	if err != nil {
		return nil, err
	}
	return menuListResponses(data), nil
}

func (s *SysRoleService) GetRoleMenuButtonsResponse(roleId, menuId int) ([]response.SysMenuButtonRes, error) {
	data, err := s.GetRoleMenuButtons(roleId, menuId)
	if err != nil {
		return nil, err
	}
	return menuButtonResponses(data), nil
}
