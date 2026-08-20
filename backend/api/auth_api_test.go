package api

import (
	"backend/config"
	"backend/internal/cache"
	error2 "backend/internal/errors"
	"errors"
	"sync"
	"testing"
	"time"
)

type memoryCacher struct {
	mu     sync.Mutex
	values map[string]interface{}
}

func (m *memoryCacher) SetIfAbsent(key string, value interface{}, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.values[key]; ok {
		return false, nil
	}
	m.values[key] = value
	return true, nil
}

func newMemoryCacher() *memoryCacher {
	return &memoryCacher{values: map[string]interface{}{}}
}

func (m *memoryCacher) Get(key string, value interface{}) error {
	return cache.ErrCacheMiss
}

func (m *memoryCacher) Set(key string, value interface{}, expiration time.Duration) error {
	m.values[key] = value
	return nil
}

func (m *memoryCacher) Del(key string) error {
	delete(m.values, key)
	return nil
}

func (m *memoryCacher) Exists(keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		if _, ok := m.values[key]; ok {
			count++
		}
	}
	return count, nil
}

func (m *memoryCacher) Expire(key string, expiration time.Duration) (bool, error) {
	if _, ok := m.values[key]; ok {
		return true, nil
	}
	return false, nil
}

func TestSmsSendIntervalSecondsUsesDefault(t *testing.T) {
	if got := smsSendIntervalSeconds(nil); got != defaultSmsSendIntervalSeconds {
		t.Fatalf("expected default interval %d, got %d", defaultSmsSendIntervalSeconds, got)
	}

	cfg := &config.Server{}
	if got := smsSendIntervalSeconds(cfg); got != defaultSmsSendIntervalSeconds {
		t.Fatalf("expected default interval %d, got %d", defaultSmsSendIntervalSeconds, got)
	}
}

func TestSmsSendIntervalSecondsUsesConfig(t *testing.T) {
	cfg := &config.Server{}
	cfg.ALiYun.SMS.SendIntervalSeconds = 90

	if got := smsSendIntervalSeconds(cfg); got != 90 {
		t.Fatalf("expected configured interval 90, got %d", got)
	}
}

func TestSmsSendRateLimitKeyDoesNotExposeMobile(t *testing.T) {
	key := smsSendRateLimitKey(1, "LOGIN_CODE", "13800138000")

	if key == "" {
		t.Fatal("expected rate limit key")
	}
	if key == "SMS_SEND_RATE_LIMIT_1:LOGIN_CODE:13800138000" {
		t.Fatalf("expected hashed rate limit key, got %s", key)
	}
}

func TestReserveSmsSendSlotRejectsDuplicate(t *testing.T) {
	sendCodeCache := cache.NewSendCodeCache(newMemoryCacher())
	key := smsSendRateLimitKey(1, "LOGIN_CODE", "13800138000")

	if err := reserveSmsSendSlot(sendCodeCache, key, 60); err != nil {
		t.Fatalf("expected first reservation to pass: %v", err)
	}

	err := reserveSmsSendSlot(sendCodeCache, key, 60)
	if !errors.Is(err, error2.ErrSmsSendTooFrequent) {
		t.Fatalf("expected too frequent error, got %v", err)
	}
}

func TestSmsVerificationCodeFromParams(t *testing.T) {
	code, ok := smsVerificationCodeFromParams(map[string]interface{}{"code": "123456"})
	if !ok || code != "123456" {
		t.Fatalf("expected sms code to be extracted, got code=%q ok=%v", code, ok)
	}

	for name, params := range map[string]map[string]interface{}{
		"empty":       {},
		"non string":  {"code": 123456},
		"blank":       {"code": ""},
		"extra field": {"code": "123456", "name": "Nan"},
	} {
		if code, ok := smsVerificationCodeFromParams(params); ok || code != "" {
			t.Fatalf("%s: expected no sms code, got code=%q ok=%v", name, code, ok)
		}
	}
}

func TestSmsSendResponseDoesNotExposeTemplateParams(t *testing.T) {
	response := map[string]interface{}{"sent": true}

	if response["sent"] != true {
		t.Fatalf("expected sent=true response, got %#v", response)
	}
	if _, exists := response["code"]; exists {
		t.Fatalf("sms send response must not expose verification code: %#v", response)
	}
}
