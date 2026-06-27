/**
 * @Author: Nan
 * @Date: 2025/2/8 15:03
 */

package cache

type SendCodeCache struct {
	*BasicCache[string]
}

const SendCodeCacheKey = "SEND_CODE_CACHE_KEY_" // 短信验证码缓存key

func NewSendCodeCache(cacher Cacher) *SendCodeCache {
	return &SendCodeCache{BasicCache: NewBasicCache[string](cacher, SendCodeCacheKey)}
}

// Set 传入短信验证码缓存短信验证码10分钟
func (s *SendCodeCache) Set(key string, code string) error {
	err := s.SetExpiration(key, code, 60*10) // 缓存时间为10分钟
	if err != nil {
		return err
	}
	return nil
}
