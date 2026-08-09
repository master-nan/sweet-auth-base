package integration

import (
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestIntegrationWorkerRunnerLifecycleAndDisabledMode(t *testing.T) {
	disabledRuntime := &workerRuntimeStub{}
	disabled, err := NewIntegrationWorkerRunner(disabledRuntime, WorkerRunnerConfig{})
	mustWorkerNoError(t, err)
	mustWorkerNoError(t, disabled.Start(context.Background()))
	if disabled.Status().Running || disabled.Status().LastErrorCategory != "worker_disabled" || disabledRuntime.claimCount() != 0 {
		t.Fatalf("disabled worker started unexpectedly: %+v", disabled.Status())
	}

	runtime := &workerRuntimeStub{}
	runner, err := NewIntegrationWorkerRunner(runtime, WorkerRunnerConfig{
		Enabled: true, WorkerID: "worker-lifecycle", PollInterval: time.Second,
		LeaseRecoveryInterval: 10 * time.Second, ShutdownTimeout: time.Second,
	})
	mustWorkerNoError(t, err)
	mustWorkerNoError(t, runner.Start(context.Background()))
	if runner.Start(context.Background()) == nil {
		t.Fatal("expected duplicate start to fail")
	}
	mustWorkerEventually(t, func() bool { return !runner.Status().LastPollAt.IsZero() }, time.Second)
	mustWorkerNoError(t, runner.Stop(context.Background()))
	mustWorkerEventually(t, func() bool { return !runner.Status().Running }, time.Second)
	mustWorkerNoError(t, runner.Stop(context.Background()))
}

func TestNewWorkerRunnerConfigRejectsUnsafeValues(t *testing.T) {
	_, err := NewWorkerRunnerConfig(WorkerRunnerConfig{Enabled: true, WorkerID: "worker", PollInterval: 0, ClaimBatchSize: 33})
	if err == nil {
		t.Fatal("expected excessive claim batch to fail")
	}
	_, err = NewWorkerRunnerConfig(WorkerRunnerConfig{Enabled: true, WorkerID: "", PollInterval: time.Second})
	if err == nil {
		t.Fatal("expected enabled worker without identity to fail")
	}
	_, err = NewWorkerRunnerConfig(WorkerRunnerConfig{Enabled: true, WorkerID: "worker", LeaseDuration: IntegrationMinimumLeaseDuration - time.Second})
	if !errors.Is(err, myerrors.ErrIntegrationLeaseMarginInsufficient) {
		t.Fatalf("insufficient lease margin error = %v", err)
	}
}

func TestIntegrationWorkerRunnerBoundsConcurrencyAndPreservesRetryWaiting(t *testing.T) {
	runtime := &workerRuntimeStub{claimed: [][]repository.ClaimedIntegrationExecution{{
		workerClaim(1), workerClaim(2), workerClaim(3),
	}}}
	runtime.run = func(context.Context, repository.ClaimedIntegrationExecution) (AttemptResult, error) {
		runtime.enterExecution()
		defer runtime.leaveExecution()
		time.Sleep(20 * time.Millisecond)
		return AttemptResult{Succeeded: true}, nil
	}
	runner, err := NewIntegrationWorkerRunner(runtime, WorkerRunnerConfig{
		Enabled: true, WorkerID: "worker-concurrency", ClaimBatchSize: 3, InstanceConcurrency: 2,
		PollInterval: time.Second, LeaseRecoveryInterval: 10 * time.Second, ShutdownTimeout: time.Second,
	})
	mustWorkerNoError(t, err)
	if runner.runPoll(context.Background()) {
		t.Fatal("successful poll returned failure")
	}
	status := runner.Status()
	if status.ClaimedTotal != 3 || status.CompletedTotal != 3 || status.ActiveExecutionCount != 0 || runtime.maxActive() > 2 || runtime.recoveryCount() != 0 {
		t.Fatalf("unexpected concurrent execution state: status=%+v max=%d recovery=%d", status, runtime.maxActive(), runtime.recoveryCount())
	}
}

func TestIntegrationWorkerRunnerPollFailureBackoffAndRecovery(t *testing.T) {
	runtime := &workerRuntimeStub{claimErr: errors.New("database unavailable"), recovered: 2}
	runner, err := NewIntegrationWorkerRunner(runtime, WorkerRunnerConfig{
		Enabled: true, WorkerID: "worker-recovery", PollInterval: time.Second,
		LeaseRecoveryInterval: 10 * time.Second, ShutdownTimeout: time.Second,
	})
	mustWorkerNoError(t, err)
	if !runner.runPoll(context.Background()) || runner.Status().LastErrorCategory != "worker_claim_failed" || workerErrorBackoff(time.Second) != 2*time.Second {
		t.Fatalf("claim failure was not safely backed off: %+v", runner.Status())
	}
	runner.runRecovery(context.Background())
	if runner.Status().RecoveredTotal != 2 || runtime.recoveryCount() != 1 {
		t.Fatalf("unexpected recovery status: %+v", runner.Status())
	}
}

func TestIntegrationWorkerRunnerRecoversExecutionPanicAndShutdownTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	runtime := &workerRuntimeStub{claimed: [][]repository.ClaimedIntegrationExecution{{workerClaim(1)}}}
	runtime.run = func(context.Context, repository.ClaimedIntegrationExecution) (AttemptResult, error) {
		close(started)
		<-release
		return AttemptResult{Succeeded: true}, nil
	}
	runner, err := NewIntegrationWorkerRunner(runtime, WorkerRunnerConfig{
		Enabled: true, WorkerID: "worker-stop", PollInterval: time.Second, ClaimBatchSize: 1,
		InstanceConcurrency: 1, LeaseRecoveryInterval: 10 * time.Second, ShutdownTimeout: time.Second,
	})
	mustWorkerNoError(t, err)
	mustWorkerNoError(t, runner.Start(context.Background()))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("execution did not start")
	}
	shortContext, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if runner.Stop(shortContext) == nil {
		t.Fatal("expected short shutdown to time out")
	}
	close(release)
	mustWorkerEventually(t, func() bool { return !runner.Status().Running }, time.Second)

	panicRuntime := &workerRuntimeStub{claimed: [][]repository.ClaimedIntegrationExecution{{workerClaim(2)}}}
	panicRuntime.run = func(context.Context, repository.ClaimedIntegrationExecution) (AttemptResult, error) {
		panic("controlled test panic")
	}
	panicRunner, err := NewIntegrationWorkerRunner(panicRuntime, WorkerRunnerConfig{
		Enabled: true, WorkerID: "worker-panic", PollInterval: time.Second,
		LeaseRecoveryInterval: 10 * time.Second, ShutdownTimeout: time.Second,
	})
	mustWorkerNoError(t, err)
	if panicRunner.runPoll(context.Background()) || panicRunner.Status().LastErrorCategory != "worker_panic_recovered" || panicRunner.Status().FailedTotal != 1 {
		t.Fatalf("execution panic was not recovered: %+v", panicRunner.Status())
	}
}

