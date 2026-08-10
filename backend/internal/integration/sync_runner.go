package integration

import (
	myerrors "backend/internal/errors"
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultSyncPollInterval    = 10 * time.Second
	defaultSyncScheduleLimit   = 8
	defaultSyncCoordinateLimit = 16
	defaultSyncShutdownTimeout = 15 * time.Second
	minSyncPollInterval        = time.Second
	maxSyncPollInterval        = 5 * time.Minute
	maxSyncBatchLimit          = 64
	maxSyncShutdownTimeout     = time.Minute
)

type SyncRuntime interface {
	RunOnce(context.Context, int, int) (SyncRuntimeSummary, error)
}

type SyncRuntimeSummary struct {
	ScheduledBatches int
	Coordinated      int
	Succeeded        int
	Failed           int
}

type SyncRunnerConfig struct {
	Enabled             bool
	RunnerID            string
	PollInterval        time.Duration
	ScheduleBatchSize   int
	CoordinateBatchSize int
	ShutdownTimeout     time.Duration
}

type SyncRunnerStatus struct {
	Enabled           bool
	Running           bool
	RunnerID          string
	StartedAt         time.Time
	LastPollAt        time.Time
	LastSuccessAt     time.Time
	LastErrorCategory string
	ScheduledTotal    int64
	CoordinatedTotal  int64
	SucceededTotal    int64
	FailedTotal       int64
}

type IntegrationSyncRunner struct {
	runtime SyncRuntime
	config  SyncRunnerConfig

	mu     sync.RWMutex
	status SyncRunnerStatus
	cancel context.CancelFunc
	done   chan struct{}
}

func NewSyncRunnerConfig(config SyncRunnerConfig) (SyncRunnerConfig, error) {
	config.RunnerID = strings.TrimSpace(config.RunnerID)
	if config.PollInterval == 0 {
		config.PollInterval = defaultSyncPollInterval
	}
	if config.ScheduleBatchSize == 0 {
		config.ScheduleBatchSize = defaultSyncScheduleLimit
	}
	if config.CoordinateBatchSize == 0 {
		config.CoordinateBatchSize = defaultSyncCoordinateLimit
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = defaultSyncShutdownTimeout
	}
	if config.PollInterval < minSyncPollInterval || config.PollInterval > maxSyncPollInterval ||
		config.ScheduleBatchSize < 1 || config.ScheduleBatchSize > maxSyncBatchLimit ||
		config.CoordinateBatchSize < 1 || config.CoordinateBatchSize > maxSyncBatchLimit ||
		config.ShutdownTimeout < time.Second || config.ShutdownTimeout > maxSyncShutdownTimeout ||
		(config.Enabled && (config.RunnerID == "" || len(config.RunnerID) > 128)) {
		return SyncRunnerConfig{}, myerrors.ErrSyncRunnerInvalidConfig
	}
	if config.RunnerID == "" {
		config.RunnerID = "sync-disabled"
	}
	return config, nil
}

func NewIntegrationSyncRunner(runtime SyncRuntime, config SyncRunnerConfig) (*IntegrationSyncRunner, error) {
	if runtime == nil {
		return nil, myerrors.ErrSyncRunnerInvalidConfig
	}
	resolved, err := NewSyncRunnerConfig(config)
	if err != nil {
		return nil, err
	}
	return &IntegrationSyncRunner{runtime: runtime, config: resolved, status: SyncRunnerStatus{Enabled: resolved.Enabled, RunnerID: resolved.RunnerID}}, nil
}

func (r *IntegrationSyncRunner) Start(ctx context.Context) error {
	if r == nil {
		return myerrors.ErrSyncRunnerStartFailed
	}
	if !r.config.Enabled {
		r.setError("sync_runner_disabled")
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.Lock()
	if r.status.Running {
		r.mu.Unlock()
		return myerrors.ErrSyncRunnerAlreadyRunning
	}
	runContext, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.done = make(chan struct{})
	r.status.Running = true
	r.status.StartedAt = time.Now()
	r.status.LastErrorCategory = ""
	done := r.done
	r.mu.Unlock()
	go func() {
		defer close(done)
		defer r.finishRun()
		defer r.recoverPanic()
		r.Run(runContext)
	}()
	return nil
}

func (r *IntegrationSyncRunner) Run(ctx context.Context) {
	if r == nil || !r.config.Enabled {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.runPoll(ctx)
	timer := time.NewTimer(r.config.PollInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			delay := r.config.PollInterval
			if r.runPoll(ctx) {
				delay = syncRunnerErrorBackoff(delay)
			}
			timer.Reset(delay)
		}
	}
}

func (r *IntegrationSyncRunner) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	cancel, done, running := r.cancel, r.done, r.status.Running
	timeout := r.config.ShutdownTimeout
	r.mu.RUnlock()
	if !running || cancel == nil || done == nil {
		return nil
	}
	cancel()
	if ctx == nil {
		ctx = context.Background()
	}
	shutdownContext, shutdownCancel := context.WithTimeout(ctx, timeout)
	defer shutdownCancel()
	select {
	case <-done:
		return nil
	case <-shutdownContext.Done():
		r.setError("sync_runner_shutdown_timeout")
		return myerrors.ErrSyncRunnerShutdownTimeout
	}
}

func (r *IntegrationSyncRunner) Status() SyncRunnerStatus {
	if r == nil {
		return SyncRunnerStatus{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

func (r *IntegrationSyncRunner) runPoll(ctx context.Context) (failed bool) {
	if ctx.Err() != nil {
		return false
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			failed = true
			r.setError("sync_runner_panic_recovered")
			zap.L().Error("integration sync runner poll panic recovered", zap.String("runner_id", r.config.RunnerID), zap.String("error_category", "sync_runner_panic_recovered"))
		}
	}()
	r.mu.Lock()
	r.status.LastPollAt = time.Now()
	r.mu.Unlock()
	summary, err := r.runtime.RunOnce(ctx, r.config.ScheduleBatchSize, r.config.CoordinateBatchSize)
	if err != nil {
		r.setError("sync_runner_poll_failed")
		zap.L().Warn("integration sync runner poll failed", zap.String("runner_id", r.config.RunnerID), zap.String("error_category", "sync_runner_poll_failed"))
		return true
	}
	r.mu.Lock()
	r.status.LastSuccessAt = time.Now()
	r.status.ScheduledTotal += int64(summary.ScheduledBatches)
	r.status.CoordinatedTotal += int64(summary.Coordinated)
	r.status.SucceededTotal += int64(summary.Succeeded)
	r.status.FailedTotal += int64(summary.Failed)
	r.mu.Unlock()
	return false
}

func (r *IntegrationSyncRunner) finishRun() {
	r.mu.Lock()
	r.status.Running = false
	r.cancel = nil
	r.mu.Unlock()
}

func (r *IntegrationSyncRunner) recoverPanic() {
	if recovered := recover(); recovered != nil {
		r.setError("sync_runner_panic_recovered")
		zap.L().Error("integration sync runner panic recovered", zap.String("runner_id", r.config.RunnerID), zap.String("error_category", "sync_runner_panic_recovered"))
	}
}

func (r *IntegrationSyncRunner) setError(value string) {
	r.mu.Lock()
	r.status.LastErrorCategory = value
	r.mu.Unlock()
}

func syncRunnerErrorBackoff(interval time.Duration) time.Duration {
	value := interval * 2
	if value < minSyncPollInterval {
		return minSyncPollInterval
	}
	if value > 30*time.Second {
		return 30 * time.Second
	}
	return value
}
