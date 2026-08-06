package integration

import (
	myerrors "backend/internal/errors"
	"backend/repository"
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultWorkerPollInterval     = 5 * time.Second
	defaultWorkerClaimBatchSize   = 8
	defaultWorkerConcurrency      = 4
	defaultWorkerRecoveryInterval = time.Minute
	defaultWorkerShutdownTimeout  = 15 * time.Second
	maxWorkerClaimBatchSize       = 32
	maxWorkerConcurrency          = 16
	minWorkerPollInterval         = time.Second
	minWorkerRecoveryInterval     = 10 * time.Second
	maxWorkerInterval             = 5 * time.Minute
	maxWorkerShutdownTimeout      = time.Minute
)

// ExecutionRuntime 只暴露已冻结的 Execution Engine 原语；Runner 不重新实现领取或状态机。
type ExecutionRuntime interface {
	ClaimCreatedExecutions(context.Context) ([]repository.ClaimedIntegrationExecution, error)
	RunExecution(context.Context, repository.ClaimedIntegrationExecution) (AttemptResult, error)
	RecoverExpiredLease(context.Context, int) (int, error)
}

// WorkerRunnerConfig 是已解析的服务端运行配置。默认禁用，需显式设置 Enabled 才会启动。
type WorkerRunnerConfig struct {
	Enabled               bool
	WorkerID              string
	PollInterval          time.Duration
	ClaimBatchSize        int
	InstanceConcurrency   int
	LeaseRecoveryInterval time.Duration
	ShutdownTimeout       time.Duration
	LeaseDuration         time.Duration
}

// WorkerStatus 是仅供受控健康检查或进程诊断读取的安全摘要。
type WorkerStatus struct {
	Enabled              bool
	Running              bool
	WorkerID             string
	StartedAt            time.Time
	LastPollAt           time.Time
	LastSuccessAt        time.Time
	LastErrorCategory    string
	ActiveExecutionCount int
	ClaimedTotal         int64
	CompletedTotal       int64
	FailedTotal          int64
	RecoveredTotal       int64
}

// IntegrationWorkerRunner 负责应用生命周期、轮询与本实例并发，不保存事务或凭证材料。
type IntegrationWorkerRunner struct {
	runtime ExecutionRuntime
	config  WorkerRunnerConfig

	mu     sync.RWMutex
	status WorkerStatus
	cancel context.CancelFunc
	done   chan struct{}
}

// NewIntegrationWorkerRunner 构造生命周期 Runner。非法配置在启动前安全失败。
func NewIntegrationWorkerRunner(runtime ExecutionRuntime, config WorkerRunnerConfig) (*IntegrationWorkerRunner, error) {
	if runtime == nil {
		return nil, myerrors.ErrIntegrationWorkerInvalidConfig
	}
	resolved, err := resolveWorkerRunnerConfig(config)
	if err != nil {
		return nil, err
	}
	return &IntegrationWorkerRunner{
		runtime: runtime,
		config:  resolved,
		status:  WorkerStatus{Enabled: resolved.Enabled, WorkerID: resolved.WorkerID},
	}, nil
}

// NewWorkerRunnerConfig 统一补齐并校验服务端 Worker 配置，供初始化层和 Runner 复用。
func NewWorkerRunnerConfig(config WorkerRunnerConfig) (WorkerRunnerConfig, error) {
	return resolveWorkerRunnerConfig(config)
}

