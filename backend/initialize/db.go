/**
 * @Author: Nan
 * @Date: 2024/10/11 11:58
 */

package initialize

import (
	"backend/config"
	"backend/internal/database"
	"backend/model"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// CustomGormLogger 自定义的 GORM 日志记录器
type CustomGormLogger struct {
	logger *zap.Logger
}

func (c *CustomGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return c
}

func (c *CustomGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	c.logger.Sugar().Infof(msg, data...)
}

func (c *CustomGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	c.logger.Sugar().Warnf(msg, data...)
}

func (c *CustomGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	c.logger.Sugar().Errorf(msg, data...)
}

func (c *CustomGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.logger.Sugar().Errorf("%s [%.2fms] [rows:%v] %s", err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		} else {
			c.logger.Sugar().Infof("[%.2fms] [rows:%v] %s", float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	} else {
		c.logger.Sugar().Infof("[%.2fms] [rows:%v] %s", float64(elapsed.Nanoseconds())/1e6, rows, sql)
	}
}

func InitDB(zapLogger *zap.Logger, serverConfig *config.Server) (map[string]*gorm.DB, error) {
	// 配置 GORM 日志记录器
	gormLogger := &CustomGormLogger{
		logger: zapLogger,
	}
	// 配置多数据库
	dbs := make(map[string]*gorm.DB)
	initialized := false
	defer func() {
		if initialized {
			return
		}
		for _, db := range dbs {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	}()
	v := reflect.ValueOf(serverConfig.DBS)
	t := reflect.TypeOf(serverConfig.DBS)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i).Interface().(config.DB)
		name := t.Field(i).Tag.Get("mapstructure")
		dsn, err := database.PostgresDSN(field)
		if err != nil {
			return nil, fmt.Errorf("database %s configuration: %w", name, err)
		}
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   field.Prefix,
				SingularTable: true,
			},
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   gormLogger,
			NowFunc:                                  model.Now,
		})
		if err != nil {
			zap.L().Error("failed to open database connection", zap.Error(err))
			return nil, err
		}
		sqlDB, err := db.DB()
		if err != nil {
			zap.L().Error("failed to get database connection", zap.Error(err))
			return nil, err
		}
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
		dbs[name] = db
		zap.L().Info("database connection initialized",
			zap.String("database_role", name),
			zap.String("tls_mode", field.TLS.Mode),
		)
	}
	initialized = true
	return dbs, nil
}
