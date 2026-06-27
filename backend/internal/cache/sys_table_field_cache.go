/**
 * @Author: Nan
 * @Date: 2024/6/12 上午9:46
 */

package cache

import (
	"backend/model"
)

type SysTableFieldCache struct {
	*BasicCache[model.SysTableField]
}

const TableFieldCacheKey = "TABLE_FIELD_CACHE_KEY_"

func NewSysTableFieldCache(cacher Cacher) *SysTableFieldCache {
	return &SysTableFieldCache{
		BasicCache: NewBasicCache[model.SysTableField](cacher, TableFieldCacheKey),
	}
}
