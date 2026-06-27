package cache

import (
	"errors"
	"strings"
	"time"
)

const (
	LoginAttemptCacheKey      = "LOGIN_ATTEMPT_"
	LoginLockCacheKey         = "LOGIN_LOCK_"
	defaultMaxLoginAttempts   = 5
	defaultLoginAttemptLockIn = 15 * time.Minute
)

type LoginAttemptCache struct {
	cacher Cacher
}

func NewLoginAttemptCache(cacher Cacher) *LoginAttemptCache {
	return &LoginAttemptCache{cacher: cacher}
}

func (c *LoginAttemptCache) IsLocked(principal string) (bool, error) {
	exists, err := c.cacher.Exists(LoginLockCacheKey + normalizeLoginPrincipal(principal))
	return exists > 0, err
}

func (c *LoginAttemptCache) RecordFailure(principal string, maxAttempts int, lockFor time.Duration) (bool, error) {
	key := normalizeLoginPrincipal(principal)
	if key == "" {
		return false, nil
	}
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxLoginAttempts
	}
	if lockFor <= 0 {
		lockFor = defaultLoginAttemptLockIn
	}

	attemptKey := LoginAttemptCacheKey + key
	var attempts int
	if err := c.cacher.Get(attemptKey, &attempts); err != nil && !errors.Is(err, ErrCacheMiss) {
		return false, err
	}
	attempts++
	if attempts >= maxAttempts {
		if err := c.cacher.Set(LoginLockCacheKey+key, 1, lockFor); err != nil {
			return false, err
		}
		_ = c.cacher.Del(attemptKey)
		return true, nil
	}
	if err := c.cacher.Set(attemptKey, attempts, lockFor); err != nil {
		return false, err
	}
	return false, nil
}

func (c *LoginAttemptCache) Clear(principal string) error {
	key := normalizeLoginPrincipal(principal)
	if key == "" {
		return nil
	}
	if err := c.cacher.Del(LoginAttemptCacheKey + key); err != nil && !errors.Is(err, ErrCacheMiss) {
		return err
	}
	if err := c.cacher.Del(LoginLockCacheKey + key); err != nil && !errors.Is(err, ErrCacheMiss) {
		return err
	}
	return nil
}

func normalizeLoginPrincipal(principal string) string {
	return strings.ToLower(strings.TrimSpace(principal))
}
