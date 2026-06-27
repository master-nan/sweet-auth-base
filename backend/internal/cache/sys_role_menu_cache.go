/**
 * @Author: Nan
 * @Date: 2024/7/25 下午10:48
 */

package cache

import (
	"backend/model"
)

type SysRoleMenuCache struct {
	*BasicCache[model.SysRoleMenu]
}

const RoleMenuCacheKey = "ROLE_MENU_CACHE_KEY_"

func NewSysRoleMenuCache(cacher Cacher) *SysRoleMenuCache {
	return &SysRoleMenuCache{
		BasicCache: NewBasicCache[model.SysRoleMenu](cacher, RoleMenuCacheKey),
	}
}
