/**
 * @Author: Nan
 * @Date: 2024/7/25 下午11:19
 */

package cache

import (
	"backend/model"
)

type SysUserRoleCache struct {
	*BasicCache[model.SysUserRole]
}

const UserRoleCacheKey = "USER_ROLE_CACHE_KEY_"

func NewSysUserRoleCache(cacher Cacher) *SysUserRoleCache {
	return &SysUserRoleCache{
		BasicCache: NewBasicCache[model.SysUserRole](cacher, UserRoleCacheKey),
	}
}
