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

type DingTalkUserIDCache struct {
	*BasicCache[string]
}

const DingTalkUserIDCacheKey = "DING_TALK_USER_ID_CACHE_KEY_"

func NewDingTalkUserIDCache(cacher Cacher) *DingTalkUserIDCache {
	return &DingTalkUserIDCache{BasicCache: NewBasicCache[string](cacher, DingTalkUserIDCacheKey)}
}

// Set缓存DingTalk用户ID七天，避免每次认证重复调用远端接口。
func (d *DingTalkUserIDCache) Set(key string, code string) error {
	return d.SetExpiration(key, code, 60*60*24*7)
}
