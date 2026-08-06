package integration

import (
	myerrors "backend/internal/errors"
	"errors"
	"testing"
	"time"
)

func TestRuntimeContractMatchesTransportHardLimits(t *testing.T) {
	limits := RuntimeLimits()
	if limits.MaxRequestTimeout != 120*time.Second || limits.MaxResponseBytes != 64*1024*1024 {
		t.Fatalf("unexpected runtime limits: %+v", limits)
	}
	if err := ValidateInterfaceRuntimeContract(int(limits.MaxRequestTimeout.Seconds()), limits.MaxResponseBytes); err != nil {
		t.Fatalf("maximum interface contract must be accepted: %v", err)
	}
	if _, err := NewTransportRequest(TransportRequestInput{
		Method: "GET", BaseURL: "https://api.example.com", RelativePath: "/health",
		Timeouts: TransportTimeouts{Request: limits.MaxRequestTimeout}, MaxResponseBytes: limits.MaxResponseBytes,
	}); err != nil {
		t.Fatalf("transport rejected configuration maximum: %v", err)
	}
	if err := ValidateInterfaceRuntimeContract(121, limits.MaxResponseBytes); !errors.Is(err, myerrors.ErrIntegrationTimeoutOutOfRange) {
		t.Fatalf("timeout upper bound error = %v", err)
	}
	if err := ValidateInterfaceRuntimeContract(120, limits.MaxResponseBytes+1); !errors.Is(err, myerrors.ErrIntegrationResponseLimitOutOfRange) {
		t.Fatalf("response upper bound error = %v", err)
	}
}

func TestLeaseBudgetIncludesCompletionAndClaimMargins(t *testing.T) {
	limits := RuntimeLimits()
	required := limits.MaxRequestTimeout + limits.CompletionMargin + limits.ClaimSafetyMargin
	if limits.MinimumLeaseDuration != required || limits.DefaultLeaseDuration <= required {
		t.Fatalf("unsafe lease budget: %+v", limits)
	}
	if err := ValidateLeaseDuration(limits.MinimumLeaseDuration); err != nil {
		t.Fatalf("minimum lease rejected: %v", err)
	}
	if err := ValidateLeaseDuration(limits.MinimumLeaseDuration - time.Second); !errors.Is(err, myerrors.ErrIntegrationLeaseMarginInsufficient) {
		t.Fatalf("insufficient margin error = %v", err)
	}
}
