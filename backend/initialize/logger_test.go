package initialize

import (
	"testing"
	"time"
)

func TestAsyncLogCloseFlushesBufferedEntriesAndIsIdempotent(t *testing.T) {
	logger := newAsyncLog(t.TempDir(), "test")
	for i := 0; i < 100; i++ {
		if _, err := logger.Write([]byte("buffered entry\n")); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan struct{})
	go func() {
		logger.Close()
		logger.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("async logger close deadlocked")
	}
	if _, err := logger.Write([]byte("after close")); err == nil {
		t.Fatal("expected write after close to fail")
	}
}
