/**
 * @Author: Nan
 * @Date: 2025/2/14 16:33
 */

package cache

type DingTalkUserIDCache struct {
	*BasicCache[string]
}

const DingTalkUserIDCacheKey = "DING_TALK_USER_ID_CACHE_KEY_" // 钉钉用户UserId 缓存key

func NewDingTalkUserIDCache(cacher Cacher) *DingTalkUserIDCache {
	return &DingTalkUserIDCache{BasicCache: NewBasicCache[string](cacher, DingTalkUserIDCacheKey)}
}

// Set 传入钉钉用户UserID缓存7天
func (d *DingTalkUserIDCache) Set(key string, code string) error {
	// 7天缓存时间
	err := d.SetExpiration(key, code, 60*60*24*7)
	if err != nil {
		return err
	}
	return nil
}
