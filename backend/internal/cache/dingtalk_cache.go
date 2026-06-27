/**
 * @Author: Nan
 * @Date: 2024/11/12 16:13
 */

package cache

import (
	"backend/internal/dingtalk"
)

type DingTalkCache struct {
	*BasicCache[dingtalk.AccessToken]
}

const DingTalkCacheKey = "DING_TALK_CACHE_KEY_"

func NewDingTalkCache(cacher Cacher) *DingTalkCache {
	return &DingTalkCache{BasicCache: NewBasicCache[dingtalk.AccessToken](cacher, DingTalkCacheKey)}
}
