/**
 * @Author: Nan
 * @Date: 2024/7/25 下午10:49
 */

package cache

import (
	"backend/model"
)

type SysRoleMenuButtonCache struct {
	*BasicCache[model.SysRoleMenuButton]
}

const RoleMenuButtonCacheKey = "ROLE_MENU_BUTTON_CACHE_KEY_"

func NewSysRoleMenuButtonCache(cacher Cacher) *SysRoleMenuButtonCache {
	return &SysRoleMenuButtonCache{
		BasicCache: NewBasicCache[model.SysRoleMenuButton](cacher, RoleMenuButtonCacheKey),
	}
}
