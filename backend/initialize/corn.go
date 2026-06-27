/**
 * @Author: Nan
 * @Date: 2024/10/18 15:02
 */

package initialize

import (
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var c *cron.Cron

// InitCron initializes the cron jobs
func InitCron(app *App) {
	c = cron.New(cron.WithSeconds())
	c.Start()

	if app.Config.Conf.Enable {
		zap.L().Warn("通用底座暂未配置业务定时任务")
	} else {
		zap.L().Warn("定时任务未开启")
	}
}

func AddCronJob(spec string, cmd func()) {
	_, err := c.AddFunc(spec, cmd)
	if err != nil {
		zap.L().Fatal("Error adding cron job: ", zap.Error(err))
	}
}
