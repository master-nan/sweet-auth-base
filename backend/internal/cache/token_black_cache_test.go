package cache

import (
	"backend/enum"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type failingCacher struct{}

func (f failingCacher) Get(key string, value interface{}) error {
	return ErrCacheMiss
}

func (f failingCacher) Set(key string, value interface{}, expiration time.Duration) error {
	return errors.New("cache set failed")
}

func (f failingCacher) Del(key string) error {
	return nil
}

func (f failingCacher) Exists(keys ...string) (int64, error) {
	return 0, nil
}

func (f failingCacher) Expire(key string, expiration time.Duration) (bool, error) {
	return false, nil
}

func TestTokenBlackCacheRevokeDoesNotLogTokenValue(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()

	tokenValue := "secret-token-value"
	tokenBlackCache := NewTokenBlackCache(failingCacher{})

	if err := tokenBlackCache.Revoke(enum.AccessToken, tokenValue, time.Now().Add(time.Hour)); err == nil {
		t.Fatal("expected cache set error")
	}

	if logs.Len() != 1 {
		t.Fatalf("expected one log entry, got %d", logs.Len())
	}
	entry := logs.All()[0]
	if entry.Message == tokenValue {
		t.Fatal("token value leaked in log message")
	}
	for key, value := range entry.ContextMap() {
		if key == "access_token" || key == "refresh_token" {
			t.Fatalf("token-specific field leaked in log context: %s", key)
		}
		if value == tokenValue {
			t.Fatalf("token value leaked in log context field %s", key)
		}
	}
}
