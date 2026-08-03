/**
 * @Author: Nan
 * @Date: 2024/10/23 17:08
 */

package cache

import (
	"backend/enum"
	"go.uber.org/zap"
)

type TokenBlackCache struct {
	*BasicCache[string]
}

const TokenBlackCacheKey = "TOKEN_BLACK_"

const RefreshTokenBlackCacheKey = "REFRESH_TOKEN_BLACK_"

func NewTokenBlackCache(cacher Cacher) *TokenBlackCache {
	return &TokenBlackCache{
		BasicCache: NewBasicCache[string](cacher, ""),
	}
}

func (t *TokenBlackCache) Exists(key string) bool {
	exists, _ := t.cacher.Exists(TokenBlackCacheKey + key)
	if exists > 0 {
		return true
	}
	exists, _ = t.cacher.Exists(RefreshTokenBlackCacheKey + key)
	return exists > 0
}

func (t *TokenBlackCache) Set(tokenType enum.TokenTypeEnum, token string) (bool, error) {
	if tokenType == enum.AccessToken {
		err := t.cacher.Set(TokenBlackCacheKey+token, true, 7200)
		if err != nil {
			zap.L().Error(TokenBlackCacheKey+"Error TokenBlackCache setting token in cache", zap.String("token_type", "access_token"), zap.Error(err))
			return false, err
		}
		return true, nil
	}
	if tokenType == enum.RefreshToken {
		err := t.cacher.Set(RefreshTokenBlackCacheKey+token, true, 3600*24*7)
		if err != nil {
			zap.L().Error(RefreshTokenBlackCacheKey+"Error TokenBlackCache setting token in cache", zap.String("token_type", "refresh_token"), zap.Error(err))
			return false, err
		}
	}
	return true, nil
}
