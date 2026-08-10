package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	myerrors "backend/internal/errors"
)

type syncRunnerRuntimeStub struct {
	mu           sync.Mutex
	calls        int
	panicAt      int
	entered      chan struct{}
	release      chan struct{}
	ignoreCancel bool
}

func (s *syncRunnerRuntimeStub) RunOnce(ctx context.Context, _, _ int) (SyncRuntimeSummary, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()
	if s.entered != nil {
		select {
		case s.entered <- struct{}{}:
		default:
		}
	}
	if call == s.panicAt {
		panic("test panic")
	}
	if s.release != nil {
		if s.ignoreCancel {
			<-s.release
		} else {
			select {
			case <-s.release:
			case <-ctx.Done():
			}
		}
	}
	return SyncRuntimeSummary{ScheduledBatches: 1, Coordinated: 1}, nil
}

func (s *syncRunnerRuntimeStub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestIntegrationSyncRunnerLifecycle(t *testing.T) {
	disabled, err := NewIntegrationSyncRunner(&syncRunnerRuntimeStub{}, SyncRunnerConfig{})
	if err != nil || disabled.Start(context.Background()) != nil || disabled.Status().Running {
		t.Fatalf("disabled runner must remain stopped: status=%+v err=%v", disabled.Status(), err)
	}

	runtime := &syncRunnerRuntimeStub{entered: make(chan struct{}, 2)}
	runner, err := NewIntegrationSyncRunner(runtime, SyncRunnerConfig{Enabled: true, RunnerID: "sync-test", PollInterval: time.Second, ScheduleBatchSize: 2, CoordinateBatchSize: 3, ShutdownTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-runtime.entered:
	case <-time.After(time.Second):
		t.Fatal("runner did not poll")
	}
	if err := runner.Start(context.Background()); !errors.Is(err, myerrors.ErrSyncRunnerAlreadyRunning) {
		t.Fatalf("duplicate start error=%v", err)
	}
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatalf("idempotent stop: %v", err)
	}
	if runner.Status().Running || runtime.count() != 1 {
		t.Fatalf("unexpected final status=%+v calls=%d", runner.Status(), runtime.count())
	}
}

func TestIntegrationSyncRunnerRecoversPollPanic(t *testing.T) {
	runtime := &syncRunnerRuntimeStub{panicAt: 1}
	runner, err := NewIntegrationSyncRunner(runtime, SyncRunnerConfig{Enabled: true, RunnerID: "panic-test", PollInterval: time.Second, ShutdownTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runner.Status().LastErrorCategory != "sync_runner_panic_recovered" {
		t.Fatalf("status=%+v", runner.Status())
	}
}

func TestIntegrationSyncRunnerShutdownTimeout(t *testing.T) {
	runtime := &syncRunnerRuntimeStub{entered: make(chan struct{}, 1), release: make(chan struct{}), ignoreCancel: true}
	runner, err := NewIntegrationSyncRunner(runtime, SyncRunnerConfig{Enabled: true, RunnerID: "shutdown-test", PollInterval: time.Second, ShutdownTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	<-runtime.entered
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := runner.Stop(ctx); !errors.Is(err, myerrors.ErrSyncRunnerShutdownTimeout) {
		t.Fatalf("shutdown error=%v", err)
	}
	close(runtime.release)
}

func TestSyncRunnerConfigBounds(t *testing.T) {
	if _, err := NewSyncRunnerConfig(SyncRunnerConfig{Enabled: true, RunnerID: "x", PollInterval: -time.Second}); !errors.Is(err, myerrors.ErrSyncRunnerInvalidConfig) {
		t.Fatalf("negative poll interval error=%v", err)
	}
	if _, err := NewSyncRunnerConfig(SyncRunnerConfig{Enabled: true}); !errors.Is(err, myerrors.ErrSyncRunnerInvalidConfig) {
		t.Fatalf("missing runner id error=%v", err)
	}
}
