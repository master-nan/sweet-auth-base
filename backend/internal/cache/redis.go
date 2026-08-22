/**
 * @Author: Nan
 * @Date: 2024/5/21 下午2:26
 */

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisUtil struct {
	client *redis.Client
}

func NewRedisUtil(client *redis.Client) *RedisUtil {
	return &RedisUtil{
		client: client,
	}
}

func withTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 200*duration)
}

func (r *RedisUtil) Set(key string, value interface{}, expiration time.Duration) error {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	case int:
		str = strconv.FormatInt(int64(v), 10)
	case *string:
		str = *v
	case *int:
		str = strconv.FormatInt(int64(*v), 10)
	case *float64:
		str = strconv.FormatFloat(*v, 'f', -1, 64)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			zap.L().Error("json marshalling failed",
				zap.String("value_type", fmt.Sprintf("%T", value)),
				zap.Error(err))
			return err
		}
		str = string(b)
	}
	err := r.client.Set(ctx, key, str, expiration).Err()
	if err != nil {
		zap.L().Error("failed to set cache key",
			zap.String("key_hash", cacheKeyDigest(key)),
			zap.Error(err))
		return err
	}
	return nil
}

func (r *RedisUtil) Get(key string, value interface{}) error {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		zap.L().Error("failed to get cache key",
			zap.String("key_hash", cacheKeyDigest(key)),
			zap.Error(err))
		return err
	}
	switch v := value.(type) {
	case *string:
		*v = string(val)
	case *int:
		iv, err := strconv.Atoi(string(val))
		if err != nil {
			zap.L().Error("failed to convert cached string to int",
				zap.String("key_hash", cacheKeyDigest(key)),
				zap.Error(err))
			return err
		}
		*v = iv
	case *float64:
		fv, err := strconv.ParseFloat(string(val), 64)
		if err != nil {
			zap.L().Error("failed to convert cached string to float64",
				zap.String("key_hash", cacheKeyDigest(key)),
				zap.Error(err))
			return err
		}
		*v = fv
	default:
		err := json.Unmarshal(val, value)
		if err != nil {
			zap.L().Error("failed to unmarshal cached value",
				zap.String("key_hash", cacheKeyDigest(key)),
				zap.Error(err))
			return err
		}
	}
	return nil
}

func (r *RedisUtil) Del(key string) error {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		zap.L().Error("failed to delete cache key",
			zap.String("key_hash", cacheKeyDigest(key)),
			zap.Error(err))
		return err
	}
	return nil
}

func (r *RedisUtil) Exists(keys ...string) (int64, error) {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	val, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to check whether cache keys exist: %w", err)
	}
	return val, nil
}

func (r *RedisUtil) Expire(key string, expiration time.Duration) (bool, error) {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	val, err := r.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set cache expiration: %w", err)
	}
	return val, nil
}

// SetIfAbsent 原子占用一次性安全状态的缓存键。
func (r *RedisUtil) SetIfAbsent(key string, value interface{}, expiration time.Duration) (bool, error) {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// ConsumeCode 原子校验并消费一次性验证码，同时限制猜测次数。
func (r *RedisUtil) ConsumeCode(key, attemptKey, expected string, maxAttempts int, ttl time.Duration) (int64, error) {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	return r.client.Eval(ctx, `
		local code = redis.call("GET", KEYS[1])
		if not code then
			return 0
		end
		if code == ARGV[1] then
			redis.call("DEL", KEYS[1], KEYS[2])
			return 1
		end
		local attempts = redis.call("INCR", KEYS[2])
		if attempts == 1 then
			redis.call("PEXPIRE", KEYS[2], ARGV[3])
		end
		if attempts >= tonumber(ARGV[2]) then
			redis.call("DEL", KEYS[1], KEYS[2])
		end
		return -1
	`, []string{key, attemptKey}, expected, maxAttempts, ttl.Milliseconds()).Int64()
}

// RecordLoginFailure 原子增加失败计数，并在配置阈值处创建锁定状态。
func (r *RedisUtil) RecordLoginFailure(attemptKey, lockKey string, maxAttempts int, ttl time.Duration) (bool, error) {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	result, err := r.client.Eval(ctx, `
		local attempts = redis.call("INCR", KEYS[1])
		if attempts == 1 then
			redis.call("PEXPIRE", KEYS[1], ARGV[2])
		end
		if attempts >= tonumber(ARGV[1]) then
			redis.call("SET", KEYS[2], "1", "PX", ARGV[2])
			redis.call("DEL", KEYS[1])
			return 1
		end
		return 0
	`, []string{attemptKey, lockKey}, maxAttempts, ttl.Milliseconds()).Int64()
	return result == 1, err
}

// CompleteLoginSuccess 关闭密码校验成功与并发创建锁定状态之间的竞态窗口。
func (r *RedisUtil) CompleteLoginSuccess(attemptKey, lockKey string) (bool, error) {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	result, err := r.client.Eval(ctx, `
		if redis.call("EXISTS", KEYS[2]) == 1 then
			return 0
		end
		redis.call("DEL", KEYS[1])
		return 1
	`, []string{attemptKey, lockKey}).Int64()
	return result == 1, err
}

func (r *RedisUtil) HSet(key, field string, value interface{}) error {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	err := r.client.HSet(ctx, key, field, value).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache hash: %w", err)
	}
	return nil
}

func (r *RedisUtil) HGet(key, field string) (string, error) {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	val, err := r.client.HGet(ctx, key, field).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get cache hash: %w", err)
	}
	return val, nil
}

func (r *RedisUtil) HDel(key string, fields ...string) (int64, error) {
	ctx, cancel := withTimeout(2 * time.Second)
	defer cancel()
	val, err := r.client.HDel(ctx, key, fields...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to delete cache hash: %w", err)
	}
	return val, nil
}
