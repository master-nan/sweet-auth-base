package service

import (
	"backend/config"
	"context"
	"strings"
	"testing"
)

func TestDevelopmentVerificationServiceRejectsSampleChangesInProduction(t *testing.T) {
	service := &DevelopmentVerificationService{config: &config.Server{Environment: "production"}}

	statuses, err := service.Statuses(context.Background())
	if err != nil {
		t.Fatalf("Statuses() error = %v", err)
	}
	if len(statuses) != len(developmentVerificationScenarios) {
		t.Fatalf("Statuses() returned %d scenarios, want %d", len(statuses), len(developmentVerificationScenarios))
	}
	for _, status := range statuses {
		if status.Available || status.State != "unavailable" {
			t.Fatalf("status %q = available %v, state %q", status.ScenarioId, status.Available, status.State)
		}
	}

	if _, err = service.Prepare(context.Background(), verificationScenarioMetadata); err == nil ||
		!strings.Contains(err.Error(), "生产环境禁止") {
		t.Fatalf("Prepare() error = %v, want production guard", err)
	}
	if _, err = service.Cleanup(context.Background(), verificationScenarioMetadata); err == nil ||
		!strings.Contains(err.Error(), "生产环境禁止") {
		t.Fatalf("Cleanup() error = %v, want production guard", err)
	}
}

func TestDevelopmentVerificationServiceRejectsUnknownScenario(t *testing.T) {
	service := &DevelopmentVerificationService{config: &config.Server{Environment: "development"}}

	if _, err := service.Prepare(context.Background(), "unknown"); err == nil ||
		!strings.Contains(err.Error(), "不支持的功能验证场景") {
		t.Fatalf("Prepare() error = %v, want unsupported scenario", err)
	}
	if _, err := service.Cleanup(context.Background(), "unknown"); err == nil ||
		!strings.Contains(err.Error(), "不支持的功能验证场景") {
		t.Fatalf("Cleanup() error = %v, want unsupported scenario", err)
	}
}

func TestDevelopmentVerificationServiceRequiresFixtureBaseURL(t *testing.T) {
	service := &DevelopmentVerificationService{config: &config.Server{Environment: "development"}}

	if _, err := service.requireVerificationFixtureBaseURL(); err == nil ||
		!strings.Contains(err.Error(), "verification_fixture_base_url") {
		t.Fatalf("requireVerificationFixtureBaseURL() error = %v, want missing fixture URL", err)
	}

	service.config.Integration.VerificationFixtureBaseURL = "http://frontend/"
	value, err := service.requireVerificationFixtureBaseURL()
	if err != nil || value != "http://frontend" {
		t.Fatalf("requireVerificationFixtureBaseURL() = %q, %v", value, err)
	}
}
