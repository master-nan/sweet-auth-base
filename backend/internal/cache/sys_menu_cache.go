/**
 * @Author: Nan
 * @Date: 2024/7/25 下午10:48
 */

package cache

import (
	"backend/model"
)

type SysMenuCache struct {
	*BasicCache[model.SysMenu]
}

const MenuCacheKey = "MENU_CACHE_KEY_"

func NewSysMenuCache(cacher Cacher) *SysMenuCache {
	return &SysMenuCache{
		BasicCache: NewBasicCache[model.SysMenu](cacher, MenuCacheKey),
	}
}
