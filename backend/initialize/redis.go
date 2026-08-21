/**
 * @Author: Nan
 * @Date: 2024/10/11 12:00
 */

package initialize

import (
	"backend/config"
	"backend/internal/cache"
	"context"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

func InitRedis(serverConfig *config.Server, zapLogger *zap.Logger) (*redis.Client, error) {
	cfg := serverConfig.Redis
	options, err := cache.RedisOptions(cfg)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(options)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		_ = client.Close()
		zap.L().Error("failed to connect to Redis", zap.Error(err))
		return nil, err
	}
	zap.L().Info("Redis connection initialized", zap.Bool("tls_enabled", cfg.TLS.Enabled))
	return client, nil
}
