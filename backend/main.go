/**
 * @Author: Nan
 * @Date: 2024/10/11 11:26
 */

package main

import (
	"backend/config"
	"backend/initialize"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// @title Sweet Admin
// @version 0.1
// @description 基于 Gin、Gorm 和 Quasar 的低代码管理底座
// @BasePath  /sweet_admin

const runtimeShutdownTimeout = 45 * time.Second

type backgroundRunner interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type expiredChunkCleaner interface {
	CleanupExpiredChunks(time.Time, time.Duration) (int, error)
}

type runtimeDependencies struct {
	worker         backgroundRunner
	syncRunner     backgroundRunner
	chunkCleaner   expiredChunkCleaner
	uploadConfig   config.Upload
	stopCron       func(context.Context) error
	closeResources func() error
	closeLogger    func()
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "sweet_admin stopped: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	app, err := initialize.InitializeApp()
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}

	initialize.InitCron(app)
	router := initialize.InitRouter(app)
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", app.Config.Port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 10 * 1024 * 1024,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		_ = initialize.StopCron(context.Background())
		_ = initialize.CloseRuntimeResources(app)
		initialize.CloseLogger()
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	}

	rootCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	return runRuntime(rootCtx, listener, server, runtimeDependencies{
		worker:         app.IntegrationWorker,
		syncRunner:     app.IntegrationSyncRunner,
		chunkCleaner:   app.FileUploadService,
		uploadConfig:   app.Config.Upload,
		stopCron:       initialize.StopCron,
		closeResources: func() error { return initialize.CloseRuntimeResources(app) },
		closeLogger:    initialize.CloseLogger,
	})
}

// runRuntime 统一管理HTTP、Runner、Cron和Chunk清理的生命周期；
// 只有在所有在途工作停止后才关闭Redis、DB和Logger。
func runRuntime(parent context.Context, listener net.Listener, server *http.Server, dependencies runtimeDependencies) error {
	runtimeCtx, cancelRuntime := context.WithCancel(parent)
	defer cancelRuntime()

	if err := dependencies.worker.Start(runtimeCtx); err != nil {
		closeRuntime(dependencies)
		return fmt.Errorf("start integration worker: %w", err)
	}
	zap.L().Info("integration worker start completed")
	if err := dependencies.syncRunner.Start(runtimeCtx); err != nil {
		_ = dependencies.worker.Stop(context.Background())
		closeRuntime(dependencies)
		return fmt.Errorf("start integration sync runner: %w", err)
	}
	zap.L().Info("integration sync runner start completed")
	chunkDone := maintainExpiredChunks(runtimeCtx, dependencies.chunkCleaner, dependencies.uploadConfig)

	zap.L().Info("HTTP server started", zap.String("address", listener.Addr().String()))
	serveError := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveError <- err
	}()

	var runtimeError error
	serveStopped := false
	select {
	case <-parent.Done():
		zap.L().Info("shutdown signal received")
	case err := <-serveError:
		serveStopped = true
		if err != nil {
			runtimeError = fmt.Errorf("serve HTTP: %w", err)
		}
	}

	var shutdownErrors []error
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), runtimeShutdownTimeout)
	defer cancelShutdown()
	httpShutdown := make(chan error, 1)
	go func() {
		httpShutdown <- server.Shutdown(shutdownCtx)
	}()
	// Shutdown 是 listener 的唯一关闭者。先等 Serve 返回，确认不再接收新请求，
	// 再停止后台任务；不要在这里调用 listener.Close，否则 Linux 可能因重复关闭报错。
	if !serveStopped {
		if err := <-serveError; err != nil {
			runtimeError = fmt.Errorf("serve HTTP: %w", err)
		}
	}
	cancelRuntime()

	if dependencies.stopCron != nil {
		if err := dependencies.stopCron(shutdownCtx); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}
	if err := dependencies.syncRunner.Stop(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("stop integration sync runner: %w", err))
	} else {
		zap.L().Info("integration sync runner stopped")
	}
	if err := dependencies.worker.Stop(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("stop integration worker: %w", err))
	} else {
		zap.L().Info("integration worker stopped")
	}
	if err := <-httpShutdown; err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("shutdown HTTP server: %w", err))
	}
	select {
	case <-chunkDone:
	case <-shutdownCtx.Done():
		shutdownErrors = append(shutdownErrors, fmt.Errorf("stop chunk cleanup: %w", shutdownCtx.Err()))
	}
	if dependencies.closeResources != nil {
		if err := dependencies.closeResources(); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}
	zap.L().Info("runtime shutdown completed")
	if dependencies.closeLogger != nil {
		dependencies.closeLogger()
	}
	return errors.Join(runtimeError, errors.Join(shutdownErrors...))
}

// maintainExpiredChunks 在启动时及固定周期清理废弃分片，并以返回通道参与Shutdown等待。
func maintainExpiredChunks(ctx context.Context, cleaner expiredChunkCleaner, upload config.Upload) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if cleaner == nil {
			return
		}
		ttl := time.Duration(upload.ChunkTTLHours) * time.Hour
		interval := time.Duration(upload.ChunkCleanupMinutes) * time.Minute
		if ttl <= 0 {
			ttl = 24 * time.Hour
		}
		if interval <= 0 {
			interval = time.Hour
		}
		cleanup := func() {
			removed, err := cleaner.CleanupExpiredChunks(time.Now(), ttl)
			if err != nil {
				zap.L().Warn("chunk staging cleanup failed", zap.Error(err))
				return
			}
			zap.L().Info("chunk staging cleanup completed", zap.Int("removed_sessions", removed))
		}
		cleanup()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup()
			}
		}
	}()
	return done
}

func closeRuntime(dependencies runtimeDependencies) {
	if dependencies.stopCron != nil {
		_ = dependencies.stopCron(context.Background())
	}
	if dependencies.closeResources != nil {
		_ = dependencies.closeResources()
	}
	if dependencies.closeLogger != nil {
		dependencies.closeLogger()
	}
}