func TestIntegrationWorkerRunnerStatusConcurrentRead(t *testing.T) {
	runtime := &workerRuntimeStub{}
	runner, err := NewIntegrationWorkerRunner(runtime, WorkerRunnerConfig{})
	mustWorkerNoError(t, err)
	var group sync.WaitGroup
	for index := 0; index < 32; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for item := 0; item < 100; item++ {
				runner.setLastPoll(time.Now())
				_ = runner.Status()
			}
		}()
	}
	group.Wait()
}

func TestIntegrationWorkerRunnerAutomaticallyExecutesCreatedExecution(t *testing.T) {
	engine, db, execution, closeServer := newExecutionEngineFixture(t, 200)
	defer closeServer()
	runner, err := NewIntegrationWorkerRunner(engine, WorkerRunnerConfig{
		Enabled: true, WorkerID: "runtime-worker-1", PollInterval: time.Second, ClaimBatchSize: 2,
		InstanceConcurrency: 2, LeaseRecoveryInterval: 10 * time.Second, ShutdownTimeout: time.Second,
	})
	mustWorkerNoError(t, err)
	mustWorkerNoError(t, runner.Start(context.Background()))
	mustWorkerEventually(t, func() bool {
		var stored model.IntegrationExecution
		return db.First(&stored, execution.Id).Error == nil && stored.Status == model.IntegrationExecutionStatusSucceeded
	}, time.Second)
	mustWorkerNoError(t, runner.Stop(context.Background()))
	if runner.Status().CompletedTotal != 1 {
		t.Fatalf("unexpected completed count: %+v", runner.Status())
	}
}

