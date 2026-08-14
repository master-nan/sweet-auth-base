/**
 * @Author: Nan
 * @Date: 2024/10/11 11:26
 */

package main

import (
	"backend/initialize"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// @title Sweet Admin
// @version 0.1
// @description 基于 Gin、Gorm 和 Quasar 的低代码管理底座
// @BasePath  /sweet_admin

func main() {
	app, err := initialize.InitializeApp()
	if err != nil {
		zap.L().Fatal("failed to initialize application", zap.Error(err))
	}
	if err := app.IntegrationWorker.Start(context.Background()); err != nil {
		zap.L().Fatal("failed to start integration worker", zap.Error(err))
	}
	if err := app.IntegrationSyncRunner.Start(context.Background()); err != nil {
		_ = app.IntegrationWorker.Stop(context.Background())
		zap.L().Fatal("failed to start integration sync runner", zap.Error(err))
	}
	initialize.InitCron(app)
	router := initialize.InitRouter(app)
	port := app.Config.Port
	// 使用一个独立的 goroutine 启动服务器
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 10 * 1024 * 1024,
	}
	zap.L().Info("Starting server on port", zap.Int("port", port))
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Fatal("listen: ", zap.Error(err))
		}
	}()
	// 创建一个通道，用于接收退出信号
	quit := make(chan os.Signal, 1)
	// 接收 SIGINT 和 SIGTERM 信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞，直到接收到信号
	zap.L().Info("Shutting down server...")
	if err := app.IntegrationSyncRunner.Stop(context.Background()); err != nil {
		zap.L().Warn("integration sync runner did not stop cleanly", zap.Error(err))
	}
	if err := app.IntegrationWorker.Stop(context.Background()); err != nil {
		zap.L().Warn("integration worker did not stop cleanly", zap.Error(err))
	}
	// 创建一个5秒的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 优雅关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server forced to shutdown:", zap.Error(err))
	}
	zap.L().Info("Server exiting")
	// 关闭日志
	initialize.CloseLogger()

}
