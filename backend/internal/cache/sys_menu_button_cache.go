/**
 * @Author: Nan
 * @Date: 2024/7/25 下午11:12
 */

package cache

import (
	"backend/model"
)

type SysMenuButtonCache struct {
	*BasicCache[model.SysMenuButton]
}

const MenuButtonCacheKey = "MENU_BUTTON_CACHE_KEY_"

func NewSysMenuButtonCache(cacher Cacher) *SysMenuButtonCache {
	return &SysMenuButtonCache{
		BasicCache: NewBasicCache[model.SysMenuButton](cacher, MenuButtonCacheKey),
	}
}
