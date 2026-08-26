package main

import (
	"backend/config"
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

type testRunner struct {
	mu     sync.Mutex
	events *[]string
}

func (r *testRunner) Start(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.events = append(*r.events, "runner-start")
	return nil
}

func (r *testRunner) Stop(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.events = append(*r.events, "runner-stop")
	return nil
}

type testChunkCleaner struct{}

func (testChunkCleaner) CleanupExpiredChunks(time.Time, time.Duration) (int, error) { return 0, nil }

type shutdownErrorListener struct {
	net.Listener
	closeErr error
}

func (l *shutdownErrorListener) Close() error {
	if err := l.Listener.Close(); err != nil {
		return err
	}
	return l.closeErr
}

func TestRunRuntimeStopsAcceptingRequestsBeforeClosingResources(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		writer.WriteHeader(http.StatusNoContent)
	})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make([]string, 0, 8)
	resourcesClosed := make(chan struct{})
	worker := &testRunner{events: &events}
	syncRunner := &testRunner{events: &events}
	done := make(chan error, 1)
	go func() {
		done <- runRuntime(ctx, listener, server, runtimeDependencies{
			worker: worker, syncRunner: syncRunner, chunkCleaner: testChunkCleaner{},
			uploadConfig: config.Upload{ChunkTTLHours: 24, ChunkCleanupMinutes: 60},
			stopCron: func(context.Context) error {
				events = append(events, "cron-stop")
				return nil
			},
			closeResources: func() error {
				close(resourcesClosed)
				events = append(events, "resources-close")
				return nil
			},
			closeLogger: func() { events = append(events, "logger-close") },
		})
	}()

	responseDone := make(chan struct{})
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			_ = response.Body.Close()
		}
		close(responseDone)
	}()
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach server")
	}
	cancel()

	deadline := time.Now().Add(2 * time.Second)
	listenerClosed := false
	for time.Now().Before(deadline) {
		connection, dialErr := net.DialTimeout("tcp", listener.Addr().String(), 25*time.Millisecond)
		if dialErr != nil {
			listenerClosed = true
			break
		}
		_ = connection.Close()
		time.Sleep(10 * time.Millisecond)
	}
	if !listenerClosed {
		t.Fatal("HTTP listener continued accepting connections after shutdown began")
	}
	select {
	case <-resourcesClosed:
		t.Fatal("runtime resources closed before the in-flight HTTP request completed")
	default:
	}
	close(releaseRequest)
	<-responseDone
	if err := <-done; err != nil {
		t.Fatalf("runRuntime: %v", err)
	}

	resourceIndex, loggerIndex := -1, -1
	for index, event := range events {
		if event == "resources-close" {
			resourceIndex = index
		}
		if event == "logger-close" {
			loggerIndex = index
		}
	}
	if resourceIndex < 0 || loggerIndex <= resourceIndex {
		t.Fatalf("unexpected shutdown order: %v", events)
	}
}

func TestRunRuntimePropagatesHTTPShutdownError(t *testing.T) {
	baseListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	shutdownErr := errors.New("listener shutdown failed")
	listener := &shutdownErrorListener{Listener: baseListener, closeErr: shutdownErr}
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make([]string, 0, 4)
	done := make(chan error, 1)
	go func() {
		done <- runRuntime(ctx, listener, server, runtimeDependencies{
			worker: &testRunner{events: &events}, syncRunner: &testRunner{events: &events},
			chunkCleaner: testChunkCleaner{},
			uploadConfig: config.Upload{ChunkTTLHours: 24, ChunkCleanupMinutes: 60},
		})
	}()

	response, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	cancel()
	if err := <-done; !errors.Is(err, shutdownErr) {
		t.Fatalf("runRuntime error = %v, want shutdown error", err)
	}
}
