/**
 * @Author: Nan
 * @Date: 2025/2/8 14:52
 */

package cache

import (
	"backend/model"
)

type SmsLogCache struct {
	*BasicCache[model.SmsLog]
}

const SmsLogCacheKey = "SMS_LOG_CACHE_KEY_" // 短信发送记录缓存key

func NewSmsLogCache(cacher Cacher) *SmsLogCache {
	return &SmsLogCache{BasicCache: NewBasicCache[model.SmsLog](cacher, SmsLogCacheKey)}
}
