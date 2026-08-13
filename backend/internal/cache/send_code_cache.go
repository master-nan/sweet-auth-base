/**
 * @Author: Nan
 * @Date: 2025/2/8 15:03
 */

package cache

import (
	"crypto/sha256"
	"fmt"
	"time"
)

type SendCodeCache struct {
	*BasicCache[string]
}

const SendCodeCacheKey = "SEND_CODE_CACHE_KEY_" // 短信验证码缓存key

const maxSMSVerificationAttempts = 5

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

type atomicCodeCacher interface {
	ConsumeCode(key, attemptKey, expected string, maxAttempts int, ttl time.Duration) (int64, error)
}

// Consume verifies and deletes a code in one cache operation.
func (s *SendCodeCache) Consume(key, expected string) (bool, error) {
	atomic, ok := s.cacher.(atomicCodeCacher)
	if !ok {
		return false, ErrAtomicCacheRequired
	}
	status, err := atomic.ConsumeCode(
		SendCodeCacheKey+key,
		SendCodeCacheKey+"ATTEMPT_"+key,
		expected,
		maxSMSVerificationAttempts,
		10*time.Minute,
	)
	return status == 1, err
}

func SMSVerificationKey(applicationID int, mobile string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", applicationID, mobile)))
	return fmt.Sprintf("SMS_VERIFY_%x", sum)
}