func TestIntegrationWorkerRunnerDoesNotReclaimRetryBeforeNextRunAt(t *testing.T) {
	engine, db, execution, closeServer := newExecutionEngineFixture(t, 503)
	defer closeServer()
	runner, err := NewIntegrationWorkerRunner(engine, WorkerRunnerConfig{
		Enabled: true, WorkerID: "runtime-worker-1", PollInterval: time.Second, ClaimBatchSize: 2,
		InstanceConcurrency: 2, LeaseRecoveryInterval: 10 * time.Second, ShutdownTimeout: time.Second,
	})
	mustWorkerNoError(t, err)
	mustWorkerNoError(t, runner.Start(context.Background()))
	mustWorkerEventually(t, func() bool {
		var stored model.IntegrationExecution
		return db.First(&stored, execution.Id).Error == nil && stored.Status == model.IntegrationExecutionStatusRetryWaiting
	}, time.Second)
	time.Sleep(30 * time.Millisecond)
	var stored model.IntegrationExecution
	mustWorkerNoError(t, db.First(&stored, execution.Id).Error)
	if stored.CurrentAttempt != 1 {
		t.Fatalf("retry waiting execution was claimed again: %+v", stored)
	}
	mustWorkerNoError(t, runner.Stop(context.Background()))
}

func mustWorkerNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustWorkerEventually(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not satisfied before timeout")
}

func workerClaim(id int) repository.ClaimedIntegrationExecution {
	return repository.ClaimedIntegrationExecution{
		Execution: model.IntegrationExecution{Basic: model.Basic{Id: id}, ExecutionNo: "EXE-WORKER", Status: model.IntegrationExecutionStatusRunning},
		Attempt:   model.IntegrationLog{Basic: model.Basic{Id: id}, AttemptNo: 1, Status: model.IntegrationLogStatusRunning},
	}
}

type workerRuntimeStub struct {
	mu           sync.Mutex
	claimed      [][]repository.ClaimedIntegrationExecution
	claimErr     error
	run          func(context.Context, repository.ClaimedIntegrationExecution) (AttemptResult, error)
	recovered    int
	recoveryErr  error
	claims       int
	recoveries   int
	active       int
	maxExecuting int
}

func (s *workerRuntimeStub) ClaimReadyExecutions(context.Context) ([]repository.ClaimedIntegrationExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claims++
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	if len(s.claimed) == 0 {
		return nil, nil
	}
	value := s.claimed[0]
	s.claimed = s.claimed[1:]
	return value, nil
}

func (s *workerRuntimeStub) RunExecution(ctx context.Context, claimed repository.ClaimedIntegrationExecution) (AttemptResult, error) {
	if s.run == nil {
		return AttemptResult{Succeeded: true}, nil
	}
	return s.run(ctx, claimed)
}

func (s *workerRuntimeStub) RecoverExpiredLease(context.Context, int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recoveries++
	return s.recovered, s.recoveryErr
}

func (s *workerRuntimeStub) enterExecution() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active++
	if s.active > s.maxExecuting {
		s.maxExecuting = s.active
	}
}

func (s *workerRuntimeStub) leaveExecution() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active--
}

func (s *workerRuntimeStub) maxActive() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxExecuting
}

func (s *workerRuntimeStub) claimCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.claims
}

func (s *workerRuntimeStub) recoveryCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recoveries
}
