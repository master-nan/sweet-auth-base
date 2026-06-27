package middleware

import (
	"backend/model"
	"testing"
	"time"
)

func TestTokenIssuedBeforePasswordChangeRejectsOldToken(t *testing.T) {
	changedAt := model.CustomTime(time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC))
	user := model.SysUser{PasswordChangedAt: &changedAt}

	if !tokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 9, 59, 58, 0, time.UTC), user, time.Date(2026, 6, 5, 10, 1, 0, 0, time.UTC)) {
		t.Fatal("expected token issued before password change to be rejected")
	}
}

func TestTokenIssuedBeforePasswordChangeAllowsFreshToken(t *testing.T) {
	changedAt := model.CustomTime(time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC))
	user := model.SysUser{PasswordChangedAt: &changedAt}

	now := time.Date(2026, 6, 5, 10, 1, 0, 0, time.UTC)
	if tokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC), user, now) {
		t.Fatal("expected token issued at password change time to be allowed")
	}
	if tokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 9, 59, 59, 0, time.UTC), user, now) {
		t.Fatal("expected token issued within clock precision tolerance to be allowed")
	}
	if tokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 10, 0, 1, 0, time.UTC), user, now) {
		t.Fatal("expected token issued after password change to be allowed")
	}
}

func TestTokenIssuedBeforePasswordChangeAllowsUserWithoutPasswordChangedAt(t *testing.T) {
	if tokenIssuedBeforePasswordChange(time.Now(), model.SysUser{}) {
		t.Fatal("expected user without password_changed_at to be allowed")
	}
}

func TestTokenIssuedBeforePasswordChangeAllowsClearlyFuturePasswordChangedAt(t *testing.T) {
	changedAt := model.CustomTime(time.Date(2026, 6, 5, 18, 0, 0, 0, time.UTC))
	user := model.SysUser{PasswordChangedAt: &changedAt}

	if tokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC), user, time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)) {
		t.Fatal("expected clearly future password_changed_at to be ignored")
	}
}