func resolveWorkerRunnerConfig(config WorkerRunnerConfig) (WorkerRunnerConfig, error) {
	config.WorkerID = strings.TrimSpace(config.WorkerID)
	if config.PollInterval == 0 {
		config.PollInterval = defaultWorkerPollInterval
	}
	if config.ClaimBatchSize == 0 {
		config.ClaimBatchSize = defaultWorkerClaimBatchSize
	}
	if config.InstanceConcurrency == 0 {
		config.InstanceConcurrency = defaultWorkerConcurrency
	}
	if config.LeaseRecoveryInterval == 0 {
		config.LeaseRecoveryInterval = defaultWorkerRecoveryInterval
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = defaultWorkerShutdownTimeout
	}
	if config.LeaseDuration == 0 {
		config.LeaseDuration = IntegrationDefaultLeaseDuration
	}
	if config.PollInterval < minWorkerPollInterval || config.PollInterval > maxWorkerInterval ||
		config.ClaimBatchSize < 1 || config.ClaimBatchSize > maxWorkerClaimBatchSize ||
		config.InstanceConcurrency < 1 || config.InstanceConcurrency > maxWorkerConcurrency ||
		config.LeaseRecoveryInterval < minWorkerRecoveryInterval || config.LeaseRecoveryInterval > maxWorkerInterval ||
		config.ShutdownTimeout < time.Second || config.ShutdownTimeout > maxWorkerShutdownTimeout {
		return WorkerRunnerConfig{}, myerrors.ErrIntegrationWorkerInvalidConfig
	}
	if err := ValidateLeaseDuration(config.LeaseDuration); err != nil {
		return WorkerRunnerConfig{}, err
	}
	if config.Enabled && (config.WorkerID == "" || len(config.WorkerID) > 128) {
		return WorkerRunnerConfig{}, myerrors.ErrIntegrationWorkerInvalidConfig
	}
	if config.WorkerID == "" {
		config.WorkerID = "integration-disabled"
	}
	return config, nil
}

// Start 按应用级 Context 启动唯一 Runner；禁用状态是受控的空操作。
func (r *IntegrationWorkerRunner) Start(ctx context.Context) error {
	if r == nil {
		return myerrors.ErrIntegrationWorkerStartFailed
	}
	if !r.config.Enabled {
		r.setErrorCategory("worker_disabled")
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.Lock()
	if r.status.Running {
		r.mu.Unlock()
		return myerrors.ErrIntegrationWorkerAlreadyRunning
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
		defer r.recoverRunnerPanic()
		r.Run(runContext)
	}()
	return nil
}

// Run 执行可取消轮询循环。通常由 Start 调用，测试可直接使用标准 Context 驱动。
func (r *IntegrationWorkerRunner) Run(ctx context.Context) {
	if r == nil || !r.config.Enabled {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.runPoll(ctx)
	pollTimer := time.NewTimer(r.config.PollInterval)
	recoveryTimer := time.NewTimer(r.config.LeaseRecoveryInterval)
	defer pollTimer.Stop()
	defer recoveryTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTimer.C:
			delay := r.config.PollInterval
			if r.runPoll(ctx) {
				delay = workerErrorBackoff(r.config.PollInterval)
			}
			pollTimer.Reset(delay)
		case <-recoveryTimer.C:
			r.runRecovery(ctx)
			recoveryTimer.Reset(r.config.LeaseRecoveryInterval)
		}
	}
}

// Stop 先取消新领取，再在有限期限内等待本地调用返回；未完成任务交由租约恢复处理。
func (r *IntegrationWorkerRunner) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	cancel := r.cancel
	done := r.done
	running := r.status.Running
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
		r.setErrorCategory("worker_shutdown_timeout")
		return myerrors.ErrIntegrationWorkerShutdownTimeout
	}
}

