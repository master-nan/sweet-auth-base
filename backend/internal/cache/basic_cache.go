/**
 * @Author: Nan
 * @Date: 2024/7/26 下午3:14
 */

package cache

import (
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type BasicCache[T any] struct {
	cacher         Cacher
	cacheKeyPrefix string
}

func NewBasicCache[T any](cacher Cacher, cacheKeyPrefix string) *BasicCache[T] {
	return &BasicCache[T]{
		cacher,
		cacheKeyPrefix,
	}
}
func (c *BasicCache[T]) Get(key string) (T, error) {
	var data T
	err := c.cacher.Get(c.cacheKeyPrefix+key, &data)
	if err != nil {
		if !errors.Is(err, ErrCacheMiss) {
			zap.L().Error(c.cacheKeyPrefix+"Error getting key in cache", zap.String("key", key), zap.Error(err))
		}
		return data, err
	}
	return data, nil
}

func (c *BasicCache[T]) Set(key string, data T) error {
	err := c.cacher.Set(c.cacheKeyPrefix+key, &data, 7200*time.Second)
	if err != nil {
		zap.L().Error(c.cacheKeyPrefix+"Error setting key in cache", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

func (c *BasicCache[T]) SetExpiration(key string, data T, expiration int64) error {
	err := c.cacher.Set(c.cacheKeyPrefix+key, &data, time.Duration(expiration)*time.Second)
	if err != nil {
		zap.L().Error(c.cacheKeyPrefix+"Error setting key in cache", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

func (c *BasicCache[T]) Delete(key string) error {
	err := c.cacher.Del(c.cacheKeyPrefix + key)
	if err != nil {
		zap.L().Error(c.cacheKeyPrefix+"Error delete key in cache", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

func (c *BasicCache[T]) Exists(key string) bool {
	exists, _ := c.cacher.Exists(c.cacheKeyPrefix + key)
	return exists > 0
}
