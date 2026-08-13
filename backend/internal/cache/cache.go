/**
 * @Author: Nan
 * @Date: 2024/5/23 下午2:33
 */

package cache

import (
	"github.com/pkg/errors"
	"time"
)

var (
	ErrCacheMiss           = errors.New("cache miss")
	ErrAtomicCacheRequired = errors.New("atomic cache capability is required")
)

type Cacher interface {
	Get(key string, value interface{}) error
	Set(key string, value interface{}, expiration time.Duration) error
	Del(key string) error
	Exists(keys ...string) (int64, error)
	Expire(key string, expiration time.Duration) (bool, error)
}
