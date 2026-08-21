/**
 * @Author: Nan
 * @Date: 2024/10/23 17:08
 */

package cache

import (
	"backend/enum"
	"crypto/sha256"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type TokenBlackCache struct {
	*BasicCache[string]
}

const TokenBlackCacheKey = "TOKEN_BLACK_"

const RefreshTokenBlackCacheKey = "REFRESH_TOKEN_BLACK_"

const UserTokenSessionCacheKey = "USER_TOKEN_SESSION_"

type atomicTokenCacher interface {
	SetIfAbsent(key string, value interface{}, expiration time.Duration) (bool, error)
}

func NewTokenBlackCache(cacher Cacher) *TokenBlackCache {
	return &TokenBlackCache{
		BasicCache: NewBasicCache[string](cacher, ""),
	}
}

func (t *TokenBlackCache) IsRevoked(tokenType enum.TokenTypeEnum, value string) (bool, error) {
	exists, err := t.cacher.Exists(tokenCacheKey(tokenType, value))
	return exists > 0, err
}

func (t *TokenBlackCache) Revoke(tokenType enum.TokenTypeEnum, value string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Second
	}
	return t.setRevoked(tokenType, value, ttl)
}

func (t *TokenBlackCache) ConsumeRefresh(value string, expiresAt time.Time) (bool, error) {
	atomic, ok := t.cacher.(atomicTokenCacher)
	if !ok {
		return false, ErrAtomicCacheRequired
	}
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return false, nil
	}
	return atomic.SetIfAbsent(tokenCacheKey(enum.RefreshToken, value), true, ttl)
}

func (t *TokenBlackCache) ActivateSession(userID int, sessionID string, ttl time.Duration) error {
	if userID <= 0 || sessionID == "" {
		return ErrCacheMiss
	}
	return t.cacher.Set(userSessionCacheKey(userID, sessionID), true, ttl)
}

func (t *TokenBlackCache) TouchSession(userID int, sessionID string, ttl time.Duration) (bool, error) {
	exists, err := t.cacher.Exists(userSessionCacheKey(userID, sessionID))
	if err != nil || exists == 0 {
		return false, err
	}
	extended, err := t.cacher.Expire(userSessionCacheKey(userID, sessionID), ttl)
	if err != nil {
		return false, err
	}
	return extended, nil
}

func (t *TokenBlackCache) IsSessionActive(userID int, sessionID string) (bool, error) {
	exists, err := t.cacher.Exists(userSessionCacheKey(userID, sessionID))
	return exists > 0, err
}

func (t *TokenBlackCache) DeactivateSession(userID int, sessionID string) error {
	err := t.cacher.Del(userSessionCacheKey(userID, sessionID))
	if errors.Is(err, ErrCacheMiss) {
		return nil
	}
	return err
}

func (t *TokenBlackCache) setRevoked(tokenType enum.TokenTypeEnum, value string, ttl time.Duration) error {
	err := t.cacher.Set(tokenCacheKey(tokenType, value), true, ttl)
	if err != nil {
		zap.L().Error("failed to revoke token", zap.String("token_type", string(tokenType)), zap.Error(err))
	}
	return err
}

func tokenCacheKey(tokenType enum.TokenTypeEnum, value string) string {
	prefix := TokenBlackCacheKey
	if tokenType == enum.RefreshToken {
		prefix = RefreshTokenBlackCacheKey
	}
	return prefix + tokenDigest(value)
}

func tokenDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum)
}

func userSessionCacheKey(userID int, sessionID string) string {
	return UserTokenSessionCacheKey + strconv.Itoa(userID) + ":" + tokenDigest(sessionID)
}
