/**
 * @Author: Nan
 * @Date: 2024/6/29 下午5:13
 */

package cache

import (
	"go.uber.org/zap"
	"time"
)

type GeneralizationCache struct {
	cacher Cacher
}

const GeneralizationCacheKey = "GENERALIZATION_CACHE_KEY_"

func NewGeneralizationCache(cacher Cacher) *GeneralizationCache {
	return &GeneralizationCache{cacher}
}

func (c *GeneralizationCache) Get(tableCode string, key string, data interface{}) (interface{}, error) {
	err := c.cacher.Get(GeneralizationCacheKey+tableCode+key, &data)
	if err != nil {
		zap.L().Error(GeneralizationCacheKey+"Error getting key in cache", zap.String("key", key), zap.Error(err))
		return data, err
	}
	return data, nil
}

func (c *GeneralizationCache) Set(tableCode string, key string, data interface{}) error {
	err := c.cacher.Set(GeneralizationCacheKey+tableCode+key, &data, 7200*time.Second)
	if err != nil {
		zap.L().Error(GeneralizationCacheKey+"Error setting key in cache", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

func (c *GeneralizationCache) Delete(tableCode string, key string) error {
	err := c.cacher.Del(GeneralizationCacheKey + tableCode + key)
	if err != nil {
		zap.L().Error(GeneralizationCacheKey+"Error delete key in cache", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}
