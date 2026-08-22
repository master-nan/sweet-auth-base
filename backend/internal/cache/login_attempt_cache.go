package cache

import (
	"errors"
	"fmt"
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

type atomicLoginAttemptCacher interface {
	RecordLoginFailure(attemptKey, lockKey string, maxAttempts int, ttl time.Duration) (bool, error)
	CompleteLoginSuccess(attemptKey, lockKey string) (bool, error)
}

// CompleteSuccess 原子拒绝与账号锁定并发发生的登录成功；未锁定时只清理失败计数。
func (c *LoginAttemptCache) CompleteSuccess(principal string) (bool, error) {
	key := normalizeLoginPrincipal(principal)
	if key == "" {
		return false, nil
	}
	atomic, ok := c.cacher.(atomicLoginAttemptCacher)
	if !ok {
		return false, ErrAtomicCacheRequired
	}
	return atomic.CompleteLoginSuccess(LoginAttemptCacheKey+key, LoginLockCacheKey+key)
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

	atomic, ok := c.cacher.(atomicLoginAttemptCacher)
	if !ok {
		return false, ErrAtomicCacheRequired
	}
	return atomic.RecordLoginFailure(
		LoginAttemptCacheKey+key,
		LoginLockCacheKey+key,
		maxAttempts,
		lockFor,
	)
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

func UserLoginPrincipal(userID int) string {
	return fmt.Sprintf("user:%d", userID)
}
