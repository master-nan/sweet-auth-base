package cache

import (
	"testing"
	"time"
)

type loginAttemptMemoryCacher struct {
	values map[string]interface{}
}

func newLoginAttemptMemoryCacher() *loginAttemptMemoryCacher {
	return &loginAttemptMemoryCacher{values: map[string]interface{}{}}
}

func (m *loginAttemptMemoryCacher) Get(key string, value interface{}) error {
	v, ok := m.values[key]
	if !ok {
		return ErrCacheMiss
	}
	switch target := value.(type) {
	case *int:
		*target = v.(int)
	default:
		t, ok := value.(*interface{})
		if ok {
			*t = v
		}
	}
	return nil
}

func (m *loginAttemptMemoryCacher) Set(key string, value interface{}, _ time.Duration) error {
	m.values[key] = value
	return nil
}

func (m *loginAttemptMemoryCacher) Del(key string) error {
	delete(m.values, key)
	return nil
}

func (m *loginAttemptMemoryCacher) Exists(keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		if _, ok := m.values[key]; ok {
			count++
		}
	}
	return count, nil
}

func (m *loginAttemptMemoryCacher) Expire(_ string, _ time.Duration) (bool, error) {
	return true, nil
}

func TestLoginAttemptCacheLocksAfterConfiguredFailures(t *testing.T) {
	store := newLoginAttemptMemoryCacher()
	attempts := NewLoginAttemptCache(store)

	locked, err := attempts.RecordFailure(" Admin ", 2, time.Minute)
	if err != nil {
		t.Fatalf("record first failure: %v", err)
	}
	if locked {
		t.Fatalf("expected first failure without lock")
	}

	locked, err = attempts.RecordFailure("admin", 2, time.Minute)
	if err != nil {
		t.Fatalf("record second failure: %v", err)
	}
	if !locked {
		t.Fatalf("expected second failure to lock")
	}

	isLocked, err := attempts.IsLocked("ADMIN")
	if err != nil {
		t.Fatalf("check lock: %v", err)
	}
	if !isLocked {
		t.Fatalf("expected normalized principal to be locked")
	}
}

func TestLoginAttemptCacheClearRemovesAttemptsAndLock(t *testing.T) {
	store := newLoginAttemptMemoryCacher()
	attempts := NewLoginAttemptCache(store)

	_, _ = attempts.RecordFailure("admin", 1, time.Minute)
	if err := attempts.Clear("admin"); err != nil {
		t.Fatalf("clear attempts: %v", err)
	}
	isLocked, err := attempts.IsLocked("admin")
	if err != nil {
		t.Fatalf("check lock: %v", err)
	}
	if isLocked {
		t.Fatalf("expected lock to be cleared")
	}
	if _, ok := store.values[LoginAttemptCacheKey+"admin"]; ok {
		t.Fatalf("expected attempt counter to be cleared")
	}
}
