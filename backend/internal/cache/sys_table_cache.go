/**
 * @Author: Nan
 * @Date: 2024/6/12 上午9:46
 */

package cache

import (
	"backend/model"
)

type SysTableCache struct {
	*BasicCache[model.SysTable]
}

const TableCacheKey = "TABLE_CACHE_KEY_"

func NewSysTableCache(cacher Cacher) *SysTableCache {
	return &SysTableCache{
		BasicCache: NewBasicCache[model.SysTable](cacher, TableCacheKey),
	}
}
