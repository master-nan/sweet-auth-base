package response

type SysRoleListRes struct {
	BasicRes
	Name string `json:"name"`
	Memo string `json:"memo"`
}

type SysRoleDetailRes struct {
	SysRoleListRes
	Menus   []SysMenuListRes   `json:"menus"`
	Buttons []SysMenuButtonRes `json:"buttons"`
}
