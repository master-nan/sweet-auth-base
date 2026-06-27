/**
 * @Author: Nan
 * @Date: 2024/5/21 下午2:22
 */

package cache

import (
	"backend/model"
)

type SysConfigureCache struct {
	*BasicCache[model.SysConfigure]
}

const ConfigureCacheKey = "CONFIGURE_CACHE_KEY_"

func NewSysConfigureCache(cacher Cacher) *SysConfigureCache {
	return &SysConfigureCache{
		BasicCache: NewBasicCache[model.SysConfigure](cacher, ConfigureCacheKey),
	}
}
