package utils

// ResolveMenuButtonType 兼容历史菜单按钮数据：新数据以 is_button 判断是否页面按钮，
// 老数据仍保留 is_hidden 原值，避免迁移时把接口权限误改成可见按钮。
func ResolveMenuButtonType(isButton *bool, legacyHidden bool) (bool, bool) {
	if isButton != nil {
		return *isButton, legacyHidden
	}
	return !legacyHidden, legacyHidden
}
