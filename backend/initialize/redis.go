/**
 * @Author: Nan
 * @Date: 2024/10/11 12:00
 */

package initialize

import (
	"backend/config"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

func InitRedis(serverConfig *config.Server, zapLogger *zap.Logger) (*redis.Client, error) {
	cfg := serverConfig.Redis
	client := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxIdleTime: time.Duration(cfg.ConnMaxIdleTime) * time.Second,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		zap.L().Error("Failed to connect to Redis: %v", zap.Error(err))
		return nil, err
	}
	return client, nil
}
