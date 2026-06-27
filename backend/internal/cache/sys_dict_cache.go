/**
 * @Author: Nan
 * @Date: 2024/6/3 下午5:41
 */

package cache

import (
	"backend/model"
)

type SysDictCache struct {
	*BasicCache[model.SysDict]
}

const DictCacheKey = "DICT_CACHE_KEY_"

func NewSysDictCache(cacher Cacher) *SysDictCache {
	return &SysDictCache{BasicCache: NewBasicCache[model.SysDict](cacher, DictCacheKey)}
}
