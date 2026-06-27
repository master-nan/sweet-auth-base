/**
 * @Author: Nan
 * @Date: 2024/10/23 21:57
 */

package cache

import (
	"backend/model"
)

type ApplicationCache struct {
	*BasicCache[model.Application]
}

const ApplicationCacheKey = "APPLICATION_CACHE_KEY_"

func NewApplicationCache(cacher Cacher) *ApplicationCache {
	return &ApplicationCache{
		BasicCache: NewBasicCache[model.Application](cacher, ApplicationCacheKey),
	}
}
