package middleware

import (
	"backend/model"
	"backend/service"
	"net/http"
	"testing"
	"time"
)

func TestTokenIssuedBeforePasswordChangeRejectsOldToken(t *testing.T) {
	changedAt := model.CustomTime(time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC))
	user := model.SysUser{PasswordChangedAt: &changedAt}

	if !service.TokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 9, 59, 58, 0, time.UTC), user, time.Date(2026, 6, 5, 10, 1, 0, 0, time.UTC)) {
		t.Fatal("expected token issued before password change to be rejected")
	}
}

func TestRequiredPasswordChangeOnlyAllowsPasswordUpdate(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{method: http.MethodPost, path: "/sweet_admin/admin/user/password", want: true},
		{method: http.MethodPost, path: "/sweet_admin/api/user/password", want: true},
		{method: http.MethodPost, path: "/sweet_admin/admin/user/password/", want: true},
		{method: http.MethodGet, path: "/sweet_admin/admin/user/password", want: false},
		{method: http.MethodGet, path: "/sweet_admin/admin/user/me", want: false},
		{method: http.MethodPost, path: "/sweet_admin/admin/report/query", want: false},
	}
	for _, tt := range tests {
		if got := allowsRequiredPasswordChange(tt.method, tt.path); got != tt.want {
			t.Fatalf("allowsRequiredPasswordChange(%q, %q)=%v want %v", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestTokenIssuedBeforePasswordChangeAllowsFreshToken(t *testing.T) {
	changedAt := model.CustomTime(time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC))
	user := model.SysUser{PasswordChangedAt: &changedAt}

	now := time.Date(2026, 6, 5, 10, 1, 0, 0, time.UTC)
	if service.TokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC), user, now) {
		t.Fatal("expected token issued at password change time to be allowed")
	}
	if !service.TokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 9, 59, 59, 0, time.UTC), user, now) {
		t.Fatal("expected token issued in the prior second to be rejected")
	}
	if service.TokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 10, 0, 1, 0, time.UTC), user, now) {
		t.Fatal("expected token issued after password change to be allowed")
	}
}

func TestTokenIssuedBeforePasswordChangeAllowsUserWithoutPasswordChangedAt(t *testing.T) {
	if service.TokenIssuedBeforePasswordChangeAt(time.Now(), model.SysUser{}, time.Now()) {
		t.Fatal("expected user without password_changed_at to be allowed")
	}
}

func TestTokenIssuedBeforePasswordChangeAllowsClearlyFuturePasswordChangedAt(t *testing.T) {
	changedAt := model.CustomTime(time.Date(2026, 6, 5, 18, 0, 0, 0, time.UTC))
	user := model.SysUser{PasswordChangedAt: &changedAt}

	if service.TokenIssuedBeforePasswordChangeAt(time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC), user, time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)) {
		t.Fatal("expected clearly future password_changed_at to be ignored")
	}
}
