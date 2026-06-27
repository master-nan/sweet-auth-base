package service

import (
	"backend/model"
	"testing"
	"time"
)

func TestPasswordChangeRequirementRequiresInitialReset(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	changedAt := model.CustomTime(now)

	required, reason := PasswordChangeRequirement(model.SysUser{
		IsReset:           true,
		PasswordChangedAt: &changedAt,
	}, model.SysConfigure{PasswordExpireTime: 90}, now)

	if !required || reason != "initial_reset" {
		t.Fatalf("expected initial reset requirement, got required=%v reason=%q", required, reason)
	}
}

func TestPasswordChangeRequirementRequiresExpiredPassword(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	changedAt := model.CustomTime(now.AddDate(0, 0, -91))

	required, reason := PasswordChangeRequirement(model.SysUser{
		PasswordChangedAt: &changedAt,
	}, model.SysConfigure{PasswordExpireTime: 90}, now)

	if !required || reason != "expired" {
		t.Fatalf("expected expired password requirement, got required=%v reason=%q", required, reason)
	}
}

func TestPasswordChangeRequirementAllowsFreshPassword(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	changedAt := model.CustomTime(now.AddDate(0, 0, -30))

	required, reason := PasswordChangeRequirement(model.SysUser{
		PasswordChangedAt: &changedAt,
	}, model.SysConfigure{PasswordExpireTime: 90}, now)

	if required || reason != "" {
		t.Fatalf("expected fresh password to be allowed, got required=%v reason=%q", required, reason)
	}
}

func TestPasswordChangeRequirementAllowsWhenExpirationDisabled(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

	required, reason := PasswordChangeRequirement(model.SysUser{}, model.SysConfigure{PasswordExpireTime: 0}, now)

	if required || reason != "" {
		t.Fatalf("expected disabled expiration to be allowed, got required=%v reason=%q", required, reason)
	}
}
