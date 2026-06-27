/**
 * @Author: Nan
 * @Date: 2024/6/28 下午3:00
 */

package cache

import (
	"backend/model"
)

type SysUserCache struct {
	*BasicCache[model.SysUser]
}

const UserCacheKey = "USER_CACHE_KEY_"

func NewSysUserCache(cacher Cacher) *SysUserCache {
	return &SysUserCache{
		BasicCache: NewBasicCache[model.SysUser](cacher, UserCacheKey),
	}
}
