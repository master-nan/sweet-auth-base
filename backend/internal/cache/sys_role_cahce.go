/**
 * @Author: Nan
 * @Date: 2024/7/25 下午10:49
 */

package cache

import (
	"backend/model"
)

type SysRoleCache struct {
	*BasicCache[model.SysRole]
}

const RoleCacheKey = "ROLE_CACHE_KEY_"

func NewSysRoleCache(cacher Cacher) *SysRoleCache {
	return &SysRoleCache{
		BasicCache: NewBasicCache[model.SysRole](cacher, RoleCacheKey),
	}
}
