package utils

// ResolveMenuButtonType 兼容历史菜单按钮数据：新数据以 is_button 判断是否页面按钮，
// 老数据缺少 is_button 时仍按 is_hidden 推断按钮类型。
func ResolveMenuButtonType(isButton *bool, legacyHidden bool) (bool, bool) {
	if isButton != nil {
		return *isButton, legacyHidden
	}
	return !legacyHidden, legacyHidden
}
