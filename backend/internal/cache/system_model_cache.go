package cache

import "backend/model"

const (
	ConfigureCacheKey  = "CONFIGURE_CACHE_KEY_"
	DictCacheKey       = "DICT_CACHE_KEY_"
	TableCacheKey      = "TABLE_CACHE_KEY_"
	TableFieldCacheKey = "TABLE_FIELD_CACHE_KEY_"
	UserCacheKey       = "USER_CACHE_KEY_"

	// 以下前缀仅供Migration清理已经退出Runtime依赖图的旧缓存键。
	MenuButtonCacheKey     = "MENU_BUTTON_CACHE_KEY_"
	MenuCacheKey           = "MENU_CACHE_KEY_"
	RoleCacheKey           = "ROLE_CACHE_KEY_"
	RoleMenuButtonCacheKey = "ROLE_MENU_BUTTON_CACHE_KEY_"
	RoleMenuCacheKey       = "ROLE_MENU_CACHE_KEY_"
	UserRoleCacheKey       = "USER_ROLE_CACHE_KEY_"
	GeneralizationCacheKey = "GENERALIZATION_CACHE_KEY_"
	BlackUserCacheKey      = "BLACK_USER_CACHE_KEY_"
)

type SysConfigureCache struct {
	*BasicCache[model.SysConfigure]
}
type SysDictCache struct{ *BasicCache[model.SysDict] }
type SysTableCache struct{ *BasicCache[model.SysTable] }
type SysTableFieldCache struct {
	*BasicCache[model.SysTableField]
}
type SysUserCache struct{ *BasicCache[model.SysUser] }

func NewSysConfigureCache(cacher Cacher) *SysConfigureCache {
	return &SysConfigureCache{NewBasicCache[model.SysConfigure](cacher, ConfigureCacheKey)}
}

func NewSysDictCache(cacher Cacher) *SysDictCache {
	return &SysDictCache{NewBasicCache[model.SysDict](cacher, DictCacheKey)}
}

func NewSysTableCache(cacher Cacher) *SysTableCache {
	return &SysTableCache{NewBasicCache[model.SysTable](cacher, TableCacheKey)}
}

func NewSysTableFieldCache(cacher Cacher) *SysTableFieldCache {
	return &SysTableFieldCache{NewBasicCache[model.SysTableField](cacher, TableFieldCacheKey)}
}

func NewSysUserCache(cacher Cacher) *SysUserCache {
	return &SysUserCache{NewBasicCache[model.SysUser](cacher, UserCacheKey)}
}
