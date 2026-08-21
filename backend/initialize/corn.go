/**
 * @Author: Nan
 * @Date: 2024/10/18 15:02
 */

package initialize

import (
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var c *cron.Cron

// InitCron 初始化定时任务。
func InitCron(app *App) {
	c = cron.New(cron.WithSeconds())
	c.Start()

	if app.Config.Conf.Enable {
		zap.L().Warn("通用底座暂未配置业务定时任务")
	} else {
		zap.L().Warn("定时任务未开启")
	}
}

// StopCron stops scheduling new jobs and waits for active jobs within the
// caller's shutdown budget.
func StopCron(ctx context.Context) error {
	if c == nil {
		return nil
	}
	done := c.Stop()
	select {
	case <-done.Done():
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop cron: %w", ctx.Err())
	}
}

func AddCronJob(spec string, cmd func()) {
	_, err := c.AddFunc(spec, cmd)
	if err != nil {
		zap.L().Fatal("Error adding cron job: ", zap.Error(err))
	}
}