// Status 返回副本，调用方无法修改内部统计。
func (r *IntegrationWorkerRunner) Status() WorkerStatus {
	if r == nil {
		return WorkerStatus{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

func (r *IntegrationWorkerRunner) runPoll(ctx context.Context) (failed bool) {
	if ctx.Err() != nil {
		return false
	}
	r.setLastPoll(time.Now())
	claimed, err := r.runtime.ClaimCreatedExecutions(ctx)
	if err != nil {
		r.setErrorCategory("worker_claim_failed")
		zap.L().Warn("integration worker claim failed", zap.String("worker_id", r.config.WorkerID), zap.String("error_category", "worker_claim_failed"))
		return true
	}
	if len(claimed) == 0 {
		return false
	}
	r.addClaimed(int64(len(claimed)))

	semaphore := make(chan struct{}, r.config.InstanceConcurrency)
	var group sync.WaitGroup
	for _, item := range claimed {
		select {
		case <-ctx.Done():
			group.Wait()
			return false
		case semaphore <- struct{}{}:
		}
		group.Add(1)
		go r.runClaimedExecution(ctx, item, semaphore, &group)
	}
	group.Wait()
	return false
}

func (r *IntegrationWorkerRunner) runClaimedExecution(
	ctx context.Context,
	claimed repository.ClaimedIntegrationExecution,
	semaphore chan struct{},
	group *sync.WaitGroup,
) {
	defer group.Done()
	defer func() { <-semaphore }()
	r.adjustActive(1)
	defer r.adjustActive(-1)
	defer func() {
		if recovered := recover(); recovered != nil {
			r.addFailed(1)
			r.setErrorCategory("worker_panic_recovered")
			zap.L().Error("integration worker execution panic recovered",
				zap.String("worker_id", r.config.WorkerID),
				zap.String("execution_no", claimed.Execution.ExecutionNo),
				zap.Int("attempt_no", claimed.Attempt.AttemptNo),
				zap.String("error_category", "worker_panic_recovered"),
			)
		}
	}()
	result, err := r.runtime.RunExecution(ctx, claimed)
	if err != nil || !result.Succeeded {
		r.addFailed(1)
		if err != nil {
			r.setErrorCategory("worker_execution_failed")
			zap.L().Warn("integration worker execution failed",
				zap.String("worker_id", r.config.WorkerID), zap.String("execution_no", claimed.Execution.ExecutionNo),
				zap.Int("attempt_no", claimed.Attempt.AttemptNo), zap.String("error_category", "worker_execution_failed"),
			)
		}
		return
	}
	r.addCompleted(1)
	r.setLastSuccess(time.Now())
}

func (r *IntegrationWorkerRunner) runRecovery(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	recovered, err := r.runtime.RecoverExpiredLease(ctx, r.config.ClaimBatchSize)
	if err != nil {
		r.setErrorCategory("worker_recovery_failed")
		zap.L().Warn("integration worker lease recovery failed", zap.String("worker_id", r.config.WorkerID), zap.String("error_category", "worker_recovery_failed"))
		return
	}
	if recovered > 0 {
		r.addRecovered(int64(recovered))
		r.setLastSuccess(time.Now())
		zap.L().Info("integration worker recovered expired leases", zap.String("worker_id", r.config.WorkerID), zap.Int("recovered_count", recovered))
	}
}

func workerErrorBackoff(interval time.Duration) time.Duration {
	backoff := interval * 2
	if backoff < minWorkerPollInterval {
		return minWorkerPollInterval
	}
	if backoff > 30*time.Second {
		return 30 * time.Second
	}
	return backoff
}

func (r *IntegrationWorkerRunner) finishRun() {
	r.mu.Lock()
	r.status.Running = false
	r.cancel = nil
	r.mu.Unlock()
}

func (r *IntegrationWorkerRunner) recoverRunnerPanic() {
	if recovered := recover(); recovered != nil {
		r.setErrorCategory("worker_panic_recovered")
		zap.L().Error("integration worker runner panic recovered", zap.String("worker_id", r.config.WorkerID), zap.String("error_category", "worker_panic_recovered"))
	}
}

func (r *IntegrationWorkerRunner) setLastPoll(value time.Time) {
	r.mu.Lock()
	r.status.LastPollAt = value
	r.mu.Unlock()
}

func (r *IntegrationWorkerRunner) setLastSuccess(value time.Time) {
	r.mu.Lock()
	r.status.LastSuccessAt = value
	r.mu.Unlock()
}

func (r *IntegrationWorkerRunner) setErrorCategory(value string) {
	r.mu.Lock()
	r.status.LastErrorCategory = value
	r.mu.Unlock()
}

func (r *IntegrationWorkerRunner) adjustActive(delta int) {
	r.mu.Lock()
	r.status.ActiveExecutionCount += delta
	r.mu.Unlock()
}

func (r *IntegrationWorkerRunner) addClaimed(value int64) {
	r.mu.Lock()
	r.status.ClaimedTotal += value
	r.mu.Unlock()
}

func (r *IntegrationWorkerRunner) addCompleted(value int64) {
	r.mu.Lock()
	r.status.CompletedTotal += value
	r.mu.Unlock()
}

func (r *IntegrationWorkerRunner) addFailed(value int64) {
	r.mu.Lock()
	r.status.FailedTotal += value
	r.mu.Unlock()
}

func (r *IntegrationWorkerRunner) addRecovered(value int64) {
	r.mu.Lock()
	r.status.RecoveredTotal += value
	r.mu.Unlock()
}
